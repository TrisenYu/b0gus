// Package configs
package configs

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"context"
	"sync"
	"sync/atomic"
)

// services structures' definitions
type genericConf struct {
	MaxClientNum      uint32 `toml:"max_client_num" mapstructure:"max_client_num"`
	ClientConnTimeout uint32 `toml:"client_conn_timeout" mapstructure:"client_conn_timeout"`
	ListenPort        uint16 `toml:"listen_port" mapstructure:"listen_port"`
}
type SSHconfig struct {
	genericConf   `toml:",inline" mapstructure:",squash"`
	LoginBanner   string `toml:"login_banner" mapstructure:"login_banner"`
	ResponseType  string `toml:"response_type" mapstructure:"response_type"`
	HashAlgorithm string `toml:"hash_algorithm" mapstructure:"hash_algorithm"`
	PemName       string `toml:"pem_name" mapstructure:"pem_name"`
	PemType       string `toml:"pem_type" mapstructure:"pem_type"`
	PemLen        uint64 `toml:"pem_len" mapstructure:"pem_len"`
	MaxAuthTries  uint32 `toml:"max_auth_tries" mapstructure:"max_auth_tries"`
	PermitLogin   bool   `toml:"permit_login" mapstructure:"permit_login"`

	fn func(conf *SSHconfig, scc *ServConcurrentCtrl, db *RuntimeDB, args ...any)
}

//type TelnetConfig struct {
//	ListenAddr string `toml:"listen_addr" mapstructure:"listen_addr"`
//	ListenPort uint16 `toml:"listen_port" mapstructure:"listen_port"`
//
//	fn func(conf *TelnetConfig, scc *ServConcurrentCtrl, db *RuntimeDB, args ...any)
//}

type NTPconfig struct {
	genericConf `toml:",inline" mapstructure:",squash"`
	ListenAddr  string `toml:"listen_addr" mapstructure:"listen_addr"`
	// CurrZone change the real zone to the fake one for specific effects
	CurrZone string `toml:"curr_zone" mapstructure:"curr_zone"`

	fn func(conf *NTPconfig, scc *ServConcurrentCtrl, db *RuntimeDB, args ...any)
}

type DNSconfig struct {
	genericConf `toml:",inline" mapstructure:",squash"`
	ListenAddr  string `toml:"listen_addr" mapstructure:"listen_addr"`
	DefaultTTL  uint32 `toml:"default_ttl" mapstructure:"default_ttl"`

	fn func(conf *DNSconfig, scc *ServConcurrentCtrl, db *RuntimeDB, args ...any)
}

type SMTPconfig struct {
	genericConf      `toml:",inline" mapstructure:",squash"`
	ListenAddr       string `toml:"listen_addr" mapstructure:"listen_addr"`
	LocalTLSCertPath string `toml:"local_tls_cert_path" mapstructure:"local_tls_cert_path"`
	LocalTLSKeyPath  string `toml:"local_tls_key_path" mapstructure:"local_tls_key_path"`
	LocalSaveDir     string `toml:"local_save_dir" mapstructure:"local_save_dir"`
	AuthRequired     bool   `toml:"auth_required" mapstructure:"auth_required"`

	fn func(conf *SMTPconfig, scc *ServConcurrentCtrl, db *RuntimeDB, args ...any)
}

type FakeDBconf struct {
	genericConf `toml:",inline" mapstructure:",squash"`
	ListenAddr  string `toml:"listen_addr" mapstructure:"listen_addr"`

	fn func(conf *FakeDBconf, scc *ServConcurrentCtrl, db *RuntimeDB, args ...any)
}

// RecDBConfig here means the recording database for attacker features
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

// TODO: push configuration from networking binary streams

type LocalConfig struct {
	// ServerConfig should refer to the toml config defined in configs/config.toml
	ServerConfig struct {
		//PemName  string `toml:"pem_name" mapstructure:"pem_name"`
		//PemType  string `toml:"pem_type" mapstructure:"pem_type"`
		//PemLen   uint64 `toml:"pem_len" mapstructure:"pem_len"`
		Language string `toml:"language" mapstructure:"language"`

		// fields defined for services
		// The reason why to use struct name as ServerConfig's member name is
		// the iteration in `services_man.go` upon struct for data/control path needs refect

		// TelnetConfig TelnetConfig `toml:"telnet" mapstructure:"telnet"`

		SSHconfig  SSHconfig  `toml:"ssh" mapstructure:"ssh"`
		NTPconfig  NTPconfig  `toml:"ntp"  mapstructure:"ntp"`
		DNSconfig  DNSconfig  `toml:"dns" mapstructure:"dns"`
		SMTPconfig SMTPconfig `toml:"smtp" mapstructure:"smtp"`
		// currently used for locally recording
		RecDBConfig RecDBConfig `toml:"rec_db_config" mapstructure:"rec_db_config"`
	} `toml:"server_config" mapstructure:"server_config"`
}

type AbsServType interface {
	// InvokeRunner is registered by specific AbsServType defined before
	InvokeRunner(*ServConcurrentCtrl, *RuntimeDB, ...any)
}

type ServConcurrentCtrl struct {
	Ctx    context.Context
	DataCh <-chan any
}

type AbsServFunc[T AbsServType] func(
	conf *T,
	scc *ServConcurrentCtrl,
	db *RuntimeDB,
	args ...any,
)

func (s SSHconfig) InvokeRunner(
	scc *ServConcurrentCtrl,
	db *RuntimeDB,
	args ...any,
) {
	if s.fn == nil {
		return
	}
	// [warn]: ... must pass to args otherwise type casting will fail
	s.fn(&s, scc, db, args...)
}

//func (t TelnetConfig) InvokeRunner(
//	scc *ServConcurrentCtrl,
//	db *RuntimeDB,
//	args ...any,
//) {
//	if t.fn == nil {
//		return
//	}
//	// [warn]: ... must pass to args otherwise type casting will fail
//	t.fn(&t, scc, db, args...)
//}

func (n NTPconfig) InvokeRunner(
	scc *ServConcurrentCtrl,
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
	scc *ServConcurrentCtrl,
	db *RuntimeDB,
	args ...any,
) {
	if d.fn == nil {
		return
	}
	// [warn]: ... must pass to args otherwise type casting will fail
	d.fn(&d, scc, db, args...)
}

func (s SMTPconfig) InvokeRunner(
	scc *ServConcurrentCtrl,
	db *RuntimeDB,
	args ...any) {
	if s.fn == nil {
		return
	}
	s.fn(&s, scc, db, args...)
}

// func (t *TelnetConfig) RegisterRunner(fn AbsServFunc[TelnetConfig]) { t.fn = fn }

func (s *SSHconfig) RegisterRunner(fn AbsServFunc[SSHconfig])   { s.fn = fn }
func (n *NTPconfig) RegisterRunner(fn AbsServFunc[NTPconfig])   { n.fn = fn }
func (d *DNSconfig) RegisterRunner(fn AbsServFunc[DNSconfig])   { d.fn = fn }
func (s *SMTPconfig) RegisterRunner(fn AbsServFunc[SMTPconfig]) { s.fn = fn }

// Naive check

func CheckSSHconfig(sshConf *SSHconfig) bool {
	// sshConf.ListenAddr might be localhost, which can not be accepted by net.ParseIP
	if sshConf == nil || sshConf.ListenPort <= 1024 {
		return false
	}
	/* At this moment and at most, we can only check whether those fields are null */
	if sshConf.MaxClientNum == 0 {
		sshConf.MaxClientNum = 1
	} else if sshConf.ClientConnTimeout == 0 {
		sshConf.ClientConnTimeout = 30
	} else if sshConf.ResponseType == "" {
		sshConf.ResponseType = "Always-Reject"
	}
	return true
}

//func CheckTelnetConfig(telnetConf *TelnetConfig) bool {
//	return telnetConf != nil &&
//		telnetConf.ListenAddr != "" &&
//		telnetConf.ListenPort > 1024
//}

func CheckNTPconfig(ntpConf *NTPconfig) bool {
	return ntpConf != nil && len(ntpConf.ListenAddr) != 0 &&
		ntpConf.ListenPort > 1024
}

func CheckDNSconfig(dnsConf *DNSconfig) bool {
	return dnsConf != nil && len(dnsConf.ListenAddr) != 0 &&
		dnsConf.ListenPort > 1024
}

func CheckSMTPconfig(smtpConf *SMTPconfig) bool {
	return smtpConf != nil && len(smtpConf.ListenAddr) != 0 &&
		smtpConf.ListenPort > 1024
}

// GenericConfChecker
//
//	in_func uses `CheckXXXconfig` defined in current source file.
func GenericConfChecker[T AbsServType](
	inConf any,
	inFunc func(*T) bool,
) bool {
	curr, ok := inConf.(T)
	if !ok {
		return false
	}
	return inFunc(&curr)
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

// Register records any subscriber wants to monitor the change of configuration
//
//	[TO-Evaluate]: Currently we don't have callback function for unregistering
func (cm *ConfigMaintainer) Register(f func(*LocalConfig)) {
	if !cm.initiated.Load() {
		return
	}
	cm.blockedSign.Lock()
	cm.updateCallback = append(cm.updateCallback, f)
	cm.blockedSign.Unlock()
}

// UpdateConfig will execute functions provided by subscribers when detecting any change of configuration
func (cm *ConfigMaintainer) UpdateConfig(updConf *LocalConfig) {
	if !cm.initiated.Load() {
		return
	}
	for _, fn := range cm.updateCallback {
		/* run each callback function in different go routines */
		if fn == nil {
			continue
		}
		go fn(updConf)
	}
}

func (cm *ConfigMaintainer) SelfDestroy() {
	cm.initiated.Store(false)
	cm.updateCallback = nil
}
