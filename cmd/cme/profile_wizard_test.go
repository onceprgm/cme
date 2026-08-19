package main

import (
	"strings"
	"testing"

	"github.com/onceprgm/cme/internal/config"
	"github.com/onceprgm/cme/internal/ui"
)

func TestProfileWizard_JVMArgs(t *testing.T) {
	cfg := &config.Config{Username: "Steve", RAM: 4}

	// Ответы: loader, version, loaderVer, username, ram, jvm1, jvm2, jvm-end
	input := "fabric\n1.21.4\n\nAlex\n8\n-XX:+UseZGC\n-XX:+PerfDisableSharedMem\n\n"
	ui.SetStdin(strings.NewReader(input))
	defer ui.SetStdin(nil)

	loader, version, loaderVer, username, ram, jvmArgs := profileWizard(cfg)

	if loader != "fabric" {
		t.Fatalf("loader = %q, want fabric", loader)
	}
	if version != "1.21.4" {
		t.Fatalf("version = %q, want 1.21.4", version)
	}
	if loaderVer != "" {
		t.Fatalf("loaderVer = %q, want empty", loaderVer)
	}
	if username != "Alex" {
		t.Fatalf("username = %q, want Alex", username)
	}
	if ram != 8 {
		t.Fatalf("ram = %d, want 8", ram)
	}
	want := []string{"-XX:+UseZGC", "-XX:+PerfDisableSharedMem"}
	if !slicesEqual(jvmArgs, want) {
		t.Fatalf("jvmArgs = %v, want %v", jvmArgs, want)
	}
}

func TestProfileWizard_NoJVMArgs(t *testing.T) {
	cfg := &config.Config{Username: "Steve", RAM: 4}

	input := "vanilla\n1.20.1\n\n\n\n"
	ui.SetStdin(strings.NewReader(input))
	defer ui.SetStdin(nil)

	_, _, _, _, _, jvmArgs := profileWizard(cfg)

	if len(jvmArgs) != 0 {
		t.Fatalf("jvmArgs = %v, want empty", jvmArgs)
	}
}

func TestProfileWizard_AbortBeforeJVM(t *testing.T) {
	cfg := &config.Config{}

	input := "vanilla\n\n"
	ui.SetStdin(strings.NewReader(input))
	defer ui.SetStdin(nil)

	_, version, _, _, _, jvmArgs := profileWizard(cfg)

	if version != "" {
		t.Fatalf("version = %q, want empty (abort)", version)
	}
	if len(jvmArgs) != 0 {
		t.Fatalf("jvmArgs = %v, want empty (wizard aborted)", jvmArgs)
	}
}
