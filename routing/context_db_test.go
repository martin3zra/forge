package routing_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/playsql"

	_ "modernc.org/sqlite" // registers the "sqlite" driver (pure Go, no CGO)
)

// testDB opens an isolated in-memory sqlite handle and wraps it with
// playsql.Use, mirroring playsql's own use_test.go fixture — it never dials a
// real server, so no network/CGO dependency is introduced just to prove a
// *playsql.DB round-trips through the request context.
func testDB(t *testing.T) *playsql.DB {
	t.Helper()
	raw, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	raw.SetMaxOpenConns(1) // :memory: is per-connection; pin to one
	t.Cleanup(func() { raw.Close() })

	db, err := playsql.Use(raw, "sqlite")
	if err != nil {
		t.Fatalf("playsql.Use: %v", err)
	}
	return db
}

// withDB simulates what the application's own boot-time middleware (e.g.
// shop's BindMiddleware) already does today: bind database.PlaysqlKey{} once
// per request. It isn't part of forge's public API — forge doesn't need to
// provide a binding mechanism, since Context.DB() only ever reads a value the
// application already knows how to set via plain context.WithValue.
func withDB(db *playsql.DB) routing.Middleware {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(ctx *routing.Context) {
			next(ctx.WithContext(context.WithValue(ctx.Request.Context(), database.PlaysqlKey{}, db)))
		}
	}
}

// TestContextDB_ReturnsBoundDB proves ctx.DB() returns exactly the *playsql.DB
// instance bound to the request, reached through a real handler dispatch.
func TestContextDB_ReturnsBoundDB(t *testing.T) {
	want := testDB(t)

	var got *playsql.DB
	r := routing.New()
	r.WithMiddleware(withDB(want))
	r.GET("/ping", func(ctx *routing.Context) {
		got = ctx.DB()
		ctx.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if got != want {
		t.Fatalf("ctx.DB() = %p, want %p", got, want)
	}
}

// TestContextDB_NoCrossRequestState proves two requests bound to different
// *playsql.DB instances never see each other's connection — there is no
// shared/global slot a second request could accidentally read.
func TestContextDB_NoCrossRequestState(t *testing.T) {
	dbA := testDB(t)
	dbB := testDB(t)

	var gotA, gotB *playsql.DB
	r := routing.New()
	r.GET("/a", withDB(dbA)(func(ctx *routing.Context) {
		gotA = ctx.DB()
		ctx.Text(http.StatusOK, "a")
	}))
	r.GET("/b", withDB(dbB)(func(ctx *routing.Context) {
		gotB = ctx.DB()
		ctx.Text(http.StatusOK, "b")
	}))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/a", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/b", nil))

	if gotA != dbA {
		t.Errorf("request A: ctx.DB() = %p, want %p", gotA, dbA)
	}
	if gotB != dbB {
		t.Errorf("request B: ctx.DB() = %p, want %p", gotB, dbB)
	}
	if gotA == gotB {
		t.Fatalf("request A and B resolved to the same *playsql.DB (%p) — state leaked across requests", gotA)
	}
}

// TestContextDB_PanicsWhenUnbound proves a request that never had
// database.PlaysqlKey bound fails loudly rather than handing back nil for the
// caller to dereference.
func TestContextDB_PanicsWhenUnbound(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("ctx.DB() did not panic when no DB was bound")
		}
	}()

	ctx := &routing.Context{
		Response: httptest.NewRecorder(),
		Request:  httptest.NewRequest("GET", "/no-db", nil),
	}
	ctx.DB()
}
