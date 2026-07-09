package validator

import (
	"fmt"
	"strings"
)

func (r Rule) Unique(table string, column ...string) *Unique {
	c := "id"
	if len(column) > 0 {
		c = column[0]
	}
	u := &Unique{
		tableName:  table,
		columnName: c,
	}

	return u
}

type Unique struct {
	tableName  string
	columnName string
	ignore     any
	idColumn   string
	wheres     []clause
}

func (u *Unique) Where(column string, value any) *Unique {
	u.wheres = append(u.wheres, clause{column: column, op: opEq, values: []any{value}})
	return u
}

func (u *Unique) WhereNull(column string) *Unique {
	u.wheres = append(u.wheres, clause{column: column, op: opIsNull})
	return u
}

func (u *Unique) WhereNotNull(column string) *Unique {
	u.wheres = append(u.wheres, clause{column: column, op: opIsNotNull})
	return u
}

func (u *Unique) WhereIn(column string, values ...any) *Unique {
	u.wheres = append(u.wheres, clause{column: column, op: opIn, values: values})
	return u
}

func (u *Unique) Ignore(id any, idColumn ...string) *Unique {
	u.ignore = id
	u.idColumn = "id"

	if len(idColumn) > 0 {
		u.idColumn = idColumn[0]
	}

	return u
}

func (u Unique) Constraints() string {
	ignore := "NULL"
	if u.ignore != nil {
		ignore = fmt.Sprintf("%v", u.ignore)
	}

	return strings.TrimRight(fmt.Sprintf("unique:%s,%s,%s,%s,%s",
		u.tableName,
		u.columnName,
		addslashes(ignore),
		u.idColumn,
		formatWheres(u.wheres),
	), ",")
}

func (u Unique) ruleName() string { return "unique" }

func (u Unique) table() string { return u.tableName }

func (u Unique) column() string { return u.columnName }

func (u Unique) clauses() []clause { return u.wheres }

func (u Unique) ignoreSpec() (any, string) {
	if u.ignore == nil {
		return nil, ""
	}

	return u.ignore, u.idColumn
}

func addslashes(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\x00", "\\0")
	return s
}
