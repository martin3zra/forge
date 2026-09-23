package i18n

import (
	"context"
	"net/http"

	"golang.org/x/text/language"
)

// LocaleKey is the context key under which the request's resolved locale
// (e.g. "en", "es") is stored. Packages that localize output — the validator,
// the app's translator — read it via Locale.
type LocaleKey struct{}

// WithLocale returns a copy of ctx carrying lang as the request locale.
func WithLocale(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, LocaleKey{}, lang)
}

// Locale returns the locale stored in ctx by WithLocale (or LocaleMiddleware).
func Locale(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	lang, ok := ctx.Value(LocaleKey{}).(string)
	return lang, ok && lang != ""
}

// NegotiateLocale picks the best entry of supported for an Accept-Language
// header value (e.g. "es-DO,es;q=0.9,en;q=0.8"), honoring q-weights and
// regional fallbacks ("es-DO" matches "es"). It returns fallback when the
// header is empty, malformed, or matches nothing in supported. The returned
// value is always one of supported's entries verbatim, or fallback.
func NegotiateLocale(acceptLanguage string, supported []string, fallback string) string {
	if acceptLanguage == "" || len(supported) == 0 {
		return fallback
	}

	desired, _, err := language.ParseAcceptLanguage(acceptLanguage)
	if err != nil || len(desired) == 0 {
		return fallback
	}

	tags := make([]language.Tag, len(supported))
	for i, s := range supported {
		tags[i] = language.Make(s)
	}

	_, index, confidence := language.NewMatcher(tags).Match(desired...)
	if confidence == language.No {
		return fallback
	}

	return supported[index]
}

// LocaleMiddleware resolves the request locale from the Accept-Language
// header against supported and stores it in the request context via
// WithLocale. A locale already present in the context (set by an earlier
// middleware, e.g. from the user's saved preference or a cookie) wins and is
// left untouched, so place this after any such middleware.
func LocaleMiddleware(supported []string, fallback string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := Locale(r.Context()); ok {
				next.ServeHTTP(w, r)
				return
			}

			lang := NegotiateLocale(r.Header.Get("Accept-Language"), supported, fallback)
			next.ServeHTTP(w, r.WithContext(WithLocale(r.Context(), lang)))
		})
	}
}
