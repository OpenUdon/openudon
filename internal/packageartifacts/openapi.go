package packageartifacts

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestInput is the subset of review handoff input metadata needed to
// validate required package inventory.
type ManifestInput struct {
	Path     string
	Required bool
}

var fixedRequiredPackagePaths = []string{
	"project.md",
	"workflows/intent.hcl",
	"workflows/workflow.hcl",
	"workflows/workflow.uws.yaml",
	"expected/plan.json",
	"expected/quality.json",
	"expected/refinement.json",
	"expected/review.md",
	"expected/symphony-handoff.json",
}

// CleanRelativePath returns a canonical slash-separated package-relative path.
func CleanRelativePath(inputPath string) (string, error) {
	for _, r := range inputPath {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("path must not contain control characters: %q", inputPath)
		}
	}
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return "", fmt.Errorf("path must be non-empty")
	}
	if strings.Contains(inputPath, `\`) {
		return "", fmt.Errorf("path must use slash separators: %q", inputPath)
	}
	if path.IsAbs(inputPath) || strings.HasPrefix(inputPath, "/") {
		return "", fmt.Errorf("path must be relative: %q", inputPath)
	}
	if hasWindowsVolumeName(inputPath) {
		return "", fmt.Errorf("path must not include a volume prefix: %q", inputPath)
	}
	for _, segment := range strings.Split(inputPath, "/") {
		if segment == ".." {
			return "", fmt.Errorf("path must not contain '..' segments: %q", inputPath)
		}
	}
	clean := path.Clean(inputPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path must stay inside package root: %q", inputPath)
	}
	return clean, nil
}

var apiSourceDirs = []string{"openapi", "google-discovery", "aws-smithy", "discovery"}

// RequiredPackagePaths returns the fixed handoff inventory plus every regular
// API source file staged for execution.
func RequiredPackagePaths(packageRoot string) ([]string, error) {
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return nil, err
	}
	paths := append([]string(nil), fixedRequiredPackagePaths...)
	openAPIPaths, err := CollectAPISourcePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	paths = append(paths, openAPIPaths...)
	return uniqueSorted(paths)
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
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return err
	}
	packageRoot = filepath.Clean(packageRoot)
	for _, inputPath := range paths {
		clean, err := CleanRelativePath(inputPath)
		if err != nil {
			return fmt.Errorf("package path %q is unsafe: %w", inputPath, err)
		}
		if err := validateRegularPackageFile(packageRoot, clean); err != nil {
			return err
		}
	}
	return nil
}

func ValidatePackageRoot(packageRoot string) error {
	packageRoot = strings.TrimSpace(packageRoot)
	if packageRoot == "" {
		return fmt.Errorf("package root is required")
	}
	packageRoot = filepath.Clean(packageRoot)
	info, err := os.Lstat(packageRoot)
	if err != nil {
		return fmt.Errorf("package root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("package root must not be a symlink: %s", packageRoot)
	}
	if !info.IsDir() {
		return fmt.Errorf("package root must be a directory: %s", packageRoot)
	}
	return nil
}

func validateRegularPackageFile(packageRoot, clean string) error {
	segments := strings.Split(clean, "/")
	current := packageRoot
	for i, segment := range segments {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("required handoff input %s: %w", clean, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("required handoff input must not be a symlink: %s", clean)
		}
		last := i == len(segments)-1
		if !last {
			if !info.IsDir() {
				return fmt.Errorf("required handoff input parent must be a directory: %s", clean)
			}
			continue
		}
		if info.IsDir() {
			return fmt.Errorf("required handoff input must be a regular file, not a directory: %s", clean)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("required handoff input must be a regular file: %s", clean)
		}
	}
	return nil
}

// CollectOpenAPIPaths returns package-relative OpenAPI artifact paths.
func CollectOpenAPIPaths(packageRoot string) ([]string, error) {
	return collectSourceDirPaths(packageRoot, "openapi")
}

// CollectAPISourcePaths returns package-relative API source artifact paths.
func CollectAPISourcePaths(packageRoot string) ([]string, error) {
	if err := ValidatePackageRoot(packageRoot); err != nil {
		return nil, err
	}
	var paths []string
	for _, dir := range apiSourceDirs {
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

func hasWindowsVolumeName(inputPath string) bool {
	if len(inputPath) < 2 || inputPath[1] != ':' {
		return false
	}
	letter := inputPath[0]
	return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
}
