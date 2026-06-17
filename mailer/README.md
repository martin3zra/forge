# mailer

Send templated HTML email over SMTP or an HTTP API, with attachments.

## Config

```go
cfg := mailer.Config{
    Driver:      mailer.SMTP, // or mailer.API
    Host:        "localhost",
    Port:        "1025",
    FromAddress: "no-reply@example.com",
    FromName:    "Forge",
    Username:    "user",
    Password:    "pass", // for API driver, this is the bearer token
}
m := mailer.New(cfg, templatesFS) // templatesFS is an embed.FS of HTML templates
```

- `SMTP` — builds a MIME message (multipart/mixed when attachments are present) and sends via `net/smtp`.
- `API` — POSTs a JSON payload to `cfg.Host` with `Authorization: Bearer <Password>` (base64 attachments).

## Mailable

Implement `Mailable` to describe a message:

```go
type Welcome struct{ Name string }

func (w Welcome) Subject() string                  { return "Welcome" }
func (w Welcome) To() []mailer.Individual           { return nil } // extra recipients beyond .To()
func (w Welcome) Content() string                   { return "emails/welcome.html" } // template path in the FS
func (w Welcome) Data() map[string]any              { return map[string]any{"Name": w.Name} }
func (w Welcome) Attachments() []mailer.Attachment  { return nil }
```

The `Content()` template is parsed from the embedded FS and rendered with `Data()`.

## Send

```go
m.To("user@example.com", "User").Send(Welcome{Name: "Ada"})
```

> Note: the SMTP path calls `log.Fatal` on send failure. `mailer_test.go` expects a local SMTP server (e.g. MailHog on `:1025`).
