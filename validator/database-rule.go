package validator

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/playsql/grammar"
)

type DatabaseRule struct {
	key              string
	attributeValue   reflect.Value
	attributes       []string
	clauses          []clause
	db               *sql.DB
	grammar          grammar.Grammar
	ignoreGivenValue any
	ignoreColumn     string
	whereValues      []any
}

func newDatabaseRule(ctx context.Context, key string, attributes []string, value reflect.Value) *DatabaseRule {
	newBDRule := &DatabaseRule{
		key:              key,
		attributes:       attributes,
		attributeValue:   value,
		clauses:          make([]clause, 0),
		ignoreGivenValue: nil,
	}

	newBDRule.resolveConnection(ctx)
	return newBDRule
}

func (d *DatabaseRule) getCount() (int, error) {
	// compileSqlStatement populates whereValues in placeholder order, so it has
	// to run before the argument list is assembled.
	stmt := d.compileSqlStatement()

	values := []any{d.resolveValue()}
	if d.hasIgnore() {
		values = append(values, d.ignoreGivenValue)
	}
	values = append(values, d.whereValues...)

	var count int
	if err := d.db.QueryRow(stmt, values...).Scan(&count); err != nil {
		return 0, fmt.Errorf("validator: %s: %w", stmt, err)
	}

	return count, nil
}

func (d *DatabaseRule) ignore(ignore any, column string) *DatabaseRule {
	d.ignoreGivenValue = ignore
	d.ignoreColumn = column
	return d
}

func (d *DatabaseRule) hasIgnore() bool {
	return d.ignoreGivenValue != nil && d.ignoreGivenValue != "NULL"
}

func (d *DatabaseRule) resolveTableName() string {
	return d.attributes[0]
}

func (d *DatabaseRule) resolveColumnName() string {
	if len(d.attributes) < 2 || d.attributes[1] == "" {
		return d.key
	}

	return d.attributes[1]
}

func (d *DatabaseRule) resolveValue() any {
	value := unwrapValue(d.attributeValue)
	switch value.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint()
	case reflect.String:
		return value.String()
	default:
		if !value.CanInterface() {
			return nil
		}
		// Hand the raw value to the driver; types such as uuid.UUID implement
		// driver.Valuer and know how to encode themselves.
		return value.Interface()
	}
}

func (d *DatabaseRule) compileSqlStatement() string {
	d.whereValues = make([]any, 0, len(d.clauses))
	placeholder := 2

	var stmt strings.Builder
	fmt.Fprintf(&stmt, "select count(*) from %s where %s = %s",
		d.resolveTableName(),
		d.resolveColumnName(),
		d.grammar.Placeholder(1),
	)

	if d.hasIgnore() {
		fmt.Fprintf(&stmt, " and %s <> %s", d.ignoreColumn, d.grammar.Placeholder(placeholder))
		placeholder++
	}

	for _, where := range d.clauses {
		switch where.op {
		case opIsNull:
			fmt.Fprintf(&stmt, " and %s is null", where.column)
		case opIsNotNull:
			fmt.Fprintf(&stmt, " and %s is not null", where.column)
		case opIn:
			// `in ()` is not valid SQL; an empty set matches nothing.
			if len(where.values) == 0 {
				stmt.WriteString(" and false")
				continue
			}

			placeholders := make([]string, 0, len(where.values))
			for _, value := range where.values {
				placeholders = append(placeholders, d.grammar.Placeholder(placeholder))
				d.whereValues = append(d.whereValues, value)
				placeholder++
			}
			fmt.Fprintf(&stmt, " and %s in (%s)", where.column, strings.Join(placeholders, ", "))
		default:
			if len(where.values) == 0 {
				continue
			}

			fmt.Fprintf(&stmt, " and %s = %s", where.column, d.grammar.Placeholder(placeholder))
			d.whereValues = append(d.whereValues, where.values[0])
			placeholder++
		}
	}

	return stmt.String()
}

func (d *DatabaseRule) resolveConnection(ctx context.Context) {
	d.db = ctx.Value(database.ConnectionKey{}).(*sql.DB)

	if d.db == nil {
		panic("database connection need to be set.")
	}

	dialect, _ := ctx.Value(database.DialectKey{}).(string)
	if dialect == "" {
		dialect = "postgres"
	}
	d.grammar = grammar.For(dialect)
	if d.grammar == nil {
		panic(fmt.Sprintf("validator: unsupported dialect %q", dialect))
	}
}

// addWheres decodes the `col__val^col__val` form carried by raw rule strings.
// Clauses built through the fluent Exists/Unique builders arrive via addClauses
// instead and never pass through this encoding.
func (d *DatabaseRule) addWheres(wheres [][]string) *DatabaseRule {
	for _, group := range wheres {
		for _, encoded := range group {
			for _, token := range strings.Split(encoded, "^") {
				parts := strings.SplitN(token, "__", 2)
				if len(parts) != 2 {
					continue
				}

				d.clauses = append(d.clauses, clause{
					column: parts[0],
					op:     opEq,
					values: []any{parts[1]},
				})
			}
		}
	}

	return d
}

func (d *DatabaseRule) addClauses(clauses []clause) *DatabaseRule {
	d.clauses = append(d.clauses, clauses...)
	return d
}
