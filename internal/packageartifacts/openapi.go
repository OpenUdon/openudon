package packageartifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/evidence/artifact"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
)

// artifactLabels surfaces OpenUdon package wording from the shared
// evidence/artifact path-safety primitives.
var artifactLabels = artifact.Options{
	RootLabel:  "package root",
	PathLabel:  "package path",
	InputLabel: "required handoff input",
}

// ManifestInput is the subset of review handoff input metadata needed to
// validate required package inventory.
type ManifestInput struct {
	Path     string
	Required bool
}

const ReviewHandoffPath = "expected/review-handoff.json"
const RuntimeDataPath = "expected/data.hcl"
const BrowserSourceReviewPath = ".icot/browser-sources.json"
const BrowserAuthenticationReviewPath = ".icot/browser-authentication.json"
const BrowserRegistrationReviewPath = ".icot/browser-registration.json"

var fixedRequiredPackagePaths = []string{
	"project.md",
	"workflows/intent.hcl",
	"workflows/workflow.hcl",
	"workflows/workflow.uws.yaml",
	"expected/plan.json",
	"expected/quality.json",
	"expected/refinement.json",
	"expected/review.md",
	ReviewHandoffPath,
}

// CleanRelativePath returns a canonical slash-separated package-relative path.
func CleanRelativePath(inputPath string) (string, error) {
	return artifact.CleanRelativePath(inputPath, artifactLabels)
}

// RequiredPackagePaths returns the fixed handoff inventory, every regular API
// source file staged for execution, and advisory evidence files tied to those
// sources.
func RequiredPackagePaths(packageRoot string) ([]string, error) {
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return nil, err
	}
	paths := append([]string(nil), fixedRequiredPackagePaths...)
	if runtimeDataFileExists(packageRoot) {
		paths = append(paths, RuntimeDataPath)
	}
	openAPIPaths, err := CollectAPISourcePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, openAPIPaths...)
	browserPaths, err := CollectBrowserProfilePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, browserPaths...)
	if len(browserPaths) > 0 {
		paths = append(paths, BrowserSourceReviewPath)
	}
	authenticationPaths, err := CollectBrowserAuthenticationProfilePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, authenticationPaths...)
	if len(authenticationPaths) > 0 {
		paths = append(paths, BrowserAuthenticationReviewPath)
	}
	registrationPaths, err := CollectBrowserRegistrationProfilePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, registrationPaths...)
	registrationBundles, err := CollectBrowserRegistrationBundlePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, registrationBundles...)
	if len(registrationPaths) > 0 {
		paths = append(paths, BrowserRegistrationReviewPath)
	}
	securitySidecars, err := CollectAdvisorySecuritySidecarPaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, securitySidecars...)
	return uniqueSorted(paths)
}

func runtimeDataFileExists(packageRoot string) bool {
	info, err := os.Lstat(filepath.Join(packageRoot, filepath.FromSlash(RuntimeDataPath)))
	return err == nil && info.Mode().IsRegular()
}

// RequiredManifestPaths verifies that every required package path is listed in
// the manifest and returns sorted canonical required manifest paths.
func RequiredManifestPaths(packageRoot string, manifestInputs []ManifestInput) ([]string, error) {
	requiredPaths, err := RequiredPackagePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	required := make(map[string]bool, len(requiredPaths))
	for _, requiredPath := range requiredPaths {
		required[requiredPath] = false
	}
	manifestSet := map[string]struct{}{}
	for _, input := range manifestInputs {
		if !input.Required {
			continue
		}
		clean, err := CleanRelativePath(input.Path)
		if err != nil {
			return nil, fmt.Errorf("handoff input path must be safe relative path: %q", input.Path)
		}
		manifestSet[clean] = struct{}{}
		if _, ok := required[clean]; ok {
			required[clean] = true
		}
	}
	var missing []string
	for requiredPath, found := range required {
		if !found {
			missing = append(missing, requiredPath)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return nil, fmt.Errorf("handoff manifest missing required input(s): %s", strings.Join(missing, ", "))
	}
	paths := make([]string, 0, len(manifestSet))
	for manifestPath := range manifestSet {
		paths = append(paths, manifestPath)
	}
	sort.Strings(paths)
	return paths, nil
}

// ValidateRegularPackageFiles rejects missing, symlinked, directory, and
// special-file package inputs.
func ValidateRegularPackageFiles(packageRoot string, paths []string) error {
	return artifact.ValidateRegularFiles(packageRoot, paths, artifactLabels)
}

func ValidatePackageRoot(packageRoot string) error {
	return artifact.ValidateRoot(packageRoot, artifactLabels)
}

// CollectAPISourcePaths returns package-relative API source artifact paths.
func CollectAPISourcePaths(packageRoot string) ([]string, error) {
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return nil, err
	}
	var paths []string
	for _, dir := range sourcecatalog.API() {
		dirPaths, err := collectSourceDirPaths(packageRoot, dir)
		if err != nil {
			return nil, err
		}
		paths = append(paths, dirPaths...)
	}
	sort.Strings(paths)
	if err := ValidateRegularPackageFiles(packageRoot, paths); err != nil {
		return nil, err
	}
	return paths, nil
}

// CollectBrowserProfilePaths returns reviewed browser-profile inputs without
// mixing them into the legacy api_source_paths executor field.
func CollectBrowserProfilePaths(packageRoot string) ([]string, error) {
	paths, err := collectSourceDirPaths(packageRoot, "browser-profiles")
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".yaml", ".yml":
		default:
			return nil, fmt.Errorf("browser profile must use .json, .yaml, or .yml: %s", path)
		}
	}
	return paths, nil
}

// CollectBrowserAuthenticationProfilePaths returns portable, secret-free
// browser sign-in profiles staged for Udon's private runtime driver.
func CollectBrowserAuthenticationProfilePaths(packageRoot string) ([]string, error) {
	paths, err := collectSourceDirPaths(packageRoot, "browser-authentication")
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(strings.ToLower(path), ".review.json") {
			continue
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".yaml", ".yml":
		default:
			return nil, fmt.Errorf("browser authentication profile must use .json, .yaml, or .yml: %s", path)
		}
		profiles = append(profiles, path)
	}
	return profiles, nil
}

// CollectBrowserRegistrationProfilePaths returns portable, secret-free
// account-registration profiles while excluding their digest-bound review
// bundles.
func CollectBrowserRegistrationProfilePaths(packageRoot string) ([]string, error) {
	paths, err := collectSourceDirPaths(packageRoot, "browser-registration")
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(strings.ToLower(path), ".review.json") {
			continue
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".yaml", ".yml":
		default:
			return nil, fmt.Errorf("browser registration profile must use .json, .yaml, or .yml: %s", path)
		}
		profiles = append(profiles, path)
	}
	return profiles, nil
}

// CollectBrowserRegistrationBundlePaths returns exactly one conventional
// Browsertools review bundle for every registration profile and rejects
// orphan bundles.
func CollectBrowserRegistrationBundlePaths(packageRoot string) ([]string, error) {
	paths, err := collectSourceDirPaths(packageRoot, "browser-registration")
	if err != nil {
		return nil, err
	}
	profiles := map[string]bool{}
	bundles := map[string]bool{}
	for _, path := range paths {
		if strings.HasSuffix(strings.ToLower(path), ".review.json") {
			bundles[path] = true
			continue
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".yaml", ".yml":
			profiles[path] = true
		default:
			return nil, fmt.Errorf("browser registration profile must use .json, .yaml, or .yml: %s", path)
		}
	}
	result := make([]string, 0, len(profiles))
	for profilePath := range profiles {
		reviewPath := BrowserRegistrationBundlePath(profilePath)
		if !bundles[reviewPath] {
			return nil, fmt.Errorf("browser registration profile %s requires review bundle %s", profilePath, reviewPath)
		}
		delete(bundles, reviewPath)
		result = append(result, reviewPath)
	}
	if len(bundles) != 0 {
		orphans := make([]string, 0, len(bundles))
		for path := range bundles {
			orphans = append(orphans, path)
		}
		sort.Strings(orphans)
		return nil, fmt.Errorf("browser registration review bundle has no matching profile: %s", strings.Join(orphans, ", "))
	}
	sort.Strings(result)
	return result, nil
}

// BrowserRegistrationBundlePath returns the conventional package-relative
// Browsertools review path for one profile.
func BrowserRegistrationBundlePath(profilePath string) string {
	ext := filepath.Ext(profilePath)
	return strings.TrimSuffix(profilePath, ext) + ".review.json"
}

// CollectExecutionSourcePaths returns every UWS source document staged in a
// package while preserving API/browser-specific collection APIs.
func CollectExecutionSourcePaths(packageRoot string) ([]string, error) {
	apiPaths, err := CollectAPISourcePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	browserPaths, err := CollectBrowserProfilePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	return uniqueSorted(append(apiPaths, browserPaths...))
}

// CollectAdvisorySecuritySidecarPaths returns package-relative security
// sidecar paths that are associated with a real API source file.
func CollectAdvisorySecuritySidecarPaths(packageRoot string) ([]string, error) {
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return nil, err
	}
	sourcePaths, err := CollectAPISourcePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	packageRoot = filepath.Clean(packageRoot)
	var paths []string
	for _, sourcePath := range sourcePaths {
		for _, candidate := range AdvisorySecuritySidecarPathCandidates(sourcePath) {
			clean, err := CleanRelativePath(filepath.ToSlash(candidate))
			if err != nil {
				return nil, err
			}
			abs := filepath.Join(packageRoot, filepath.FromSlash(clean))
			if _, err := os.Lstat(abs); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			paths = append(paths, clean)
		}
	}
	paths, err = uniqueSorted(paths)
	if err != nil {
		return nil, err
	}
	if err := ValidateRegularPackageFiles(packageRoot, paths); err != nil {
		return nil, err
	}
	return paths, nil
}

// AdvisorySecuritySidecarPathCandidates returns the supported sidecar filenames
// next to a source path. The input may be package-relative or absolute.
func AdvisorySecuritySidecarPathCandidates(sourcePath string) []string {
	ext := filepath.Ext(sourcePath)
	base := strings.TrimSuffix(sourcePath, ext)
	return []string{
		sourcePath + ".security.json",
		sourcePath + ".security.yaml",
		base + ".security.json",
		base + ".security.yaml",
		base + ".security-overlay.json",
		base + ".security-overlay.yaml",
	}
}

func collectSourceDirPaths(packageRoot, dir string) ([]string, error) {
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return nil, err
	}
	packageRoot = filepath.Clean(packageRoot)
	sourceRoot := filepath.Join(packageRoot, dir)
	info, err := os.Lstat(sourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s path must not be a symlink: %s", dir, sourceRoot)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s path must be a directory: %s", dir, sourceRoot)
	}

	var paths []string
	if err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if isAdvisorySecuritySidecarPath(path) {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s artifact must not be a symlink: %s", dir, path)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s artifact must be a regular file: %s", dir, path)
		}
		if isAdvisorySecuritySidecarPath(path) {
			return nil
		}
		rel, err := filepath.Rel(packageRoot, path)
		if err != nil {
			return err
		}
		clean, err := CleanRelativePath(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		paths = append(paths, clean)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func isAdvisorySecuritySidecarPath(filePath string) bool {
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	return strings.HasSuffix(base, ".security") || strings.HasSuffix(base, ".security-overlay")
}

// IsAdvisorySecuritySidecarPath reports whether filePath uses a supported
// advisory security-sidecar suffix.
func IsAdvisorySecuritySidecarPath(filePath string) bool {
	return isAdvisorySecuritySidecarPath(filePath)
}

func uniqueSorted(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, inputPath := range paths {
		clean, err := CleanRelativePath(inputPath)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out, nil
}
