package support

import (
	"context"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/validator"
)

type FormRequestContract interface {
	PrepareForValidation()
	PassedValidation(validated map[string]any)
	Authorize() bool
	Validate(object any, rules map[string]any, prepareForValidation func()) validator.Validator
	Rules() map[string]any
	Errors() validator.Errors
	Validated() map[string]any
	SetContext(ctx context.Context)
	Context() context.Context
	User() foundation.Authenticatable
	SetPathParams(params map[string]string)
	Param(key string) string
	Messages() map[string]string
}

type FormRequest struct {
	validator *validator.Validator
	ctx       context.Context
	user      foundation.Authenticatable
	params    map[string]string
}

func (f *FormRequest) SetContext(ctx context.Context) {
	f.ctx = ctx
	f.user = auth.User(ctx)
	f.getValidatorInstance()
}

func (f *FormRequest) Context() context.Context {
	return f.ctx
}

func (f *FormRequest) SetPathParams(params map[string]string) {
	f.params = params
}

func (f *FormRequest) Param(key string) string {
	if f.params == nil {
		return ""
	}

	return f.params[key]
}

func (f *FormRequest) setValidatorInstance(validator *validator.Validator) {
	f.validator = validator
}

func (f *FormRequest) getValidatorInstance() *validator.Validator {
	if f.validator != nil {
		return f.validator
	}

	validator := new(validator.Validator)
	f.setValidatorInstance(validator)
	return validator
}

func (f *FormRequest) Validate(object any, rules map[string]any, prepareForValidation func()) validator.Validator {

	if f.validator == nil {
		f.getValidatorInstance()
	}

	f.validator.Validate(f.ctx, object, rules, prepareForValidation)

	return *f.validator
}

func (f *FormRequest) Errors() validator.Errors {
	return f.validator.Errors()
}

// Validated returns the subset of the request body that was covered by
// Rules() and present in the input — see validator.Validator.Validated. Any
// mutation PassedValidation makes to the map it receives is visible here
// too, since maps are reference types.
func (f *FormRequest) Validated() map[string]any {
	return f.validator.Validated()
}

func (f *FormRequest) Rules() map[string]any {
	return map[string]any{}
}

func (f *FormRequest) Authorize() bool { return true }

func (f *FormRequest) PrepareForValidation() {}

func (f *FormRequest) PassedValidation(validated map[string]any) {}

func (f *FormRequest) PassesAuthorization() bool { return true }

func (f *FormRequest) FailedAuthorization() {
	// return here everything for a 403 status code
}

func (f *FormRequest) User() foundation.Authenticatable {
	return f.user
}

func (f *FormRequest) Messages() map[string]string {
	return make(map[string]string)
}
