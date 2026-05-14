package databases

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"b0gus/configs"
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

// GuessAndDetermine casts the type of db(interface{}) to the knowns
// and return the iota-type representation of SQL.
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

// SelectDatabaseBackend will attempt to cast dbConfig into proper type
// by checking its Type member.
// this function return nil as RuntimeDB if there is any error during
// execution context.
//
// [TODO-other-db]:
//  1. https://www.cockroachlabs.com/docs/v26.1/build-a-go-app-with-cockroachdb-gorm?
//  2. https://pkg.go.dev/github.com/gocql/gocql
//
// [TODO]:
//  1. the choice of certain database should integrate into services configuration
//  2. sslmode=verify-full&sslrootcert=/var/lib/postgresql/ssl/root.crt.
//     which means configuration can assign a certificate for postgreSQL
func SelectDatabaseBackend(dbConfig *configs.RecDBConfig) (any, string, error) {
	switch strings.ToLower(dbConfig.Type) {
	case "postgresql": // deploy on certain port
		dbAddr := dbConfig.Addr
		if net.ParseIP(dbAddr) == nil && dbAddr != "localhost" {
			payload := configs.GetLocalizedMsg(
				"configs.InvalidDatabaseAddrError",
				map[string]any{"DatabaseAddr": dbAddr},
			)
			configs.Logger.Fatal(payload)
			return nil, "", errors.New(payload)
		}

		pgDbConfig := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s search_path=public",
			dbAddr, dbConfig.Port, dbConfig.AdminName,
			dbConfig.AdminPassword, dbConfig.Name,
		)

		db, err := gorm.Open(
			gormpg.Open(pgDbConfig), &gorm.Config{
				// set default timeout
				DefaultTransactionTimeout: time.Duration(dbConfig.Timeout) * time.Second,
			},
		)
		return db, "postgresql", err
	case "sqlite":
		// localdatabase as a file?
		sqlitePath, _ := filepath.Abs(dbConfig.Path)
		db, err := gorm.Open(gormsqlite.Open(sqlitePath), &gorm.Config{})
		return db, "sqlite", err
	case "mongodb":
		dbAddr := dbConfig.Addr
		if net.ParseIP(dbAddr) == nil && strings.ToLower(dbAddr) != "localhost" {
			payload := configs.GetLocalizedMsg(
				"configs.InvalidDatabaseAddrError",
				map[string]any{"DatabaseAddr": dbAddr},
			)
			configs.Logger.Error(payload)
			return nil, "", errors.New(payload)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		mongoClient, err := mongo.Connect(
			ctx,
			// TODO: check configuration and add TLS.
			mongoopts.Client().ApplyURI(fmt.Sprintf(
				"mongodb://%s:%s@%s:%d",
				// could not include `: / ? # [ ] @` in admin_name or password,
				// otherwise they need convert in the way that url encoding criterion
				// that enforces
				dbConfig.AdminName, dbConfig.AdminPassword,
				dbAddr, dbConfig.Port,
			)),
		)
		if err != nil {
			configs.Logger.Error(err.Error())
			return nil, "", err
		}
		err = mongoClient.Ping(ctx, nil)
		if err != nil {
			_ = mongoClient.Disconnect(ctx)
			configs.Logger.Error(err.Error())
			return nil, "", err
		}
		db := mongoClient.Database("b0gus-db")
		return db, "mongodb", err
	default:
		payload := configs.GetLocalizedMsg(
			"configs.UnsupportDatabaseTypeError",
			map[string]any{"DatabaseType": dbConfig.Type},
		)
		configs.Logger.Error(payload)
		return nil, "", errors.New(payload)
	}
}

// callback functions and context might be required

type DBstruct interface {
	// HasCounter inspect if current struct has counter. If true, then autoincrement when
	// creating or updating an item
	HasCounter() bool

	// CreatedTSname will return the name of createdTimestamp if possible.
	CreatedTSname() string

	// UpdatedTSname will return the name of updatedTimestamp if possible.
	UpdatedTSname() string

	// TableName gets the name of table
	TableName() string

	// UpdateCounter Changes the Counter member if having defined in structure
	UpdateCounter()

	// GetPrimKey is designed for Foreign key. As the input value of setPrimKey
	GetPrimKey() int64

	// PrimKeyName will return the name of primary key
	PrimKeyName() string
}

// won't provide querying interface, privileges should be kept for O&M

type DBhandler interface {
	// CreateTable creates table for relation database
	CreateTable(structures ...DBstruct) error

	// CreateOrUpdateItem single inserts/updates task
	CreateOrUpdateItem(cond, goal DBstruct) error

	// CreateOrUpdateItemsInSeq executes multiple inserting /updating tasks
	CreateOrUpdateItemsInSeq(...DBstruct) error

	AlterDatabaseHandler(dstDB any)
}

// RuntimeDB is only used as a database **write** descriptor.
type RuntimeDB struct {
	// db holds the current database instance address
	db any

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
		db, ok := r.db.(*mongo.Database)
		if !ok {
			payload := configs.GetLocalizedMsg("databases.InvalidMongoDBclientErr", nil)
			configs.Logger.Error(payload)
			return errors.New(payload)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, tabStruct := range structures {
			err := db.CreateCollection(ctx, tabStruct.TableName())
			if err != nil && !mongo.IsDuplicateKeyError(err) {
				configs.Logger.Error(err.Error())
			}
		}
		return nil
	default:
		unsupportedMsg := configs.GetLocalizedMsg(
			"databases.DatabaseTypeError",
			map[string]any{"Database": r.db},
		)
		return errors.New(unsupportedMsg)
	}
}

func (r *RuntimeDB) CreateOrUpdateItem(cond, goal DBstruct) error {
	// counter or foreign key should be maintained by assignments
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		var gormDB = r.db.(*gorm.DB)
		if goal.HasCounter() {
			err := gormDB.Where(cond).Find(goal).Error
			if err == nil {
				goal.UpdateCounter()
				gormDB.Save(goal)
				return nil
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				configs.Logger.Warn(err.Error())
				return err
			}
		}
		t := goal.UpdatedTSname()
		if len(t) > 0 {
			gormDB.Where(cond).
				Assign(map[string]interface{}{t: time.Now()}).
				FirstOrCreate(goal, cond)
		} else {
			gormDB.Where(cond).FirstOrCreate(goal, cond)
		}

	case MongodbSQL:
		// [TODO] yet to be fully tested
		// MongoDB is like a JSON manager
		// collection for specific name and the interface is for struct
		update := bson.M{"$set": goal}
		var mongoDB = r.db.(*mongo.Database)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		currRes := mongoDB.Collection(cond.TableName()).FindOne(ctx, cond)
		if currRes.Err() != nil && !errors.Is(currRes.Err(), mongo.ErrNoDocuments) {
			configs.Logger.Error(currRes.Err().Error())
			return currRes.Err()
		} else if currRes.Err() == nil {
			// update here
			_, err := mongoDB.Collection(goal.TableName()).UpdateOne(ctx, cond, update)
			return err
		}
		_, err := currRes.Raw()
		if err == nil {
			err := currRes.Decode(&goal)
			if err != nil { // return here because we do not get the value of counter
				configs.Logger.Error(err.Error())
				return currRes.Err()
			}
			// keep doing if nil
		}
		if goal.HasCounter() {
			goal.UpdateCounter()
		}
		_, err = mongoDB.Collection(goal.TableName()).InsertOne(ctx, goal)
		return err
	default:
		unsupportedMsg := configs.GetLocalizedMsg(
			"databases.DatabaseTypeError",
			map[string]any{"Database": r.db},
		)
		return errors.New(unsupportedMsg)
	}
	return nil
}

// AlterDatabaseHandler will analyze database fd created from SelectDatabaseBackend
// and attempt to set it into RuntimeDB.db.
func (r *RuntimeDB) AlterDatabaseHandler(dstDB any) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(dstDB) {
	case GormSQL:
		r.db = dstDB // revoke by GC
	case MongodbSQL:
		casted, ok := r.db.(*mongo.Client)
		if ok {
			_ = casted.Disconnect(context.Background())
		}
		r.db = dstDB
	default:
		// Won't change but show an error if such condition is satisfied
		unsupportedMsg := configs.GetLocalizedMsg(
			"databases.SQLtypeError",
			map[string]any{"Database": r.db},
		)
		configs.Logger.Error(unsupportedMsg)
	}

}

// CreateOrUpdateItemsInSeq Creates/Updates items sequentially
func (r *RuntimeDB) CreateOrUpdateItemsInSeq(targetItems ...DBstruct) error {
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
