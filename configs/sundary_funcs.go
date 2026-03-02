// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package configs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	b0gus_assets "b0gus/assets"

	mongo "go.mongodb.org/mongo-driver/mongo"
	mongo_opts "go.mongodb.org/mongo-driver/mongo/options"
	gorm_pg "gorm.io/driver/postgres" // pg stands for PostGreSQL
	gorm_sqlite "gorm.io/driver/sqlite"
	gorm "gorm.io/gorm"

	fsnotify "github.com/fsnotify/fsnotify"
	viper "github.com/spf13/viper"
)

func SelectDatabaseBackend(dbdbConfonf *RecDBConfig) (any, string, error) {
	switch strings.ToLower(dbdbConfonf.Type) {
	case "postgresql": // deploy on certain port
		dbAddr := dbdbConfonf.Addr
		if net.ParseIP(dbAddr) == nil && dbAddr != "localhost" {
			payload := b0gus_assets.GetLocalizedMsg(
				GetLang(),
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
		pg_db_config := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s search_path=public",
			dbAddr, dbdbConfonf.Port, dbdbConfonf.AdminName,
			dbdbConfonf.AdminPassword, dbdbConfonf.Name,
		)
		db, err := gorm.Open(gorm_pg.Open(pg_db_config), &gorm.Config{})
		return db, "postgresql", err
	case "sqlite": // localdatabase as a file
		abs_assets_dir_path, _ := filepath.Abs(AssetsDirAsStr)
		sqlite_path := dbdbConfonf.Path
		sqlite_path = filepath.Join(abs_assets_dir_path, sqlite_path)
		db, err := gorm.Open(gorm_sqlite.Open(sqlite_path), &gorm.Config{})
		return db, "sqlite", err
	case "mongodb":
		db_addr := dbdbConfonf.Addr
		if net.ParseIP(db_addr) == nil && strings.ToLower(db_addr) != "localhost" {
			payload := b0gus_assets.GetLocalizedMsg(
				GetLang(),
				"configs.InvalidDatabaseAddrError",
				map[string]any{"DatabaseAddr": db_addr},
			)
			Logger.Fatal(payload)
			return nil, "", errors.New(payload)
		}
		mongo_client, err := mongo.Connect(
			context.Background(),
			// TODO: There are multiple ways to connect to standalone database
			mongo_opts.Client().ApplyURI(fmt.Sprintf(
				"mongodb://%s:%s@%s:%d",
				// could not include `: / ? # [ ] @` in admin_name or password,
				// otherwise they need convert in the way that url encoding criterion
				// that enforces
				dbdbConfonf.AdminName, dbdbConfonf.AdminPassword,
				db_addr, dbdbConfonf.Port,
			)),
		)
		return mongo_client, "mongodb", err
	default:
		payload := b0gus_assets.GetLocalizedMsg(
			GetLang(), "configs.UnsupportDatabaseTypeError",
			map[string]any{
				"DatabaseType": dbdbConfonf.Type,
			},
		)
		Logger.Error(payload)
		return nil, "", errors.New(payload)
	}

}

// GetLang Access expected language defined in configuration.
//
//	Warn: loose concurrent restriction
func GetLang() string {
	return curr_config.ServerConfig.Language
}

// TODO: This function should be taken down due to the ambiguous definition is acceptable at b0gus now.
// Some service can selectively update its domain
func inspectConfig(lc *LocalConfig) bool {
	// Logger.Infof("%v", conf_data)
	// res &= CheckDNSconfig(conf_data.ServerConfig.DNS)
	return true
}

var tmpConf = &LocalConfig{}

func LoadDefaultConfig(confPath string) *LocalConfig {
	if confPath == "" {
		confPath = ConfigPathAsStr
	}
	viper.SetConfigFile(confPath)
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		payload := b0gus_assets.GetLocalizedMsg(
			GetLang(),
			"configs.ReadingConfigurationFailure",
			map[string]any{"ErrInfo": err},
		)
		Logger.Fatal(payload)
		return nil
	}

	viper.OnConfigChange(func(fsnotify.Event) {
		err := viper.Unmarshal(&tmpConf)
		if err != nil {
			payload := b0gus_assets.GetLocalizedMsg(
				GetLang(),
				"configs.ReloadingConfigurationFailure",
				map[string]any{"ErrInfo": err},
			)
			Logger.Error(payload)
			return
		}
		if inspectConfig(tmpConf) {
			// change and update
			UpdateFlag <- tmpConf
			curr_config = *tmpConf
		}
	})
	viper.WatchConfig()
	err = viper.Unmarshal(&curr_config)
	if err != nil {
		payload := b0gus_assets.GetLocalizedMsg(
			GetLang(),
			"configs.UnmarshalingConfigurationFailure",
			map[string]any{"ErrInfo": err},
		)
		Logger.Fatal(payload)
		return nil
	}
	return &curr_config
}
