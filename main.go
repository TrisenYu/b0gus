// / Last modified at 2025/11/15 星期六 22:22:42
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	postgres "github.com/go-pg/pg/v10"
	postgres_orm "github.com/go-pg/pg/v10/orm"
	logrus "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var logger = logrus.New()

/*- bogus ssh server.
 * inParam: conf_path string; absolute path to configuration file
 * 		   config_content *LocalConfig; pointer to LocalConfig object
 * 		   db *postgres.DB; pointer to connected database object
 */
func b0gus_ssh_server(conf_path string, config_content *LocalConfig, db *postgres.DB) {
	// Fill SSH Server Configuration from config file
	ssh_server_conf := SSHserverConf{
		Addr:              config_content.ServerConfig.ListenAddr,
		Port:              config_content.ServerConfig.ListenPort,
		MaxClientNum:      config_content.ServerConfig.MaxClientNum,
		ClientConnTimeout: time.Duration(config_content.ServerConfig.ClientConnTimeout) * time.Second,
	}
	// elliptic.P256, elliptic.P384, or elliptic.P521
	// yet provide as configuration

	var host_key ssh.Signer
	pem_path, _ := filepath.Abs(filepath.Join(conf_path, config_content.ServerConfig.PemName))
	pem_obj, err := LoadHostPem(pem_path)
	if err != nil {
		// Consider generate a new host key when PEM file is invalid/corrupted or not exists
		// TODO: also make this configurable
		logger.WithField("Err", err).Warn("Pem seems to be invalid or unsupported")
		host_pem, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			logger.Error("Failed to generate host private key!")
			return
		}
		host_key, err = ssh.NewSignerFromKey(host_pem)
		if err != nil {
			logger.Error("Failed to set private key for ssh!")
			return
		}
		pem_file, err := os.Create(pem_path)
		if err != nil {
			logger.WithField("pem_path:", pem_path).
				Error("Failed to create pem file!\n")
			return
		}
		host_pem_bytes, err := x509.MarshalPKCS8PrivateKey(host_pem)
		if err != nil {
			logger.Error("Failed to marshal pem bytes!\n")
			return
		}
		host_pem_block := pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: host_pem_bytes,
		}
		err = pem.Encode(pem_file, &host_pem_block)
		if err != nil {
			logger.Error("Failed to encode pem bytes into pem file!\n")
			return
		}
		pem_file.Close()
	} else {
		host_key = pem_obj
	}

	ssh_server_conf.SSHMaliciousClientHandler(host_key)
}

type serviceHandler func(conf_path string, config_content *LocalConfig, db *postgres.DB)

func servicesBrancher(service_name string) serviceHandler {
	// TODO: add branch for different bogus service instead of only SSH server
	switch service_name {
	case "ssh":
		return b0gus_ssh_server
	case "telnet":
		fallthrough
	case "ftp":
		fallthrough
	default:
		logger.WithField("name", service_name).
			Fatal("unsupported protocol service was found:")
		// In fact, Fatal will cease the program by os.exit(1).
		return nil
	}
}

func main() {
	logrus.SetFormatter(
		&logrus.TextFormatter{
			TimestampFormat: time.StampMilli,
		},
	)
	conf_path, err := filepath.Abs(Config_path_as_str)
	if err != nil {
		logger.Info("Failed to get absolute path of config file")
		return
	}
	config_content := TomlConfigReader(conf_path)
	logger.WithFields(
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
			"admin_name":     config_content.ServerConfig.AdminName,
		},
	).Info("Current configuration:\n")
	// TODO: judge the type of database for correctly usage
	// 1. local database running as a service or generating db file for further interaction;
	// 2. an online distributed database cluster

	// Connect to Database and Create Table
	if config_content.ServerConfig.HasDatabaseAdmin {

	}
	db := postgres.Connect(&postgres.Options{
		Addr: fmt.Sprintf(
			"%s:%d",
			config_content.ServerConfig.DatabaseAddr,
			config_content.ServerConfig.DatabasePort,
		),
		User:     config_content.ServerConfig.AdminName,
		Password: config_content.ServerConfig.AdminPass,
		Database: config_content.ServerConfig.DatabaseName,
	})
	defer db.Close()
	var (
		infoDef AttackerInfoDef
		cmdDef  AttackerCmdDef
	)
	err = db.Model(&infoDef).
		CreateTable(&postgres_orm.CreateTableOptions{
			IfNotExists:   true,
			Temp:          false,
			FKConstraints: false,
		})
	if err != nil {
		logger.WithField("err", err.Error()).
			Error("Failed to create AttackerInfoTable in database due to ")
		return
	}
	err = db.Model(&cmdDef).
		CreateTable(&postgres_orm.CreateTableOptions{
			IfNotExists:   true,
			Temp:          false,
			FKConstraints: true,
		})
	if err != nil {
		logger.WithField("err:", err.Error()).
			Error("Failed to create AttackerCmdTable in database due to ")
		return
	}
	conf_dir_str, _ := filepath.Abs(Config_dir_as_str)
	servicesBrancher(config_content.ServerConfig.ServiceName)(conf_dir_str, &config_content, db)
}
