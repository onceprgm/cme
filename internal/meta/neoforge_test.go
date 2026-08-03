package meta

import "testing"

func TestNeoForgePrefix(t *testing.T) {
	ok := map[string]string{
		"1.21.1": "21.1.",
		"1.21":   "21.0.",
		"1.20.4": "20.4.",
		"1.20.2": "20.2.",
	}
	for mc, want := range ok {
		got, err := NeoForgePrefix(mc)
		if err != nil {
			t.Errorf("NeoForgePrefix(%q) error: %v", mc, err)
			continue
		}
		if got != want {
			t.Errorf("NeoForgePrefix(%q) = %q, want %q", mc, got, want)
		}
	}

	for _, bad := range []string{"2.0", "1", "", "21.1"} {
		if _, err := NeoForgePrefix(bad); err == nil {
			t.Errorf("NeoForgePrefix(%q) expected error", bad)
		}
	}
}
