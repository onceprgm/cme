package profile

import "testing"

func TestValidName(t *testing.T) {
	good := []string{"modpack", "vanilla19", "my.pack", "a_b-c", "1.21"}
	for _, n := range good {
		if !ValidName(n) {
			t.Errorf("ValidName(%q) = false, want true", n)
		}
	}
	bad := []string{"", ".", "..", "../evil", "a/b", "a\\b", "sub/../x"}
	for _, n := range bad {
		if ValidName(n) {
			t.Errorf("ValidName(%q) = true, want false", n)
		}
	}
}
