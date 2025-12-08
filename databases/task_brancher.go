package databases

import (
	b0gus_config "b0gus/configs"
	"context"
	"fmt"
	"sync"

	redis "github.com/redis/go-redis/v9"
	mongo "go.mongodb.org/mongo-driver/mongo"
	gorm "gorm.io/gorm"
)

const (
	GormSQL = iota
	RedisSQL
	MongodbSQL
	UnkSQL
)

func GuessAndDetermine(db any) int {
	switch db.(type) {
	case *gorm.DB:
		return GormSQL
	case *redis.Client:
		return RedisSQL
	case *mongo.Client:
		return MongodbSQL
	default:
		resp := fmt.Sprintf("Unknown datatype<%v> was provided", db)
		b0gus_config.Logger.Warn(resp)
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
}

type RecordDB struct {
	db    any
	Mutex sync.Mutex
}

func (r *RecordDB) CreateTable(structures ...any) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		return r.db.(*gorm.DB).AutoMigrate(structures...)
	case RedisSQL:
		return nil
	case MongodbSQL:
		return nil
	default:
		return fmt.Errorf("unsupported database<%v> was found", r.db)
	}
}

type DBtype interface {
	interface{} | string
}

func (r *RecordDB) CreateOrUpdateItem(
	target_item_ptr *struct{},
	update_obj any,
) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	switch GuessAndDetermine(r.db) {
	case GormSQL:
		tx := r.db.(*gorm.DB).Where(*target_item_ptr).FirstOrCreate(target_item_ptr)
		if update_obj != nil {
			tx.Updates(update_obj)
		}
	case RedisSQL:
		ctx := context.Background()
		// First of all, we don't have keyID at all so we have to generate a unique key ID
		// Second, we also want to search a row via fuzzy content and bypass key ID but it is not trival to do
		// TODO:
		// *target_item_ptr is a structure which can represent the target_item
		// what we want is to extract struct as a key-value dict, where the key is the name of field defined in struct,
		// and the value is the corresponding value that has assigned before
		//
		// we also have defined primaryKey in structure by tagging techique
		// So maybe we can utilize them as hash

		// HGet(ctx context.Context, key string, field string)
		_, err := r.db.(*redis.Client).HGet(ctx, "", "").Result()
		if err != nil {
			if err != redis.Nil {
				b0gus_config.Logger.Warnf(
					"Catpure an unknown error when trying to get result from redis: %v", err,
				)
				return
			}
			// then we could insert it to redis
			r.db.(*redis.Client).HSet(ctx, "", fmt.Errorf("yet to implement AND TODO TODO TODO TODO"))
			return
		}
		// otherwise we have to update this value
	case MongodbSQL:
		// MongoDB is like a json manager
	default:
		// fmt.Errorf("unsupported database<%v> was found", r.db)
	}

}

func (r *RecordDB) AlterDatabaseHandler(dst_db any) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL:
		r.db = dst_db
	case RedisSQL:
		err := r.db.(*redis.Client).Close()
		if err != nil {
			b0gus_config.Logger.Warnf("Catpure an err: %v", err)
		}
		r.db = dst_db
	case MongodbSQL:
		// TODO
		r.db.(*mongo.Client).Disconnect(context.Background())
		r.db = dst_db
	default:
		// Won't change but show an error if such condition is satisfied
		b0gus_config.Logger.Errorf("unsupported database<%v> was found, won't change database", r.db)
	}

}

// Create/Update items sequentially
func (r *RecordDB) CreateOrUpdateItemsInSeq(target_items []any, update_obj []any) error {
	return nil
}
