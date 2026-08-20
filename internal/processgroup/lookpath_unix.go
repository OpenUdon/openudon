//go:build !windows

package processgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveCommand(name string, environment []string) (string, error) {
	if strings.ContainsRune(name, filepath.Separator) {
		return name, nil
	}
	pathValue := environmentValue(environment, "PATH", false)
	for _, directory := range filepath.SplitList(pathValue) {
		if directory == "" || !filepath.IsAbs(directory) {
			continue
		}
		candidate := filepath.Join(directory, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			absolute, err := filepath.Abs(candidate)
			if err != nil {
				return "", err
			}
			return absolute, nil
		}
	}
	return "", fmt.Errorf("executable %q was not found in the invocation PATH", name)
}

func environmentValue(environment []string, wanted string, fold bool) string {
	for _, item := range environment {
		name, value, ok := strings.Cut(item, "=")
		if ok && ((!fold && name == wanted) || (fold && strings.EqualFold(name, wanted))) {
			return value
		}
	}
	return ""
}
