package manifest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const manifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

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
	resp, err := httpClient.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest: unexpected status %s", resp.Status)
	}

	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
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
