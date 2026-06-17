package validator

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/martin3zra/forge/auth"
)

type ValidatesAttributes struct {
	ctx                context.Context
	sometimes          bool
	canBail            bool
	stopOnFirstFailure bool
	// needsToIgnore       bool
	ignore              any
	column              string
	currentPosition     int
	parentKey           string
	keySeparator        string
	customErrorMessages map[string]string
}

func (va *ValidatesAttributes) shouldStopOnFirstFailure(shouldStop bool) {
	va.stopOnFirstFailure = shouldStop
}

func (va *ValidatesAttributes) setKeySeparator(separator string) {
	va.keySeparator = separator
}

func (va *ValidatesAttributes) setParentKey(key string) {
	va.parentKey = key
}

func (va *ValidatesAttributes) resetParentKey() {
	va.parentKey = ""
}

func (va *ValidatesAttributes) hasParentKey() bool {
	return va.parentKey != ""
}

func (v *ValidatesAttributes) onStructEnd(path string) {
	fmt.Println("✅ Finished processing:", path)
	// Optional: push to log, call hook, etc.
}

func (va *ValidatesAttributes) Ignore(ignore any, column ...string) {
	va.ignore = ignore
	if len(column) == 0 {
		va.column = "id"
		return
	}
	va.column = column[0]
}

func (va *ValidatesAttributes) validateNumericRules(rule string, fieldValue, ruleValue int) bool {
	if rule == "max" {
		return fieldValue <= ruleValue
	}

	if rule == "min" {
		return fieldValue >= ruleValue
	}

	if rule == "gte" {
		return fieldValue >= ruleValue
	}

	if rule == "gt" {
		return fieldValue > ruleValue
	}

	if rule == "lte" {
		return fieldValue <= ruleValue
	}

	if rule == "lt" {
		return fieldValue < ruleValue
	}

	if rule == "different" {
		return fieldValue != ruleValue
	}

	return true
}

func (va *ValidatesAttributes) validateBetween(value reflect.Value, params []string) bool {
	va.requireParameterCount(2, params, "bewteen")

	minValue, _ := strconv.Atoi(params[0])
	maxValue, _ := strconv.Atoi(params[1])

	switch value.Kind() {
	case reflect.Int, reflect.Int64:
		return value.Int() >= int64(minValue) && value.Int() <= int64(maxValue)
	case reflect.Float32, reflect.Float64:
		return value.Float() >= float64(minValue) && value.Float() <= float64(maxValue)
	default:
	}

	return true
}

func (va *ValidatesAttributes) validateRuleWithoutAttributes(rule string, value reflect.Value) bool {

	if rule == "required" {
		if value.Kind() == reflect.Bool {
			return true
		}
		return !value.IsZero()
	}

	if rule == "email" {
		return va.validateEmail(value.String())
	}

	if rule == "current_password" {
		return va.validateCurrentPassword(value)
	}

	if rule == "lowercase" {
		return strings.ToLower(value.String()) == value.String()
	}

	if rule == "uppercase" {
		return strings.ToUpper(value.String()) == value.String()
	}

	if rule == "date" {
		return !value.Interface().(time.Time).IsZero()
	}

	return true
}

func (va *ValidatesAttributes) validateCurrentPassword(password reflect.Value) bool {

	if password.IsZero() || !password.IsValid() {
		return false
	}

	user := auth.User(va.ctx)
	// match the given password against the logged user.
	guard := auth.NewAuth(va.ctx)
	authPassword, err := guard.GetCurrentPassword(user.GetAuthIdentifier())
	if err != nil {
		return false
	}
	return guard.EnsureIsCurrentPassword(authPassword, password.String())
}

func (va *ValidatesAttributes) validateEmail(email string) bool {
	return newEmailRule().validEmailAddress(email)
}

func (va *ValidatesAttributes) validateDatabaseRules(key, rule string, attributes []string, value reflect.Value) bool {
	if rule == "exists" {
		return va.validateExists(key, attributes, rule, value)
	}

	if rule == "unique" {
		return va.validateUnique(key, attributes, rule, value)
	}

	return true
}

func (va *ValidatesAttributes) validateExists(key string, attributes []string, rule string, value reflect.Value) bool {
	va.requireParameterCount(1, attributes, rule)

	count := newDatabaseRule(va.ctx, key, attributes, value).getCount()

	return count > 0
}

func (va *ValidatesAttributes) validateUnique(key string, attributes []string, rule string, value reflect.Value) bool {
	va.requireParameterCount(1, attributes, rule)

	dbRule := newDatabaseRule(va.ctx, key, attributes, value)

	contrainsts := attributes[2:]
	if len(contrainsts) >= 2 {
		dbRule.ignore(contrainsts[0], contrainsts[1])

		if len(contrainsts[2:]) > 0 {
			wheres := splitWheres(contrainsts[2:])
			return dbRule.addWheres(wheres).getCount() == 0
		}
	}

	return dbRule.getCount() == 0
}

// Require a certain number of parameters to be present.
func (va *ValidatesAttributes) requireParameterCount(count int, params []string, rule string) {
	if len(params) < count {
		panic(fmt.Sprintf("Validation rule %s requires at least %d parameters.", rule, count))
	}
}

func (va *ValidatesAttributes) hasMultipleAttributes(ruleAttributes string) (bool, []string) {
	parts := strings.Split(ruleAttributes, ",")
	return len(parts) > 1, parts
}

func (va *ValidatesAttributes) validateArrayRules(rule string, attributes []string, value reflect.Value) bool {
	if rule == "in" {
		return va.validateIn(attributes, value)
	}

	return true
}

func (va *ValidatesAttributes) validateIn(attributes []string, value reflect.Value) bool {
	return slices.Contains(attributes, value.String())
}

func splitWheres(attr []string) [][]string {
	var result [][]string
	for i := 0; i < len(attr); i += 2 {
		end := min(i+2, len(attr))
		result = append(result, attr[i:end])
	}
	return result
}
