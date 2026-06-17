package validator

import "reflect"

var defaultRules = []string{
	"required",
	"required_if",
	"max",
	"min",
	"gt",
	"gte",
	"lte",
	"lt",
	"between",
	"different",
	"email",
	"exists",
	"unique",
	"current_password",
	"in",
	"uppercase",
	"lowercase",
	"date",
	"after",
	"before",
	"before_or_equal",
	"format",
	"digits",
	"digits_between",
	"max_digits",
	"min_digits",
}

var dateRules = []string{
	"date",
	"after",
	"before",
	"before_or_equal",
	"format",
}

var arrayRules = []string{
	"in",
}

var databaseRules = []string{
	"exists",
	"unique",
}

type Rule struct{}

type RuleConstraints interface {
	Constraints() string
}

type Errors map[string][]string

type RuleContract interface {
	Rules() map[string]any
}

type Validator struct {
	ValidatesAttributes
	errors   Errors
	language *string
	object   reflect.Value
}

type ConditionalRules struct {
	condition    bool
	rules        string
	defaultRules string
}

func (c ConditionalRules) Constraints() string {

	if c.condition {
		return c.rules
	}

	return c.defaultRules
}

func (r Rule) When(condition bool, rules string, defaultRules string) ConditionalRules {
	return ConditionalRules{
		condition:    condition,
		rules:        rules,
		defaultRules: defaultRules,
	}
}
