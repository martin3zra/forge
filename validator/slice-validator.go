package validator

import "reflect"

func (va *ValidatesAttributes) validateSliceRules(rule string, fieldValue reflect.Value, ruleValue int) bool {
	if rule == "max" {
		return fieldValue.Len() <= ruleValue
	}

	if rule == "min" {
		return fieldValue.Len() >= ruleValue
	}

	return true
}
