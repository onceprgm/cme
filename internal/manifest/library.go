package manifest

import "strings"

type Library struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
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

func (l *Library) Artifact() (LibFile, bool) {
	if l.Downloads.Artifact != nil && l.Downloads.Artifact.Path != "" {
		return *l.Downloads.Artifact, true
	}
	path := mavenPath(l.Name)
	if path == "" {
		return LibFile{}, false
	}
	f := LibFile{Path: path}
	if l.URL != "" {
		f.URL = strings.TrimSuffix(l.URL, "/") + "/" + path
	}
	return f, true
}

func (l *Library) MavenKey() string {
	parts := strings.Split(l.Name, ":")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + ":" + parts[1]
}

func mavenPath(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) < 3 {
		return ""
	}
	group := strings.ReplaceAll(parts[0], ".", "/")
	artifact, version := parts[1], parts[2]
	file := artifact + "-" + version
	if len(parts) > 3 {
		file += "-" + parts[3]
	}
	return group + "/" + artifact + "/" + version + "/" + file + ".jar"
}

func (m *VersionMeta) ClasspathPaths(ctx RuleContext) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range m.ResolvedLibraries(ctx) {
		f, ok := l.Artifact()
		if !ok || f.Path == "" {
			continue
		}
		key := l.MavenKey()
		if key == "" {
			key = f.Path
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f.Path)
	}
	return out
}
