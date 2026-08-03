package installer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/download"
	"github.com/onceprgm/cme/internal/java"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/meta"
	"github.com/onceprgm/cme/internal/store"
)

func InstallNeoForge(mc, version string, progress func(stage string, done, total int)) (string, error) {
	if version == "" {
		v, err := meta.NeoForgeLatest(mc)
		if err != nil {
			return "", err
		}
		version = v
	}
	id := "neoforge-" + version

	javaBin, err := javaForVersion(mc)
	if err != nil {
		return "", err
	}

	installerJar, err := downloadInstaller(version, progress)
	if err != nil {
		return "", err
	}

	if err := ensureLauncherProfiles(); err != nil {
		return "", err
	}

	if err := runNeoForgeInstaller(javaBin, installerJar, progress); err != nil {
		return "", err
	}

	if _, err := os.Stat(filepath.Join(store.VersionDir(id), id+".json")); err != nil {
		return "", fmt.Errorf("neoforge installer did not produce %s", id)
	}

	clog.Info("installed neoforge", "id", id, "mc", mc, "java", javaBin)
	return id, nil
}

func javaForVersion(mc string) (string, error) {
	m, err := manifest.FetchFresh()
	if err != nil {
		return "", err
	}
	v := m.Find(mc)
	if v == nil {
		return "", fmt.Errorf("version %q not found, try: cme version list", mc)
	}
	vmeta, _, err := manifest.FetchVersionMeta(v)
	if err != nil {
		return "", err
	}
	return java.Resolve(vmeta.JavaVersion.MajorVersion, "")
}

func downloadInstaller(version string, progress func(stage string, done, total int)) (string, error) {
	url := meta.NeoForgeInstallerURL(version)
	sha, err := mavenSHA1(url)
	if err != nil {
		clog.Warn("neoforge: no sha1 sidecar for installer", "err", err.Error())
	}

	if err := store.Ensure(store.CacheDir()); err != nil {
		return "", err
	}
	dest := filepath.Join(store.CacheDir(), "neoforge-"+version+"-installer.jar")

	progress("installer", 0, 1)
	if err := download.File(url, dest, sha, 0); err != nil {
		return "", err
	}
	progress("installer", 1, 1)
	return dest, nil
}

func ensureLauncherProfiles() error {
	p := filepath.Join(store.DataDir(), "launcher_profiles.json")
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	if err := store.Ensure(store.DataDir()); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(`{"profiles":{},"selectedProfile":"","clientToken":""}`), 0o644)
}

func runNeoForgeInstaller(javaBin, installerJar string, progress func(stage string, done, total int)) error {
	if err := store.Ensure(store.StateDir()); err != nil {
		return err
	}
	logPath := filepath.Join(store.StateDir(), "neoforge-installer.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	progress("neoforge", 0, 1)
	cmd := exec.Command(javaBin, "-Djava.awt.headless=true", "-jar", installerJar, "--install-client", store.DataDir())
	cmd.Dir = store.StateDir()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	clog.Info("running neoforge installer", "java", javaBin, "log", logPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("neoforge installer failed: %w (see %s)", err, logPath)
	}
	progress("neoforge", 1, 1)
	return nil
}
