package databases

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	mongo "go.mongodb.org/mongo-driver/mongo"
	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_misc_utils "b0gus/misc_utils"
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

type DBhandler interface {
	// close database when there is indeed an operating interface
	Close()

	// create table for relation database
	CreateTable(structures ...any) error

	// single inserting/updating task
	CreateOrUpdateItem(target_item *any, update_obj any) error

	// multiple inserting /updating tasks
	CreateOrUpdateItemsInSeq(target_items []any, update_obj []any) error

	// set context for foreign fields
	SetupContext()
}

// callback functions and context might be required
type RuntimeDB struct {
	db    any
	Mutex sync.Mutex
}

func (r *RuntimeDB) CreateTable(structures ...any) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		return r.db.(*gorm.DB).AutoMigrate(structures...)
	// won't provide an interface for Redis due to the infeasibility to maintain primary key
	case MongodbSQL:
		for tab_struct := range structures {
			// TODO: reflect help to unwrap ? can not make me convinced
			r.db.(*mongo.Database).Collection(b0gus_misc_utils.GetTypeNameViaType(tab_struct))
		}
		return nil
	default:
		return fmt.Errorf("unsupported database<%v> was found", r.db)
	}
}

type DBtype interface {
	any | string
}

func (r *RuntimeDB) CreateOrUpdateItem(
	cond, goal_state *struct{},
	update_obj any,
) {
	// TODO: how to help counter or foreign key?
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	switch GuessAndDetermine(r.db) {
	case GormSQL:
		tx := r.db.(*gorm.DB).Where(*cond).FirstOrCreate(goal_state)
		if update_obj != nil {
			tx.Updates(update_obj)
		}
	case MongodbSQL:
		// MongoDB is like a json manager
		// collection for specific name and the interface is for struct
		// r.db.(*mongo.Database).Collection().InsertOne(context.TODO(), )

		// however, we met problem if we want to partially modify
		update := bson.M{ // changed position
			"$set": *goal_state,
		}
		res, err := r.db.(*mongo.Database).
			Collection(b0gus_misc_utils.GetStructNameByType(cond)).
			UpdateOne(context.TODO(), cond, update)
		if err != nil {
			b0gus_config.Logger.Errorf(
				"update item to collection met an error: %v, id:%d",
				err, res.UpsertedID,
			)
		}
	default:
		b0gus_config.Logger.Errorf("unsupported database<%v> was found", r.db)
	}

}

func (r *RuntimeDB) AlterDatabaseHandler(dst_db any) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		r.db = dst_db
	case MongodbSQL:
		// TODO
		r.db.(*mongo.Client).Disconnect(context.Background())
		r.db = dst_db
	default:
		// Won't change but show an error if such condition is satisfied
		b0gus_config.Logger.Errorf(
			"unsupported database<%v> was found, won't change database",
			r.db,
		)
	}

}

// Create/Update items sequentially
func (r *RuntimeDB) CreateOrUpdateItemsInSeq(
	target_items []any, update_obj []any,
) error {
	// TODO.
	return nil
}
