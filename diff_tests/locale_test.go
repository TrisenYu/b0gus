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
	assert.Equal(t, "Won't update the database handler!", payload)
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
		"PEM-Datei </etc/hosts.deny> zum Einrichten des lokalen SSH-Public-Keys konnte nicht geöffnet werden!",
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
		"SSH接続の確立に失敗しました。（リモートアドレス、現在のSSHバージョン番号、失敗理由）："+
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
	assert.Equal(t, "지원되지 않는 데이터베이스<MySQL>가 발견되었습니다!", payload)
	currConf.ServerConfig.Language = "en"
	configs.GlobConf.Store(currConf)
}
