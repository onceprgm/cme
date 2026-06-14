package manifest

import (
	"encoding/json"
	"testing"
)

func TestArgumentUnmarshalString(t *testing.T) {
	var a Argument
	if err := json.Unmarshal([]byte(`"--demo"`), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Values) != 1 || a.Values[0] != "--demo" {
		t.Errorf("Values = %v, want [--demo]", a.Values)
	}
	if len(a.Rules) != 0 {
		t.Errorf("Rules = %v, want empty", a.Rules)
	}
}

func TestArgumentUnmarshalObjectSingleValue(t *testing.T) {
	data := `{"rules":[{"action":"allow","os":{"name":"linux"}}],"value":"-Dfoo=bar"}`
	var a Argument
	if err := json.Unmarshal([]byte(data), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Values) != 1 || a.Values[0] != "-Dfoo=bar" {
		t.Errorf("Values = %v, want [-Dfoo=bar]", a.Values)
	}
	if len(a.Rules) != 1 {
		t.Errorf("Rules len = %d, want 1", len(a.Rules))
	}
}

func TestArgumentUnmarshalObjectArrayValue(t *testing.T) {
	data := `{"rules":[{"action":"allow"}],"value":["-x","-y"]}`
	var a Argument
	if err := json.Unmarshal([]byte(data), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Values) != 2 || a.Values[0] != "-x" || a.Values[1] != "-y" {
		t.Errorf("Values = %v, want [-x -y]", a.Values)
	}
}

func TestSubstitute(t *testing.T) {
	vars := map[string]string{"name": "Steve", "dir": "/home/x"}
	if got := substitute("hello ${name}", vars); got != "hello Steve" {
		t.Errorf("got %q", got)
	}
	if got := substitute("${dir}/saves", vars); got != "/home/x/saves" {
		t.Errorf("got %q", got)
	}
	if got := substitute("no placeholders", vars); got != "no placeholders" {
		t.Errorf("got %q", got)
	}
}

func TestGameArgsLegacyString(t *testing.T) {
	m := &VersionMeta{MinecraftArguments: "--username ${auth_player_name} --version ${version_name}"}
	vars := map[string]string{"auth_player_name": "Steve", "version_name": "1.8.9"}
	got := m.GameArgs(linuxCtx(), vars)
	want := []string{"--username", "Steve", "--version", "1.8.9"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGameArgsModern(t *testing.T) {
	m := &VersionMeta{Arguments: &Arguments{
		Game: []Argument{
			{Values: []string{"--username"}},
			{Values: []string{"${auth_player_name}"}},
			{Rules: []Rule{osRule("allow", "osx")}, Values: []string{"--mac-only"}},
		},
	}}
	vars := map[string]string{"auth_player_name": "Steve"}
	got := m.GameArgs(linuxCtx(), vars)
	want := []string{"--username", "Steve"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v (mac-only arg should be filtered out)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestJVMArgsLegacyFallback(t *testing.T) {
	m := &VersionMeta{}
	vars := map[string]string{"natives_directory": "/n", "classpath": "/cp"}
	got := m.JVMArgs(linuxCtx(), vars)
	want := []string{"-Djava.library.path=/n", "-cp", "/cp"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
