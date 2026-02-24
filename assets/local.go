package assets

import (
	"embed"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml"
	"golang.org/x/text/language"
)

//go:embed locale/*.toml
var local_description embed.FS

var bundle *i18n.Bundle

func init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	entries, _ := local_description.ReadDir("locale")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		filePath := filepath.Join("locale", e.Name())
		content, err := local_description.ReadFile(filePath)
		if err != nil {
			continue
		}
		bundle.ParseMessageFileBytes(content, e.Name())
	}
}

func GetLocalizedMsg(
	lang, msgID string,
	data map[string]any,
) string {
	localizer := i18n.NewLocalizer(bundle, lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: data,
	})
	if err != nil {
		return msgID
	}
	return msg
}
