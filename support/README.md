# support

Request parsing and Laravel-style form requests. Bridges `routing`, `validator`, and `auth`.

## ParseRequest

Decodes a request body into a struct pointer, then — if the struct implements `FormRequestContract` — runs authorization and validation.

```go
err := support.ParseRequest(r, &payload, pathParams)
```

- JSON bodies are decoded directly; a malformed body returns `foundation.HTTPError{400, "invalid JSON request body"}`.
- `multipart/form-data` is parsed and hydrated via `HydrateFromForm` (reflection over `json` tags; supports `string`, `int`, `bool`, and `multipart.File` fields).
- A **plain struct** (not a `FormRequestContract`) needs only a successful decode — no validation is run.

## FormRequest

Embed `support.FormRequest` and override what you need; it satisfies `FormRequestContract`:

```go
type StorePost struct {
    support.FormRequest
    Title string `json:"title"`
}

func (r StorePost) Rules() map[string]any {
    return map[string]any{"title": "required|max:255"}
}

func (r StorePost) Authorize() bool {
    return r.User() != nil // user is pulled from context on SetContext
}

func (r StorePost) Messages() map[string]string { return nil }
```

Lifecycle hooks mirror Laravel: `Authorize()`, `Rules()`, `Messages()`, `PrepareForValidation()`, `PassedValidation(validated map[string]any)`. On `SetContext`, the authenticated user (`auth.User`) and a validator instance are attached; `SetPathParams` exposes route params via `Param(key)`.

When validation fails, errors are flashed into the session (`FormErrors`) and surfaced through `Errors()`.

When validation passes, `Validated()` returns the subset of the request that had a rule attached — nested back into the original dot/wildcard shape, pointers dereferenced, absent fields excluded (see `validator.Validator.Validated`). The same map is handed to `PassedValidation`, so overriding it can mutate the map in place (add computed fields, strip ones you don't want a caller to see) before the controller reads it back via `r.Validated()`.
