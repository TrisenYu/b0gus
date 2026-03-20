package assets

import (
	"embed"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml"
	"golang.org/x/text/language"
)

//go:embed locale/*.toml
var localDescription embed.FS

var Bundle *i18n.Bundle

func init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	entries, _ := localDescription.ReadDir("locale")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		content, err := localDescription.ReadFile("locale/" + e.Name())
		if err != nil {
			continue
		}
		Bundle.ParseMessageFileBytes(content, e.Name())
	}
}
