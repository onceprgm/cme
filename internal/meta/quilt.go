package meta

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/onceprgm/cme/internal/manifest"
)

const quiltBase = "https://meta.quiltmc.org/v3"

func QuiltLatestLoader(game string) (string, error) {
	list, err := loaderList(quiltBase, game, "quilt")
	if err != nil {
		return "", err
	}
	v, ok := selectNewestStable(list)
	if !ok {
		return "", fmt.Errorf("quilt: no loader available for minecraft %s", game)
	}
	return v, nil
}

func selectNewestStable(list []loaderEntry) (string, bool) {
	best := ""
	for _, e := range list {
		v := e.Loader.Version
		if strings.Contains(v, "-") {
			continue
		}
		if best == "" || versionLess(best, v) {
			best = v
		}
	}
	if best != "" {
		return best, true
	}
	if len(list) > 0 {
		return list[0].Loader.Version, true
	}
	return "", false
}

func versionLess(a, b string) bool {
	as, bs := parseVersion(a), parseVersion(b)
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x = as[i]
		}
		if i < len(bs) {
			y = bs[i]
		}
		if x != y {
			return x < y
		}
	}
	return false
}

func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		out[i] = n
	}
	return out
}

func QuiltProfile(game, loader string) (*manifest.VersionMeta, []byte, error) {
	return profile(quiltBase, game, loader, "quilt")
}
