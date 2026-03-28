// Package configs
package configs

// SPDX-LICENSE-IDENTIFIER: 3-Clauses-BSD
// (C) All rights reserved. Author: <kisfg@hotmail.com> in 2025

import (
	"b0gus/assets"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// since go use compiler rather than interpreter, we can not assign a . as relative path
// otherwise the executable file will deem there is a configuration in the same directory as its configuration

// TODO: can we use environment variables as meta configuration?
//		can updates come from different sources like trusted remote network activities or remote client's commands?
//		should we record the configuration change as a log?

var (
	GlobConfigMan ConfigMaintainer

	confLock = sync.Mutex{}
	// UpdateFlag requires for closing the whole services of b0gus
	UpdateFlag     = make(chan *LocalConfig, 1)
	AssetsDirAsStr string
	// LocalConfigPathAsStr is relative path in the perspective of `b0gus.go`.
	LocalConfigPathAsStr   = "./configs/config.toml"
	RemotePullSource       string
	LocalRootCaCertAbsPath string
	LocalRootCaKeyAbsPath  string
	// TrustedCertAbsPath for outer clients to verify, the trusted cert should generate in advanced.
	TrustedCertAbsPath string
)

// Runtime updates
var (
	PermitSelfAlterInRuntime  bool // control whether meta-configuration can change
	PermitConfAlterInRuntime  bool
	PermitShellAlterInRuntime bool
	PermitRemotePullInRuntime bool
)

// GetLang Access expected language defined in configuration.
func GetLang() string {
	confLock.Lock()
	defer confLock.Unlock()

	return currConfig.ServerConfig.Language
}

// GetLocalizedMsg will fill arguments into translated message and
// return it if there is not any error. Otherwise, it will only return the messageID.
func GetLocalizedMsg(
	msgID string,
	args map[string]any,
) string {
	msg, err := i18n.NewLocalizer(
		assets.Bundle, GetLang(),
	).Localize(
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
