# auth

Authentication that never hardcodes your user schema. The framework owns the contract (`foundation.Authenticatable`); your app owns the table and registers lookups.

## Wiring (once, at boot)

```go
auth.SetCredentialResolver(func(db *playsql.DB, column string, value any) (foundation.Authenticatable, error) {
    // db.Model(&yourRow{}).WhereEq(column, value).First(ctx, &row), map to your User type
})
auth.SetPasswordResolver(func(db *playsql.DB, userID int) (string, error) {
    // current password hash for userID
})
auth.SetUserDecoder(func(ctx context.Context) foundation.Authenticatable {
    // rebuild the identity stored in the request context / session
})
```

Calling an `Auth` method before its resolver is registered returns an error rather than panicking.

## Per-request

`NewAuth(ctx)` pulls `*playsql.DB` from the context under `database.PlaysqlKey{}` — it **panics if the connection is absent**, so ensure your DB middleware runs first.

```go
a := auth.NewAuth(ctx)
user, err := a.Authenticate("user@example.com", "secret") // looks up by "email", checks bcrypt
user, err := a.LoginUsingId(42)                            // looks up by "id"
```

## Reading the current identity

```go
user := auth.User(ctx) // via the registered UserDecoder; nil if none
id   := auth.ID(ctx)   // 0 if unauthenticated
```

Password checks use `foundation.Hash` (bcrypt) internally.
