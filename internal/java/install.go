package java

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/store"
)

var httpClient = &http.Client{Timeout: 20 * time.Minute}

type Managed struct {
	Major int    `json:"major"`
	Path  string `json:"path"`
}

func Install(major int, progress func(done, total int64)) (string, error) {
	arch, err := adoptiumArch()
	if err != nil {
		return "", err
	}

	link, sum, size, err := fetchAdoptium(major, arch)
	if err != nil {
		return "", err
	}
	clog.Info("java install", "major", major, "arch", arch, "size", size)

	if err := store.Ensure(store.JavaDir()); err != nil {
		return "", err
	}

	archive, err := downloadVerify(link, sum, size, progress)
	if err != nil {
		return "", err
	}
	defer os.Remove(archive)

	dir := filepath.Join(store.JavaDir(), strconv.Itoa(major))
	staging := dir + ".new"
	os.RemoveAll(staging)
	if err := store.Ensure(staging); err != nil {
		return "", err
	}
	if err := extractTarGz(archive, staging); err != nil {
		os.RemoveAll(staging)
		return "", err
	}
	os.RemoveAll(dir)
	if err := os.Rename(staging, dir); err != nil {
		return "", err
	}

	bin := filepath.Join(dir, "bin", "java")
	got, ok := probe(bin)
	if !ok {
		return "", fmt.Errorf("installed java did not run: %s", bin)
	}
	if !satisfies(got, major) {
		return "", fmt.Errorf("installed java reports major %d, wanted %d", got, major)
	}
	return bin, nil
}

func List() ([]Managed, error) {
	out := []Managed{}
	entries, err := os.ReadDir(store.JavaDir())
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		major, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		bin := filepath.Join(store.JavaDir(), e.Name(), "bin", "java")
		if _, ok := probe(bin); !ok {
			continue
		}
		out = append(out, Managed{Major: major, Path: bin})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Major < out[j].Major })
	return out, nil
}

func adoptiumArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("no Temurin builds for GOARCH %q", runtime.GOARCH)
	}
}

func fetchAdoptium(major int, arch string) (link, sum string, size int64, err error) {
	url := fmt.Sprintf("https://api.adoptium.net/v3/assets/latest/%d/hotspot?os=linux&architecture=%s&image_type=jre", major, arch)
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch temurin metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("fetch temurin metadata: unexpected status %s", resp.Status)
	}

	var assets []struct {
		Binary struct {
			Package struct {
				Link     string `json:"link"`
				Checksum string `json:"checksum"`
				Size     int64  `json:"size"`
			} `json:"package"`
		} `json:"binary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return "", "", 0, fmt.Errorf("parse temurin metadata: %w", err)
	}
	if len(assets) == 0 {
		return "", "", 0, fmt.Errorf("no Temurin JRE for java %d on linux/%s", major, arch)
	}
	p := assets[0].Binary.Package
	if p.Link == "" || p.Checksum == "" {
		return "", "", 0, fmt.Errorf("temurin metadata missing link or checksum for java %d", major)
	}
	return p.Link, p.Checksum, p.Size, nil
}

func downloadVerify(url, wantSum string, size int64, progress func(done, total int64)) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("download temurin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download temurin: unexpected status %s", resp.Status)
	}

	tmp, err := os.CreateTemp(store.JavaDir(), "jre-*.tar.gz")
	if err != nil {
		return "", err
	}

	h := sha256.New()
	src := &countReader{r: resp.Body, total: size, cb: progress}
	_, err = io.Copy(io.MultiWriter(tmp, h), src)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("download temurin: %w", err)
	}

	if got := hex.EncodeToString(h.Sum(nil)); got != wantSum {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("temurin sha256 mismatch: want %s, got %s", wantSum, got)
	}
	return tmp.Name(), nil
}

type countReader struct {
	r     io.Reader
	done  int64
	total int64
	cb    func(done, total int64)
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.done += int64(n)
	if c.cb != nil {
		c.cb(c.done, c.total)
	}
	return n, err
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := stripFirst(hdr.Name)
		if name == "" {
			continue
		}
		target, err := store.SafeJoin(dest, name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o777)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if !safeSymlink(dest, target, hdr.Linkname) {
				clog.Warn("java: skipping unsafe symlink", "name", name, "link", hdr.Linkname)
				continue
			}
			os.MkdirAll(filepath.Dir(target), 0o755)
			os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func safeSymlink(dest, linkPath, linkname string) bool {
	if filepath.IsAbs(linkname) {
		return false
	}
	resolved := filepath.Join(filepath.Dir(linkPath), linkname)
	rel, err := filepath.Rel(dest, resolved)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func stripFirst(name string) string {
	name = strings.TrimPrefix(name, "./")
	i := strings.IndexByte(name, '/')
	if i < 0 {
		return ""
	}
	return name[i+1:]
}
