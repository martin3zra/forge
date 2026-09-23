package routing_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/support"
	"github.com/romsar/gonertia/v3"
)

type nameForm struct {
	support.FormRequest
	Name string `json:"name"`
}

func (nameForm) Rules() map[string]any {
	return map[string]any{"name": "required"}
}

// TestWithRequestValidationFlashesFieldErrorsOnly pins that a failed form
// submit leaves exactly the per-field errors in the session — no duplicate
// "status" entry holding the same errors as a raw JSON string.
func TestWithRequestValidationFlashesFieldErrorsOnly(t *testing.T) {
	i, err := gonertia.New(`<html><body>{{ .inertia }}</body></html>`)
	if err != nil {
		t.Fatalf("gonertia.New: %v", err)
	}

	// No cookie on the request, so Start never touches the store.
	manager := session.NewSessionManager(nil, time.Hour, time.Hour, time.Hour, "test", "", false, true)
	req := httptest.NewRequest(http.MethodPost, "/things", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Inertia", "true")
	sess, req := manager.Start(req)

	called := false
	handler := routing.WithRequest(func(ctx *routing.Context, form *nameForm) { called = true })

	rec := httptest.NewRecorder()
	handler(&routing.Context{Response: rec, Request: req, Inertia: i})

	if called {
		t.Fatal("handler ran despite failed validation")
	}
	var errs map[string][]string
	switch v := sess.Get("errors").(type) {
	case map[string][]string:
		errs = v
	case foundation.ErrorBag:
		errs = v
	}
	if len(errs["name"]) == 0 {
		t.Fatalf("errors = %v, want a name error", sess.Get("errors"))
	}
	if _, ok := errs["status"]; ok {
		t.Fatalf("errors[status] = %q, want no status entry", errs["status"])
	}
}

// TestBackWithAfterFormErrors pins that a status error can be added after
// form errors on a fresh session — FormErrors used to store ErrorBag, which
// Errors' map[string][]string assertion panicked on.
func TestBackWithAfterFormErrors(t *testing.T) {
	manager := session.NewSessionManager(nil, time.Hour, time.Hour, time.Hour, "test", "", false, true)
	sess, _ := manager.Start(httptest.NewRequest(http.MethodPost, "/things", nil))

	sess.FormErrors(foundation.ErrorBag{"name": {"required"}})
	sess.Errors("status", "boom")

	errs, _ := sess.Get("errors").(map[string][]string)
	if len(errs["name"]) == 0 || len(errs["status"]) == 0 {
		t.Fatalf("errors = %v, want name and status", sess.Get("errors"))
	}
}
