// Copyright (c) Greetingland LLC

// Package atomicfile provides atomic file write helpers shared by apitools
// subpackages. Writes go to a sibling temp file and are renamed into place
// so concurrent readers never observe a partial file.
package atomicfile

import (
	"os"
	"path/filepath"
	"strings"
)

// Write writes data to path atomically with the given permission. The data
// is first written to a temp file in the same directory, then renamed.
func Write(path string, data []byte, perm os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	// Set every fallible file attribute before the atomic rename. Once the
	// rename succeeds, callers can install their prepared in-memory state
	// without a later error claiming that an already-visible draft failed.
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
