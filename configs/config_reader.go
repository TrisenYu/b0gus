// SPDX-LICENSE-IDENTIFIER: GPL2.0
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 20:05:01
// Last modified at 2025/11/15 星期六 22:24:47
package configs

import (
	"io"
	"os"

	// "path"
	// "path/filepath"
	// "runtime"
	// "strings"

	fsnotify "github.com/fsnotify/fsnotify"
	toml "github.com/pelletier/go-toml/v2"
	viper "github.com/spf13/viper"
)

const (
	// since go use compiler rather than interpreter, we can not assign a . as relative path
	// otherwise the executable file will deem there is a configuration in the same direnctory as its,
	Config_dir_as_str  string = "./configs/"
	Assets_dir_as_str  string = "./assets/"
	Config_path_as_str string = Config_dir_as_str + "config.toml"
)

type SSHconfig struct {
	ListenAddr        string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort        uint16 `toml:"listen_port" mapstructure:"listen_port"`
	MaxClientNum      uint32 `toml:"max_client_num" mapstructure:"max_client_num"`
	ClientConnTimeout uint32 `toml:"client_conn_timeout" mapstructure:"client_conn_timeout"`
	PermitLogin       bool   `toml:"permit_login" mapstructure:"permit_login"`
	EmptyShell        bool   `toml:"empty_shell" mapstructure:"empty_shell"`
	ResponseType      string `toml:"response_type" mapstructure:"response_type"`
}

type TelnetConfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`
}

type DatabaseConfig struct {
	DatabaseType          string `toml:"database_type" mapstructure:"database_type"`
	DatabaseName          string `toml:"database_name" mapstructure:"database_name"`
	DatabasePath          string `toml:"database_path" mapstructure:"database_path"`
	DatabaseAddr          string `toml:"database_addr" mapstructure:"database_addr"`
	DatabasePort          uint16 `toml:"database_port" mapstructure:"database_port"`
	DatabaseAdminName     string `toml:"database_admin_name" mapstructure:"database_admin_name"`
	DatabaseAdminPassword string `toml:"database_admin_password" mapstructure:"database_admin_password"`
}

type LocalConfig struct {
	// This field should refer the toml config defined in configs/config.toml
	ServerConfig struct {
		PemName  string         `toml:"pem_name" mapstructure:"pem_name"`
		SSH      SSHconfig      `toml:"ssh" mapstructure:"ssh"`
		Telnet   TelnetConfig   `toml:"telnet" mapstructure:"telnet"`
		Database DatabaseConfig `toml:"database"`
	} `toml:"server_config" mapstructure:"server_config"`
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
	if ssh_conf.MaxClientNum == 0 {
		ssh_conf.MaxClientNum = 1
	} else if ssh_conf.ClientConnTimeout == 0 {
		ssh_conf.ClientConnTimeout = 30
	} else if ssh_conf.ResponseType == "" {
		ssh_conf.ResponseType = "Always-Reject"
	}
	return true
}

func DefaultConfig() *LocalConfig {
	// Logger.Info(pathChecker())
	viper.SetConfigFile(Config_path_as_str)
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		Logger.Fatalf("Read config failed: %v", err)
	}

	var curr_config LocalConfig
	viper.OnConfigChange(func(conf_changes fsnotify.Event) {
		Logger.Infof("Configuration change: %s", conf_changes.Name)
		// TODO: What if the configuration was completely destroyed by accidental operation?
		// should we still update to the last used configuration in this scenario?
		err := viper.Unmarshal(&curr_config)
		if err != nil {
			Logger.Errorf("Reload config failed: %v", err)
			return
		}
		// TODO: now we have to create a distributor to distribute the update information
		UpdateFlag <- struct{}{}
	})
	viper.WatchConfig()
	err = viper.Unmarshal(&curr_config)
	if err != nil {
		Logger.Fatalf("Unmarshal config failed: %v", err)
	}
	return &curr_config
}

/*
(Deprecated)
Read config from given toml file

	inParam: abs_path, absolute path to configuration
	Return:  LocalConfig
*/
func TomlConfigReader(abs_path string) LocalConfig {
	file, err := os.Open(abs_path)
	if err != nil {
		Logger.Fatal(err.Error())
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		Logger.Fatal(err.Error())
	}
	if len(bytes) == 0 {
		Logger.Fatal("Empty config file!")
	}
	var curr_config LocalConfig
	err = toml.Unmarshal(bytes, &curr_config)
	if err != nil {
		Logger.Fatal(err.Error())
	}
	return curr_config
}
