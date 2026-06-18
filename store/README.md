# store

Loads SQL files from a filesystem into a named query map, so queries live in `.sql` files instead of string literals.

## Usage

Given files like `queries/users_by_email.sql`, `queries/active_users.sql`:

```go
//go:embed queries/*.sql
var queriesFS embed.FS

qs, err := store.NewQueryStore(queriesFS, "queries/")

row := db.QueryRowContext(ctx, qs.Q("users_by_email"), email)
```

`NewQueryStore(fsys, dir)` globs `<dir>*.sql`, keying each query by its filename without the `.sql` extension. `qs.Q(name)` returns the query text (empty string if absent).
