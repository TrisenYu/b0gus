// SPDX-LICENSE-IDENTIFIER: 3-Clause-BSD
//
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
// Created at 2025/11/15 星期六 20:05:01
// Last modified at 2025/11/15 星期六 22:24:47
package configs

const (
	// since go use compiler rather than interpreter, we can not assign a . as relative path
	// otherwise the executable file will deem there is a configuration in the same direnctory as its,
	Config_dir_as_str  string = "./configs/"
	Assets_dir_as_str  string = "./assets/"
	Config_path_as_str string = Config_dir_as_str + "config.toml"
)

var (
	curr_config LocalConfig
	/* UpdateFlag requires manually close after the whole lifetime of b0gus */
	UpdateFlag           chan struct{} = make(chan struct{}, 1)
	GlobConfigMaintainer ConfigMaintainer
)
