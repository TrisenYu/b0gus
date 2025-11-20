// Last modified at 2025/11/15 星期六 22:22:42
package main

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	logrus "github.com/sirupsen/logrus"
	ssh "golang.org/x/crypto/ssh"
	gorm_pg "gorm.io/driver/postgres" // pg stands for PostGreSQL
	gorm_sqlite "gorm.io/driver/sqlite"
	gorm "gorm.io/gorm"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_datatypes "b0gus/datatypes"
	b0gus_misc_utils "b0gus/misc_utils"
	b0gus_services "b0gus/services"
)

// bogus ssh server
//
//	inParam: conf_path [string]; absolute path to configuration file
//	bogus_conf [*b0gus_config.LocalConfig]; pointer to LocalConfig object
//	db [*gorm.DB]; pointer to connected database object
func b0gusSSHserver(
	conf_path string,
	bogus_conf *b0gus_config.LocalConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	defer wait_group.Done()

	// ssh-related tables
	db.AutoMigrate(
		&b0gus_datatypes.AddrInfoDef{},
		&b0gus_datatypes.PortInfoDef{},
		&b0gus_datatypes.UsernameDef{},
		&b0gus_datatypes.SSHClientversionStrDef{},
		&b0gus_datatypes.PubInfoDef{},
		&b0gus_datatypes.PassInfoDef{},
		&b0gus_datatypes.CommandTextDef{},
		// relation table
		&b0gus_datatypes.PortNameRelated{},
		&b0gus_datatypes.PortVerRelated{},
		&b0gus_datatypes.PortPubKeyRelated{},
		&b0gus_datatypes.PortPassRelated{},
		&b0gus_datatypes.PortCmdRelated{},
	)

	ssh_conf_obj := bogus_conf.ServerConfig.SSH
	// Fill SSH Server Configuration with definitions in config file
	ssh_server_conf := b0gus_services.SSHserverConf{
		Addr:              ssh_conf_obj.ListenAddr,
		Port:              ssh_conf_obj.ListenPort,
		MaxClientNum:      ssh_conf_obj.MaxClientNum,
		ClientConnTimeout: time.Duration(ssh_conf_obj.ClientConnTimeout) * time.Second,
		DB_fd:             db,
	}
	var host_key ssh.Signer
	pem_path, _ := filepath.Abs(filepath.Join(conf_path, bogus_conf.ServerConfig.PemName))
	pem_obj, err := b0gus_crypto_aux.LoadHostPem(pem_path)
	if err != nil {
		b0gus_config.Logger.
			WithField("Err", err).
			Warn("Pem seems to be invalid or unsupported, b0gus will create one and store it to the path you assigned")
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

func b0gusTelnetServer(
	conf_path string,
	bogus_conf *b0gus_config.LocalConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	wait_group.Done()
}

func checkTelnetConfig(
	server_conf *b0gus_config.LocalConfig,
	v map[string]any,
) bool {
	return false
}

func checkSSHconfig(
	server_conf *b0gus_config.LocalConfig,
	v map[string]any,
) bool {
	for kk, vv := range v {
		switch kk {
		case "ListenAddr":
			// if so, do not execute
			if vv == "" {
				return false
			}
		case "ListenPort":
			// if so, do not execute
			// there must be a port number but not zero.
			// but since we do not make up using privilege port
			// so just set vv > 1024
			if vv.(uint16) <= 1024 {
				return false
			}
		case "MaxClientNum":
			if vv == 0 {
				(*server_conf).ServerConfig.SSH.MaxClientNum = 1
			}
		case "ClientConnTimeout":
			if vv == 0 {
				(*server_conf).ServerConfig.SSH.ClientConnTimeout = 30
			}
		case "ResponseType":
			if vv == "" {
				(*server_conf).ServerConfig.SSH.ResponseType = "Always-Reject"
			}
		case "PermitLogin":
		case "EmptyShell":
		}
	}
	return true
}

func servicesBrancher(
	conf_path string,
	server_conf *b0gus_config.LocalConfig,
	db *gorm.DB,
	wait_group *sync.WaitGroup,
) {
	// check the port given by service whether is in used
	// if so, b0gus won't running this and instead push such information into log
	conf_mapper := b0gus_misc_utils.TurnStruct2Map(*server_conf)
	if conf_mapper == nil {
		b0gus_config.Logger.
			Fatal("Unable to reflect config to `map[string]any`!")
	}
	server_conf_mapper := conf_mapper["ServerConfig"]

	// we can run go routine in this `for loop`
	for k, v := range server_conf_mapper.(map[string]any) {
		switch k {
		case "PemName":
			fallthrough
		case "Database":
			// do nothing
		case "SSH":
			flag := checkSSHconfig(server_conf, v.(map[string]any))
			if !flag {
				continue
			}
			wait_group.Add(1)
			// run given service in different go routine if the necessary fields are not empty
			go b0gusSSHserver(conf_path, server_conf, db, wait_group)
			// Understanding how `wait_group.Go()` work
		case "Telnet":
			b0gus_config.Logger.Warn("Yet to implement b0gus telnet shell!")
			if !checkTelnetConfig(server_conf, v.(map[string]any)) {
				continue
			}
			wait_group.Add(1)
			go b0gusTelnetServer(conf_path, server_conf, db, wait_group)
		default:
			b0gus_config.Logger.
				Fatal("unknown field in configuration")
				// In fact, Fatal will cease the program by executing os.exit(1).
			return
		}
	}

	wait_group.Wait()
}

/*- The entry of b0gus
 * configuration in `./configs/` should be set up before executing
 */
func main() {
	// TODO: neccessary executable binary files/dependencies imediate check inside b0gus
	conf_path, err := filepath.Abs(b0gus_config.Config_path_as_str)
	if err != nil {
		b0gus_config.Logger.Info("Failed to get absolute path of config file")
		return
	}
	conf_dir_str, _ := filepath.Abs(b0gus_config.Config_dir_as_str)
	bogus_conf := b0gus_config.TomlConfigReader(conf_path)
	b0gus_config.Logger.WithFields(
		logrus.Fields{
			"config_path":        conf_path,
			"ssh_server_conf":    bogus_conf.ServerConfig.SSH,
			"telnet_server_conf": bogus_conf.ServerConfig.Telnet,
		},
	).Info("Current configuration:\n")

	var (
		db     *gorm.DB = nil
		db_str string   = ""
	)

	// **Connect** to Database and Create Table
	switch strings.ToLower(bogus_conf.ServerConfig.Database.DatabaseType) {
	case "postgresql":
		db_addr := bogus_conf.ServerConfig.Database.DatabaseAddr
		if net.ParseIP(db_addr) == nil && strings.ToLower(db_addr) != "localhost" {
			b0gus_config.Logger.
				WithField("database addr", db_addr).
				Fatal("Invalid database address was gained from configuration!")
		}
		pg_db_config := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s",
			db_addr, bogus_conf.ServerConfig.Database.DatabasePort,
			bogus_conf.ServerConfig.Database.DatabaseAdminName,
			bogus_conf.ServerConfig.Database.DatabaseAdminPassword,
			bogus_conf.ServerConfig.Database.DatabaseName,
		)
		db, err = gorm.Open(gorm_pg.Open(pg_db_config), &gorm.Config{})
		db_str = "postgresql"

	case "sqlite":
		abs_assets_dir_path, _ := filepath.Abs(b0gus_config.Assets_dir_as_str)
		sqlite_path := bogus_conf.ServerConfig.Database.DatabasePath
		sqlite_path = filepath.Join(abs_assets_dir_path, sqlite_path)
		b0gus_config.Logger.Info(sqlite_path)
		db, err = gorm.Open(gorm_sqlite.Open(sqlite_path), &gorm.Config{})
		db_str = "sqlite"

	default:
		b0gus_config.Logger.
			WithField(
				"database you selected",
				bogus_conf.ServerConfig.Database.DatabaseType,
			).Fatal(
			"Unknown and unsupported database type was found, only support PostgreSQL and SQLite at present...",
		)
	}
	if err != nil {
		b0gus_config.Logger.Errorf(
			"Failed to connect to %s due to %s",
			db_str, err.Error(),
		)
		return
	} else if db == nil {
		b0gus_config.Logger.Fatal(
			"Empty database file descriptor",
		)
	}
	var wait_group sync.WaitGroup
	servicesBrancher(conf_dir_str, &bogus_conf, db, &wait_group)
}
