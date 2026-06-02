package assets

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"embed"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml"
	"golang.org/x/text/language"
)

var (
	//go:embed locale/go-proj/*.toml
	localDescription embed.FS

	localeDir = "locale/go-proj/"
	Bundle *i18n.Bundle
)

func init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	entries, _ := localDescription.ReadDir(localeDir[:len(localeDir)-1])
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		content, err := localDescription.ReadFile(localeDir + e.Name())
		if err != nil {
			continue
		}
		_, _ = Bundle.ParseMessageFileBytes(content, e.Name())
	}
}
