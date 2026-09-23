package routing_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/playsql"
	"github.com/romsar/gonertia/v3"
)

// TestErrorMapsNotFoundTo404 pins ctx.Error's not-found mapping: both the
// stdlib sentinel and playsql's own (First/Find return playsql.ErrNotFound,
// which does not wrap sql.ErrNoRows) render with 404, wrapped or not.
func TestErrorMapsNotFoundTo404(t *testing.T) {
	i, err := gonertia.New(`<html><body>{{ .inertia }}</body></html>`)
	if err != nil {
		t.Fatalf("gonertia.New: %v", err)
	}

	cases := map[string]struct {
		err  error
		want int
	}{
		"sql.ErrNoRows":            {sql.ErrNoRows, http.StatusNotFound},
		"playsql.ErrNotFound":      {playsql.ErrNotFound, http.StatusNotFound},
		"wrapped playsql notfound": {fmt.Errorf("get account: %w", playsql.ErrNotFound), http.StatusNotFound},
		"other error":              {errors.New("boom"), http.StatusInternalServerError},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := &routing.Context{
				Response: rec,
				Request:  httptest.NewRequest(http.MethodGet, "/accounts/x", nil),
				Inertia:  i,
			}

			ctx.Error(tc.err)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}
