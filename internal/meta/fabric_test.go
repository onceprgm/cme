package meta

import (
	"encoding/json"
	"testing"
)

func TestSelectLoaderPrefersStable(t *testing.T) {
	raw := `[
		{"loader":{"version":"0.17.0-beta.1","stable":false}},
		{"loader":{"version":"0.16.9","stable":true}},
		{"loader":{"version":"0.16.8","stable":true}}
	]`
	var list []loaderEntry
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatal(err)
	}
	got, ok := selectLoader(list)
	if !ok || got != "0.16.9" {
		t.Errorf("selectLoader = %q, %v; want 0.16.9, true", got, ok)
	}
}

func TestSelectLoaderFallsBackToNewest(t *testing.T) {
	list := []loaderEntry{}
	list = append(list, loaderEntry{})
	list[0].Loader.Version = "0.17.0-beta.1"
	got, ok := selectLoader(list)
	if !ok || got != "0.17.0-beta.1" {
		t.Errorf("selectLoader = %q, %v; want newest fallback", got, ok)
	}
}

func TestSelectLoaderEmpty(t *testing.T) {
	if _, ok := selectLoader(nil); ok {
		t.Error("expected no loader for empty list")
	}
}
