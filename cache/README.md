# cache

A small key/value cache abstraction, backed by playsql so it works unchanged
across SQLite, Postgres, and MySQL.

## Interface

```go
type Cache interface {
    Get(ctx, key) ([]byte, bool, error)
    Set(ctx, key, value []byte) error
    Delete(ctx, key) error
}
```

## SQL store

`NewSQLCache(db *playsql.DB)` stores entries in a `preview_cache (key, payload, updated_at)` table, upserting on `key`.

```go
c := cache.NewSQLCache(db)
```

## Remember

`Remember` is the read-through helper: return the cached value if present, otherwise run `fn`, cache its JSON, and return it. Generic over the value type; a cached entry that fails to unmarshal is evicted and recomputed.

```go
user, err := cache.Remember(ctx, c, "user:42", func() (User, error) {
    return loadUser(42)
})
```
