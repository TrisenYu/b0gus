// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package configs

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	mongo "go.mongodb.org/mongo-driver/mongo"
	mongo_opts "go.mongodb.org/mongo-driver/mongo/options"
	gorm_pg "gorm.io/driver/postgres" // pg stands for PostGreSQL
	gorm_sqlite "gorm.io/driver/sqlite"
	gorm "gorm.io/gorm"

	fsnotify "github.com/fsnotify/fsnotify"
	viper "github.com/spf13/viper"
)

// any here is always a pointer
func SelectDatabaseBackend(db_conf DatabaseConfig) (any, string, error) {
	switch strings.ToLower(db_conf.DatabaseType) {
	case "postgresql": // deploy on certain port
		db_addr := db_conf.DatabaseAddr
		if net.ParseIP(db_addr) == nil && strings.ToLower(db_addr) != "localhost" {
			Logger.WithField("database addr", db_addr).
				Fatal("Invalid database address was gained from configuration!")
		}
		/* TODO: sslmode=verify-full&sslrootcert=/var/lib/postgresql/ssl/root.crt */
		pg_db_config := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s search_path=public",
			db_addr, db_conf.DatabasePort, db_conf.DatabaseAdminName,
			db_conf.DatabaseAdminPassword, db_conf.DatabaseName,
		)
		db, err := gorm.Open(gorm_pg.Open(pg_db_config), &gorm.Config{})
		return db, "postgresql", err
	case "sqlite": // as a file
		abs_assets_dir_path, _ := filepath.Abs(Assets_dir_as_str)
		sqlite_path := db_conf.DatabasePath
		sqlite_path = filepath.Join(abs_assets_dir_path, sqlite_path)
		db, err := gorm.Open(gorm_sqlite.Open(sqlite_path), &gorm.Config{})
		return db, "sqlite", err
	/* TODO: no-relation database, we might need a more generic handler and concurrent protector */
	// case "redis":
	// 	db_addr := db_conf.DatabaseAddr
	// 	rdb := redis.NewClient(&redis.Options{
	// 		Addr:     fmt.Sprintf("%s:%d", db_addr, db_conf.DatabasePort),
	// 		Password: db_conf.DatabaseAdminPassword,
	// 		DB:       0,
	// 		/*TODO:
	// 		TLSConfig: &tls.Config{
	// 			MinVersion: tls.VersionTLS12,
	// 			ServerName: "you domain",
	// 			//Certificates: []tls.Certificate{cert}
	// 		},
	// 		*/
	// 	})
	// 	if rdb == nil {
	// 		return nil, "", fmt.Errorf("can't create redis-client")
	// 	}
	// 	ctx := context.Background()
	// 	_, err := rdb.Ping(ctx).Result()
	// 	if err != nil {
	// 		return nil, "", fmt.Errorf("can't create redis-client due to: %v", err)
	// 	}
	// 	// rdb.Do(ctx, "cmd-type1", "val1", ..., "typen", "valn")
	// 	return rdb, "redis", nil
	case "mongodb":
		db_addr := db_conf.DatabaseAddr
		if net.ParseIP(db_addr) == nil && strings.ToLower(db_addr) != "localhost" {
			Logger.WithField("database addr", db_addr).
				Fatal("Invalid database address was gained from configuration!")
		}
		mongo_client, err := mongo.Connect(
			context.Background(),
			// TODO: There are multiple ways to connect to standalone database
			mongo_opts.Client().ApplyURI(fmt.Sprintf(
				"mongodb://%s:%s@%s:%d",
				// could not include `: / ? # [ ] @` in admin_name or password,
				// otherwise they need convert in the way that url encoding criterion
				// that enforces
				db_conf.DatabaseAdminName, db_conf.DatabaseAdminPassword,
				db_addr, db_conf.DatabasePort,
			)),
		)
		return mongo_client, "mongodb", err
	default:
		Logger.Errorf(
			`Unknown or unsupported database type was found, 
only support PostgreSQL and SQLite at present...Database you selecte: %v`,
			db_conf.DatabaseType,
		)
		return nil, "", fmt.Errorf(
			"unknown or unsupported database type<%v> was found",
			db_conf.DatabaseType,
		)
	}

}

func CheckSSHconfig(ssh_conf *SSHconfig) bool {
	// ssh_conf.ListenAddr might be localhost, which can not be accepted by net.ParseIP
	if ssh_conf == nil || ssh_conf.ListenAddr == "" || ssh_conf.ListenPort <= 1024 {
		return false
	}
	/* At this moment and at most, we can only check whether those fields are null */
	if ssh_conf.MaxClientNum == 0 {
		ssh_conf.MaxClientNum = 1
	} else if ssh_conf.ClientConnTimeout == 0 {
		ssh_conf.ClientConnTimeout = 30
	} else if ssh_conf.ResponseType == "" {
		ssh_conf.ResponseType = "Always-Reject"
	}
	return true
}

func CheckTelnetConfig(telnet_conf *TelnetConfig) bool {
	return telnet_conf != nil &&
		telnet_conf.ListenAddr != "" &&
		telnet_conf.ListenPort > 1024
}

func CheckNTPconfig(ntp_conf *NTPconfig) bool {
	return ntp_conf != nil && ntp_conf.ListenAddr != "" &&
		ntp_conf.ListenPort > 1024
}

func CheckDNSconfig(dns_conf *DNSconfig) bool {
	return dns_conf != nil && dns_conf.ListenAddr != "" &&
		dns_conf.ListenPort > 1024
}

// TODO: This function should be taken down due to the ambiguous definition is acceptable at b0gus now.
// Some service can selectively update its domain
func inspectConfig(conf_data *LocalConfig) bool {
	Logger.Infof("%v", conf_data)
	// res &= CheckDNSconfig(conf_data.ServerConfig.DNS)
	return false
}

func GetLang() string {
	// loose concurrent restriction
	return curr_config.ServerConfig.Language
}

func LoadDefaultConfig(conf_path string) *LocalConfig {
	if conf_path == "" {
		conf_path = Config_path_as_str
	}
	viper.SetConfigFile(conf_path)
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		Logger.Fatalf("Read configuration failed: %v", err)
	}

	viper.OnConfigChange(func(conf_changes fsnotify.Event) {
		Logger.Infof("Configuration change: %s", conf_changes.Name)
		var tmp_conf LocalConfig
		err := viper.Unmarshal(&tmp_conf)
		if err != nil {
			Logger.Errorf("Reload configuration failed: %v, won't update current configuration", err)
			return
		}
		if inspectConfig(&tmp_conf) {
			UpdateFlag <- &tmp_conf
			// change and update
			curr_config = tmp_conf
		}
	})
	viper.WatchConfig()
	err = viper.Unmarshal(&curr_config)
	if err != nil {
		Logger.Fatalf("Unmarshal config failed: %v", err)
	}
	return &curr_config
}

func genericChecker[T AbsServType](inp any, inf func(*T) bool) bool {
	curr, ok := inp.(T)
	if !ok {
		return false
	}
	return inf(&curr)
}

func InfoEvalator(tag string, conf any) bool {
	switch tag {
	case "SSHconfig":
		return genericChecker(conf, CheckSSHconfig)
	case "TelnetConfig":
		return genericChecker(conf, CheckTelnetConfig)
	case "NTPconfig":
		return genericChecker(conf, CheckNTPconfig)
	case "DNSconfig":
		return genericChecker(conf, CheckDNSconfig)
	default:
		return false
	}
}

func (cm *ConfigMaintainer) Init() {
	if cm.initiated.Load() {
		return
	}
	cm.initiated.Store(true)
	cm.updateCallback = make([]func(), 0)
}

// TO-Evaluate: Currently we don't have callback function for unregistering
func (cm *ConfigMaintainer) Regist(f func()) {
	if !cm.initiated.Load() {
		return
	}
	cm.blockedSign.Lock()
	cm.updateCallback = append(cm.updateCallback, f)
	cm.blockedSign.Unlock()
}

func (cm *ConfigMaintainer) UpdateConfig() {
	if !cm.initiated.Load() {
		return
	}
	for _, fn := range cm.updateCallback {
		/* run each callback function in different go routines */
		if fn == nil {
			continue
		}
		go fn()
	}
}

func (cm *ConfigMaintainer) SelfDestroy() {
	cm.initiated.Store(false)
	cm.updateCallback = nil
}
