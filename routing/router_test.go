package routing_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/martin3zra/forge/routing"
)

// -----------------
// MatchRoute Tests
// -----------------

func TestMatchRoute(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		match   bool
		params  map[string]string
	}{
		{
			name:    "Static route match",
			pattern: "/users",
			path:    "/users",
			match:   true,
			params:  map[string]string{},
		},
		{
			name:    "Static route match with trailing slash in path",
			pattern: "/users",
			path:    "/users/",
			match:   true,
			params:  map[string]string{},
		},
		{
			name:    "Dynamic parameter match",
			pattern: "/users/:id",
			path:    "/users/123",
			match:   true,
			params:  map[string]string{"id": "123"},
		},
		{
			name:    "Dynamic parameter missing segment",
			pattern: "/users/:id",
			path:    "/users/",
			match:   false,
			params:  nil,
		},
		{
			name:    "Multiple dynamic parameters",
			pattern: "/users/:id/messages/:msgId",
			path:    "/users/45/messages/789",
			match:   true,
			params:  map[string]string{"id": "45", "msgId": "789"},
		},
		{
			name:    "Partial dynamic route match fails",
			pattern: "/users/:id/messages/:msgId",
			path:    "/users/45/messages",
			match:   false,
			params:  nil,
		},
		{
			name:    "Static route mismatch",
			pattern: "/about",
			path:    "/contact",
			match:   false,
			params:  nil,
		},
		{
			name:    "Root route match",
			pattern: "/",
			path:    "/",
			match:   true,
			params:  map[string]string{},
		},
		{
			name:    "Empty pattern and path match",
			pattern: "",
			path:    "",
			match:   true,
			params:  map[string]string{},
		},
		{
			name:    "Extra path segments should fail",
			pattern: "/users/:id",
			path:    "/users/123/extra",
			match:   false,
			params:  nil,
		},
		{
			name:    "Trailing slash in pattern",
			pattern: "/users/:id/",
			path:    "/users/123",
			match:   true,
			params:  map[string]string{"id": "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotParams := routing.MatchRoute(tt.pattern, tt.path)
			if gotMatch != tt.match {
				t.Errorf("MatchRoute(%q, %q) match = %v; want %v", tt.pattern, tt.path, gotMatch, tt.match)
			}
			if !reflect.DeepEqual(gotParams, tt.params) {
				t.Errorf("MatchRoute(%q, %q) params = %v; want %v", tt.pattern, tt.path, gotParams, tt.params)
			}
		})
	}
}

// ---------------------
// Middleware Chain Tests
// ---------------------

// Dummy middleware m1 writes "m1-" to the response.
var m1 routing.Middleware = func(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		ctx.Response.Write([]byte("m1-"))
		next(ctx)
	}
}

// Dummy middleware m2 writes "m2-" to the response.
var m2 routing.Middleware = func(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		ctx.Response.Write([]byte("m2-"))
		next(ctx)
	}
}

// finalHandler writes "handler" to the response.
func finalHandler(ctx *routing.Context) {
	ctx.Response.Write([]byte("handler"))
}

// TestMiddlewareChain verifies that middleware are applied in the proper order.
func TestMiddlewareChain(t *testing.T) {
	r := routing.New()
	r.WithMiddleware(m1, m2)
	r.GET("/test", finalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	bodyBytes, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}
	body := string(bodyBytes)
	expected := "m1-m2-handler"
	if body != expected {
		t.Errorf("Expected response %q, got %q", expected, body)
	}
}

// TestWithoutNextMiddleware verifies that a route can exclude a middleware.
func TestWithoutNextMiddleware(t *testing.T) {
	r := routing.New()
	r.WithMiddleware(m1, m2)
	r.GET("/test2", finalHandler).WithoutMiddleware(m1)

	req := httptest.NewRequest("GET", "/test2", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	bodyBytes, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}
	body := string(bodyBytes)
	expected := "m2-handler"
	if body != expected {
		t.Errorf("Expected response %q, got %q", expected, body)
	}
}

var testMiddleware routing.Middleware = func(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		ctx.Response.Header().Set("X-Test-Middleware", "yes")
		next(ctx)
	}
}

var skipMiddleware routing.Middleware = func(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		ctx.Response.Header().Set("X-Skip-Middleware", "yes")
		next(ctx)
	}
}

// TestGroupMiddlewareAndSkipNext verifies that after removing a middleware from the chain,
// only the remaining ones are applied in a group.
func TestGroupMiddlewareAndSkipNext(t *testing.T) {
	r := routing.New()
	r.WithMiddleware(testMiddleware, skipMiddleware)
	r2 := r.WithoutNextMiddleware(skipMiddleware)

	// Use Prefix to create an "/api" group.
	r2.GroupPrefix("/api", func(api *routing.Router) {
		api.GET("/grouped", func(ctx *routing.Context) {
			ctx.Text(http.StatusOK, "grouped")
		})
	})

	req := httptest.NewRequest("GET", "/api/grouped", nil)
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)

	if w.Header().Get("X-Test-Middleware") != "yes" {
		t.Errorf("expected X-Test-Middleware header to be set to 'yes'")
	}
	if w.Header().Get("X-Skip-Middleware") != "" {
		t.Errorf("expected X-Skip-Middleware header to be skipped")
	}
}

// ---------------------
// WithRequest Tests
// ---------------------

func TestWithRequest_Success(t *testing.T) {
	type RequestPayload struct {
		Message string `json:"message"`
	}

	r := routing.New()
	r.POST("/echo", routing.WithRequest(func(ctx *routing.Context, payload *RequestPayload) {
		ctx.JSON(http.StatusOK, map[string]string{"echo": payload.Message})
	}))

	validJSON := `{"message": "Hello, world!"}`
	req := httptest.NewRequest("POST", "/echo", strings.NewReader(validJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("error unmarshalling response: %v", err)
	}
	if res["echo"] != "Hello, world!" {
		t.Errorf("expected echo %q, got %q", "Hello, world!", res["echo"])
	}
}

func TestWithRequest_InvalidJSON(t *testing.T) {
	type RequestPayload struct {
		Message string `json:"message"`
	}

	r := routing.New()
	r.POST("/echo", routing.WithRequest(func(ctx *routing.Context, payload *RequestPayload) {
		ctx.JSON(http.StatusOK, map[string]string{"echo": payload.Message})
	}))

	invalidJSON := `{"message": "Hello, world!`
	req := httptest.NewRequest("POST", "/echo", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "invalid JSON") {
		t.Errorf("expected error message containing 'invalid JSON', got %q", rr.Body.String())
	}
}

// ---------------------
// New Group Enhancement Tests
// ---------------------

// containsRoute returns true if the target route string is present.
func containsRoute(routes []string, target string) bool {
	for _, r := range routes {
		if r == target {
			return true
		}
	}
	return false
}

// TestPrefixGroupRoutes verifies that routes created from a prefixed group are registered correctly.
func TestPrefixGroupRoutes(t *testing.T) {
	r := routing.New()
	r.GroupPrefix("/admin", func(rg *routing.Router) {
		// Use "users" without a leading slash so the full route becomes "/admin/users"
		rg.GET("/users", func(ctx *routing.Context) {
			ctx.Response.Write([]byte("admin users"))
		})
	})

	routes := r.Routes()
	if !containsRoute(routes, "GET /admin/users") {
		t.Errorf("expected route GET /admin/users, got %v", routes)
	}

	req := httptest.NewRequest("GET", "/admin/users", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Result().Body)
	if string(body) != "admin users" {
		t.Errorf("expected response 'admin users', got %q", body)
	}
}

// TestPrefixGroupMiddlewareExclusion verifies that middleware exclusions work within a prefixed group.
func TestPrefixGroupMiddlewareExclusion(t *testing.T) {
	r := routing.New()
	r.WithMiddleware(m1, m2)
	r.GroupPrefix("/blog", func(rg *routing.Router) {
		// Exclude m1 from this route.
		rg.GET("/post", finalHandler).WithoutMiddleware(m1)
	})

	routes := r.Routes()
	if !containsRoute(routes, "GET /blog/post") {
		t.Errorf("expected route GET /blog/post, got %v", routes)
	}

	req := httptest.NewRequest("GET", "/blog/post", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Result().Body)
	expected := "m2-handler" // Only m2 should wrap the handler
	if string(body) != expected {
		t.Errorf("expected response %q, got %q", expected, body)
	}
}

// TestEmptyPrefixGroup verifies that an empty prefix does not alter route paths.
func TestEmptyPrefixGroup(t *testing.T) {
	r := routing.New()
	r.GroupPrefix("", func(rg *routing.Router) {
		rg.GET("noprefix", func(ctx *routing.Context) {
			ctx.Response.Write([]byte("noprefix"))
		})
	})
	routes := r.Routes()
	if !containsRoute(routes, "GET /noprefix") {
		t.Errorf("expected route GET /noprefix, got %v", routes)
	}

	req := httptest.NewRequest("GET", "/noprefix", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Result().Body)
	if string(body) != "noprefix" {
		t.Errorf("expected response 'noprefix', got %q", body)
	}
}

// TestNestedPrefixGroupRoutes verifies that nested prefix groups correctly concatenate paths.
func TestNestedPrefixGroupRoutes(t *testing.T) {
	r := routing.New()
	r.GroupPrefix("/api", func(api *routing.Router) {
		api.GroupPrefix("/v1", func(v1 *routing.Router) {
			v1.GET("/resource", func(ctx *routing.Context) {
				ctx.Response.Write([]byte("api v1 resource"))
			})
		})
		api.GET("/status", func(ctx *routing.Context) {
			ctx.Response.Write([]byte("api status"))
		})
	})

	routes := r.Routes()
	if !containsRoute(routes, "GET /api/v1/resource") {
		t.Errorf("expected route GET /api/v1/resource, got %v", routes)
	}
	if !containsRoute(routes, "GET /api/status") {
		t.Errorf("expected route GET /api/status, got %v", routes)
	}

	req := httptest.NewRequest("GET", "/api/v1/resource", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	body, _ := io.ReadAll(rr.Result().Body)
	if string(body) != "api v1 resource" {
		t.Errorf("expected response 'api v1 resource', got %q", body)
	}

	req2 := httptest.NewRequest("GET", "/api/status", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	body2, _ := io.ReadAll(rr2.Result().Body)
	if string(body2) != "api status" {
		t.Errorf("expected response 'api status', got %q", body2)
	}
}
