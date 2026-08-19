package main

import (
	"testing"
)

func TestParseRunArgs_Simple(t *testing.T) {
	name, noInstall, quiet, extra, err := parseRunArgs([]string{"modpack"})
	if err != nil {
		t.Fatal(err)
	}
	if name != "modpack" {
		t.Fatalf("name = %q, want modpack", name)
	}
	if noInstall || quiet {
		t.Fatalf("flags = %v %v, want false", noInstall, quiet)
	}
	if len(extra) != 0 {
		t.Fatalf("extra = %v, want empty", extra)
	}
}

func TestParseRunArgs_Flags(t *testing.T) {
	name, noInstall, quiet, extra, err := parseRunArgs([]string{"modpack", "--no-install", "--no-output"})
	if err != nil {
		t.Fatal(err)
	}
	if name != "modpack" {
		t.Fatalf("name = %q, want modpack", name)
	}
	if !noInstall || !quiet {
		t.Fatalf("flags = %v %v, want true", noInstall, quiet)
	}
	if len(extra) != 0 {
		t.Fatalf("extra = %v, want empty", extra)
	}
}

func TestParseRunArgs_JVMArg(t *testing.T) {
	_, _, _, extra, err := parseRunArgs([]string{"modpack", "--jvm-arg", "-Xmx8G"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-Xmx8G"}; !slicesEqual(extra, want) {
		t.Fatalf("extra = %v, want %v", extra, want)
	}
}

func TestParseRunArgs_MultipleJVMArgs(t *testing.T) {
	_, _, _, extra, err := parseRunArgs([]string{
		"modpack",
		"--jvm-arg", "-Xmx8G",
		"--jvm-arg", "-XX:+UseZGC",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-Xmx8G", "-XX:+UseZGC"}
	if !slicesEqual(extra, want) {
		t.Fatalf("extra = %v, want %v", extra, want)
	}
}

func TestParseRunArgs_JVMArgNoValue(t *testing.T) {
	_, _, _, _, err := parseRunArgs([]string{"modpack", "--jvm-arg"})
	if err == nil {
		t.Fatal("expected usage error, got nil")
	}
	if !isUsage(err) {
		t.Fatalf("err = %v, want usageError", err)
	}
}

func TestParseRunArgs_DuplicateProfileName(t *testing.T) {
	_, _, _, _, err := parseRunArgs([]string{"modpack", "vanilla"})
	if err == nil {
		t.Fatal("expected usage error, got nil")
	}
	if !isUsage(err) {
		t.Fatalf("err = %v, want usageError", err)
	}
}

func TestParseRunArgs_Mixed(t *testing.T) {
	name, noInstall, quiet, extra, err := parseRunArgs([]string{
		"--no-install",
		"modpack",
		"--jvm-arg", "-Xmx8G",
		"--no-output",
		"--jvm-arg", "-XX:+UseZGC",
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != "modpack" {
		t.Fatalf("name = %q, want modpack", name)
	}
	if !noInstall || !quiet {
		t.Fatalf("flags = %v %v, want true", noInstall, quiet)
	}
	want := []string{"-Xmx8G", "-XX:+UseZGC"}
	if !slicesEqual(extra, want) {
		t.Fatalf("extra = %v, want %v", extra, want)
	}
}

func TestParseRunArgs_OrderPreserved(t *testing.T) {
	// Порядок важен: временные args должны идти после профильных
	// (проверяется в cmdRun, но здесь проверяем, что parseRunArgs сохраняет порядок)
	_, _, _, extra, err := parseRunArgs([]string{
		"p", "--jvm-arg", "-a", "--jvm-arg", "-b", "--jvm-arg", "-c",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-a", "-b", "-c"}
	if !slicesEqual(extra, want) {
		t.Fatalf("extra = %v, want %v", extra, want)
	}
}
