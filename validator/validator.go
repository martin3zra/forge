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

	v.validateAttributes(object, rules, "")

	return len(v.errors) == 0
}

// Validate a given struct/object/attribute against a rule set
func (v *Validator) validateAttributes(object any, rules map[string]any, prefix string) {
	val := reflect.ValueOf(object)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	v.customErrorMessages = map[string]string{}
	hasMessages := val.MethodByName("Messages")
	if hasMessages.IsValid() && hasMessages.Kind() == reflect.Func {
		result := hasMessages.Call([]reflect.Value{})[0].Interface()
		if messages, ok := result.(map[string]string); ok {
			v.customErrorMessages = messages
		}
	}

	// Ensure struct
	if val.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		key := v.resolveKeyBasedOnJsonTag(fieldType)
		isStruct := field.Kind() == reflect.Struct && field.Type() != reflect.TypeOf(time.Time{})
		isSlice := field.Kind() == reflect.Slice

		// Build fullKey with prefix
		var fullKey string
		if key != "" {
			if prefix != "" {
				fullKey = prefix + "." + key
			} else {
				fullKey = key
			}
		} else if fieldType.Anonymous {
			// Embedded/anonymous struct inherits prefix
			v.validateAttributes(field.Interface(), rules, prefix)
			continue
		} else {
			// Skip unnamed fields that are not anonymous
			continue
		}

		switch {
		case isStruct:
			if field.Type() == reflect.TypeOf(time.Time{}) {
				if rule, ok := rules[fullKey]; ok {
					v.compileRuleSet(fullKey, field, v.resolveRuleComponents(rule))
				}
				continue
			}
			v.validateAttributes(field.Interface(), rules, fullKey)

		case isSlice:
			if field.Len() == 0 {
				if rule, ok := rules[fullKey]; ok {
					v.compileRuleSet(fullKey, field, v.resolveRuleComponents(rule))
				}
				continue
			}
			for j := 0; j < field.Len(); j++ {
				v.validateAttributes(field.Index(j).Interface(), rules, fullKey+"[*]")
			}

		default:
			if rule, ok := rules[fullKey]; ok {
				v.object = val
				v.compileRuleSet(fullKey, field, v.resolveRuleComponents(rule))
			}
		}
	}
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

// func (v *Validator) resolveKeyBasedOnJsonTag(field reflect.Type, index int) string {
// 	return strings.Split(field.Field(index).Tag.Get("json"), ",")[0]
// }

func (v *Validator) resolveKeyBasedOnJsonTag(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func (v *Validator) resolveRuleComponents(data any) []string {
	mixedData, ok := data.([]any)
	if ok {
		rules := make([]string, 0)
		for index := range mixedData {
			switch attributes := mixedData[index].(type) {
			case ConditionalRules:
				rules = append(rules, strings.Split(attributes.Constraints(), "|")...)
			case RuleConstraints:
				rules = append(rules, strings.Split(attributes.Constraints(), "|")...)
			case string:
				rules = append(rules, strings.Split(attributes, "|")...)
			default:
				fmt.Println("Field rules not supported!")
			}
		}
		return rules
	}

	ruleContractValue, ok := data.(RuleConstraints)
	if ok {
		return strings.Split(ruleContractValue.Constraints(), "|")
	}

	return strings.Split(data.(string), "|")
}

func (v *Validator) compileRuleSet(key string, value reflect.Value, rules []string) {
	v.sometimes = slices.Contains(rules, "sometimes")
	v.canBail = slices.Contains(rules, "bail")
	rules = slices.DeleteFunc(rules, func(cmp string) bool {
		return cmp == "sometimes" || cmp == "bail"
	})

	for _, rule := range rules {
		if v.stopOnFirstFailure {
			v.shouldStopOnFirstFailure(false)
			break
		}
		// v.needsToIgnore = false
		ruleComponents := strings.Split(rule, ":")

		rule := ruleComponents[0]
		// prepends := strings.Split(rule, ".")

		// if len(prepends) > 1 {
		// 	v.needsToIgnore = prepends[1] == "ignore"
		// 	rule = prepends[0]
		// 	ruleComponents[0] = rule
		// }

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
		siblingValue, e := getFieldValueByJSONTag(v.object, attributes[0])
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

func getFieldValueByJSONTag(v reflect.Value, tag string) (string, bool) {
	// If v is a pointer, resolve it
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", false
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		// Handle omitempty etc.
		jsonTag = strings.Split(jsonTag, ",")[0]

		if jsonTag == tag {
			return v.Field(i).String(), true
		}
	}
	return "", false
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
