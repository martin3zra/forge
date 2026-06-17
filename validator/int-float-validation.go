package validator

import (
	"strconv"
)

func (va *ValidatesAttributes) validateIntRules(rule string, fieldValue, ruleValue int) bool {
	if rule == "max" {
		return fieldValue <= ruleValue
	}

	if rule == "max_digits" {
		return ruleValue >= fieldValue
	}

	if rule == "min" {
		return fieldValue >= ruleValue
	}

	if rule == "min_digits" {
		return ruleValue <= fieldValue
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

	if rule == "digits" {
		return digits(fieldValue) == ruleValue
	}

	return true
}

func (va *ValidatesAttributes) validateFloat64Rules(rule string, fieldValue, ruleValue float64) bool {
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

func digits(value int) int {
	return len(strconv.Itoa(value))
}
