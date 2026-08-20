//go:build windows

// Package executablefile validates explicit local executable paths using the
// host platform's executable-file conventions.
package executablefile

import (
	"os"
	"path/filepath"
	"strings"
)

func Is(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".com", ".bat", ".cmd":
		return true
	default:
		return false
	}
}
