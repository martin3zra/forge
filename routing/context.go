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
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/session"
	"github.com/romsar/gonertia/v2"
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

	isProduction := os.Getenv("APP_ENV")
	if slices.Contains([]string{"prod", "production"}, isProduction) {
		ctx.Render("Error/Index", map[string]any{
			"status": defaultStatus,
		})
		return
	}

	title, ok := titleHttpCode[defaultStatus]
	if !ok {
		title = "Something went wrong."
	}

	// display errors when on dev mode. otherwise logged this error.
	data := make(map[string]any)
	data["title"] = title
	data["message"] = foundation.ResolveError(err)
	data["status"] = defaultStatus

	tmpl, err := template.ParseFS(ctx.resources, "resources/views/error/500.html")
	if err != nil {
		log.Println(err.Error())
		http.Error(ctx.Response, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmplErr := tmpl.Execute(ctx.Response, data)
	if tmplErr != nil {
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
