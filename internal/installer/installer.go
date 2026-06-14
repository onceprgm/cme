package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/onceprgm/cme/internal/download"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/store"
)

type nativeLib struct {
	lib  manifest.Library
	file manifest.LibFile
}

func Install(v *manifest.Version, progress func(stage string, done, total int)) (*manifest.VersionMeta, error) {
	meta, raw, err := manifest.FetchVersionMeta(v)
	if err != nil {
		return nil, err
	}

	dir := store.VersionDir(meta.ID)
	if err := store.Ensure(dir); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, meta.ID+".json"), raw, 0o644); err != nil {
		return nil, err
	}

	progress("client", 0, 1)
	jar := filepath.Join(dir, meta.ID+".jar")
	if err := download.File(meta.Downloads.Client.URL, jar, meta.Downloads.Client.SHA1); err != nil {
		return nil, err
	}
	progress("client", 1, 1)

	ctx := manifest.CurrentContext()
	libs := meta.ResolvedLibraries(ctx)

	var tasks []download.Task
	var natives []nativeLib
	seen := map[string]bool{}

	add := func(f manifest.LibFile) {
		if f.URL == "" || seen[f.Path] {
			return
		}
		seen[f.Path] = true
		tasks = append(tasks, download.Task{
			URL:  f.URL,
			Dest: filepath.Join(store.LibrariesDir(), filepath.FromSlash(f.Path)),
			SHA1: f.SHA1,
		})
	}

	for _, l := range libs {
		if l.Downloads.Artifact != nil {
			add(*l.Downloads.Artifact)
		}
		if f, ok := l.NativeClassifier(ctx); ok {
			add(f)
			natives = append(natives, nativeLib{lib: l, file: f})
		}
	}

	if err := download.All(tasks, download.DefaultWorkers(), func(done, total int) {
		progress("libraries", done, total)
	}); err != nil {
		return nil, err
	}

	if len(natives) > 0 {
		nativesDir := filepath.Join(dir, "natives")
		if err := store.Ensure(nativesDir); err != nil {
			return nil, err
		}
		for i, n := range natives {
			src := filepath.Join(store.LibrariesDir(), filepath.FromSlash(n.file.Path))
			if err := extract(src, nativesDir, n.lib.ExcludePatterns()); err != nil {
				return nil, fmt.Errorf("extract natives from %s: %w", n.lib.Name, err)
			}
			progress("natives", i+1, len(natives))
		}
	}

	if err := installAssets(meta, progress); err != nil {
		return nil, err
	}

	return meta, nil
}

func installAssets(meta *manifest.VersionMeta, progress func(stage string, done, total int)) error {
	if meta.AssetIndex.URL == "" {
		return nil
	}

	idx, raw, err := manifest.FetchAssetIndex(meta.AssetIndex.URL, meta.AssetIndex.SHA1)
	if err != nil {
		return err
	}

	indexesDir := filepath.Join(store.AssetsDir(), "indexes")
	if err := store.Ensure(indexesDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(indexesDir, meta.AssetIndex.ID+".json"), raw, 0o644); err != nil {
		return err
	}

	objectsDir := filepath.Join(store.AssetsDir(), "objects")
	seen := map[string]bool{}
	var tasks []download.Task
	for name, o := range idx.Objects {
		if len(o.Hash) < 2 {
			return fmt.Errorf("asset %q has malformed hash %q", name, o.Hash)
		}
		if seen[o.Hash] {
			continue
		}
		seen[o.Hash] = true
		tasks = append(tasks, download.Task{
			URL:  o.URL(),
			Dest: filepath.Join(objectsDir, filepath.FromSlash(o.Path())),
			SHA1: o.Hash,
		})
	}

	return download.All(tasks, download.DefaultWorkers(), func(done, total int) {
		progress("assets", done, total)
	})
}

func extract(jarPath, destDir string, exclude []string) error {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer r.Close()

next:
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		for _, ex := range exclude {
			if strings.HasPrefix(f.Name, ex) {
				continue next
			}
		}
		dest := filepath.Join(destDir, filepath.Base(f.Name))
		if err := writeZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func writeZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	part := dest + ".part"
	out, err := os.Create(part)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, rc)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, dest)
}
