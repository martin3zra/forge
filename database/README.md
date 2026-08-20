# database

Thin helpers over `database/sql`. Query building for SQLite/Postgres/MySQL goes
through [`playsql`](https://github.com/martin3zra/playsql); code in this
package (and `PrepareBulkInsert` below) still assumes Postgres `$1` placeholders.

## Connection in context

forge passes the DB through `context.Context` keyed by an empty struct:

```go
ctx = context.WithValue(ctx, database.ConnectionKey{}, db)          // *sql.DB
ctx = context.WithValue(ctx, database.PlaysqlKey{}, play)           // *playsql.DB
ctx = context.WithValue(ctx, database.DialectKey{}, "postgres")     // "postgres" | "mysql" | "sqlite"
```

`PlaysqlKey` is what `auth.NewAuth(ctx)` and other playsql-backed code read
back — prefer it for anything querying through playsql's `Builder`.
`ConnectionKey` remains for code that still needs a raw `*sql.DB`/`*sql.Tx`.
`DialectKey` is for code that builds SQL by hand instead of going through
playsql's `Builder` (e.g. `validator`'s dynamic exists/unique rules) — it picks
the right placeholder style via `playsql/grammar.For(dialect)`.

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
