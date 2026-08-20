package validator

import (
	"fmt"
	"strings"
)

// clauseOp enumerates the predicates a database rule can append to the
// existence check it compiles.
type clauseOp int

const (
	opEq clauseOp = iota
	opIsNull
	opIsNotNull
	opIn
)

// clause is a single additional predicate on a database rule, such as the
// tenant scope in `exists:tax_receipts,id` + `company_id = ?`.
type clause struct {
	column string
	op     clauseOp
	values []any
}

// dbRuleBuilder is implemented by the fluent Exists and Unique builders. The
// validator hands these to the database rule as objects rather than flattening
// them through Constraints(), because the string encoding has no room for
// clauses that carry zero values (IS NULL) or many (IN).
type dbRuleBuilder interface {
	RuleConstraints
	ruleName() string
	table() string
	// column returns "" when the rule should fall back to the field's json key.
	column() string
	clauses() []clause
	// ignoreSpec returns the value and column to exclude, or (nil, "") when the
	// rule has no ignore semantics.
	ignoreSpec() (any, string)
}

// formatWheres encodes the equality clauses into the legacy `col__val^col__val`
// string form used by Constraints(). Clauses that cannot be expressed in that
// form (IS NULL, IN) are omitted; they only travel through the object path.
func formatWheres(wheres []clause) string {
	parts := make([]string, 0, len(wheres))
	for _, where := range wheres {
		if where.op != opEq || len(where.values) == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s__%v", where.column, where.values[0]))
	}

	return strings.Join(parts, "^")
}
