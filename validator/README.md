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

## Database rules

`exists` and `unique` accept a fluent builder that adds where-clauses to the lookup. Use it to scope a row to the tenant that owns it, so `exists:tax_receipts,id` cannot match a receipt belonging to another company:

```go
var r validator.Rule

map[string]any{
    "tax_receipt_id": []any{
        "required",
        r.Exists("tax_receipts", "id").
            Where("company_id", companyID).
            WhereNull("deleted_at"),
    },
    "email": []any{
        "required", "email",
        r.Unique("users", "email").
            Where("company_id", companyID).
            Ignore(user.ID, "id"),
    },
}
```

Both builders support `Where(column, value)`, `WhereNull(column)`, `WhereNotNull(column)` and `WhereIn(column, values...)`; `Unique` adds `Ignore(value, column)`. When `Exists` is given no column, the field's `json` key is used — the same shorthand as the `exists:users` rule string.

Inside a `support.FormRequest`, the request context is populated before `Rules()` runs, so `f.Context()` and `f.User()` are available when building the clauses. Wrap the repetition in one helper rather than threading the tenant id through every rule:

```go
type ScopedRequest struct{ support.FormRequest }

func (f *ScopedRequest) exists(table string, column ...string) *validator.Exists {
    return validator.Rule{}.Exists(table, column...).
        Where("company_id", tenant.From(f.Context()))
}
```

Values passed to the builder become bound query parameters, so any type the driver understands works — including `uuid.UUID`. The equivalent rule *strings* (`exists:table,column`, `unique:table,column`) still work, but they encode clause values into the rule text and so cannot carry a value containing `,`, `:`, `|`, `^` or `__`, nor express `IS NULL`/`IN`. Prefer the builder.

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
