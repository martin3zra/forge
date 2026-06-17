# console

Interactive stdin prompts for CLI commands and scaffolding tools.

```go
name := console.Ask("Project name?")
env  := console.AskWithDefault("Environment?", "local")

if console.AskYesOrNo("Run migrations?", true) {
    // ...
}

email := console.AskWithValidator("Email?", func(s string) error {
    if !strings.Contains(s, "@") {
        return errors.New("not an email")
    }
    return nil
}) // re-prompts until the validator passes

console.Info("done")
```

| Function | Behavior |
|---|---|
| `Ask(prompt)` | Read a trimmed line. |
| `AskWithDefault(prompt, def)` | Empty input → `def`. |
| `AskYesOrNo(prompt, defaultYes)` | Loops until y/yes/n/no; empty → default. |
| `AskWithValidator(prompt, validate)` | Loops until `validate` returns nil. |
| `Info(...)` | `fmt.Println` wrapper. |
