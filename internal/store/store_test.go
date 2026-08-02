package store

import (
	"path/filepath"
	"testing"
)

func TestSafeJoin(t *testing.T) {
	base := filepath.Join("/data", "cme", "libraries")

	ok := []struct{ rel, want string }{
		{"net/fabricmc/loader/0.1/loader-0.1.jar", filepath.Join(base, "net/fabricmc/loader/0.1/loader-0.1.jar")},
		{"a/b/c", filepath.Join(base, "a/b/c")},
		{"/etc/passwd", filepath.Join(base, "etc/passwd")}, // leading slash is stripped, stays inside
		{"a/../b", filepath.Join(base, "b")},
	}
	for _, c := range ok {
		got, err := SafeJoin(base, c.rel)
		if err != nil {
			t.Errorf("SafeJoin(%q) unexpected error: %v", c.rel, err)
			continue
		}
		if got != c.want {
			t.Errorf("SafeJoin(%q) = %q, want %q", c.rel, got, c.want)
		}
	}

	bad := []string{
		"../evil",
		"../../etc/passwd",
		"a/../../escape",
		"..",
		"foo/../../../bar",
	}
	for _, rel := range bad {
		if got, err := SafeJoin(base, rel); err == nil {
			t.Errorf("SafeJoin(%q) = %q, want error", rel, got)
		}
	}
}
