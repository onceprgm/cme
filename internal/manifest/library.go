package manifest

import "strings"

type Library struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact    *LibFile           `json:"artifact"`
		Classifiers map[string]LibFile `json:"classifiers"`
	} `json:"downloads"`
	Natives map[string]string `json:"natives"`
	Rules   []Rule            `json:"rules"`
	Extract *struct {
		Exclude []string `json:"exclude"`
	} `json:"extract"`
}

type LibFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
}

func (l *Library) NativeClassifier(ctx RuleContext) (LibFile, bool) {
	key, ok := l.Natives[ctx.OSName]
	if !ok {
		return LibFile{}, false
	}
	bits := "64"
	if ctx.Arch == "x86" {
		bits = "32"
	}
	key = strings.ReplaceAll(key, "${arch}", bits)
	f, ok := l.Downloads.Classifiers[key]
	return f, ok
}

func (l *Library) ExcludePatterns() []string {
	if l.Extract == nil {
		return []string{"META-INF/"}
	}
	return l.Extract.Exclude
}

func (m *VersionMeta) ClasspathPaths(ctx RuleContext) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range m.ResolvedLibraries(ctx) {
		if l.Downloads.Artifact == nil || l.Downloads.Artifact.Path == "" {
			continue
		}
		p := l.Downloads.Artifact.Path
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
