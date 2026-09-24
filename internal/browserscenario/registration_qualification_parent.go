package browserscenario

import (
	"errors"
	"os"
	"path/filepath"
)

// createRegistrationQualificationExampleParent creates the repository-local
// example parent without following symlinks. Its cleanup removes only empty
// directories created by this qualification run.
func createRegistrationQualificationExampleParent(root string) (string, func() error, error) {
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return "", nil, errors.New("invalid qualification repository root")
	}

	evalDir := filepath.Join(root, "eval")
	runsDir := filepath.Join(evalDir, "runs")
	created := make([]string, 0, 2)
	cleanup := func() error {
		var cleanupErr error
		for i := len(created) - 1; i >= 0; i-- {
			if err := os.Remove(created[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
				cleanupErr = errors.Join(cleanupErr, err)
			}
		}
		return cleanupErr
	}

	for _, directory := range []string{evalDir, runsDir} {
		info, statErr := os.Lstat(directory)
		if statErr == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return "", nil, errors.Join(errors.New("invalid qualification example parent"), cleanup())
			}
			continue
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", nil, errors.Join(errors.New("inspect qualification example parent"), statErr, cleanup())
		}
		if mkdirErr := os.Mkdir(directory, 0o700); mkdirErr != nil {
			return "", nil, errors.Join(errors.New("create qualification example parent"), mkdirErr, cleanup())
		}
		created = append(created, directory)
	}
	return runsDir, cleanup, nil
}
