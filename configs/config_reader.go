// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 20:05:01
// Last modified at 2025/11/15 星期六 22:24:47
package configs

import (
	"io"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	Config_dir_as_str  string = "./configs/"
	Assets_dir_as_str  string = "./assets/"
	Config_path_as_str string = Config_dir_as_str + "config.toml"
)

type LocalConfig struct {
	// This field should refer the toml config defined in configs/config.toml
	ServerConfig struct {
		PemName string `toml:"pem_name"`

		SSH struct {
			ListenAddr        string `toml:"listen_addr"`
			ListenPort        uint16 `toml:"listen_port"`
			MaxClientNum      uint32 `toml:"max_client_num"`
			ClientConnTimeout uint32 `toml:"client_conn_timeout"`
			PermitLogin       bool   `toml:"permit_login"`
			EmptyShell        bool   `toml:"empty_shell"`
			ResponseType      string `toml:"response_type"`
		} `toml:"ssh"`

		Telnet struct {
			ListenAddr string `toml:"listen_addr"`
			ListenPort uint16 `toml:"listen_port"`
		} `toml:"telnet"`

		Database struct {
			DatabaseType          string `toml:"database_type"`
			DatabaseName          string `toml:"database_name"`
			DatabasePath          string `toml:"database_path"`
			DatabaseAddr          string `toml:"database_addr"`
			DatabasePort          uint16 `toml:"database_port"`
			DatabaseAdminName     string `toml:"database_admin_name"`
			DatabaseAdminPassword string `toml:"database_admin_password"`
		} `toml:"database"`
	} `toml:"server_config"`
}

/*
Read config from given toml file

	inParam: abs_path, absolute path to configuration
	Return:  LocalConfig
*/
func TomlConfigReader(abs_path string) LocalConfig {
	file, err := os.Open(abs_path)
	if err != nil {
		Logger.Fatal(string(err.Error()))
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		Logger.Fatal(string(err.Error()))
	}
	if len(bytes) == 0 {
		Logger.Fatal("Empty config file!")
	}
	var curr_config LocalConfig
	err = toml.Unmarshal(bytes, &curr_config)
	if err != nil {
		Logger.Fatal(string(err.Error()))
	}
	return curr_config
}
