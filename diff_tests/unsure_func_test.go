// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package diff_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	b0gus_config "b0gus/configs"
	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_misc_utils "b0gus/misc_utils"

	assert "github.com/stretchr/testify/assert"
)

func TestReflection(t *testing.T) {
	// assemble to b0gus_config.Config_path_as_str
	exam_conf_path := "../configs/example.toml"
	test_conf_path, _ := filepath.Abs(exam_conf_path)
	local_conf := b0gus_config.LoadDefaultConfig(test_conf_path)
	res_map := b0gus_misc_utils.TurnStruct2Map(local_conf.ServerConfig)
	assert.NotEqual(t, res_map, nil)
	for k, v := range res_map {
		fmt.Println(k, v)
	}
	ssh_name := b0gus_misc_utils.GetTypeNameViaType(local_conf.ServerConfig.SSHconfig)
	assert.Equal(t, "SSHconfig", ssh_name)
	pssh_name := b0gus_misc_utils.GetTypeNameViaType(&local_conf.ServerConfig.SSHconfig)
	fmt.Println(pssh_name)
	_, ok := res_map[ssh_name]
	assert.Equal(t, true, ok)
	res_map = b0gus_misc_utils.TurnStruct2Map(local_conf.ServerConfig.SSHconfig)
	for k := range res_map {
		fmt.Println(k)
	}

	curr, err := b0gus_misc_utils.GetFieldValueByName(local_conf.ServerConfig, ssh_name)
	assert.Equal(t, nil, err)
	_, ok = curr.(*b0gus_config.SSHconfig)
	assert.Equal(t, false, ok)
	_, ok = curr.(b0gus_config.SSHconfig)
	assert.Equal(t, true, ok)
	recur, ok := curr.(b0gus_config.SSHconfig)
	assert.Equal(t, true, ok)
	assert.IsType(t, &b0gus_config.SSHconfig{}, &recur)

	curr, err = b0gus_misc_utils.GetFieldValueByName(
		local_conf.ServerConfig,
		b0gus_misc_utils.GetTypeNameViaType(local_conf.ServerConfig.TelnetConfig),
	)
	assert.Equal(t, nil, err)
	assert.Equal(t, local_conf.ServerConfig.TelnetConfig, curr)
	_, ok = curr.(b0gus_config.TelnetConfig)
	assert.Equal(t, true, ok)
}

func TestAnyType(t *testing.T) {
	type innerStruct struct {
		A int
		B string
		C func()
		D *testing.T
	}
	var (
		a int = 1
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
	)
	curr, err := b0gus_misc_utils.GetFieldValueByName(a, "")
	assert.Equal(t, nil, err)
	t.Logf("%v", curr)
	var aa any
	assert.IsNotType(t, struct{}{}, nil)
	assert.IsNotType(t, struct{}{}, aa)

	b_name := b0gus_misc_utils.GetTypeNameViaType(b)
	assert.NotEqual(t, "", b_name)
	fmt.Println(b_name)

	c_name := b0gus_misc_utils.GetTypeNameViaType(c.HellYeah)
	assert.NotEqual(t, "", c_name)
	fmt.Println(c_name)
	c_name = b0gus_misc_utils.GetTypeNameViaType(c.WhatCanIsay)
	assert.NotEqual(t, "", c_name)
	fmt.Println(c_name)

	c_name = b0gus_misc_utils.GetTypeNameViaType(c.JustTestIt)
	assert.NotEqual(t, "", c_name)
	fmt.Println(c_name)

	c_name = b0gus_misc_utils.GetTypeNameViaType(c.AnOpenFunc)
	assert.NotEqual(t, "", c_name)
	fmt.Println(c_name)

	c_name = b0gus_misc_utils.GetTypeNameViaType(c.ManHaha)
	assert.NotEqual(t, "", c_name)
	fmt.Println(c_name)

	c_name = b0gus_misc_utils.GetTypeNameViaType(c)
	assert.NotEqual(t, "", c_name)
	fmt.Println(c_name)

	d_name := b0gus_misc_utils.GetTypeNameViaType(d)
	assert.NotEqual(t, "", d_name)
	fmt.Println(d_name)

	e_name := b0gus_misc_utils.GetTypeNameViaType(e)
	assert.NotEqual(t, "", e_name)
	fmt.Println(e_name)
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
			str, num := b0gus_misc_utils.IPaddrSplit(tt.input)
			assert.Equal(t, tt.expected.Addr, str, "wrong addr")
			assert.Equal(t, tt.expected.Port, num, "wrong port")
		})
	}
}

func TestPemHelper(t *testing.T) {

	var (
		pem_path = "../assets/test.pem"
		del_pem  = func() {
			err := os.Remove(pem_path)
			assert.Equal(t, err, nil, "Unable to delete test.pem")
		}
	)

	what_we_have := b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "ed25519", 0)
	assert.NotEqual(t, what_we_have, nil, "Still got an nil after creating")
	del_pem()
	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "elliptic", 311)
	assert.Equal(t, what_we_have, nil, "unexpected nil pem")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "elliptic", 521)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	del_pem()

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "elliptic", 384)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	del_pem()

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "elliptic", 0)
	assert.Equal(t, what_we_have, nil, "unexpected nil pem")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "ell1ptic", 0)
	assert.Equal(t, what_we_have, nil, "unexpected nil pem")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "rsa", 384)
	assert.Equal(t, what_we_have, nil, "unexpected nil pem")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "rsa", 1024)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	del_pem()

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "rsa", 2048)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	del_pem()

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "rsa", 4096)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	del_pem()
	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "rsa", 1234)
	assert.Equal(t, what_we_have, nil, "unexpected nil pem")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "ed25519", 123456)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	what_we_have = nil
	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "ed25519", 123456)
	assert.NotEqual(t, what_we_have, nil, "unexpected nil pem")
	del_pem()

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "ecdh", 123456)
	assert.Equal(t, what_we_have, nil, "Still got an nil after creating")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "dsa", 123456)
	assert.Equal(t, what_we_have, nil, "Still got an nil after creating")

	what_we_have = b0gus_crypto_aux.LoadOrCreateSSHpem(pem_path, "sm2", 123456)
	assert.Equal(t, what_we_have, nil, "Still got an nil after creating")
}
