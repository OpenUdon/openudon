package packagepipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	GenerationVersion = "openudon.package-generation.v1"
	SelectionVersion  = "openudon.package-selection.v1"

	promotionGenerationsDir = "generations"
	promotionCurrentFile    = "current.json"
	promotionLockFile       = ".openudon-package-promotion.lock"
	promotionStagePrefix    = ".openudon-package-stage-"
	promotionCurrentPrefix  = ".openudon-current-"
)

var (
	promotionBeforeSelectHook func()
	promotionFaultHook        func(PromotionBoundary) error
)

type PromotionBoundary string

const (
	PromotionAfterIntentWrite       PromotionBoundary = "after_intent_write"
	PromotionBeforeGenerationRename PromotionBoundary = "before_generation_rename"
	PromotionAfterGenerationRename  PromotionBoundary = "after_generation_rename"
	PromotionBeforeGenerationSync   PromotionBoundary = "before_generation_sync"
	PromotionAfterGenerationSync    PromotionBoundary = "after_generation_sync"
	PromotionBeforeSelectionRename  PromotionBoundary = "before_selection_rename"
	PromotionAfterSelectionRename   PromotionBoundary = "after_selection_rename"
	PromotionBeforeSelectionSync    PromotionBoundary = "before_selection_sync"
	PromotionAfterSelectionSync     PromotionBoundary = "after_selection_sync"
	PromotionBeforeIntentCleanup    PromotionBoundary = "before_intent_cleanup"
	PromotionBeforeLockCleanup      PromotionBoundary = "before_lock_cleanup"
)

type PromotionState string

const (
	PromotionFailedState           PromotionState = "failed"
	PromotionRolledBackState       PromotionState = "rolled_back"
	PromotionIndeterminateState    PromotionState = "indeterminate"
	PromotionRecoveryRequiredState PromotionState = "recovery_required"
)

type PromotionCode string

const (
	PromotionInvalidQualification PromotionCode = "invalid_qualification"
	PromotionStoreFailed          PromotionCode = "store_failed"
	PromotionBusy                 PromotionCode = "busy"
	PromotionStageFailed          PromotionCode = "stage_failed"
	PromotionGenerationCollision  PromotionCode = "generation_collision"
	PromotionSelectionFailed      PromotionCode = "selection_failed"
	PromotionRecoveryRequired     PromotionCode = "recovery_required"
	PromotionRecoveryDrift        PromotionCode = "recovery_drift"
	PromotionCanceled             PromotionCode = "canceled"
)

// PromotionError is the closed atomic-promotion failure surface. Its public
// text never includes a store path or package byte.
type PromotionError struct {
	Code             PromotionCode
	State            PromotionState
	GenerationSHA256 string
	err              error
}

func (err *PromotionError) Error() string {
	if err == nil {
		return "package promotion failed"
	}
	return fmt.Sprintf("package promotion failed: %s", err.Code)
}

func (err *PromotionError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.err
}

func PromotionFailureCode(err error) (PromotionCode, bool) {
	var typed *PromotionError
	if !errors.As(err, &typed) {
		return "", false
	}
	return typed.Code, true
}

func PromotionFailureState(err error) (PromotionState, bool) {
	var typed *PromotionError
	if !errors.As(err, &typed) {
		return "", false
	}
	return typed.State, true
}

func PromotionFailureGeneration(err error) (string, bool) {
	var typed *PromotionError
	if !errors.As(err, &typed) || !validTaggedSHA256(typed.GenerationSHA256) {
		return "", false
	}
	return typed.GenerationSHA256, true
}

// PromotionOptions selects an existing canonical generation-store root.
type PromotionOptions struct {
	StoreDir string
}

// GenerationRecord binds one immutable on-disk generation to the exact
// preparation and qualification evidence that authorized its publication.
// GenerationSHA256 hashes this structure with that field empty.
type GenerationRecord struct {
	Version          string              `json:"version"`
	Preparation      Manifest            `json:"preparation"`
	Qualification    QualificationReport `json:"qualification"`
	GenerationSHA256 string              `json:"generation_sha256"`
}

// Selection is the atomically replaced, value-free current-generation
// pointer. SelectionSHA256 hashes this structure with that field empty.
type Selection struct {
	Version                  string `json:"version"`
	SelectedGenerationSHA256 string `json:"selected_generation_sha256"`
	PriorGenerationSHA256    string `json:"prior_generation_sha256,omitempty"`
	Scope                    string `json:"scope"`
	PackageSHA256            string `json:"package_sha256"`
	QualificationSHA256      string `json:"qualification_sha256"`
	SelectionSHA256          string `json:"selection_sha256"`
}

// Promoted is one exact selected generation with defensive byte access.
type Promoted struct {
	selection Selection
	files     map[string][]byte
}

func (promoted Promoted) Selection() Selection     { return promoted.selection }
func (promoted Promoted) Files() map[string][]byte { return cloneFiles(promoted.files) }

// Promote publishes a fully qualified immutable generation, then performs
// exactly one atomic selector replacement. Publishing never changes the
// current selection; a selector can name only a completely validated
// generation already present in the store.
func Promote(ctx context.Context, qualified Qualified, opts PromotionOptions) (result Promoted, resultErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Promoted{}, promotionError(PromotionCanceled, err)
	}
	if err := validateQualified(qualified); err != nil {
		return Promoted{}, promotionError(PromotionInvalidQualification, err)
	}
	record, err := buildGenerationRecord(qualified)
	if err != nil {
		return Promoted{}, promotionError(PromotionInvalidQualification, err)
	}
	store, err := canonicalPromotionStore(opts.StoreDir)
	if err != nil {
		return Promoted{}, promotionError(PromotionStoreFailed, err)
	}
	unlock, err := acquirePromotionLock(store, record.GenerationSHA256)
	if err != nil {
		return Promoted{}, err
	}
	skipDeferredUnlock := false
	defer func() {
		if skipDeferredUnlock {
			return
		}
		if err := unlock(); err != nil {
			result = Promoted{}
			resultErr = promotionStateErrorFor(PromotionRecoveryRequired, PromotionIndeterminateState, record.GenerationSHA256, err)
		}
	}()

	if err := ctx.Err(); err != nil {
		return Promoted{}, promotionError(PromotionCanceled, err)
	}
	if recovery, err := recoveryTransients(store); err != nil {
		return Promoted{}, promotionError(PromotionStoreFailed, err)
	} else if len(recovery) != 1 || recovery[0] != promotionLockFile {
		return Promoted{}, promotionStateError(PromotionRecoveryRequired, PromotionRecoveryRequiredState, errors.New("promotion store contains unreconciled transient state"))
	}
	if err := ensureGenerationDirectory(store); err != nil {
		return Promoted{}, promotionError(PromotionStoreFailed, err)
	}
	current, hasCurrent, err := readSelection(ctx, store)
	if err != nil {
		return Promoted{}, promotionError(PromotionSelectionFailed, err)
	}
	if hasCurrent {
		if _, err := loadGeneration(ctx, store, current.SelectedGenerationSHA256); err != nil {
			return Promoted{}, promotionError(PromotionSelectionFailed, errors.New("selected generation is incomplete or invalid"))
		}
		if current.PriorGenerationSHA256 != "" {
			if _, err := loadGeneration(ctx, store, current.PriorGenerationSHA256); err != nil {
				return Promoted{}, promotionError(PromotionSelectionFailed, errors.New("prior generation is incomplete or invalid"))
			}
		}
	}

	if hasCurrent && current.SelectedGenerationSHA256 == record.GenerationSHA256 {
		generation, err := loadGeneration(ctx, store, record.GenerationSHA256)
		if err != nil || !reflect.DeepEqual(generation.record, record) {
			return Promoted{}, promotionError(PromotionGenerationCollision, errors.New("selected generation identity does not match its contents"))
		}
		return Promoted{selection: current, files: generation.files}, nil
	}
	selection, err := buildSelection(record, current, hasCurrent)
	if err != nil {
		return Promoted{}, promotionError(PromotionSelectionFailed, err)
	}
	intent, err := buildPromotionIntent(selection, current, hasCurrent)
	if err != nil {
		return Promoted{}, promotionError(PromotionSelectionFailed, err)
	}
	if err := writePromotionIntent(store, intent); err != nil {
		if _, statErr := os.Lstat(filepath.Join(store, promotionIntentFile)); statErr == nil {
			skipDeferredUnlock = true
			return Promoted{}, promotionStateErrorFor(PromotionRecoveryRequired, PromotionIndeterminateState, record.GenerationSHA256, err)
		}
		return Promoted{}, promotionError(PromotionSelectionFailed, err)
	}
	if err := hitPromotionBoundary(PromotionAfterIntentWrite); err != nil {
		rolledBack, keepLock := rollbackPromotionIntent(store, record.GenerationSHA256, err)
		skipDeferredUnlock = keepLock
		return Promoted{}, rolledBack
	}
	if err := publishGeneration(ctx, store, qualified.Prepared(), record); err != nil {
		rolledBack, keepLock := rollbackPromotionIntent(store, record.GenerationSHA256, err)
		skipDeferredUnlock = keepLock
		return Promoted{}, rolledBack
	}
	if promotionBeforeSelectHook != nil {
		promotionBeforeSelectHook()
	}
	if err := ctx.Err(); err != nil {
		rolledBack, keepLock := rollbackPromotionIntent(store, record.GenerationSHA256, promotionError(PromotionCanceled, err))
		skipDeferredUnlock = keepLock
		return Promoted{}, rolledBack
	}
	selected, err := writeSelection(store, selection)
	if err != nil {
		if selected {
			skipDeferredUnlock = true
			return Promoted{}, promotionStateErrorFor(PromotionSelectionFailed, PromotionIndeterminateState, record.GenerationSHA256, err)
		}
		rolledBack, keepLock := rollbackPromotionIntent(store, record.GenerationSHA256, promotionError(PromotionSelectionFailed, err))
		skipDeferredUnlock = keepLock
		return Promoted{}, rolledBack
	}
	generation, err := loadGeneration(ctx, store, record.GenerationSHA256)
	if err != nil {
		skipDeferredUnlock = true
		return Promoted{}, promotionStateErrorFor(PromotionSelectionFailed, PromotionIndeterminateState, record.GenerationSHA256, errors.New("selected generation failed post-selection validation"))
	}
	if err := unlock(); err != nil {
		skipDeferredUnlock = true
		return Promoted{}, promotionStateErrorFor(PromotionRecoveryRequired, PromotionIndeterminateState, record.GenerationSHA256, err)
	}
	skipDeferredUnlock = true
	if err := removePromotionIntent(store); err != nil {
		return Promoted{}, promotionStateErrorFor(PromotionRecoveryRequired, PromotionIndeterminateState, record.GenerationSHA256, err)
	}
	return Promoted{selection: selection, files: generation.files}, nil
}

// ReadCurrent returns either the complete old or complete new generation
// across a concurrent promotion. It never consults a staging directory.
func ReadCurrent(ctx context.Context, storeDir string) (Promoted, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Promoted{}, promotionError(PromotionCanceled, err)
	}
	store, err := canonicalPromotionStore(storeDir)
	if err != nil {
		return Promoted{}, promotionError(PromotionStoreFailed, err)
	}
	selection, ok, err := readSelection(ctx, store)
	if err != nil {
		return Promoted{}, promotionError(PromotionSelectionFailed, err)
	}
	if !ok {
		return Promoted{}, promotionError(PromotionSelectionFailed, errors.New("package store has no current selection"))
	}
	generation, err := loadGeneration(ctx, store, selection.SelectedGenerationSHA256)
	if err != nil || generation.record.Preparation.Scope != selection.Scope || generation.record.Preparation.PackageSHA256 != selection.PackageSHA256 || generation.record.Qualification.QualificationSHA256 != selection.QualificationSHA256 {
		return Promoted{}, promotionError(PromotionSelectionFailed, errors.New("current selection does not resolve to its exact complete generation"))
	}
	return Promoted{selection: selection, files: generation.files}, nil
}

type loadedGeneration struct {
	record GenerationRecord
	files  map[string][]byte
}

func buildGenerationRecord(qualified Qualified) (GenerationRecord, error) {
	record := GenerationRecord{
		Version: GenerationVersion, Preparation: qualified.Prepared().Manifest(), Qualification: qualified.Report(),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return GenerationRecord{}, err
	}
	digest := sha256.Sum256(data)
	record.GenerationSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return record, nil
}

func validateGenerationRecord(record GenerationRecord) error {
	want := record.GenerationSHA256
	if !validTaggedSHA256(want) || record.Version != GenerationVersion {
		return errors.New("generation record has unsupported identity")
	}
	record.GenerationSHA256 = ""
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if want != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("generation record digest mismatch")
	}
	return nil
}

func buildSelection(record GenerationRecord, current Selection, hasCurrent bool) (Selection, error) {
	selection := Selection{
		Version: SelectionVersion, SelectedGenerationSHA256: record.GenerationSHA256,
		Scope: record.Preparation.Scope, PackageSHA256: record.Preparation.PackageSHA256,
		QualificationSHA256: record.Qualification.QualificationSHA256,
	}
	if hasCurrent {
		selection.PriorGenerationSHA256 = current.SelectedGenerationSHA256
	}
	data, err := json.Marshal(selection)
	if err != nil {
		return Selection{}, err
	}
	digest := sha256.Sum256(data)
	selection.SelectionSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return selection, nil
}

func validateSelection(selection Selection) error {
	want := selection.SelectionSHA256
	if selection.Version != SelectionVersion || !validTaggedSHA256(selection.SelectedGenerationSHA256) || !validTaggedSHA256(want) {
		return errors.New("selection has unsupported identity")
	}
	if selection.PriorGenerationSHA256 != "" && !validTaggedSHA256(selection.PriorGenerationSHA256) {
		return errors.New("selection has invalid prior generation identity")
	}
	if selection.PriorGenerationSHA256 == selection.SelectedGenerationSHA256 {
		return errors.New("selection cannot name the selected generation as prior")
	}
	if _, err := canonicalScope(selection.Scope); err != nil || !validUntaggedSHA256(selection.PackageSHA256) || !validTaggedSHA256(selection.QualificationSHA256) {
		return errors.New("selection has invalid package identity")
	}
	selection.SelectionSHA256 = ""
	data, err := json.Marshal(selection)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if want != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("selection digest mismatch")
	}
	return nil
}

func validateQualified(qualified Qualified) error {
	if err := validatePrepared(qualified.prepared); err != nil {
		return err
	}
	return validateQualificationReport(qualified.Report(), qualified.Prepared().Manifest())
}

func validateQualificationReport(report QualificationReport, manifest Manifest) error {
	if report.Version != QualificationVersion || report.PreparationSHA256 != manifest.ManifestSHA256 || report.InputSHA256 != manifest.InputSHA256 || report.PackageSHA256 != manifest.PackageSHA256 || report.QualitySHA256 != manifest.QualitySHA256 || !validTaggedSHA256(report.QualificationSHA256) {
		return errors.New("qualification report does not bind the prepared package")
	}
	if report.ScratchPosture != qualificationScratchPosture || report.NetworkPosture != qualificationNetworkPosture || report.DryRunPosture != qualificationDryRunPosture || !reflect.DeepEqual(report.Gates, qualificationPassingGates()) {
		return errors.New("qualification report does not contain the exact passing gate posture")
	}
	want := report.QualificationSHA256
	report.QualificationSHA256 = ""
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if want != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("qualification report digest mismatch")
	}
	return nil
}

func publishGeneration(ctx context.Context, store string, prepared Prepared, record GenerationRecord) error {
	final := generationPath(store, record.GenerationSHA256)
	if _, err := os.Lstat(final); err == nil {
		loaded, loadErr := loadGeneration(ctx, store, record.GenerationSHA256)
		if loadErr != nil || !reflect.DeepEqual(loaded.record, record) {
			return promotionError(PromotionGenerationCollision, errors.New("generation identity is already occupied by different or invalid contents"))
		}
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return promotionError(PromotionStageFailed, errors.New("inspect generation destination"))
	}

	stage, err := os.MkdirTemp(store, promotionStagePrefix)
	if err != nil {
		return promotionError(PromotionStageFailed, errors.New("create promotion staging directory"))
	}
	if err := os.Chmod(stage, 0o700); err != nil {
		_ = removePromotionStage(store, stage)
		return promotionError(PromotionStageFailed, errors.New("restrict promotion staging directory"))
	}
	removeStage := true
	defer func() {
		if removeStage {
			_ = removePromotionStage(store, stage)
		}
	}()
	packageBase := filepath.Join(stage, "package")
	if err := os.Mkdir(packageBase, 0o700); err != nil {
		return promotionError(PromotionStageFailed, errors.New("create staged package root"))
	}
	if err := materializePrepared(packageBase, prepared.Manifest().Scope, prepared.Files()); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	if err := validateScratchPackage(filepath.Join(packageBase, filepath.FromSlash(prepared.Manifest().Scope)), prepared.Manifest()); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	recordBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	recordBytes = append(recordBytes, '\n')
	if err := writeSyncedExclusive(filepath.Join(stage, "generation.json"), recordBytes, 0o600); err != nil {
		return promotionError(PromotionStageFailed, errors.New("write generation record"))
	}
	if err := syncTreeDirectories(stage); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	if err := ctx.Err(); err != nil {
		return promotionError(PromotionCanceled, err)
	}
	if err := hitPromotionBoundary(PromotionBeforeGenerationRename); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	if err := os.Rename(stage, final); err != nil {
		if loaded, loadErr := loadGeneration(ctx, store, record.GenerationSHA256); loadErr == nil && reflect.DeepEqual(loaded.record, record) {
			return nil
		}
		return promotionError(PromotionGenerationCollision, errors.New("publish generation without overwrite"))
	}
	removeStage = false
	if err := hitPromotionBoundary(PromotionAfterGenerationRename); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	if err := hitPromotionBoundary(PromotionBeforeGenerationSync); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	if err := syncDirectory(filepath.Join(store, promotionGenerationsDir)); err != nil {
		return promotionError(PromotionStageFailed, errors.New("synchronize generation directory"))
	}
	if err := hitPromotionBoundary(PromotionAfterGenerationSync); err != nil {
		return promotionError(PromotionStageFailed, err)
	}
	loaded, err := loadGeneration(ctx, store, record.GenerationSHA256)
	if err != nil || !reflect.DeepEqual(loaded.record, record) {
		return promotionError(PromotionStageFailed, errors.New("published generation failed complete validation"))
	}
	return nil
}

func loadGeneration(ctx context.Context, store, taggedDigest string) (loadedGeneration, error) {
	root := generationPath(store, taggedDigest)
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return loadedGeneration{}, errors.New("generation root is unavailable or unsafe")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 || entries[0].Name() != "generation.json" || entries[1].Name() != "package" || !entries[1].IsDir() {
		return loadedGeneration{}, errors.New("generation root inventory is invalid")
	}
	recordPath := filepath.Join(root, "generation.json")
	recordBytes, info, err := evidencefile.ReadRegular(recordPath, evidencefile.DefaultMaxBytes)
	if err != nil || info.Mode().Perm() != 0o600 {
		return loadedGeneration{}, errors.New("generation record is unavailable or unsafe")
	}
	if links, ok := fileLinkCount(info); !ok || links != 1 {
		return loadedGeneration{}, errors.New("generation record link count is unsafe or unavailable")
	}
	var record GenerationRecord
	if err := evidencefile.DecodeStrict(recordBytes, &record); err != nil || validateGenerationRecord(record) != nil || record.GenerationSHA256 != taggedDigest || validateQualificationReport(record.Qualification, record.Preparation) != nil {
		return loadedGeneration{}, errors.New("generation record is invalid")
	}
	if err := validateManifestIdentity(record.Preparation); err != nil {
		return loadedGeneration{}, err
	}
	packageBase := filepath.Join(root, "package")
	packageInfo, err := os.Lstat(packageBase)
	if err != nil || !packageInfo.IsDir() || packageInfo.Mode()&os.ModeSymlink != 0 || packageInfo.Mode().Perm() != 0o700 {
		return loadedGeneration{}, errors.New("generation package root is unsafe")
	}
	packageRoot := filepath.Join(packageBase, filepath.FromSlash(record.Preparation.Scope))
	if err := validateScratchPackage(packageRoot, record.Preparation); err != nil {
		return loadedGeneration{}, err
	}
	prepared, err := PrepareCurrent(ctx, PrepareOptions{ExampleDir: packageRoot, Scope: record.Preparation.Scope, ExpectedInputSHA256: record.Preparation.InputSHA256})
	if err != nil || !reflect.DeepEqual(prepared.Manifest(), record.Preparation) {
		return loadedGeneration{}, errors.New("generation package differs from its preparation record")
	}
	return loadedGeneration{record: record, files: prepared.Files()}, nil
}

func validateManifestIdentity(manifest Manifest) error {
	if manifest.Version != PreparationVersion || !validTaggedSHA256(manifest.ManifestSHA256) || !validUntaggedSHA256(manifest.InputSHA256) || !validUntaggedSHA256(manifest.PackageSHA256) || !validUntaggedSHA256(manifest.HandoffSHA256) || !validUntaggedSHA256(manifest.QualitySHA256) || len(manifest.Files) == 0 {
		return errors.New("generation preparation manifest has invalid identity fields")
	}
	if _, err := canonicalScope(manifest.Scope); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, file := range manifest.Files {
		canonical, err := canonicalFilePath(file.Path)
		if err != nil || seen[canonical] || !validUntaggedSHA256(file.SHA256) || file.Bytes < 0 {
			return errors.New("generation preparation manifest has invalid file identity")
		}
		seen[canonical] = true
	}
	return nil
}

func canonicalFilePath(value string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." || clean != value || strings.HasPrefix(clean, "../") || filepath.IsAbs(value) {
		return "", errors.New("invalid relative file path")
	}
	return clean, nil
}

func readSelection(ctx context.Context, store string) (Selection, bool, error) {
	if err := ctx.Err(); err != nil {
		return Selection{}, false, err
	}
	path := filepath.Join(store, promotionCurrentFile)
	data, info, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return Selection{}, false, nil
	}
	if err != nil || info.Mode().Perm() != 0o600 {
		return Selection{}, false, errors.New("current selection is unavailable or unsafe")
	}
	if links, ok := fileLinkCount(info); !ok || links != 1 {
		return Selection{}, false, errors.New("current selection link count is unsafe or unavailable")
	}
	var selection Selection
	if err := evidencefile.DecodeStrict(data, &selection); err != nil || validateSelection(selection) != nil {
		return Selection{}, false, errors.New("current selection is invalid")
	}
	return selection, true, nil
}

func writeSelection(store string, selection Selection) (bool, error) {
	data, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(store, promotionCurrentPrefix)
	if err != nil {
		return false, errors.New("create selection staging file")
	}
	tmpPath := tmp.Name()
	remove := true
	defer func() {
		_ = tmp.Close()
		if remove {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return false, errors.New("restrict selection staging file")
	}
	if _, err := io.Copy(tmp, bytes.NewReader(data)); err != nil {
		return false, errors.New("write selection staging file")
	}
	if err := tmp.Sync(); err != nil {
		return false, errors.New("synchronize selection staging file")
	}
	if err := tmp.Close(); err != nil {
		return false, errors.New("close selection staging file")
	}
	if err := hitPromotionBoundary(PromotionBeforeSelectionRename); err != nil {
		return false, err
	}
	if err := os.Rename(tmpPath, filepath.Join(store, promotionCurrentFile)); err != nil {
		return false, errors.New("atomically replace current selection")
	}
	remove = false
	if err := hitPromotionBoundary(PromotionAfterSelectionRename); err != nil {
		return true, err
	}
	if err := hitPromotionBoundary(PromotionBeforeSelectionSync); err != nil {
		return true, err
	}
	if err := syncDirectory(store); err != nil {
		return true, errors.New("synchronize current selection directory")
	}
	if err := hitPromotionBoundary(PromotionAfterSelectionSync); err != nil {
		return true, err
	}
	return true, nil
}

func canonicalPromotionStore(value string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(value))
	if clean == "." || !filepath.IsAbs(clean) {
		return "", errors.New("promotion store must be an absolute canonical directory")
	}
	resolved, err := filepath.EvalSymlinks(clean)
	info, statErr := os.Lstat(clean)
	if err != nil || statErr != nil || resolved != clean || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("promotion store must be an existing canonical non-symlink directory")
	}
	return clean, nil
}

func ensureGenerationDirectory(store string) error {
	path := filepath.Join(store, promotionGenerationsDir)
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return errors.New("create generation directory")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return errors.New("generation directory is unavailable or unsafe")
	}
	return syncDirectory(store)
}

func acquirePromotionLock(store, targetGeneration string) (func() error, error) {
	path := filepath.Join(store, promotionLockFile)
	lock, err := buildPromotionLock(targetGeneration)
	if err != nil {
		return nil, promotionError(PromotionStoreFailed, err)
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err == nil {
		err = writeAtomicNew(path, append(data, '\n'), 0o600, promotionLockPrefix)
	}
	if errors.Is(err, fs.ErrExist) {
		return nil, promotionError(PromotionBusy, errors.New("another promotion owns the store lock"))
	}
	if err != nil {
		if transients, inspectErr := recoveryTransients(store); inspectErr == nil && len(transients) > 0 {
			return nil, promotionStateError(PromotionRecoveryRequired, PromotionIndeterminateState, errors.New("promotion lock creation left recoverable transient state"))
		}
		return nil, promotionError(PromotionStoreFailed, errors.New("create promotion lock"))
	}
	return func() error {
		if err := hitPromotionBoundary(PromotionBeforeLockCleanup); err != nil {
			return err
		}
		root, err := os.OpenRoot(store)
		if err != nil {
			return errors.New("open promotion store for lock cleanup")
		}
		defer root.Close()
		if err := root.Remove(promotionLockFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return errors.New("remove promotion lock")
		}
		return syncDirectory(store)
	}, nil
}

func generationPath(store, taggedDigest string) string {
	if !validTaggedSHA256(taggedDigest) {
		return filepath.Join(store, promotionGenerationsDir, "invalid")
	}
	return filepath.Join(store, promotionGenerationsDir, strings.TrimPrefix(taggedDigest, "sha256:"))
}

func removePromotionStage(store, stage string) error {
	relative, err := filepath.Rel(store, stage)
	if err != nil || filepath.Dir(relative) != "." || !strings.HasPrefix(filepath.Base(relative), promotionStagePrefix) {
		return errors.New("refuse unsafe promotion staging cleanup target")
	}
	root, err := os.OpenRoot(store)
	if err != nil {
		return errors.New("open promotion store for staging cleanup")
	}
	defer root.Close()
	return root.RemoveAll(filepath.Base(relative))
}

func writeSyncedExclusive(path string, data []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(data)
	syncErr, closeErr := file.Sync(), file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || written != len(data) {
		return errors.New("write and synchronize complete file")
	}
	return nil
}

func syncTreeDirectories(root string) error {
	var directories []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			directories = append(directories, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, directory := range directories {
		if err := syncDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

func syncDirectory(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	err = file.Sync()
	closeErr := file.Close()
	if err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) && !errors.Is(err, syscall.EBADF) {
		return err
	}
	return closeErr
}

func validTaggedSHA256(value string) bool {
	return strings.HasPrefix(value, "sha256:") && validUntaggedSHA256(strings.TrimPrefix(value, "sha256:"))
}

func validUntaggedSHA256(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func promotionError(code PromotionCode, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = PromotionCanceled
	}
	if typed := new(PromotionError); errors.As(err, &typed) {
		return err
	}
	return &PromotionError{Code: code, State: PromotionFailedState, err: err}
}

func promotionStateError(code PromotionCode, state PromotionState, err error) error {
	return promotionStateErrorFor(code, state, "", err)
}

func promotionStateErrorFor(code PromotionCode, state PromotionState, generation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = PromotionCanceled
	}
	return &PromotionError{Code: code, State: state, GenerationSHA256: generation, err: err}
}

func promotionCodeOr(err error, fallback PromotionCode) PromotionCode {
	if code, ok := PromotionFailureCode(err); ok {
		return code
	}
	return fallback
}

func rollbackPromotionIntent(store, generation string, cause error) (error, bool) {
	if err := removePromotionIntent(store); err != nil {
		return promotionStateErrorFor(PromotionRecoveryRequired, PromotionIndeterminateState, generation, errors.Join(cause, err)), true
	}
	return promotionStateErrorFor(promotionCodeOr(cause, PromotionSelectionFailed), PromotionRolledBackState, generation, cause), false
}

func hitPromotionBoundary(boundary PromotionBoundary) error {
	if promotionFaultHook == nil {
		return nil
	}
	return promotionFaultHook(boundary)
}
