package java

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/onceprgm/cme/internal/store"
)

var versionRe = regexp.MustCompile(`version "([^"]+)"`)

func Resolve(wantMajor int, override string) (string, error) {
	if override != "" {
		if major, ok := probe(override); ok && satisfies(major, wantMajor) {
			return override, nil
		}
		return "", fmt.Errorf("java at %s does not satisfy major %d", override, wantMajor)
	}

	managed := filepath.Join(store.JavaDir(), strconv.Itoa(wantMajor), "bin", "java")
	if major, ok := probe(managed); ok && satisfies(major, wantMajor) {
		return managed, nil
	}

	var candidates []string
	if p, err := exec.LookPath("java"); err == nil {
		candidates = append(candidates, p)
	}
	matches, _ := filepath.Glob("/usr/lib/jvm/*/bin/java")
	candidates = append(candidates, matches...)

	for _, c := range candidates {
		if major, ok := probe(c); ok && satisfies(major, wantMajor) {
			return c, nil
		}
	}

	return "", fmt.Errorf("no java %d+ found; install one or run: cme java install (not working) %d", wantMajor, wantMajor)
}

func satisfies(have, want int) bool {
	return have >= want
}

func probe(path string) (int, bool) {
	if _, err := os.Stat(path); err != nil {
		return 0, false
	}
	out, err := exec.Command(path, "-version").CombinedOutput()
	if err != nil {
		return 0, false
	}
	m := versionRe.FindSubmatch(out)
	if m == nil {
		return 0, false
	}
	return parseMajor(string(m[1])), true
}

func parseMajor(v string) int {
	v = strings.TrimPrefix(v, "1.")
	if i := strings.IndexAny(v, "._-"); i >= 0 {
		v = v[:i]
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
