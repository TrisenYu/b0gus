// Package configs
package configs

// Last modified at 2026/03/30 星期一 21:46:46
// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD

import (
	"context"
	"os"
)

// services structures' definitions
type genericConf struct {
	// path to Certificate signed-off by real CA for helping authenticate the communicating entity
	TLSCertPath string `toml:"tls_cert_path" mapstructure:"tls_cert_path"`
	// path to private key of one certificate signed-off by real CA
	TLSKeyPath        string `toml:"tls_key_path" mapstructure:"tls_key_path"`
	MaxClientNum      uint32 `toml:"max_client_num" mapstructure:"max_client_num"`
	ClientConnTimeout uint32 `toml:"client_conn_timeout" mapstructure:"client_conn_timeout"`
	ListenPort        uint16 `toml:"listen_port" mapstructure:"listen_port"`
}
type SSHconfig struct {
	genericConf   `toml:",inline" mapstructure:",squash"`
	LoginBanner   string `toml:"login_banner" mapstructure:"login_banner"`
	ResponseType  string `toml:"response_type" mapstructure:"response_type"`
	HashAlgorithm string `toml:"hash_algorithm" mapstructure:"hash_algorithm"`
	// PemName only set one time after running up the whole server
	PemName string `toml:"pem_name" mapstructure:"pem_name"`
	// PemType only set one time after running up the whole server
	PemType string `toml:"pem_type" mapstructure:"pem_type"`
	// PemLen only set one time after running up the whole server
	PemLen       uint64 `toml:"pem_len" mapstructure:"pem_len"`
	MaxAuthTries uint32 `toml:"max_auth_tries" mapstructure:"max_auth_tries"`
	PermitLogin  bool   `toml:"permit_login" mapstructure:"permit_login"`
}

type NTPconfig struct {
	genericConf `toml:",inline" mapstructure:",squash"`
	ListenAddr  string `toml:"listen_addr" mapstructure:"listen_addr"`
	// CurrZone change the real zone to the fake one for specific effects
	CurrZone string `toml:"curr_zone" mapstructure:"curr_zone"`
}

type DNSconfig struct {
	genericConf `toml:",inline" mapstructure:",squash"`
	ListenAddr  string `toml:"listen_addr" mapstructure:"listen_addr"`
	DefaultTTL  uint32 `toml:"default_ttl" mapstructure:"default_ttl"`
}

type SMTPconfig struct {
	genericConf      `toml:",inline" mapstructure:",squash"`
	ListenAddr       string `toml:"listen_addr" mapstructure:"listen_addr"`
	LocalTLSCertPath string `toml:"local_tls_cert_path" mapstructure:"local_tls_cert_path"`
	LocalTLSKeyPath  string `toml:"local_tls_key_path" mapstructure:"local_tls_key_path"`
	LocalSaveDir     string `toml:"local_save_dir" mapstructure:"local_save_dir"`
	NaturalTLS       bool   `toml:"natural_tls" mapstructure:"natural_tls"`
	AuthRequired     bool   `toml:"auth_required" mapstructure:"auth_required"`
}

type HTTPconfig struct {
	genericConf `toml:",inline" mapstructure:",squash"`
}

type FakeDBconf struct {
	genericConf `toml:",inline" mapstructure:",squash"`
	ListenAddr  string `toml:"listen_addr" mapstructure:"listen_addr"`
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
		Language string `toml:"language" mapstructure:"language"`

		// fields defined for services
		// The reason why to use struct name as ServerConfig's member name is
		// the iteration in `services_man.go` upon struct for data/control path needs refection

		SSHconfig  SSHconfig  `toml:"ssh" mapstructure:"ssh"`
		NTPconfig  NTPconfig  `toml:"ntp"  mapstructure:"ntp"`
		DNSconfig  DNSconfig  `toml:"dns" mapstructure:"dns"`
		SMTPconfig SMTPconfig `toml:"smtp" mapstructure:"smtp"`
		HTTPconfig HTTPconfig `toml:"http" mapstructure:"http"`
		// currently used for locally recording
		RecDBConfig RecDBConfig `toml:"rec_db_config" mapstructure:"rec_db_config"`
	} `toml:"server_config" mapstructure:"server_config"`
}

type ServEnum int
type ServEnumInterface interface {
	// GetPort return the port number in the representation of uint16
	GetPort() uint16
	// GetMaxNum return the max number of client in the representation of uint32
	GetMaxNum() uint32
	// GetTimeout return the tolerant timeout in the representation of uint32
	GetTimeout() uint32
}

// Which is extremely dumb

func (x SSHconfig) GetPort() uint16     { return x.ListenPort }
func (x SSHconfig) GetMaxNum() uint32   { return x.MaxClientNum }
func (x SSHconfig) GetTimeout() uint32  { return x.ClientConnTimeout }
func (x DNSconfig) GetPort() uint16     { return x.ListenPort }
func (x DNSconfig) GetMaxNum() uint32   { return x.MaxClientNum }
func (x DNSconfig) GetTimeout() uint32  { return x.ClientConnTimeout }
func (x NTPconfig) GetPort() uint16     { return x.ListenPort }
func (x NTPconfig) GetMaxNum() uint32   { return x.MaxClientNum }
func (x NTPconfig) GetTimeout() uint32  { return x.ClientConnTimeout }
func (x SMTPconfig) GetPort() uint16    { return x.ListenPort }
func (x SMTPconfig) GetMaxNum() uint32  { return x.MaxClientNum }
func (x SMTPconfig) GetTimeout() uint32 { return x.ClientConnTimeout }
func (x HTTPconfig) GetPort() uint16    { return x.ListenPort }
func (x HTTPconfig) GetMaxNum() uint32  { return x.MaxClientNum }
func (x HTTPconfig) GetTimeout() uint32 { return x.ClientConnTimeout }

const (
	RawEnum ServEnum = iota
	SSHEnum
	NTPEnum
	DNSEnum
	SMTPEnum
	HTTPEnum
	ENDofEnum
)

// ServLUT is a Read-Only look-up table for
// querying different services literal representation
var ServLUT = map[ServEnum]string{
	SSHEnum:  "<SSH>: ",
	HTTPEnum: "<HTTP>: ",
	SMTPEnum: "<SMTP>: ",
	NTPEnum:  "<NTP>: ",
}

func (lc *LocalConfig) SelectTerm(s ServEnum) any {
	sc := lc.ServerConfig
	switch s {
	case SSHEnum:
		if CheckSSHconfig(&sc.SSHconfig) {
			return sc.SSHconfig
		}
	case NTPEnum:
		if CheckNTPconfig(&sc.NTPconfig) {
			return sc.NTPconfig
		}
	case DNSEnum:
		if CheckDNSconfig(&sc.DNSconfig) {
			return sc.DNSconfig
		}
	case SMTPEnum:
		if CheckSMTPconfig(&sc.SMTPconfig) {
			return sc.SMTPconfig
		}
	case HTTPEnum:
		if CheckHTTPconfig(&sc.HTTPconfig) {
			return sc.HTTPconfig
		}
	default:
	}
	return nil
}

func (lc *LocalConfig) SelectPort(s ServEnum) uint16 {
	t := lc.SelectTerm(s)
	if t == nil {
		return 0
	}
	return max(t.(ServEnumInterface).GetPort(), 1025)
}

func (lc *LocalConfig) SelectMaxClient(s ServEnum) uint32 {
	t := lc.SelectTerm(s)
	if t == nil {
		return 1
	}
	return max(t.(ServEnumInterface).GetMaxNum(), 1)
}
func (lc *LocalConfig) SelectTimeout(s ServEnum) uint32 {
	t := lc.SelectTerm(s)
	if t == nil {
		return 3
	}
	return max(t.(ServEnumInterface).GetTimeout(), 3)
}

// SelectTLSpairWithRemoteHost will decide for whether returning tuple (cert, key, remoteServer).
// sometimes should be upgraded to TLS in further protocol process (like `starttls` in SMTP),
// not always start TLS from the very beginning.
// so this function will have to check the specific field inside the configuration.
func (lc *LocalConfig) SelectTLSpairWithRemoteHost(s ServEnum) (certPath, keyPath, RemoteServ string) {
	sc := lc.ServerConfig
	switch s {
	case HTTPEnum:
		if CheckHTTPconfig(&sc.HTTPconfig) {
			http := sc.HTTPconfig
			certPath, keyPath, RemoteServ = http.TLSCertPath, http.TLSKeyPath, ""
		}
	case SMTPEnum:
		if CheckSMTPconfig(&sc.SMTPconfig) {
			smtp := sc.SMTPconfig
			if smtp.NaturalTLS {
				certPath, keyPath, RemoteServ = smtp.TLSCertPath, smtp.TLSKeyPath, ""
			}
		}
	default:
	}
	return
}

type PortReloadDef struct {
	NetType string
	genericConf
}

type ServConcurrentCtrl struct {
	Ctx           context.Context
	ServNetTypeCh <-chan string
	DBPortCh      <-chan PortReloadDef
}

// Naive check

func CheckSSHconfig(sshConf *SSHconfig) bool {
	// sshConf.ListenAddr might be localhost, which can not be accepted by net.ParseIP
	if sshConf == nil || sshConf.ListenPort <= 1024 {
		return false
	}
	// At this moment and at most, we can only check whether those fields are null
	if sshConf.MaxClientNum == 0 {
		sshConf.MaxClientNum = 1
	} else if sshConf.ClientConnTimeout == 0 {
		sshConf.ClientConnTimeout = 30
	} else if sshConf.ResponseType == "" {
		sshConf.ResponseType = "repeat"
	}
	return true
}

func CheckNTPconfig(ntpConf *NTPconfig) bool {
	return ntpConf != nil && len(ntpConf.ListenAddr) != 0 &&
		ntpConf.ListenPort > 1024
}

func CheckDNSconfig(dnsConf *DNSconfig) bool {
	return dnsConf != nil && len(dnsConf.ListenAddr) != 0 &&
		dnsConf.ListenPort > 1024
}

// fPairChecking will check if both file path (noted as fpath1, fpath2) exist at the same time or not.
func fPairChecking(fpath1, fpath2 string) bool {
	_, err1 := os.Stat(fpath1)
	_, err2 := os.Stat(fpath2)
	return (os.IsNotExist(err1) && os.IsNotExist(err2) &&
		len(fpath1) == 0 && len(fpath2) == 0) ||
		(os.IsExist(err1) && os.IsExist(err2))
}

func CheckSMTPconfig(smtpConf *SMTPconfig) bool {
	if smtpConf == nil {
		return false
	}
	return len(smtpConf.ListenAddr) != 0 && smtpConf.ListenPort > 1024 &&
		fPairChecking(smtpConf.TLSKeyPath, smtpConf.TLSCertPath)
}

func CheckHTTPconfig(httpConf *HTTPconfig) bool {
	if httpConf == nil {
		return false
	}
	return httpConf.ListenPort > 1024 &&
		fPairChecking(httpConf.TLSKeyPath, httpConf.TLSCertPath)
}
