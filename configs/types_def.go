// Package configs
package configs

// Last modified at 2026/03/30 星期一 21:46:46
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"context"
	"os"
	"path/filepath"

	"b0gus/internal/misc_utils"
)

// services structures' definitions

// TLSFilesConf holds files' paths that are set to TLS pair(i.e. cert, key).
type TLSFilesConf struct {
	// path to Certificate signed-off by real CA for helping authenticate the communicating entity
	TLSCertPath string `toml:"tls_cert_path,omitempty" json:"tls_cert_path,omitempty" mapstructure:"tls_cert_path"`
	// path to private key of one certificate signed-off by real CA
	TLSKeyPath string `toml:"tls_key_path,omitempty" json:"tls_key_path,omitempty" mapstructure:"tls_key_path"`
}

func (t *TLSFilesConf) GetCertPath() string {
	if t == nil {
		return ""
	}
	return t.TLSCertPath
}

func (t *TLSFilesConf) GetKeyPath() string {
	if t == nil {
		return ""
	}
	return t.TLSKeyPath
}

func (t *TLSFilesConf) SelfCheck() bool {
	return CheckFilePair(t.TLSKeyPath, t.TLSCertPath)
}

// GenericServConf defines:
//   - port number for the service
//   - the maximum client number
//   - the connection timeout
//   - TLS (key, cert) path.
type GenericServConf struct {
	TLSFilesConf      `toml:",inline" json:",inline" mapstructure:",squash"`
	MaxClientNum      uint32 `toml:"max_client_num" json:"max_client_num,omitempty" mapstructure:"max_client_num" default:"32"`
	ClientConnTimeout uint32 `toml:"client_conn_timeout" json:"client_conn_timeout,omitempty" mapstructure:"client_conn_timeout" default:"5"`
	ListenPort        uint16 `toml:"listen_port" json:"listen_port,omitempty" mapstructure:"listen_port"`
}

// RecDBConfig here means the recording database for attacker features
// not as a bogus service.
type RecDBConfig struct {
	DBType          string `toml:"db_type" json:"db_type,omitempty" mapstructure:"db_type"`
	DBName          string `toml:"db_name" json:"db_name,omitempty" mapstructure:"db_name"`
	DBPath          string `toml:"db_path" json:"db_path,omitempty" mapstructure:"db_path"`
	DBAddr          string `toml:"db_addr" json:"db_addr,omitempty" mapstructure:"db_addr"`
	DBAdminName     string `toml:"db_admin_name" json:"db_admin_name,omitempty" mapstructure:"db_admin_name"`
	DBAdminPassword string `toml:"db_admin_password" json:"db_admin_password,omitempty" mapstructure:"db_admin_password"`
	DBTimeout       uint64 `toml:"db_timeout" json:"db_timeout,omitempty" mapstructure:"db_timeout"`
	DBPort          uint16 `toml:"db_port" json:"db_port,omitempty" mapstructure:"db_port"`
}

func (r RecDBConfig) CheckDBport() bool {
	return misc_utils.IsPort(r.DBPort) && r.DBPort > 1024
}

func (r RecDBConfig) CheckDBpath() bool {
	return len(r.DBPath) == 0 || misc_utils.IsFilePath(r.DBPath)
}

func (r RecDBConfig) CheckDBaddr() bool {
	return misc_utils.IsIP(r.DBAddr) || misc_utils.IsHostnameRFC1123(r.DBAddr)
}

// DBSelfCheck returns true once the preset configuration is valid.
func (r RecDBConfig) DBSelfCheck() bool {
	// [TODO]: enhance configuration checking
	// could not include `: / ? # [ ] @` in admin_name or password for certain DB,
	// otherwise they need convert in the way that url encoding criterion
	// that enforces
	//		r.DBAdminName
	//		r.DBAdminPassword
	return r.CheckDBport() && (r.CheckDBpath() || r.CheckDBaddr())
}

type WrappedConf struct {
	GenericServConf `toml:",inline" json:",inline" mapstructure:",squash"`
	RecDBConfig     `toml:",inline" json:",inline" mapstructure:",squash"`
}

func (w WrappedConf) CheckAll() bool {
	return w.SelfCheck() && w.CheckPort()
}

// SSHconfig is defined for fake SSH service
type SSHconfig struct {
	WrappedConf `toml:",inline" json:",inline" mapstructure:",squash"`
	// LLMConfPath points to the path of LLM's configuration
	LLMConfPath   string `toml:"llm_conf_path" json:"llm_conf_path,omitempty" mapstructure:"llm_conf_path"`
	LoginBanner   string `toml:"login_banner" json:"login_banner,omitempty" mapstructure:"login_banner"`
	ResponseType  string `toml:"response_type" json:"response_type,omitempty" mapstructure:"response_type"`
	HashAlgorithm string `toml:"hash_algorithm" json:"hash_algorithm,omitempty" mapstructure:"hash_algorithm"`
	// PemName only set one time after running up the whole server.
	// And it indicates the name of used pem file
	PemName string `toml:"pem_name" json:"pem_name,omitempty" mapstructure:"pem_name"`
	// PemType only set one time after running up the whole server
	PemType string `toml:"pem_type" json:"pem_type,omitempty" mapstructure:"pem_type"`
	// PemLen only set one time after running up the whole server
	PemLen       uint64 `toml:"pem_len" json:"pem_len,omitempty" mapstructure:"pem_len"`
	MaxAuthTries uint32 `toml:"max_auth_tries" json:"max_auth_tries,omitempty" mapstructure:"max_auth_tries"`
	PermitLogin  bool   `toml:"permit_login" json:"permit_login,omitempty" mapstructure:"permit_login"`
}

// NTPconfig is defined for fake NTP service
type NTPconfig struct {
	WrappedConf `toml:",inline" json:",inline" mapstructure:",squash"`
	// CurrZone change the real zone to the fake one for specific effects
	CurrZone string `toml:"curr_zone" json:"curr_zone,omitempty" mapstructure:"curr_zone"`
}

// DNSconfig is defined for fake DNS service
type DNSconfig struct {
	WrappedConf `toml:",inline" json:",inline" mapstructure:",squash"`
	DefaultTTL  uint32 `toml:"default_ttl" json:"default_ttl,omitempty" mapstructure:"default_ttl"`
}

// SMTPconfig is defined for fake SMTP service
type SMTPconfig struct {
	WrappedConf  `toml:",inline" json:",inline" mapstructure:",squash"`
	LocalSaveDir string `toml:"local_save_dir" json:"local_save_dir,omitempty" mapstructure:"local_save_dir"`
	RemoteAddr   string `toml:"remote_addr" json:"remote_addr,omitempty" mapstructure:"remote_addr"`
	NaturalTLS   bool   `toml:"natural_tls" json:"natural_tls,omitempty" mapstructure:"natural_tls"`
	AuthRequired bool   `toml:"auth_required" json:"auth_required,omitempty" mapstructure:"auth_required"`
}

// HTTPconfig is defined for fake SMTP service
type HTTPconfig struct {
	WrappedConf `toml:",inline" json:",inline" mapstructure:",squash"`
	RemoteAddr  string `toml:"remote_addr" json:"remote_addr,omitempty" mapstructure:"remote_addr"`
}

// FakeDBconf is defined for fake SMTP service
type FakeDBconf struct {
	HTTPconfig `toml:",inline" json:",inline" mapstructure:",squash"`
}

// TODO: push configuration from networking binary streams

type LocalConfig struct {
	Language           string `toml:"language" json:"language" mapstructure:"language" default:"en"`
	MsgQueueAddr       string `toml:"message_queue_addr" json:"message_queue_addr" mapstructure:"message_queue_addr"`
	RegisterCenterAddr string `toml:"register_center_addr" json:"register_center_addr" mapstructure:"register_center_addr"`
	// fields defined for services
	// The reason why to use struct name as ServerConfig's member name is
	// the iteration in `services_man.go` upon struct for data/control path needs refection
	SSHconfig  SSHconfig  `toml:"ssh" json:"ssh,omitempty" mapstructure:"ssh"`
	NTPconfig  NTPconfig  `toml:"ntp"  json:"ntp,omitempty" mapstructure:"ntp"`
	DNSconfig  DNSconfig  `toml:"dns" json:"dns,omitempty" mapstructure:"dns"`
	SMTPconfig SMTPconfig `toml:"smtp" json:"smtp,omitempty" mapstructure:"smtp"`
	HTTPconfig HTTPconfig `toml:"http" json:"http,omitempty" mapstructure:"http"`
	// currently used for locally recording
}

type ServEnumInterface interface {
	// GetPort return the port number in the representation of uint16
	GetPort() uint16
	// GetMaxNum return the max number of client in the representation of uint32
	GetMaxNum() uint32
	// GetTimeout return the tolerant timeout in the representation of uint32
	GetTimeout() uint32
}

type KeyCertPathPair interface {
	// GetCertPath return the certPath registered as the member of one struct
	GetCertPath() string
	// GetKeyPath return the keyPath registered as the member of one struct
	GetKeyPath() string
	// SelfCheck will check if the path pair is validated
	SelfCheck() bool
}

func (g GenericServConf) CheckPort() bool {
	val := g.GetPort()
	return misc_utils.IsPort(val) && val > 1024
}
func (g GenericServConf) GetPort() uint16    { return g.ListenPort }
func (g GenericServConf) GetMaxNum() uint32  { return g.MaxClientNum }
func (g GenericServConf) GetTimeout() uint32 { return g.ClientConnTimeout }

type ServEnum int

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
		DNSEnum:  "<DNS>: ",
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
	var ret any = nil
	switch s {
	case SSHEnum:
		ret = lc.SSHconfig
	case NTPEnum:
		ret = lc.NTPconfig
	case DNSEnum:
		ret = lc.DNSconfig
	case SMTPEnum:
		ret = lc.SMTPconfig
	case HTTPEnum:
		ret = lc.HTTPconfig
	default:
	}
	return ret
}

func (lc *LocalConfig) SelectPort(s ServEnum) uint16 {
	if lc == nil {
		return 0
	}
	t := lc.SelectTerm(s)
	if t == nil {
		return 0
	}
	cast, ok := t.(ServEnumInterface)
	if !ok {
		return 0
	}
	return max(cast.GetPort(), 1025)
}

func (lc *LocalConfig) SelectMaxClient(s ServEnum) uint32 {
	if lc == nil {
		return 1
	}
	t := lc.SelectTerm(s)
	if t == nil {
		return 1
	}
	cast, ok := t.(ServEnumInterface)
	if !ok {
		return 1
	}
	return max(cast.GetMaxNum(), 1)
}

func (lc *LocalConfig) SelectTimeout(s ServEnum) uint32 {
	if lc == nil {
		return 3
	}
	t := lc.SelectTerm(s)
	if t == nil {
		return 3
	}
	cast, ok := t.(ServEnumInterface)
	if !ok {
		return 3
	}
	return max(cast.GetTimeout(), 3)
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
	switch s {
	case HTTPEnum:
		if CheckHTTPconfig(&lc.HTTPconfig) {
			http := lc.HTTPconfig
			return http.TLSCertPath, http.TLSKeyPath, http.RemoteAddr
		}
	case SMTPEnum:
		if CheckSMTPconfig(&lc.SMTPconfig) {
			smtp := lc.SMTPconfig
			if smtp.NaturalTLS { // that means it is a SMTPS proto
				return smtp.TLSCertPath, smtp.TLSKeyPath, smtp.RemoteAddr
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
	TerminatedCtx context.Context // TerminatedCtx is used for checking the signal of termination
	ServNetTypeCh <-chan string
	ConfCh        <-chan PortReloadDef
}

// Naive check

func LoadSSHpemPathWithPemName(conf *SSHconfig) string {
	if conf == nil {
		return ""
	}
	res, _ := filepath.Abs(filepath.Join(filepath.Dir(LocalConfigPathAsStr), conf.PemName))
	return res
}

func CheckSSHconfig(sshConf *SSHconfig) bool {
	// sshConf.ListenAddr might be localhost, which can not be accepted by net.ParseIP
	InvalidPathChecker := func(path string) bool {
		// only when the path is not empty and valid, the return true
		return len(path) == 0 || !misc_utils.IsFilePath(path)
	}
	if sshConf == nil || !sshConf.CheckPort() ||
		(InvalidPathChecker(sshConf.TLSKeyPath) && InvalidPathChecker(LoadSSHpemPathWithPemName(sshConf))) {
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
	return sshConf.DBSelfCheck()
}

func CheckSMTPconfig(smtpConf *SMTPconfig) bool {
	if smtpConf == nil {
		return false
	}
	check := len(smtpConf.RemoteAddr) == 0 || misc_utils.IsIP(smtpConf.RemoteAddr) ||
		misc_utils.IsHostnameRFC1123(smtpConf.RemoteAddr)
	return check && smtpConf.CheckAll() && CheckFilePair(smtpConf.TLSKeyPath, smtpConf.TLSCertPath)
}

func CheckHTTPconfig(httpConf *HTTPconfig) bool {
	if httpConf == nil {
		return false
	}
	check := len(httpConf.RemoteAddr) == 0 || misc_utils.IsIP(httpConf.RemoteAddr) ||
		misc_utils.IsHostnameRFC1123(httpConf.RemoteAddr)
	check = check && httpConf.CheckAll()
	return check && CheckFilePair(httpConf.TLSKeyPath, httpConf.TLSCertPath)
}

// CheckFilePair will check if both file path (noted as fpath1, fpath2) exist at the same time or not.
func CheckFilePair(fpath1, fpath2 string) bool {
	_, err1 := os.Stat(fpath1)
	_, err2 := os.Stat(fpath2)
	return (os.IsNotExist(err1) && os.IsNotExist(err2) &&
		len(fpath1) == 0 && len(fpath2) == 0) ||
		(os.IsExist(err1) && os.IsExist(err2))
}
