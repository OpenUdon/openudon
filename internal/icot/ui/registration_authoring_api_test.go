package ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorresult"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	icotengine "github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
)

type fakeRegistrationAuthoringSession struct {
	events   chan browserauthor.RegistrationEvent
	commands chan browserauthor.RegistrationCommand
	once     sync.Once
	mu       sync.Mutex
	canceled bool
	sendErr  error
}

func newFakeRegistrationAuthoringSession() *fakeRegistrationAuthoringSession {
	return &fakeRegistrationAuthoringSession{
		events: make(chan browserauthor.RegistrationEvent, 8), commands: make(chan browserauthor.RegistrationCommand, 8),
	}
}

func (f *fakeRegistrationAuthoringSession) Events() <-chan browserauthor.RegistrationEvent {
	return f.events
}

func (f *fakeRegistrationAuthoringSession) Send(ctx context.Context, command browserauthor.RegistrationCommand) error {
	f.mu.Lock()
	err := f.sendErr
	f.mu.Unlock()
	if err != nil {
		return err
	}
	select {
	case f.commands <- command:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *fakeRegistrationAuthoringSession) Cancel() {
	f.mu.Lock()
	f.canceled = true
	f.mu.Unlock()
	f.once.Do(func() { close(f.events) })
}

func (f *fakeRegistrationAuthoringSession) close() { f.once.Do(func() { close(f.events) }) }

func TestRegistrationAuthoringAPILifecycleIsAuthenticatedRevisionBoundAndQuerySafe(t *testing.T) {
	fake := &fakeEngine{}
	session := newFakeRegistrationAuthoringSession()
	var configured browserauthor.RegistrationConfig
	handler, err := NewHandler(HandlerConfig{
		Context: context.Background(), Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example",
		Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: "/tmp/private",
		BrowserTransactions: newFakeBrowserTransactions(),
		StartRegistration: func(_ context.Context, config browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			configured = config
			return session, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := currentResponse(t, handler)
	startBody := registrationStartBody(initial, "https://app.example.test/register?action=startnew")
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", startBody, "application/json", false); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized start = %d %s", response.Code, response.Body.String())
	}
	if response := doRequest(handler, http.MethodGet, "/api/v4/registration-authoring/start", "", "", true); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("start method response = %d allow=%q", response.Code, response.Header().Get("Allow"))
	}
	stale := strings.Replace(startBody, initial.RegistrationRevision, "sha256:stale", 1)
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", stale, "application/json", true); response.Code != http.StatusConflict {
		t.Fatalf("stale start = %d %s", response.Code, response.Body.String())
	}
	started := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", startBody, "application/json", true)
	if started.Code != http.StatusAccepted || strings.Contains(started.Body.String(), "action=startnew") || strings.Contains(started.Body.String(), "startnew") {
		t.Fatalf("start response = %d %s", started.Code, started.Body.String())
	}
	if configured.Protocol != registrationauthorsession.ProtocolV2 || configured.PrivateRoot != "/tmp/private" || !strings.HasPrefix(configured.TransactionID, "registration-") {
		t.Fatalf("registration config = %#v", configured)
	}

	session.events <- browserauthor.RegistrationEvent{State: "ready"}
	select {
	case command := <-session.commands:
		if command.Type != "start" || command.URL != "https://app.example.test/register?action=startnew" || len(command.Origins) != 1 {
			t.Fatalf("private start command = %#v", command)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for private start command")
	}
	session.events <- browserauthor.RegistrationEvent{State: "observing", Phase: "observing", Bounds: &registrationauthorsession.Bounds{
		NavigationTimeoutMS: 20_000, TotalTimeoutMS: 300_000, MaxRequests: 256, MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128,
	}}
	observing := waitForRegistrationState(t, handler, "observing")
	if observing.RegistrationRevision == initial.RegistrationRevision || strings.Contains(mustJSON(t, observing), "action=startnew") {
		t.Fatalf("observing response did not advance safely: %s", mustJSON(t, observing))
	}

	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", `{"revision":"`+observing.Revision+`","registration_revision":"sha256:stale","type":"observe"}`, "application/json", true); response.Code != http.StatusConflict {
		t.Fatalf("stale command = %d %s", response.Code, response.Body.String())
	}
	for _, forbidden := range []string{"credential", "verification", "profile", "value"} {
		body := `{"revision":"` + observing.Revision + `","registration_revision":"` + observing.RegistrationRevision + `","type":"observe","` + forbidden + `":"do-not-retain"}`
		response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", body, "application/json", true)
		if response.Code != http.StatusBadRequest || strings.Contains(response.Body.String(), "do-not-retain") {
			t.Fatalf("forbidden %s response = %d %s", forbidden, response.Code, response.Body.String())
		}
	}
	commandResponse := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", `{"revision":"`+observing.Revision+`","registration_revision":"`+observing.RegistrationRevision+`","type":"observe"}`, "application/json", true)
	if commandResponse.Code != http.StatusAccepted {
		t.Fatalf("observe command = %d %s", commandResponse.Code, commandResponse.Body.String())
	}
	if command := <-session.commands; command.Type != "observe" {
		t.Fatalf("observe command = %#v", command)
	}
	session.events <- browserauthor.RegistrationEvent{State: "observation", Phase: "observing", Observation: &registrationauthorsession.Observation{
		Generation: 1, Origin: "https://app.example.test", Path: "/register",
		Candidates: []registrationauthorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "button", Label: "Register", Matches: 1}}, Diagnostics: []string{},
	}}
	observed := waitForRegistrationState(t, handler, "observation")
	if strings.Contains(mustJSON(t, observed), "action=startnew") || strings.Contains(mustJSON(t, observed), "startnew") {
		t.Fatalf("observation leaked retained query: %s", mustJSON(t, observed))
	}
	navigateBody := `{"revision":"` + observed.Revision + `","registration_revision":"` + observed.RegistrationRevision + `","type":"navigate","method":"GET","url":"https://app.example.test/register?action=startnew"}`
	navigated := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", navigateBody, "application/json", true)
	if navigated.Code != http.StatusAccepted || strings.Contains(navigated.Body.String(), "action=startnew") || strings.Contains(navigated.Body.String(), "startnew") {
		t.Fatalf("navigate response = %d %s", navigated.Code, navigated.Body.String())
	}
	if command := <-session.commands; command.Type != "navigate" || !strings.Contains(command.URL, "action=startnew") {
		t.Fatalf("private navigate command = %#v", command)
	}

	current := currentResponse(t, handler)
	canceled := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/cancel", `{"registration_revision":"`+current.RegistrationRevision+`"}`, "application/json", true)
	if canceled.Code != http.StatusAccepted {
		t.Fatalf("cancel response = %d %s", canceled.Code, canceled.Body.String())
	}
	final := waitForRegistrationState(t, handler, "canceled")
	if final.RegistrationAuthoring.ResultReady || strings.Contains(mustJSON(t, final), "action=startnew") {
		t.Fatalf("canceled state = %s", mustJSON(t, final))
	}
}

func TestRegistrationAuthoringAPIEnforcesSingleSessionAndContainmentFailure(t *testing.T) {
	fake := &fakeEngine{}
	session := newFakeRegistrationAuthoringSession()
	handler, err := NewHandler(HandlerConfig{
		Context: context.Background(), Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example",
		Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: "/tmp/private",
		BrowserTransactions: newFakeBrowserTransactions(),
		StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return session, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := currentResponse(t, handler)
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", registrationStartBody(initial, "https://app.example.test/register"), "application/json", true); response.Code != http.StatusAccepted {
		t.Fatalf("start = %d %s", response.Code, response.Body.String())
	}
	active := currentResponse(t, handler)
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", registrationStartBody(active, "https://app.example.test/register"), "application/json", true); response.Code != http.StatusConflict {
		t.Fatalf("second start = %d %s", response.Code, response.Body.String())
	}
	if response := doRequest(handler, http.MethodPost, "/api/v4/round", `{"revision":"`+active.Revision+`","answers":[]}`, "application/json", true); response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "registration_authoring_active") {
		t.Fatalf("concurrent authoring mutation = %d %s", response.Code, response.Body.String())
	}
	session.events <- browserauthor.RegistrationEvent{State: "failed", ErrorCode: "worker_teardown"}
	session.close()
	failed := waitForRegistrationState(t, handler, "failed")
	if !failed.RegistrationAuthoring.ContainmentFailed {
		t.Fatalf("containment failure = %s", mustJSON(t, failed))
	}
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", registrationStartBody(failed, "https://app.example.test/register"), "application/json", true); response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "browser_teardown_failed") {
		t.Fatalf("restart after containment failure = %d %s", response.Code, response.Body.String())
	}
}

func TestRegistrationAuthoringAPIStartFailureIsClosed(t *testing.T) {
	fake := &fakeEngine{}
	handler, err := NewHandler(HandlerConfig{
		Context: context.Background(), Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example",
		Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: "/tmp/private",
		BrowserTransactions: newFakeBrowserTransactions(),
		StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return nil, errors.New("private path and do-not-retain")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", registrationStartBody(initial, "https://app.example.test/register?action=startnew"), "application/json", true)
	if response.Code != http.StatusUnprocessableEntity || strings.Contains(response.Body.String(), "private path") || strings.Contains(response.Body.String(), "action=startnew") || strings.Contains(response.Body.String(), "startnew") {
		t.Fatalf("start failure = %d %s", response.Code, response.Body.String())
	}
}

func TestRegistrationAuthoringAPIBuildsDraftServerSideThenRequiresExplicitReviewAndFinish(t *testing.T) {
	fake := &fakeEngine{}
	session := newFakeRegistrationAuthoringSession()
	clock := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	handler, err := NewHandler(HandlerConfig{
		Context: context.Background(), Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example",
		Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: "/tmp/private", Now: func() time.Time { return clock },
		BrowserTransactions: newFakeBrowserTransactions(),
		StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return session, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := currentResponse(t, handler)
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", registrationStartBody(initial, "https://app.example.test/register?action=startnew"), "application/json", true); response.Code != http.StatusAccepted {
		t.Fatalf("start = %d %s", response.Code, response.Body.String())
	}
	session.events <- browserauthor.RegistrationEvent{State: "ready"}
	<-session.commands
	session.events <- browserauthor.RegistrationEvent{State: "observing", Phase: "observing", Bounds: &registrationauthorsession.Bounds{
		NavigationTimeoutMS: 20_000, TotalTimeoutMS: 300_000, MaxRequests: 256, MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128,
	}}
	observing := waitForRegistrationState(t, handler, "observing")
	observe := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", `{"revision":"`+observing.Revision+`","registration_revision":"`+observing.RegistrationRevision+`","type":"observe"}`, "application/json", true)
	if observe.Code != http.StatusAccepted {
		t.Fatalf("observe = %d %s", observe.Code, observe.Body.String())
	}
	<-session.commands
	observation := registrationDraftObservation()
	session.events <- browserauthor.RegistrationEvent{State: "observation", Phase: "observing", Observation: &observation}
	observed := waitForRegistrationState(t, handler, "observation")

	draftRequest := map[string]any{
		"revision": observed.Revision, "registration_revision": observed.RegistrationRevision, "type": "draft", "draft": validRegistrationDraftRequest(),
	}
	draftData, err := json.Marshal(draftRequest)
	if err != nil {
		t.Fatal(err)
	}
	drafted := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", string(draftData), "application/json", true)
	if drafted.Code != http.StatusOK || !strings.Contains(drafted.Body.String(), `"key":"action","value":"startnew"`) ||
		!strings.Contains(drafted.Body.String(), `"ambiguous_outcome":"stop_without_retry"`) ||
		!strings.Contains(drafted.Body.String(), `"review_kind":"operator_reviewed_deferred","observed_during_authoring":false,"runtime_proof_required":true`) {
		t.Fatalf("draft response = %d %s", drafted.Code, drafted.Body.String())
	}
	draftState := decodeResponse(t, drafted)
	if draftState.RegistrationAuthoring == nil || draftState.RegistrationAuthoring.Draft == nil || draftState.RegistrationAuthoring.Draft.ProfileSHA256 == "" {
		t.Fatalf("draft state = %#v", draftState.RegistrationAuthoring)
	}
	if response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", `{"revision":"`+draftState.Revision+`","registration_revision":"`+draftState.RegistrationRevision+`","type":"review","confirmed":false}`, "application/json", true); response.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed review = %d %s", response.Code, response.Body.String())
	}
	reviewed := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", `{"revision":"`+draftState.Revision+`","registration_revision":"`+draftState.RegistrationRevision+`","type":"review","confirmed":true}`, "application/json", true)
	if reviewed.Code != http.StatusAccepted {
		t.Fatalf("review command = %d %s", reviewed.Code, reviewed.Body.String())
	}
	command := <-session.commands
	if command.Type != "review" || !command.Confirmed || len(command.Profile) == 0 || len(command.CandidateIDs) != 6 || len(command.CredentialBindings) != 3 ||
		command.Flow != "create_dedicated_test_user" || command.CleanupDisposition != "delete_separately" || strings.Contains(string(command.Profile), "dedicated_test_identifier") {
		t.Fatalf("private review command = %#v profile=%s", command, command.Profile)
	}
	session.events <- browserauthor.RegistrationEvent{State: "reviewed", Phase: "reviewed"}
	reviewedState := waitForRegistrationState(t, handler, "reviewed")
	finish := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/command", `{"revision":"`+reviewedState.Revision+`","registration_revision":"`+reviewedState.RegistrationRevision+`","type":"finish","confirmed":true}`, "application/json", true)
	if finish.Code != http.StatusAccepted {
		t.Fatalf("finish command = %d %s", finish.Code, finish.Body.String())
	}
	if command := <-session.commands; command.Type != "finish" || !command.Confirmed || len(command.Profile) != 0 {
		t.Fatalf("finish command = %#v", command)
	}
	current := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/cancel", `{"registration_revision":"`+current.RegistrationRevision+`"}`, "application/json", true)
	if response.Code != http.StatusAccepted {
		t.Fatalf("cancel after finish assertion = %d %s", response.Code, response.Body.String())
	}
}

func TestRegistrationAuthoringCleanTeardownStartsTransactionV2AndSelectsReviewedVirtualSource(t *testing.T) {
	observedAt := time.Date(2026, 8, 26, 12, 5, 0, 0, time.UTC)
	clock := observedAt.Add(2 * time.Minute)
	candidate := registrationCandidateV2(t, observedAt)
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	testRoot, err := os.MkdirTemp(repoRoot, ".a21-registration-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	example := filepath.Join(testRoot, "example")
	authoringEngine, authoringSnapshot, err := icotengine.Open(context.Background(), icotengine.Config{
		ExampleDir: example, NetworkPolicy: "never", Now: func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	scratch, store := filepath.Join(root, "scratch"), filepath.Join(root, "store")
	for _, path := range []string{scratch, store} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	transactions, _, err := transactionengine.New(transactionengine.Config{
		Package: packagepipeline.CurrentOptions{ExampleDir: example, Scope: "examples/registration", ScratchParent: scratch, StoreDir: store},
		Now:     func() time.Time { return clock },
	})
	if err != nil {
		t.Fatal(err)
	}
	session := newFakeRegistrationAuthoringSession()
	var internalErrors strings.Builder
	handler, err := NewHandler(HandlerConfig{
		Context: context.Background(), Engine: authoringEngine, Snapshot: authoringSnapshot, ExampleDir: example,
		Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: t.TempDir(), Now: func() time.Time { return clock },
		BrowserTransactions: transactions, ErrOut: &internalErrors, RepoRoot: repoRoot,
		StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return session, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := currentResponse(t, handler)
	started := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", registrationStartBody(initial, "https://app.example.test/register?action=startnew"), "application/json", true)
	if started.Code != http.StatusAccepted {
		t.Fatalf("start = %d %s", started.Code, started.Body.String())
	}
	session.events <- browserauthor.RegistrationEvent{State: "ready"}
	<-session.commands
	session.events <- browserauthor.RegistrationEvent{State: "candidate", Candidate: candidate}
	session.close()
	pending := waitForRegistrationState(t, handler, "transaction_review")
	if pending.BrowserTransaction == nil || pending.BrowserTransaction.Transaction == nil || pending.BrowserTransaction.Transaction.Version != browsertransaction.VersionV2 || pending.BrowserTransaction.Transaction.State != browsertransaction.StateCandidate {
		t.Fatalf("pending registration transaction = %#v", pending.BrowserTransaction)
	}
	blockedRound := doRequest(handler, http.MethodPost, "/api/v4/round", `{"revision":"`+pending.Revision+`","answers":[]}`, "application/json", true)
	if blockedRound.Code != http.StatusConflict || !strings.Contains(blockedRound.Body.String(), `"code":"registration_authoring_active"`) {
		t.Fatalf("authoring mutation during transaction review = %d %s", blockedRound.Code, blockedRound.Body.String())
	}
	transactionSnapshot, err := transactions.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reviewData, err := json.Marshal(transactionengine.ReviewRequest{Authority: transactionengine.Authority{
		ExpectedRevision: transactionSnapshot.Revision, ExpectedTransactionSHA256: transactionSnapshot.TransactionSHA256, HumanApproved: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	reviewed := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/review", string(reviewData), "application/json", true)
	if reviewed.Code != http.StatusOK {
		t.Fatalf("transaction review = %d %s", reviewed.Code, reviewed.Body.String())
	}
	adopted := currentResponse(t, handler)
	if adopted.RegistrationAuthoring == nil || adopted.RegistrationAuthoring.State != "adopted" || len(adopted.Snapshot.SelectedSources) != 1 ||
		len(adopted.Snapshot.SourceCandidates.VirtualBrowser.Candidates) != 1 || !adopted.Snapshot.SourceCandidates.VirtualBrowser.Candidates[0].Selected ||
		adopted.Snapshot.SourceCandidates.VirtualBrowser.Candidates[0].Kind != browsertransaction.CandidateRegistration {
		t.Fatalf("adopted virtual source = %s", mustJSON(t, adopted))
	}
	for _, forbidden := range []string{"action=startnew", "startnew", "do-not-retain", "secret-value"} {
		if strings.Contains(mustJSON(t, adopted), forbidden) {
			t.Fatalf("adopted public state exposed %q: %s", forbidden, mustJSON(t, adopted))
		}
	}
	reviewedSnapshot, err := transactions.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	prepareData, err := json.Marshal(transactionengine.PrepareRequest{Authority: transactionengine.Authority{
		ExpectedRevision: reviewedSnapshot.Revision, ExpectedTransactionSHA256: reviewedSnapshot.TransactionSHA256, HumanApproved: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	prepare := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/prepare", string(prepareData), "application/json", true)
	if prepare.Code != http.StatusConflict || !strings.Contains(prepare.Body.String(), `"code":"registration_package_not_ready"`) {
		t.Fatalf("prepare before package handoff = %d %s", prepare.Code, prepare.Body.String())
	}

	authoring := adopted
	for round := 0; !authoring.Snapshot.ApprovalRequired && round < 16; round++ {
		if len(authoring.Snapshot.Frontier) == 0 {
			t.Fatalf("authoring stalled before approval: %s", mustJSON(t, authoring.Snapshot))
		}
		answers := make([]map[string]string, 0, len(authoring.Snapshot.Frontier))
		for _, question := range authoring.Snapshot.Frontier {
			value := strings.TrimSpace(question.Recommendation)
			if value == "" {
				value = strings.TrimSpace(question.SuggestedAnswer)
			}
			if value == "" {
				value = "reviewed registration package"
			}
			answers = append(answers, map[string]string{"question_id": question.ID, "value": value})
		}
		body, err := json.Marshal(map[string]any{"revision": authoring.Revision, "answers": answers})
		if err != nil {
			t.Fatal(err)
		}
		response := doRequest(handler, http.MethodPost, "/api/v4/round", string(body), "application/json", true)
		if response.Code != http.StatusOK {
			t.Fatalf("authoring round %d = %d %s", round+1, response.Code, response.Body.String())
		}
		authoring = decodeResponse(t, response)
	}
	if !authoring.Snapshot.ApprovalRequired || !authoring.Snapshot.Ready {
		t.Fatalf("registration package did not reach explicit approval: %s", mustJSON(t, authoring.Snapshot))
	}
	approveBody, err := json.Marshal(map[string]any{"revision": authoring.Revision, "human_approved": true})
	if err != nil {
		t.Fatal(err)
	}
	approvedResponse := doRequest(handler, http.MethodPost, "/api/v4/author/approve", string(approveBody), "application/json", true)
	if approvedResponse.Code != http.StatusOK {
		t.Fatalf("authoring approval = %d %s", approvedResponse.Code, approvedResponse.Body.String())
	}
	approved := decodeResponse(t, approvedResponse)
	for _, relative := range []string{"browser-registration/guided-registration.json", "browser-registration/guided-registration.review.json"} {
		if info, err := os.Stat(filepath.Join(example, filepath.FromSlash(relative))); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("reviewed registration artifact %s = %v, %v", relative, info, err)
		}
	}
	buildBody, err := json.Marshal(map[string]any{"revision": approved.Revision, "confirmed": true})
	if err != nil {
		t.Fatal(err)
	}
	builtResponse := doRequest(handler, http.MethodPost, "/api/v4/package/build", string(buildBody), "application/json", true)
	if builtResponse.Code != http.StatusOK {
		t.Fatalf("package build = %d %s; internal=%s", builtResponse.Code, builtResponse.Body.String(), internalErrors.String())
	}
	built := decodeResponse(t, builtResponse)
	if built.Lifecycle != lifecycleHandoffReady || built.Package == nil || built.Package.Status != "pass" {
		t.Fatalf("qualified package = %s", mustJSON(t, built))
	}
	reviewedSnapshot, err = transactions.Observe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	prepareData, err = json.Marshal(transactionengine.PrepareRequest{Authority: transactionengine.Authority{
		ExpectedRevision: reviewedSnapshot.Revision, ExpectedTransactionSHA256: reviewedSnapshot.TransactionSHA256, HumanApproved: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	preparedResponse := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/prepare", string(prepareData), "application/json", true)
	if preparedResponse.Code != http.StatusOK {
		t.Fatalf("transaction prepare/qualify = %d %s", preparedResponse.Code, preparedResponse.Body.String())
	}
	preparedSnapshot, err := transactions.Observe(context.Background())
	if err != nil || preparedSnapshot.Preparation == nil {
		t.Fatalf("prepared transaction = %#v, %v", preparedSnapshot, err)
	}
	promoteData, err := json.Marshal(transactionengine.PromoteRequest{
		Authority: transactionengine.Authority{
			ExpectedRevision: preparedSnapshot.Revision, ExpectedTransactionSHA256: preparedSnapshot.TransactionSHA256, HumanApproved: true,
		},
		ExpectedPreparationSHA256: preparedSnapshot.Preparation.PreparationSHA256, ExpectedQualificationSHA256: preparedSnapshot.Preparation.QualificationSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	promotedResponse := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/promote", string(promoteData), "application/json", true)
	if promotedResponse.Code != http.StatusOK {
		t.Fatalf("transaction promote = %d %s", promotedResponse.Code, promotedResponse.Body.String())
	}
	promoted := currentResponse(t, handler)
	if promoted.RegistrationAuthoring == nil || promoted.RegistrationAuthoring.State != "promoted" || promoted.BrowserTransaction == nil ||
		promoted.BrowserTransaction.Transaction == nil || promoted.BrowserTransaction.Transaction.State != browsertransaction.StatePromoted || promoted.BrowserTransaction.RuntimeExecutionSupported {
		t.Fatalf("promoted registration package = %s", mustJSON(t, promoted))
	}
}

func registrationCandidateV2(t *testing.T, now time.Time) *browsercandidate.Registration {
	t.Helper()
	observation := registrationDraftObservation()
	profileBytes, candidateIDs, bindings, disclosure, err := buildRegistrationDraft(
		validRegistrationDraftRequest(),
		registrationAuthoringStartRequest{ProfileID: "synthetic_registration", Origins: []string{"https://app.example.test"}},
		observation,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := registrationprofile.Parse(profileBytes)
	if err != nil {
		t.Fatal(err)
	}
	reviewedCandidates := make([]registrationauthorsession.ReviewedCandidate, 0, len(candidateIDs))
	for _, id := range candidateIDs {
		for _, observed := range observation.Candidates {
			if observed.ID == id {
				reviewedCandidates = append(reviewedCandidates, registrationauthorsession.ReviewedCandidate{
					ID: observed.ID, Generation: observation.Generation, Role: observed.Role, Label: observed.Label, Matches: observed.Matches,
				})
			}
		}
	}
	bounds := registrationauthorsession.Bounds{
		NavigationTimeoutMS: 20_000, TotalTimeoutMS: 300_000, MaxRequests: 256,
		MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128,
	}
	envelope, err := registrationauthorresult.Build(registrationauthorresult.BuildRequest{
		CreatedAt: now.Add(time.Minute),
		Completion: &registrationauthorsession.Completion{
			Protocol: registrationauthorsession.ProtocolV2, ProfileID: "synthetic_registration", Profile: *profile, ProfileBytes: profileBytes,
			ReviewedCandidates: reviewedCandidates, Flow: disclosure.Flow, CleanupDisposition: disclosure.CleanupDisposition,
			Origins: []string{"https://app.example.test"}, ObservedAt: now, Bounds: bounds, Observations: 1,
			Network: registrationauthorsession.NetworkSummary{Requests: 1, GETRequests: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	privateRoot := t.TempDir()
	if err := os.Chmod(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	inbox, err := browsercandidate.OpenPrivateInbox(privateRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer inbox.Close()
	if _, err := registrationauthorresult.WritePrivateExclusive(privateRoot, envelope); err != nil {
		t.Fatal(err)
	}
	candidate, err := inbox.AdoptNewRegistration(browsercandidate.AdoptRegistrationRequest{
		TransactionID: "guided-registration", CredentialBindings: bindings,
		Review: browsercandidate.RegistrationReview{
			Confirmed: true, ProfileID: envelope.Candidate.ProfileID, Flow: envelope.Flow.Name, SourceSHA256: envelope.Candidate.SourceDigest,
			ReviewedCandidates: reviewedCandidates, CleanupDisposition: envelope.CallPolicy.CleanupDisposition,
			Origins: append([]string(nil), envelope.Origins...), Bounds: envelope.Bounds, Observations: envelope.Observations, MinimumRequests: 1,
		},
		AssessedAt: now.Add(time.Minute + time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	return candidate
}

func registrationStartBody(response Response, target string) string {
	payload := map[string]any{
		"revision": response.Revision, "registration_revision": response.RegistrationRevision,
		"profile_id": "synthetic_registration", "url": target, "origins": []string{"https://app.example.test"},
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func waitForRegistrationState(t *testing.T, handler http.Handler, want string) Response {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response := currentResponse(t, handler)
		if response.RegistrationAuthoring != nil && response.RegistrationAuthoring.State == want {
			return response
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for registration state %q", want)
	return Response{}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
