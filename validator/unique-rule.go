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
		table:  table,
		column: c,
	}

	return u
}

type Unique struct {
	table    string
	column   string
	ignore   any
	idColumn string
	wheres   map[string]any
}

func (u *Unique) Where(column string, value any) *Unique {
	if u.wheres == nil {
		u.wheres = make(map[string]any)
	}
	u.wheres[column] = value
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
		u.table,
		u.column,
		addslashes(ignore),
		u.idColumn,
		u.formatWheres(),
	), ",")
}

func (u Unique) formatWheres() string {
	wheres := ""
	for c, v := range u.wheres {
		wheres += fmt.Sprintf("%s__%v^", c, v)
	}
	return strings.TrimRight(wheres, "^")
}

func addslashes(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\x00", "\\0")
	return s
}
