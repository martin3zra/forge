package mailer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIReturnsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Error("missing bearer token")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	m := New(Config{Driver: API, Host: server.URL, Password: "secret"}, testTemplates).To("grace@example.com", "Grace")
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("API rejection: %v", err)
	}
	m.cfg.Host = ":invalid"
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil {
		t.Fatal("invalid API URL accepted")
	}
}
