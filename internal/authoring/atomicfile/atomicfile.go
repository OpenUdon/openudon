// Copyright (c) Greetingland LLC

// Package atomicfile provides atomic file write helpers shared by apitools
// subpackages. Writes go to a sibling temp file and are renamed into place
// so concurrent readers never observe a partial file.
package atomicfile

import (
	"os"

	"github.com/OpenUdon/authoring/lifecycle"
)

// Write writes data to path atomically with the given permission. The data
// is first written to a temp file in the same directory, then renamed.
func Write(path string, data []byte, perm os.FileMode) error {
	return lifecycle.AtomicWrite(path, data, perm)
}
