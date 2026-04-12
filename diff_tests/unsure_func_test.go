// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package diff_tests

import (
	"b0gus/crypto_aux"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"b0gus/configs"
	"b0gus/misc_utils"

	"github.com/stretchr/testify/assert"
)

func TestBase64(t *testing.T) {
	_, err := crypto_aux.Base64Recover("abc")
	assert.NotNil(t, err)
	whatWeHave, err := crypto_aux.Base64Recover("")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(whatWeHave))
	base64str := crypto_aux.Base64Convert([]byte("abc"))
	originStr, err := crypto_aux.Base64Recover(base64str)
	assert.Nil(t, err)
	assert.Equal(t, []byte("abc"), originStr)
}

func TestReflection(t *testing.T) {
	// assemble to configs.Config_path_as_str
	examConfPath := "../configs/example.toml"
	testConfPath, _ := filepath.Abs(examConfPath)
	localConf := configs.LoadDefaultConfig(testConfPath)
	resMap := misc_utils.TurnStruct2Map(localConf.ServerConfig)
	assert.NotEqual(t, resMap, nil)
	sshName := misc_utils.GetTypeNameViaType(localConf.ServerConfig.SSHconfig)
	assert.Equal(t, "SSHconfig", sshName)
	misc_utils.GetTypeNameViaType(&localConf.ServerConfig.SSHconfig)
	_, ok := resMap[sshName]
	assert.Equal(t, true, ok)
	resMap = misc_utils.TurnStruct2Map(localConf.ServerConfig.SSHconfig)
	curr, err := misc_utils.GetFieldValueByName(localConf.ServerConfig, sshName)
	assert.Equal(t, nil, err)
	_, ok = curr.(*configs.SSHconfig)
	assert.Equal(t, false, ok)
	_, ok = curr.(configs.SSHconfig)
	assert.Equal(t, true, ok)
	recur, ok := curr.(configs.SSHconfig)
	assert.Equal(t, true, ok)
	assert.IsType(t, &configs.SSHconfig{}, &recur)

	curr, err = misc_utils.GetFieldValueByName(
		localConf.ServerConfig,
		misc_utils.GetTypeNameViaType(localConf.ServerConfig.SMTPconfig),
	)
	assert.Equal(t, nil, err)
	assert.Equal(t, localConf.ServerConfig.SMTPconfig, curr)
	_, ok = curr.(configs.SMTPconfig)
	assert.Equal(t, true, ok)

	curr, err = misc_utils.GetFieldValueByName(
		map[string]map[string]any{
			"ok": nil, "nok": {
				"hello": "world",
				"123":   123,
			},
		},
		"nok",
	)
	assert.Equal(t, nil, err)
}

func TestAnyType(t *testing.T) {
	type innerStruct struct {
		A int
		B string
		C func()
		D *testing.T
	}
	var (
		a = 1
		b ***int
		c struct {
			concealedPtr **int
			HellYeah     *string
			WhatCanIsay  []int
			JustTestIt   []string
			AnOpenFunc   func() int
			ManHaha      map[int]string
		}
		d = &c
		e innerStruct
		f struct {
			io.Writer
			pkix.Name
			tmp string
		}
	)
	curr, err := misc_utils.GetFieldValueByName(a, "")
	assert.Equal(t, nil, err)
	t.Logf("%v", curr)
	var aa any
	assert.IsNotType(t, struct{}{}, nil)
	assert.IsNotType(t, struct{}{}, aa)

	bName := misc_utils.GetTypeNameViaType(b)
	assert.NotEqual(t, "", bName)

	cName := misc_utils.GetTypeNameViaType(c.HellYeah)
	assert.NotEqual(t, "", cName)
	cName = misc_utils.GetTypeNameViaType(c.WhatCanIsay)
	assert.NotEqual(t, "", cName)

	cName = misc_utils.GetTypeNameViaType(c.JustTestIt)
	assert.NotEqual(t, "", cName)

	cName = misc_utils.GetTypeNameViaType(c.AnOpenFunc)
	assert.NotEqual(t, "", cName)

	cName = misc_utils.GetTypeNameViaType(c.ManHaha)
	assert.NotEqual(t, "", cName)

	cName = misc_utils.GetTypeNameViaType(c)
	assert.NotEqual(t, "", cName)

	dName := misc_utils.GetTypeNameViaType(d)
	assert.NotEqual(t, "", dName)

	eName := misc_utils.GetTypeNameViaType(e)
	assert.NotEqual(t, "", eName)

	fName := misc_utils.GetTypeNameViaType(f)
	assert.NotEqual(t, "", fName)
}

type ipPort struct {
	Addr string
	Port uint16
}

func TestIPvXparser(t *testing.T) {
	tests := []struct {
		input    string
		expected ipPort
	}{
		{
			"127.0.0.1:1234",
			ipPort{"127.0.0.1", 1234},
		},
		{
			"::1:1234",
			ipPort{"::1", 1234},
		},
		{
			"::1:123444",
			ipPort{"", 0},
		},
		{
			"3.1.4.5:26",
			ipPort{"3.1.4.5", 26},
		},
		{
			"12345.14159.26525.89:0001001",
			ipPort{"", 0},
		},
		{
			"what the hell of such testcase:123",
			ipPort{"", 0},
		},
		{
			"197.34.21.6:20251",
			ipPort{"197.34.21.6", 20251},
		},
		{
			"2001:25a:4402:22ba:8883:3f67:8342:4a9b:11451",
			ipPort{"2001:25a:4402:22ba:8883:3f67:8342:4a9b", 11451},
		},
		{
			"2001::8883:3f67:8342:4a9b:11451",
			ipPort{"2001::8883:3f67:8342:4a9b", 11451},
		},
		{
			"2001::8883:3f67:8342:4a9b:nonport",
			ipPort{"", 0},
		},
		{
			"localhost:1234",
			ipPort{"", 0},
		},
		{
			":::123",
			ipPort{"::", 123},
		},
		{
			"12:f7:e7:b9:9e:13",
			ipPort{"", 0},
		},
		{
			"[2001:db8::1]:80",
			ipPort{"2001:db8::1", 80},
		},
		{
			"[]:80",
			ipPort{"", 0},
		},
		{
			"[::]:80",
			ipPort{"::", 80},
		},
		{
			"[::1]:80",
			ipPort{"::1", 80},
		},
		{
			":80",
			ipPort{"", 0},
		},
		{
			"80",
			ipPort{"", 0},
		},
		{
			"",
			ipPort{"", 0},
		},
		{
			"[:]78",
			ipPort{"", 0},
		},
		{
			"[]1919",
			ipPort{"", 0},
		},
		{
			"[::]13",
			ipPort{"", 0},
		},
		{
			"[::]:BillieJean",
			ipPort{"", 0},
		},
		{
			"::1:BillieJean",
			ipPort{"", 0},
		},
		{
			"fe80::1cc0:3e8c:119f:c2e1%ens18/1234",
			ipPort{"fe80::1cc0:3e8c:119f:c2e1", 1234},
		},
	}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			str, num := misc_utils.IPAddrSplit(tt.input)
			assert.Equal(t, tt.expected.Addr, str, "wrong addr")
			assert.Equal(t, tt.expected.Port, num, "wrong port")
		})
	}
}
