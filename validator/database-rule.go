package validator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/martin3zra/forge/database"
)

type DatabaseRule struct {
	key              string
	attributeValue   reflect.Value
	attributes       []string
	wheres           []string
	db               *sql.DB
	ignoreGivenValue any
	ignoreColumn     string
	whereValues      []any
}

func newDatabaseRule(ctx context.Context, key string, attributes []string, value reflect.Value) *DatabaseRule {
	newBDRule := &DatabaseRule{
		key:              key,
		attributes:       attributes,
		attributeValue:   value,
		wheres:           make([]string, 0),
		ignoreGivenValue: nil,
	}

	newBDRule.resolveConnection(ctx)
	return newBDRule
}

func (d *DatabaseRule) getCount() int {

	stmt := d.compileSqlStatement()
	var count int64
	var values = []any{d.resolveValue()}
	if d.ignoreGivenValue != "NULL" && d.ignoreGivenValue != nil {
		values = append(values, d.ignoreGivenValue)
	}
	values = append(values, d.resolveWhereValues()...)

	err := d.db.QueryRow(stmt, values...).Scan(&count)
	if err != nil {
		fmt.Printf("an error has occurred: %v\n%s\n%v\n", err, stmt, values)
		return 0
	}

	return int(count)
}

func (d *DatabaseRule) ignore(ignore any, column string) *DatabaseRule {
	d.ignoreGivenValue = ignore
	d.ignoreColumn = column
	return d
}

func (d *DatabaseRule) resolveTableName() string {
	return d.attributes[0]
}

func (d *DatabaseRule) resolveColumnName() string {
	if len(d.attributes) == 1 {
		return d.key
	}

	return d.attributes[1]
}

func (d *DatabaseRule) resolveValue() any {
	switch d.attributeValue.Kind() {
	case reflect.Int:
		return d.attributeValue.Int()
	case reflect.String:
		return d.attributeValue.String()
	default:
		log.Fatalf("not accepted type: %v", d.attributeValue)
		return ""
	}
}

func (d *DatabaseRule) resolveWhereValues() []any {
	if d.whereValues == nil {
		return make([]any, 0)
	}
	return d.whereValues
}

func (d *DatabaseRule) compileSqlStatement() string {
	d.whereValues = make([]any, 0)
	placeholder := 2

	if d.ignoreGivenValue != "NULL" && d.ignoreGivenValue != nil {
		placeholder = 3
	}
	var stmt = " AND"
	for _, whereValue := range d.wheres {
		whereComponents := strings.Split(whereValue, "^")
		for i, where := range whereComponents {
			parts := strings.Split(where, "__")
			stmt += fmt.Sprintf(" %s = $%d AND", parts[0], placeholder+i)

			d.whereValues = append(d.whereValues, parts[1])
		}
	}

	return fmt.Sprintf(
		"select count(*) from %s where %s = $1 %s%s",
		d.resolveTableName(),
		d.resolveColumnName(),
		d.criteriaIfIgnoringGivenValue(),
		strings.TrimRight(stmt, " AND"),
	)
}

func (d *DatabaseRule) criteriaIfIgnoringGivenValue() string {
	if d.ignoreGivenValue != "NULL" && d.ignoreGivenValue != nil {
		return fmt.Sprintf("AND %s <> $2", d.ignoreColumn)
	}

	return ""
}

func (d *DatabaseRule) resolveConnection(ctx context.Context) {
	d.db = ctx.Value(database.ConnectionKey{}).(*sql.DB)

	if d.db == nil {
		panic("database connection need to be set.")
	}
}

func (d *DatabaseRule) addWheres(wheres [][]string) *DatabaseRule {
	for _, where := range wheres {
		d.wheres = append(d.wheres, strings.Join(where, " = "))
	}

	return d
}
