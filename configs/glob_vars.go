// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025
package configs

const (
	// since go use compiler rather than interpreter, we can not assign a . as relative path
	// otherwise the executable file will deem there is a configuration in the same direnctory as its,
	ConfigDirAsStr  string = "./configs/"
	AssetsDirAsStr  string = "./assets/"
	ConfigPathAsStr string = ConfigDirAsStr + "config.toml"
)

var (
	curr_config LocalConfig
	/* UpdateFlag requires manually close after the whole lifetime of b0gus */
	UpdateFlag           chan *LocalConfig = make(chan *LocalConfig, 1)
	GlobConfigMaintainer ConfigMaintainer
)
