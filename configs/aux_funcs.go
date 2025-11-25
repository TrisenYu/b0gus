package configs

import (

	// "path"
	// "path/filepath"
	// "runtime"
	// "strings"
	"fmt"
	"net"
	"path/filepath"
	"strings"

	gorm_pg "gorm.io/driver/postgres" // pg stands for PostGreSQL
	gorm_sqlite "gorm.io/driver/sqlite"
	gorm "gorm.io/gorm"

	fsnotify "github.com/fsnotify/fsnotify"
	viper "github.com/spf13/viper"
)

func SelectDatabaseBackend(db_conf DatabaseConfig) (*gorm.DB, string, error) {
	switch strings.ToLower(db_conf.DatabaseType) {
	case "postgresql":
		db_addr := db_conf.DatabaseAddr
		if net.ParseIP(db_addr) == nil && strings.ToLower(db_addr) != "localhost" {
			Logger.WithField("database addr", db_addr).
				Fatal("Invalid database address was gained from configuration!")
		}
		pg_db_config := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s search_path=public",
			db_addr, db_conf.DatabasePort, db_conf.DatabaseAdminName,
			db_conf.DatabaseAdminPassword, db_conf.DatabaseName,
		)
		db, err := gorm.Open(gorm_pg.Open(pg_db_config), &gorm.Config{})
		return db, "postgresql", err
	case "sqlite":
		abs_assets_dir_path, _ := filepath.Abs(Assets_dir_as_str)
		sqlite_path := db_conf.DatabasePath
		sqlite_path = filepath.Join(abs_assets_dir_path, sqlite_path)
		Logger.Info(sqlite_path)
		db, err := gorm.Open(gorm_sqlite.Open(sqlite_path), &gorm.Config{})
		return db, "sqlite", err
	case "mongodb":
		fallthrough
	case "redis":
		fallthrough
	default:
		Logger.Errorf(
			`Unknown and unsupported database type was found, 
only support PostgreSQL and SQLite at present...database you selecte: %v`,
			db_conf.DatabaseType,
		)
		return nil, "", fmt.Errorf("unsupported database type<%v> was found", db_conf.DatabaseType)
	}

}

// func pathChecker() string {
// 	exePath, err := os.Executable()
// 	if err != nil {
// 		Logger.Fatal(err.Error())
// 	}
// 	res, _ := filepath.EvalSymlinks(filepath.Dir(exePath))
// 	dir := os.Getenv("TEMP")
// 	if dir == "" {
// 		dir = os.Getenv("TMP")
// 	}
// 	ans, _ := filepath.EvalSymlinks(dir)

// 	if strings.Contains(res, ans) {
// 		var abs_path string
// 		_, filename, _, ok := runtime.Caller(0)
// 		if ok {
// 			abs_path = path.Dir(filename)
// 		}
// 		return abs_path
// 	}
// 	return res
// }

func CheckSSHconfig(ssh_conf *SSHconfig) bool {
	// ssh_conf.ListenAddr might be localhost, which can not be accepted by net.ParseIP
	if ssh_conf == nil || ssh_conf.ListenAddr == "" || ssh_conf.ListenPort <= 1024 {
		return false
	}
	// At this moment, we can only check whether those domains are null
	if ssh_conf.MaxClientNum == 0 {
		ssh_conf.MaxClientNum = 1
	} else if ssh_conf.ClientConnTimeout == 0 {
		ssh_conf.ClientConnTimeout = 30
	} else if ssh_conf.ResponseType == "" {
		ssh_conf.ResponseType = "Always-Reject"
	}
	return true
}

func inspectConfig(conf_data *LocalConfig) bool {
	var flag bool = conf_data.ServerConfig.SSH == SSHconfig{}
	return !(flag || conf_data.ServerConfig.Telnet == TelnetConfig{})
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
			curr_config = tmp_conf
			UpdateFlag <- struct{}{}
		}
	})
	viper.WatchConfig()
	err = viper.Unmarshal(&curr_config)
	if err != nil {
		Logger.Fatalf("Unmarshal config failed: %v", err)
	}
	return &curr_config
}

func (cm *ConfigMaintainer) Init() {
	if cm.initiated.Load() {
		return
	}
	cm.initiated.Store(true)
	cm.blockedSign = make(chan struct{}, 1)
	cm.updateCallback = make([]func(), 0)
}

// Currently it seems that we don't have to unregister callback functions
func (cm *ConfigMaintainer) Regist(f func()) {
	if !cm.initiated.Load() {
		return
	}
	cm.blockedSign <- struct{}{}
	cm.updateCallback = append(cm.updateCallback, f)
	<-cm.blockedSign
}

func (cm *ConfigMaintainer) UpdateConfig() {
	if !cm.initiated.Load() {
		return
	}
	for _, fn := range cm.updateCallback {
		// run each callback function in different go routine
		if fn == nil {
			continue
		}
		go fn()
	}
}

func (cm *ConfigMaintainer) SelfDestroy() {
	cm.initiated.Store(false)
	close(cm.blockedSign)
	cm.updateCallback = nil
}
