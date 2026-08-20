// Copyright (c) Greetingland LLC

// Package atomicfile provides atomic file write helpers shared by OpenUdon
// packages. Writes go to a sibling temp file and are renamed into place
// so concurrent readers never observe a partial file.
package atomicfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Write writes data to path atomically with the given permission. The data
// is first written to a temp file in the same directory, then renamed.
func Write(path string, data []byte, perm os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("atomic file path is required")
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
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return syncParent(filepath.Dir(path))
}

// WriteNew writes data durably without replacing an existing path. The hard
// link installs the fully prepared sibling temp file only when path is still
// absent, avoiding a check-then-rename overwrite race for create-only files.
func WriteNew(path string, data []byte, perm os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("atomic file path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Link(tmpPath, path); err != nil {
		return err
	}
	if err := os.Remove(tmpPath); err != nil {
		return err
	}
	return syncParent(dir)
}

// syncParent makes the rename durable on filesystems that support directory
// synchronization. Some platforms and filesystems reject directory Sync even
// though the rename itself is durable; those unsupported cases are ignored.
func syncParent(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil &&
		!errors.Is(err, syscall.EINVAL) &&
		!errors.Is(err, syscall.ENOTSUP) &&
		!errors.Is(err, syscall.EBADF) {
		return err
	}
	return nil
}
