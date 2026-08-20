//go:build !windows

// Package executablefile validates explicit local executable paths using the
// host platform's executable-file conventions.
package executablefile

import "os"

func Is(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0
}
