package databases

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License
/// Last modified at 2026/05/16 星期六 12:18:07

import (
	"b0gus/crypto_aux"
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongoopts "go.mongodb.org/mongo-driver/v2/mongo/options"
	gormpg "gorm.io/driver/postgres"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"b0gus/configs"
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

// won't provide querying interface currently, privileges should be kept for O&M

type DBhandler interface {
	// Setup will extract from given conf and set the corresponding database backend,
	// and setup table structures
	Setup(conf *configs.RecDBConfig, structures ...DBstruct) error

	// CreateTable creates table for relation database
	CreateTable(structures ...DBstruct) error

	// CreateOrUpdateItem single inserts/updates task
	CreateOrUpdateItem(cond, goal DBstruct) error

	// CreateOrUpdateItemsInSeq executes multiple inserting /updating tasks
	CreateOrUpdateItemsInSeq(...DBstruct) error

	// AlterDatabaseHandler typically set the dstDB into one of the member of interfaced structure.
	AlterDatabaseHandler(dstDB any)
}

// RuntimeDB is only used as a database **write** descriptor.
type RuntimeDB struct {
	// db holds the current database instance address
	db                      any
	TLSKeyPath, TLSCertPath string
	// Mutex protects the operation upon db specifically targeting at altering database instance
	Mutex sync.Mutex
}

func (r *RuntimeDB) Setup(conf *configs.RecDBConfig, structures ...DBstruct) error {
	dbInstance, str, err := r.SelectDatabaseBackend(conf)
	if err != nil {
		configs.Logger().Error(configs.GetLocalizedMsg(
			"databases.DatabaseConnectionError",
			map[string]any{
				"DatabaseStr": str,
				"ErrInfo":     err.Error(),
			},
		))
		return err
	} else if dbInstance == nil {
		configs.Logger().Error(configs.GetLocalizedMsg(
			"Databases.DatabaseEmptyError", nil,
		))
		return errors.New("empty database instance")
	}
	r.AlterDatabaseHandler(dbInstance)
	return r.CreateTable(structures...)
}

// CreateTable will create table if the target table does not exist
// or try to mitigate table if possible
func (r *RuntimeDB) CreateTable(structures ...DBstruct) error {
	if len(structures) < 1 || structures[0] == nil {
		return nil
	}
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
			configs.Logger().Error(payload)
			return errors.New(payload)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, tabStruct := range structures {
			err := db.CreateCollection(ctx, tabStruct.TableName())
			if err != nil && !mongo.IsDuplicateKeyError(err) {
				configs.Logger().Error(err.Error())
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
				configs.Logger().Warn(err.Error())
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
			configs.Logger().Error(currRes.Err().Error())
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
				configs.Logger().Error(err.Error())
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
		configs.Logger().Error(unsupportedMsg)
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
func (r *RuntimeDB) SelectDatabaseBackend(dbConfig *configs.RecDBConfig) (any, string, error) {
	switch strings.ToLower(dbConfig.DBType) {
	case "postgresql":
		return r.handlePostgreSQL(dbConfig)
	case "sqlite":
		// localdatabase as a file?
		sqlitePath, _ := filepath.Abs(dbConfig.DBPath)
		db, err := gorm.Open(gormsqlite.Open(sqlitePath), &gorm.Config{})
		return db, "sqlite", err
	case "mongodb":
		return r.handleMongoDB(dbConfig)
	default:
		payload := configs.GetLocalizedMsg(
			"configs.UnsupportDatabaseTypeError",
			map[string]any{"DatabaseType": dbConfig.DBType},
		)
		configs.Logger().Error(payload)
		return nil, "", errors.New(payload)
	}
}

func (r *RuntimeDB) handlePostgreSQL(dbConfig *configs.RecDBConfig) (any, string, error) {
	if !dbConfig.DBSelfCheck() {
		payload := configs.GetLocalizedMsg(
			"configs.InvalidDatabaseAddrError",
			map[string]any{"DatabaseAddr": dbConfig.DBAddr},
		)
		configs.Logger().Fatal(payload)
		return nil, "", errors.New(payload)
	}
	var sb strings.Builder
	sb.WriteString("host=")
	sb.WriteString(dbConfig.DBAddr)
	sb.WriteString(" port=") // deploy on certain port
	sb.WriteString(strconv.Itoa(int(dbConfig.DBPort)))
	sb.WriteString(" user=")
	sb.WriteString(dbConfig.DBAdminName)
	sb.WriteString(" password=")
	sb.WriteString(dbConfig.DBAdminPassword)
	sb.WriteString(" dbname=")
	sb.WriteString(dbConfig.DBName)
	sb.WriteString(" search_path=public")
	if configs.CheckFilePair(r.TLSKeyPath, r.TLSCertPath) && len(r.TLSKeyPath) > 0 {
		sb.WriteString(" sslmode=verify-full")
		sb.WriteString(" sslcert=")
		sb.WriteString(r.TLSCertPath) // "sslcert=/path/to/client.crt "
		sb.WriteString(" sslkey=")
		sb.WriteString(r.TLSKeyPath) // "sslkey=/path/to/client.key"
	}
	pgDbConfig := sb.String()
	db, err := gorm.Open(
		gormpg.Open(pgDbConfig),
		&gorm.Config{
			// set default timeout
			DefaultTransactionTimeout: time.Duration(dbConfig.DBTimeout) * time.Second,
		},
	)
	return db, "postgresql", err
}

// handleMongoDB
func (r *RuntimeDB) handleMongoDB(dbConfig *configs.RecDBConfig) (any, string, error) {
	if !dbConfig.DBSelfCheck() {
		payload := configs.GetLocalizedMsg(
			"configs.InvalidDatabaseAddrError",
			map[string]any{"DatabaseAddr": dbConfig.DBAddr},
		)
		configs.Logger().Fatal(payload)
		return nil, "", errors.New(payload)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var sb strings.Builder
	sb.WriteString("mongodb://")
	sb.WriteString(dbConfig.DBAdminName)
	sb.WriteRune(':')
	sb.WriteString(dbConfig.DBAdminPassword)
	sb.WriteRune('@')
	sb.WriteString(dbConfig.DBAddr)
	sb.WriteString(strconv.Itoa(int(dbConfig.DBPort)))

	mongoPayload := []*mongoopts.ClientOptions{
		mongoopts.Client().ApplyURI(sb.String()),
		mongoopts.Client().SetConnectTimeout(time.Duration(dbConfig.DBTimeout) * time.Second),
	}
	if configs.CheckFilePair(r.TLSKeyPath, r.TLSCertPath) && len(r.TLSKeyPath) > 0 {
		mongoPayload = append(mongoPayload,
			mongoopts.Client().SetTLSConfig(crypto_aux.LoadNormalCertAsTLSClient(
				r.TLSCertPath, r.TLSKeyPath, dbConfig.DBAddr,
			)),
		)
	}
	mongoClient, err := mongo.Connect(mongoPayload...)
	if err != nil {
		configs.Logger().Error(err.Error())
		return nil, "", err
	}
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		_ = mongoClient.Disconnect(ctx)
		configs.Logger().Error(err.Error())
		return nil, "", err
	}
	db := mongoClient.Database("b0gus-db")
	return db, "mongodb", err
}
