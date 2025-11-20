package diff_tests

import (
	"fmt"
	"path/filepath"
	"testing"

	b0gus_config "b0gus/configs"
	b0gus_misc_utils "b0gus/misc_utils"

	assert "github.com/stretchr/testify/assert"
)

func TestConfigReader(t *testing.T) {
	// assemble to b0gus_config.Config_path_as_str
	exam_conf_path := "../configs/example.toml"
	test_conf_path, _ := filepath.Abs(exam_conf_path)
	local_conf := b0gus_config.TomlConfigReader(test_conf_path)
	res_map := b0gus_misc_utils.TurnStruct2Map(local_conf)
	if res_map == nil {
		t.Error("reflection on struct failed!")
	}
	for k, v := range res_map["ServerConfig"].(map[string]any) {
		fmt.Println(k, v)
		switch k {
		case "PemName":
			fallthrough
		case "SSH":
			fallthrough
		case "Telnet":
			fallthrough
		case "Database":
			t.Log(k, v)
		default:
			t.Errorf("not matched fields %s were found in `local_conf`", k)
		}
	}
}

func TestIPvXparser(t *testing.T) {
	tests := []struct {
		input    string
		expected struct {
			Addr string
			Port uint16
		}
	}{
		{
			"127.0.0.1:1234",
			struct {
				Addr string
				Port uint16
			}{"127.0.0.1", 1234},
		},
		{
			"::1:1234",
			struct {
				Addr string
				Port uint16
			}{"::1", 1234},
		},
		{
			"::1:123444",
			struct {
				Addr string
				Port uint16
			}{"", 0},
		},
		{
			"3.1.4.5:26",
			struct {
				Addr string
				Port uint16
			}{"3.1.4.5", 26},
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
