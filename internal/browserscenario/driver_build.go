package browserscenario

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/browsercheck"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

// StageBrowserdriver compiles supplied source into a new disposable directory.
// Installed modules are referenced read-only; this never installs dependencies
// or writes dist/build-info into the supplied source worktree.
func StageBrowserdriver(ctx context.Context, source, target string) error {
	return StageBrowserdriverWithNodeModules(ctx, source, filepath.Join(source, "node_modules"), target)
}

// ValidateBrowserdriverNodeModules checks the installed direct dependency
// versions against the pinned Browserdriver package lock without invoking npm.
func ValidateBrowserdriverNodeModules(source, nodeModules string) error {
	bad := errors.New("Browserdriver build dependencies are invalid")
	if !filepath.IsAbs(source) || filepath.Clean(source) != source || !filepath.IsAbs(nodeModules) || filepath.Clean(nodeModules) != nodeModules {
		return bad
	}
	sourceRoot, err := filepath.EvalSymlinks(source)
	if err != nil {
		return bad
	}
	modulesRoot, err := filepath.EvalSymlinks(nodeModules)
	if err != nil {
		return bad
	}
	moduleInfo, err := os.Lstat(modulesRoot)
	if err != nil || !moduleInfo.IsDir() || moduleInfo.Mode()&os.ModeSymlink != 0 {
		return bad
	}
	rel, err := filepath.Rel(sourceRoot, modulesRoot)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return bad
	}
	lockPath := filepath.Join(sourceRoot, "package-lock.json")
	lockInfo, err := os.Lstat(lockPath)
	if err != nil || !lockInfo.Mode().IsRegular() {
		return bad
	}
	lockData, err := os.ReadFile(lockPath)
	if err != nil || len(lockData) == 0 || len(lockData) > 8<<20 {
		return bad
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	if json.Unmarshal(lockData, &lock) != nil {
		return bad
	}
	for _, name := range []string{"@types/node", "playwright", "playwright-core", "typescript"} {
		locked := lock.Packages["node_modules/"+name].Version
		if locked == "" {
			return bad
		}
		packageRoot := modulesRoot
		for _, part := range strings.Split(name, "/") {
			packageRoot = filepath.Join(packageRoot, part)
			packageInfo, err := os.Lstat(packageRoot)
			if err != nil || !packageInfo.IsDir() || packageInfo.Mode()&os.ModeSymlink != 0 {
				return bad
			}
		}
		packagePath := filepath.Join(packageRoot, "package.json")
		packageInfo, err := os.Lstat(packagePath)
		if err != nil || !packageInfo.Mode().IsRegular() {
			return bad
		}
		packageData, err := os.ReadFile(packagePath)
		if err != nil {
			return bad
		}
		var installed struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(packageData, &installed) != nil || installed.Version != locked {
			return bad
		}
	}
	return nil
}

// StageBrowserdriverWithNodeModules builds the exact source with a separately
// supplied, read-only dependency directory. Both compiler output and the
// temporary runtime package live outside the supplied source worktree.
func StageBrowserdriverWithNodeModules(ctx context.Context, source, nodeModules, target string) (resultErr error) {
	bad := errors.New("Browserdriver build prerequisites are invalid")
	if !filepath.IsAbs(source) || !filepath.IsAbs(target) || !filepath.IsAbs(nodeModules) {
		return bad
	}
	modules, err := os.Stat(nodeModules)
	if err != nil || !modules.IsDir() {
		return bad
	}
	data, _, err := evidencefile.ReadRegular(filepath.Join(source, "package.json"), 1<<20)
	if err != nil {
		return bad
	}
	if os.Mkdir(target, 0700) != nil {
		return bad
	}
	if os.WriteFile(filepath.Join(target, "package.json"), data, 0600) != nil || os.Symlink(nodeModules, filepath.Join(target, "node_modules")) != nil {
		return bad
	}
	buildRoot, err := os.MkdirTemp("", "openudon-browserdriver-build-")
	if err != nil {
		return bad
	}
	defer func() {
		if err := os.RemoveAll(buildRoot); err != nil && resultErr == nil {
			resultErr = bad
		}
	}()
	buildSource := filepath.Join(buildRoot, "browserdriver")
	if cloneBrowserdriverSource(ctx, source, buildSource) != nil || os.Symlink(nodeModules, filepath.Join(buildSource, "node_modules")) != nil {
		return bad
	}
	environment := os.Environ()
	for i, value := range environment {
		if strings.HasPrefix(value, "PATH=") {
			environment[i] = "PATH=" + filepath.Join(nodeModules, ".bin") + string(os.PathListSeparator) + strings.TrimPrefix(value, "PATH=")
			break
		}
	}
	err = browsercheck.Build(ctx, "browserdriver", filepath.Join(target, "dist"), func(output string) error {
		// Keep TypeScript resolution rooted at the staged package: realpathing
		// the external read-only node_modules symlink hides sibling packages.
		return processgroup.Run(ctx, buildDeadline, processgroup.Invocation{Args: []string{"npm", "run", "build", "--silent", "--", "--outDir", output, "--incremental", "false", "--preserveSymlinks"}, Dir: buildSource, Env: environment, Stdout: io.Discard, Stderr: io.Discard})
	})
	if err != nil {
		return bad
	}
	return nil
}

func cloneBrowserdriverSource(ctx context.Context, source, target string) error {
	if !filepath.IsAbs(source) || filepath.Clean(source) != source || !filepath.IsAbs(target) || filepath.Clean(target) != target {
		return errors.New("Browserdriver source clone paths are invalid")
	}
	commit := runBounded(ctx, probeDeadline, source, []string{"git", "--no-replace-objects", "rev-parse", "HEAD"}, nil, "")
	revision := strings.TrimSpace(string(commit.stdout))
	if commit.err != nil || !commitPattern.MatchString(revision) {
		return errors.New("Browserdriver source revision is unavailable")
	}
	cloned := runBounded(ctx, buildDeadline, source, []string{
		"git", "--no-replace-objects", "clone", "--shared", "--no-checkout", "--quiet", source, target,
	}, nil, "")
	if cloned.err != nil {
		return errors.New("Browserdriver source clone failed")
	}
	checkedOut := runBounded(ctx, buildDeadline, target, []string{"git", "--no-replace-objects", "checkout", "--quiet", "--detach", revision}, nil, "")
	if checkedOut.err != nil {
		return errors.New("Browserdriver source checkout failed")
	}
	verified := runBounded(ctx, probeDeadline, target, []string{"git", "--no-replace-objects", "rev-parse", "HEAD"}, nil, "")
	if verified.err != nil || strings.TrimSpace(string(verified.stdout)) != revision {
		return errors.New("Browserdriver source clone revision drifted")
	}
	return nil
}
