package inertia

import "testing"

func TestSSREnabled(t *testing.T) {
	cases := map[string]bool{
		"":       true,
		"true":   true,
		"1":      true,
		"false":  false,
		"FALSE":  false,
		" off ":  false,
		"0":      false,
		"no":     false,
		"banana": true,
	}
	for value, want := range cases {
		t.Setenv("INERTIA_SSR", value)
		if got := ssrEnabled(); got != want {
			t.Errorf("INERTIA_SSR=%q: ssrEnabled() = %v, want %v", value, got, want)
		}
	}
}
