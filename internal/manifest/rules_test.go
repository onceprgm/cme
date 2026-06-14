package manifest

import "testing"

func linuxCtx() RuleContext {
	return RuleContext{OSName: "linux", Arch: "x64", OSVersion: "6.1.0"}
}

func osRule(action, name string) Rule {
	r := Rule{Action: action}
	r.OS = &struct {
		Name    string `json:"name"`
		Arch    string `json:"arch"`
		Version string `json:"version"`
	}{Name: name}
	return r
}

func TestAllowedEmptyRules(t *testing.T) {
	if !Allowed(nil, linuxCtx()) {
		t.Error("nil rules should allow")
	}
	if !Allowed([]Rule{}, linuxCtx()) {
		t.Error("empty rules should allow")
	}
}

func TestAllowedOSMatch(t *testing.T) {
	rules := []Rule{osRule("allow", "linux")}
	if !Allowed(rules, linuxCtx()) {
		t.Error("allow linux should match on linux")
	}
	rules = []Rule{osRule("allow", "osx")}
	if Allowed(rules, linuxCtx()) {
		t.Error("allow osx should not match on linux")
	}
}

func TestAllowedDisallowWins(t *testing.T) {
	rules := []Rule{
		{Action: "allow"},
		osRule("disallow", "linux"),
	}
	if Allowed(rules, linuxCtx()) {
		t.Error("disallow linux after blanket allow should deny on linux")
	}
	rules = []Rule{
		{Action: "allow"},
		osRule("disallow", "osx"),
	}
	if !Allowed(rules, linuxCtx()) {
		t.Error("disallow osx should not affect linux")
	}
}

func TestAllowedFeatures(t *testing.T) {
	rules := []Rule{{Action: "allow", Features: map[string]bool{"is_demo_user": true}}}
	if Allowed(rules, linuxCtx()) {
		t.Error("feature rule should not match when feature is absent/false")
	}
	ctx := linuxCtx()
	ctx.Features = map[string]bool{"is_demo_user": true}
	if !Allowed(rules, ctx) {
		t.Error("feature rule should match when feature is true")
	}
}

func TestAllowedOSVersion(t *testing.T) {
	r := osRule("allow", "linux")
	r.OS.Version = `^6\.`
	if !Allowed([]Rule{r}, linuxCtx()) {
		t.Error("version regex ^6\\. should match 6.1.0")
	}
	r.OS.Version = `^5\.`
	if Allowed([]Rule{r}, linuxCtx()) {
		t.Error("version regex ^5\\. should not match 6.1.0")
	}
}
