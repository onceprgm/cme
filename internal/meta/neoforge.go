package meta

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const neoforgeGroup = "https://maven.neoforged.net/releases/net/neoforged/neoforge"

var nfVersionRe = regexp.MustCompile(`<version>([^<]+)</version>`)

func NeoForgeInstallerURL(version string) string {
	return fmt.Sprintf("%s/%s/neoforge-%s-installer.jar", neoforgeGroup, version, version)
}

func NeoForgePrefix(mc string) (string, error) {
	parts := strings.Split(mc, ".")
	if parts[0] != "1" || len(parts) < 2 {
		return "", fmt.Errorf("neoforge: unsupported minecraft version %q", mc)
	}
	if len(parts) == 2 {
		return parts[1] + ".0.", nil
	}
	return parts[1] + "." + parts[2] + ".", nil
}

func NeoForgeLatest(mc string) (string, error) {
	prefix, err := NeoForgePrefix(mc)
	if err != nil {
		return "", err
	}
	versions, err := neoforgeVersions()
	if err != nil {
		return "", err
	}

	best := ""
	for _, v := range versions {
		if !strings.HasPrefix(v, prefix) || strings.Contains(v, "-") {
			continue
		}
		if best == "" || versionLess(best, v) {
			best = v
		}
	}
	if best == "" {
		return "", fmt.Errorf("neoforge: no build for minecraft %s (is that version supported?)", mc)
	}
	return best, nil
}

func neoforgeVersions() ([]string, error) {
	resp, err := httpClient.Get(neoforgeGroup + "/maven-metadata.xml")
	if err != nil {
		return nil, fmt.Errorf("neoforge: fetch metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("neoforge: fetch metadata: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, m := range nfVersionRe.FindAllStringSubmatch(string(body), -1) {
		out = append(out, m[1])
	}
	return out, nil
}
