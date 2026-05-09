package packageartifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// CollectOpenAPIPaths returns package-relative OpenAPI artifact paths.
func CollectOpenAPIPaths(packageRoot string) ([]string, error) {
	packageRoot = filepath.Clean(packageRoot)
	openAPIRoot := filepath.Join(packageRoot, "openapi")
	info, err := os.Lstat(openAPIRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("openapi path must not be a symlink: %s", openAPIRoot)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("openapi path must be a directory: %s", openAPIRoot)
	}

	var paths []string
	if err := filepath.WalkDir(openAPIRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("openapi artifact must not be a symlink: %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("openapi artifact must be a regular file: %s", path)
		}
		rel, err := filepath.Rel(packageRoot, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
