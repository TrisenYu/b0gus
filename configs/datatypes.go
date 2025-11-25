package configs

import "sync/atomic"

type SSHconfig struct {
	ListenAddr        string `toml:"listen_addr" mapstructure:"listen_addr"`
	ResponseType      string `toml:"response_type" mapstructure:"response_type"`
	MaxClientNum      uint32 `toml:"max_client_num" mapstructure:"max_client_num"`
	ClientConnTimeout uint32 `toml:"client_conn_timeout" mapstructure:"client_conn_timeout"`
	ListenPort        uint16 `toml:"listen_port" mapstructure:"listen_port"`
	PermitLogin       bool   `toml:"permit_login" mapstructure:"permit_login"`
	EmptyShell        bool   `toml:"empty_shell" mapstructure:"empty_shell"`
}

type TelnetConfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`
}

type NTPconfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`
}

type DatabaseConfig struct {
	DatabaseType          string `toml:"database_type" mapstructure:"database_type"`
	DatabaseName          string `toml:"database_name" mapstructure:"database_name"`
	DatabasePath          string `toml:"database_path" mapstructure:"database_path"`
	DatabaseAddr          string `toml:"database_addr" mapstructure:"database_addr"`
	DatabaseAdminName     string `toml:"database_admin_name" mapstructure:"database_admin_name"`
	DatabaseAdminPassword string `toml:"database_admin_password" mapstructure:"database_admin_password"`
	DatabasePort          uint16 `toml:"database_port" mapstructure:"database_port"`
}

type LocalConfig struct {
	// This field should refer the toml config defined in configs/config.toml
	ServerConfig struct {
		PemName  string         `toml:"pem_name" mapstructure:"pem_name"`
		SSH      SSHconfig      `toml:"ssh" mapstructure:"ssh"`
		Telnet   TelnetConfig   `toml:"telnet" mapstructure:"telnet"`
		NTP      NTPconfig      `toml:"ntp"  mapstructure:"ntp"`
		Database DatabaseConfig `toml:"database"`
	} `toml:"server_config" mapstructure:"server_config"`
}

type ConfigMaintainer struct {
	updateCallback []func()
	blockedSign    chan struct{}
	initiated      atomic.Bool
}
