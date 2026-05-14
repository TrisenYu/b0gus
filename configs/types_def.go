// Package configs
package configs

// Last modified at 2026/03/30 星期一 21:46:46
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"context"
	"os"
)

// services structures' definitions

// GenericServConf defines:
//   - port number for the service
//   - the maximum client number
//   - the connection timeout
//   - TLS (key, cert) path.
type GenericServConf struct {
	// path to Certificate signed-off by real CA for helping authenticate the communicating entity
	TLSCertPath string `toml:"tls_cert_path" json:"tls_cert_path" mapstructure:"tls_cert_path"`
	// path to private key of one certificate signed-off by real CA
	TLSKeyPath        string `toml:"tls_key_path" json:"tls_key_path" mapstructure:"tls_key_path"`
	MaxClientNum      uint32 `toml:"max_client_num" json:"max_client_num" mapstructure:"max_client_num"`
	ClientConnTimeout uint32 `toml:"client_conn_timeout" json:"client_conn_timeout" mapstructure:"client_conn_timeout"`
	ListenPort        uint16 `toml:"listen_port" json:"listen_port" mapstructure:"listen_port"`
}
type SSHconfig struct {
	GenericServConf `toml:",inline" json:",inline" mapstructure:",squash"`
	LoginBanner     string `toml:"login_banner" json:"login_banner" mapstructure:"login_banner"`
	ResponseType    string `toml:"response_type" json:"response_type" mapstructure:"response_type"`
	HashAlgorithm   string `toml:"hash_algorithm" json:"hash_algorithm" mapstructure:"hash_algorithm"`
	// PemName only set one time after running up the whole server
	PemName string `toml:"pem_name" json:"pem_name" mapstructure:"pem_name"`
	// PemType only set one time after running up the whole server
	PemType string `toml:"pem_type" json:"pem_type" mapstructure:"pem_type"`
	// PemLen only set one time after running up the whole server
	PemLen       uint64 `toml:"pem_len" json:"pem_len" mapstructure:"pem_len"`
	MaxAuthTries uint32 `toml:"max_auth_tries" json:"max_auth_tries" mapstructure:"max_auth_tries"`
	PermitLogin  bool   `toml:"permit_login" json:"permit_login" mapstructure:"permit_login"`
}

type NTPconfig struct {
	GenericServConf `toml:",inline" json:",inline" mapstructure:",squash"`
	ListenAddr      string `toml:"listen_addr" json:"listen_addr" mapstructure:"listen_addr"`
	// CurrZone change the real zone to the fake one for specific effects
	CurrZone string `toml:"curr_zone" json:"curr_zone" mapstructure:"curr_zone"`
}

type DNSconfig struct {
	GenericServConf `toml:",inline" json:",inline" mapstructure:",squash"`
	ListenAddr      string `toml:"listen_addr" json:"listen_addr" mapstructure:"listen_addr"`
	DefaultTTL      uint32 `toml:"default_ttl" json:"default_ttl" mapstructure:"default_ttl"`
}

type SMTPconfig struct {
	GenericServConf  `toml:",inline" json:",inline" mapstructure:",squash"`
	ListenAddr       string `toml:"listen_addr" json:"listen_addr" mapstructure:"listen_addr"`
	LocalTLSCertPath string `toml:"local_tls_cert_path" json:"local_tls_cert_path" mapstructure:"local_tls_cert_path"`
	LocalTLSKeyPath  string `toml:"local_tls_key_path" json:"local_tls_key_path" mapstructure:"local_tls_key_path"`
	LocalSaveDir     string `toml:"local_save_dir" json:"local_save_dir" mapstructure:"local_save_dir"`
	NaturalTLS       bool   `toml:"natural_tls" json:"natural_tls" mapstructure:"natural_tls"`
	AuthRequired     bool   `toml:"auth_required" json:"auth_required" mapstructure:"auth_required"`
}

type HTTPconfig struct {
	GenericServConf `toml:",inline" mapstructure:",squash"`
}

type FakeDBconf struct {
	GenericServConf `toml:",inline" mapstructure:",squash"`
	ListenAddr      string `toml:"listen_addr" json:"listen_addr" mapstructure:"listen_addr"`
}

// RecDBConfig here means the recording database for attacker features
// not as a bogus service.
type RecDBConfig struct {
	Type          string `toml:"type" json:"type" mapstructure:"type"`
	Name          string `toml:"name" json:"name" mapstructure:"name"`
	Path          string `toml:"path" json:"path" mapstructure:"path"`
	Addr          string `toml:"addr" json:"addr" mapstructure:"addr"`
	AdminName     string `toml:"admin_name" json:"admin_name" mapstructure:"admin_name"`
	AdminPassword string `toml:"admin_password" json:"admin_password" mapstructure:"admin_password"`
	Timeout       uint64 `toml:"timeout" json:"timeout" mapstructure:"timeout"`
	Port          uint16 `toml:"port" json:"port" mapstructure:"port"`
}

// TODO: push configuration from networking binary streams

type LocalConfig struct {
	// ServerConfig should refer to the toml config defined in configs/config.toml
	ServerConfig struct {
		Language string `toml:"language" json:"language" mapstructure:"language"`

		// fields defined for services
		// The reason why to use struct name as ServerConfig's member name is
		// the iteration in `services_man.go` upon struct for data/control path needs refection

		SSHconfig  SSHconfig  `toml:"ssh" json:"ssh" mapstructure:"ssh"`
		NTPconfig  NTPconfig  `toml:"ntp"  json:"ntp" mapstructure:"ntp"`
		DNSconfig  DNSconfig  `toml:"dns" json:"dns" mapstructure:"dns"`
		SMTPconfig SMTPconfig `toml:"smtp" json:"smtp" mapstructure:"smtp"`
		HTTPconfig HTTPconfig `toml:"http" json:"http" mapstructure:"http"`
		// currently used for locally recording
		RecDBConfig RecDBConfig `toml:"rec_db_config" json:"rec_db_config" mapstructure:"rec_db_config"`
	} `toml:"server_config" json:"server_config" mapstructure:"server_config"`
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

/* Which is extremely dumb */

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
	RawEnum ServEnum = iota + 1
	SSHEnum
	NTPEnum
	DNSEnum
	SMTPEnum
	HTTPEnum
	SIPEnum
	ENDofEnum
)

var (
	// ServLUT is a Read-Only look-up table for
	// querying different services literal representation
	ServLUT = map[ServEnum]string{
		SSHEnum:  "<SSH>: ",
		HTTPEnum: "<HTTP>: ",
		SMTPEnum: "<SMTP>: ",
		NTPEnum:  "<NTP>: ",
	}
	ServInit = map[ServEnum]any{
		RawEnum:   struct{}{},
		SSHEnum:   SSHconfig{},
		NTPEnum:   NTPconfig{},
		DNSEnum:   DNSconfig{},
		SMTPEnum:  SMTPconfig{},
		HTTPEnum:  HTTPconfig{},
		SIPEnum:   struct{}{},
		ENDofEnum: struct{}{},
	}
)

// SelectTerm will return the sub-struct defined in LocalConfig
// when the given configuration is not nil
func (lc *LocalConfig) SelectTerm(s ServEnum) any {
	if lc == nil {
		return nil
	}
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
	if lc == nil {
		return 0
	}
	t := lc.SelectTerm(s)
	if t == nil {
		return 0
	}
	return max(t.(ServEnumInterface).GetPort(), 1025)
}

func (lc *LocalConfig) SelectMaxClient(s ServEnum) uint32 {
	if lc == nil {
		return 1
	}
	t := lc.SelectTerm(s)
	if t == nil {
		return 1
	}
	return max(t.(ServEnumInterface).GetMaxNum(), 1)
}
func (lc *LocalConfig) SelectTimeout(s ServEnum) uint32 {
	if lc == nil {
		return 3
	}
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
// For returned tuple, [string, string, string] owns the meaning of [certPath, keyPath, RemoteServ].
func (lc *LocalConfig) SelectTLSpairWithRemoteHost(s ServEnum) (string, string, string) {
	if lc == nil {
		return "", "", ""
	}
	sc := lc.ServerConfig
	switch s {
	case HTTPEnum:
		if CheckHTTPconfig(&sc.HTTPconfig) {
			http := sc.HTTPconfig
			return http.TLSCertPath, http.TLSKeyPath, ""
		}
	case SMTPEnum:
		if CheckSMTPconfig(&sc.SMTPconfig) {
			smtp := sc.SMTPconfig
			if smtp.NaturalTLS {
				return smtp.TLSCertPath, smtp.TLSKeyPath, ""
			}
		}
	default:
	}
	return "", "", ""
}

type PortReloadDef struct {
	NetType string
	GenericServConf
}

// ServConcurrentCtrl is used for notifying the target services to terminate
// or to update its configuration upon network.
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

// fPairChecking will check if both file path (noted as fpath1, fpath2) exist at the same time or not.
func fPairChecking(fpath1, fpath2 string) bool {
	_, err1 := os.Stat(fpath1)
	_, err2 := os.Stat(fpath2)
	return (os.IsNotExist(err1) && os.IsNotExist(err2) &&
		len(fpath1) == 0 && len(fpath2) == 0) ||
		(os.IsExist(err1) && os.IsExist(err2))
}
