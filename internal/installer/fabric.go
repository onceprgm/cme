package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/download"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/meta"
	"github.com/onceprgm/cme/internal/store"
)

var sidecarClient = &http.Client{Timeout: 15 * time.Second}

func InstallFabric(vanilla *manifest.Version, loader string, progress func(stage string, done, total int)) (*manifest.VersionMeta, error) {
	base, err := Install(vanilla, progress)
	if err != nil {
		return nil, err
	}

	if loader == "" {
		loader, err = meta.FabricLatestLoader(vanilla.ID)
		if err != nil {
			return nil, err
		}
	}

	profile, raw, err := meta.FabricProfile(vanilla.ID, loader)
	if err != nil {
		return nil, err
	}

	if err := downloadLoaderLibraries(profile.Libraries, progress); err != nil {
		return nil, err
	}

	dir := store.VersionDir(profile.ID)
	if err := store.Ensure(dir); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, profile.ID+".json"), raw, 0o644); err != nil {
		return nil, err
	}

	clog.Info("installed fabric", "id", profile.ID, "loader", loader, "game", vanilla.ID)
	return manifest.Merge(base, profile), nil
}

func downloadLoaderLibraries(libs []manifest.Library, progress func(stage string, done, total int)) error {
	var tasks []download.Task
	for _, l := range libs {
		f, ok := l.Artifact()
		if !ok || f.URL == "" {
			continue
		}
		sha, err := mavenSHA1(f.URL)
		if err != nil {
			clog.Warn("fabric: no sha1 sidecar, downloading unverified", "lib", l.Name, "err", err.Error())
		}
		tasks = append(tasks, download.Task{
			URL:  f.URL,
			Dest: filepath.Join(store.LibrariesDir(), filepath.FromSlash(f.Path)),
			SHA1: sha,
		})
	}

	return download.All(tasks, download.DefaultWorkers(), func(done, total int) {
		progress("loader", done, total)
	})
}

func mavenSHA1(jarURL string) (string, error) {
	resp, err := sidecarClient.Get(jarURL + ".sha1")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s", resp.Status)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return "", err
	}

	s := strings.TrimSpace(string(b))
	if i := strings.IndexAny(s, " \t"); i > 0 {
		s = s[:i]
	}
	if len(s) != 40 {
		return "", fmt.Errorf("unexpected sha1 %q", s)
	}
	return s, nil
}
