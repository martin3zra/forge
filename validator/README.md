# validator

Reflection-driven, Laravel-style struct validation. Rules are a `map[string]any` keyed by **`json` struct tag**, with pipe-delimited rule strings.

## Basic usage

```go
type CreateUser struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

v := new(validator.Validator)
ok := v.Validate(ctx, payload, map[string]any{
    "name":  "required|min:2|max:255",
    "email": "required|email|unique:users,email",
    "age":   "gte:18",
})
if !ok {
    errs := v.Errors() // validator.Errors == map[string][]string
}
```

## Rules

Defined in `types.go` (`defaultRules`). Highlights:

- Presence/comparison: `required`, `required_if:<field>,<value>`, `min`, `max`, `gt`/`gte`/`lt`/`lte`, `between:a,b`, `different`.
- Strings: `email`, `in:a,b,c`, `uppercase`, `lowercase`, `format`, `digits`, `digits_between`, `max_digits`, `min_digits`.
- Dates (`time.Time`): `date`, `after:today|yesterday`, `before:…`, `before_or_equal:…`.
- Database (`databaseRules`): `exists:table,column`, `unique:table,column`, `current_password`.

Two modifiers:
- `sometimes` — only validate when the field is non-zero.
- `bail` — stop at the first failure for that field.

## Nested structs & slices

The validator recurses into nested structs and slices automatically. Address nested fields with dotted keys, and slice elements with `[*]`:

```go
map[string]any{
    "address.city":   "required",
    "items[*].price": "gte:0",
}
```

Embedded (anonymous) structs inherit the parent prefix.

## Custom messages

Add a `Messages() map[string]string` method on the validated struct, keyed `"<field>.<rule>"`:

```go
func (CreateUser) Messages() map[string]string {
    return map[string]string{"email.unique": "That email is taken."}
}
```

## Conditional rules

```go
var r validator.Rule
rules := map[string]any{
    "card": []any{r.When(isPaid, "required|digits:16", "")},
}
```

## Localization

Messages come from `validator/locale/{en,es}.go`. Language defaults to **`es`**; set `Validator.language` to override.

## Extending

Adding a rule means registering its name in the appropriate slice in `types.go` (`defaultRules`, `dateRules`, `arrayRules`, `databaseRules`) **and** implementing the check in `validates-attributes.go` / the relevant `*-validation.go` file.
