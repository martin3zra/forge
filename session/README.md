# session

Server-side sessions with a pluggable store, CSRF protection, and flash/error bags. Integrates with Inertia (`gonertia`).

## Setup

```go
store := session.NewDatabaseStore(db) // requires a "sessions" table
mgr := session.NewSessionManager(
    store,
    1*time.Hour,    // gc interval
    2*time.Hour,    // idle expiration
    24*time.Hour,   // absolute expiration
    "forge_session",// cookie name
    "",             // domain
    true,           // secure
    true,           // httpOnly
)

handler = mgr.Handle(next) // middleware: starts session, enforces CSRF, persists on response
```

`NewSessionManager` spawns a background goroutine that GCs expired sessions on the gc interval.

## Middleware behavior (`Handle`)

- Loads the session from the cookie (or creates a fresh one) and attaches it to the request context under `SessionContextKey{}`.
- On `POST/PUT/PATCH/DELETE`, verifies the CSRF token from the `csrf_token` form field or `X-CSRF-Token` header — mismatch → `403`.
- Writes the session cookie and persists the session after the handler runs.

## Working with a session

```go
s := session.GetSession(r) // panics if no session in context (i.e. middleware not mounted)
s.Put("key", value)
v := s.Get("key")
s.Flash("status", "saved")      // one-request flash
s.Errors("field", "message")    // error bag
s.FormErrors(errorBag)          // merge a foundation.ErrorBag
s.ClearErrors()
```

## Lifecycle helpers

- `ReGenerate(r, user, attrs)` — rotate the session id + CSRF token on login, store the user.
- `Invalidate(r)` — rotate and clear the user on logout.
- `Migrate(session)` — low-level id/token rotation.
- `AgeFlash(session)` — clear flash/errors and persist.

## Custom store

Implement the unexported `SessionStore` interface (`read`, `write`, `destroy`, `gc`) within the `session` package, modeled on `DatabaseStore` in `database.go`.
