package manifest

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type VersionMeta struct {
	ID          string `json:"id"`
	MainClass   string `json:"mainClass"`
	JavaVersion struct {
		MajorVersion int `json:"majorVersion"`
	} `json:"javaVersion"`
	AssetIndex struct {
		ID   string `json:"id"`
		URL  string `json:"url"`
		SHA1 string `json:"sha1"`
	} `json:"assetIndex"`
	Downloads struct {
		Client Artifact `json:"client"`
	} `json:"downloads"`
	Libraries []Library `json:"libraries"`
}

type Artifact struct {
	URL  string `json:"url"`
	SHA1 string `json:"sha1"`
	Size int64  `json:"size"`
}

func FetchVersionMeta(v *Version) (*VersionMeta, []byte, error) {
	resp, err := httpClient.Get(v.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch version json: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch version json: unexpected status %s", resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if v.SHA1 != "" {
		sum := sha1.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != v.SHA1 {
			return nil, nil, fmt.Errorf("version json sha1 mismatch: want %s, got %s", v.SHA1, got)
		}
	}

	var meta VersionMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, nil, fmt.Errorf("parse version json: %w", err)
	}
	return &meta, raw, nil
}

func (m *VersionMeta) ResolvedLibraries(ctx RuleContext) []Library {
	out := make([]Library, 0, len(m.Libraries))
	for _, l := range m.Libraries {
		if Allowed(l.Rules, ctx) {
			out = append(out, l)
		}
	}
	return out
}
