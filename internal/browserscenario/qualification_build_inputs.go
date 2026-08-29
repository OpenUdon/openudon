package browserscenario

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

const qualificationBuildInputLockVersion = "openudon.browser-transaction-build-input-lock.v1"

// QualificationBuildInput binds one local sibling replacement used while
// building the locked Udon executor. The lock is embedded in OpenUdon, so the
// OpenUdon revision retained by the public qualification report transitively
// binds every auxiliary source input without changing that report contract.
// Udon's Browsertools/UWS build inputs intentionally retain its reviewed E11
// module closure and may differ from the current producer/report pins.
type QualificationBuildInput struct {
	Name        string `json:"name"`
	Module      string `json:"module"`
	Replacement string `json:"replacement"`
	Commit      string `json:"commit"`
}

type QualificationBuildInputLock struct {
	Version    string                    `json:"version"`
	Components []QualificationBuildInput `json:"components"`
}

func LoadQualificationBuildInputLock(compatibility CompatibilityLock) (QualificationBuildInputLock, error) {
	data, err := contracts.ReadFile("qualification-build-inputs.json")
	if err != nil {
		return QualificationBuildInputLock{}, err
	}
	var lock QualificationBuildInputLock
	if err := decodeStrict(data, &lock); err != nil {
		return QualificationBuildInputLock{}, err
	}
	if err := ValidateQualificationBuildInputLock(lock, compatibility); err != nil {
		return QualificationBuildInputLock{}, err
	}
	return lock, nil
}

func ValidateQualificationBuildInputLock(lock QualificationBuildInputLock, compatibility CompatibilityLock) error {
	if err := ValidateCompatibilityLock(compatibility); err != nil {
		return err
	}
	if lock.Version != qualificationBuildInputLockVersion || len(lock.Components) == 0 || len(lock.Components) > 32 {
		return errors.New("browser transaction qualification build-input lock is incomplete")
	}
	seenNames := map[string]bool{}
	seenModules := map[string]bool{}
	previous := ""
	for _, component := range lock.Components {
		if !idPattern.MatchString(component.Name) || component.Name <= previous || seenNames[component.Name] ||
			component.Module == "" || seenModules[component.Module] || component.Replacement != "../"+component.Name ||
			!commitPattern.MatchString(component.Commit) {
			return errors.New("browser transaction qualification build-input lock is invalid")
		}
		seenNames[component.Name] = true
		seenModules[component.Module] = true
		previous = component.Name
	}
	for _, name := range []string{"browsertools", "uws"} {
		if !seenNames[name] {
			return fmt.Errorf("%s qualification build input is missing", name)
		}
	}
	return nil
}

// ValidateQualificationBuildInputs proves that every local Udon replacement
// is named by the embedded lock and resolves to that exact clean sibling
// commit. It rejects extra, missing, redirected, dirty, or substituted inputs
// before any browser process is launched.
func ValidateQualificationBuildInputs(ctx context.Context, udonRoot string, compatibility CompatibilityLock) error {
	lock, err := LoadQualificationBuildInputLock(compatibility)
	if err != nil {
		return err
	}
	udonCommit := ""
	for _, component := range compatibility.Components {
		if component.Name == "udon" {
			udonCommit = component.Commit
			break
		}
	}
	commit, dirty, err := exactQualificationRevision(ctx, filepath.Clean(udonRoot))
	if err != nil || commit != udonCommit {
		return errors.New("Udon qualification build-input revision does not match the lock")
	}
	if dirty {
		return errors.New("Udon qualification build-input worktree is dirty")
	}
	return validateQualificationBuildInputs(ctx, udonRoot, lock)
}

func validateQualificationBuildInputs(ctx context.Context, udonRoot string, lock QualificationBuildInputLock) error {
	goModPath := filepath.Join(udonRoot, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return errors.New("Udon qualification module is unavailable")
	}
	parsed, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return errors.New("Udon qualification module is invalid")
	}
	localReplacements := map[string]string{}
	for _, replacement := range parsed.Replace {
		if replacement.New.Version != "" || !strings.HasPrefix(filepath.ToSlash(replacement.New.Path), "../") {
			continue
		}
		path := filepath.ToSlash(filepath.Clean(replacement.New.Path))
		if path != replacement.New.Path || strings.Count(strings.TrimPrefix(path, "../"), "/") != 0 || localReplacements[replacement.Old.Path] != "" {
			return errors.New("Udon qualification local replacement is invalid")
		}
		localReplacements[replacement.Old.Path] = path
	}
	if len(localReplacements) != len(lock.Components) {
		return errors.New("Udon qualification local replacements do not match the build-input lock")
	}
	udonRoot = filepath.Clean(udonRoot)
	for _, component := range lock.Components {
		if localReplacements[component.Module] != component.Replacement {
			return fmt.Errorf("%s Udon qualification replacement does not match the build-input lock", component.Name)
		}
		root := filepath.Clean(filepath.Join(udonRoot, component.Replacement))
		if root != filepath.Join(filepath.Dir(udonRoot), component.Name) {
			return fmt.Errorf("%s qualification build-input path is invalid", component.Name)
		}
		info, statErr := os.Lstat(root)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s qualification build input is unavailable", component.Name)
		}
		commit, dirty, revisionErr := exactQualificationRevision(ctx, root)
		if revisionErr != nil || commit != component.Commit {
			return fmt.Errorf("%s qualification build-input revision does not match the lock", component.Name)
		}
		if dirty {
			return fmt.Errorf("%s qualification build-input worktree is dirty", component.Name)
		}
	}
	return nil
}

func exactQualificationRevision(ctx context.Context, root string) (string, bool, error) {
	topResult := runBounded(ctx, probeDeadline, root, []string{"git", "--no-replace-objects", "rev-parse", "--show-toplevel"}, nil, "")
	if topResult.err != nil || filepath.Clean(strings.TrimSpace(string(topResult.stdout))) != filepath.Clean(root) {
		return "", false, errors.New("qualification build-input repository root is invalid")
	}
	commitResult := runBounded(ctx, probeDeadline, root, []string{"git", "--no-replace-objects", "rev-parse", "HEAD"}, nil, "")
	commit := strings.TrimSpace(string(commitResult.stdout))
	if commitResult.err != nil || !commitPattern.MatchString(commit) {
		return "", false, errors.New("qualification build-input revision is unavailable")
	}
	statusResult := runBounded(ctx, probeDeadline, root,
		[]string{"git", "--no-replace-objects", "status", "--porcelain=v1", "--ignored=matching", "--untracked-files=all", "--", "."}, nil, "")
	if statusResult.err != nil {
		return "", false, errors.New("qualification build-input status is unavailable")
	}
	return commit, strings.TrimSpace(string(statusResult.stdout)) != "", nil
}
