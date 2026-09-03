package routing

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/playsql"
	"github.com/romsar/gonertia/v3"
)

// Context wraps request and response and provides helper methods.
type Context struct {
	Response  http.ResponseWriter
	Request   *http.Request
	Params    map[string]string
	Inertia   *gonertia.Inertia
	resources embed.FS
}

// User fetch user from Request Context
func (c *Context) User() foundation.Authenticatable {
	return auth.User(c.Request.Context())
}

// DB returns the *playsql.DB bound to the current request via
// database.PlaysqlKey — the same key auth.NewAuth already reads, set once by
// the application at server setup (see forge/database). It panics if none was
// bound, matching NewAuth's existing convention for this same context value:
// a missing DB is a startup/wiring bug, not a per-request condition to
// recover from.
func (c *Context) DB() *playsql.DB {
	db, ok := c.Request.Context().Value(database.PlaysqlKey{}).(*playsql.DB)
	if !ok || db == nil {
		panic("routing: no *playsql.DB bound to request context (see forge/database.PlaysqlKey)")
	}
	return db
}

// Text writes plain text response with status code.
func (c *Context) Text(status int, text string) {
	c.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(status)
	_, _ = c.Response.Write([]byte(text))
}

// JSON sends a JSON response with the given status code.
func (ctx *Context) JSON(status int, data any) {
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(status)
	_ = json.NewEncoder(ctx.Response).Encode(data)
}

// Inertia sends the component and props to be use for inertiajs protocol
func (ctx *Context) Render(component string, props map[string]any) {
	abilities := ctx.Request.Context().Value(PermissionKey{})
	if abilities != nil {
		props["abilities"] = abilities.(map[string]bool)
	}

	if t, ok := props["translations"]; ok {
		trans, ok := t.(map[string]string)
		if ok {
			// Merge shared translations with page-specific ones
			props["translations"] = ctx.mergeTranslations(trans)
		}
	}
	err := ctx.Inertia.Render(ctx.Response, ctx.Request, component, props)
	if err != nil {
		log.Println("Error rending the inertia component: ", err)
		ctx.Error(err)
	}
}

// Error sends a parsed HTML to the client
func (ctx *Context) Error(err error, status ...int) {
	var titleHttpCode = map[int]string{
		500: "Internal Error.",
		403: "Forbidden.",
		401: "Unauthorized.",
		404: "Not Found.",
		412: "Precondition Failed.",
	}

	defaultStatus := 500
	if len(status) > 0 {
		defaultStatus = status[0]
	}

	if errors.Is(err, sql.ErrNoRows) {
		defaultStatus = http.StatusNotFound
	}

	if e, ok := err.(foundation.ErrorFormatter); ok {
		defaultStatus = e.Status()
	}

	isDev := !slices.Contains([]string{"prod", "production"}, os.Getenv("APP_ENV"))

	props := map[string]any{"status": defaultStatus}
	if isDev {
		// Outside production, hand the developer the underlying error on
		// the page as well (it is logged either way); production never
		// leaks it.
		log.Printf("ctx.Error [%d]: %v", defaultStatus, err)
		props["message"] = foundation.ResolveError(err)
	}

	// Render the Inertia Error/Index component — the same screen in dev and
	// prod, so its styling and navigation affordances show in both.
	//
	// gonertia's Render always writes a 200 itself with no way to override
	// it, so wrap the writer to force the real status however the response
	// is committed. Call Inertia.Render directly rather than ctx.Render,
	// which recurses back into Error on failure; the template below is the
	// genuine fallback (no Inertia configured, or the component failed
	// before writing anything).
	if ctx.Inertia != nil {
		w := &statusOverrideWriter{ResponseWriter: ctx.Response, status: defaultStatus}
		original := ctx.Response
		ctx.Response = w
		renderErr := ctx.Inertia.Render(ctx.Response, ctx.Request, "Error/Index", props)
		ctx.Response = original
		if renderErr == nil {
			return
		}
		log.Println("ctx.Error: Error/Index render failed:", renderErr)
		if w.wrote {
			return // response already committed — nothing safe left to do
		}
	}

	title, ok := titleHttpCode[defaultStatus]
	if !ok {
		title = "Something went wrong."
	}

	data := map[string]any{
		"title":   title,
		"message": foundation.ResolveError(err),
		"status":  defaultStatus,
	}

	tmpl, tmplParseErr := template.ParseFS(ctx.resources, "resources/views/error/500.html")
	if tmplParseErr != nil {
		log.Println(tmplParseErr.Error())
		http.Error(ctx.Response, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	ctx.Response.WriteHeader(defaultStatus)
	if tmplErr := tmpl.Execute(ctx.Response, data); tmplErr != nil {
		log.Println(tmplErr.Error())
		http.Error(ctx.Response, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (ctx *Context) WithContext(c context.Context) *Context {
	ctx.Request = ctx.Request.WithContext(c)
	return ctx
}

func (ctx *Context) Flash(name string, value any) {
	session.GetSession(ctx.Request).Flash(name, value)
}
func (ctx *Context) Errors(name string, value string) {
	session.GetSession(ctx.Request).Errors(name, value)
}

func (ctx *Context) Redirect(path string, status ...int) {
	if len(status) > 0 {
		http.Redirect(ctx.Response, ctx.Request, path, status[0])
		return
	}
	http.Redirect(ctx.Response, ctx.Request, path, http.StatusSeeOther)
}

func (ctx *Context) BackWithError(err error, status ...int) {
	// TODO: do some checking on this error.
	log.Println(err)
	ctx.Back(status...)
}

func (ctx *Context) Back(status ...int) {
	if len(status) > 0 {
		ctx.Inertia.Back(ctx.Response, ctx.Request, status[0])
		return
	}
	ctx.Inertia.Back(ctx.Response, ctx.Request, http.StatusSeeOther)
}

func (ctx *Context) BackWith(name string, value string, status ...int) {
	ctx.Errors(name, value)
	ctx.Back(status...)
}

func (ctx *Context) BackWithQuery(attributes map[string]any) {
	// Get the referer (previous page URL)
	referer := ctx.Request.Referer()
	if referer == "" {
		// Default fallback if referer is not present
		referer = "/"
	}

	// Parse the referer URL
	parsedURL, err := url.Parse(referer)
	if err != nil {
		http.Error(ctx.Response, "Invalid referer", http.StatusBadRequest)
		return
	}

	// Add or update query parameters
	q := parsedURL.Query()
	for k, v := range attributes {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	parsedURL.RawQuery = q.Encode()
	http.Redirect(ctx.Response, ctx.Request, parsedURL.String(), http.StatusFound)
}

// Query retrieves the query value for a key.
func (ctx *Context) Query(key string) string {
	return ctx.Request.URL.Query().Get(key)
}

func (ctx *Context) QueryHas(key string) bool {
	return strings.TrimSpace(ctx.Request.URL.Query().Get(key)) != ""
}

// QueryWithDefault retrieves the query value for a key or returns the fallback value.
func (ctx *Context) QueryWithDefault(key, fallback string) string {
	val := ctx.Request.URL.Query().Get(key)
	if val == "" {
		return fallback
	}
	return val
}

// QueryValues returns all query parameters.
func (ctx *Context) QueryValues() url.Values {
	return ctx.Request.URL.Query()
}

// Param returns a path parameter by key or empty string if not present.
func (ctx *Context) Param(key string) string {
	if ctx.Params == nil {
		return ""
	}

	// Check if the given key contains "id" and validate it as a UUID.
	if strings.Contains(strings.ToLower(key), "id") {
		var re = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)
		if re.MatchString(ctx.Params[key]) {
			reUUID := `^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`
			if regexp.MustCompile(reUUID).MatchString(ctx.Params[key]) {
				return ctx.Params[key]
			}
		}
	}

	return ctx.Params[key]
}

func (ctx *Context) Int(key string) int {

	if intValue, err := strconv.Atoi(ctx.Param(key)); err == nil {
		return intValue
	}
	return 0
}

func (ctx *Context) WantsJson() bool {
	acceptHeader := ctx.Request.Header.Get("Accept")
	return strings.Contains(acceptHeader, "application/json")
}

// mergeTranslations merges shared "translations" with page-specific ones.
func (ctx *Context) mergeTranslations(pageTranslations map[string]string) map[string]string {
	merged := map[string]string{}

	// ✅ Get existing props from context
	sharedProps := gonertia.PropsFromContext(ctx.Request.Context())

	// Get shared translations if available
	if shared, ok := sharedProps["translations"].(map[string]string); ok {
		for k, v := range shared {
			merged[k] = v
		}
	}

	// Merge page-specific
	for k, v := range pageTranslations {
		merged[k] = v
	}

	return merged
}

// statusOverrideWriter forces the eventual response status to a fixed
// value, however it's committed — an explicit WriteHeader call, or an
// implicit 200 on the first Write. Used by Error's production path since
// gonertia's Render has no way to take a status code directly.
type statusOverrideWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusOverrideWriter) WriteHeader(int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.ResponseWriter.WriteHeader(w.status)
}

func (w *statusOverrideWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(w.status)
	}
	return w.ResponseWriter.Write(b)
}
