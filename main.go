// Last modified at 2025/11/15 星期六 22:22:42
package main

// PG stands for PostGreSQL
import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	// pg "github.com/go-pg/pg/v10"
	// pg_orm "github.com/go-pg/pg/v10/orm"
	logrus "github.com/sirupsen/logrus"
	ssh "golang.org/x/crypto/ssh"
	gorm_pg "gorm.io/driver/postgres"
	gorm_sqlite "gorm.io/driver/sqlite"
	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_datatypes "b0gus/datatypes"
	b0gus_services "b0gus/services"
)

/*- bogus ssh server.
 * inParam: conf_path string; absolute path to configuration file
 * 		   config_content *b0gus_config.LocalConfig; pointer to LocalConfig object
 * 		   db *gorm.DB; pointer to connected database object
 */
func b0gusSSHserver(
	conf_path string,
	config_content *b0gus_config.LocalConfig,
	db *gorm.DB,
) {
	// ssh-related tables
	db.AutoMigrate(
		&b0gus_datatypes.AttackerAddrDef{}, &b0gus_datatypes.AttackerPortInfoDef{},
		&b0gus_datatypes.AttackerPubInfoDef{}, &b0gus_datatypes.AttackerPassInfoDef{},
		&b0gus_datatypes.AttackerCmdDef{},
	)

	ssh_conf_obj := config_content.ServerConfig
	// Fill SSH Server Configuration from config file
	ssh_server_conf := b0gus_services.SSHserverConf{
		Addr:              ssh_conf_obj.ListenAddr,
		Port:              ssh_conf_obj.ListenPort,
		MaxClientNum:      ssh_conf_obj.MaxClientNum,
		ClientConnTimeout: time.Duration(ssh_conf_obj.ClientConnTimeout) * time.Second,
		DB_fd:             db,
	}
	var host_key ssh.Signer
	pem_path, _ := filepath.Abs(filepath.Join(conf_path, ssh_conf_obj.PemName))
	pem_obj, err := b0gus_crypto_aux.LoadHostPem(pem_path)
	if err != nil {
		b0gus_config.Logger.
			WithField("Err", err).
			Warn("Pem seems to be invalid or unsupported")
		host_key, err = b0gus_crypto_aux.CreatePem(pem_path)
		if err != nil {
			b0gus_config.Logger.
				WithFields(logrus.Fields{
					"Err":  err,
					"path": pem_path,
				}).
				Error("Unable to create pem")
			return
		}
	} else {
		host_key = pem_obj
	}

	ssh_server_conf.SSHMaliciousClientHandler(host_key)
}

type serviceHandleFunc func(
	conf_path string,
	config_content *b0gus_config.LocalConfig,
	db *gorm.DB,
)

func servicesBrancher(service_name string) serviceHandleFunc {
	// TODO: add branch for different bogus service instead of only SSH server
	switch service_name {
	case "ssh":
		return b0gusSSHserver
	case "telnet":
		fallthrough
	case "ftp":
		fallthrough
	default:
		b0gus_config.Logger.WithField("name", service_name).
			Fatal("unsupported protocol service was found:")
		// In fact, Fatal will cease the program by executing os.exit(1).
		return nil
	}
}

/*- The entry of b0gus
 * configuration in `./configs/` should be set up before executing
 */
func main() {
	conf_path, err := filepath.Abs(b0gus_config.Config_path_as_str)
	if err != nil {
		b0gus_config.Logger.Info("Failed to get absolute path of config file")
		return
	}
	conf_dir_str, _ := filepath.Abs(b0gus_config.Config_dir_as_str)
	config_content := b0gus_config.TomlConfigReader(conf_path)
	b0gus_config.Logger.WithFields(
		logrus.Fields{
			"config_path":    conf_path,
			"server_addr":    config_content.ServerConfig.ListenAddr,
			"server_port":    config_content.ServerConfig.ListenPort,
			"service_name":   config_content.ServerConfig.ServiceName,
			"max_client_num": config_content.ServerConfig.MaxClientNum,
			"permit_login":   config_content.ServerConfig.PermitLogin,
			"database_path":  config_content.ServerConfig.DatabasePath,
			"database_addr":  config_content.ServerConfig.DatabaseAddr,
			"database_port":  config_content.ServerConfig.DatabasePort,
			"database_name":  config_content.ServerConfig.DatabaseName,
			"admin_name":     config_content.ServerConfig.DatabaseAdminName,
		},
	).Info("Current configuration:\n")
	var db *gorm.DB = nil
	// Connect to Database and Create Table
	switch strings.ToLower(config_content.ServerConfig.DatabaseType) {
	case "postgresql":
		db_addr := config_content.ServerConfig.DatabaseAddr
		if net.ParseIP(db_addr) == nil && strings.ToLower(db_addr) != "localhost" {
			b0gus_config.Logger.
				WithField("database addr", db_addr).
				Fatal("invalid database address was gained from configuration!")
		}
		pg_db_config := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s",
			// sslmode=disable TimeZone=Asia/Shanghai
			db_addr, config_content.ServerConfig.DatabasePort,
			config_content.ServerConfig.DatabaseAdminName,
			config_content.ServerConfig.DatabaseAdminPassword,
			config_content.ServerConfig.DatabaseName,
		)
		db, err = gorm.Open(gorm_pg.Open(pg_db_config), &gorm.Config{})
		if err != nil {
			b0gus_config.Logger.WithField("err:", err).
				Error("Failed to connect to PostgreSQL due to ")
			return
		}
	case "sqlite":
		sqlite_path := config_content.ServerConfig.DatabasePath
		if _, err := os.Stat(sqlite_path); err != nil {
			b0gus_config.Logger.
				WithField("err:", err).
				Error("Path of database seems to be invalid, while an error happended")
			return
		}
		db, err = gorm.Open(gorm_sqlite.Open(sqlite_path), &gorm.Config{})
		if err != nil {
			b0gus_config.Logger.WithField("err:", err).
				Error("Failed to connect to SQLlite due to ")
			return
		}

	default:
		b0gus_config.Logger.
			WithField("database", config_content.ServerConfig.DatabaseType).
			Fatal("Unknow and unsupported database type was found")
	}

	if db == nil {
		b0gus_config.Logger.Fatal("Empty database file descriptor")
	}

	servicesBrancher(config_content.ServerConfig.ServiceName)(
		conf_dir_str, &config_content, db,
	)
}
