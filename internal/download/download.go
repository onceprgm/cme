package download

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Minute}

func File(url, dest, wantSHA1 string) error {
	if wantSHA1 != "" {
		ok, err := verify(dest, wantSHA1)
		if err == nil && ok {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}

	h := sha1.New()
	_, err = io.Copy(f, io.TeeReader(resp.Body, h))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(part)
		return fmt.Errorf("download %s: %w", url, err)
	}

	if wantSHA1 != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if got != wantSHA1 {
			os.Remove(part)
			return fmt.Errorf("download %s: sha1 mismatch: want %s, got %s", url, wantSHA1, got)
		}
	}

	return os.Rename(part, dest)
}

func verify(path, wantSHA1 string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == wantSHA1, nil
}
