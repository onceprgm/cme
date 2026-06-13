package manifest

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type AssetIndex struct {
	Objects        map[string]AssetObject `json:"objects"`
	MapToResources bool                   `json:"map_to_resources"`
	Virtual        bool                   `json:"virtual"`
}

type AssetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

func FetchAssetIndex(url, wantSHA1 string) (*AssetIndex, []byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch asset index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch asset index: unexpected status %s", resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if wantSHA1 != "" {
		sum := sha1.Sum(raw)
		if got := hex.EncodeToString(sum[:]); got != wantSHA1 {
			return nil, nil, fmt.Errorf("asset index sha1 mismatch: want %s, got %s", wantSHA1, got)
		}
	}

	var idx AssetIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, nil, fmt.Errorf("parse asset index: %w", err)
	}
	return &idx, raw, nil
}

func (o AssetObject) Path() string {
	return o.Hash[:2] + "/" + o.Hash
}

func (o AssetObject) URL() string {
	return "https://resources.download.minecraft.net/" + o.Path()
}
