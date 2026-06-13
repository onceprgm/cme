package manifest

import (
	"regexp"
	"runtime"
	"syscall"
)

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
	OSName    string
	Arch      string
	OSVersion string
	Features  map[string]bool
}

func CurrentContext() RuleContext {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	}
	return RuleContext{OSName: "linux", Arch: arch, OSVersion: kernelVersion()}
}

func kernelVersion() string {
	var u syscall.Utsname
	if err := syscall.Uname(&u); err != nil {
		return ""
	}
	buf := make([]byte, 0, len(u.Release))
	for _, c := range u.Release {
		if c == 0 {
			break
		}
		buf = append(buf, byte(c))
	}
	return string(buf)
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
		if r.OS.Version != "" {
			if ctx.OSVersion == "" {
				return false
			}
			ok, err := regexp.MatchString(r.OS.Version, ctx.OSVersion)
			if err != nil || !ok {
				return false
			}
		}
	}
	for name, want := range r.Features {
		if ctx.Features[name] != want {
			return false
		}
	}
	return true
}
