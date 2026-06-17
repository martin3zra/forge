package validator

import (
	"fmt"
	"strings"
)

func (r Rule) Numeric() *Numeric {
	return new(Numeric)
}

type Numeric struct {
	constraints []string
}

func (n Numeric) Constraints() string {
	var rules = make([]string, 0)

	rules = append(rules, n.constraints...)

	return strings.Join(rules, "|")
}

func (n Numeric) Between(min, max int) Numeric {
	return n.addRule(fmt.Sprintf("between:%d,%d", min, max))
}

func (n Numeric) Min(value int) Numeric {
	return n.addRule(fmt.Sprintf("min:%d", value))
}

func (n Numeric) Max(value int) Numeric {
	return n.addRule(fmt.Sprintf("max:%d", value))
}

func (n Numeric) Different(value int) Numeric {
	return n.addRule(fmt.Sprintf("different:%d", value))
}

func (n Numeric) GreaterThan(value int) Numeric {
	return n.addRule(fmt.Sprintf("gt:%d", value))
}

func (n Numeric) GreaterThanOrEqual(value int) Numeric {
	return n.addRule(fmt.Sprintf("gte:%d", value))
}
func (n Numeric) LessThan(value int) Numeric {
	return n.addRule(fmt.Sprintf("lt:%d", value))
}

func (n Numeric) LessThanOrEqual(value int) Numeric {
	return n.addRule(fmt.Sprintf("lte:%d", value))
}

func (n Numeric) Digits(value int) Numeric {
	return n.addRule(fmt.Sprintf("digits:%d", value))
}

func (n Numeric) addRule(rules string) Numeric {
	n.constraints = append(n.constraints, rules)
	return n
}
