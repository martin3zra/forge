package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

// LoadTranslations reads the requested language (with an optional fallback) from
// the provided filesystem. The caller owns the locale files and their embedding;
// i18n stays locale-agnostic. Locale files are expected at "locales/<lang>.json".
func LoadTranslations(fsys fs.FS, lang string, fallbackLang string, namespaces ...string) (map[string]string, error) {
	primary, err := loadLangFromFS(fsys, lang)
	if err != nil {
		return nil, fmt.Errorf("load primary language: %w", err)
	}

	fallback := map[string]interface{}{}
	if fallbackLang != "" && fallbackLang != lang {
		fallback, _ = loadLangFromFS(fsys, fallbackLang)
	}

	result := make(map[string]string)

	for _, ns := range namespaces {
		// Try from primary
		val, ok := getNestedKey(primary, ns)
		if !ok {
			// fallback
			val, ok = getNestedKey(fallback, ns)
		}

		if ok {
			if sub, ok := val.(map[string]interface{}); ok {
				flat := flatten(sub, ns)
				for k, v := range flat {
					result[k] = v
				}
			}
		}
	}

	return result, nil
}

func loadLangFromFS(fsys fs.FS, lang string) (map[string]any, error) {
	filePath := fmt.Sprintf("locales/%s.json", lang)
	data, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", filePath, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}
	return out, nil
}

// Replacements is a map for placeholder replacements
type Replacements map[string]string

// Translator holds the translations map plus the locale source used to lazily
// load additional namespaces. The locale source (fsys), primary language and
// fallback are supplied by the caller so the framework stays locale-agnostic.
type Translator struct {
	Translations map[string]string
	fsys         fs.FS
	lang         string
	fallback     string
}

// NewTranslator creates a new Translator bound to a locale source.
func NewTranslator(fsys fs.FS, lang, fallback string, translations map[string]string) *Translator {
	if translations == nil {
		translations = map[string]string{}
	}
	return &Translator{
		Translations: translations,
		fsys:         fsys,
		lang:         lang,
		fallback:     fallback,
	}
}

func (t *Translator) Load(namespaces ...string) error {
	translations, err := LoadTranslations(t.fsys, t.lang, t.fallback, namespaces...)
	if err != nil {
		return fmt.Errorf("load translations: %w", err)
	}
	t.mergeMaps(t.Translations, translations)
	return nil
}

func (t *Translator) mergeMaps(dst, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

// Trans returns the translated string with replacements
func (t *Translator) Trans(key string, replacements ...Replacements) string {
	translation, ok := t.Translations[key]
	if !ok {
		translation = key
	}

	if len(replacements) == 0 {
		return translation
	}

	for k, v := range replacements[0] {

		nounKey := getNounFromKey(v)
		if strings.HasPrefix(v, "@") {
			refKey := strings.TrimPrefix(v, "@")
			if val, exists := t.Translations[refKey]; exists {
				v = val
			}
		}
		translation = strings.ReplaceAll(translation, ":"+k, v)

		gender, ok := t.Translations[fmt.Sprintf("global.nouns.%s.gender", nounKey)]
		if ok {
			actionNoun := "a"
			if gender == "m" {
				actionNoun = "o"
			}
			translation = strings.ReplaceAll(translation, "@action", actionNoun)
		} else {
			translation = strings.ReplaceAll(translation, "@action", "o")
		}

	}

	return translation
}

func getNounFromKey(key string) string {
	// Remove leading "@" if present
	key = strings.TrimPrefix(key, "@")

	// Split by dot
	parts := strings.Split(key, ".")

	if len(parts) > 1 {
		return parts[1] // returns "customer" or "invoice"
	}

	return ""
}
