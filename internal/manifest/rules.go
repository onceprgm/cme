package manifest

import "runtime"

type Rule struct {
	Action string `json:"action"`
	OS     *struct {
		Name    string `json:"name"`
		Arch    string `json:"arch"`
		Version string `json:"version"`
	} `json:"os"`
	Features map[string]bool `json:"features"`
}

type RuleContext struct {
	OSName   string
	Arch     string
	Features map[string]bool
}

func CurrentContext() RuleContext {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	}
	return RuleContext{OSName: "linux", Arch: arch}
}

func Allowed(rules []Rule, ctx RuleContext) bool {
	if len(rules) == 0 {
		return true
	}
	allowed := false
	for _, r := range rules {
		if !r.matches(ctx) {
			continue
		}
		allowed = r.Action == "allow"
	}
	return allowed
}

func (r Rule) matches(ctx RuleContext) bool {
	if r.OS != nil {
		if r.OS.Name != "" && r.OS.Name != ctx.OSName {
			return false
		}
		if r.OS.Arch != "" && r.OS.Arch != ctx.Arch {
			return false
		}
	}
	for name, want := range r.Features {
		if ctx.Features[name] != want {
			return false
		}
	}
	return true
}
