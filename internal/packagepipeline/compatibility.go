package packagepipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

// CurrentOptions binds the prepare/qualify/promote lifecycle to one existing
// reviewed package and caller-owned scratch/store directories.
type CurrentOptions struct {
	ExampleDir          string
	Scope               string
	ExpectedInputSHA256 string
	ScratchParent       string
	StoreDir            string
}

// PrepareAndQualifyCurrent applies the write-free preparation boundary and
// restrictive qualification boundary to one existing package generation.
func PrepareAndQualifyCurrent(ctx context.Context, opts CurrentOptions) (Qualified, error) {
	prepared, err := PrepareCurrent(ctx, PrepareOptions{
		ExampleDir: opts.ExampleDir, Scope: opts.Scope, ExpectedInputSHA256: opts.ExpectedInputSHA256,
	})
	if err != nil {
		return Qualified{}, err
	}
	return Qualify(ctx, prepared, QualifyOptions{ScratchParent: opts.ScratchParent})
}

// PromoteCurrent re-prepares and qualifies the exact current bytes before
// atomically selecting them. It performs no runtime handoff.
func PromoteCurrent(ctx context.Context, opts CurrentOptions) (Promoted, error) {
	qualified, err := PrepareAndQualifyCurrent(ctx, opts)
	if err != nil {
		return Promoted{}, err
	}
	return Promote(ctx, qualified, PromotionOptions{StoreDir: opts.StoreDir})
}

// InspectSelected routes package review through one exact promoted selector.
func InspectSelected(ctx context.Context, storeDir, expectedSelectionSHA256 string) (trustedrunner.PackageInspection, error) {
	selected, err := resolveSelectedPackage(ctx, storeDir, expectedSelectionSHA256)
	if err != nil {
		return trustedrunner.PackageInspection{}, err
	}
	inspection, err := trustedrunner.InspectPackage(ctx, trustedrunner.TemplateOptions{
		RepoRoot: selected.packageBase, ExampleDir: selected.packageRoot,
	})
	if err != nil {
		return trustedrunner.PackageInspection{}, err
	}
	if inspection.Scope != selected.selection.Scope || inspection.PackageSHA256 != selected.selection.PackageSHA256 || inspection.HandoffSHA256 != selected.generation.record.Preparation.HandoffSHA256 {
		return trustedrunner.PackageInspection{}, promotionError(PromotionSelectionFailed, errors.New("selected package inspection differs from the exact selector"))
	}
	return inspection, nil
}

// ApprovalTemplateSelected creates an approval for one exact promoted
// selector. The selected generation is revalidated before template creation.
func ApprovalTemplateSelected(ctx context.Context, storeDir, expectedSelectionSHA256 string, opts trustedrunner.TemplateOptions) (trustedrunner.Approval, error) {
	selected, err := resolveSelectedPackage(ctx, storeDir, expectedSelectionSHA256)
	if err != nil {
		return trustedrunner.Approval{}, err
	}
	opts.RepoRoot, opts.ExampleDir = selected.packageBase, selected.packageRoot
	opts.Assess = nil
	approval, err := trustedrunner.ApprovalTemplate(ctx, opts)
	if err != nil {
		return trustedrunner.Approval{}, err
	}
	if approval.Scope != selected.selection.Scope || approval.PackageSHA256 != selected.selection.PackageSHA256 {
		return trustedrunner.Approval{}, promotionError(PromotionSelectionFailed, errors.New("selected approval template differs from the exact selector"))
	}
	return approval, nil
}

// RunSelected performs the existing trusted-runner handoff against one exact
// promoted generation. Dry-run still never invokes an executor; non-dry
// authority and all existing registration rejection remain in trustedrunner.
func RunSelected(ctx context.Context, storeDir, expectedSelectionSHA256 string, opts trustedrunner.Options) (*trustedrunner.RunResult, error) {
	selected, err := resolveSelectedPackage(ctx, storeDir, expectedSelectionSHA256)
	if err != nil {
		return nil, err
	}
	store, err := canonicalPromotionStore(storeDir)
	if err != nil {
		return nil, promotionError(PromotionStoreFailed, err)
	}
	workDir, err := canonicalProspectivePath(opts.WorkDir)
	if err != nil || pathWithin(store, workDir) {
		return nil, promotionError(PromotionSelectionFailed, errors.New("selected trusted-run requires an explicit work directory outside the generation store"))
	}
	approvalPath, err := canonicalProspectivePath(opts.ApprovalPath)
	if err != nil || pathWithin(store, approvalPath) {
		return nil, promotionError(PromotionSelectionFailed, errors.New("selected trusted-run requires an approval outside the generation store"))
	}
	opts.WorkDir, opts.ApprovalPath = workDir, approvalPath
	opts.RepoRoot, opts.ExampleDir = selected.packageBase, selected.packageRoot
	opts.Assess = nil
	result, err := trustedrunner.Run(ctx, opts)
	if err != nil {
		return result, err
	}
	if result == nil || result.Scope != selected.selection.Scope || result.PackageSHA256 != selected.selection.PackageSHA256 {
		return nil, promotionError(PromotionSelectionFailed, errors.New("selected trusted-run result differs from the exact selector"))
	}
	return result, nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative)
}

func canonicalProspectivePath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("path is required")
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	cursor := abs
	var suffix []string
	for {
		_, err := os.Lstat(cursor)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) || filepath.Dir(cursor) == cursor {
			return "", errors.New("path has no resolvable existing ancestor")
		}
		suffix = append(suffix, filepath.Base(cursor))
		cursor = filepath.Dir(cursor)
	}
	resolved, err := filepath.EvalSymlinks(cursor)
	if err != nil {
		return "", err
	}
	for index := len(suffix) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, suffix[index])
	}
	return filepath.Clean(resolved), nil
}

type selectedPackage struct {
	selection   Selection
	generation  loadedGeneration
	packageBase string
	packageRoot string
}

func resolveSelectedPackage(ctx context.Context, storeDir, expectedSelectionSHA256 string) (selectedPackage, error) {
	store, err := canonicalPromotionStore(storeDir)
	if err != nil {
		return selectedPackage{}, promotionError(PromotionStoreFailed, err)
	}
	expected := strings.TrimSpace(expectedSelectionSHA256)
	if !validTaggedSHA256(expected) {
		return selectedPackage{}, promotionError(PromotionSelectionFailed, errors.New("exact selected-generation observation digest is required"))
	}
	selection, ok, err := readSelection(ctx, store)
	if err != nil || !ok || selection.SelectionSHA256 != expected {
		return selectedPackage{}, promotionError(PromotionSelectionFailed, errors.New("selected generation changed or is unavailable"))
	}
	generation, err := loadGeneration(ctx, store, selection.SelectedGenerationSHA256)
	if err != nil || generation.record.Preparation.Scope != selection.Scope || generation.record.Preparation.PackageSHA256 != selection.PackageSHA256 || generation.record.Qualification.QualificationSHA256 != selection.QualificationSHA256 {
		return selectedPackage{}, promotionError(PromotionSelectionFailed, errors.New("selected generation does not resolve to its exact package"))
	}
	if selection.PriorGenerationSHA256 != "" {
		if _, err := loadGeneration(ctx, store, selection.PriorGenerationSHA256); err != nil {
			return selectedPackage{}, promotionError(PromotionSelectionFailed, errors.New("selected generation has an invalid prior generation"))
		}
	}
	packageBase := filepath.Join(generationPath(store, selection.SelectedGenerationSHA256), "package")
	packageRoot := filepath.Join(packageBase, filepath.FromSlash(selection.Scope))
	prepared, err := PrepareCurrent(ctx, PrepareOptions{ExampleDir: packageRoot, Scope: selection.Scope, ExpectedInputSHA256: generation.record.Preparation.InputSHA256})
	if err != nil || !reflect.DeepEqual(prepared.Manifest(), generation.record.Preparation) {
		return selectedPackage{}, promotionError(PromotionSelectionFailed, errors.New("selected package changed during compatibility revalidation"))
	}
	return selectedPackage{selection: selection, generation: generation, packageBase: packageBase, packageRoot: packageRoot}, nil
}
