package validator_test

import (
	"context"
	"testing"

	"github.com/martin3zra/forge/i18n"
	"github.com/martin3zra/forge/validator"
)

func requiredNameMessage(t *testing.T, ctx context.Context, configure func(*validator.Validator)) string {
	t.Helper()

	var v validator.Validator
	if configure != nil {
		configure(&v)
	}
	v.Validate(ctx, &Contact{}, map[string]any{"name": "required"})

	messages := v.Errors()["name"]
	if len(messages) != 1 {
		t.Fatalf("expected one error for name, got %v", v.Errors())
	}
	return messages[0]
}

func TestLanguageDefaultsToSpanish(t *testing.T) {
	got := requiredNameMessage(t, context.Background(), nil)
	if want := "El campo name es obligatorio."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLanguageFromContextLocale(t *testing.T) {
	ctx := i18n.WithLocale(context.Background(), "en-US")
	got := requiredNameMessage(t, ctx, nil)
	if want := "The name field is required."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetLanguageOverridesContextLocale(t *testing.T) {
	ctx := i18n.WithLocale(context.Background(), "en")
	got := requiredNameMessage(t, ctx, func(v *validator.Validator) { v.SetLanguage("es") })
	if want := "El campo name es obligatorio."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSetDefaultLanguage(t *testing.T) {
	validator.SetDefaultLanguage("en")
	t.Cleanup(func() { validator.SetDefaultLanguage("es") })

	got := requiredNameMessage(t, context.Background(), nil)
	if want := "The name field is required."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

type pointerInForm struct {
	Kind *string `json:"kind"`
}

// Regression: validateIn called reflect.Value.String() on the pointer itself
// ("<*string Value>"), so a valid *string value always failed `in:`.
func TestInRuleOnPointerField(t *testing.T) {
	kind := "estimate"
	var v validator.Validator
	v.Validate(context.Background(), &pointerInForm{Kind: &kind}, map[string]any{
		"kind": "required|in:invoice,estimate,order",
	})
	if len(v.Errors()) > 0 {
		t.Errorf("expected *string in allowed set to pass, got %v", v.Errors())
	}

	other := "receipt"
	v = validator.Validator{}
	v.Validate(context.Background(), &pointerInForm{Kind: &other}, map[string]any{
		"kind": "required|in:invoice,estimate,order",
	})
	if len(v.Errors()["kind"]) == 0 {
		t.Error("expected *string outside allowed set to fail")
	}
}
