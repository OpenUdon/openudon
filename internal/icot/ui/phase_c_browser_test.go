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

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
)

type phaseCBrowserEngine struct {
	mu sync.Mutex

	snapshot      engine.Snapshot
	proposal      engine.Snapshot
	workspace     engine.WorkspaceStatus
	workspaceErr  error
	roundErr      error
	roundErrOnce  bool
	roundCalls    int
	approvalCalls int
	answers       []authoring.RoundAnswer
	approval      engine.Approval
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
	if !strings.HasSuffix(r.URL.Path, "/api/v2/snapshot") {
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

type phaseCBrowserFixture struct {
	engine     *phaseCBrowserEngine
	server     *http.Server
	listener   net.Listener
	token      string
	accessCode string
	authority  string
	url        string
	delay      *phaseCSnapshotDelay
}

func newPhaseCBrowserFixture(t *testing.T, browserEngine *phaseCBrowserEngine) *phaseCBrowserFixture {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	authority := listener.Addr().String()
	token := "phase-c-browser-test-capability"
	accessCode := "0123456789AB"
	handler, err := NewHandler(HandlerConfig{
		Engine: browserEngine, Snapshot: browserEngine.snapshot, ExampleDir: "/tmp/phase-c-browser", Token: token, AccessCode: accessCode, Authority: authority,
	})
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
		url: "http://" + authority + "/", delay: delay,
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
		Boundary:         elicitor.WorkflowBoundary{Outcome: "Create a reviewed report", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"report is available"}, Confirmed: true},
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
		SourceCandidates: engine.SourceCandidates{},
	}
}

func launchPhaseCBrowser(t *testing.T) (*playwright.Playwright, playwright.Browser) {
	t.Helper()
	pw, err := playwright.Run()
	if err != nil {
		t.Fatalf("start Playwright-Go: %v (install with: go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6201.0 install chromium)", err)
	}
	sandbox := os.Getenv("OPENUDON_ICOT_UI_BROWSER_DISABLE_SANDBOX") != "1"
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
	if err := page.Locator(`button[type="submit"]`).Click(); err != nil {
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
	if err != nil || active != "workspace-details-toggle" {
		t.Fatalf("workspace disclosure keyboard order = %#v, %v", active, err)
	}
	if err := page.Keyboard().Press("Tab"); err != nil {
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
	requireVisible(t, page.Locator("#completion-banner"))
	waitForActiveID(t, page, "completion-banner")
	waitForLocatorText(t, page.Locator("#mutation-status"), "Approval committed")
	waitForLocatorText(t, page.Locator("#written-list"), "project.md")

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
		t.Fatalf("page horizontal overflow = %#v, %v", noOverflow, err)
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
	requireVisible(t, page.Locator("#completion-banner"))
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
		roundErr: &engine.Failure{Class: engine.FailureRejected, Code: "engine_rejected", Cause: authoring.WithQuestionID("actor", errors.New("answer does not satisfy the reviewed contract"))},
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
	var requests atomic.Int64
	page.OnRequest(func(request playwright.Request) {
		if strings.HasSuffix(request.URL(), "/api/v2/snapshot") {
			requests.Add(1)
		}
	})
	if _, err := page.Goto(fixture.url, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
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
	request, err := http.NewRequest(http.MethodPost, "http://"+fixture.authority+"/api/v2/round", bytes.NewReader(body))
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
