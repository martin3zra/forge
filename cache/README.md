# cache

A small key/value cache abstraction with a Postgres-backed implementation.

## Interface

```go
type Cache interface {
    Get(ctx, key) ([]byte, bool, error)
    Set(ctx, key, value []byte) error
    Delete(ctx, key) error
}
```

## Postgres store

`NewPgCache(q database.Querier)` stores entries in a `preview_cache (key, payload, updated_at)` table, upserting on `key`. Accepts any `database.Querier`, so it works with a `*sql.DB` or a transaction.

```go
c := cache.NewPgCache(db)
```

## Remember

`Remember` is the read-through helper: return the cached value if present, otherwise run `fn`, cache its JSON, and return it. Generic over the value type; a cached entry that fails to unmarshal is evicted and recomputed.

```go
user, err := cache.Remember(ctx, c, "user:42", func() (User, error) {
    return loadUser(42)
})
```
