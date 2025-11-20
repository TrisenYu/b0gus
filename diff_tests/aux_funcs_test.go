package diff_tests

import (
	b0gus_config "b0gus/configs"
	b0gus_misc_utils "b0gus/misc_utils"
	"fmt"

	"path/filepath"
	"testing"
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
