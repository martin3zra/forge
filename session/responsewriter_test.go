package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The Secure flag follows the manager's setting alone: behind a
// TLS-terminating proxy the request is plain HTTP, and the cookie must
// still be Secure.
func TestSessionCookieSecureFollowsSetting(t *testing.T) {
	for _, secure := range []bool{true, false} {
		m := &SessionManager{cookieName: "s", idleExpiration: time.Hour, secure: secure, httpOnly: true}
		r := httptest.NewRequest(http.MethodGet, "http://app.internal/", nil) // no TLS
		r = r.WithContext(context.WithValue(r.Context(), SessionContextKey{}, &Session{Id: "abc"}))
		rec := httptest.NewRecorder()

		w := &sessionResponseWriter{ResponseWriter: rec, sessionManager: m, request: r}
		w.WriteHeader(http.StatusOK)

		cookies := rec.Result().Cookies()
		if len(cookies) != 1 {
			t.Fatalf("secure=%v: %d cookies", secure, len(cookies))
		}
		if cookies[0].Secure != secure {
			t.Errorf("secure=%v: cookie Secure=%v", secure, cookies[0].Secure)
		}
	}
}
