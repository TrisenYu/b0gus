/// Last modified at 2025/11/15 星期六 22:22:42
package main

import (
	"fmt"
	"path/filepath"

	postgres "github.com/go-pg/pg/v10"
	postgres_orm "github.com/go-pg/pg/v10/orm"
	logrus "github.com/sirupsen/logrus"
)

var logger = logrus.New()

func main() {
	config_path, err := filepath.Abs("./configs/config.toml")
	if err != nil {
		logger.Info("Failed to get absolute path of config file")
		return
	}
	config_content := TomlConfigReader(config_path)
	logger.WithFields(
		logrus.Fields{
			"config_path":    config_path,
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
	).Info("current configuration:\n")
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
	var rowdef AttackerRowDef
	err = db.
		Model(&rowdef).
		CreateTable(&postgres_orm.CreateTableOptions{
			IfNotExists:   true,
			Temp:          false,
			FKConstraints: false,
		})
	if err != nil {
		logger.
			WithField("err", err.Error()).
			Error("Failed to create table in database:\n")
		return
	}
}
