package diff_tests

/// Last modified at 2026/05/15 星期五 10:25:50
import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"

	"b0gus/configs"
)

func TestDumpToml(t *testing.T) {
	data, err := dumpToml("zh_cn", "main.DatabaseEmptyError")
	assert.Nil(t, err)
	assert.Equal(t, "按给定配置获取到了空的数据库操作符。", data)
}

// [TODO]: there might be a combinatorial method to test all language.
func localeExam(
	t *testing.T,
	currConf *configs.LocalConfig,
	langTag, ErrDescriptor string,
	msgPayload map[string]any,
) {
	currConf.ServerConfig.Language = langTag
	configs.GlobConf.Store(currConf)
	langTag = strings.ToLower(langTag)
	payload := configs.GetLocalizedMsg(ErrDescriptor, msgPayload)
	data, err := dumpToml(langTag, ErrDescriptor)
	assert.Nil(t, err)
	ds, ok := data.(string)
	assert.True(t, ok)

	for msg, vv := range msgPayload {
		var strVal string
		switch v := vv.(type) {
		case string:
			strVal = v
		case int:
			strVal = strconv.Itoa(v)
		default:
			t.Errorf("Unexpected type for %s: %T", msg, v)
			t.FailNow()
		}
		ds = string(bytes.ReplaceAll([]byte(ds), []byte("{{."+msg+"}}"), []byte(strVal)))
	}
	assert.Equal(t, ds, payload)
}

func dumpToml(langTag, localeTag string) (any, error) {
	var (
		sb     strings.Builder
		config map[string]any
	)
	sb.WriteString("../assets/locale/active.")
	sb.WriteString(langTag)
	sb.WriteString(".toml")

	_, err := toml.DecodeFile(sb.String(), &config)
	if err != nil {
		return "", err
	}
	sb.Reset()
	sb.WriteString(localeTag)
	sb.WriteString(".other")
	return getTomlValue(config, sb.String()), err
}

func getTomlValue(data map[string]any, path string) any {
	keys := strings.Split(path, ".")
	var val any = data
	for _, k := range keys {
		v, ok := val.(map[string]any)
		if !ok {
			return nil
		}
		val = v[k]
	}
	return val
}

type localeCasesStruct struct {
	langTag    string
	Descriptor string
	msgPayload map[string]any
}

var localeCases = []localeCasesStruct{
	{"zh_cn", "main.DatabaseEmptyError", nil},
	{"en", "main.DatabaseChangingWarn", nil},
	{
		"de", "crypto_aux.PemFileOpenFailure",
		map[string]any{"PemPath": "/etc/hosts.deny"},
	},
	{
		"JA", "services.SSHEstablishConnectionFailure",
		map[string]any{
			"RemoteAddr": "192.168.0.2",
			"CurrSSHver": "SSH-2.0-OpenSSH_11.1p2_3.4.5 Debian-67",
			"ErrInfo":    "Broken network",
		},
	},
	{
		"ko", "databases.DatabaseTypeError",
		map[string]any{"Database": "MySQL"},
	},
	{
		"fr", "services.SSHSwitchListenerError",
		map[string]any{"ListenAddr": net.IPv4(127, 0, 0, 1).String()},
	},
}

func TestLocale(t *testing.T) {
	currConf := configs.LoadDefaultConfig("../configs/config.toml")
	for _, l := range localeCases {
		localeExam(t, currConf, l.langTag, l.Descriptor, l.msgPayload)
	}
	currConf.ServerConfig.Language = "en"
	configs.GlobConf.Store(currConf)
}
