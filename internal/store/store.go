package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const appName = "cme"

func xdg(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return filepath.Join(v, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", appName)
	}
	return filepath.Join(home, fallback, appName)
}

func ConfigDir() string {
	return xdg("XDG_CONFIG_HOME", ".config")
}

func DataDir() string {
	return xdg("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

func CacheDir() string {
	return xdg("XDG_CACHE_HOME", ".cache")
}

func StateDir() string {
	return xdg("XDG_STATE_HOME", filepath.Join(".local", "state"))
}

func VersionsDir() string {
	return filepath.Join(DataDir(), "versions")
}
func LibrariesDir() string {
	return filepath.Join(DataDir(), "libraries")
}
func AssetsDir() string {
	return filepath.Join(DataDir(), "assets")
}
func JavaDir() string {
	return filepath.Join(DataDir(), "java")
}
func InstancesDir() string {
	return filepath.Join(DataDir(), "instances")
}

func VersionDir(id string) string {
	return filepath.Join(VersionsDir(), id)
}

func Ensure(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func SafeJoin(base, rel string) (string, error) {
	full := filepath.Join(base, filepath.FromSlash(rel))
	within, err := filepath.Rel(base, full)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe path %q escapes %q", rel, base)
	}
	return full, nil
}
