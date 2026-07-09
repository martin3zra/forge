package validator

import (
	"reflect"
	"testing"
)

// newTestRule builds a DatabaseRule without touching resolveConnection, so the
// SQL compiler can be exercised without a database.
func newTestRule(key string, attributes []string, value any) *DatabaseRule {
	return &DatabaseRule{
		key:            key,
		attributes:     attributes,
		attributeValue: reflect.ValueOf(value),
		clauses:        make([]clause, 0),
	}
}

func fromBuilder(key string, builder dbRuleBuilder, value any) *DatabaseRule {
	rule := newTestRule(key, []string{builder.table(), builder.column()}, value)
	if ignoreValue, ignoreColumn := builder.ignoreSpec(); ignoreValue != nil {
		rule.ignore(ignoreValue, ignoreColumn)
	}

	return rule.addClauses(builder.clauses())
}

func TestCompileSqlStatement(t *testing.T) {
	tests := []struct {
		name        string
		rule        *DatabaseRule
		stmt        string
		whereValues []any
	}{
		{
			name: "exists without clauses",
			rule: fromBuilder("id", Rule{}.Exists("t", "id"), 1),
			stmt: "select count(*) from t where id = $1",
		},
		{
			name:        "exists scoped to a tenant",
			rule:        fromBuilder("id", Rule{}.Exists("t", "id").Where("company_id", 7), 1),
			stmt:        "select count(*) from t where id = $1 and company_id = $2",
			whereValues: []any{7},
		},
		{
			name: "exists with a null clause",
			rule: fromBuilder("id", Rule{}.Exists("t", "id").WhereNull("deleted_at"), 1),
			stmt: "select count(*) from t where id = $1 and deleted_at is null",
		},
		{
			name: "exists with a not null clause",
			rule: fromBuilder("id", Rule{}.Exists("t", "id").WhereNotNull("approved_at"), 1),
			stmt: "select count(*) from t where id = $1 and approved_at is not null",
		},
		{
			name:        "exists with an in clause",
			rule:        fromBuilder("id", Rule{}.Exists("t", "id").WhereIn("status", "a", "b", "c"), 1),
			stmt:        "select count(*) from t where id = $1 and status in ($2, $3, $4)",
			whereValues: []any{"a", "b", "c"},
		},
		{
			name: "exists with an empty in clause matches nothing",
			rule: fromBuilder("id", Rule{}.Exists("t", "id").WhereIn("status"), 1),
			stmt: "select count(*) from t where id = $1 and false",
		},
		{
			name:        "exists mixing every clause kind keeps placeholder order",
			rule:        fromBuilder("id", Rule{}.Exists("t", "id").Where("company_id", 7).WhereNull("deleted_at").WhereIn("status", "a", "b"), 1),
			stmt:        "select count(*) from t where id = $1 and company_id = $2 and deleted_at is null and status in ($3, $4)",
			whereValues: []any{7, "a", "b"},
		},
		{
			name: "exists falls back to the field key when no column is given",
			rule: fromBuilder("email", Rule{}.Exists("users"), "a@b.c"),
			stmt: "select count(*) from users where email = $1",
		},
		{
			name:        "unique with ignore and a where clause",
			rule:        fromBuilder("email", Rule{}.Unique("t", "email").Ignore(5, "id").Where("company_id", 7), "a@b.c"),
			stmt:        "select count(*) from t where email = $1 and id <> $2 and company_id = $3",
			whereValues: []any{7},
		},
		{
			name: "unique without ignore",
			rule: fromBuilder("email", Rule{}.Unique("t", "email"), "a@b.c"),
			stmt: "select count(*) from t where email = $1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stmt := test.rule.compileSqlStatement()
			if stmt != test.stmt {
				t.Errorf("stmt\n got: %s\nwant: %s", stmt, test.stmt)
			}

			want := test.whereValues
			if want == nil {
				want = []any{}
			}
			if !reflect.DeepEqual(test.rule.whereValues, want) {
				t.Errorf("whereValues: got %v, want %v", test.rule.whereValues, want)
			}
		})
	}
}

// The raw string rules must compile to the same SQL as their builder
// equivalents, so existing rule strings keep working.
func TestLegacyStringPathMatchesBuilderPath(t *testing.T) {
	tests := []struct {
		name    string
		legacy  *DatabaseRule
		builder *DatabaseRule
	}{
		{
			name: "exists with a where clause",
			legacy: func() *DatabaseRule {
				rule := newTestRule("id", []string{"t", "id", "company_id__7"}, 1)
				return rule.addWheres(splitWheres([]string{"company_id__7"}))
			}(),
			builder: fromBuilder("id", Rule{}.Exists("t", "id").Where("company_id", "7"), 1),
		},
		{
			name: "unique with ignore and two where clauses",
			legacy: func() *DatabaseRule {
				rule := newTestRule("email", []string{"t", "email", "5", "id", "a__1^b__2"}, "a@b.c")
				rule.ignore("5", "id")
				return rule.addWheres(splitWheres([]string{"a__1^b__2"}))
			}(),
			builder: fromBuilder("email", Rule{}.Unique("t", "email").Ignore("5", "id").Where("a", "1").Where("b", "2"), "a@b.c"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			legacy := test.legacy.compileSqlStatement()
			builder := test.builder.compileSqlStatement()
			if legacy != builder {
				t.Errorf("legacy: %s\nbuilder: %s", legacy, builder)
			}

			if !reflect.DeepEqual(test.legacy.whereValues, test.builder.whereValues) {
				t.Errorf("whereValues: legacy %v, builder %v", test.legacy.whereValues, test.builder.whereValues)
			}
		})
	}
}

func TestConstraints(t *testing.T) {
	tests := []struct {
		name string
		rule RuleConstraints
		want string
	}{
		{
			name: "exists without clauses",
			rule: Rule{}.Exists("t", "id"),
			want: "exists:t,id",
		},
		{
			name: "exists without a column",
			rule: Rule{}.Exists("t"),
			want: "exists:t",
		},
		{
			name: "exists encodes equality clauses in order",
			rule: Rule{}.Exists("t", "id").Where("company_id", 7).Where("branch_id", 2),
			want: "exists:t,id,company_id__7^branch_id__2",
		},
		{
			name: "exists omits clauses the string form cannot express",
			rule: Rule{}.Exists("t", "id").WhereNull("deleted_at"),
			want: "exists:t,id",
		},
		{
			name: "unique keeps its historical shape",
			rule: Rule{}.Unique("t", "email").Ignore(5, "id").Where("company_id", 7),
			want: "unique:t,email,5,id,company_id__7",
		},
		{
			name: "unique without ignore",
			rule: Rule{}.Unique("t", "email"),
			want: "unique:t,email,NULL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.rule.Constraints(); got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveValue(t *testing.T) {
	id := 42
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "int", value: 7, want: int64(7)},
		{name: "int64", value: int64(7), want: int64(7)},
		{name: "uint", value: uint(7), want: uint64(7)},
		{name: "string", value: "a@b.c", want: "a@b.c"},
		{name: "pointer is unwrapped", value: &id, want: int64(42)},
		{name: "unknown kinds reach the driver", value: [2]byte{1, 2}, want: [2]byte{1, 2}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rule := newTestRule("id", []string{"t", "id"}, test.value)
			if got := rule.resolveValue(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %#v, want %#v", got, test.want)
			}
		})
	}
}
