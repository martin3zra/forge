# foundation

Shared contracts, value types, and helpers used across every forge package. Depends on nothing else in the framework, so it is safe to import anywhere.

## Contracts

- `Authenticatable` — the identity contract auth works against (`GetAuthIdentifier`, `GetAuthPassword`, `GetRole`, …). The concrete user type lives in your app.
- `MustVerifyPassword` — identities that can be forced to rotate their password.
- `BodyContract` — anything that exposes `Validate() ErrorBag`.
- `ErrorFormatter` — an error carrying an HTTP status (`Status() int` + `Error() string`).

## Error types

Ready-made `ErrorFormatter` implementations: `Unauthorized` (403), `BadRequest` (400), `UnprocessableEntity` (422). For a status with a custom message use `HTTPError`:

```go
return foundation.HTTPError{StatusCode: http.StatusBadRequest, Message: "invalid JSON request body"}
```

`ErrorBag` (`map[string][]string`) is the shared validation/error shape.

## Helpers

| Function | Purpose |
|---|---|
| `GetIpAddress(r)` | Client IP, honoring `X-Forwarded-For`. |
| `AsMap(obj)` | Struct → `map[string]any` of exported fields (reflection). |
| `MapSlice(s, f)` | Generic slice map. |
| `ToJSON` / `AsJSON` | Marshal to spaced string / raw bytes. |
| `ResolveError(err)` | Unwrap to the root error. |
| `FormatAmount(n, sym?)` | Thousands-formatted money string. |
| `AsUUID(s)` | Parse string → `uuid.UUID` (empty → `uuid.Nil`). |
| `GeneratePrefixedNumber(prefix, len, val)` | Zero-padded prefixed code. |

## Hashing

```go
h := foundation.NewHashable()
hash := h.Make("secret")          // bcrypt, cost 14
ok := h.Check("secret", hash)     // bcrypt compare
sig := h.Sha1HMAC(raw, secret)    // webhook signatures
h.HMACEquals(raw, sig, secret)    // constant-time compare
```

## Flash cookies

`SetFlash(w, name, value)` / `GetFlash(w, r, name)` — base64 cookie, read-once (deleted on read).

## Money to words

`MoneyToText(cents, currency)` renders an amount in **Spanish** words (e.g. for invoices/checks).

## Assets

`GetBuildAssets(assets, dir)` returns an `fs.FS`. Build-tag selected: `os.DirFS` in dev, the embedded `embed.FS` under `-tags prod`.

## Subpackage `str`

`str.Uuid()` and `str.GenerateRandom(byteLen?)` (returns a `base64:`-prefixed secure random string, e.g. app keys).
