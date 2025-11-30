// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
package diff_tests

import (
	"fmt"
	"os"
	"testing"

	b0gus_crypto_aux "b0gus/crypto_aux"
	b0gus_misc_utils "b0gus/misc_utils"

	assert "github.com/stretchr/testify/assert"
)

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
		// {
		// 	"fe80::1cc0:3e8c:119f:c2e1%ens18/1234",
		// 	ipPort{"fe80::1cc0:3e8c:119f:c2e1", 1234},
		// },
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
