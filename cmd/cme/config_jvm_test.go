package main

import (
	"testing"

	"github.com/onceprgm/cme/internal/config"
)

func TestConfigAddJVMArg_Appends(t *testing.T) {
	isolateCLI(t)
	if err := configAddJVMArg("-XX:+UseZGC"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-XX:+UseZGC"}; !slicesEqual(cfg.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", cfg.JVMArgs, want)
	}
}

func TestConfigAddJVMArg_Duplicate(t *testing.T) {
	isolateCLI(t)
	if err := configAddJVMArg("-XX:+UseZGC"); err != nil {
		t.Fatal(err)
	}
	if err := configAddJVMArg("-XX:+UseZGC"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.JVMArgs) != 1 {
		t.Fatalf("len(JVMArgs) = %d, want 1 (not deduped)", len(cfg.JVMArgs))
	}
}

func TestConfigAddJVMArg_Blank(t *testing.T) {
	isolateCLI(t)
	err := configAddJVMArg("   ")
	if err == nil || !isUsage(err) {
		t.Fatalf("blank arg err = %v, want usage error", err)
	}
}

func TestConfigRmJVMArg_Exists(t *testing.T) {
	isolateCLI(t)
	if err := configAddJVMArg("-a"); err != nil {
		t.Fatal(err)
	}
	if err := configAddJVMArg("-b"); err != nil {
		t.Fatal(err)
	}
	if err := configRmJVMArg("-a"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-b"}; !slicesEqual(cfg.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", cfg.JVMArgs, want)
	}
}

func TestConfigRmJVMArg_Missing(t *testing.T) {
	isolateCLI(t)
	if err := configAddJVMArg("-a"); err != nil {
		t.Fatal(err)
	}
	err := configRmJVMArg("-zzz")
	if err == nil {
		t.Fatal("rm missing = nil, want error")
	}
	if isUsage(err) {
		t.Fatalf("rm missing should not be usage error, got %v", err)
	}
}

func TestConfigAddJVMArg_BadUsage(t *testing.T) {
	isolateCLI(t)
	// Вызываем cmdConfig напрямую, чтобы проверить usage
	err := cmdConfig([]string{"add-jvm-arg"})
	if err == nil || !isUsage(err) {
		t.Fatalf("add-jvm-arg without arg err = %v, want usage error", err)
	}
}

func TestConfigRmJVMArg_BadUsage(t *testing.T) {
	isolateCLI(t)
	err := cmdConfig([]string{"rm-jvm-arg"})
	if err == nil || !isUsage(err) {
		t.Fatalf("rm-jvm-arg without arg err = %v, want usage error", err)
	}
}
