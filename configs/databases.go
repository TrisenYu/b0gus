package configs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mongoopts "go.mongodb.org/mongo-driver/mongo/options"
	gormpg "gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	GormSQL = iota
	MongodbSQL
	UnkSQL
)

func GuessAndDetermine(db any) int {
	switch db.(type) {
	case *gorm.DB:
		return GormSQL
	case *mongo.Database:
		return MongodbSQL
	default:
		return UnkSQL
	}
}

func SelectDatabaseBackend(dbConfig *RecDBConfig) (any, string, error) {
	switch strings.ToLower(dbConfig.Type) {
	case "postgresql": // deploy on certain port
		dbAddr := dbConfig.Addr
		if net.ParseIP(dbAddr) == nil && dbAddr != "localhost" {
			payload := GetLocalizedMsg(
				"configs.InvalidDatabaseAddrError",
				map[string]any{"DatabaseAddr": dbAddr},
			)
			Logger.Fatal(payload)
			return nil, "", errors.New(payload)
		}
		/*
			TODO: sslmode=verify-full&sslrootcert=/var/lib/postgresql/ssl/root.crt
			which means configuration can assign a certificate for postgreSQL
		*/
		pgDbConfig := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s search_path=public",
			dbAddr, dbConfig.Port, dbConfig.AdminName,
			dbConfig.AdminPassword, dbConfig.Name,
		)
		db, err := gorm.Open(gormpg.Open(pgDbConfig), &gorm.Config{})
		return db, "postgresql", err
	case "sqlite": // localdatabase as a file
		absAssetsDirPath, _ := filepath.Abs(AssetsDirAsStr)
		sqlitePath := dbConfig.Path
		sqlitePath = filepath.Join(absAssetsDirPath, sqlitePath)
		db, err := gorm.Open(gormsqlite.Open(sqlitePath), &gorm.Config{})
		return db, "sqlite", err
	case "mongodb":
		dbAddr := dbConfig.Addr
		if net.ParseIP(dbAddr) == nil && strings.ToLower(dbAddr) != "localhost" {
			payload := GetLocalizedMsg(

				"configs.InvalidDatabaseAddrError",
				map[string]any{"DatabaseAddr": dbAddr},
			)
			Logger.Fatal(payload)
			return nil, "", errors.New(payload)
		}
		mongoClient, err := mongo.Connect(
			context.Background(),
			// TODO: There are multiple ways to connect to standalone database
			mongoopts.Client().ApplyURI(fmt.Sprintf(
				"mongodb://%s:%s@%s:%d",
				// could not include `: / ? # [ ] @` in admin_name or password,
				// otherwise they need convert in the way that url encoding criterion
				// that enforces
				dbConfig.AdminName, dbConfig.AdminPassword,
				dbAddr, dbConfig.Port,
			)),
		)
		return mongoClient, "mongodb", err
	default:
		payload := GetLocalizedMsg(
			"configs.UnsupportDatabaseTypeError",
			map[string]any{
				"DatabaseType": dbConfig.Type,
			},
		)
		Logger.Error(payload)
		return nil, "", errors.New(payload)
	}

}

// callback functions and context might be required

type RuntimeDB struct {
	db any // db holds the current database instance address
	// Mutex protects the operation upon db specifically targeting at altering database instance
	Mutex sync.Mutex
}

// CreateTable will create table if the target table does not exist
// or try to mitigate table if possible
func (r *RuntimeDB) CreateTable(structures ...DBstruct) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL: // after debugging
		var castInterfaceArr []any
		for _, tabStruct := range structures {
			castInterfaceArr = append(castInterfaceArr, tabStruct)
		}
		return r.db.(*gorm.DB).AutoMigrate(castInterfaceArr...)
	/* won't provide an interface for Redis due to the infeasibility to maintain primary key */
	case MongodbSQL:
		for _, tabStruct := range structures {
			r.db.(*mongo.Database).Collection(tabStruct.TableName())
		}
		return nil
	default:
		unsupportedMsg := GetLocalizedMsg(

			"databases.DatabaseTypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		return errors.New(unsupportedMsg)
	}
}

type DBstruct interface {
	// HasCounter inspect if current struct has counter. If true, then autoincrement when
	// creating or updating an item
	HasCounter() bool

	// TableName gets the name of table
	TableName() string

	// UpdateCounter Changes the Counter member if having defined in structure
	UpdateCounter()

	// GetPrimKey is designed for Foreign key. As the input value of setPrimKey
	GetPrimKey() int64

	// SetOuterPrimKey maintains relation tables.
	// As it shown, it will set 2-Dimension (i, j) coordinate in sequence.
	// But there might be some relation table having multiple foreign keys from diverse outer tables
	// So the in-parameter is kept as arbitrary number of int64, which is `...int64`.
	SetOuterPrimKey(...int64)
}

// won't provide querying interface, privileges should be kept for O&M

type DBhandler interface {
	// Close closes database when there is indeed an operating interface
	Close()

	// CreateTable creates table for relation database
	CreateTable(structures ...DBstruct) error

	// CreateOrUpdateItem single inserts/updates task
	CreateOrUpdateItem(cond *DBstruct, goal DBstruct) error

	// CreateOrUpdateItemsInSeq executes multiple inserting /updating tasks
	CreateOrUpdateItemsInSeq(...DBstruct) error

	// SetupContext form the context and returns function
	// for executing in current context roughly for foreign fields
	SetupContext() func(...DBstruct)
}

func (r *RuntimeDB) QueryItem(condItem DBstruct) DBstruct {
	// TODO
	return nil
}

func (r *RuntimeDB) CreateOrUpdateItem(
	cond, goal DBstruct,
) error {
	// counter or foreign key should be maintained by assignments
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	switch GuessAndDetermine(r.db) {
	case GormSQL:
		var gormDB = r.db.(*gorm.DB)
		gormDB.Where(cond).FirstOrCreate(goal)
		if goal.HasCounter() {
			goal.UpdateCounter()
			gormDB.Where(goal).Updates(goal)
		}
	case MongodbSQL:
		// MongoDB is like a JSON manager
		// collection for specific name and the interface is for struct
		update := bson.M{ // changed position
			"$set": goal,
		}
		var mongoDB = r.db.(*mongo.Database)
		mongoDB.Collection(cond.TableName()).
			FindOneAndUpdate(context.Background(), cond, update)
		if goal.HasCounter() {
			goal.UpdateCounter()
			mongoDB.Collection(cond.TableName()).
				FindOneAndUpdate(context.Background(), cond, update)
		}
	default:
		unsupportedMsg := GetLocalizedMsg(
			"databases.DatabaseTypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		return errors.New(unsupportedMsg)
	}
	return nil
}

func (r *RuntimeDB) AlterDatabaseHandler(dstDB any) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(dstDB) {
	case GormSQL:
		r.db = dstDB
	case MongodbSQL:
		// TODO and to evaluate.
		_ = r.db.(*mongo.Client).Disconnect(context.Background())
		r.db = dstDB
	default:
		// Won't change but show an error if such condition is satisfied
		unsupportedMsg := GetLocalizedMsg(
			"databases.SQLtypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		Logger.Error(unsupportedMsg)
	}

}

// CreateOrUpdateItemsInSeq Creates/Updates items sequentially
func (r *RuntimeDB) CreateOrUpdateItemsInSeq(
	targetItems ...DBstruct,
) error {
	var err error = nil
	// we can do one thing in this function: evaluate whether we can
	for _, item := range targetItems {
		err = r.CreateOrUpdateItem(item, item)
		if err != nil {
			// TODO: rollback or skip?
			// Log might be better because we can not avoid side-effects.
			continue
		}
	}
	return err
}

// SetupContext will return relation structure if needed.
func (r *RuntimeDB) SetupContext(targetItems ...DBstruct) {
	// Consider how to maintain context for databases...
	_ = r.CreateOrUpdateItemsInSeq(targetItems...)
	// we need a map structure to check this case
	// some table have primary key but not use as foreign key
	// O(n^2) brutal iteration.
	for i := range len(targetItems) {
		for j := i + 1; j < len(targetItems); j++ {
			// Firstly, access corresponding relation.
			item, jtem := targetItems[i], targetItems[j]
			if item.TableName() > jtem.TableName() {
				item, jtem = jtem, item
			} // In dictionary Order
			relationTab, ok := RecRelationMap[item.TableName()][jtem.TableName()]
			if !ok {
				continue
			}
			castTab, ok := relationTab.(DBstruct)
			if !ok {
				continue
			}
			// Maintain item.PrimaryKey and jtem.PrimaryKey
			castTab.SetOuterPrimKey([]int64{item.GetPrimKey(), jtem.GetPrimKey()}...)
			_ = r.CreateOrUpdateItem(castTab, castTab)
		}
	}
}
