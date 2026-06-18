# validator

Rule-driven, Laravel-style validation. Rules are a `map[string]any` keyed by dot-path, with pipe-delimited rule strings. The data source can be a **struct** (keyed by `json` tag) or a **`map[string]any`** (e.g. a decoded JSON body) — the same rule set works for both.

## Basic usage (struct)

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

## Validating a map

Pass a `map[string]any` instead of a struct — handy for dynamic or partial payloads where you don't have a type:

```go
var data map[string]any
json.NewDecoder(r.Body).Decode(&data)

v := new(validator.Validator)
v.Validate(ctx, data, map[string]any{
    "name":            "required|min:2",
    "address.line":    "required",
    "contacts.*.name": "required|min:3",
})
```

Notes for map input:

- **Absent keys fail `required`** (Laravel semantics). A struct field, by contrast, is always present as its zero value.
- JSON numbers decode to `float64`; integral values are coerced to `int` so integer rules (`digits`, `min_digits`, …) apply.
- A non-`required` rule on an absent key is skipped; `sometimes` likewise skips absent keys.
- Custom messages via a `Messages()` method are only available for **struct** input (a map has no method to call).

## Rules

Defined in `types.go` (`defaultRules`). Highlights:

- Presence/comparison: `required`, `required_if:<field>,<value>`, `min`, `max`, `gt`/`gte`/`lt`/`lte`, `between:a,b`, `different`.
- Strings: `email`, `in:a,b,c`, `uppercase`, `lowercase`, `format`, `digits`, `digits_between`, `max_digits`, `min_digits`.
- Dates (`time.Time`): `date`, `after:today|yesterday`, `before:…`, `before_or_equal:…`.
- Database (`databaseRules`): `exists:table,column`, `unique:table,column`, `current_password`.

Two modifiers:
- `sometimes` — only validate when the field is non-zero.
- `bail` — stop at the first failure for that field.

## Nested fields & wildcards

Address nested fields with dotted keys, and collection elements with a `*` wildcard segment:

```go
map[string]any{
    "address.city":   "required",
    "items.*.price":  "gte:0",
}
```

The `*` expands over the elements present in the data, and errors are recorded under the concrete index — e.g. a failure on the second item is keyed `items.1.price`.

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
