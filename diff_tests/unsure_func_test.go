package diff_tests

/// Last modified at 2026/05/18 星期一 10:16:57
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"b0gus/crypto_aux"
	"b0gus/internal/misc_utils"
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
			"What the hell is wrong with this test case:123",
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

func TestIsIP(t *testing.T) {
	assert.True(t, misc_utils.IsIP("127.0.0.1"))
	assert.True(t, misc_utils.IsIP("2001:db8::1"))
	assert.False(t, misc_utils.IsIP("256.0.0.1"))
	assert.False(t, misc_utils.IsIP(""))
	assert.False(t, misc_utils.IsIP("localhost"))
}

func TestIsHostnameRFC1123(t *testing.T) {
	assert.True(t, misc_utils.IsHostnameRFC1123("localhost"))
	assert.True(t, misc_utils.IsHostnameRFC1123("my-host"))
	assert.True(t, misc_utils.IsHostnameRFC1123("a.example.com"))
	assert.False(t, misc_utils.IsHostnameRFC1123("-host"))
	assert.False(t, misc_utils.IsHostnameRFC1123("host_underscore"))
}

func TestIsPort(t *testing.T) {
	assert.True(t, misc_utils.IsPort(1))
	assert.True(t, misc_utils.IsPort(80))
	assert.True(t, misc_utils.IsPort(65535))
	assert.False(t, misc_utils.IsPort(0))
	assert.False(t, misc_utils.IsPort(65536))
}

func TestIsURL(t *testing.T) {
	assert.True(t, misc_utils.IsURL("https://example.com/path"))
	assert.True(t, misc_utils.IsURL("file:///tmp/file"))
	assert.False(t, misc_utils.IsURL("http://"))
	assert.False(t, misc_utils.IsURL("just-text"))
}

func TestStripMarkdownSignIfAny(t *testing.T) {
	cases := []struct {
		input  string
		expect string
	}{
		{"```go\nhello world\n```", "hello world"},
		{"no code block", "no code block"},
		{"```", ""},
		{"``````", ""},
		{"`````````", ""},
		{"```t```", ""},
		{"```\nt\n```", ""},
		{"t```", ""},
		{"", ""},
		{"t", "t"},
		{"```toml\nhello world```", ""},
		{"```toml\nhello world\n```", "hello world"},
		{"```tomlhello world", ""},
		{"```toml\n你好输出\n```", "你好输出"},
		{"```你好输出\ntoml\n```", "toml"},
		{strings.Repeat("`", 100), ""},
		{"```a\nq" + strings.Repeat("```a\nq", 11), ""},
		{`{"helo": [1, 2, 3], "world": "456"}`, `{"helo": [1, 2, 3], "world": "456"}`},
		{"```json\n{\"helo\": [1, 2, 3], \"world\": \"456\"}\n```", "{\"helo\": [1, 2, 3], \"world\": \"456\"}"},
		{"```json\\n{\"helo\": [1, 2, 3], \"world\": \"456\"}\\n```", ""},
		{"a```b", ""},
	}
	for _, c := range cases {
		actual := misc_utils.StripMarkdownSignIfAny(c.input)
		assert.Equal(t, c.expect, actual)
	}
}
