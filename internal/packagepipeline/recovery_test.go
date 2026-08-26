package packagepipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPromotionFaultBoundariesHaveTypedRecoverableOutcomes(t *testing.T) {
	rolledBack := []PromotionBoundary{
		PromotionAfterIntentWrite,
		PromotionBeforeGenerationRename,
		PromotionAfterGenerationRename,
		PromotionBeforeGenerationSync,
		PromotionAfterGenerationSync,
		PromotionBeforeSelectionRename,
	}
	indeterminate := []PromotionBoundary{
		PromotionAfterSelectionRename,
		PromotionBeforeSelectionSync,
		PromotionAfterSelectionSync,
		PromotionBeforeIntentCleanup,
		PromotionBeforeLockCleanup,
	}
	for _, boundary := range append(rolledBack, indeterminate...) {
		t.Run(string(boundary), func(t *testing.T) {
			qualified, _, _ := qualifiedPromotionFixture(t)
			store := promotionStore(t)
			triggered := false
			promotionFaultHook = func(observed PromotionBoundary) error {
				if observed == boundary && !triggered {
					triggered = true
					return errors.New("injected promotion boundary failure")
				}
				return nil
			}
			_, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
			promotionFaultHook = nil
			if !triggered {
				t.Fatalf("promotion boundary %s was not reached", boundary)
			}
			state, ok := PromotionFailureState(err)
			if !ok {
				t.Fatalf("fault error was not typed: %v", err)
			}
			wantState := PromotionRolledBackState
			if containsPromotionBoundary(indeterminate, boundary) {
				wantState = PromotionIndeterminateState
			}
			if state != wantState || strings.Contains(err.Error(), store) {
				t.Fatalf("fault state = %q, error=%v, want=%q", state, err, wantState)
			}
			generation, ok := PromotionFailureGeneration(err)
			if !ok || !validTaggedSHA256(generation) {
				t.Fatalf("fault omitted recoverable target identity: %q, present=%t", generation, ok)
			}
			if wantState == PromotionRolledBackState {
				report, reportErr := InspectRecovery(context.Background(), store)
				if reportErr != nil || report.Resolution != RecoveryClean {
					t.Fatalf("safe rollback retained recovery state: %#v, %v", report, reportErr)
				}
				if _, currentErr := ReadCurrent(context.Background(), store); currentErr == nil {
					t.Fatal("safe rollback selected the target generation")
				}
				return
			}

			if _, retryErr := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store}); retryErr == nil {
				t.Fatal("indeterminate promotion allowed a blind retry")
			}
			report, reportErr := InspectRecovery(context.Background(), store)
			if reportErr != nil || report.Resolution != RecoveryPromoted || !validTaggedSHA256(report.RecoverySHA256) {
				t.Fatalf("indeterminate recovery report = %#v, %v", report, reportErr)
			}
			_, wrongErr := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: strings.Repeat("0", 64)})
			if code, ok := PromotionFailureCode(wrongErr); !ok || code != PromotionRecoveryDrift {
				t.Fatalf("wrong recovery digest error = %v, code=%q", wrongErr, code)
			}
			reconciled, reconcileErr := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256})
			if reconcileErr != nil || reconciled.Resolution != RecoveryPromoted || reconciled.TargetGenerationSHA256 != report.TargetGenerationSHA256 {
				t.Fatalf("reconciliation = %#v, %v", reconciled, reconcileErr)
			}
			current, currentErr := ReadCurrent(context.Background(), store)
			if currentErr != nil || !reflect.DeepEqual(current.Files(), qualified.Prepared().Files()) {
				t.Fatalf("reconciled selected generation = %#v, %v", current.Selection(), currentErr)
			}
			assertNoPromotionTransients(t, store)
		})
	}
}

func TestInterruptedPreselectionReconcilesAsRollbackAndPreservesGeneration(t *testing.T) {
	qualified, _, _ := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	promotionFaultHook = func(boundary PromotionBoundary) error {
		switch boundary {
		case PromotionAfterGenerationRename:
			return context.Canceled
		case PromotionBeforeIntentCleanup:
			return errors.New("simulate process loss before rollback cleanup")
		default:
			return nil
		}
	}
	_, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	promotionFaultHook = nil
	if state, ok := PromotionFailureState(err); !ok || state != PromotionIndeterminateState {
		t.Fatalf("interrupted preselection state = %q, %v", state, err)
	}
	report, err := InspectRecovery(context.Background(), store)
	if err != nil || report.Resolution != RecoveryRolledBack || report.TargetGenerationStatus != "complete" {
		t.Fatalf("rollback recovery report = %#v, %v", report, err)
	}
	repeated, err := InspectRecovery(context.Background(), store)
	if err != nil || !reflect.DeepEqual(repeated, report) {
		t.Fatalf("recovery report is not deterministic: first=%#v second=%#v error=%v", report, repeated, err)
	}
	generationPath := generationPath(store, report.TargetGenerationSHA256)
	reconciled, err := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256})
	if err != nil || reconciled.Resolution != RecoveryRolledBack {
		t.Fatalf("rollback reconciliation = %#v, %v", reconciled, err)
	}
	if _, err := os.Stat(generationPath); err != nil {
		t.Fatalf("recoverable unselected generation was deleted: %v", err)
	}
	if _, err := ReadCurrent(context.Background(), store); err == nil {
		t.Fatal("rollback reconciliation treated recoverable bytes as current")
	}
	assertNoPromotionTransients(t, store)
}

func TestInterruptedIntentBeforePublicationReconcilesAsRollback(t *testing.T) {
	qualified, _, _ := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	promotionFaultHook = func(boundary PromotionBoundary) error {
		switch boundary {
		case PromotionAfterIntentWrite:
			return context.Canceled
		case PromotionBeforeIntentCleanup:
			return errors.New("simulate process loss with durable intent")
		default:
			return nil
		}
	}
	_, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	promotionFaultHook = nil
	if state, ok := PromotionFailureState(err); !ok || state != PromotionIndeterminateState {
		t.Fatalf("interrupted intent state = %q, %v", state, err)
	}
	report, err := InspectRecovery(context.Background(), store)
	if err != nil || report.Resolution != RecoveryRolledBack || report.TargetGenerationStatus != "absent" {
		t.Fatalf("intent-only recovery report = %#v, %v", report, err)
	}
	if _, err := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256}); err != nil {
		t.Fatal(err)
	}
	if entries := mustReadDir(t, filepath.Join(store, promotionGenerationsDir)); len(entries) != 0 {
		t.Fatalf("intent-only interruption published a generation: %#v", entries)
	}
	assertNoPromotionTransients(t, store)
}

func TestRecoveryRejectsSelectionDriftWithoutCleanup(t *testing.T) {
	qualified, _, _ := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	promotionFaultHook = func(boundary PromotionBoundary) error {
		if boundary == PromotionAfterSelectionRename {
			return errors.New("interrupt after selector replacement")
		}
		return nil
	}
	_, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	promotionFaultHook = nil
	if state, ok := PromotionFailureState(err); !ok || state != PromotionIndeterminateState {
		t.Fatalf("indeterminate setup = %q, %v", state, err)
	}
	currentPath := filepath.Join(store, promotionCurrentFile)
	if err := os.WriteFile(currentPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := InspectRecovery(context.Background(), store)
	if err != nil || report.Resolution != RecoveryDrift {
		t.Fatalf("drift report = %#v, %v", report, err)
	}
	_, err = Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256})
	if code, ok := PromotionFailureCode(err); !ok || code != PromotionRecoveryDrift {
		t.Fatalf("drift reconciliation error = %v, code=%q", err, code)
	}
	if _, err := os.Stat(filepath.Join(store, promotionIntentFile)); err != nil {
		t.Fatalf("drift reconciliation removed intent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, promotionLockFile)); err != nil {
		t.Fatalf("drift reconciliation removed lock: %v", err)
	}
}

func TestRollbackReconciliationPreservesSelectedPriorAndTarget(t *testing.T) {
	first, source, scratch := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	firstPromoted, err := Promote(context.Background(), first, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	second := changedQualifiedPromotionFixture(t, source, scratch)
	secondPromoted, err := Promote(context.Background(), second, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	third := changedQualifiedPromotionFixture(t, source, scratch)
	promotionFaultHook = func(boundary PromotionBoundary) error {
		switch boundary {
		case PromotionAfterGenerationRename:
			return errors.New("interrupt third generation before selection")
		case PromotionBeforeIntentCleanup:
			return errors.New("retain recovery intent")
		default:
			return nil
		}
	}
	_, promotionErr := Promote(context.Background(), third, PromotionOptions{StoreDir: store})
	promotionFaultHook = nil
	target, ok := PromotionFailureGeneration(promotionErr)
	if !ok {
		t.Fatalf("third promotion omitted target: %v", promotionErr)
	}
	report, err := InspectRecovery(context.Background(), store)
	if err != nil || report.Resolution != RecoveryRolledBack || report.TargetGenerationSHA256 != target || report.ObservedSelectedGenerationSHA256 != secondPromoted.Selection().SelectedGenerationSHA256 || report.ObservedPriorGenerationSHA256 != firstPromoted.Selection().SelectedGenerationSHA256 {
		t.Fatalf("three-generation recovery report = %#v, %v", report, err)
	}
	if _, err := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256}); err != nil {
		t.Fatal(err)
	}
	for _, generation := range []string{firstPromoted.Selection().SelectedGenerationSHA256, secondPromoted.Selection().SelectedGenerationSHA256, target} {
		if _, err := loadGeneration(context.Background(), store, generation); err != nil {
			t.Fatalf("reconciliation removed generation %s: %v", generation, err)
		}
	}
	current, err := ReadCurrent(context.Background(), store)
	if err != nil || !reflect.DeepEqual(current.Selection(), secondPromoted.Selection()) {
		t.Fatalf("rollback reconciliation changed current: %#v, %v", current.Selection(), err)
	}
}

func TestRecoveryTransientInventoryIsBounded(t *testing.T) {
	store := promotionStore(t)
	for index := 0; index <= maxRecoveryTransients; index++ {
		path := filepath.Join(store, promotionStagePrefix+strings.Repeat("x", 4)+string(rune('a'+index%26))+strings.Repeat("y", index/26))
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := InspectRecovery(context.Background(), store); err == nil {
		t.Fatal("oversized recovery inventory was accepted")
	}
}

func TestRecoveryCleansOrphanedStagingWithoutChangingCurrent(t *testing.T) {
	qualified, _, _ := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	promoted, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(store, promotionStagePrefix+"restart")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "recoverable"), []byte("unselected\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, promotionCurrentPrefix+"restart"), []byte("partial selector\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := InspectRecovery(context.Background(), store)
	if err != nil || report.Resolution != RecoveryRolledBack || report.TransientCount != 2 {
		t.Fatalf("orphan recovery report = %#v, %v", report, err)
	}
	if _, err := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256}); err != nil {
		t.Fatal(err)
	}
	current, err := ReadCurrent(context.Background(), store)
	if err != nil || !reflect.DeepEqual(current.Selection(), promoted.Selection()) {
		t.Fatalf("orphan cleanup changed current: %#v, %v", current.Selection(), err)
	}
	assertNoPromotionTransients(t, store)
}

func TestInterruptedPromotionCancellationStates(t *testing.T) {
	for _, test := range []struct {
		name     string
		boundary PromotionBoundary
		state    PromotionState
	}{
		{name: "before selection", boundary: PromotionAfterGenerationRename, state: PromotionRolledBackState},
		{name: "after selection", boundary: PromotionAfterSelectionRename, state: PromotionIndeterminateState},
	} {
		t.Run(test.name, func(t *testing.T) {
			qualified, _, _ := qualifiedPromotionFixture(t)
			store := promotionStore(t)
			promotionFaultHook = func(boundary PromotionBoundary) error {
				if boundary == test.boundary {
					return context.Canceled
				}
				return nil
			}
			_, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
			promotionFaultHook = nil
			code, codeOK := PromotionFailureCode(err)
			state, stateOK := PromotionFailureState(err)
			if !codeOK || !stateOK || code != PromotionCanceled || state != test.state {
				t.Fatalf("cancellation = code %q state %q error %v", code, state, err)
			}
			if state == PromotionIndeterminateState {
				report, reportErr := InspectRecovery(context.Background(), store)
				if reportErr != nil {
					t.Fatal(reportErr)
				}
				if _, reconcileErr := Reconcile(context.Background(), ReconcileOptions{StoreDir: store, ExpectedRecoverySHA256: report.RecoverySHA256}); reconcileErr != nil {
					t.Fatal(reconcileErr)
				}
			}
		})
	}
}

func containsPromotionBoundary(values []PromotionBoundary, target PromotionBoundary) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
