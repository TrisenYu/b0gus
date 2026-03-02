package configs

import (
	"context"
	"errors"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	mongo "go.mongodb.org/mongo-driver/mongo"
	gorm "gorm.io/gorm"

	b0gus_assets "b0gus/assets"
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

// create table if the table does not exist
// or try to mitigate table if possible
func (r *RuntimeDB) CreateTable(structures ...DBstruct) error {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	switch GuessAndDetermine(r.db) {
	case GormSQL: // after debugging
		var cast_interface_arr []any
		for _, tab_struct := range structures {
			cast_interface_arr = append(cast_interface_arr, tab_struct)
		}
		return r.db.(*gorm.DB).AutoMigrate(cast_interface_arr...)
	/* won't provide an interface for Redis due to the infeasibility to maintain primary key */
	case MongodbSQL:

		for _, tab_struct := range structures {
			r.db.(*mongo.Database).Collection(tab_struct.TableName())
		}
		return nil
	default:
		unsupported_msg := b0gus_assets.GetLocalizedMsg(
			GetLang(),
			"databases.DatabaseTypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		return errors.New(unsupported_msg)
	}
}

type DBstruct interface {
	// inspect if current struct has counter. If true, then autoincrement when
	// creating or updating an item
	HasCounter() bool

	// restrict and get table name
	TableName() string

	// Change Counter member if having defined in structure
	UpdateCounter()

	// Used for Foreign key. As the input value of setPrimKey
	GetPrimKey() int64

	// This function maintains relation tables.
	// As it shown, it will set 2-Dimension (i, j) coordinate in sequence.
	// But there might be some relation table having multiple foreign keys from diverse outer tables
	// So the in-parameter is kept as arbitrary number of int64, which is `...int64`.
	SetOuterPrimKey(...int64)
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
	CreateOrUpdateItemsInSeq(target_items ...DBstruct) error

	// set context and return function for executing in current context
	// roughly for foreign fields
	SetupContext() func(structures ...DBstruct)
}

func (r *RuntimeDB) QueryItem(cond_item DBstruct) DBstruct {
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
		// MongoDB is like a json manager
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
		unsupported_msg := b0gus_assets.GetLocalizedMsg(
			GetLang(),
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
	switch GuessAndDetermine(dst_db) {
	case GormSQL:
		r.db = dst_db
	case MongodbSQL:
		// TODO and to evaluate.
		r.db.(*mongo.Client).Disconnect(context.Background())
		r.db = dst_db
	default:
		// Won't change but show an error if such condition is satisfied
		unsupported_msg := b0gus_assets.GetLocalizedMsg(
			GetLang(),
			"databases.SQLtypeError",
			map[string]any{
				"Database": r.db,
			},
		)
		Logger.Error(unsupported_msg)
	}

}

// Create/Update items sequentially
func (r *RuntimeDB) CreateOrUpdateItemsInSeq(
	target_items ...DBstruct,
) error {
	var err error = nil
	// we can do one thing in this function: evaluate whether we can
	for _, item := range target_items {
		err = r.CreateOrUpdateItem(item, item)
		if err != nil {
			// TODO: rollback or skip?
			// Log might be better because we can not avoid side-effects.
			continue
		}

	}
	return err
}

// return relation structure if needed.
func (r *RuntimeDB) SetupContext(target_items ...DBstruct) {
	// Consider how to maintain context for databases...
	r.CreateOrUpdateItemsInSeq(target_items...)
	// we need a map structure to check this case
	// some table have primary key but not use as foreign key
	// O(n^2) brutal iteration.
	for i := range len(target_items) {
		for j := i + 1; j < len(target_items); j++ {
			// Firstly, access corresponding relation.
			item, jtem := target_items[i], target_items[j]
			if item.TableName() > jtem.TableName() {
				item, jtem = jtem, item
			} // In dictionary Order
			relation_tab, ok := RecRelationMap[item.TableName()][jtem.TableName()]
			if !ok {
				continue
			}
			cast_tab, ok := relation_tab.(DBstruct)
			if !ok {
				continue
			}
			// Maintain item.PrimaryKey and jtem.PrimaryKey
			cast_tab.SetOuterPrimKey([]int64{item.GetPrimKey(), jtem.GetPrimKey()}...)
			r.CreateOrUpdateItem(cast_tab, cast_tab)
		}
	}
}
