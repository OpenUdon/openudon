package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func registrationQualificationRunParent(t *testing.T, root string) string {
	t.Helper()
	parent := filepath.Join(root, "eval", "runs")
	created := make([]string, 0, 2)
	t.Cleanup(func() {
		for index := len(created) - 1; index >= 0; index-- {
			if err := os.Remove(created[index]); err != nil && !os.IsNotExist(err) {
				t.Errorf("registration qualification workspace cleanup: %v", err)
			}
		}
	})
	for _, directory := range []string{filepath.Join(root, "eval"), parent} {
		info, err := os.Lstat(directory)
		if os.IsNotExist(err) {
			if err := os.Mkdir(directory, 0700); err != nil {
				t.Fatal("registration qualification workspace")
			}
			created = append(created, directory)
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatal("registration qualification workspace")
		}
	}
	return parent
}
