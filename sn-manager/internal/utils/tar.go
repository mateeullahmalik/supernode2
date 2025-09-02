package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExtractFileFromTarGz extracts a single file (by base name) from a .tar.gz to outPath.
func ExtractFileFromTarGz(tarGzPath, targetBase, outPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tarball: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}
		if filepath.Base(hdr.Name) != targetBase {
			continue
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(outPath)
			return fmt.Errorf("failed to extract %s: %w", targetBase, err)
		}
		if err := out.Close(); err != nil {
			os.Remove(outPath)
			return fmt.Errorf("failed to close output file: %w", err)
		}
		return nil
	}
	return fmt.Errorf("%s not found in tarball", targetBase)
}

// ExtractMultipleFromTarGz extracts multiple files by base name to given output paths.
// Pass a map of baseName -> outPath. Returns a map of baseName -> found.
func ExtractMultipleFromTarGz(tarGzPath string, targets map[string]string) (map[string]bool, error) {
	found := make(map[string]bool, len(targets))

	f, err := os.Open(tarGzPath)
	if err != nil {
		return found, fmt.Errorf("failed to open tarball: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return found, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return found, fmt.Errorf("tar read error: %w", err)
		}
		base := filepath.Base(hdr.Name)
		outPath, ok := targets[base]
		if !ok {
			continue
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			return found, fmt.Errorf("failed to create %s: %w", outPath, err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(outPath)
			return found, fmt.Errorf("failed to extract %s: %w", base, err)
		}
		if err := out.Close(); err != nil {
			os.Remove(outPath)
			return found, fmt.Errorf("failed to close %s: %w", outPath, err)
		}
		found[base] = true
		// If all targets found, we could stop early
		if len(found) == len(targets) {
			break
		}
	}
	return found, nil
}
