# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`forge` (`github.com/martin3zra/forge`) is a Laravel-inspired web framework for Go, built on the standard `net/http`. It is a **library**, not an application — there is no `main` package and no binary. Each top-level directory is an independent feature package (`auth`, `routing`, `session`, `validator`, `foundation`, `i18n`, `database`, `cache`, `mailer`, `inertia`, `console`, `support`, `store`). Consuming applications wire these together.

## Commands

```bash
go build ./...                          # compile all packages
go vet ./...                            # static checks
go test ./...                           # run all tests
go test ./validator/ -run TestName      # single test (packages with tests: validator, routing, mailer)
go test -v ./routing/                   # verbose, one package
go build -tags prod ./...               # build with embedded production assets (see Build tags)
```

Go 1.23.6. No Makefile, no lint config beyond `go vet` — use standard tooling.

## Architecture

### Framework-owns-contracts, app-owns-schema

The framework never hardcodes the application's data model. Two mechanisms enforce this:

1. **Interfaces in `foundation`** — `foundation/contracts.go` defines `Authenticatable`, `MustVerifyPassword`, etc. The concrete user struct and its persistence live in the consuming app, not here.
2. **Registered resolvers** — `auth` holds package-level function vars (`credentialResolver`, `passwordResolver`, `userDecoder`) set via `auth.SetCredentialResolver`/`SetPasswordResolver`/`SetUserDecoder`. The app registers these at boot so `auth` can look up users without knowing the schema. Calling an auth method before its resolver is registered returns an error.

When extending the framework, follow this pattern: define an interface or resolver here, push the concrete implementation to the app.

### Context as the wiring bus

Cross-cutting dependencies travel through `context.Context`, keyed by empty structs:
- `database.ConnectionKey{}` → `*sql.DB`. `auth.NewAuth(ctx)` pulls the DB out of context and **panics** if absent.
- `auth.User(ctx)` / `auth.ID(ctx)` decode the current identity via the registered `userDecoder`.

### Request lifecycle (Laravel parallel)

- **`routing`** — wraps `net/http`. `routing.Context` carries `Params`, `BindQuery`, `BindJSON`. Middleware is functional (`func(HandlerFunc) HandlerFunc`), composed via `WithMiddleware`, excluded per-route with `WithoutMiddleware`, and grouped with `GroupPrefix` (Laravel-style nestable prefixes). See `routing/README.md` for the full API.
- **`support.FormRequest`** — mirrors Laravel form requests: `Authorize()`, `Rules()`, `Messages()`, `PrepareForValidation()`, plus `SetContext`/`SetPathParams`. Embed `support.FormRequest` and override methods. It composes a `validator.Validator` and pulls the authed user from context.
- **`validator`** — reflection-driven. Rules are a `map[string]any` keyed by **`json` struct tag**, with pipe-delimited rule strings (`"required|max:255"`), exactly like Laravel. It recurses into nested structs and slices (`field[*]` / `field.N` keys), supports `bail` and `sometimes`, custom messages via a `Messages()` method on the validated struct, and DB-backed rules (`unique`, `exists`). Localized messages live in `validator/locale/{en,es}.go`; language defaults to `es`. Adding a rule means registering it in the rule-set slices (`defaultRules`, `databaseRules`, etc. in `validator/types.go`) **and** implementing the check.

### Build tags for assets

Static-asset embedding is selected at compile time:
- `foundation/fs-dev.go` (`//go:build !prod`) serves from disk via `os.DirFS` for live reloading.
- `foundation/fs-prod.go` (`//go:build prod`) serves the embedded `embed.FS`.

Build the app with `-tags prod` for a self-contained binary. `inertia/vite.go` ties asset resolution to the Inertia/Vite frontend.

### Sessions

`session.SessionManager` (`session/manager.go`) runs a background GC goroutine and enforces both idle and absolute expiration. The store is pluggable via the `SessionStore` interface; `session/database.go` is the SQL-backed implementation. Integrates with `gonertia` (Inertia.js server adapter).

## Conventions

- **Errors as HTTP status** — `foundation.ErrorFormatter` (`Status() int` + `Error() string`) lets domain errors carry their HTTP code; `Unauthorized`, `UnprocessableEntity`, `BadRequest` are ready-made.
- **`foundation.ErrorBag`** (`map[string][]string`) is the shared validation/error-collection shape across packages.
- Reusable generic/reflection helpers live in `foundation/helpers.go` (`AsMap`, `MapSlice`, `ResolveError`, `FormatAmount`, `AsUUID`). Check here before writing new utility code.
- Postgres is the assumed DB driver (`github.com/lib/pq`).
