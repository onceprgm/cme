package manifest

import (
	"bytes"
	"encoding/json"
	"strings"
)

type Argument struct {
	Rules  []Rule
	Values []string
}

func (a *Argument) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		a.Values = []string{s}
		return nil
	}

	var obj struct {
		Rules []Rule          `json:"rules"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	a.Rules = obj.Rules

	trimmed := bytes.TrimSpace(obj.Value)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		return json.Unmarshal(trimmed, &a.Values)
	}
	var s string
	if err := json.Unmarshal(trimmed, &s); err != nil {
		return err
	}
	a.Values = []string{s}
	return nil
}

type Arguments struct {
	Game []Argument `json:"game"`
	JVM  []Argument `json:"jvm"`
}

func resolveArgs(args []Argument, ctx RuleContext, vars map[string]string) []string {
	var out []string
	for _, a := range args {
		if !Allowed(a.Rules, ctx) {
			continue
		}
		for _, v := range a.Values {
			out = append(out, substitute(v, vars))
		}
	}
	return out
}

func substitute(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

func (m *VersionMeta) GameArgs(ctx RuleContext, vars map[string]string) []string {
	if m.Arguments != nil {
		return resolveArgs(m.Arguments.Game, ctx, vars)
	}
	var out []string
	for _, tok := range strings.Fields(m.MinecraftArguments) {
		out = append(out, substitute(tok, vars))
	}
	return out
}

func (m *VersionMeta) JVMArgs(ctx RuleContext, vars map[string]string) []string {
	if m.Arguments != nil && len(m.Arguments.JVM) > 0 {
		return resolveArgs(m.Arguments.JVM, ctx, vars)
	}
	return []string{
		"-Djava.library.path=" + vars["natives_directory"],
		"-cp", vars["classpath"],
	}
}
