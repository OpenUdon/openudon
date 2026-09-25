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

// LoadCurrentQualificationBuildInputLock returns the active current-stack build
// closure. New current-stack evidence uses the Browser 1.10 v4 closure.
func LoadCurrentQualificationBuildInputLock(compatibility CompatibilityLock) (QualificationBuildInputLock, error) {
	return LoadCurrentQualificationBuildInputLockV4(compatibility)
}

// LoadCurrentQualificationBuildInputLockV3 returns E21's frozen 14-source
// closure for retained v3 report verification.
func LoadCurrentQualificationBuildInputLockV3(compatibility CompatibilityLock) (QualificationBuildInputLock, error) {
	data, err := contracts.ReadFile("current-qualification-build-inputs-v3.json")
	if err != nil {
		return QualificationBuildInputLock{}, err
	}
	var lock QualificationBuildInputLock
	if err := decodeStrict(data, &lock); err != nil {
		return QualificationBuildInputLock{}, err
	}
	if err := ValidateQualificationBuildInputLock(lock, compatibility); err != nil || len(lock.Components) != 14 {
		return QualificationBuildInputLock{}, errors.New("current Udon qualification build-input lock is invalid")
	}
	compatibilityComponents := map[string]LockedRevision{}
	for _, component := range compatibility.Components {
		compatibilityComponents[component.Name] = component
	}
	for _, name := range []string{"browsertools", "uws"} {
		locked := compatibilityComponents[name]
		for _, component := range lock.Components {
			if component.Name == name && locked.Commit != component.Commit {
				return QualificationBuildInputLock{}, fmt.Errorf("current %s build input differs from the compatibility lock", name)
			}
		}
	}
	return lock, nil
}

// LoadCurrentQualificationBuildInputLockV4 reads the Browser 1.10 current
// stack closure.
func LoadCurrentQualificationBuildInputLockV4(compatibility CompatibilityLock) (QualificationBuildInputLock, error) {
	data, err := contracts.ReadFile("current-qualification-build-inputs-v4.json")
	if err != nil {
		return QualificationBuildInputLock{}, err
	}
	var lock QualificationBuildInputLock
	if err := decodeStrict(data, &lock); err != nil {
		return QualificationBuildInputLock{}, err
	}
	if err := ValidateQualificationBuildInputLock(lock, compatibility); err != nil || len(lock.Components) != 14 {
		return QualificationBuildInputLock{}, errors.New("Browser 1.10 qualification build-input lock is invalid")
	}
	compatibilityComponents := map[string]LockedRevision{}
	for _, component := range compatibility.Components {
		compatibilityComponents[component.Name] = component
	}
	for _, name := range []string{"browsertools", "uws"} {
		locked := compatibilityComponents[name]
		matched := false
		for _, component := range lock.Components {
			if component.Name == name {
				matched = component.Commit == locked.Commit
				break
			}
		}
		if !matched {
			return QualificationBuildInputLock{}, fmt.Errorf("Browser 1.10 %s build input differs from the compatibility lock", name)
		}
	}
	return lock, nil
}

// LoadQualificationBuildInputLockForStack selects the historical M86 closure
// or the active Browser 1.10 v4 closure by explicit stack name.
func LoadQualificationBuildInputLockForStack(stack string) (QualificationBuildInputLock, error) {
	compatibility, err := LoadCompatibilityLockForStack(stack)
	if err != nil {
		return QualificationBuildInputLock{}, err
	}
	switch stack {
	case StackHistorical:
		return LoadQualificationBuildInputLock(compatibility)
	case StackCurrent:
		return LoadCurrentQualificationBuildInputLock(compatibility)
	default:
		return QualificationBuildInputLock{}, errors.New("browser scenario stack must be historical or current")
	}
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
	return validateQualificationBuildInputsForLock(ctx, udonRoot, compatibility, lock)
}

// ValidateQualificationBuildInputsForStack proves that the selected Udon tree
// and every local replacement are clean and match the selected stack closure.
func ValidateQualificationBuildInputsForStack(ctx context.Context, udonRoot, stack string) error {
	compatibility, err := LoadCompatibilityLockForStack(stack)
	if err != nil {
		return err
	}
	lock, err := LoadQualificationBuildInputLockForStack(stack)
	if err != nil {
		return err
	}
	return validateQualificationBuildInputsForLock(ctx, udonRoot, compatibility, lock)
}

// StageCurrentQualificationBuildWorkspace clones the exact current Udon source
// and its locked sibling closure into a disposable workspace. Go tests may
// create ignored test output in their working tree, so qualification runs them
// here instead of in supplied source checkouts.
func StageCurrentQualificationBuildWorkspace(ctx context.Context, udonRoot, targetRoot string) (stagedUdon string, resultErr error) {
	bad := errors.New("current Udon qualification workspace staging failed")
	if ctx == nil || !filepath.IsAbs(udonRoot) || filepath.Clean(udonRoot) != udonRoot || !filepath.IsAbs(targetRoot) || filepath.Clean(targetRoot) != targetRoot {
		return "", bad
	}
	udonRoot, err := filepath.EvalSymlinks(udonRoot)
	if err != nil || ValidateQualificationBuildInputsForStack(ctx, udonRoot, StackCurrent) != nil {
		return "", bad
	}
	targetParent, err := filepath.EvalSymlinks(filepath.Dir(targetRoot))
	if err != nil {
		return "", bad
	}
	targetRoot = filepath.Join(targetParent, filepath.Base(targetRoot))
	rel, err := filepath.Rel(filepath.Dir(udonRoot), targetRoot)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", bad
	}
	compatibility, err := LoadCurrentCompatibilityLock()
	if err != nil {
		return "", bad
	}
	lock, err := LoadCurrentQualificationBuildInputLock(compatibility)
	if err != nil {
		return "", bad
	}
	var udonCommit string
	for _, component := range compatibility.Components {
		if component.Name == "udon" {
			udonCommit = component.Commit
			break
		}
	}
	if udonCommit == "" || os.Mkdir(targetRoot, 0700) != nil {
		return "", bad
	}
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, os.RemoveAll(targetRoot))
		}
	}()
	type source struct {
		name, path, commit string
	}
	sources := []source{{name: "udon", path: udonRoot, commit: udonCommit}}
	for _, component := range lock.Components {
		sources = append(sources, source{
			name: component.Name, path: filepath.Join(filepath.Dir(udonRoot), component.Name), commit: component.Commit,
		})
	}
	for _, source := range sources {
		if err := cloneQualificationSource(ctx, source.path, filepath.Join(targetRoot, source.name), source.commit); err != nil {
			return "", bad
		}
	}
	if ValidateQualificationBuildInputsForStack(ctx, udonRoot, StackCurrent) != nil {
		return "", bad
	}
	return filepath.Join(targetRoot, "udon"), nil
}

func cloneQualificationSource(ctx context.Context, source, target, commit string) error {
	if !filepath.IsAbs(source) || filepath.Clean(source) != source || !filepath.IsAbs(target) || filepath.Clean(target) != target || !commitPattern.MatchString(commit) {
		return errors.New("qualification source clone inputs are invalid")
	}
	actual, dirty, err := exactQualificationRevision(ctx, source)
	if err != nil || dirty || actual != commit {
		return errors.New("qualification source clone is not at its locked clean revision")
	}
	cloned := runBounded(ctx, probeDeadline, source, []string{
		"git", "--no-replace-objects", "clone", "--shared", "--no-checkout", "--quiet", source, target,
	}, nil, "")
	if cloned.err != nil {
		return errors.New("qualification source clone failed")
	}
	checkedOut := runBounded(ctx, probeDeadline, target, []string{
		"git", "--no-replace-objects", "checkout", "--quiet", "--detach", commit,
	}, nil, "")
	if checkedOut.err != nil {
		return errors.New("qualification source checkout failed")
	}
	actual, dirty, err = exactQualificationRevision(ctx, target)
	if err != nil || dirty || actual != commit {
		return errors.New("qualification source clone failed exact revision validation")
	}
	return nil
}

func validateQualificationBuildInputsForLock(ctx context.Context, udonRoot string, compatibility CompatibilityLock, lock QualificationBuildInputLock) error {
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
