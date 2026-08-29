//go:build icot_ui_browser

package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

type phaseCBrowserTransactionEngine struct {
	mu       sync.Mutex
	snapshot transactionengine.Snapshot
	sequence int
}

func newPhaseCBrowserTransactionEngine() *phaseCBrowserTransactionEngine {
	transaction := apiRegistrationTransaction()
	transaction.Provenance.ExpiresAt = "2099-08-26T13:00:00Z"
	digest, _ := browsertransaction.Digest(transaction)
	return &phaseCBrowserTransactionEngine{snapshot: transactionengine.Snapshot{
		Version: transactionengine.Version, Revision: apiDigest("a"), Transaction: &transaction, TransactionSHA256: digest,
		AllowedOperations: []transactionengine.Operation{transactionengine.OperationObserve, transactionengine.OperationReview, transactionengine.OperationCancel},
	}}
}

func (e *phaseCBrowserTransactionEngine) forceRevision() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snapshot.Revision = apiDigest("0")
}

func (e *phaseCBrowserTransactionEngine) expire() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snapshot.Transaction.Provenance.ExpiresAt = "2000-01-01T00:00:00Z"
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
}

func (e *phaseCBrowserTransactionEngine) useAuthenticationCapability() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snapshot.Transaction.Kind = browsertransaction.KindAuthenticationCapability
	e.snapshot.Transaction.Candidates = []browsertransaction.Candidate{
		{Kind: browsertransaction.CandidateAuthentication, Schema: "uws.browser-authentication.1.1", SourceSHA256: apiDigest("1"), ReviewSHA256: apiDigest("2")},
		{Kind: browsertransaction.CandidateCapability, Schema: "uws.browser.1.7", SourceSHA256: apiDigest("3"), ReviewSHA256: apiDigest("4")},
	}
	e.snapshot.Transaction.Provenance.ResultVersion = browsertransaction.ResultAuthenticatedAuthoringV2
	e.snapshot.Transaction.Session = "account_session"
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
}

func (e *phaseCBrowserTransactionEngine) Observe(context.Context) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshot, nil
}

func (e *phaseCBrowserTransactionEngine) advance(operation transactionengine.Operation, revision string) error {
	if revision != e.snapshot.Revision {
		return &transactionengine.Error{Class: browsertransaction.FailureConflict, Code: transactionengine.ErrorStaleRevision, Operation: operation, Retryable: true}
	}
	e.sequence++
	characters := "bcdef0123456789"
	e.snapshot.Revision = apiDigest(string(characters[(e.sequence-1)%len(characters)]))
	return nil
}

func (e *phaseCBrowserTransactionEngine) Start(context.Context, transactionengine.StartRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureConflict, Code: transactionengine.ErrorInvalidState, Operation: transactionengine.OperationStart}
}

func (e *phaseCBrowserTransactionEngine) Review(_ context.Context, request transactionengine.ReviewRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationReview, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	if !request.HumanApproved || request.ExpectedTransactionSHA256 != e.snapshot.TransactionSHA256 {
		return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureRejected, Code: transactionengine.ErrorDigestMismatch, Operation: transactionengine.OperationReview}
	}
	e.snapshot.Transaction.State = browsertransaction.StateReviewed
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
	e.snapshot.AllowedOperations = []transactionengine.Operation{transactionengine.OperationObserve, transactionengine.OperationPrepare, transactionengine.OperationCancel}
	return e.snapshot, nil
}

func (e *phaseCBrowserTransactionEngine) Prepare(_ context.Context, request transactionengine.PrepareRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationPrepare, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	if !request.HumanApproved || request.ExpectedTransactionSHA256 != e.snapshot.TransactionSHA256 {
		return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureRejected, Code: transactionengine.ErrorDigestMismatch, Operation: transactionengine.OperationPrepare}
	}
	preparation := &transactionengine.PreparationEvidence{
		PreparationSHA256: apiDigest("1"), InputSHA256: apiDigest("2"), PackageSHA256: apiDigest("3"), HandoffSHA256: apiDigest("4"), QualitySHA256: apiDigest("5"), QualificationSHA256: apiDigest("6"),
	}
	e.snapshot.Preparation = preparation
	e.snapshot.Transaction.State = browsertransaction.StatePrepared
	e.snapshot.Transaction.Preparation = &browsertransaction.Preparation{PackageSHA256: preparation.PackageSHA256, QualificationSHA256: preparation.QualificationSHA256}
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
	e.snapshot.AllowedOperations = []transactionengine.Operation{transactionengine.OperationObserve, transactionengine.OperationPromote, transactionengine.OperationCancel, transactionengine.OperationInspectRecovery}
	return e.snapshot, nil
}

func (e *phaseCBrowserTransactionEngine) Promote(_ context.Context, request transactionengine.PromoteRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationPromote, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	if !request.HumanApproved || request.ExpectedTransactionSHA256 != e.snapshot.TransactionSHA256 || request.ExpectedPreparationSHA256 != e.snapshot.Preparation.PreparationSHA256 || request.ExpectedQualificationSHA256 != e.snapshot.Preparation.QualificationSHA256 {
		return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureRejected, Code: transactionengine.ErrorDigestMismatch, Operation: transactionengine.OperationPromote}
	}
	target, recovery := apiDigest("7"), apiDigest("8")
	e.snapshot.Transaction.State = browsertransaction.StateIndeterminate
	e.snapshot.Transaction.Failure = &browsertransaction.Failure{Class: browsertransaction.FailureIndeterminate, Code: browsertransaction.FailurePromotionIndeterminate}
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
	e.snapshot.LastFailure = &transactionengine.OperationFailure{Class: browsertransaction.FailureIndeterminate, Code: transactionengine.ErrorPromotionIndeterminate, Operation: transactionengine.OperationPromote, PromotionState: packagepipeline.PromotionIndeterminateState, TargetGenerationSHA256: target}
	e.snapshot.Recovery = &transactionengine.RecoveryEvidence{Report: &packagepipeline.RecoveryReport{
		Version: packagepipeline.RecoveryReportVersion, Resolution: packagepipeline.RecoveryPromoted, TargetGenerationSHA256: target,
		ObservedSelectionSHA256: apiDigest("9"), ObservedSelectedGenerationSHA256: target, RecoverySHA256: recovery,
	}}
	e.snapshot.AllowedOperations = []transactionengine.Operation{transactionengine.OperationObserve, transactionengine.OperationInspectRecovery, transactionengine.OperationRecover, transactionengine.OperationCancel}
	return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureIndeterminate, Code: transactionengine.ErrorPromotionIndeterminate, Operation: transactionengine.OperationPromote}
}

func (e *phaseCBrowserTransactionEngine) Cancel(_ context.Context, request transactionengine.CancelRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationCancel, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	e.snapshot.Transaction.State, e.snapshot.Transaction.Failure = browsertransaction.StateCancelled, nil
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
	e.snapshot.AllowedOperations = []transactionengine.Operation{transactionengine.OperationObserve}
	return e.snapshot, nil
}

func (e *phaseCBrowserTransactionEngine) InspectRecovery(_ context.Context, request transactionengine.InspectRecoveryRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationInspectRecovery, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	return e.snapshot, nil
}

func (e *phaseCBrowserTransactionEngine) Recover(_ context.Context, request transactionengine.RecoverRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationRecover, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	report := e.snapshot.Recovery.Report
	if !request.HumanApproved || request.ExpectedTransactionSHA256 != e.snapshot.TransactionSHA256 || request.ExpectedRecoverySHA256 != report.RecoverySHA256 {
		return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureConflict, Code: transactionengine.ErrorRecoveryDrift, Operation: transactionengine.OperationRecover}
	}
	target := report.TargetGenerationSHA256
	e.snapshot.Transaction.State, e.snapshot.Transaction.Failure = browsertransaction.StatePromoted, nil
	e.snapshot.Transaction.Promotion = &browsertransaction.Promotion{GenerationSHA256: target}
	e.snapshot.TransactionSHA256, _ = browsertransaction.Digest(*e.snapshot.Transaction)
	e.snapshot.Promotion = &transactionengine.PromotionEvidence{GenerationSHA256: target, SelectionSHA256: report.ObservedSelectionSHA256, SelectedGenerationSHA256: target}
	e.snapshot.Recovery.Reconciliation = &packagepipeline.Reconciliation{Version: packagepipeline.ReconciliationVersion, Resolution: packagepipeline.RecoveryPromoted, TargetGenerationSHA256: target, SelectedGenerationSHA256: target, ObservedRecoverySHA256: report.RecoverySHA256}
	e.snapshot.LastFailure = nil
	e.snapshot.AllowedOperations = []transactionengine.Operation{transactionengine.OperationObserve, transactionengine.OperationInspectSelected}
	return e.snapshot, nil
}

func (e *phaseCBrowserTransactionEngine) InspectSelected(_ context.Context, request transactionengine.InspectSelectedRequest) (transactionengine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.advance(transactionengine.OperationInspectSelected, request.ExpectedRevision); err != nil {
		return e.snapshot, err
	}
	if request.ExpectedSelectionSHA256 != e.snapshot.Promotion.SelectionSHA256 {
		return e.snapshot, &transactionengine.Error{Class: browsertransaction.FailureConflict, Code: transactionengine.ErrorDigestMismatch, Operation: transactionengine.OperationInspectSelected}
	}
	e.snapshot.Inspection = &trustedrunner.PackageInspection{Scope: "examples/registration", PackageSHA256: apiDigest("3"), HandoffSHA256: apiDigest("4"), ExecutionPolicy: authoring.ReviewExecutionPolicy{SideEffectful: true}}
	return e.snapshot, nil
}

type phaseCBrowserEngine struct {
	mu sync.Mutex

	snapshot         engine.Snapshot
	proposal         engine.Snapshot
	reopenProposal   engine.Snapshot
	workspace        engine.WorkspaceStatus
	workspaceErr     error
	roundErr         error
	roundErrOnce     bool
	roundCalls       int
	reopenCalls      int
	approvalCalls    int
	answers          []authoring.RoundAnswer
	reopenedQuestion string
	approval         engine.Approval
}

func (e *phaseCBrowserEngine) ReopenDecision(_ context.Context, questionID string) (engine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reopenCalls++
	e.reopenedQuestion = questionID
	e.snapshot = e.reopenProposal
	return e.snapshot, nil
}

type phaseCSnapshotDelay struct {
	mu sync.Mutex

	armed     bool
	started   chan struct{}
	release   chan struct{}
	completed chan struct{}
}

func (d *phaseCSnapshotDelay) arm(t *testing.T) (<-chan struct{}, <-chan struct{}, func()) {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.armed {
		t.Fatal("snapshot delay is already armed")
	}
	d.armed = true
	d.started = make(chan struct{})
	d.release = make(chan struct{})
	d.completed = make(chan struct{})
	started, release, completed := d.started, d.release, d.completed
	var once sync.Once
	releaseRequest := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseRequest)
	return started, completed, releaseRequest
}

func (d *phaseCSnapshotDelay) serveHTTP(next http.Handler, w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/api/v4/snapshot") {
		next.ServeHTTP(w, r)
		return
	}
	d.mu.Lock()
	if !d.armed {
		d.mu.Unlock()
		next.ServeHTTP(w, r)
		return
	}
	d.armed = false
	started, release, completed := d.started, d.release, d.completed
	d.mu.Unlock()

	// Capture a full same-revision payload before the mutation, then delay only
	// its delivery. This recreates a poll that has left the server but overlaps
	// the browser's POST lifecycle.
	r.Header.Del("If-None-Match")
	recorded := httptest.NewRecorder()
	next.ServeHTTP(recorded, r)
	close(started)
	<-release
	for name, values := range recorded.Header() {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(recorded.Code)
	_, _ = w.Write(recorded.Body.Bytes())
	close(completed)
}

func (e *phaseCBrowserEngine) ApplyRound(_ context.Context, answers []authoring.RoundAnswer) (engine.Snapshot, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.roundCalls++
	e.answers = append([]authoring.RoundAnswer(nil), answers...)
	if e.roundErr != nil {
		err := e.roundErr
		if e.roundErrOnce {
			e.roundErr = nil
		}
		return engine.Snapshot{}, err
	}
	e.snapshot = e.proposal
	return e.snapshot, nil
}

func (e *phaseCBrowserEngine) ApproveAndWrite(_ context.Context, approval engine.Approval) (engine.ApprovalResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.approvalCalls++
	e.approval = approval
	if !approval.HumanApproved {
		return engine.ApprovalResult{}, &engine.Failure{Class: engine.FailureRejected, Code: "engine_rejected", Cause: errors.New("explicit approval required")}
	}
	preview := engine.Preview{}
	if e.snapshot.Preview != nil {
		preview = *e.snapshot.Preview
	}
	return engine.ApprovalResult{
		Snapshot: e.snapshot,
		WriteResult: engine.WriteResult{
			Written:    []string{"project.md", preview.IntentPath},
			Incomplete: preview.Incomplete,
			Preview:    preview,
		},
	}, nil
}

func (e *phaseCBrowserEngine) WorkspaceStatus(context.Context) (engine.WorkspaceStatus, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.workspace, e.workspaceErr
}

func (e *phaseCBrowserEngine) setWorkspace(status engine.WorkspaceStatus) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workspace = status
}

func (e *phaseCBrowserEngine) setWorkspaceError(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workspaceErr = err
}

func (e *phaseCBrowserEngine) mutationRecord() (int, int, []authoring.RoundAnswer, engine.Approval) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.roundCalls, e.approvalCalls, append([]authoring.RoundAnswer(nil), e.answers...), e.approval
}

func (e *phaseCBrowserEngine) reopenRecord() (int, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reopenCalls, e.reopenedQuestion
}

type phaseCBrowserFixture struct {
	engine     *phaseCBrowserEngine
	server     *http.Server
	listener   net.Listener
	token      string
	accessCode string
	authority  string
	url        string
	delay      *phaseCSnapshotDelay
	terminal   *bytes.Buffer
}

func newPhaseCBrowserFixture(t *testing.T, browserEngine *phaseCBrowserEngine) *phaseCBrowserFixture {
	return newPhaseCBrowserFixtureWithConfig(t, browserEngine, nil)
}

func newPhaseCBrowserFixtureWithConfig(t *testing.T, browserEngine *phaseCBrowserEngine, configure func(*HandlerConfig)) *phaseCBrowserFixture {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	authority := listener.Addr().String()
	token := "phase-c-browser-test-capability"
	accessCode := "0123456789AB"
	terminal := &bytes.Buffer{}
	config := HandlerConfig{
		Engine: browserEngine, Snapshot: browserEngine.snapshot, ExampleDir: "/tmp/phase-c-browser", Token: token, AccessCode: accessCode, Authority: authority, AccessCodeOut: terminal,
	}
	if configure != nil {
		configure(&config)
	}
	handler, err := NewHandler(config)
	if err != nil {
		_ = listener.Close()
		t.Fatal(err)
	}
	delay := &phaseCSnapshotDelay{}
	baseHandler := handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delay.serveHTTP(baseHandler, w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	fixture := &phaseCBrowserFixture{
		engine: browserEngine, server: server, listener: listener, token: token, accessCode: accessCode, authority: authority,
		url: "http://" + authority + "/", delay: delay, terminal: terminal,
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		select {
		case serveErr := <-done:
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				t.Errorf("phase C browser server: %v", serveErr)
			}
		case <-time.After(2 * time.Second):
			t.Error("phase C browser server did not stop")
		}
	})
	return fixture
}

func phaseCFrontierSnapshot() engine.Snapshot {
	issue := elicitor.ReadinessIssue{
		Code: "boundary.review_required", Severity: "blocking", Slot: "workflow.goal", Message: "Confirm the active outcome.", SuggestedAnswer: "Create a reviewed report",
	}
	return engine.Snapshot{
		Frontier: []elicitor.QuestionPlan{
			{ID: "boundary.outcome", Prompt: "What outcome should this workflow achieve?", Slots: []string{"workflow.goal"}, Required: true, Forced: true, Recommendation: "Create a reviewed report", Priority: 100, Rationale: "The active boundary controls later decisions."},
			{ID: "boundary.actor_trigger", Prompt: "Who starts the workflow and when?", Slots: []string{"workflow.actor", "workflow.trigger"}, Required: true, Priority: 90},
		},
		Readiness:        []elicitor.ReadinessIssue{issue},
		TopIssue:         &issue,
		ProposedActions:  []elicitor.FileAction{{Action: "write", Path: "/tmp/phase-c-browser/project.md", Reason: "render the reviewed boundary"}},
		WriteConflicts:   []engine.WriteConflict{},
		SourceCandidates: engine.SourceCandidates{},
	}
}

func phaseCProposalSnapshot(incomplete bool, withConflict bool) engine.Snapshot {
	intentPath := "/tmp/phase-c-browser/workflows/intent.hcl"
	intentContent := "workflow {\n  name = \"reviewed\"\n}\n"
	ready := true
	var readiness []elicitor.ReadinessIssue
	if incomplete {
		intentPath = "/tmp/phase-c-browser/workflows/intent.draft.hcl"
		intentContent = "workflow {\n  name = \"reviewed_draft\"\n}\n"
		ready = false
		readiness = []elicitor.ReadinessIssue{{Code: "source.selection_deferred", Severity: "blocking", Slot: "source.selection", Message: "Source selection is explicitly deferred."}}
	}
	conflicts := []engine.WriteConflict{}
	if withConflict {
		conflicts = append(conflicts, engine.WriteConflict{Code: "overwrite_required", Action: "write", Path: "/tmp/phase-c-browser/project.md"})
	}
	return engine.Snapshot{
		Boundary:           elicitor.WorkflowBoundary{Outcome: "Create a reviewed report", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report is available"}, Confirmed: true},
		CandidateWorkflows: []elicitor.CandidateWorkflow{{Title: "Archive reports", Outcome: "Archive reviewed reports", DeferralReason: "outside the active boundary", PromotionTrigger: "active workflow is complete"}},
		Evidence: []publicinterview.Evidence{{
			ID: "evidence.operator-boundary", Kind: publicinterview.EvidenceUserDecision, Summary: "The operator confirmed the active report boundary.", Source: "user", References: []string{"boundary.outcome"},
		}},
		Readiness:        readiness,
		Ready:            ready,
		ApprovalRequired: true,
		SelectedSources: []elicitor.SourceMaterialization{{
			Kind: "openapi", ID: "reports", TargetPath: "openapi/reports.yaml", SHA256: strings.Repeat("a", 64),
		}},
		ProposedActions: []elicitor.FileAction{
			{Action: "write", Path: "/tmp/phase-c-browser/project.md", Reason: "render the reviewed boundary"},
			{Action: "write", Path: intentPath, Reason: "render the reviewed workflow intent"},
		},
		WriteConflicts: conflicts,
		Preview: &engine.Preview{
			ProjectMD: "# Reviewed project\n", IntentHCL: intentContent, Incomplete: incomplete,
			ProjectPath: "/tmp/phase-c-browser/project.md", IntentPath: intentPath,
		},
		SourceCandidates: engine.SourceCandidates{Remote: []elicitor.RemoteSourceCandidate{{Kind: "openapi", ID: "reports", Title: "Reports API", Provenance: "reviewed catalog hint"}}},
	}
}

func launchPhaseCBrowser(t *testing.T) (*playwright.Playwright, playwright.Browser) {
	t.Helper()
	disableSandbox := os.Getenv("OPENUDON_ICOT_UI_BROWSER_DISABLE_SANDBOX") == "1"
	requireSandbox := os.Getenv("OPENUDON_ICOT_UI_BROWSER_SANDBOX_REQUIRED") == "1"
	if requireSandbox && disableSandbox {
		t.Fatal("required iCoT UI browser qualification rejects the sandbox-disable override")
	}
	sandbox := !disableSandbox
	t.Logf("chromium_sandbox_enabled=%t sandbox_required=%t", sandbox, requireSandbox)
	if requireSandbox && !sandbox {
		t.Fatal("required iCoT UI browser qualification must enable the Chromium sandbox")
	}
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("start Playwright-Go: %v (install with: go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.0 install chromium)", err)
	}
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), ChromiumSandbox: playwright.Bool(sandbox),
	})
	if err != nil {
		_ = pw.Stop()
		t.Fatalf("launch required Chromium: %v", err)
	}
	t.Cleanup(func() {
		_ = browser.Close()
		_ = pw.Stop()
	})
	return pw, browser
}

func newPhaseCPage(t *testing.T, browser playwright.Browser, fixture *phaseCBrowserFixture) playwright.Page {
	t.Helper()
	page, err := browser.NewPage(playwright.BrowserNewPageOptions{
		Viewport: &playwright.Size{Width: 960, Height: 900},
	})
	if err != nil {
		t.Fatal(err)
	}
	page.OnPageError(func(err error) { t.Errorf("iCoT UI page error: %v", err) })
	t.Cleanup(func() { _ = page.Close() })
	if _, err := page.Goto(fixture.url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator(`input[name="code"]`).Fill(fixture.accessCode); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Continue", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "Connected")
	if strings.Contains(page.URL(), "token=") {
		t.Fatalf("bootstrap token remained in browser URL: %s", page.URL())
	}
	return page
}

func waitForLocatorText(t *testing.T, locator playwright.Locator, want string) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		text, err := locator.TextContent()
		if err == nil {
			last = text
			if strings.Contains(text, want) {
				return text
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("locator text %q does not contain %q", last, want)
	return ""
}

func requireVisible(t *testing.T, locator playwright.Locator) {
	t.Helper()
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible, Timeout: playwright.Float(10000)}); err != nil {
		t.Fatal(err)
	}
}

func requireEnabled(t *testing.T, locator playwright.Locator, want bool) {
	t.Helper()
	enabled, err := locator.IsEnabled()
	if err != nil {
		t.Fatal(err)
	}
	if enabled != want {
		t.Fatalf("enabled = %t, want %t", enabled, want)
	}
}

func waitForPhaseCSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func waitForActiveID(t *testing.T, page playwright.Page, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var last any
	for time.Now().Before(deadline) {
		value, err := page.Evaluate("document.activeElement && document.activeElement.id")
		if err == nil {
			last = value
			if value == want {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("active element = %#v, want id %q", last, want)
}

func phaseCBrowserTransactionSnapshot(transactions *phaseCBrowserTransactionEngine) engine.Snapshot {
	transactions.mu.Lock()
	defer transactions.mu.Unlock()
	transaction := transactions.snapshot.Transaction
	snapshot := phaseCFrontierSnapshot()
	snapshot.SourceCandidates.VirtualBrowser = engine.VirtualBrowserCandidateSet{
		Generation: 1,
		Candidates: []elicitor.VirtualBrowserCandidate{{
			ID: "registration-api/registration", TransactionID: transaction.ID, TransactionSHA256: transactions.snapshot.TransactionSHA256,
			Kind: browsertransaction.CandidateRegistration, Schema: transaction.Candidates[0].Schema,
			SourceSHA256: transaction.Candidates[0].SourceSHA256, ReviewSHA256: transaction.Candidates[0].ReviewSHA256,
			TargetPath: "browser-registration/registration-api.json", Flow: "create_account", CleanupDisposition: "delete_separately",
			CredentialBindings: append([]browsertransaction.CredentialBinding(nil), transaction.CredentialBindings...),
		}},
	}
	return snapshot
}

func TestPhaseCBrowserTransactionReviewRecoveryAndAccessibility(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	transactions := newPhaseCBrowserTransactionEngine()
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCBrowserTransactionSnapshot(transactions)}
	fixture := newPhaseCBrowserFixtureWithConfig(t, browserEngine, func(config *HandlerConfig) {
		config.BrowserTransactions = transactions
		config.Now = func() time.Time { return apiTransactionTime }
	})
	page := newPhaseCPage(t, browser, fixture)

	section := page.Locator("#browser-transaction-section")
	requireVisible(t, section)
	waitForLocatorText(t, page.Locator("#browser-transaction-state"), "BRP · candidate")
	waitForLocatorText(t, page.Locator("#browser-registration-label-disclosure"), "heuristic")
	waitForLocatorText(t, page.Locator("#browser-registration-label-disclosure"), "not data loss prevention")
	waitForLocatorText(t, page.Locator("#browser-registration-policy"), "GET/HEAD only")
	waitForLocatorText(t, page.Locator("#browser-registration-policy"), "does not grant execution authority")
	waitForLocatorText(t, page.Locator("#browser-transaction-candidates"), "browser-registration/registration-api.json · delete_separately")
	waitForLocatorText(t, page.Locator("#browser-transaction-authority"), "grants no browser or workflow runtime authority")

	actionText, err := page.Locator("#browser-transaction-actions button").AllTextContents()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range actionText {
		lower := strings.ToLower(label)
		for _, forbidden := range []string{"execute", "register", "sign in", "submit"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("transaction action %q offered forbidden %q authority", label, forbidden)
			}
		}
	}

	reviewConfirmation := page.GetByLabel("I reviewed the exact candidates, origins, allowed actions, outputs, checkpoints, cleanup, and disclosures.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	reviewButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Accept candidate review", Exact: playwright.Bool(true)})
	requireEnabled(t, reviewButton, false)
	if err := reviewConfirmation.Check(); err != nil {
		t.Fatal(err)
	}
	requireEnabled(t, reviewButton, true)

	// A second local client advances the transaction revision. The stale
	// confirmation must fail closed, refresh, and require fresh review.
	transactions.forceRevision()
	if err := reviewButton.Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#error-message"), "stale_revision")
	waitForActiveID(t, page, "browser-transaction-heading")
	if checked, err := reviewConfirmation.IsChecked(); err != nil || checked {
		t.Fatalf("stale revision retained review confirmation = %t, %v", checked, err)
	}
	if err := reviewConfirmation.Check(); err != nil {
		t.Fatal(err)
	}
	if err := reviewButton.Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#browser-transaction-state"), "BRP · reviewed")
	waitForActiveID(t, page, "browser-transaction-heading")

	prepareConfirmation := page.GetByLabel("I authorize non-promoting scratch preparation and restrictive offline qualification.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	prepareButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Prepare and qualify", Exact: playwright.Bool(true)})
	requireEnabled(t, prepareButton, false)
	if err := prepareConfirmation.Check(); err != nil {
		t.Fatal(err)
	}
	if err := prepareButton.Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#browser-transaction-state"), "BRP · prepared")
	waitForLocatorText(t, page.Locator("#browser-transaction-package"), apiDigest("6"))

	promoteConfirmation := page.GetByLabel("I authorize atomic promotion of the exact qualified generation.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	promoteButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Promote exact generation", Exact: playwright.Bool(true)})
	requireEnabled(t, promoteButton, false)
	if err := promoteConfirmation.Check(); err != nil {
		t.Fatal(err)
	}
	if err := promoteButton.Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#browser-transaction-state"), "BRP · indeterminate")
	waitForLocatorText(t, page.Locator("#browser-transaction-status"), "promotion_indeterminate")
	waitForLocatorText(t, page.Locator("#browser-transaction-recovery"), apiDigest("8"))

	recoverConfirmation := page.GetByLabel("I accept the exact recovery report digest shown above.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	recoverButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Reconcile exact recovery report", Exact: playwright.Bool(true)})
	requireEnabled(t, recoverButton, false)
	if err := recoverConfirmation.Check(); err != nil {
		t.Fatal(err)
	}
	if err := recoverButton.Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#browser-transaction-state"), "BRP · promoted")
	inspect := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Inspect exact selected package", Exact: playwright.Bool(true)})
	requireEnabled(t, inspect, true)
	if err := inspect.Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#browser-transaction-authority"), "requiring a separate trusted-runner approval")

	if err := page.SetViewportSize(360, 760); err != nil {
		t.Fatal(err)
	}
	noOverflow, err := page.Evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")
	if err != nil || noOverflow != true {
		t.Fatalf("360px transaction layout horizontal overflow = %#v, %v", noOverflow, err)
	}
	if err := page.EmulateMedia(playwright.PageEmulateMediaOptions{ReducedMotion: playwright.ReducedMotionReduce}); err != nil {
		t.Fatal(err)
	}
	reduced, err := page.Evaluate(`matchMedia("(prefers-reduced-motion: reduce)").matches && getComputedStyle(document.querySelector("button")).transitionDuration === "0s"`)
	if err != nil || reduced != true {
		t.Fatalf("reduced-motion transaction journey = %#v, %v", reduced, err)
	}
}

func TestPhaseCBrowserGuidedRegistrationDraftReview(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	session := newFakeRegistrationAuthoringSession()
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot()}
	fixture := newPhaseCBrowserFixtureWithConfig(t, browserEngine, func(config *HandlerConfig) {
		config.PrivateRoot = "/tmp/phase-c-registration-private"
		config.Now = func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) }
		config.BrowserTransactions = newFakeBrowserTransactions()
		config.StartRegistration = func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return session, nil
		}
	})
	page := newPhaseCPage(t, browser, fixture)

	for id, value := range map[string]string{
		"#registration-profile-id": "synthetic_registration",
		"#registration-title":      "Synthetic dedicated test registration",
		"#registration-provider":   "Synthetic loopback",
		"#registration-url":        "https://app.example.test/register?action=startnew",
		"#registration-origins":    "https://app.example.test",
	} {
		if err := page.Locator(id).Fill(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Launch no-submit observation", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	session.events <- browserauthor.RegistrationEvent{State: "ready"}
	select {
	case command := <-session.commands:
		if command.Type != "start" || !strings.Contains(command.URL, "action=startnew") {
			t.Fatalf("private browser start = %#v", command)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("registration browser start command timed out")
	}
	session.events <- browserauthor.RegistrationEvent{State: "observing", Phase: "observing", Bounds: &registrationauthorsession.Bounds{
		NavigationTimeoutMS: 20_000, TotalTimeoutMS: 300_000, MaxRequests: 256, MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128,
	}}
	observe := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Observe current page", Exact: playwright.Bool(true)})
	requireVisible(t, observe)
	if err := observe.Click(); err != nil {
		t.Fatal(err)
	}
	if command := <-session.commands; command.Type != "observe" {
		t.Fatalf("browser observe command = %#v", command)
	}
	observation := registrationDraftObservation()
	session.events <- browserauthor.RegistrationEvent{State: "observation", Phase: "observing", Observation: &observation}
	requireVisible(t, page.Locator("#registration-draft-form"))
	waitForLocatorText(t, page.Locator("#registration-observation-panel"), "Confirm password")

	rows := page.Locator("#registration-step-list .registration-step-row")
	if count, err := rows.Count(); err != nil || count != 7 {
		t.Fatalf("default registration steps = %d, %v", count, err)
	}
	if err := rows.Nth(0).Locator(`[data-registration-step="navigate"]`).Fill("https://app.example.test/register?action=startnew"); err != nil {
		t.Fatal(err)
	}
	selections := []struct {
		row             int
		candidate, slot string
	}{
		{1, "candidate-0000000000000001", "identifier"},
		{2, "candidate-0000000000000002", "password"},
		{3, "candidate-0000000000000003", "password"},
		{4, "candidate-0000000000000004", "contact_name"},
		{5, "candidate-0000000000000005", ""},
		{6, "candidate-0000000000000006", ""},
	}
	for _, selection := range selections {
		values := []string{selection.candidate}
		if _, err := rows.Nth(selection.row).Locator(`[data-registration-step="candidate"]`).SelectOption(playwright.SelectOptionValues{Values: &values}); err != nil {
			t.Fatal(err)
		}
		if selection.slot != "" {
			values := []string{selection.slot}
			if _, err := rows.Nth(selection.row).Locator(`[data-registration-step="slot"]`).SelectOption(playwright.SelectOptionValues{Values: &values}); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := page.Locator("#registration-confirmation-prompt").Fill("Approve creation of one dedicated test identity."); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator("#registration-success-path").Fill("/registration-complete"); err != nil {
		t.Fatal(err)
	}
	originValues := []string{"https://app.example.test"}
	if _, err := page.Locator("#registration-success-origin").SelectOption(playwright.SelectOptionValues{Values: &originValues}); err != nil {
		t.Fatal(err)
	}
	roleValues := []string{"status"}
	if _, err := page.Locator("#registration-success-role").SelectOption(playwright.SelectOptionValues{Values: &roleValues}); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator("#registration-success-name").Fill("Registration complete"); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator("#registration-success-reviewed").Check(); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Build canonical draft for review", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#registration-canonical-profile"), "action=startnew")
	waitForLocatorText(t, page.Locator("#registration-retained-queries"), "action=startnew")
	if err := page.Locator("#registration-draft-confirmed").Check(); err != nil {
		t.Fatal(err)
	}
	review := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Confirm worker review", Exact: playwright.Bool(true)})
	requireEnabled(t, review, true)
	if err := review.Click(); err != nil {
		t.Fatal(err)
	}
	if command := <-session.commands; command.Type != "review" || len(command.Profile) == 0 || len(command.CandidateIDs) != 6 || len(command.CredentialBindings) != 3 {
		t.Fatalf("browser review command = %#v", command)
	}
	session.events <- browserauthor.RegistrationEvent{State: "reviewed", Phase: "reviewed"}
	finish := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Finish and tear down worker", Exact: playwright.Bool(true)})
	requireVisible(t, finish)
	if err := finish.Click(); err != nil {
		t.Fatal(err)
	}
	if command := <-session.commands; command.Type != "finish" || !command.Confirmed {
		t.Fatalf("browser finish command = %#v", command)
	}
	session.Cancel()
}

func TestPhaseCBrowserExpiredTransactionBlocksReview(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	transactions := newPhaseCBrowserTransactionEngine()
	transactions.expire()
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCBrowserTransactionSnapshot(transactions)}
	fixture := newPhaseCBrowserFixtureWithConfig(t, browserEngine, func(config *HandlerConfig) {
		config.BrowserTransactions = transactions
		config.Now = func() time.Time { return apiTransactionTime }
	})
	page := newPhaseCPage(t, browser, fixture)

	waitForLocatorText(t, page.Locator("#browser-transaction-status"), "Candidate freshness expired")
	reviewConfirmation := page.GetByLabel("I reviewed the exact candidates, origins, allowed actions, outputs, checkpoints, cleanup, and disclosures.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if disabled, err := reviewConfirmation.IsDisabled(); err != nil || !disabled {
		t.Fatalf("expired review confirmation disabled = %t, %v", disabled, err)
	}
	cancelConfirmation := page.GetByLabel("I want to cancel this transaction without runtime execution.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if disabled, err := cancelConfirmation.IsDisabled(); err != nil || disabled {
		t.Fatalf("expired cancellation confirmation disabled = %t, %v", disabled, err)
	}
}

func TestPhaseCBrowserDistinguishesAuthenticationCapabilityComposition(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	transactions := newPhaseCBrowserTransactionEngine()
	transactions.useAuthenticationCapability()
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot()}
	fixture := newPhaseCBrowserFixtureWithConfig(t, browserEngine, func(config *HandlerConfig) {
		config.BrowserTransactions = transactions
		config.Now = func() time.Time { return apiTransactionTime }
	})
	page := newPhaseCPage(t, browser, fixture)

	waitForLocatorText(t, page.Locator("#browser-transaction-state"), "BAP+BCP · candidate")
	waitForLocatorText(t, page.Locator("#browser-transaction-symbols"), "browser session → account_session")
	if visible, err := page.Locator("#browser-registration-disclosure").IsVisible(); err != nil || visible {
		t.Fatalf("registration disclosure visible for BAP+BCP = %t, %v", visible, err)
	}
	waitForLocatorText(t, page.Locator("#browser-transaction-candidates"), "authentication")
	waitForLocatorText(t, page.Locator("#browser-transaction-candidates"), "capability")
}

func TestPhaseCBrowserAccessibleRoundAndFinalApproval(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, true)}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	form := page.GetByRole("form", playwright.PageGetByRoleOptions{Name: "Authoring questions", Exact: playwright.Bool(true)})
	requireVisible(t, form)
	outcome := page.GetByLabel("Your answer for: What outcome should this workflow achieve?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	actor := page.GetByLabel("Your answer for: Who starts the workflow and when?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if count, err := page.Locator("fieldset.question").Count(); err != nil || count != 2 {
		t.Fatalf("accessible question groups = %d, %v", count, err)
	}
	if err := page.Keyboard().Press("Tab"); err != nil {
		t.Fatal(err)
	}
	active, err := page.Evaluate("document.activeElement && document.activeElement.className")
	if err != nil || active != "skip-link" {
		t.Fatalf("first keyboard focus = %#v, %v", active, err)
	}
	outline, err := page.Evaluate("getComputedStyle(document.activeElement).outlineStyle")
	if err != nil || outline == "none" {
		t.Fatalf("keyboard focus outline = %#v, %v", outline, err)
	}
	if err := page.Keyboard().Press("Tab"); err != nil {
		t.Fatal(err)
	}
	active, err = page.Evaluate("document.activeElement && document.activeElement.id")
	if err != nil || active != "journey-api" {
		t.Fatalf("journey starter keyboard order = %#v, %v", active, err)
	}
	if err := outcome.Focus(); err != nil {
		t.Fatal(err)
	}
	active, err = page.Evaluate("document.activeElement && document.activeElement.id")
	if err != nil || active != "frontier-answer-1" {
		t.Fatalf("question keyboard order = %#v, %v", active, err)
	}

	recommendation := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Use recommendation for What outcome should this workflow achieve?", Exact: playwright.Bool(true)})
	if err := recommendation.Click(); err != nil {
		t.Fatal(err)
	}
	if value, err := outcome.InputValue(); err != nil || value != "Create a reviewed report" {
		t.Fatalf("recommended value = %q, %v", value, err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	active, err = page.Evaluate("document.activeElement && document.activeElement.id")
	if err != nil || active != "frontier-answer-2" {
		t.Fatalf("blank-round focus = %#v, %v", active, err)
	}
	if invalid, err := actor.GetAttribute("aria-invalid"); err != nil || invalid != "true" {
		t.Fatalf("blank answer aria-invalid = %q, %v", invalid, err)
	}
	if err := actor.Fill("operator | on demand"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}

	finalButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true)})
	requireVisible(t, finalButton)
	waitForActiveID(t, page, "review-heading")
	waitForLocatorText(t, page.Locator("#mutation-status"), "proposal is ready for review")
	waitForLocatorText(t, page.Locator("#project-preview"), "Reviewed project")
	waitForLocatorText(t, page.Locator("#actions-body"), "intent.hcl")
	waitForLocatorText(t, page.Locator("#conflicts-list"), "overwrite authorization")
	waitForLocatorText(t, page.Locator("#candidate-workflows-list"), "Archive reports")
	waitForLocatorText(t, page.Locator("#decision-evidence-list"), "operator confirmed the active report boundary")
	waitForLocatorText(t, page.Locator("#source-evidence-list"), "Reports API")
	requireEnabled(t, finalButton, false)

	review := page.GetByLabel("I reviewed the preview, readiness findings, proposed actions, and conflicts.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	overwrite := page.GetByLabel("I authorize replacement of every listed accepted-baseline conflict.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := review.Check(); err != nil {
		t.Fatal(err)
	}
	requireEnabled(t, finalButton, false)
	if err := overwrite.Check(); err != nil {
		t.Fatal(err)
	}
	requireEnabled(t, finalButton, true)
	if err := finalButton.Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.Locator("#package-section"))
	waitForActiveID(t, page, "package-heading")
	waitForLocatorText(t, page.Locator("#mutation-status"), "Approval committed")
	waitForLocatorText(t, page.Locator("#written-list"), "project.md")
	waitForLocatorText(t, page.Locator("#package-status"), "Ready to build")

	rounds, approvals, answers, approval := browserEngine.mutationRecord()
	if rounds != 1 || approvals != 1 || len(answers) != 2 {
		t.Fatalf("mutations = rounds %d approvals %d answers %#v", rounds, approvals, answers)
	}
	if answers[0].QuestionID != "boundary.outcome" || answers[0].Value != "Create a reviewed report" || answers[1].QuestionID != "boundary.actor_trigger" || answers[1].Value != "operator | on demand" {
		t.Fatalf("complete round answers = %#v", answers)
	}
	if !approval.HumanApproved || !approval.AllowOverwrite || approval.ApproveIncomplete {
		t.Fatalf("final approval = %#v", approval)
	}

	noOverflow, err := page.Evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")
	if err != nil || noOverflow != true {
		overflowing, _ := page.Evaluate(`({document: {scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth}, body: {scrollWidth: document.body.scrollWidth, clientWidth: document.body.clientWidth}, nodes: Array.from(document.querySelectorAll("*")).filter((node) => node.scrollWidth > node.clientWidth + 1 && getComputedStyle(node).overflowX === "visible").map((node) => ({tag: node.tagName, id: node.id, className: node.className, scrollWidth: node.scrollWidth, clientWidth: node.clientWidth})).slice(0, 12)})`)
		t.Fatalf("page horizontal overflow = %#v, %v; nodes=%#v", noOverflow, err, overflowing)
	}
	if err := page.SetViewportSize(1280, 900); err != nil {
		t.Fatal(err)
	}
	if _, err := page.Evaluate(`document.documentElement.style.zoom = "2"`); err != nil {
		t.Fatal(err)
	}
	zoomReflows, err := page.Evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")
	if err != nil || zoomReflows != true {
		t.Fatalf("200%% zoom horizontal overflow = %#v, %v", zoomReflows, err)
	}
}

func TestPhaseCBrowserRecoversAfterSessionCookieLoss(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	proposal := phaseCProposalSnapshot(false, false)
	browserEngine := &phaseCBrowserEngine{snapshot: proposal, proposal: proposal}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	scopedURL := page.URL()
	if err := page.Context().ClearCookies(); err != nil {
		t.Fatal(err)
	}
	if _, err := page.Goto(scopedURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
		t.Fatal(err)
	}
	if page.URL() != fixture.url {
		t.Fatalf("lost session did not return to recovery root: %s", page.URL())
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Print a fresh terminal code", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.GetByRole("status"), "printed in the terminal")
	terminalLine := strings.TrimSpace(fixture.terminal.String())
	const prefix = "icot ui replacement access code: "
	if !strings.HasPrefix(terminalLine, prefix) {
		t.Fatalf("replacement terminal output = %q", terminalLine)
	}
	replacementCode := strings.TrimPrefix(terminalLine, prefix)
	content, err := page.Locator("body").TextContent()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, replacementCode) || strings.Contains(page.URL(), replacementCode) {
		t.Fatalf("replacement code escaped into browser content or URL")
	}
	if err := page.Locator(`input[name="code"]`).Fill(replacementCode); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Continue", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "Connected")
}

func TestPhaseCBrowserSuccessfulRoundFocusesNextFrontier(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	next := phaseCFrontierSnapshot()
	next.Frontier = []elicitor.QuestionPlan{{
		ID: "source.selection", Prompt: "Which reviewed source should be used?", Slots: []string{"source.selection"}, Required: true, Priority: 80,
	}}
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot(), proposal: next}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	if err := page.GetByLabel("Your answer for: What outcome should this workflow achieve?").Fill("reviewed outcome"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByLabel("Your answer for: Who starts the workflow and when?").Fill("operator | on demand"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.GetByLabel("Your answer for: Which reviewed source should be used?"))
	waitForActiveID(t, page, "frontier-answer-1")
	waitForLocatorText(t, page.Locator("#mutation-status"), "Continue with the next authoring question")
}

func TestPhaseCBrowserStructuredChoiceAndDeferral(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	snapshot := engine.Snapshot{
		Frontier: []elicitor.QuestionPlan{
			{ID: "source.remote_lookup", Prompt: "Allow remote lookup?", Required: true},
			{ID: "workflow.fallback", Prompt: "What should happen if the step fails?", Required: true},
		},
		QuestionControls: []elicitor.QuestionControl{
			{QuestionID: "source.remote_lookup", InputKind: elicitor.QuestionInputChoice, Options: []elicitor.QuestionOption{{Value: "never", Label: "Never use the network"}, {Value: "allow", Label: "Allow one lookup"}}},
			{QuestionID: "workflow.fallback", InputKind: elicitor.QuestionInputText, Deferrable: true, Syntax: "Describe a bounded fallback."},
		},
		Readiness:        []elicitor.ReadinessIssue{{Code: "review", Severity: "blocking", Message: "Answer the current frontier."}},
		ProposedActions:  []elicitor.FileAction{},
		WriteConflicts:   []engine.WriteConflict{},
		SourceCandidates: engine.SourceCandidates{},
	}
	browserEngine := &phaseCBrowserEngine{snapshot: snapshot, proposal: phaseCProposalSnapshot(true, false)}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	lookup := page.GetByLabel("Your answer for: Allow remote lookup?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	values := []string{"never"}
	if _, err := lookup.SelectOption(playwright.SelectOptionValues{Values: &values}); err != nil {
		t.Fatal(err)
	}
	deferToggle := page.GetByLabel("Defer this decision with an explicit owner and unblock plan.", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := deferToggle.Check(); err != nil {
		t.Fatal(err)
	}
	for label, value := range map[string]string{
		"Owner": "API owner", "Impact of deferring": "draft remains incomplete", "Unblock condition": "provider publishes a spec", "Suggested next action": "add the reviewed source",
	} {
		if err := page.GetByLabel(label, playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)}).Fill(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve incomplete draft", Exact: playwright.Bool(true)}))
	rounds, _, answers, _ := browserEngine.mutationRecord()
	if rounds != 1 || len(answers) != 2 || answers[0].Value != "never" || answers[1].Value != "defer:API owner | draft remains incomplete | provider publishes a spec | add the reviewed source" {
		t.Fatalf("structured round = calls %d answers %#v", rounds, answers)
	}
}

func TestPhaseCBrowserReopensSettledAnswerBeforeApproval(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	initial := phaseCProposalSnapshot(false, false)
	initial.RevisableDecisions = []elicitor.RevisableDecision{{
		QuestionID: "boundary.actor_trigger", Prompt: "Who starts the workflow and when?", Slots: []string{"boundary.actor", "boundary.trigger"},
		Value: "operator | on demand", Impact: "Reopening clears this value and re-runs readiness.",
	}}
	reopened := engine.Snapshot{
		Boundary: elicitor.WorkflowBoundary{Outcome: "Create a reviewed report", SuccessEvidence: []string{"report is available"}},
		Frontier: []elicitor.QuestionPlan{{
			ID: "boundary.actor_trigger", Prompt: "Who starts the workflow and when?", Slots: []string{"boundary.actor", "boundary.trigger"}, Required: true,
		}},
		QuestionControls: []elicitor.QuestionControl{{QuestionID: "boundary.actor_trigger", InputKind: elicitor.QuestionInputText, Syntax: "actor | trigger"}},
		Readiness:        []elicitor.ReadinessIssue{{Code: "actor_trigger_required", Severity: "blocking", Message: "Replace the reopened actor and trigger."}},
	}
	replacement := phaseCProposalSnapshot(false, false)
	replacement.RevisableDecisions = []elicitor.RevisableDecision{{
		QuestionID: "boundary.actor_trigger", Prompt: "Who starts the workflow and when?", Value: "reviewer | after approval",
	}}
	browserEngine := &phaseCBrowserEngine{snapshot: initial, reopenProposal: reopened, proposal: replacement}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	reopenButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Reopen answer for Who starts the workflow and when?", Exact: playwright.Bool(true)})
	requireVisible(t, reopenButton)
	requireEnabled(t, reopenButton, true)
	waitForLocatorText(t, page.Locator("#revisions-list"), "operator | on demand")
	if err := reopenButton.Click(); err != nil {
		t.Fatal(err)
	}
	answer := page.GetByLabel("Your answer for: Who starts the workflow and when?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	requireVisible(t, answer)
	waitForActiveID(t, page, "frontier-answer-1")
	waitForLocatorText(t, page.Locator("#mutation-status"), "Answer reopened")
	if visible, err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", IncludeHidden: playwright.Bool(true)}).IsVisible(); err != nil || visible {
		t.Fatalf("approval visible while replacement pending = %t, %v", visible, err)
	}
	if err := answer.Fill("reviewer | after approval"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true)}))
	waitForLocatorText(t, page.Locator("#revisions-list"), "reviewer | after approval")
	if calls, questionID := browserEngine.reopenRecord(); calls != 1 || questionID != "boundary.actor_trigger" {
		t.Fatalf("reopen calls = %d, question = %q", calls, questionID)
	}
	rounds, _, answers, _ := browserEngine.mutationRecord()
	if rounds != 1 || len(answers) != 1 || answers[0].QuestionID != "boundary.actor_trigger" || answers[0].Value != "reviewer | after approval" {
		t.Fatalf("replacement round = calls %d answers %#v", rounds, answers)
	}
}

func TestPhaseCBrowserIncompleteApprovalIsExplicit(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	proposal := phaseCProposalSnapshot(true, false)
	browserEngine := &phaseCBrowserEngine{snapshot: proposal, proposal: proposal}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	incomplete := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve incomplete draft", Exact: playwright.Bool(true)})
	requireVisible(t, incomplete)
	if visible, err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true), IncludeHidden: playwright.Bool(true)}).IsVisible(); err != nil || visible {
		t.Fatalf("final approval visible for incomplete proposal = %t, %v", visible, err)
	}
	if err := page.GetByLabel("I reviewed the preview, readiness findings, proposed actions, and conflicts.").Check(); err != nil {
		t.Fatal(err)
	}
	if err := incomplete.Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.Locator("#package-section"))
	waitForLocatorText(t, page.Locator("#package-status"), "Ready to build")
	_, approvals, _, approval := browserEngine.mutationRecord()
	if approvals != 1 || !approval.HumanApproved || !approval.ApproveIncomplete || approval.AllowOverwrite {
		t.Fatalf("incomplete approval = %#v (calls %d)", approval, approvals)
	}
}

func TestPhaseCBrowserStaleRevisionPreservesUnsentAnswersAndDriftLocks(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false)}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	outcome := page.GetByLabel("Your answer for: What outcome should this workflow achieve?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := outcome.Fill("unsent operator answer"); err != nil {
		t.Fatal(err)
	}
	revision, err := page.Locator("#revision").TextContent()
	if err != nil {
		t.Fatal(err)
	}
	postRoundFromSecondClient(t, fixture, revision)
	requireVisible(t, page.Locator("#state-warning"))
	if value, err := outcome.InputValue(); err != nil || value != "unsent operator answer" {
		t.Fatalf("stale input = %q, %v", value, err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Discard the old form and review latest", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.Locator("#unsent-section"))
	waitForLocatorText(t, page.Locator("#unsent-answers"), "unsent operator answer")
	requireVisible(t, page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true)}))

	browserEngine.setWorkspace(engine.WorkspaceStatus{ExternallyModified: true})
	waitForLocatorText(t, page.Locator("#connection"), "restart required")
	requireVisible(t, page.Locator("#workspace-warning"))
	if disabled, err := page.Locator("#review-confirmed").IsDisabled(); err != nil || !disabled {
		t.Fatalf("drift review control disabled = %t, %v", disabled, err)
	}
}

func TestPhaseCBrowserDirtyAnswersBlockAcquisitionAndSurviveCaptureUpdate(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false)}
	fixture := newPhaseCBrowserFixtureWithConfig(t, browserEngine, func(config *HandlerConfig) {
		config.PrivateRoot = "/tmp/private"
		config.DoctorBrowser = func(context.Context, string, string) (browserauthor.DoctorReport, error) {
			return browserauthor.DoctorReport{Version: browserauthor.DoctorVersion, Engine: browserauthor.EngineChromium, DriverReady: true, BrowserReady: true}, nil
		}
	})
	page := newPhaseCPage(t, browser, fixture)

	outcome := page.GetByLabel("Your answer for: What outcome should this workflow achieve?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := outcome.Fill("unsent answer survives capture state"); err != nil {
		t.Fatal(err)
	}
	for _, selector := range []string{"#journey-api", "#source-file", "#browser-preflight", "#capture-url"} {
		if disabled, err := page.Locator(selector).IsDisabled(); err != nil || !disabled {
			t.Fatalf("dirty acquisition control %s disabled = %t, %v", selector, disabled, err)
		}
	}
	current := getPhaseCSnapshot(t, fixture)
	postBrowserPreflightFromSecondClient(t, fixture, current)
	requireVisible(t, page.Locator("#state-warning"))
	waitForLocatorText(t, page.Locator("#connection"), "capture update requires review")
	if value, err := outcome.InputValue(); err != nil || value != "unsent answer survives capture state" {
		t.Fatalf("answer after capture-only update = %q, %v", value, err)
	}
}

func TestPhaseCBrowserRetryableFailureRequiresExplicitRetry(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{
		snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false), roundErrOnce: true,
		roundErr: &engine.Failure{Class: engine.FailureOperational, Code: "engine_operation_failed", Cause: errors.New("temporary source refresh failure")},
	}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	outcome := page.GetByLabel("Your answer for: What outcome should this workflow achieve?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	actor := page.GetByLabel("Your answer for: Who starts the workflow and when?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := outcome.Fill("retry-safe outcome"); err != nil {
		t.Fatal(err)
	}
	if err := actor.Fill("operator | retry"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	retry := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Retry this request", Exact: playwright.Bool(true)})
	requireVisible(t, retry)
	requireVisible(t, page.Locator("#error-request-row"))
	requestID, err := page.Locator("#error-request-id").TextContent()
	if err != nil || len(requestID) != 32 {
		t.Fatalf("request ID = %q, %v", requestID, err)
	}
	if value, err := outcome.InputValue(); err != nil || value != "retry-safe outcome" {
		t.Fatalf("retryable failure lost input = %q, %v", value, err)
	}
	rounds, _, _, _ := browserEngine.mutationRecord()
	if rounds != 1 {
		t.Fatalf("retryable POST was automatically retried: %d calls", rounds)
	}
	if err := retry.Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true)}))
	rounds, _, _, _ = browserEngine.mutationRecord()
	if rounds != 2 {
		t.Fatalf("explicit retry calls = %d, want 2", rounds)
	}
}

func TestPhaseCBrowserDomainRejectionRetainsEditableRound(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{
		snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false), roundErrOnce: true,
		roundErr: &engine.Failure{Class: engine.FailureRejected, Code: "engine_rejected", Cause: authoring.WithQuestionID("boundary.actor_trigger", errors.New("answer does not satisfy the reviewed contract"))},
	}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	outcome := page.GetByLabel("Your answer for: What outcome should this workflow achieve?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	actor := page.GetByLabel("Your answer for: Who starts the workflow and when?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := outcome.Fill("rejected outcome"); err != nil {
		t.Fatal(err)
	}
	if err := actor.Fill("operator | rejected"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#error-message"), "does not satisfy")
	if invalid, err := actor.GetAttribute("aria-invalid"); err != nil || invalid != "true" {
		t.Fatalf("rejected actor aria-invalid = %q, %v", invalid, err)
	}
	waitForActiveID(t, page, "frontier-answer-2")
	if visible, err := page.Locator("#retry-mutation").IsVisible(); err != nil || visible {
		t.Fatalf("domain rejection offered retry = %t, %v", visible, err)
	}
	if value, err := outcome.InputValue(); err != nil || value != "rejected outcome" {
		t.Fatalf("domain rejection lost input = %q, %v", value, err)
	}
	if err := actor.Fill("operator | corrected"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	requireVisible(t, page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true)}))
}

func TestPhaseCBrowserDiscardsPollsOverlappingMutation(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{
		snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false), roundErrOnce: true,
		roundErr: &engine.Failure{Class: engine.FailureRejected, Code: "engine_rejected", Cause: errors.New("overlapped answer rejected")},
	}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	outcome := page.GetByLabel("Your answer for: What outcome should this workflow achieve?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	actor := page.GetByLabel("Your answer for: Who starts the workflow and when?", playwright.PageGetByLabelOptions{Exact: playwright.Bool(true)})
	if err := outcome.Fill("preserve this overlapped answer"); err != nil {
		t.Fatal(err)
	}
	if err := actor.Fill("operator | first attempt"); err != nil {
		t.Fatal(err)
	}

	started, completed, release := fixture.delay.arm(t)
	if _, err := page.Evaluate(`document.dispatchEvent(new Event("visibilitychange"))`); err != nil {
		t.Fatal(err)
	}
	waitForPhaseCSignal(t, started, "overlapping rejected-request poll")
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#error-message"), "overlapped answer rejected")
	release()
	waitForPhaseCSignal(t, completed, "delivery of rejected-request poll")
	time.Sleep(100 * time.Millisecond)
	if value, err := outcome.InputValue(); err != nil || value != "preserve this overlapped answer" {
		t.Fatalf("late poll replaced rejected answer = %q, %v", value, err)
	}
	if value, err := actor.InputValue(); err != nil || value != "operator | first attempt" {
		t.Fatalf("late poll replaced rejected actor = %q, %v", value, err)
	}

	if err := actor.Fill("operator | corrected attempt"); err != nil {
		t.Fatal(err)
	}
	started, completed, release = fixture.delay.arm(t)
	if _, err := page.Evaluate(`document.dispatchEvent(new Event("visibilitychange"))`); err != nil {
		t.Fatal(err)
	}
	waitForPhaseCSignal(t, started, "overlapping successful-request poll")
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	finalButton := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Approve final artifacts", Exact: playwright.Bool(true)})
	requireVisible(t, finalButton)
	release()
	waitForPhaseCSignal(t, completed, "delivery of successful-request poll")
	time.Sleep(100 * time.Millisecond)
	requireVisible(t, finalButton)
	waitForLocatorText(t, page.Locator("#project-preview"), "Reviewed project")
	if visible, err := outcome.IsVisible(); err != nil || visible {
		t.Fatalf("late poll restored old frontier after successful mutation = %t, %v", visible, err)
	}
}

func TestPhaseCBrowserIndeterminateFailureLocksMutation(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{
		snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false), roundErrOnce: true,
		roundErr: &engine.Failure{Class: engine.FailureIndeterminate, Code: "transaction_indeterminate", Cause: errors.New("rollback did not complete")},
	}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page := newPhaseCPage(t, browser, fixture)

	if err := page.GetByLabel("Your answer for: What outcome should this workflow achieve?").Fill("indeterminate outcome"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByLabel("Your answer for: Who starts the workflow and when?").Fill("operator | uncertain"); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Submit complete round", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#error-message"), "indeterminate")
	if visible, err := page.Locator("#retry-mutation").IsVisible(); err != nil || visible {
		t.Fatalf("indeterminate failure offered retry = %t, %v", visible, err)
	}
	if disabled, err := page.Locator("#round-submit").IsDisabled(); err != nil || !disabled {
		t.Fatalf("indeterminate failure locked round = %t, %v", disabled, err)
	}
}

func TestPhaseCBrowserPollingBackoffAndVisibility(t *testing.T) {
	_, browser := launchPhaseCBrowser(t)
	browserEngine := &phaseCBrowserEngine{snapshot: phaseCFrontierSnapshot(), proposal: phaseCProposalSnapshot(false, false)}
	fixture := newPhaseCBrowserFixture(t, browserEngine)
	page, err := browser.NewPage(playwright.BrowserNewPageOptions{Viewport: &playwright.Size{Width: 360, Height: 760}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = page.Close() })
	if err := page.Clock().Install(); err != nil {
		t.Fatal(err)
	}
	page.OnPageError(func(err error) { t.Errorf("iCoT UI page error: %v", err) })
	var requests atomic.Int64
	page.OnRequest(func(request playwright.Request) {
		if strings.HasSuffix(request.URL(), "/api/v4/snapshot") {
			requests.Add(1)
		}
	})
	if _, err := page.Goto(fixture.url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
		t.Fatal(err)
	}
	if err := page.Locator(`input[name="code"]`).Fill(fixture.accessCode); err != nil {
		t.Fatal(err)
	}
	if err := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Continue", Exact: playwright.Bool(true)}).Click(); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "Connected")
	initialRequests := requests.Load()
	browserEngine.setWorkspaceError(errors.New("temporary workspace inspection failure"))
	if err := page.Clock().RunFor(2000); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "retrying in 2 seconds")
	if requests.Load() != initialRequests+1 {
		t.Fatalf("first failed poll requests = %d, initial %d", requests.Load(), initialRequests)
	}
	if err := page.Clock().RunFor(1999); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != initialRequests+1 {
		t.Fatalf("poll fired before backoff elapsed: %d", requests.Load())
	}
	if err := page.Clock().RunFor(1); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "retrying in 4 seconds")
	if err := page.Clock().RunFor(4000); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "retrying in 8 seconds")
	if err := page.Clock().RunFor(8000); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "retrying in 16 seconds")
	if err := page.Clock().RunFor(16000); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "retrying in 30 seconds")

	if _, err := page.Evaluate(`Object.defineProperty(document, "hidden", {configurable: true, get: () => true}); document.dispatchEvent(new Event("visibilitychange"));`); err != nil {
		t.Fatal(err)
	}
	hiddenRequests := requests.Load()
	if err := page.Clock().RunFor(60000); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != hiddenRequests {
		t.Fatalf("hidden page continued polling: before %d after %d", hiddenRequests, requests.Load())
	}
	browserEngine.setWorkspaceError(nil)
	if _, err := page.Evaluate(`Object.defineProperty(document, "hidden", {configurable: true, get: () => false}); document.dispatchEvent(new Event("visibilitychange"));`); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "Connected")
	if requests.Load() != hiddenRequests+1 {
		t.Fatalf("visible page did not refresh immediately: before %d after %d", hiddenRequests, requests.Load())
	}
	if err := page.Clock().RunFor(2000); err != nil {
		t.Fatal(err)
	}
	waitForLocatorText(t, page.Locator("#connection"), "no changes")

	noOverflow, err := page.Evaluate("document.documentElement.scrollWidth <= document.documentElement.clientWidth")
	if err != nil || noOverflow != true {
		overflowing, _ := page.Evaluate(`({document: {scrollWidth: document.documentElement.scrollWidth, clientWidth: document.documentElement.clientWidth}, body: {scrollWidth: document.body.scrollWidth, clientWidth: document.body.clientWidth}, nodes: Array.from(document.querySelectorAll("*")).filter((node) => node.scrollWidth > node.clientWidth + 1 && getComputedStyle(node).overflowX === "visible").map((node) => ({tag: node.tagName, id: node.id, className: node.className, scrollWidth: node.scrollWidth, clientWidth: node.clientWidth})).slice(0, 12)})`)
		t.Fatalf("360px layout horizontal overflow = %#v, %v; nodes=%#v", noOverflow, err, overflowing)
	}
	mobileActionDisplay, err := page.Evaluate(`getComputedStyle(document.querySelector("#actions-body td")).display`)
	if err != nil || mobileActionDisplay != "grid" {
		t.Fatalf("360px action rows did not reflow to labeled records: display=%#v, %v", mobileActionDisplay, err)
	}
	workspaceDetailsOpen, err := page.Evaluate(`document.querySelector("#workspace-details").open`)
	if err != nil || workspaceDetailsOpen != false {
		t.Fatalf("technical workspace details should be collapsed by default: open=%#v, %v", workspaceDetailsOpen, err)
	}
}

func postRoundFromSecondClient(t *testing.T, fixture *phaseCBrowserFixture, revision string) {
	t.Helper()
	body, err := json.Marshal(roundRequest{Revision: revision, Answers: []roundAnswer{
		{QuestionID: "boundary.outcome", Value: "second client outcome"},
		{QuestionID: "boundary.actor_trigger", Value: "second client | scheduled"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://"+fixture.authority+"/api/v4/round", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+fixture.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("second-client round = %d", response.StatusCode)
	}
}

func getPhaseCSnapshot(t *testing.T, fixture *phaseCBrowserFixture) Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, "http://"+fixture.authority+"/api/v4/snapshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+fixture.token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("second-client snapshot = %d", response.StatusCode)
	}
	var payload Response
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func postBrowserPreflightFromSecondClient(t *testing.T, fixture *phaseCBrowserFixture, current Response) {
	t.Helper()
	body, err := json.Marshal(captureMutationRequest{Revision: current.Revision, CaptureRevision: current.CaptureRevision})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, "http://"+fixture.authority+"/api/v4/browser/preflight", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+fixture.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("second-client browser preflight = %d", response.StatusCode)
	}
}
