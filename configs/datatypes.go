// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package configs

import (
	"context"
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

	fn func(conf *SSHconfig, scc *ServicesConcurrencyCtrl, db *RuntimeDB, args ...any)
}

type TelnetConfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`

	fn func(conf *TelnetConfig, scc *ServicesConcurrencyCtrl, db *RuntimeDB, args ...any)
}

type NTPconfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`

	fn func(conf *NTPconfig, scc *ServicesConcurrencyCtrl, db *RuntimeDB, args ...any)
}

type DNSconfig struct {
	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`

	fn func(conf *DNSconfig, scc *ServicesConcurrencyCtrl, db *RuntimeDB, args ...any)
}

// RecDB here means the recording database for attacker features
// not as a bogus service.
type RecDBConfig struct {
	Type          string `toml:"type" mapstructure:"type"`
	Name          string `toml:"name" mapstructure:"name"`
	Path          string `toml:"path" mapstructure:"path"`
	Addr          string `toml:"addr" mapstructure:"addr"`
	AdminName     string `toml:"admin_name" mapstructure:"admin_name"`
	AdminPassword string `toml:"admin_password" mapstructure:"admin_password"`
	Port          uint16 `toml:"port" mapstructure:"port"`
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
		SSHconfig    SSHconfig    `toml:"ssh" mapstructure:"ssh"`
		TelnetConfig TelnetConfig `toml:"telnet" mapstructure:"telnet"`
		NTPconfig    NTPconfig    `toml:"ntp"  mapstructure:"ntp"`
		DNSconfig    DNSconfig    `toml:"dns" mapstructure:"dns"`
		RecDBConfig  RecDBConfig  `toml:"rec_db_config" mapstructure:"rec_db_config"`
		// currently used for local recording
	} `toml:"server_config" mapstructure:"server_config"`
}

type AbsServType interface {
	SSHconfig | TelnetConfig | NTPconfig | DNSconfig | any | struct{}

	// Invoke runner which be registed by specific AbsServType before
	InvokeRunner(*ServicesConcurrencyCtrl, *RuntimeDB, ...any)
}

type ServicesConcurrencyCtrl struct {
	Ctx     context.Context
	Data_ch <-chan any
}

type AbsServFunc[T AbsServType] func(
	conf *T,
	scc *ServicesConcurrencyCtrl,
	db *RuntimeDB,
	args ...any,
)

type ServicesEntry[T AbsServType] interface {
	// Regist Runner via outer function
	RegistRunner(outer_serv_fn AbsServFunc[T])
}

func (s SSHconfig) InvokeRunner(
	scc *ServicesConcurrencyCtrl,
	db *RuntimeDB,
	args ...any,
) {
	if s.fn == nil {
		return
	}
	// [warn]: ... must pass to args otherwise type casting will fail
	s.fn(&s, scc, db, args...)
}
func (t TelnetConfig) InvokeRunner(
	scc *ServicesConcurrencyCtrl,
	db *RuntimeDB,
	args ...any,
) {
	if t.fn == nil {
		return
	}
	// [warn]: ... must pass to args otherwise type casting will fail
	t.fn(&t, scc, db, args...)
}
func (n NTPconfig) InvokeRunner(
	scc *ServicesConcurrencyCtrl,
	db *RuntimeDB,
	args ...any,
) {
	if n.fn == nil {
		return
	}
	// [warn]: ... must pass to args otherwise type casting will fail
	n.fn(&n, scc, db, args...)
}
func (d DNSconfig) InvokeRunner(
	scc *ServicesConcurrencyCtrl,
	db *RuntimeDB,
	args ...any,
) {
	if d.fn == nil {
		return
	}
	// [warn]: ... must pass to args otherwise type casting will fail
	d.fn(&d, scc, db, args...)
}

func (s *SSHconfig) RegistRunner(fn AbsServFunc[SSHconfig])       { s.fn = fn }
func (t *TelnetConfig) RegistRunner(fn AbsServFunc[TelnetConfig]) { t.fn = fn }
func (n *NTPconfig) RegistRunner(fn AbsServFunc[NTPconfig])       { n.fn = fn }
func (d *DNSconfig) RegistRunner(fn AbsServFunc[DNSconfig])       { d.fn = fn }

// Naive check
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

// generic configuration checker
//
//	in_func uses `CheckXXXconfig` defined in current source file.
func GenericConfChecker[T AbsServType](
	in_conf any,
	in_func func(*T) bool,
) bool {
	curr, ok := in_conf.(T)
	if !ok {
		return false
	}
	return in_func(&curr)
}

type ConfigMaintainer struct {
	updateCallback []func(*LocalConfig)
	blockedSign    sync.Mutex
	initiated      atomic.Bool
}

func (cm *ConfigMaintainer) Init() {
	if cm.initiated.Load() {
		return
	}
	cm.initiated.Store(true)
	cm.updateCallback = make([]func(*LocalConfig), 0)
}

// TO-Evaluate: Currently we don't have callback function for unregistering
func (cm *ConfigMaintainer) Regist(f func(*LocalConfig)) {
	if !cm.initiated.Load() {
		return
	}
	cm.blockedSign.Lock()
	cm.updateCallback = append(cm.updateCallback, f)
	cm.blockedSign.Unlock()
}

func (cm *ConfigMaintainer) UpdateConfig(updated_conf *LocalConfig) {
	if !cm.initiated.Load() {
		return
	}
	for _, fn := range cm.updateCallback {
		/* run each callback function in different go routines */
		if fn == nil {
			continue
		}
		go fn(updated_conf)
	}
}

func (cm *ConfigMaintainer) SelfDestroy() {
	cm.initiated.Store(false)
	cm.updateCallback = nil
}
