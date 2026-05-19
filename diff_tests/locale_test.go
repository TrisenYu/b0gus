package diff_tests

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

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
	langTag string,
	ErrDescriptor string,
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
		switch vv.(type) {
		case string:
			strVal = vv.(string)
		case int:
			strVal = strconv.Itoa(vv.(int))
		default:
			t.Errorf("Unexpected type for %s: %T", msg, vv)
			t.FailNow()
		}
		ds = string(bytes.ReplaceAll([]byte(ds), []byte("{{."+msg+"}}"), []byte(strVal)))
	}

	assert.Equal(t, ds, payload)
}

func dumpToml(
	langTag string,
	localeTag string,
) (any, error) {
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

func TestLocale(t *testing.T) {
	currConf := configs.LoadDefaultConfig("../configs/config.toml")
	localeExam(t, currConf, "zh_cn", "main.DatabaseEmptyError", nil)
	localeExam(t, currConf, "en", "main.DatabaseChangingWarn", nil)
	localeExam(
		t, currConf, "de",
		"crypto_aux.PemFileOpenFailure",
		map[string]any{
			"PemPath": "/etc/hosts.deny",
		},
	)
	localeExam(
		t, currConf, "JA",
		"services.SSHEstablishConnectionFailure",
		map[string]any{
			"RemoteAddr": "192.168.0.2",
			"CurrSSHver": "SSH-2.0-OpenSSH_11.1p2_3.4.5 Debian-67",
			"ErrInfo":    "Broken network",
		},
	)
	localeExam(
		t, currConf, "ko",
		"databases.DatabaseTypeError",
		map[string]any{"Database": "MySQL"},
	)
	localeExam(
		t, currConf, "fr",
		"services.SSHSwitchListenerError",
		map[string]any{"ListenAddr": net.IPv4(127, 0, 0, 1).String()},
	)
	currConf.ServerConfig.Language = "en"
	configs.GlobConf.Store(currConf)
}
