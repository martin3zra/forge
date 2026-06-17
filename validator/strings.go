package validator

import "strings"

type String struct {
	constraints []string
}

func (s String) Constraints() string {
	var rules = make([]string, 0)

	rules = append(rules, s.constraints...)

	return strings.Join(rules, "|")
}

func (s String) Email() String {
	return s.addRule("email")
}

func (s String) addRule(rules string) String {
	s.constraints = append(s.constraints, rules)
	return s
}
