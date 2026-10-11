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

func TestSSRURL(t *testing.T) {
	for value, want := range map[string]string{
		"":                         defaultSSRURL,
		"  ":                       defaultSSRURL,
		"http://127.0.0.1:13716":   "http://127.0.0.1:13716",
		" http://127.0.0.1:13716 ": "http://127.0.0.1:13716",
	} {
		t.Setenv("INERTIA_SSR_URL", value)
		if got := ssrURL(); got != want {
			t.Errorf("INERTIA_SSR_URL=%q: ssrURL() = %q, want %q", value, got, want)
		}
	}
}
