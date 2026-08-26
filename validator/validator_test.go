package validator_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/validator"

	_ "github.com/lib/pq"
)

type Contact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Age   int    `json:"age"`
}

type Address struct {
	Line string `json:"line"`
}

type Person struct {
	Name     string    `json:"name,omitempty"`
	LastName string    `json:"last_name"`
	Email    string    `json:"email"`
	Age      int       `json:"age"`
	Gender   string    `json:"gender"`
	Contacts []Contact `json:"contacts"`
	Address  Address   `json:"address"`
}

func (p Person) Rules() map[string]any {
	return map[string]any{
		"name":      "required|max:2",
		"last_name": "required",
		"email":     "required|email",
		"age":       validator.Rule{}.Numeric().GreaterThan(18).Different(35).Max(55),
	}
}

func (p Person) Messages() map[string]string {
	return map[string]string{
		"name.required": "Hey debes de especificar el Nombre.",
	}
}

func TestNumericRule(t *testing.T) {
	person := Person{
		Name: "Jane",
		Age:  25,
	}
	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"name": "required|min:2|max:4",
		"age":  "between:18,30",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestMinOnNumberRule(t *testing.T) {
	person := Person{
		Age: 12,
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"age": "min:1",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestBetweenRule(t *testing.T) {
	person := Person{
		Age: 36,
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"age": "between:32,40",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestRuleWithoutAttributes(t *testing.T) {
	person := Person{
		Email: "jane@example.com",
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"email": "required|email",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestRuleSometimes(t *testing.T) {
	person := Person{
		Email: "jane@example.com",
		Age:   20,
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"email": "sometimes|email",
		"age":   "required|gte:20",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestExistsRule(t *testing.T) {
	person := Person{
		Email: "martin3zra@gmail.com",
	}

	db, err := sql.Open("postgres", "host=localhost port=5433 dbname=acme user=postgres password=secret sslmode=disable")
	if err != nil {
		t.Fail()
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), database.ConnectionKey{}, db)
	var valid = validator.Validator{}
	valid.Validate(ctx, &person, map[string]any{
		"email": "required|email|exists:users",
	})
	if len(valid.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", valid.Errors())
	}
}

func TestUniqueRule(t *testing.T) {
	person := Person{
		Email: "martin3zra@gmail.com",
	}

	db, err := sql.Open("postgres", "host=localhost port=5433 dbname=acme user=postgres password=secret sslmode=disable")
	if err != nil {
		t.Fail()
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), database.ConnectionKey{}, db)
	var valid = validator.Validator{}
	// valid.Ignore(1, "id")
	valid.Validate(ctx, &person, map[string]any{
		// "email": "required|email|unique.ignore:users,email",
		"email": []any{
			"required",
			"email",
			validator.Rule{}.
				Unique("users", "email").
				Where("id", 1).
				WhereNull("deleted_at").
				Ignore(person.Email, "email"), //"unique.ignore:users,email",
		},
	})
	if len(valid.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", valid.Errors())
	}
}

// A scoped exists rule must only match rows that satisfy every clause.
func TestExistsRuleWithClauses(t *testing.T) {
	person := Person{
		Email: "martin3zra@gmail.com",
	}

	db, err := sql.Open("postgres", "host=localhost port=5433 dbname=acme user=postgres password=secret sslmode=disable")
	if err != nil {
		t.Fail()
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), database.ConnectionKey{}, db)

	var matches = validator.Validator{}
	matches.Validate(ctx, &person, map[string]any{
		"email": []any{
			"required",
			validator.Rule{}.Exists("users", "email").WhereNull("deleted_at"),
		},
	})
	if len(matches.Errors()) > 0 {
		t.Errorf("expected the row to be found:\n %v", matches.Errors())
	}

	// The same row, scoped to an id it does not have, must not be found.
	var scopedOut = validator.Validator{}
	scopedOut.Validate(ctx, &person, map[string]any{
		"email": []any{
			"required",
			validator.Rule{}.Exists("users", "email").Where("id", -1),
		},
	})
	if len(scopedOut.Errors()) == 0 {
		t.Error("expected the scoped exists rule to reject a row outside its scope")
	}
}

func TestSliceFields(t *testing.T) {

	person := Person{
		Email: "martin3zra@gmail.com",
		Contacts: []Contact{
			{
				Name:  "Natasha Martinez Garcia",
				Phone: "8099879235",
				Age:   23,
			},
			{
				Name:  "Massiel Natali Garcia",
				Phone: "8099879232",
				Age:   18,
			},
		},
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"email":            "required|email",
		"contacts.*.name":  "required|min:10",
		"contacts.*.phone": "required|min:3|max:11",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestNestedFields(t *testing.T) {

	person := Person{
		Email:   "martin3zra@gmail.com",
		Address: Address{Line: "C/Mama Tingo"},
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"email":        "required|email",
		"address.line": "required|min:10",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestInRule(t *testing.T) {

	person := Person{
		Gender: "f",
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"gender": "required|in:m,f",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestLowerAndUpperCaseRule(t *testing.T) {

	person := Person{
		Name:  "ALFREDO",
		Email: "martin3zra@gmail.com",
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"name":  "required|uppercase",
		"email": "required|lowercase",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestDigitsRule(t *testing.T) {

	person := Person{
		Age: 22,
	}

	var validator = validator.Validator{}
	validator.Validate(context.Background(), &person, map[string]any{
		"age": "required|min_digits:2",
	})
	if len(validator.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", validator.Errors())
	}
}

func TestConditionalRule(t *testing.T) {

	person := Person{
		Age:  22,
		Name: "Jane",
	}

	var vali = validator.Validator{}
	vali.Validate(context.Background(), &person, map[string]any{
		"age":  "required|min:18",
		"name": []any{"required", validator.Rule{}.When(person.Age > 30, "min:10|max:100", "min:3")},
	})
	if len(vali.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", vali.Errors())
	}
}

func TestCustomErrorMessage(t *testing.T) {

	person := Person{
		Age: 22,
	}

	var vali = validator.Validator{}
	vali.Validate(context.Background(), &person, map[string]any{
		"age":  "required|min:18",
		"name": "required",
	})
	if len(vali.Errors()) == 0 {
		t.Errorf("validation fails:\n %v", vali.Errors())
	}
}

func TestRequiredIfStringRule(t *testing.T) {
	form := Contact{
		Name: "estimate",
		// Age:  10,
	}

	var v = validator.Validator{}
	v.Validate(context.Background(), &form, map[string]any{
		"name": "required|in:invoice,estimate,order",
		"age":  "required_if:name,invoice,template",
	})

	if len(v.Errors()) > 0 {
		t.Errorf("expected validation to fail for missing type when transaction_kind is invoice: %v", v.Errors())
	}
}

// ---------------------
// map[string]any input
// ---------------------

func TestMapValidInput(t *testing.T) {
	data := map[string]any{
		"name":  "Jane",
		"email": "jane@example.com",
		"age":   25.0, // JSON numbers decode to float64
		"address": map[string]any{
			"line": "C/Mama Tingo 123",
		},
		"contacts": []any{
			map[string]any{"name": "Natasha Martinez Garcia", "phone": "8099879235"},
		},
	}

	var v = validator.Validator{}
	v.Validate(context.Background(), data, map[string]any{
		"name":             "required|min:2|max:4",
		"email":            "required|email",
		"age":              "between:18,30",
		"address.line":     "required|min:10",
		"contacts.*.name":  "required|min:10",
		"contacts.*.phone": "required|min:3|max:11",
	})
	if len(v.Errors()) > 0 {
		t.Errorf("validation should pass for map input:\n %v", v.Errors())
	}
}

func TestMapInRule(t *testing.T) {
	var v = validator.Validator{}
	v.Validate(context.Background(), map[string]any{"gender": "f"}, map[string]any{
		"gender": "required|in:m,f",
	})
	if len(v.Errors()) > 0 {
		t.Errorf("validation should pass:\n %v", v.Errors())
	}
}

func TestMapSometimesSkipsAbsent(t *testing.T) {
	var v = validator.Validator{}
	// "email" is absent; sometimes means it is skipped rather than failing.
	v.Validate(context.Background(), map[string]any{"age": 20.0}, map[string]any{
		"email": "sometimes|email",
		"age":   "required|gte:20",
	})
	if len(v.Errors()) > 0 {
		t.Errorf("validation should pass:\n %v", v.Errors())
	}
}

func TestMapDigitsOnIntegralFloat(t *testing.T) {
	var v = validator.Validator{}
	v.Validate(context.Background(), map[string]any{"age": 22.0}, map[string]any{
		"age": "required|min_digits:2",
	})
	if len(v.Errors()) > 0 {
		t.Errorf("integer rules must apply to integral floats:\n %v", v.Errors())
	}
}

// Absent required key must fail — the Laravel semantic the struct path cannot
// express (struct fields are always present as zero values).
func TestMapAbsentRequiredFails(t *testing.T) {
	var v = validator.Validator{}
	v.Validate(context.Background(), map[string]any{"age": 20.0}, map[string]any{
		"name": "required",
		"age":  "required",
	})
	if _, ok := v.Errors()["name"]; !ok {
		t.Errorf("expected a 'name' required error for the absent key, got: %v", v.Errors())
	}
}

func TestMapNullRequiredFails(t *testing.T) {
	var v = validator.Validator{}
	v.Validate(context.Background(), map[string]any{"name": nil}, map[string]any{
		"name": "required",
	})
	if _, ok := v.Errors()["name"]; !ok {
		t.Errorf("expected a 'name' required error for null value, got: %v", v.Errors())
	}
}

// ---------------------
// wildcard now actually applies (previously a no-op)
// ---------------------

func TestWildcardSliceFailsStruct(t *testing.T) {
	person := Person{
		Email: "martin3zra@gmail.com",
		Contacts: []Contact{
			{Name: "Bob", Phone: "809"}, // name too short for min:10
		},
	}

	var v = validator.Validator{}
	v.Validate(context.Background(), &person, map[string]any{
		"contacts.*.name": "required|min:10",
	})
	if _, ok := v.Errors()["contacts.0.name"]; !ok {
		t.Errorf("expected error at contacts.0.name, got: %v", v.Errors())
	}
}

func TestWildcardSliceFailsMap(t *testing.T) {
	data := map[string]any{
		"contacts": []any{
			map[string]any{"name": "Bob"},
		},
	}

	var v = validator.Validator{}
	v.Validate(context.Background(), data, map[string]any{
		"contacts.*.name": "required|min:10",
	})
	if _, ok := v.Errors()["contacts.0.name"]; !ok {
		t.Errorf("expected error at contacts.0.name, got: %v", v.Errors())
	}
}

// ---------------------
// Validated()
// ---------------------

// Only fields that appear in the rule set are reflected into Validated —
// LastName has no rule and must not leak in, even though it has a value.
func TestValidatedIncludesOnlyRuleCoveredFields(t *testing.T) {
	person := Person{
		Name:     "Jane",
		LastName: "Doe",
		Email:    "jane@example.com",
	}

	var v = validator.Validator{}
	if ok := v.Validate(context.Background(), &person, map[string]any{
		"name":  "required",
		"email": "required|email",
	}); !ok {
		t.Fatalf("validation fails:\n %v", v.Errors())
	}

	want := map[string]any{"name": "Jane", "email": "jane@example.com"}
	if got := v.Validated(); !reflect.DeepEqual(got, want) {
		t.Errorf("Validated() = %#v, want %#v", got, want)
	}
}

// Dotted and "*"-wildcard rule keys reconstruct into the original nested
// shape — a map for "address.line", a slice of maps for "contacts.*.name" —
// and fields with no rule (Phone, Age) are excluded from each nested item.
func TestValidatedNestedAndWildcardFields(t *testing.T) {
	person := Person{
		Email:   "jane@example.com",
		Address: Address{Line: "C/Mama Tingo"},
		Contacts: []Contact{
			{Name: "Natasha Martinez Garcia", Phone: "8099879235", Age: 23},
			{Name: "Massiel Natali Garcia", Phone: "8099879232", Age: 18},
		},
	}

	var v = validator.Validator{}
	if ok := v.Validate(context.Background(), &person, map[string]any{
		"email":           "required|email",
		"address.line":    "required",
		"contacts.*.name": "required",
	}); !ok {
		t.Fatalf("validation fails:\n %v", v.Errors())
	}

	want := map[string]any{
		"email":   "jane@example.com",
		"address": map[string]any{"line": "C/Mama Tingo"},
		"contacts": []any{
			map[string]any{"name": "Natasha Martinez Garcia"},
			map[string]any{"name": "Massiel Natali Garcia"},
		},
	}
	if got := v.Validated(); !reflect.DeepEqual(got, want) {
		t.Errorf("Validated() = %#v, want %#v", got, want)
	}
}

// A non-nil pointer field is dereferenced to its underlying value; a nil
// pointer is excluded entirely, the same way an absent field is.
func TestValidatedDereferencesPointersAndExcludesNil(t *testing.T) {
	type Form struct {
		Nickname *string `json:"nickname"`
		Notes    *string `json:"notes"`
	}
	nickname := "JD"
	form := Form{Nickname: &nickname}

	var v = validator.Validator{}
	if ok := v.Validate(context.Background(), &form, map[string]any{
		"nickname": "sometimes",
		"notes":    "sometimes",
	}); !ok {
		t.Fatalf("validation fails:\n %v", v.Errors())
	}

	want := map[string]any{"nickname": "JD"}
	if got := v.Validated(); !reflect.DeepEqual(got, want) {
		t.Errorf("Validated() = %#v, want %#v", got, want)
	}
}

// map[string]any input works the same way as a struct: only rule-covered
// keys are reflected, nested paths are rebuilt, and unrelated sibling keys
// (both top-level and nested) are excluded.
func TestValidatedMapSource(t *testing.T) {
	data := map[string]any{
		"name":  "Jane",
		"email": "jane@example.com",
		"extra": "should not appear",
		"address": map[string]any{
			"line": "C/Mama Tingo 123",
			"city": "should not appear either",
		},
	}

	var v = validator.Validator{}
	if ok := v.Validate(context.Background(), data, map[string]any{
		"name":         "required",
		"address.line": "required",
	}); !ok {
		t.Fatalf("validation should pass:\n %v", v.Errors())
	}

	want := map[string]any{
		"name":    "Jane",
		"address": map[string]any{"line": "C/Mama Tingo 123"},
	}
	if got := v.Validated(); !reflect.DeepEqual(got, want) {
		t.Errorf("Validated() = %#v, want %#v", got, want)
	}
}

// A "sometimes" field absent from a map source is skipped entirely, so it
// must not appear in Validated either.
func TestValidatedExcludesAbsentSometimesMapKey(t *testing.T) {
	var v = validator.Validator{}
	if ok := v.Validate(context.Background(), map[string]any{"age": 20.0}, map[string]any{
		"email": "sometimes|email",
		"age":   "required|gte:20",
	}); !ok {
		t.Fatalf("validation should pass:\n %v", v.Errors())
	}

	want := map[string]any{"age": 20}
	if got := v.Validated(); !reflect.DeepEqual(got, want) {
		t.Errorf("Validated() = %#v, want %#v", got, want)
	}
}

// Validated must not surface partial/stale data from a failed run.
func TestValidatedIsNilWhenValidationFails(t *testing.T) {
	var v = validator.Validator{}
	if ok := v.Validate(context.Background(), map[string]any{"age": 20.0}, map[string]any{
		"name": "required",
		"age":  "required",
	}); ok {
		t.Fatalf("expected validation to fail")
	}

	if got := v.Validated(); got != nil {
		t.Errorf("Validated() = %#v, want nil after a failed validation", got)
	}
}
