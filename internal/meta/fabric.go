package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/onceprgm/cme/internal/manifest"
)

const fabricBase = "https://meta.fabricmc.net/v2"

var httpClient = &http.Client{Timeout: 30 * time.Second}

type loaderEntry struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}

func FabricLatestLoader(game string) (string, error) {
	url := fmt.Sprintf("%s/versions/loader/%s", fabricBase, game)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("fabric: fetch loaders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fabric: fetch loaders: unexpected status %s", resp.Status)
	}

	var list []loaderEntry
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return "", fmt.Errorf("fabric: parse loaders: %w", err)
	}

	loader, ok := selectLoader(list)
	if !ok {
		return "", fmt.Errorf("fabric: no loader available for minecraft %s", game)
	}
	return loader, nil
}

func selectLoader(list []loaderEntry) (string, bool) {
	for _, e := range list {
		if e.Loader.Stable {
			return e.Loader.Version, true
		}
	}
	if len(list) > 0 {
		return list[0].Loader.Version, true
	}
	return "", false
}

func FabricProfile(game, loader string) (*manifest.VersionMeta, []byte, error) {
	url := fmt.Sprintf("%s/versions/loader/%s/%s/profile/json", fabricBase, game, loader)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("fabric: fetch profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fabric: profile for %s / loader %s not found (status %s)", game, loader, resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var m manifest.VersionMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, nil, fmt.Errorf("fabric: parse profile: %w", err)
	}
	return &m, raw, nil
}
