package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
)

//go:embed locales/*.json
var localeFS embed.FS

var messages = loadMessages()

func loadMessages() map[string]map[string]string {
	entries, err := fs.ReadDir(localeFS, "locales")
	if err != nil {
		panic(fmt.Errorf("read locales dir: %w", err))
	}

	loaded := make(map[string]map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}
		raw, err := localeFS.ReadFile(filepath.Join("locales", name))
		if err != nil {
			panic(fmt.Errorf("read locale %s: %w", name, err))
		}
		var messagesForLocale map[string]string
		if err := json.Unmarshal(raw, &messagesForLocale); err != nil {
			panic(fmt.Errorf("parse locale %s: %w", name, err))
		}
		locale := name[:len(name)-len(filepath.Ext(name))]
		loaded[locale] = messagesForLocale
	}

	return loaded
}

func translate(locale string, key string) string {
	if localeMessages, ok := messages[locale]; ok {
		if message, ok := localeMessages[key]; ok {
			return message
		}
	}
	if locale != DefaultLocale {
		if message, ok := messages[DefaultLocale][key]; ok {
			return message
		}
	}
	return key
}
