package validator

import (
	"math"
	"reflect"
	"strconv"
	"strings"
)

// source abstracts a validation data source — either a struct or a
// map[string]any — and resolves values by Laravel-style dot paths
// (e.g. "address.line", "contacts.0.name").
type source interface {
	// get returns the value at a concrete (wildcard-free) path and whether the
	// path exists in the data.
	get(path string) (reflect.Value, bool)
	// expand turns a rule key that may contain "*" into the concrete indexed
	// paths present in the data (e.g. "contacts.*.name" -> "contacts.0.name").
	expand(key string) []string
	// messages returns custom error messages declared by the source. Structs may
	// expose them through a Messages() map[string]string method; maps have none.
	messages() map[string]string
}

func newSource(object any) source {
	val := reflect.ValueOf(object)
	for val.Kind() == reflect.Ptr {
		if val.IsNil() {
			break
		}
		val = val.Elem()
	}

	if val.Kind() == reflect.Map {
		if m, ok := val.Interface().(map[string]any); ok {
			return mapSource{data: m}
		}
	}

	return structSource{val: val}
}

// expandPath recursively replaces each "*" segment with the concrete indices
// available at its parent path, as reported by lenAt.
func expandPath(key string, lenAt func(prefix string) (int, bool)) []string {
	idx := strings.Index(key, "*")
	if idx < 0 {
		return []string{key}
	}

	before := key[:idx] // includes the trailing "."
	after := key[idx+1:]
	prefix := strings.TrimSuffix(before, ".")

	n, ok := lenAt(prefix)
	if !ok {
		return nil // no collection present -> nothing to validate
	}

	var out []string
	for i := 0; i < n; i++ {
		out = append(out, expandPath(before+strconv.Itoa(i)+after, lenAt)...)
	}
	return out
}

// coerceNumeric converts an integral float64 (how JSON numbers decode) into an
// int so integer rules such as digits/min_digits apply. Genuine decimals are
// left untouched.
func coerceNumeric(v reflect.Value) reflect.Value {
	if v.IsValid() && v.Kind() == reflect.Float64 {
		if f := v.Float(); f == math.Trunc(f) {
			return reflect.ValueOf(int(f))
		}
	}
	return v
}

func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// ---- struct source ----

type structSource struct {
	val reflect.Value
}

func (s structSource) get(path string) (reflect.Value, bool) {
	cur := s.val
	for _, p := range strings.Split(path, ".") {
		cur = deref(cur)
		if !cur.IsValid() {
			return reflect.Value{}, false
		}

		if i, err := strconv.Atoi(p); err == nil {
			if cur.Kind() != reflect.Slice && cur.Kind() != reflect.Array {
				return reflect.Value{}, false
			}
			if i < 0 || i >= cur.Len() {
				return reflect.Value{}, false
			}
			cur = cur.Index(i)
			continue
		}

		fv, ok := fieldByJSONTag(cur, p)
		if !ok {
			return reflect.Value{}, false
		}
		cur = fv
	}
	return cur, true
}

func (s structSource) expand(key string) []string {
	return expandPath(key, func(prefix string) (int, bool) {
		v, ok := s.get(prefix)
		if !ok {
			return 0, false
		}
		v = deref(v)
		if v.IsValid() && (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
			return v.Len(), true
		}
		return 0, false
	})
}

func (s structSource) messages() map[string]string {
	m := s.val.MethodByName("Messages")
	if !m.IsValid() && s.val.CanAddr() {
		m = s.val.Addr().MethodByName("Messages")
	}
	if !m.IsValid() || m.Kind() != reflect.Func {
		return nil
	}
	res := m.Call(nil)
	if len(res) == 0 {
		return nil
	}
	if msgs, ok := res[0].Interface().(map[string]string); ok {
		return msgs
	}
	return nil
}

// fieldByJSONTag finds the struct field whose json tag matches name, descending
// into embedded (anonymous) structs.
func fieldByJSONTag(v reflect.Value, name string) (reflect.Value, bool) {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			if fv, ok := fieldByJSONTag(deref(v.Field(i)), name); ok {
				return fv, true
			}
			continue
		}
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == name {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// ---- map source ----

type mapSource struct {
	data map[string]any
}

func (s mapSource) get(path string) (reflect.Value, bool) {
	var cur any = s.data
	for _, p := range strings.Split(path, ".") {
		switch c := cur.(type) {
		case map[string]any:
			v, ok := c[p]
			if !ok {
				return reflect.Value{}, false
			}
			cur = v
		case []any:
			i, err := strconv.Atoi(p)
			if err != nil || i < 0 || i >= len(c) {
				return reflect.Value{}, false
			}
			cur = c[i]
		default:
			return reflect.Value{}, false
		}
	}

	if cur == nil {
		// Present but null: treat as empty so required fails.
		return reflect.Value{}, true
	}
	return coerceNumeric(reflect.ValueOf(cur)), true
}

func (s mapSource) expand(key string) []string {
	return expandPath(key, func(prefix string) (int, bool) {
		v, ok := s.get(prefix)
		if !ok || !v.IsValid() {
			return 0, false
		}
		if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
			return v.Len(), true
		}
		return 0, false
	})
}

func (s mapSource) messages() map[string]string { return nil }
