package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// DownloadTimeout is the timeout for  downloads
	DownloadTimeout = 5 * time.Minute
)

// DownloadFile downloads a file from the given URL to destPath, creating parent
// directories as needed. If progress is non-nil, it is called with (downloaded, total).
func DownloadFile(url, destPath string, progress func(downloaded, total int64)) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmp := destPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp) }()

	client := &http.Client{Timeout: DownloadTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "sn-manager")
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		f.Close()
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if progress != nil {
		pr := &progressWriter{total: total, cb: progress}
		if _, err := io.Copy(f, io.TeeReader(resp.Body, pr)); err != nil {
			f.Close()
			return fmt.Errorf("download error: %w", err)
		}
	} else {
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			return fmt.Errorf("download error: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}
	return nil
}

type progressWriter struct {
	total   int64
	written int64
	cb      func(downloaded, total int64)
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.written += int64(n)
	if p.cb != nil {
		p.cb(p.written, p.total)
	}
	return n, nil
}
