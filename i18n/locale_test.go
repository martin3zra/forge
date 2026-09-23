package i18n_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martin3zra/forge/i18n"
)

func TestNegotiateLocale(t *testing.T) {
	supported := []string{"en", "es"}

	cases := []struct {
		header string
		want   string
	}{
		{"", "en"},
		{"es", "es"},
		{"es-DO,es;q=0.9,en;q=0.8", "es"},
		{"fr-FR,en;q=0.5", "en"},
		{"en;q=0.2,es;q=0.9", "es"},
		{"fr, de", "en"},
		{";;;garbage", "en"},
	}

	for _, c := range cases {
		if got := i18n.NegotiateLocale(c.header, supported, "en"); got != c.want {
			t.Errorf("NegotiateLocale(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}

func TestLocaleMiddleware(t *testing.T) {
	var seen string
	handler := i18n.LocaleMiddleware([]string{"en", "es"}, "en")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = i18n.Locale(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es-DO,es;q=0.9")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "es" {
		t.Errorf("header negotiation: got %q, want %q", seen, "es")
	}

	// A locale set upstream (e.g. from the user's saved preference) wins.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es")
	req = req.WithContext(i18n.WithLocale(context.Background(), "en"))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "en" {
		t.Errorf("upstream locale: got %q, want %q", seen, "en")
	}
}
