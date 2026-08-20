package routing_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martin3zra/forge/routing"
)

func marker(name string) routing.Middleware {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(ctx *routing.Context) {
			ctx.Response.Header().Add("X-MW", name)
			next(ctx)
		}
	}
}

// Documents a known limitation of the r.WithMiddleware(x).Group(fn) chain: x
// mutates r.middlewares in place before Group ever runs, so it leaks onto
// every route r registers afterward, not just the ones inside fn. This mirrors
// the exact nesting that broke acme's app/route.go — RequiresVariants, added
// via WithMiddleware right before the /attributes Group, ended up applied to
// /home, /invoices and everything else in the enclosing group too. Use
// GroupWithMiddleware (see TestNestedGroupWithMiddlewareScoping) to avoid this.
func TestGroupChainedWithMiddlewareLeaksToSiblings(t *testing.T) {
	root := routing.New()

	root.WithMiddleware(marker("Auth")).Group(func(a *routing.Router) {
		a.GET("/a", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "a") })

		a.WithMiddleware(marker("Mid")).Group(func(b *routing.Router) {
			b.GET("/b", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "b") })

			b.WithMiddleware(marker("Inner")).Group(func(c *routing.Router) {
				c.GET("/c", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "c") })
			})

			b.GET("/d", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "d") })
		})
	})

	// The leak: Mid and Inner end up on routes that never should have seen them.
	leaked := map[string][]string{
		"/a": {"Auth", "Mid"},         // Mid leaked backward onto a route registered before it existed
		"/b": {"Auth", "Mid", "Inner"}, // Inner leaked from the nested /attributes-equivalent group
		"/c": {"Auth", "Mid", "Inner"}, // happens to be right, by coincidence
		"/d": {"Auth", "Mid", "Inner"}, // Inner leaked forward too
	}
	for path, wantMW := range leaked {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		root.ServeHTTP(w, req)
		got := w.Header().Values("X-MW")
		if len(got) != len(wantMW) {
			t.Fatalf("%s: X-MW=%v, want %v (documented leak shape changed)", path, got, wantMW)
		}
		for i := range wantMW {
			if got[i] != wantMW[i] {
				t.Fatalf("%s: X-MW=%v, want %v (documented leak shape changed)", path, got, wantMW)
			}
		}
	}
}

// Same shape as acme's app/route.go, but using GroupWithMiddleware to scope
// each level's middleware explicitly instead of the leaky WithMiddleware(x).Group(fn)
// chain. Each route should get exactly the middleware for its own nesting depth.
func TestNestedGroupWithMiddlewareScoping(t *testing.T) {
	root := routing.New()

	root.GroupWithMiddleware([]routing.Middleware{marker("Auth")}, func(a *routing.Router) {
		a.GET("/a", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "a") })

		a.GroupWithMiddleware([]routing.Middleware{marker("Mid")}, func(b *routing.Router) {
			b.GET("/b", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "b") })

			b.GroupWithMiddleware([]routing.Middleware{marker("Inner")}, func(c *routing.Router) {
				c.GET("/c", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "c") })
			})

			b.GET("/d", func(ctx *routing.Context) { ctx.Text(http.StatusOK, "d") })
		})
	})

	want := map[string][]string{
		"/a": {"Auth"},
		"/b": {"Auth", "Mid"},
		"/c": {"Auth", "Mid", "Inner"},
		"/d": {"Auth", "Mid"},
	}
	for path, wantMW := range want {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		root.ServeHTTP(w, req)
		got := w.Header().Values("X-MW")
		if w.Code != http.StatusOK {
			t.Errorf("%s: status=%d, want 200", path, w.Code)
		}
		if len(got) != len(wantMW) {
			t.Fatalf("%s: X-MW=%v, want %v", path, got, wantMW)
		}
		for i := range wantMW {
			if got[i] != wantMW[i] {
				t.Errorf("%s: X-MW=%v, want %v", path, got, wantMW)
				break
			}
		}
	}
}
