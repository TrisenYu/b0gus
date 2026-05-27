package diff_tests

import (
	"b0gus/crypto_aux"
	"testing"

	"github.com/stretchr/testify/assert"
)

var hashAlgCases = []struct {
	Payload string
	State   bool
}{
	{"", false},
	{"abcdefg", false},
	{"123456", false},
	{"\x00\u1234", false},
	{"3.1415926e+19", false},
	{"blake256", true},
	{"sha256", true},
	{"sm3", true},
	{"sm2", true},
	{"rsa", false},
	{"我的世界", false},
	{"我的世界", false},
	{"\t\n", false},
	{" ", false},
	{"おはよう", false},
	{"blake2s", true},
	{"blake384", true},
	{"blake512", true},
	{"sha1", true},
	{"sha3", true},
	{"sha3-384", true},
	{"sha224", true},
	{"sha256", true},
	{"sha384", true},
	{"sha3-512", true},
	{"sha512", true},
	{"md", true},
	{"md4", true},
	{"mp3", true},
	{"mp5", true},
	{"mp7", true},
	{"md5", true},
}

func TestHash(t *testing.T) {
	for _, s := range hashAlgCases {
		res := crypto_aux.OnceHashByChoice(s.Payload)(nil, nil)
		assert.True(t, len(res) > 0)
		res = crypto_aux.OnceHashByChoice(s.Payload)([]byte(s.Payload), nil)
		assert.True(t, len(res) > 0)
		res = crypto_aux.OnceHashByChoice(s.Payload)(nil, []byte(s.Payload))
		assert.True(t, len(res) > 0)
	}
}

func FuzzHash(f *testing.F) {
	for _, seed := range hashAlgCases {
		f.Add(seed.Payload)
	}
	f.Fuzz(func(t *testing.T, data string) {
		res := crypto_aux.OnceHashByChoice(data)(nil, nil)
		assert.True(t, len(res) > 0)
		res = crypto_aux.OnceHashByChoice(data)([]byte(data), nil)
		assert.True(t, len(res) > 0)
		res = crypto_aux.OnceHashByChoice(data)(nil, []byte(data))
		assert.True(t, len(res) > 0)
		res = crypto_aux.OnceHashByChoice(data)([]byte(data), []byte(data))
		assert.True(t, len(res) > 0)
	})
}
