// Package configs
package configs

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025

import (
	"b0gus/assets"
	"sync/atomic"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// since go use compiler rather than interpreter, we can not assign a . as relative path
// otherwise the executable file will deem there is a configuration in the same directory as its configuration

// TODO: can we use environment variables as meta configuration?
//		can updates come from different sources like trusted remote network activities or remote client's commands?
//		should we record the configuration change as a log?

var (
	GlobConf atomic.Pointer[LocalConfig]

	AssetsDirAsStr string
	// LocalConfigPathAsStr is a relative path defined in the perspective of `b0gus-arm64.go`.
	// Read-only except the diff_tests.
	LocalConfigPathAsStr   = "./configs/config.toml"
	RemotePullSource       string
	LocalRootCaCertAbsPath string
	LocalRootCaKeyAbsPath  string
)

// GetLang Access expected language defined in configuration.
func GetLang() string {
	snapshot := GlobConf.Load()
	if snapshot == nil {
		return "en"
	}
	res := snapshot.ServerConfig.Language
	if len(res) == 0 {
		return "en"
	}
	return res
}

// GetLocalizedMsg will fill arguments into translated message and
// return it if there is not any error. Otherwise, it will only return the messageID.
func GetLocalizedMsg(
	msgID string, args map[string]any,
) string {
	msg, err := i18n.NewLocalizer(assets.Bundle, GetLang()).Localize(
		&i18n.LocalizeConfig{
			MessageID:    msgID,
			TemplateData: args,
		},
	)
	if err != nil {
		return msgID
	}
	return msg
}
