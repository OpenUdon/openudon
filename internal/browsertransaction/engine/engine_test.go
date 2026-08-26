package engine

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

var transactionTime = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func TestSharedEngineRealPackageLifecycleIsValueFreeAndRuntimeFree(t *testing.T) {
	config := transactionEngineConfig(t)
	engine, initial, err := New(config)
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	if !reflect.DeepEqual(initial.AllowedOperations, []Operation{OperationObserve, OperationStart}) || initial.RuntimeExecutionSupported {
		t.Fatalf("initial snapshot = %#v", initial)
	}
	started := startCandidate(t, engine, initial, registrationTransaction(browsertransaction.StateCandidate))
	if started.Transaction.State != browsertransaction.StateCandidate || started.Transaction.Session != "" {
		t.Fatalf("registration start = %#v", started.Transaction)
	}
	reviewed, err := engine.Review(context.Background(), ReviewRequest{Authority{ExpectedRevision: started.Revision, ExpectedTransactionSHA256: started.TransactionSHA256, HumanApproved: true}})
	if err != nil || reviewed.Transaction.State != browsertransaction.StateReviewed {
		t.Fatalf("review = %#v, %v", reviewed, err)
	}
	prepared, err := engine.Prepare(context.Background(), PrepareRequest{Authority{ExpectedRevision: reviewed.Revision, ExpectedTransactionSHA256: reviewed.TransactionSHA256, HumanApproved: true}})
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	if prepared.Transaction.State != browsertransaction.StatePrepared || prepared.Preparation == nil || prepared.Preparation.PreparationSHA256 == "" || prepared.Preparation.QualificationSHA256 == "" {
		t.Fatalf("preparation evidence = %#v", prepared)
	}
	promoted, err := engine.Promote(context.Background(), PromoteRequest{
		Authority:                 Authority{ExpectedRevision: prepared.Revision, ExpectedTransactionSHA256: prepared.TransactionSHA256, HumanApproved: true},
		ExpectedPreparationSHA256: prepared.Preparation.PreparationSHA256, ExpectedQualificationSHA256: prepared.Preparation.QualificationSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Transaction.State != browsertransaction.StatePromoted || promoted.Promotion == nil ||
		promoted.Promotion.GenerationSHA256 != promoted.Transaction.Promotion.GenerationSHA256 || promoted.Promotion.SelectionSHA256 == "" {
		t.Fatalf("promotion evidence = %#v", promoted)
	}
	inspected, err := engine.InspectSelected(context.Background(), InspectSelectedRequest{ExpectedRevision: promoted.Revision, ExpectedSelectionSHA256: promoted.Promotion.SelectionSHA256})
	if err != nil || inspected.Inspection == nil || taggedSHA256(inspected.Inspection.PackageSHA256) != prepared.Preparation.PackageSHA256 {
		t.Fatalf("inspection = %#v, %v", inspected.Inspection, err)
	}
	encoded, err := json.Marshal(inspected)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{config.Package.ExampleDir, config.Package.ScratchParent, config.Package.StoreDir, "private candidate body", "runtime_path"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("snapshot contains forbidden material %q: %s", forbidden, encoded)
		}
	}
	if strings.Contains(string(encoded), `"runtime_execution_supported":true`) {
		t.Fatalf("snapshot granted runtime authority: %s", encoded)
	}
}

func TestAuthorizationRevisionConcurrencyAndCancellationPreserveState(t *testing.T) {
	engine, initial, err := New(transactionEngineConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	started := startCandidate(t, engine, initial, authenticationCapabilityTransaction(browsertransaction.StateCandidate))
	denied, err := engine.Review(context.Background(), ReviewRequest{Authority{ExpectedRevision: started.Revision, ExpectedTransactionSHA256: started.TransactionSHA256}})
	assertEngineError(t, err, ErrorAuthorizationRequired, OperationReview)
	if denied.Revision != started.Revision || denied.Transaction.State != browsertransaction.StateCandidate {
		t.Fatalf("denied review mutated state: %#v", denied)
	}

	request := ReviewRequest{Authority{ExpectedRevision: started.Revision, ExpectedTransactionSHA256: started.TransactionSHA256, HumanApproved: true}}
	results := make(chan struct {
		snapshot Snapshot
		err      error
	}, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	begin := make(chan struct{})
	for range 2 {
		go func() {
			ready.Done()
			<-begin
			snapshot, callErr := engine.Review(context.Background(), request)
			results <- struct {
				snapshot Snapshot
				err      error
			}{snapshot, callErr}
		}()
	}
	ready.Wait()
	close(begin)
	var successes, stale int
	var reviewed Snapshot
	var concurrentErrors []error
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			reviewed = result.snapshot
		} else {
			concurrentErrors = append(concurrentErrors, result.err)
			_, code, operation, _, _ := ErrorDetails(result.err)
			if code == ErrorStaleRevision && operation == OperationReview {
				stale++
			}
		}
	}
	if successes != 1 || stale != 1 || reviewed.Transaction.State != browsertransaction.StateReviewed {
		t.Fatalf("concurrent results successes=%d stale=%d reviewed=%#v errors=%v", successes, stale, reviewed.Transaction, concurrentErrors)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	afterCancel, err := engine.Prepare(canceled, PrepareRequest{Authority{ExpectedRevision: reviewed.Revision, ExpectedTransactionSHA256: reviewed.TransactionSHA256, HumanApproved: true}})
	assertEngineError(t, err, ErrorCanceled, OperationPrepare)
	if afterCancel.Revision != reviewed.Revision || afterCancel.Transaction.State != browsertransaction.StateReviewed {
		t.Fatalf("context cancellation mutated state: %#v", afterCancel)
	}
}

func TestIndeterminatePromotionRequiresExactInspectedRecovery(t *testing.T) {
	config := transactionEngineConfig(t)
	qualified, err := packagepipeline.PrepareAndQualifyCurrent(context.Background(), config.Package)
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	target := taggedDigest("9")
	selectionDigest := taggedDigest("8")
	recoveryDigest := taggedDigest("7")
	operations := &fakePackageOperations{
		qualified:  qualified,
		promoteErr: &packagepipeline.PromotionError{Code: packagepipeline.PromotionSelectionFailed, State: packagepipeline.PromotionIndeterminateState, GenerationSHA256: target},
		baseline: packagepipeline.RecoveryReport{
			Version: packagepipeline.RecoveryReportVersion, Resolution: packagepipeline.RecoveryClean,
			RecoverySHA256: taggedDigest("5"),
		},
		recovery: packagepipeline.RecoveryReport{
			Version: packagepipeline.RecoveryReportVersion, Resolution: packagepipeline.RecoveryPromoted,
			TargetGenerationSHA256: target, ObservedSelectionSHA256: selectionDigest,
			ObservedSelectedGenerationSHA256: target, RecoverySHA256: recoveryDigest,
		},
		reconciliation: packagepipeline.Reconciliation{
			Version: packagepipeline.ReconciliationVersion, Resolution: packagepipeline.RecoveryPromoted,
			TargetGenerationSHA256: target, SelectedGenerationSHA256: target, ObservedRecoverySHA256: recoveryDigest,
		},
	}
	config.operations = operations
	engine, initial, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	started := startCandidate(t, engine, initial, registrationTransaction(browsertransaction.StateReviewed))
	prepared, err := engine.Prepare(context.Background(), PrepareRequest{Authority{ExpectedRevision: started.Revision, ExpectedTransactionSHA256: started.TransactionSHA256, HumanApproved: true}})
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	indeterminate, err := engine.Promote(context.Background(), PromoteRequest{
		Authority:                 Authority{ExpectedRevision: prepared.Revision, ExpectedTransactionSHA256: prepared.TransactionSHA256, HumanApproved: true},
		ExpectedPreparationSHA256: prepared.Preparation.PreparationSHA256, ExpectedQualificationSHA256: prepared.Preparation.QualificationSHA256,
	})
	assertEngineError(t, err, ErrorPromotionIndeterminate, OperationPromote)
	if indeterminate.Transaction.State != browsertransaction.StateIndeterminate || indeterminate.LastFailure == nil ||
		indeterminate.LastFailure.TargetGenerationSHA256 != target || indeterminate.Recovery == nil {
		t.Fatalf("indeterminate snapshot = %#v", indeterminate)
	}

	wrong, err := engine.Recover(context.Background(), RecoverRequest{
		Authority:              Authority{ExpectedRevision: indeterminate.Revision, ExpectedTransactionSHA256: indeterminate.TransactionSHA256, HumanApproved: true},
		ExpectedRecoverySHA256: taggedDigest("6"),
	})
	assertEngineError(t, err, ErrorRecoveryDrift, OperationRecover)
	if wrong.Revision != indeterminate.Revision || operations.reconcileCalls != 0 {
		t.Fatalf("blind recovery changed state or invoked reconcile: %#v calls=%d", wrong, operations.reconcileCalls)
	}

	recovered, err := engine.Recover(context.Background(), RecoverRequest{
		Authority:              Authority{ExpectedRevision: indeterminate.Revision, ExpectedTransactionSHA256: indeterminate.TransactionSHA256, HumanApproved: true},
		ExpectedRecoverySHA256: recoveryDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Transaction.State != browsertransaction.StatePromoted || recovered.Transaction.Promotion.GenerationSHA256 != target ||
		recovered.Promotion.SelectionSHA256 != selectionDigest || recovered.Recovery.Reconciliation == nil || operations.reconcileCalls != 1 {
		t.Fatalf("recovered snapshot = %#v calls=%d", recovered, operations.reconcileCalls)
	}
}

func TestSnapshotSerializationIsDefensiveAndCandidateExpiryFailsClosed(t *testing.T) {
	config := transactionEngineConfig(t)
	config.Now = func() time.Time { return transactionTime.Add(3 * time.Hour) }
	engine, initial, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	started := startCandidate(t, engine, initial, registrationTransaction(browsertransaction.StateCandidate))
	started.Transaction.Candidates[0].SourceSHA256 = taggedDigest("0")
	observed, err := engine.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if observed.Transaction.Candidates[0].SourceSHA256 == taggedDigest("0") {
		t.Fatal("caller changed the engine transaction through a snapshot")
	}
	unchanged, err := engine.Review(context.Background(), ReviewRequest{Authority{ExpectedRevision: observed.Revision, ExpectedTransactionSHA256: observed.TransactionSHA256, HumanApproved: true}})
	assertEngineError(t, err, ErrorCandidateStale, OperationReview)
	if unchanged.Revision != observed.Revision || unchanged.Transaction.State != browsertransaction.StateCandidate {
		t.Fatalf("expired candidate changed state: %#v", unchanged)
	}
	first, _ := json.Marshal(observed)
	again, _ := engine.Observe(context.Background())
	second, _ := json.Marshal(again)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("observation serialization changed\n%s\n%s", first, second)
	}
}

type fakePackageOperations struct {
	qualified      packagepipeline.Qualified
	promoteErr     error
	baseline       packagepipeline.RecoveryReport
	recovery       packagepipeline.RecoveryReport
	reconciliation packagepipeline.Reconciliation
	reconcileCalls int
	inspectCalls   int
}

func (fake *fakePackageOperations) prepareAndQualify(context.Context, packagepipeline.CurrentOptions) (packagepipeline.Qualified, error) {
	return fake.qualified, nil
}
func (fake *fakePackageOperations) promote(context.Context, packagepipeline.Qualified, packagepipeline.PromotionOptions) (packagepipeline.Promoted, error) {
	return packagepipeline.Promoted{}, fake.promoteErr
}
func (fake *fakePackageOperations) readCurrent(context.Context, string) (packagepipeline.Promoted, error) {
	return packagepipeline.Promoted{}, errors.New("no current selection")
}
func (fake *fakePackageOperations) inspectRecovery(context.Context, string) (packagepipeline.RecoveryReport, error) {
	fake.inspectCalls++
	if fake.inspectCalls == 1 {
		return fake.baseline, nil
	}
	return fake.recovery, nil
}
func (fake *fakePackageOperations) reconcile(context.Context, packagepipeline.ReconcileOptions) (packagepipeline.Reconciliation, error) {
	fake.reconcileCalls++
	return fake.reconciliation, nil
}
func (fake *fakePackageOperations) inspectSelected(context.Context, string, string) (trustedrunner.PackageInspection, error) {
	return trustedrunner.PackageInspection{}, nil
}

func transactionEngineConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "examples", "support-priority-routing")
	if err := os.CopyFS(source, os.DirFS(filepath.Join(transactionRepoRoot(t), "examples", "support-priority-routing"))); err != nil {
		t.Fatal(err)
	}
	if _, err := synthesize.Build(context.Background(), synthesize.Options{ExampleDir: source}); err != nil {
		t.Fatalf("build package fixture: %v", err)
	}
	scratch, store := filepath.Join(root, "scratch"), filepath.Join(root, "store")
	if err := os.Mkdir(scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(store, 0o700); err != nil {
		t.Fatal(err)
	}
	return Config{
		Package: packagepipeline.CurrentOptions{ExampleDir: source, Scope: "examples/support-priority-routing", ScratchParent: scratch, StoreDir: store},
		Now:     func() time.Time { return transactionTime },
	}
}

func startCandidate(t *testing.T, engine *Engine, initial Snapshot, transaction browsertransaction.Transaction) Snapshot {
	t.Helper()
	data, err := browsertransaction.CanonicalBytes(transaction)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := browsertransaction.Digest(transaction)
	if err != nil {
		t.Fatal(err)
	}
	started, err := engine.Start(context.Background(), StartRequest{ExpectedRevision: initial.Revision, ExpectedTransactionSHA256: digest, TransactionJSON: data})
	if err != nil {
		t.Fatal(err)
	}
	return started
}

func registrationTransaction(state browsertransaction.State) browsertransaction.Transaction {
	return browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "registration-transaction", Kind: browsertransaction.KindRegistration, State: state,
		Candidates:         []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: taggedDigest("1"), ReviewSHA256: taggedDigest("2")}},
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1, ResultSHA256: taggedDigest("3"), ObservedAt: transactionTime.Format(time.RFC3339Nano), ExpiresAt: transactionTime.Add(2 * time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://example.test"}},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "email", Binding: "account_email"}},
	}
}

func authenticationCapabilityTransaction(state browsertransaction.State) browsertransaction.Transaction {
	return browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "authentication-capability-transaction", Kind: browsertransaction.KindAuthenticationCapability, State: state,
		Candidates: []browsertransaction.Candidate{
			{Kind: browsertransaction.CandidateAuthentication, Schema: "uws.browser-authentication.1.1", SourceSHA256: taggedDigest("1"), ReviewSHA256: taggedDigest("2")},
			{Kind: browsertransaction.CandidateCapability, Schema: "uws.browser.1.7", SourceSHA256: taggedDigest("3"), ReviewSHA256: taggedDigest("4")},
		},
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultAuthenticatedAuthoringV2, ResultSHA256: taggedDigest("5"), ObservedAt: transactionTime.Format(time.RFC3339Nano), ExpiresAt: transactionTime.Add(2 * time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://example.test"}},
		CredentialBindings: []browsertransaction.CredentialBinding{}, Session: "account_session",
	}
}

func taggedDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }

func assertEngineError(t *testing.T, err error, wantCode ErrorCode, wantOperation Operation) {
	t.Helper()
	if err == nil {
		t.Fatalf("wanted %s error", wantCode)
	}
	_, code, operation, _, ok := ErrorDetails(err)
	if !ok || code != wantCode || operation != wantOperation {
		t.Fatalf("error = %v, details=(%q,%q,%v)", err, code, operation, ok)
	}
}

func transactionRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
