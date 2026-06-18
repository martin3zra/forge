# i18n

Locale-agnostic translation loading. The package reads JSON locale files from a filesystem you own and embed; it never bundles locales itself.

## Locale files

Place files at `locales/<lang>.json` in your filesystem (typically an `embed.FS`). Values may be nested:

```json
{
  "auth": {
    "failed": "These credentials do not match our records."
  }
}
```

## Loading

```go
//go:embed locales/*.json
var localesFS embed.FS

trans, err := i18n.LoadTranslations(localesFS, "es", "en", "auth", "validation")
// trans["auth.failed"] == "..."
```

`LoadTranslations(fsys, lang, fallbackLang, namespaces...)`:

- Loads `lang`, falling back to `fallbackLang` per missing key.
- Restricts output to the requested `namespaces` (top-level keys).
- Flattens nested objects into dotted keys (`auth.failed`) and returns a flat `map[string]string`.

Pass the returned map to the frontend (e.g. as Inertia shared props) or look up keys server-side.
