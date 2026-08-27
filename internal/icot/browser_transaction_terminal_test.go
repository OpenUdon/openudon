package icot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

type browserTransactionTerminalFixture struct {
	args            []string
	transactionPath string
	exampleDir      string
	scratchParent   string
	storeDir        string
	candidateSHA256 string
	reviewedSHA256  string
	registration    browsertransaction.Transaction
}

func TestBrowserTransactionTerminalPreparePromoteAndExistingPackageCompatibility(t *testing.T) {
	fixture := newBrowserTransactionTerminalFixture(t)
	prepareInput := strings.Join([]string{"review " + fixture.candidateSHA256, "prepare " + fixture.reviewedSHA256, ""}, "\n")
	var preparedOut, preparedErr bytes.Buffer
	if code := Main(append([]string{"browser-transaction"}, append(fixture.args, "--prepare")...), strings.NewReader(prepareInput), &preparedOut, &preparedErr); code != 0 {
		t.Fatalf("prepare code = %d\nstdout=%s\nstderr=%s", code, preparedOut.String(), preparedErr.String())
	}
	preparedEvents := decodeBrowserTransactionTerminalEvents(t, preparedOut.Bytes())
	if len(preparedEvents) != 3 || preparedEvents[0].Event != "started" || preparedEvents[1].Event != "review" || preparedEvents[2].Event != "prepare" {
		t.Fatalf("prepare events = %#v", preparedEvents)
	}
	prepared := preparedEvents[2].BrowserTransaction
	if prepared.Transaction == nil || prepared.Transaction.State != browsertransaction.StatePrepared || prepared.Preparation == nil {
		t.Fatalf("prepared resource = %#v", prepared)
	}
	if prepared.Review == nil || prepared.Review.Composition != "BRP" || prepared.Review.RegistrationAuthoring == nil ||
		prepared.Review.RegistrationAuthoring.SubmitSupported || prepared.Review.RegistrationAuthoring.RuntimeSupported ||
		!strings.Contains(prepared.Review.RegistrationAuthoring.AccessibilityLabels, "heuristic") || !strings.Contains(prepared.Review.RegistrationAuthoring.AccessibilityLabels, "not data loss prevention") {
		t.Fatalf("terminal BRP disclosure = %#v", prepared.Review)
	}

	promoteInput := strings.Join([]string{
		"review " + fixture.candidateSHA256,
		"prepare " + fixture.reviewedSHA256,
		strings.Join([]string{"promote", prepared.TransactionSHA256, prepared.Preparation.PreparationSHA256, prepared.Preparation.QualificationSHA256}, " "),
		"",
	}, "\n")
	var promotedOut, promotedErr bytes.Buffer
	promoteArgs := append([]string{}, fixture.args...)
	promoteArgs = append(promoteArgs, "--promote", "--inspect-selected")
	if code := Main(append([]string{"browser-transaction"}, promoteArgs...), strings.NewReader(promoteInput), &promotedOut, &promotedErr); code != 0 {
		t.Fatalf("promote code = %d\nstdout=%s\nstderr=%s", code, promotedOut.String(), promotedErr.String())
	}
	promotedEvents := decodeBrowserTransactionTerminalEvents(t, promotedOut.Bytes())
	last := promotedEvents[len(promotedEvents)-1]
	if last.Event != "inspect_selected" || last.BrowserTransaction.Transaction.State != browsertransaction.StatePromoted || last.BrowserTransaction.Promotion == nil || last.BrowserTransaction.Inspection == nil {
		t.Fatalf("promoted events = %#v", promotedEvents)
	}
	if last.BrowserTransaction.RuntimeExecutionSupported || last.BrowserTransaction.Inspection.ExecutionPolicy.DirectProductionExecution ||
		(last.BrowserTransaction.Inspection.ExecutionPolicy.SideEffectful && last.BrowserTransaction.Inspection.ExecutionPolicy.RequiredNextState == "") {
		t.Fatalf("terminal inspection granted runtime authority = %#v", last.BrowserTransaction)
	}
	if _, err := os.Stat(filepath.Join(fixture.storeDir, "current.json")); err != nil {
		t.Fatalf("existing package was not atomically promoted: %v", err)
	}

	combined := preparedOut.String() + preparedErr.String() + promotedOut.String() + promotedErr.String()
	for _, forbidden := range []string{fixture.transactionPath, fixture.exampleDir, fixture.scratchParent, fixture.storeDir, "credential-value", `"runtime_execution_supported":true`} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("terminal output contains forbidden material %q:\n%s", forbidden, combined)
		}
	}
}

func TestBrowserTransactionTerminalDenialCancellationExpiryAndUsage(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := Main([]string{"browser-transaction", "--help"}, strings.NewReader(""), &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "value-free NDJSON") || stderr.Len() != 0 {
			t.Fatalf("help = code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
		}
	})

	t.Run("non-interactive denial", func(t *testing.T) {
		fixture := newBrowserTransactionTerminalFixture(t)
		var stdout, stderr bytes.Buffer
		code := Main(append([]string{"browser-transaction"}, append(fixture.args, "--review")...), strings.NewReader(""), &stdout, &stderr)
		if code != 1 || !strings.Contains(stderr.String(), `"code":"authorization_required"`) || strings.Contains(stderr.String(), fixture.transactionPath) {
			t.Fatalf("denial = code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
		}
		if _, err := os.Stat(filepath.Join(fixture.storeDir, "current.json")); !os.IsNotExist(err) {
			t.Fatalf("denied review selected a package: %v", err)
		}
	})

	t.Run("exact cancellation", func(t *testing.T) {
		fixture := newBrowserTransactionTerminalFixture(t)
		var stdout, stderr bytes.Buffer
		input := "cancel " + fixture.candidateSHA256 + "\n"
		code := Main(append([]string{"browser-transaction"}, append(fixture.args, "--review")...), strings.NewReader(input), &stdout, &stderr)
		events := decodeBrowserTransactionTerminalEvents(t, stdout.Bytes())
		if code != 0 || len(events) != 2 || events[1].Event != "cancel" || events[1].BrowserTransaction.Transaction.State != browsertransaction.StateCancelled {
			t.Fatalf("cancel = code %d events %#v stderr %s", code, events, stderr.String())
		}
	})

	t.Run("expired candidate", func(t *testing.T) {
		fixture := newBrowserTransactionTerminalFixture(t)
		fixture.registration.Provenance.ObservedAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339Nano)
		fixture.registration.Provenance.ExpiresAt = time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
		writeBrowserTransactionTerminalFixture(t, fixture.transactionPath, fixture.registration)
		fixture.candidateSHA256, _ = browsertransaction.Digest(fixture.registration)
		var stdout, stderr bytes.Buffer
		input := "review " + fixture.candidateSHA256 + "\n"
		code := Main(append([]string{"browser-transaction"}, append(fixture.args, "--review")...), strings.NewReader(input), &stdout, &stderr)
		if code != 1 || !strings.Contains(stderr.String(), `"code":"candidate_stale"`) {
			t.Fatalf("expiry = code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
		}
	})

	t.Run("closed option combinations", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := Main([]string{"browser-transaction", "--recover"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "--recover requires --promote") {
			t.Fatalf("recover usage = code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
		}
	})
}

func TestBrowserTransactionTerminalIndeterminateRecoveryUsesExactSharedState(t *testing.T) {
	fake := newBrowserTransactionTerminalLifecycle()
	options := browserTransactionTerminalOptions{review: true, prepare: true, promote: true, recover: true, inspectSelected: true}
	input := strings.NewReader(strings.Join([]string{
		"review " + terminalDigest("1"),
		"prepare " + terminalDigest("2"),
		strings.Join([]string{"promote", terminalDigest("3"), terminalDigest("4"), terminalDigest("5")}, " "),
		strings.Join([]string{"recover", terminalDigest("6"), terminalDigest("7")}, " "),
		"",
	}, "\n"))
	authorizations := newBrowserTransactionAuthorizationReader(input)
	defer authorizations.Close()
	var stdout, stderr bytes.Buffer
	if code := runBrowserTransactionLifecycle(context.Background(), fake, fake.snapshot, options, authorizations, &stdout, &stderr); code != 0 {
		t.Fatalf("recovery code = %d stdout %s stderr %s", code, stdout.String(), stderr.String())
	}
	events := decodeBrowserTransactionTerminalEvents(t, stdout.Bytes())
	want := []string{"review", "prepare", "promote", "inspect_recovery", "recover", "inspect_selected"}
	if len(events) != len(want) {
		t.Fatalf("recovery events = %#v", events)
	}
	for index := range want {
		if events[index].Event != want[index] {
			t.Fatalf("recovery events = %#v", events)
		}
	}
	if !strings.Contains(stderr.String(), `"code":"promotion_indeterminate"`) || events[4].BrowserTransaction.Transaction.State != browsertransaction.StatePromoted || events[5].BrowserTransaction.Inspection == nil {
		t.Fatalf("recovery journey events=%#v stderr=%s", events, stderr.String())
	}
}

func TestBrowserTransactionTerminalContextCancellationStopsPrompt(t *testing.T) {
	fixture := newBrowserTransactionTerminalFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	code := runBrowserTransactionContext(ctx, append(fixture.args, "--review"), strings.NewReader(""), &stdout, &stderr)
	if code != 130 || !strings.Contains(stderr.String(), `"code":"canceled"`) || strings.Contains(stderr.String(), fixture.exampleDir) {
		t.Fatalf("cancellation = code %d stdout %s stderr %s", code, stdout.String(), stderr.String())
	}
}

func newBrowserTransactionTerminalFixture(t *testing.T) browserTransactionTerminalFixture {
	t.Helper()
	root := t.TempDir()
	example := filepath.Join(root, "legacy-package")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(example, os.DirFS(filepath.Join(repoRoot, "examples", "slack-message-audit-log"))); err != nil {
		t.Fatal(err)
	}
	if _, err := synthesize.Build(context.Background(), synthesize.Options{ExampleDir: example}); err != nil {
		t.Fatalf("build existing package fixture: %v", err)
	}
	scratch, store := filepath.Join(root, "scratch-private"), filepath.Join(root, "store-private")
	for _, path := range []string{scratch, store} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "terminal-registration", Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates:         []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: terminalDigest("a"), ReviewSHA256: terminalDigest("b")}},
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1, ResultSHA256: terminalDigest("c"), ObservedAt: now.Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://example.test"}},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "email", Binding: "account_email"}},
	}
	transactionPath := filepath.Join(root, "public-transaction.json")
	writeBrowserTransactionTerminalFixture(t, transactionPath, transaction)
	candidateSHA256, err := browsertransaction.Digest(transaction)
	if err != nil {
		t.Fatal(err)
	}
	reviewed := transaction
	reviewed.State = browsertransaction.StateReviewed
	reviewedSHA256, err := browsertransaction.Digest(reviewed)
	if err != nil {
		t.Fatal(err)
	}
	return browserTransactionTerminalFixture{
		args:            []string{"--transaction", transactionPath, "--example", example, "--scope", "examples/legacy-package", "--scratch", scratch, "--store", store},
		transactionPath: transactionPath, exampleDir: example, scratchParent: scratch, storeDir: store,
		candidateSHA256: candidateSHA256, reviewedSHA256: reviewedSHA256, registration: transaction,
	}
}

func writeBrowserTransactionTerminalFixture(t *testing.T, path string, transaction browsertransaction.Transaction) {
	t.Helper()
	data, err := browsertransaction.CanonicalBytes(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func decodeBrowserTransactionTerminalEvents(t *testing.T, data []byte) []browserTransactionTerminalEvent {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var events []browserTransactionTerminalEvent
	for decoder.More() {
		var event browserTransactionTerminalEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatalf("decode terminal event: %v\n%s", err, data)
		}
		events = append(events, event)
	}
	return events
}

type fakeBrowserTransactionTerminalLifecycle struct {
	snapshot transactionengine.Snapshot
}

func newBrowserTransactionTerminalLifecycle() *fakeBrowserTransactionTerminalLifecycle {
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "terminal-fake", Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates:         []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: terminalDigest("a"), ReviewSHA256: terminalDigest("b")}},
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1, ResultSHA256: terminalDigest("c"), ObservedAt: "2026-08-26T12:00:00Z", ExpiresAt: "2099-08-26T13:00:00Z", Origins: []string{"https://example.test"}},
		CredentialBindings: []browsertransaction.CredentialBinding{},
	}
	return &fakeBrowserTransactionTerminalLifecycle{snapshot: transactionengine.Snapshot{Version: transactionengine.Version, Revision: terminalDigest("a"), Transaction: &transaction, TransactionSHA256: terminalDigest("1"), AllowedOperations: []transactionengine.Operation{transactionengine.OperationReview}}}
}

func (fake *fakeBrowserTransactionTerminalLifecycle) Observe(context.Context) (transactionengine.Snapshot, error) {
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactionTerminalLifecycle) Start(context.Context, transactionengine.StartRequest) (transactionengine.Snapshot, error) {
	return fake.snapshot, errors.New("not used")
}
func (fake *fakeBrowserTransactionTerminalLifecycle) Review(_ context.Context, request transactionengine.ReviewRequest) (transactionengine.Snapshot, error) {
	if request.ExpectedTransactionSHA256 != terminalDigest("1") || !request.HumanApproved {
		return fake.snapshot, errors.New("review authority mismatch")
	}
	fake.snapshot.Transaction.State, fake.snapshot.TransactionSHA256, fake.snapshot.Revision = browsertransaction.StateReviewed, terminalDigest("2"), terminalDigest("b")
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactionTerminalLifecycle) Prepare(_ context.Context, request transactionengine.PrepareRequest) (transactionengine.Snapshot, error) {
	if request.ExpectedTransactionSHA256 != terminalDigest("2") || !request.HumanApproved {
		return fake.snapshot, errors.New("prepare authority mismatch")
	}
	fake.snapshot.Transaction.State, fake.snapshot.TransactionSHA256, fake.snapshot.Revision = browsertransaction.StatePrepared, terminalDigest("3"), terminalDigest("c")
	fake.snapshot.Preparation = &transactionengine.PreparationEvidence{PreparationSHA256: terminalDigest("4"), QualificationSHA256: terminalDigest("5")}
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactionTerminalLifecycle) Promote(_ context.Context, request transactionengine.PromoteRequest) (transactionengine.Snapshot, error) {
	if request.ExpectedTransactionSHA256 != terminalDigest("3") || request.ExpectedPreparationSHA256 != terminalDigest("4") || request.ExpectedQualificationSHA256 != terminalDigest("5") || !request.HumanApproved {
		return fake.snapshot, errors.New("promote authority mismatch")
	}
	fake.snapshot.Transaction.State, fake.snapshot.TransactionSHA256, fake.snapshot.Revision = browsertransaction.StateIndeterminate, terminalDigest("6"), terminalDigest("d")
	fake.snapshot.LastFailure = &transactionengine.OperationFailure{Class: browsertransaction.FailureIndeterminate, Code: transactionengine.ErrorPromotionIndeterminate, Operation: transactionengine.OperationPromote}
	return fake.snapshot, &transactionengine.Error{Class: browsertransaction.FailureIndeterminate, Code: transactionengine.ErrorPromotionIndeterminate, Operation: transactionengine.OperationPromote}
}
func (fake *fakeBrowserTransactionTerminalLifecycle) Cancel(context.Context, transactionengine.CancelRequest) (transactionengine.Snapshot, error) {
	fake.snapshot.Transaction.State = browsertransaction.StateCancelled
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactionTerminalLifecycle) InspectRecovery(context.Context, transactionengine.InspectRecoveryRequest) (transactionengine.Snapshot, error) {
	fake.snapshot.Revision = terminalDigest("e")
	fake.snapshot.Recovery = &transactionengine.RecoveryEvidence{Report: &packagepipeline.RecoveryReport{Version: packagepipeline.RecoveryReportVersion, Resolution: packagepipeline.RecoveryPromoted, RecoverySHA256: terminalDigest("7")}}
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactionTerminalLifecycle) Recover(_ context.Context, request transactionengine.RecoverRequest) (transactionengine.Snapshot, error) {
	if request.ExpectedTransactionSHA256 != terminalDigest("6") || request.ExpectedRecoverySHA256 != terminalDigest("7") || !request.HumanApproved {
		return fake.snapshot, errors.New("recovery authority mismatch")
	}
	fake.snapshot.Transaction.State, fake.snapshot.Revision = browsertransaction.StatePromoted, terminalDigest("f")
	fake.snapshot.Promotion = &transactionengine.PromotionEvidence{SelectionSHA256: terminalDigest("8"), SelectedGenerationSHA256: terminalDigest("9")}
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactionTerminalLifecycle) InspectSelected(_ context.Context, request transactionengine.InspectSelectedRequest) (transactionengine.Snapshot, error) {
	if request.ExpectedSelectionSHA256 != terminalDigest("8") {
		return fake.snapshot, errors.New("selection authority mismatch")
	}
	fake.snapshot.Inspection = &trustedrunner.PackageInspection{Scope: "examples/fake", ExecutionPolicy: authoring.ReviewExecutionPolicy{SideEffectful: true}}
	return fake.snapshot, nil
}

func terminalDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
