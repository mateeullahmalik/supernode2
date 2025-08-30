//go:build linux
// +build linux

package configurer

import (
	"os"
	"path/filepath"
)

var defaultConfigPaths = []string{
    "$HOME/.supernode",
    ".",
}

// DefaultPath returns the default config path for Linux OS.
func DefaultPath() string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".supernode")
}
