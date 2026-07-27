package meta

import (
	"fmt"

	"github.com/onceprgm/cme/internal/manifest"
)

const fabricBase = "https://meta.fabricmc.net/v2"

func FabricLatestLoader(game string) (string, error) {
	list, err := loaderList(fabricBase, game, "fabric")
	if err != nil {
		return "", err
	}
	v, ok := selectStable(list)
	if !ok {
		return "", fmt.Errorf("fabric: no loader available for minecraft %s", game)
	}
	return v, nil
}

func selectStable(list []loaderEntry) (string, bool) {
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
	return profile(fabricBase, game, loader, "fabric")
}
