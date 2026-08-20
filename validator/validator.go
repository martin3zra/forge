package validator

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/martin3zra/forge/validator/locale"
)

func (v *Validator) Validate(ctx context.Context, object any, rules map[string]any, beforeValidation ...func()) bool {

	for _, cb := range beforeValidation {
		cb()
	}

	v.ctx = ctx
	v.src = newSource(object)
	v.customErrorMessages = v.src.messages()

	// Rule-driven walk: iterate the rule set and resolve each attribute's value
	// from the data source (struct or map). A "*" segment is expanded into the
	// concrete indices present in the data.
	for key, rule := range rules {
		for _, path := range v.src.expand(key) {
			value, present := v.src.get(path)
			v.compileRuleSet(path, value, present, v.resolveRuleComponents(rule))
		}
	}

	return len(v.errors) == 0
}

func (v *Validator) Errors() Errors {
	return v.errors
}

func (v *Validator) messages(attribute, rule, kind string, value ...any) string {
	messages := v.resolveMessages()
	var message any
	var ok bool

	if len(v.customErrorMessages) > 0 {
		customKey := attribute + "." + rule
		message, ok = v.customErrorMessages[customKey]
		if !ok {
			message, ok = messages[rule]
		}
	} else {
		message, ok = messages[rule]
	}

	if ok {
		switch message := message.(type) {
		case map[string]any:
			return v.composeMessage(message[kind].(string), attribute, value...)
		case string:
			return v.composeMessage(message, attribute, value...)
		default:
			return fmt.Sprintf("The %s fail the %s rule.", attribute, rule)
		}
	}

	return fmt.Sprintf("The %s fail the %s rule.", attribute, rule)
}

func (v *Validator) composeMessage(message, attribute string, value ...any) string {
	if v.hasParentKey() {
		if v.keySeparator == "." {
			attribute = fmt.Sprintf("%s.%s", v.parentKey, attribute)
		} else {
			attribute = fmt.Sprintf("%s %d %s", v.parentKey, v.currentPosition+1, attribute)
		}
	}

	re := regexp.MustCompile("%v")
	matches := re.FindAllStringIndex(message, -1)
	if len(matches) >= 1 {
		// join all values to prepare the message
		var args = []any{attribute}
		args = append(args, value...)

		return fmt.Sprintf(message, args...)
	}

	if strings.Contains(message, "%s") {
		return fmt.Sprintf(message, attribute)
	}

	return message
}

func (v *Validator) record(key, message string) {
	if v.errors == nil {
		v.errors = make(map[string][]string)
	}

	v.shouldStopOnFirstFailure(v.canBail)

	// If we're validating a nested object (Array|Slice) we'll pre-append the parent key
	// to the error message, and add the human position to the message, so it's more
	// clear to the user to understand the error.
	if v.hasParentKey() {
		nestedKey := fmt.Sprintf("%s.%d.%s", v.parentKey, v.currentPosition+1, key)
		if v.keySeparator == "." {
			nestedKey = fmt.Sprintf("%s.%s", v.parentKey, key)
		}
		v.errors[nestedKey] = append(v.errors[key], message)
		return
	}

	v.errors[key] = append(v.errors[key], message)
}

func (v *Validator) ensureRuleExists(rule string) bool {
	return slices.Contains(defaultRules, rule)
}

func (v *Validator) resolveKeyBasedOnJsonTag(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

// resolveRuleComponents flattens a field's rule set into individual rules. A
// rule is either a string token or a dbRuleBuilder, which is carried through as
// an object so that its clauses survive intact.
func (v *Validator) resolveRuleComponents(data any) []any {
	switch attributes := data.(type) {
	case []any:
		rules := make([]any, 0, len(attributes))
		for index := range attributes {
			rules = append(rules, v.resolveRuleComponents(attributes[index])...)
		}
		return rules
	case dbRuleBuilder:
		return []any{attributes}
	case ConditionalRules:
		return splitRuleTokens(attributes.Constraints())
	case RuleConstraints:
		return splitRuleTokens(attributes.Constraints())
	case string:
		return splitRuleTokens(attributes)
	default:
		fmt.Println("Field rules not supported!")
		return nil
	}
}

func splitRuleTokens(rules string) []any {
	parts := strings.Split(rules, "|")
	tokens := make([]any, 0, len(parts))
	for _, part := range parts {
		tokens = append(tokens, part)
	}

	return tokens
}

func containsRuleToken(rules []any, token string) bool {
	for _, rule := range rules {
		if name, ok := rule.(string); ok && name == token {
			return true
		}
	}

	return false
}

func (v *Validator) compileRuleSet(key string, value reflect.Value, present bool, rules []any) {
	v.sometimes = containsRuleToken(rules, "sometimes")
	v.canBail = containsRuleToken(rules, "bail")
	hasRequired := containsRuleToken(rules, "required")
	rules = slices.DeleteFunc(rules, func(cmp any) bool {
		name, ok := cmp.(string)
		return ok && (name == "sometimes" || name == "bail")
	})

	// Absent or null attribute: fail "required" (unless guarded by "sometimes"),
	// then skip the remaining rules — they have no value to act on.
	if !present || !value.IsValid() {
		if hasRequired && !v.sometimes {
			v.record(key, v.messages(key, "required", ""))
		}
		return
	}

	for _, entry := range rules {
		if v.stopOnFirstFailure {
			v.shouldStopOnFirstFailure(false)
			break
		}

		if builder, ok := entry.(dbRuleBuilder); ok {
			if v.sometimes && value.IsZero() {
				break
			}

			v.evaluateDatabaseRuleBuilder(key, builder, value)
			continue
		}

		rule, ok := entry.(string)
		if !ok {
			continue
		}

		ruleComponents := strings.Split(rule, ":")

		rule = ruleComponents[0]

		if v.ensureRuleExists(rule) {
			if v.sometimes && value.IsZero() {
				break
			}

			if len(ruleComponents) == 2 {
				v.evaluateRuleWithValues(key, ruleComponents, value)
				continue
			}

			v.evaluateSingleRule(key, ruleComponents[0], value)
		}
	}
}

func (v *Validator) evaluateDatabaseRuleBuilder(key string, builder dbRuleBuilder, value reflect.Value) {
	if !v.validateDatabaseRuleBuilder(key, builder, value) {
		v.record(key, v.messages(key, builder.ruleName(), value.Kind().String(), builder.table()))
	}
}

func (v *Validator) evaluateRuleWithValues(key string, ruleComponents []string, value reflect.Value) {
	hasMultipleAttributes, attributes := v.hasMultipleAttributes(ruleComponents[1])
	if slices.Contains(databaseRules, ruleComponents[0]) {
		//set database handler for the validation.
		if !v.validateDatabaseRules(key, ruleComponents[0], attributes, value) {
			v.record(key, v.messages(key, ruleComponents[0], value.Kind().String(), attributes[0]))
		}
		return
	}

	if slices.Contains(arrayRules, ruleComponents[0]) {
		if !v.validateArrayRules(ruleComponents[0], attributes, value) {
			v.record(key, v.messages(key, ruleComponents[0], value.String(), attributes))
		}
		return
	}

	if slices.Contains(dateRules, ruleComponents[0]) {
		if !v.evaluateDateRule(ruleComponents[0], ruleComponents[1], value.Interface().(time.Time)) {
			v.record(key, v.messages(key, ruleComponents[0], value.String(), attributes))
		}
		return
	}

	if hasMultipleAttributes {
		v.evaluateMultipleValueRule(key, ruleComponents[0], value, attributes)
		return
	}

	v.evaluateSingleValueRule(key, ruleComponents[0], attributes[0], value)
}

func (v *Validator) evaluateDateRule(rule string, ruleValue string, value time.Time) bool {
	now := time.Now()
	switch rule {
	case "after":
		if ruleValue == "yesterday" {
			return value.After(now.AddDate(0, 0, -1))
		}
		if ruleValue == "today" {
			return value.After(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
		}
	case "before":
		if ruleValue == "yesterday" {
			return value.Before(now.AddDate(0, 0, -1))
		}
		if ruleValue == "today" {
			return value.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
		}
	case "before_or_equals":
		if ruleValue == "yesterday" {
			return !value.After(now.AddDate(0, 0, -1))
		}
		if ruleValue == "today" {
			return !value.After(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
		}
	}

	return true
}

func (v *Validator) evaluateMultipleValueRule(key, rule string, value reflect.Value, attributes []string) {
	fieldValue := value
	if rule == "digits_between" {
		if value.Kind() != reflect.Int {
			return
		}
		fieldValue = reflect.ValueOf(digits(int(value.Int())))
	}

	if rule == "required_if" {
		siblingValue, e := v.sibling(attributes[0])
		if !e {
			return
		}
		if !slices.Contains(attributes[1:], siblingValue) {
			return
		}
		if !v.validateRuleWithoutAttributes("required", value) {
			v.record(key, v.messages(key, "required", getDataTypeUsingReflection(value), attributes[0], attributes[1]))
		}
		return
	}

	if !v.validateBetween(fieldValue, attributes) {
		v.record(key, v.messages(key, rule, getDataTypeUsingReflection(value), attributes[0], attributes[1]))
	}
}

func (v *Validator) evaluateSingleValueRule(key, rule string, ruleValue any, value reflect.Value) {
	// 👇 Handle pointers
	value = unwrapValue(value)

	castedRuleValue, _ := strconv.Atoi(ruleValue.(string))
	switch value.Kind() {
	case reflect.Int:
		v.evaluateIntRules(key, rule, int(value.Int()), castedRuleValue)
	case reflect.String:
		v.evaluateStringRules(key, rule, value.String(), castedRuleValue)
	case reflect.Float64:
		castedRuleValue, _ := strconv.ParseFloat(ruleValue.(string), 64)
		v.evaluateFloat64Rules(key, rule, value.Float(), castedRuleValue)
	case reflect.Slice:
		v.evaluateSliceRules(key, rule, value, castedRuleValue)
	default:
		fmt.Println("data type not supported yet!", value.Type(), key)
	}

}

func (v *Validator) evaluateSingleRule(key, rule string, value reflect.Value) {
	if !v.validateRuleWithoutAttributes(rule, value) {
		v.record(key, v.messages(key, rule, value.Kind().String(), value))
	}
}

func (v *Validator) evaluateIntRules(key, rule string, fieldValue, ruleValue int) {
	if rule == "max_digits" || rule == "min_digits" {
		if !v.validateIntRules(rule, digits(fieldValue), ruleValue) {
			v.record(key, v.messages(key, rule, "int", ruleValue))
		}
		return
	}

	if !v.validateIntRules(rule, fieldValue, ruleValue) {
		v.record(key, v.messages(key, rule, "int", ruleValue))
	}
}

func (v *Validator) evaluateSliceRules(key, rule string, fieldValue reflect.Value, ruleValue int) {
	if !v.validateSliceRules(rule, fieldValue, ruleValue) {
		v.record(key, v.messages(key, rule, "slice", ruleValue))
	}
}

func (v *Validator) evaluateFloat64Rules(key, rule string, fieldValue, ruleValue float64) {
	if !v.validateFloat64Rules(rule, fieldValue, ruleValue) {
		v.record(key, v.messages(key, rule, "int", ruleValue))
	}
}

func (v *Validator) evaluateStringRules(key, rule string, fieldValue string, ruleValue int) {
	if !v.validateIntRules(rule, len(fieldValue), ruleValue) {
		v.record(key, v.messages(key, rule, "string", ruleValue))
	}
}

func (v *Validator) resolveLanguage(fallback string) string {
	if v.language == nil {
		return fallback
	}

	return *v.language
}

func (v *Validator) resolveMessages() map[string]any {
	if v.resolveLanguage("es") == "es" {
		return locale.SpanishMessages()
	}

	return locale.EnglishMessages()
}

// sibling resolves another attribute's value from the data source as a string,
// for cross-field rules such as required_if. Works for both struct and map.
func (v *Validator) sibling(name string) (string, bool) {
	if v.src == nil {
		return "", false
	}
	val, ok := v.src.get(name)
	if !ok || !val.IsValid() {
		return "", false
	}
	val = unwrapValue(val)
	if !val.IsValid() {
		return "", false
	}
	if val.Kind() == reflect.String {
		return val.String(), true
	}
	return fmt.Sprintf("%v", val.Interface()), true
}

func getDataTypeUsingReflection(value reflect.Value) string {
	switch value.Kind() {
	case reflect.Float32, reflect.Float64, reflect.Int32, reflect.Int64:
		return "int"
	default:
		return "string"
	}
}

func unwrapValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}
