package databases

import (
	"context"
	"errors"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	mongo "go.mongodb.org/mongo-driver/mongo"
	gorm "gorm.io/gorm"

	b0gus_assets "b0gus/assets"
	b0gus_config "b0gus/configs"
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

// callback functions and context might be required
type RuntimeDB struct {
	db    any
	Mutex sync.Mutex
}

func (r *RuntimeDB) CreateTable(structures ...DBstruct) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		return r.db.(*gorm.DB).AutoMigrate(structures)
	/* won't provide an interface for Redis due to the infeasibility to maintain primary key */
	case MongodbSQL:
		for _, tab_struct := range structures {
			r.db.(*mongo.Database).Collection(tab_struct.TableName())
		}
		return nil
	default:
		unsupported_msg := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"databases.DatabaseTypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		return errors.New(unsupported_msg)
	}
}

type DBtype interface {
	any | string
}

type DBstruct interface {
	// inspect if current struct has counter. If true, then autoincrement when
	// creating or updating an item
	HasCounter() bool

	// restrict and get table name
	TableName() string

	// Change Counter member if having defined in structure
	UpdateCounter()

	// Auto-Set Foreign Key.
}

// won't provide querying interface, priviledges should be kept for O&M
type DBhandler interface {
	// close database when there is indeed an operating interface
	Close()

	// create table for relation database
	CreateTable(structures ...DBstruct) error

	// single inserting/updating task
	CreateOrUpdateItem(cond *DBstruct, goal DBstruct) error

	// multiple inserting /updating tasks
	CreateOrUpdateItemsInSeq(target_items []DBstruct) error

	// set context for foreign fields
	SetupContext()
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
		gormDB.Where(&cond).FirstOrCreate(&goal)
		if goal.HasCounter() {
			goal.UpdateCounter()
			gormDB.Where(goal).Updates(&goal)
		}
	case MongodbSQL:
		// MongoDB is like a json manager
		// collection for specific name and the interface is for struct
		// r.db.(*mongo.Database).Collection().InsertOne(context.TODO(), )
		update := bson.M{ // changed position
			"$set": goal,
		}
		var mongoDB = r.db.(*mongo.Database)
		mongoDB.Collection(cond.TableName()).
			FindOneAndUpdate(context.TODO(), cond, update)
		if goal.HasCounter() {
			goal.UpdateCounter()
			mongoDB.Collection(cond.TableName()).
				FindOneAndUpdate(context.TODO(), cond, update)
		}
	default:
		unsupported_msg := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"databases.DatabaseTypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		return errors.New(unsupported_msg)
	}
	return nil
}

func (r *RuntimeDB) AlterDatabaseHandler(dst_db any) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		r.db = dst_db
	case MongodbSQL:
		// TODO
		r.db.(*mongo.Client).Disconnect(context.TODO())
		r.db = dst_db
	default:
		// Won't change but show an error if such condition is satisfied
		unsupported_msg := b0gus_assets.GetLocalizedMsg(
			b0gus_config.GetLang(),
			"databases.SQLtypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		b0gus_config.Logger.Error(unsupported_msg)
	}

}

// Create/Update items sequentially
func (r *RuntimeDB) CreateOrUpdateItemsInSeq(
	target_items []DBstruct,
) error {
	var err error = nil
	// we can do one thing in this function: evaluate whether we can
	for _, item := range target_items {
		err = r.CreateOrUpdateItem(item, item)
		if err != nil {
			// TODO: rollback?
		}
	}
	return err
}

func (r *RuntimeDB) SetupContext() {
	// Consider how to maintain context for databases...
	//
}
