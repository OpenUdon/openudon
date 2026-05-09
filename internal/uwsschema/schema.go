package uwsschema

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"golang.org/x/mod/module"
)

const uwsModulePath = "github.com/OpenUdon/uws"

// PathForVersion returns the best local schema path for a UWS version.
func PathForVersion(anchorDir, version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "1.0.0"
	}
	name := version + ".json"
	if dir := strings.TrimSpace(os.Getenv("OPENUDON_UWS_SCHEMA_DIR")); dir != "" {
		return filepath.Join(dir, name)
	}
	if path, ok := siblingSchemaPath(name); ok {
		return path
	}
	if path, ok := moduleCacheSchemaPath(name); ok {
		return path
	}
	return filepath.Join(anchorDir, "..", "..", "..", "uws", "versions", name)
}

func siblingSchemaPath(name string) (string, bool) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	path := filepath.Join(repoRoot, "..", "uws", "versions", name)
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

func moduleCacheSchemaPath(name string) (string, bool) {
	version, ok := uwsModuleVersion()
	if !ok {
		return "", false
	}
	path, err := escapedModuleCachePath(uwsModulePath, version)
	if err != nil {
		return "", false
	}
	schema := filepath.Join(path, "versions", name)
	if _, err := os.Stat(schema); err == nil {
		return schema, true
	}
	return "", false
}

func uwsModuleVersion() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}
	for _, dep := range info.Deps {
		if dep.Path != uwsModulePath {
			continue
		}
		if dep.Version != "" {
			return dep.Version, true
		}
		if dep.Replace != nil && dep.Replace.Version != "" {
			return dep.Replace.Version, true
		}
	}
	return "", false
}

func escapedModuleCachePath(path, version string) (string, error) {
	escapedPath, err := module.EscapePath(path)
	if err != nil {
		return "", err
	}
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return "", err
	}
	return filepath.Join(moduleCacheDir(), escapedPath+"@"+escapedVersion), nil
}

func moduleCacheDir() string {
	if dir := strings.TrimSpace(os.Getenv("GOMODCACHE")); dir != "" {
		return dir
	}
	gopath := strings.TrimSpace(os.Getenv("GOPATH"))
	if gopath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath == "" {
		return ""
	}
	first := filepath.SplitList(gopath)[0]
	if first == "" {
		return ""
	}
	return filepath.Join(first, "pkg", "mod")
}
