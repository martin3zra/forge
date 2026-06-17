package validator_test

import (
	"context"
	"database/sql"
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
				Where("current_company_id", 1).
				Ignore(person.Email, "email"), //"unique.ignore:users,email",
		},
	})
	if len(valid.Errors()) > 0 {
		t.Errorf("validation fails:\n %v", valid.Errors())
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
