package validator

import (
	"reflect"
	"strconv"
	"strings"
)

// Validated returns the subset of input data that was covered by the rule
// set passed to the last Validate call, nested back into the original
// dot/index shape (e.g. a "contacts.*.name" rule reconstructs into
// contacts: []any{map[string]any{"name": ...}, ...}). Only fields that
// appear in the rule set and were present in the input are included — a
// field with no rule is never reflected into it, and an absent field (or a
// nil pointer) is excluded the same way `required` treats it as not
// present. Returns nil if Validate hasn't been called yet or its last run
// failed.
func (v *Validator) Validated() map[string]any {
	return v.validated
}

// recordValidated stores the value present at a validated path, to be
// nested back into shape once Validate finishes walking the rule set. A
// non-nil pointer is dereferenced to its pointed-to value; a nil pointer is
// excluded entirely.
func (v *Validator) recordValidated(key string, value reflect.Value) {
	uv := unwrapValue(value)
	if !uv.IsValid() || uv.Kind() == reflect.Ptr {
		return
	}

	if v.validatedFlat == nil {
		v.validatedFlat = make(map[string]any)
	}
	v.validatedFlat[key] = uv.Interface()
}

// buildNested reconstructs a nested map[string]any (with []any for indexed
// segments) from the flat dot/index paths recordValidated collected,
// mirroring the shape source.expand flattened them from.
func buildNested(flat map[string]any) map[string]any {
	root := map[string]any{}
	for path, value := range flat {
		insertNested(root, strings.Split(path, "."), value)
	}

	nested, _ := finalizeNested(root).(map[string]any)
	return nested
}

func insertNested(node map[string]any, segments []string, value any) {
	key := segments[0]
	if len(segments) == 1 {
		node[key] = value
		return
	}

	child, ok := node[key].(map[string]any)
	if !ok {
		child = map[string]any{}
		node[key] = child
	}
	insertNested(child, segments[1:], value)
}

// finalizeNested recurses into nested maps and converts any map whose keys
// are all-numeric (i.e. built from array indices) into an []any ordered by
// index.
func finalizeNested(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}

	for k, child := range m {
		m[k] = finalizeNested(child)
	}

	if !isIndexedMap(m) {
		return m
	}

	out := make([]any, len(m))
	for k, child := range m {
		i, _ := strconv.Atoi(k)
		if i >= 0 && i < len(out) {
			out[i] = child
		}
	}
	return out
}

func isIndexedMap(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}
	for k := range m {
		if _, err := strconv.Atoi(k); err != nil {
			return false
		}
	}
	return true
}
