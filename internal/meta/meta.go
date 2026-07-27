package meta

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/onceprgm/cme/internal/manifest"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type loaderEntry struct {
	Loader struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	} `json:"loader"`
}

func loaderList(base, game, provider string) ([]loaderEntry, error) {
	url := fmt.Sprintf("%s/versions/loader/%s", base, game)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%s: fetch loaders: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: fetch loaders: unexpected status %s", provider, resp.Status)
	}

	var list []loaderEntry
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("%s: parse loaders: %w", provider, err)
	}
	return list, nil
}

func profile(base, game, loader, provider string) (*manifest.VersionMeta, []byte, error) {
	url := fmt.Sprintf("%s/versions/loader/%s/%s/profile/json", base, game, loader)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: fetch profile: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%s: profile for %s / loader %s not found (status %s)", provider, game, loader, resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var m manifest.VersionMeta
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, nil, fmt.Errorf("%s: parse profile: %w", provider, err)
	}
	return &m, raw, nil
}
