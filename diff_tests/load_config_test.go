package diff_tests

import (
	"b0gus/crypto_aux"
	"bytes"
	"os"
	"testing"

	"b0gus/configs"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var RecDBConfigArr = []struct{
	conf configs.RecDBConfig
	ans bool
}{
	{
		conf: configs.RecDBConfig{
			DBType: "postgresql", DBName: "postgresql", DBAddr: "localhost",
			DBAdminName: "admin", DBAdminPassword: "admin",
			DBTimeout: 10, DBPort: 10101,
		},
		ans: true,
	},
}

func TestRecDBConf(t *testing.T) {
	for _, testcase := range RecDBConfigArr {
		conf, ans := testcase.conf, testcase.ans
		assert.Equal(t, ans, conf.DBSelfCheck())
	}
}

var MockContent = map[configs.ServEnum]string{
	configs.SSHEnum: `
language = "zh_cn"
[ssh]
    listen_addr = "localhost" # "localhost"
    listen_port = 2222 # 2333
    max_client_num = 8
    client_conn_timeout = 180  # in seconds
    # 为假的则默认拒绝所有连接，只记录ip, username, password
    permit_login = true
    # 什么也不交互
    response_type = "repeat"
    login_banner = ''
	tls_key_path = "./test_dir/b0gus-host.pem"
    pem_name = "b0gus-host.pem"
    pem_type = "elliptic"
    pem_len = 384
	# 所嵌入的DB
	db_type = "postgresql" # mongodb
	db_name = "postgres" # 二选一
	db_addr = "localhost"
	db_port = 54321 # 5432 for postgres, 27017 for mongodb
	db_timeout = 10
	db_admin_name = "myGod" # admin for mongodb
	db_admin_password = "I am the storm that is approaching"
	# 这个是类似本地sqlite的用法
	db_path = ""
`,
	configs.NTPEnum: `language = "en"
[ntp]
    listen_addr = "127.0.0.1"
    listen_port = 2053
    max_client_num = 32
	db_type = "sqlite"
	db_name = "sqlite"
	db_addr = ""
	db_port = 0
	db_timeout = 10 # second
	db_admin_name = ""
	db_admin_password = "YouP@ssw0rdShouldBeStrongAndInvisibleToOthers"
	db_path = "/home/ops/baka-v9.db"
`,
	configs.SMTPEnum: `language = "fr"
[smtp]
    listen_addr = "0.0.0.0"
    listen_port = 2555
    # (cert, key) should be signed by real CA
    # or keep both blank
    tls_cert_path = ""
    tls_key_path = ""
    local_save_dir = ""
    max_client_num = 32
	db_type = "MongoDB"
	db_name = "mongo"
	db_addr = "api.mongo0x1234567.com"
	db_port = 27017
	db_timeout = 10 # second
	db_admin_name = "admin"
	db_admin_password = "YouP@ssw0rdShouldBeStrongAndInvisibleToOthers"
	db_path = ""
`,
}

func TestNormalParsingConfig(t *testing.T) {
	delPem := func(path string) {
		err := os.Remove(path)
		assert.Equal(t, nil, err, "Unable to delete test.pem")
	}
	defer delPem("./test_dir/b0gus-host.pem")

	// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-= SSH -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
	var currConf configs.LocalConfig
	configs.LocalConfigPathAsStr = "./test_dir/"
	viper.SetConfigType("toml")
	err := viper.ReadConfig(bytes.NewBuffer([]byte(MockContent[configs.SSHEnum])))
	assert.Nil(t, err)
	assert.Nil(t, viper.Unmarshal(&currConf))

	// create as normal use
	_ = crypto_aux.LoadOrCreateSSHpem(
		currConf.SSHconfig.TLSKeyPath,
		currConf.SSHconfig.PemType, currConf.SSHconfig.PemLen,
	)
	ok := configs.CheckSSHconfig(&currConf.SSHconfig)
	assert.True(t, ok)
	_, ok = currConf.SelectTerm(configs.SSHEnum).(configs.SSHconfig)
	assert.True(t, ok)
	if !ok {
		t.FailNow()
	}
	configs.LocalConfigPathAsStr = "./configs/"
	// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-= NTP -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
	err = viper.ReadConfig(bytes.NewBuffer([]byte(MockContent[configs.NTPEnum])))
	assert.Nil(t, err)
	assert.Nil(t, viper.Unmarshal(&currConf))
	ok = currConf.NTPconfig.CheckAll()
	assert.True(t, ok)
	// -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-= SMTP -=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
	err = viper.ReadConfig(bytes.NewBuffer([]byte(MockContent[configs.NTPEnum])))
	assert.Nil(t, err)
	assert.Nil(t, viper.Unmarshal(&currConf))
	ok = currConf.NTPconfig.CheckAll()
	assert.True(t, ok)
}

func TestAbnormalParsingConfigForSSH(t *testing.T) {
	viper.SetConfigType("toml")
	err := viper.ReadConfig(bytes.NewBuffer([]byte(MockContent[configs.SSHEnum])))
	assert.Nil(t, err)
	var currConf configs.LocalConfig
	assert.Nil(t, viper.Unmarshal(&currConf))

	// this will create b0gus-host.pem under ./diff_tests/
	_ = crypto_aux.LoadOrCreateSSHpem(
		currConf.SSHconfig.PemName,
		currConf.SSHconfig.PemType, currConf.SSHconfig.PemLen,
	)

	ok := configs.CheckSSHconfig(&currConf.SSHconfig)
	assert.False(t, ok)
	sshConf, ok := currConf.SelectTerm(configs.SSHEnum).(configs.SSHconfig)
	assert.True(t, ok)
	assert.False(t, configs.CheckSSHconfig(&sshConf))
	err = os.Remove(currConf.SSHconfig.PemName)
	assert.Equal(t, nil, err, "Unable to delete test.pem")
}
