package packagepipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	PromotionIntentVersion = "openudon.package-promotion-intent.v1"
	PromotionLockVersion   = "openudon.package-promotion-lock.v1"
	RecoveryReportVersion  = "openudon.package-recovery-report.v1"
	ReconciliationVersion  = "openudon.package-reconciliation.v1"

	promotionIntentFile   = ".openudon-package-promotion-intent.json"
	promotionLockPrefix   = ".openudon-lock-"
	promotionIntentPrefix = ".openudon-intent-"
	maxRecoveryTransients = 128
)

type PromotionLock struct {
	Version                string `json:"version"`
	TargetGenerationSHA256 string `json:"target_generation_sha256"`
	LockSHA256             string `json:"lock_sha256"`
}

type PromotionIntent struct {
	Version                 string    `json:"version"`
	TargetGenerationSHA256  string    `json:"target_generation_sha256"`
	BaselineSelectionSHA256 string    `json:"baseline_selection_sha256,omitempty"`
	TargetSelection         Selection `json:"target_selection"`
	IntentSHA256            string    `json:"intent_sha256"`
}

type RecoveryResolution string

const (
	RecoveryClean      RecoveryResolution = "clean"
	RecoveryRolledBack RecoveryResolution = "rolled_back"
	RecoveryPromoted   RecoveryResolution = "promoted"
	RecoveryDrift      RecoveryResolution = "drift"
)

// RecoveryReport is a value-free, immutable observation that must be supplied
// back by digest before any transient cleanup occurs.
type RecoveryReport struct {
	Version                          string             `json:"version"`
	Resolution                       RecoveryResolution `json:"resolution"`
	TargetGenerationSHA256           string             `json:"target_generation_sha256,omitempty"`
	BaselineSelectionSHA256          string             `json:"baseline_selection_sha256,omitempty"`
	ObservedSelectionSHA256          string             `json:"observed_selection_sha256,omitempty"`
	ObservedSelectedGenerationSHA256 string             `json:"observed_selected_generation_sha256,omitempty"`
	ObservedPriorGenerationSHA256    string             `json:"observed_prior_generation_sha256,omitempty"`
	TargetGenerationStatus           string             `json:"target_generation_status,omitempty"`
	LockPresent                      bool               `json:"lock_present"`
	IntentPresent                    bool               `json:"intent_present"`
	TransientCount                   int                `json:"transient_count"`
	RecoverySHA256                   string             `json:"recovery_sha256"`
}

type ReconcileOptions struct {
	StoreDir               string
	ExpectedRecoverySHA256 string
}

type Reconciliation struct {
	Version                  string             `json:"version"`
	Resolution               RecoveryResolution `json:"resolution"`
	TargetGenerationSHA256   string             `json:"target_generation_sha256,omitempty"`
	SelectedGenerationSHA256 string             `json:"selected_generation_sha256,omitempty"`
	PriorGenerationSHA256    string             `json:"prior_generation_sha256,omitempty"`
	ObservedRecoverySHA256   string             `json:"observed_recovery_sha256"`
}

// InspectRecovery deterministically classifies an interrupted promotion. It
// is read-only and validates the selected and immediately prior generations.
func InspectRecovery(ctx context.Context, storeDir string) (RecoveryReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return RecoveryReport{}, promotionError(PromotionCanceled, err)
	}
	store, err := canonicalPromotionStore(storeDir)
	if err != nil {
		return RecoveryReport{}, promotionError(PromotionStoreFailed, err)
	}
	return inspectRecovery(ctx, store)
}

// Reconcile removes only exact bounded promotion transients after rechecking
// an operator-observed report digest. It never changes current.json and never
// deletes any generation.
func Reconcile(ctx context.Context, opts ReconcileOptions) (Reconciliation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := canonicalPromotionStore(opts.StoreDir)
	if err != nil {
		return Reconciliation{}, promotionError(PromotionStoreFailed, err)
	}
	report, err := inspectRecovery(ctx, store)
	if err != nil {
		return Reconciliation{}, err
	}
	if !validTaggedSHA256(opts.ExpectedRecoverySHA256) || opts.ExpectedRecoverySHA256 != report.RecoverySHA256 {
		return Reconciliation{}, promotionStateError(PromotionRecoveryDrift, PromotionRecoveryRequiredState, errors.New("recovery observation changed or was not explicitly accepted"))
	}
	if report.Resolution == RecoveryDrift {
		return Reconciliation{}, promotionStateError(PromotionRecoveryDrift, PromotionRecoveryRequiredState, errors.New("promotion recovery observed unresolvable store drift"))
	}
	if err := ctx.Err(); err != nil {
		return Reconciliation{}, promotionError(PromotionCanceled, err)
	}
	confirmed, err := inspectRecovery(ctx, store)
	if err != nil || !reflect.DeepEqual(confirmed, report) {
		return Reconciliation{}, promotionStateError(PromotionRecoveryDrift, PromotionRecoveryRequiredState, errors.New("promotion recovery state changed before reconciliation"))
	}
	if report.Resolution != RecoveryClean {
		if err := cleanupRecoveryTransients(store); err != nil {
			return Reconciliation{}, promotionStateError(PromotionRecoveryRequired, PromotionIndeterminateState, err)
		}
	}
	after, err := inspectRecovery(ctx, store)
	if err != nil || after.Resolution != RecoveryClean {
		return Reconciliation{}, promotionStateError(PromotionRecoveryRequired, PromotionIndeterminateState, errors.New("promotion recovery cleanup did not reach a clean state"))
	}
	return Reconciliation{
		Version: ReconciliationVersion, Resolution: report.Resolution,
		TargetGenerationSHA256:   report.TargetGenerationSHA256,
		SelectedGenerationSHA256: report.ObservedSelectedGenerationSHA256,
		PriorGenerationSHA256:    report.ObservedPriorGenerationSHA256,
		ObservedRecoverySHA256:   report.RecoverySHA256,
	}, nil
}

func inspectRecovery(ctx context.Context, store string) (RecoveryReport, error) {
	transients, err := recoveryTransients(store)
	if err != nil {
		return RecoveryReport{}, promotionStateError(PromotionRecoveryDrift, PromotionRecoveryRequiredState, err)
	}
	lock, hasLock, lockErr := readPromotionLock(store)
	intent, hasIntent, intentErr := readPromotionIntent(store)
	selection, hasSelection, selectionErr := readSelection(ctx, store)
	report := RecoveryReport{
		Version: RecoveryReportVersion, Resolution: RecoveryClean,
		LockPresent: hasLock, IntentPresent: hasIntent, TransientCount: len(transients),
	}
	if hasSelection {
		report.ObservedSelectionSHA256 = selection.SelectionSHA256
		report.ObservedSelectedGenerationSHA256 = selection.SelectedGenerationSHA256
		report.ObservedPriorGenerationSHA256 = selection.PriorGenerationSHA256
	}
	if lockErr != nil || intentErr != nil || selectionErr != nil {
		report.Resolution = RecoveryDrift
		return finalizeRecoveryReport(report)
	}
	if hasSelection {
		if _, err := loadGeneration(ctx, store, selection.SelectedGenerationSHA256); err != nil {
			report.Resolution = RecoveryDrift
			return finalizeRecoveryReport(report)
		}
		if selection.PriorGenerationSHA256 != "" {
			if _, err := loadGeneration(ctx, store, selection.PriorGenerationSHA256); err != nil {
				report.Resolution = RecoveryDrift
				return finalizeRecoveryReport(report)
			}
		}
	}
	if hasIntent {
		report.TargetGenerationSHA256 = intent.TargetGenerationSHA256
		report.BaselineSelectionSHA256 = intent.BaselineSelectionSHA256
		if hasLock && lock.TargetGenerationSHA256 != intent.TargetGenerationSHA256 {
			report.Resolution = RecoveryDrift
			return finalizeRecoveryReport(report)
		}
	} else if hasLock {
		report.TargetGenerationSHA256 = lock.TargetGenerationSHA256
	}
	if report.TargetGenerationSHA256 != "" {
		path := generationPath(store, report.TargetGenerationSHA256)
		if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
			report.TargetGenerationStatus = "absent"
		} else if err != nil {
			report.TargetGenerationStatus = "invalid"
			report.Resolution = RecoveryDrift
			return finalizeRecoveryReport(report)
		} else if _, err := loadGeneration(ctx, store, report.TargetGenerationSHA256); err != nil {
			report.TargetGenerationStatus = "invalid"
			report.Resolution = RecoveryDrift
			return finalizeRecoveryReport(report)
		} else {
			report.TargetGenerationStatus = "complete"
		}
	}
	if hasIntent {
		if selectionMatchesTarget(selection, hasSelection, intent.TargetSelection) {
			if report.TargetGenerationStatus != "complete" {
				report.Resolution = RecoveryDrift
			} else {
				report.Resolution = RecoveryPromoted
			}
		} else if observedSelectionDigest(selection, hasSelection) == intent.BaselineSelectionSHA256 {
			report.Resolution = RecoveryRolledBack
		} else {
			report.Resolution = RecoveryDrift
		}
	} else if hasLock {
		if hasSelection && selection.SelectedGenerationSHA256 == lock.TargetGenerationSHA256 {
			if report.TargetGenerationStatus == "complete" {
				report.Resolution = RecoveryPromoted
			} else {
				report.Resolution = RecoveryDrift
			}
		} else {
			report.Resolution = RecoveryRolledBack
		}
	} else if len(transients) > 0 {
		report.Resolution = RecoveryRolledBack
	}
	return finalizeRecoveryReport(report)
}

func buildPromotionLock(target string) (PromotionLock, error) {
	lock := PromotionLock{Version: PromotionLockVersion, TargetGenerationSHA256: target}
	data, err := json.Marshal(lock)
	if err != nil {
		return PromotionLock{}, err
	}
	digest := sha256.Sum256(data)
	lock.LockSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return lock, nil
}

func validatePromotionLock(lock PromotionLock) error {
	want := lock.LockSHA256
	if lock.Version != PromotionLockVersion || !validTaggedSHA256(lock.TargetGenerationSHA256) || !validTaggedSHA256(want) {
		return errors.New("promotion lock has invalid identity")
	}
	lock.LockSHA256 = ""
	data, err := json.Marshal(lock)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if want != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("promotion lock digest mismatch")
	}
	return nil
}

func buildPromotionIntent(target Selection, baseline Selection, hasBaseline bool) (PromotionIntent, error) {
	intent := PromotionIntent{
		Version: PromotionIntentVersion, TargetGenerationSHA256: target.SelectedGenerationSHA256,
		TargetSelection: target,
	}
	if hasBaseline {
		intent.BaselineSelectionSHA256 = baseline.SelectionSHA256
	}
	data, err := json.Marshal(intent)
	if err != nil {
		return PromotionIntent{}, err
	}
	digest := sha256.Sum256(data)
	intent.IntentSHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return intent, nil
}

func validatePromotionIntent(intent PromotionIntent) error {
	want := intent.IntentSHA256
	if intent.Version != PromotionIntentVersion || !validTaggedSHA256(want) || intent.TargetGenerationSHA256 != intent.TargetSelection.SelectedGenerationSHA256 || validateSelection(intent.TargetSelection) != nil {
		return errors.New("promotion intent has invalid identity")
	}
	if intent.BaselineSelectionSHA256 != "" && !validTaggedSHA256(intent.BaselineSelectionSHA256) {
		return errors.New("promotion intent has invalid baseline")
	}
	intent.IntentSHA256 = ""
	data, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if want != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("promotion intent digest mismatch")
	}
	return nil
}

func writePromotionIntent(store string, intent PromotionIntent) error {
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomicNew(filepath.Join(store, promotionIntentFile), append(data, '\n'), 0o600, promotionIntentPrefix); err != nil {
		return errors.New("persist promotion intent")
	}
	return nil
}

func removePromotionIntent(store string) error {
	if err := hitPromotionBoundary(PromotionBeforeIntentCleanup); err != nil {
		return err
	}
	root, err := os.OpenRoot(store)
	if err != nil {
		return errors.New("open promotion store for intent cleanup")
	}
	defer root.Close()
	if err := root.Remove(promotionIntentFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return errors.New("remove promotion intent")
	}
	return syncDirectory(store)
}

func readPromotionLock(store string) (PromotionLock, bool, error) {
	var lock PromotionLock
	ok, err := readStrictPromotionRecord(filepath.Join(store, promotionLockFile), &lock)
	if err != nil || !ok {
		return lock, ok, err
	}
	return lock, true, validatePromotionLock(lock)
}

func readPromotionIntent(store string) (PromotionIntent, bool, error) {
	var intent PromotionIntent
	ok, err := readStrictPromotionRecord(filepath.Join(store, promotionIntentFile), &intent)
	if err != nil || !ok {
		return intent, ok, err
	}
	return intent, true, validatePromotionIntent(intent)
}

func readStrictPromotionRecord(path string, out any) (bool, error) {
	data, info, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil || info.Mode().Perm() != 0o600 {
		return true, errors.New("promotion recovery record is unavailable or unsafe")
	}
	if links, ok := fileLinkCount(info); !ok || links != 1 {
		return true, errors.New("promotion recovery record link count is unsafe or unavailable")
	}
	if err := evidencefile.DecodeStrict(data, out); err != nil {
		return true, errors.New("promotion recovery record is invalid")
	}
	return true, nil
}

func finalizeRecoveryReport(report RecoveryReport) (RecoveryReport, error) {
	data, err := json.Marshal(report)
	if err != nil {
		return RecoveryReport{}, err
	}
	digest := sha256.Sum256(data)
	report.RecoverySHA256 = "sha256:" + hex.EncodeToString(digest[:])
	return report, nil
}

func selectionMatchesTarget(observed Selection, hasObserved bool, target Selection) bool {
	return hasObserved && reflect.DeepEqual(observed, target)
}

func observedSelectionDigest(selection Selection, present bool) string {
	if !present {
		return ""
	}
	return selection.SelectionSHA256
}

func recoveryTransients(store string) ([]string, error) {
	entries, err := os.ReadDir(store)
	if err != nil {
		return nil, errors.New("inspect promotion recovery inventory")
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if name == promotionLockFile || name == promotionIntentFile || strings.HasPrefix(name, promotionStagePrefix) || strings.HasPrefix(name, promotionCurrentPrefix) || strings.HasPrefix(name, promotionLockPrefix) || strings.HasPrefix(name, promotionIntentPrefix) {
			names = append(names, name)
		}
	}
	if len(names) > maxRecoveryTransients {
		return nil, errors.New("promotion recovery transient inventory exceeds the bounded limit")
	}
	sort.Strings(names)
	return names, nil
}

func cleanupRecoveryTransients(store string) error {
	names, err := recoveryTransients(store)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(store)
	if err != nil {
		return errors.New("open promotion store for recovery")
	}
	defer root.Close()
	for _, name := range names {
		if name == promotionLockFile {
			continue
		}
		if strings.HasPrefix(name, promotionStagePrefix) {
			if err := root.RemoveAll(name); err != nil {
				return errors.New("remove promotion recovery staging directory")
			}
		} else if err := root.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return errors.New("remove promotion recovery transient")
		}
	}
	if err := root.Remove(promotionLockFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return errors.New("remove promotion recovery lock")
	}
	return syncDirectory(store)
}

func writeAtomicNew(path string, data []byte, mode fs.FileMode, prefix string) error {
	directory := filepath.Dir(path)
	tmp, err := os.CreateTemp(directory, prefix)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	written, writeErr := tmp.Write(data)
	syncErr, closeErr := tmp.Sync(), tmp.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil || written != len(data) {
		return errors.New("write atomic create-only record")
	}
	if err := os.Link(tmpPath, path); err != nil {
		return err
	}
	if err := os.Remove(tmpPath); err != nil {
		return err
	}
	return syncDirectory(directory)
}
