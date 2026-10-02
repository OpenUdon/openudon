//go:build !unix

package browserpackage

import "os"

// No ACL/owner equivalence has been qualified on this platform. Other public
// commands remain buildable; reviewed capture-package admission fails closed.
func safeOwnedPath(os.FileInfo, bool) bool { return false }
