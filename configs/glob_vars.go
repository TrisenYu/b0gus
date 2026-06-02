package configs

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"b0gus/assets"

	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// since go use compiler rather than interpreter, we can not assign a . as relative path
// otherwise the executable file will deem there is a configuration in the same directory as its config.
// and config can be updated from different sources like trusted remote network activities
// or remote client's commands?

// TODO: shall we record the configuration change as a log?

var (
	GlobConf atomic.Pointer[LocalConfig]
	// LocalConfigPathAsStr is a relative path defined in the perspective of `b0gus.go`.
	// Read-only except the diff_tests.
	LocalConfigPathAsStr = "./configs/config.toml"

	versionStr       string
	buildTimeStr     string
	hashValStr       string
	builtByStr       string
	RemotePullSource string
)

// InitConfFlags will
func InitConfFlags() {
	flag.StringVar(
		&LocalConfigPathAsStr, "conf-path", "./configs/config.toml",
		GetLocalizedMsg("meta_conf.ConfPath", nil),
	)
	GlobConf.Store(LoadDefaultConfig(""))
	f := flag.Lookup("conf-path")
	if f != nil { // reset the configuration path description if feasible
		f.Usage = GetLocalizedMsg("meta_conf.ConfPath", nil)
	}
	showVersion := flag.Bool("version", false, "")
	flag.BoolVar(showVersion, "v", false, "")
	flag.StringVar(&RemotePullSource, "remote-pull-source", ":65431", "")

	flag.Parse()
	if *showVersion {
		var sb strings.Builder
		sb.WriteString("b0gus Version: ")
		sb.WriteString(versionStr)
		sb.WriteRune('-')
		sb.WriteString(BuildTypeStr)
		sb.WriteString("\nBuild Time:    ")
		sb.WriteString(buildTimeStr)
		sb.WriteString("\nBuild Hash:    ")
		sb.WriteString(hashValStr)
		sb.WriteString("\nBuild Name:    ")
		sb.WriteString(builtByStr)
		sb.WriteString("\nCurrent ISA:   ")
		sb.WriteString(runtime.GOARCH)
		sb.WriteString("\nCurrent OS:    ")
		sb.WriteString(runtime.GOOS)
		println(sb.String())
		os.Exit(0)
	} else if GlobConf.Load() == nil {
		payload := GetLocalizedMsg("main.FailToApplyConfiguration", nil)
		Logger().Fatal(payload)
	}
}

// GetFilePathUnderConfigDir will return absolute path of LocalConfigPathAsStr/../x
// as its result.
func GetFilePathUnderConfigDir(x string) string {
	x, _ = filepath.Abs(filepath.Join(filepath.Dir(LocalConfigPathAsStr), x))
	return x
}

// GetLang Access expected language defined in configuration.
// When the pointee is nil or the definition in configuration is empty, return "en" as default value.
func GetLang() string {
	snapshot := GlobConf.Load()
	if snapshot == nil {
		return "en"
	}
	res := snapshot.Language
	if len(res) == 0 {
		return "en"
	}
	return res
}

// GetLocalizedMsg will fill arguments into translated message and
// return it if there is not any error. Otherwise, it will only return the messageID.
func GetLocalizedMsg(msgID string, args map[string]any) string {
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
