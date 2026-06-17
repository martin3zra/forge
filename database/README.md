# database

Thin helpers over `database/sql`. Postgres is assumed (`$1` placeholders, `lib/pq`).

## Connection in context

forge passes the DB through `context.Context` keyed by an empty struct:

```go
ctx = context.WithValue(ctx, database.ConnectionKey{}, db)
```

`auth.NewAuth(ctx)` and other packages read it back from here.

## Querier

`Querier` is the read/write surface shared by `*sql.DB` and `*sql.Tx`, so code can accept either:

```go
type Querier interface {
    QueryRowContext(ctx, query, args...) *sql.Row
    QueryContext(ctx, query, args...) (*sql.Rows, error)
    ExecContext(ctx, query, args...) (sql.Result, error)
}
```

## Transactions

`WithTransaction` runs `fn` in a transaction, committing on `nil` and rolling back otherwise. A rollback error is joined with the original via `errors.Join`.

```go
err := database.WithTransaction(db, func(tx *sql.Tx) error {
    // ... use tx ...
    return nil
})
```

## Bulk insert

`PrepareBulkInsert(columns, values)` builds the `($1,$2),($3,$4),…` placeholder tail for a multi-row insert:

```go
placeholders := database.PrepareBulkInsert(2, len(rows)) // "($1,$2),($3,$4)"
stmt := "INSERT INTO t (a, b) VALUES " + placeholders
```
