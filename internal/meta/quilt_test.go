package meta

import "testing"

func loaders(versions ...string) []loaderEntry {
	list := make([]loaderEntry, len(versions))
	for i, v := range versions {
		list[i].Loader.Version = v
	}
	return list
}

func TestSelectNewestStable(t *testing.T) {
	// unordered list with betas mixed in; must pick the highest non-pre-release
	list := loaders("0.20.0-beta.9", "0.24.0", "0.30.0", "0.23.1", "0.31.0-beta.1", "0.19.5")
	got, ok := selectNewestStable(list)
	if !ok || got != "0.30.0" {
		t.Errorf("selectNewestStable = %q, %v; want 0.30.0, true", got, ok)
	}
}

func TestSelectNewestStableAllBeta(t *testing.T) {
	list := loaders("0.20.0-beta.9", "0.20.0-beta.1")
	got, ok := selectNewestStable(list)
	if !ok || got != "0.20.0-beta.9" {
		t.Errorf("selectNewestStable = %q, %v; want first as fallback", got, ok)
	}
}

func TestSelectNewestStableEmpty(t *testing.T) {
	if _, ok := selectNewestStable(nil); ok {
		t.Error("expected no loader for empty list")
	}
}

func TestVersionLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.24.0", "0.30.0", true},
		{"0.30.0", "0.24.0", false},
		{"0.23.0", "0.23.1", true},
		{"0.9.0", "0.10.0", true}, // numeric, not lexical
		{"0.30.0", "0.30.0", false},
		{"1.0", "1.0.0", false},
	}
	for _, c := range cases {
		if got := versionLess(c.a, c.b); got != c.want {
			t.Errorf("versionLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
