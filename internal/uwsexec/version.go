package uwsexec

import (
	"errors"
	"fmt"
	"os"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// DeclaredVersion reads captured regular artifacts and delegates the version
// policy to the public model. Empty means no generated document exists yet.
func DeclaredVersion(paths ...string) (string, error) {
	version := ""
	for _, path := range paths {
		if path == "" {
			continue
		}
		data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read existing UWS version: %w", err)
		}
		document, err := DecodeDocument(data, resolveFormat(path, DocumentFormatAuto))
		if err != nil {
			return "", fmt.Errorf("existing UWS document could not be decoded")
		}
		// Delegate version policy to the public model, retaining its exact
		// declared value. Other document gaps belong to normal assessment.
		for _, issue := range document.ValidateResult().Errors {
			if issue.Path == "uws" {
				return "", fmt.Errorf("existing UWS document declares an unsupported version")
			}
		}
		if version != "" && version != document.UWS {
			return "", fmt.Errorf("existing workflow and exported UWS versions disagree")
		}
		version = document.UWS
	}
	return version, nil
}
