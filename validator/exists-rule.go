package validator

import (
	"fmt"
	"strings"
)

// Exists starts an `exists` rule. When column is omitted the field's json key
// is used, matching the `exists:users` shorthand.
func (r Rule) Exists(table string, column ...string) *Exists {
	e := &Exists{tableName: table}
	if len(column) > 0 {
		e.columnName = column[0]
	}

	return e
}

type Exists struct {
	tableName  string
	columnName string
	wheres     []clause
}

// Where scopes the lookup to rows matching column = value. Use it to bind a
// row to the tenant that owns it.
func (e *Exists) Where(column string, value any) *Exists {
	e.wheres = append(e.wheres, clause{column: column, op: opEq, values: []any{value}})
	return e
}

func (e *Exists) WhereNull(column string) *Exists {
	e.wheres = append(e.wheres, clause{column: column, op: opIsNull})
	return e
}

func (e *Exists) WhereNotNull(column string) *Exists {
	e.wheres = append(e.wheres, clause{column: column, op: opIsNotNull})
	return e
}

func (e *Exists) WhereIn(column string, values ...any) *Exists {
	e.wheres = append(e.wheres, clause{column: column, op: opIn, values: values})
	return e
}

func (e Exists) Constraints() string {
	return strings.TrimRight(fmt.Sprintf("exists:%s,%s,%s",
		e.tableName,
		e.columnName,
		formatWheres(e.wheres),
	), ",")
}

func (e Exists) ruleName() string { return "exists" }

func (e Exists) table() string { return e.tableName }

func (e Exists) column() string { return e.columnName }

func (e Exists) clauses() []clause { return e.wheres }

func (e Exists) ignoreSpec() (any, string) { return nil, "" }
