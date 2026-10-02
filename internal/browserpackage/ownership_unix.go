//go:build unix

package browserpackage

import (
	"os"
	"syscall"
)

// Keep the original effective-owner, writable-mode and single-link boundary.
func safeOwnedPath(info os.FileInfo, regular bool) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Geteuid()) && info.Mode().Perm()&0022 == 0 && (!regular || st.Nlink == 1)
}
