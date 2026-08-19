package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onceprgm/cme/internal/profile"
)

func isolateCLI(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, ".cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".local", "state"))
}

func seed(t *testing.T, name string, args []string) {
	t.Helper()
	p := &profile.Profile{Name: name, Version: "1.20.1"}
	if len(args) > 0 {
		p.SetJVMArgs(args)
	}
	if err := profile.Save(p); err != nil {
		t.Fatal(err)
	}
}

func mustGet(t *testing.T, name string) *profile.Profile {
	t.Helper()
	p, err := profile.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	w.Close()
	os.Stdout = old
	return <-done
}

func TestProfileJVMAdd_Appends(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", nil)
	if err := profileJVM([]string{"add", "p1", "-XX:+UseZGC"}); err != nil {
		t.Fatal(err)
	}
	p := mustGet(t, "p1")
	if want := []string{"-XX:+UseZGC"}; !slicesEqual(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestProfileJVMAdd_Duplicate(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-XX:+UseZGC"})
	if err := profileJVM([]string{"add", "p1", "-XX:+UseZGC"}); err != nil {
		t.Fatal(err)
	}
	p := mustGet(t, "p1")
	if len(p.JVMArgs) != 1 {
		t.Fatalf("len(JVMArgs) = %d, want 1 (not deduped)", len(p.JVMArgs))
	}
}

func TestProfileJVMAdd_BadUsage(t *testing.T) {
	isolateCLI(t)
	for _, args := range [][]string{
		{"add"},
		{"add", "p1"},
		{"add", "p1", "-a", "-b"},
	} {
		err := profileJVM(args)
		if err == nil {
			t.Fatalf("profileJVM(%v) = nil, want usage error", args)
		}
		if !isUsage(err) {
			t.Fatalf("profileJVM(%v) err = %v, want usageError", args, err)
		}
	}
}

func TestProfileJVMAdd_BlankRejected(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", nil)
	err := profileJVM([]string{"add", "p1", "   "})
	if err == nil || !isUsage(err) {
		t.Fatalf("blank arg err = %v, want usage error", err)
	}
}

func TestProfileJVMRm_Exists(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-a", "-b", "-c"})
	if err := profileJVM([]string{"rm", "p1", "-b"}); err != nil {
		t.Fatal(err)
	}
	p := mustGet(t, "p1")
	if want := []string{"-a", "-c"}; !slicesEqual(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestProfileJVMRm_Missing(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-a"})
	err := profileJVM([]string{"rm", "p1", "-zzz"})
	if err == nil {
		t.Fatal("rm missing = nil, want error")
	}
	if isUsage(err) {
		t.Fatalf("rm missing should not be usage error, got %v", err)
	}
}

func TestProfileJVMSet_Replaces(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-old"})
	if err := profileJVM([]string{"set", "p1", "-x", "-y"}); err != nil {
		t.Fatal(err)
	}
	p := mustGet(t, "p1")
	if want := []string{"-x", "-y"}; !slicesEqual(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestProfileJVMSet_Clear(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-a", "-b"})
	if err := profileJVM([]string{"set", "p1"}); err != nil {
		t.Fatal(err)
	}
	p := mustGet(t, "p1")
	if len(p.JVMArgs) != 0 {
		t.Fatalf("len(JVMArgs) = %d, want 0", len(p.JVMArgs))
	}
}

func TestProfileJVMList_Plain(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-a", "-b", "-c"})
	out := captureStdout(t, func() {
		if err := profileJVM([]string{"list", "p1"}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if want := []string{"-a", "-b", "-c"}; !slicesEqual(lines, want) {
		t.Fatalf("lines = %v, want %v", lines, want)
	}
}

func TestProfileJVMList_JSON(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", []string{"-a", "-b"})
	out := captureStdout(t, func() {
		if err := profileJVM([]string{"list", "p1", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	var got []string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON %q: %v", out, err)
	}
	if want := []string{"-a", "-b"}; !slicesEqual(got, want) {
		t.Fatalf("JSON = %v, want %v", got, want)
	}
}

func TestProfileJVMList_JSON_EmptyArray(t *testing.T) {
	isolateCLI(t)
	seed(t, "p1", nil)
	out := captureStdout(t, func() {
		if err := profileJVM([]string{"list", "p1", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	// must be "[]" not "null" so jq/clients don't choke
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" {
		t.Fatalf("empty list JSON = %q, want %q", trimmed, "[]")
	}
}

func TestProfileJVM_UnknownSub(t *testing.T) {
	isolateCLI(t)
	err := profileJVM([]string{"wat", "p1"})
	if err == nil || !isUsage(err) {
		t.Fatalf("unknown sub err = %v, want usageError", err)
	}
}

// helpers

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isUsage(err error) bool {
	_, ok := err.(*usageError)
	return ok
}
