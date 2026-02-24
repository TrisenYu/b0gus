// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package configs

import (
	"sync"
	"sync/atomic"
)

type SSHconfig struct {
	ListenAddr        string `toml:"listen_addr" mapstructure:"listen_addr"`
	ResponseType      string `toml:"response_type" mapstructure:"response_type"`
	LoginBanner       string `toml:"login_banner" mapstructure:"login_banner"`
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

type DNSconfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`
}

// TODO: database here means the recording database for attacker features
//
//	If we want database as a masquerading honeypot, then the name of this datatype needs altering
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
		PemName  string `toml:"pem_name" mapstructure:"pem_name"`
		PemType  string `toml:"pem_type" mapstructure:"pem_type"`
		PemLen   uint64 `toml:"pem_len" mapstructure:"pem_len"`
		Language string `toml:"language" mapstructure:"language"`
		// services
		// The reason why to use struct name as ServerConfig's member name is
		// the iteration in `services_man.go` upon struct for data/control path needs refect
		SSHconfig      SSHconfig      `toml:"ssh" mapstructure:"ssh"`
		TelnetConfig   TelnetConfig   `toml:"telnet" mapstructure:"telnet"`
		NTPconfig      NTPconfig      `toml:"ntp"  mapstructure:"ntp"`
		DNSconfig      DNSconfig      `toml:"dns" mapstructure:"dns"`
		DatabaseConfig DatabaseConfig `toml:"database"` // currently used for local recording
	} `toml:"server_config" mapstructure:"server_config"`
}

type AbsServType interface {
	SSHconfig | TelnetConfig | NTPconfig | DNSconfig | DatabaseConfig | any | struct{}
}

type ConfigMaintainer struct {
	updateCallback []func()
	blockedSign    sync.Mutex
	initiated      atomic.Bool
}
