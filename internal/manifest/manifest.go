package manifest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/store"
)

const manifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

const cacheTTL = 3 * time.Hour

type VersionType string

const (
	TypeRelease  VersionType = "release"
	TypeSnapshot VersionType = "snapshot"
	TypeOldBeta  VersionType = "old_beta"
	TypeOldAlpha VersionType = "old_alpha"
)

type Manifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []Version `json:"versions"`
}

type Version struct {
	ID          string      `json:"id"`
	Type        VersionType `json:"type"`
	URL         string      `json:"url"`
	ReleaseTime time.Time   `json:"releaseTime"`
	SHA1        string      `json:"sha1"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

func Fetch() (*Manifest, error) {
	path := cachePath()
	if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < cacheTTL {
		if m, err := readCache(path); err == nil {
			clog.Debug("manifest from cache", "age", time.Since(info.ModTime()).Round(time.Second).String())
			return m, nil
		}
	}
	return FetchFresh()
}

func FetchFresh() (*Manifest, error) {
	m, raw, err := fetchNetwork()
	if err != nil {
		if cached, cerr := readCache(cachePath()); cerr == nil {
			clog.Warn("manifest fetch failed, using cached copy", "err", err.Error())
			return cached, nil
		}
		return nil, err
	}
	if werr := writeCache(cachePath(), raw); werr != nil {
		clog.Warn("could not write manifest cache", "err", werr.Error())
	}
	return m, nil
}

func fetchNetwork() (*Manifest, []byte, error) {
	resp, err := httpClient.Get(manifestURL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch manifest: unexpected status %s", resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read manifest: %w", err)
	}
	m, err := parseManifest(raw)
	if err != nil {
		return nil, nil, err
	}
	return m, raw, nil
}

func parseManifest(raw []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

func readCache(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseManifest(raw)
}

func writeCache(path string, raw []byte) error {
	if err := store.Ensure(filepath.Dir(path)); err != nil {
		return err
	}
	tmp := path + ".part"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func cachePath() string {
	return filepath.Join(store.CacheDir(), "version_manifest_v2.json")
}

func (m *Manifest) Find(id string) *Version {
	for i := range m.Versions {
		if m.Versions[i].ID == id {
			return &m.Versions[i]
		}
	}
	return nil
}

func (m *Manifest) Filter(t VersionType) []Version {
	if t == "" {
		return m.Versions
	}
	out := make([]Version, 0, len(m.Versions))
	for _, v := range m.Versions {
		if v.Type == t {
			out = append(out, v)
		}
	}
	return out
}
