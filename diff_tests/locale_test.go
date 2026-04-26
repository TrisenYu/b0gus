package diff_tests

import (
	"b0gus/configs"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: there might be a combinatorial method to test all language.

func TestLocale(t *testing.T) {
	currConf := configs.LoadDefaultConfig("../configs/config.toml")
	currConf.ServerConfig.Language = "zh_cn"
	configs.GlobConf.Store(currConf)
	payload := configs.GetLocalizedMsg("main.DatabaseEmptyError", nil)
	assert.Equal(t, "按给定配置获取到了空的数据库操作符。", payload)
	currConf.ServerConfig.Language = "en"
	configs.GlobConf.Store(currConf)
	payload = configs.GetLocalizedMsg("main.DatabaseChangingWarn", nil)
	assert.Equal(t, "Due to internal error, the currently used database will not be replaced.", payload)
	currConf.ServerConfig.Language = "de"
	configs.GlobConf.Store(currConf)
	payload = configs.GetLocalizedMsg(
		"crypto_aux.PemFileOpenFailure",
		map[string]any{
			"PemPath": "/etc/hosts.deny",
		},
	)
	assert.Equal(
		t,
		"PEM-Datei </etc/hosts.deny> kann nicht zum Konfigurieren des lokalen SSH-Public Keys geöffnet werden!",
		payload,
	)
	currConf.ServerConfig.Language = "ja"
	configs.GlobConf.Store(currConf)
	payload = configs.GetLocalizedMsg(
		"services.SSHEstablishConnectionFailure",
		map[string]any{
			"RemoteAddr": "192.168.0.2",
			"CurrSSHver": "SSH-2.0-OpenSSH_11.1p2_3.4.5 Debian-67",
			"ErrInfo":    "Broken network",
		},
	)
	assert.Equal(
		t,
		"SSH接続を確立できません。（リモートアドレス、現在のSSHバージョン、失敗原因）："+
			"（<192.168.0.2>, <SSH-2.0-OpenSSH_11.1p2_3.4.5 Debian-67>, <Broken network>）。",
		payload,
	)
	currConf.ServerConfig.Language = "ko"
	configs.GlobConf.Store(currConf)
	payload = configs.GetLocalizedMsg(
		"databases.DatabaseTypeError",
		map[string]any{
			"Database": "MySQL",
		},
	)
	assert.Equal(t, "지원되지 않는 데이터베이스 <MySQL>를 발견했습니다!", payload)
	currConf.ServerConfig.Language = "en"
	configs.GlobConf.Store(currConf)
}
