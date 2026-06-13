package download

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const maxRetries = 3

var httpClient = &http.Client{
	Timeout: 10 * time.Minute,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxConnsPerHost:       16,
		IdleConnTimeout:       90 * time.Second,
	},
}

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

func File(url, dest, wantSHA1 string) error {
	if wantSHA1 != "" {
		if ok, err := verify(dest, wantSHA1); err == nil && ok {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = fetch(url, dest, wantSHA1)
		if err == nil {
			return nil
		}
		var perm permanentError
		if errors.As(err, &perm) {
			return err
		}
		if attempt < maxRetries {
			time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
		}
	}
	return fmt.Errorf("download %s: gave up after %d attempts: %w", url, maxRetries, err)
}

func fetch(url, dest, wantSHA1 string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return permanentError{err}
		}
		return err
	}

	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return permanentError{err}
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
		if got := hex.EncodeToString(h.Sum(nil)); got != wantSHA1 {
			os.Remove(part)
			return permanentError{fmt.Errorf("download %s: sha1 mismatch: want %s, got %s", url, wantSHA1, got)}
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
