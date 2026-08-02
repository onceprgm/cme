package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/download"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/store"
)

type VerifyReport struct {
	OK            int
	Repaired      int
	LoaderPresent int
	LoaderMissing int
}

type checkItem struct {
	dest string
	url  string
	sha1 string
	size int64
}

func Verify(id string, progress func(stage string, done, total int)) (*VerifyReport, error) {
	meta, owner, err := resolveInstalled(id)
	if err != nil {
		return nil, err
	}
	ctx := manifest.CurrentContext()

	var hashed []checkItem
	var loose []checkItem

	client := meta.Downloads.Client
	if client.URL != "" {
		hashed = append(hashed, checkItem{
			dest: filepath.Join(store.VersionDir(owner), owner+".jar"),
			url:  client.URL, sha1: client.SHA1, size: client.Size,
		})
	}

	for _, l := range meta.ResolvedLibraries(ctx) {
		f, ok := l.Artifact()
		if !ok || f.URL == "" {
			continue
		}
		dest, err := store.SafeJoin(store.LibrariesDir(), f.Path)
		if err != nil {
			return nil, err
		}
		if f.SHA1 != "" {
			hashed = append(hashed, checkItem{dest: dest, url: f.URL, sha1: f.SHA1, size: f.Size})
		} else {
			loose = append(loose, checkItem{dest: dest, url: f.URL})
		}
	}

	assets, err := assetCheckItems(meta)
	if err != nil {
		return nil, err
	}
	hashed = append(hashed, assets...)

	report := &VerifyReport{}
	var broken []download.Task

	for i, it := range hashed {
		if download.Check(it.dest, it.sha1, it.size) {
			report.OK++
		} else {
			broken = append(broken, download.Task{URL: it.url, Dest: it.dest, SHA1: it.sha1, Size: it.size})
		}
		progress("checking", i+1, len(hashed))
	}
	report.Repaired = len(broken)

	for _, it := range loose {
		if fileExists(it.dest) {
			report.LoaderPresent++
			continue
		}
		report.LoaderMissing++
		sha, err := mavenSHA1(it.url)
		if err != nil {
			clog.Warn("verify: no sha1 sidecar for missing loader lib", "url", it.url, "err", err.Error())
		}
		broken = append(broken, download.Task{URL: it.url, Dest: it.dest, SHA1: sha})
	}

	clog.Info("verify", "id", id, "ok", report.OK, "repaired", report.Repaired,
		"loader_present", report.LoaderPresent, "loader_missing", report.LoaderMissing)

	if len(broken) > 0 {
		if err := download.All(broken, download.DefaultWorkers(), func(done, total int) {
			progress("repairing", done, total)
		}); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func resolveInstalled(id string) (*manifest.VersionMeta, string, error) {
	meta, err := manifest.LoadVersionMeta(filepath.Join(store.VersionDir(id), id+".json"))
	if err != nil {
		return nil, "", fmt.Errorf("load version %s: %w (is it installed?)", id, err)
	}
	owner := id
	if meta.InheritsFrom != "" {
		parentID := meta.InheritsFrom
		parent, err := manifest.LoadVersionMeta(filepath.Join(store.VersionDir(parentID), parentID+".json"))
		if err != nil {
			return nil, "", fmt.Errorf("load parent version %s: %w", parentID, err)
		}
		meta = manifest.Merge(parent, meta)
		owner = parentID
	}
	return meta, owner, nil
}

func assetCheckItems(meta *manifest.VersionMeta) ([]checkItem, error) {
	if meta.AssetIndex.ID == "" {
		return nil, nil
	}

	idx, err := loadAssetIndex(meta.AssetIndex.ID, meta.AssetIndex.URL, meta.AssetIndex.SHA1)
	if err != nil {
		return nil, err
	}

	objectsDir := filepath.Join(store.AssetsDir(), "objects")
	var items []checkItem
	seen := map[string]bool{}
	for _, o := range idx.Objects {
		if len(o.Hash) < 2 || seen[o.Hash] {
			continue
		}
		seen[o.Hash] = true
		dest, err := store.SafeJoin(objectsDir, o.Path())
		if err != nil {
			return nil, err
		}
		items = append(items, checkItem{dest: dest, url: o.URL(), sha1: o.Hash, size: o.Size})
	}
	return items, nil
}

func loadAssetIndex(id, url, sha1 string) (*manifest.AssetIndex, error) {
	local := filepath.Join(store.AssetsDir(), "indexes", id+".json")
	if raw, err := os.ReadFile(local); err == nil {
		var idx manifest.AssetIndex
		if err := json.Unmarshal(raw, &idx); err != nil {
			return nil, fmt.Errorf("parse asset index %s: %w", id, err)
		}
		return &idx, nil
	}
	if url == "" {
		return nil, fmt.Errorf("asset index %s missing and no source url", id)
	}
	idx, _, err := manifest.FetchAssetIndex(url, sha1)
	return idx, err
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
