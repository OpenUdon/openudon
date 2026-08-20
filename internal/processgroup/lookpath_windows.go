//go:build windows

package processgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveCommand(name string, environment []string) (string, error) {
	if strings.ContainsAny(name, `/\\:`) {
		return name, nil
	}
	extensions := filepath.SplitList(environmentValue(environment, "PATHEXT", true))
	if len(extensions) == 0 {
		extensions = []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	if filepath.Ext(name) != "" {
		extensions = append([]string{""}, extensions...)
	}
	for _, directory := range filepath.SplitList(environmentValue(environment, "PATH", true)) {
		if directory == "" || !filepath.IsAbs(directory) {
			continue
		}
		for _, extension := range extensions {
			candidate := filepath.Join(directory, name+extension)
			info, err := os.Stat(candidate)
			if err == nil && info.Mode().IsRegular() {
				absolute, err := filepath.Abs(candidate)
				if err != nil {
					return "", err
				}
				return absolute, nil
			}
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
