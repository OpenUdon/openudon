package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
)

func TestV4BrowserTransactionResourceAndV3RouteClosure(t *testing.T) {
	transactionEngine, initial, err := transactionengine.New(transactionengine.Config{
		Package: packagepipeline.CurrentOptions{ExampleDir: "/private/example", Scope: "examples/private", ScratchParent: "/private/scratch", StoreDir: "/private/store"},
		Now:     func() time.Time { return apiTransactionTime },
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(HandlerConfig{
		Engine: &fakeEngine{}, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
		BrowserTransactions: transactionEngine, Now: func() time.Time { return apiTransactionTime },
	})
	if err != nil {
		t.Fatal(err)
	}

	current := doRequest(handler, http.MethodGet, "/api/v4/browser-transactions/current", "", "", true)
	if current.Code != http.StatusOK || !strings.Contains(current.Body.String(), `"version":"openudon.icot-ui-api.v4"`) || !strings.Contains(current.Body.String(), initial.Revision) {
		t.Fatalf("initial transaction resource = %d %s", current.Code, current.Body.String())
	}
	transaction := apiRegistrationTransaction()
	transactionJSON, err := browsertransaction.CanonicalBytes(transaction)
	if err != nil {
		t.Fatal(err)
	}
	transactionDigest, _ := browsertransaction.Digest(transaction)
	requestJSON, err := json.Marshal(transactionengine.StartRequest{ExpectedRevision: initial.Revision, ExpectedTransactionSHA256: transactionDigest, TransactionJSON: transactionJSON})
	if err != nil {
		t.Fatal(err)
	}
	started := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/start", string(requestJSON), "application/json", true)
	if started.Code != http.StatusOK || !strings.Contains(started.Body.String(), `"kind":"registration"`) || !strings.Contains(started.Body.String(), `"runtime_execution_supported":false`) ||
		!strings.Contains(started.Body.String(), `"composition":"BRP"`) || !strings.Contains(started.Body.String(), `heuristic`) || !strings.Contains(started.Body.String(), `not data loss prevention`) ||
		!strings.Contains(started.Body.String(), `"freshness_check":"engine_rechecks_expires_at_before_review_and_prepare"`) || !strings.Contains(started.Body.String(), `"network_methods":["GET","HEAD"]`) || !strings.Contains(started.Body.String(), `"observation_status":"producer_accepted"`) ||
		!strings.Contains(started.Body.String(), `"approval_symbol_is_authority":false`) {
		t.Fatalf("started transaction = %d %s", started.Code, started.Body.String())
	}
	for _, forbidden := range []string{"/private/example", "/private/scratch", "/private/store", "result_path", "worker_output", "page_content"} {
		if strings.Contains(started.Body.String(), forbidden) {
			t.Fatalf("transaction response contains %q: %s", forbidden, started.Body.String())
		}
	}

	snapshot := doRequest(handler, http.MethodGet, "/api/v4/snapshot", "", "", true)
	if snapshot.Code != http.StatusOK || !strings.Contains(snapshot.Body.String(), `"browser_transaction"`) || !strings.Contains(snapshot.Body.String(), transactionDigest) {
		t.Fatalf("combined snapshot = %d %s", snapshot.Code, snapshot.Body.String())
	}
	for _, closed := range []string{
		"/api/v3/snapshot", "/api/v3/browser/preflight", "/api/v3/capture/start", "/api/v3/package/build",
		"/api/v4/browser-transactions/run", "/api/v4/browser-transactions/execute", "/api/v4/browser-transactions/registration/submit",
	} {
		response := doRequest(handler, http.MethodGet, closed, "", "", true)
		if response.Code != http.StatusNotFound {
			t.Fatalf("closed route %s = %d %s", closed, response.Code, response.Body.String())
		}
	}
}

func TestBrowserTransactionRoutesUseSharedTypedOperations(t *testing.T) {
	fake := newFakeBrowserTransactions()
	handler := newBrowserTransactionHandler(t, fake)
	cases := []struct {
		path string
		body string
		want transactionengine.Operation
	}{
		{"/api/v4/browser-transactions/start", `{"revision":"sha256:r","transaction_sha256":"sha256:t","transaction":{}}`, transactionengine.OperationStart},
		{"/api/v4/browser-transactions/review", `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true}`, transactionengine.OperationReview},
		{"/api/v4/browser-transactions/prepare", `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true}`, transactionengine.OperationPrepare},
		{"/api/v4/browser-transactions/promote", `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true,"preparation_sha256":"sha256:p","qualification_sha256":"sha256:q"}`, transactionengine.OperationPromote},
		{"/api/v4/browser-transactions/cancel", `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true}`, transactionengine.OperationCancel},
		{"/api/v4/browser-transactions/recovery/inspect", `{"revision":"sha256:r"}`, transactionengine.OperationInspectRecovery},
		{"/api/v4/browser-transactions/recovery/reconcile", `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true,"recovery_sha256":"sha256:x"}`, transactionengine.OperationRecover},
		{"/api/v4/browser-transactions/selected/inspect", `{"revision":"sha256:r","selection_sha256":"sha256:s"}`, transactionengine.OperationInspectSelected},
	}
	for _, test := range cases {
		t.Run(string(test.want), func(t *testing.T) {
			response := doRequest(handler, http.MethodPost, test.path, test.body, "application/json", true)
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"browser_transaction"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			if got := fake.lastOperation(); got != test.want {
				t.Fatalf("operation = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBrowserTransactionReviewIsKindSpecificAndSymbolic(t *testing.T) {
	transaction := apiRegistrationTransaction()
	transaction.Kind = browsertransaction.KindAuthenticationCapability
	transaction.Candidates = []browsertransaction.Candidate{
		{Kind: browsertransaction.CandidateAuthentication, Schema: "uws.browser-authentication.1.1", SourceSHA256: apiDigest("1"), ReviewSHA256: apiDigest("2")},
		{Kind: browsertransaction.CandidateCapability, Schema: "uws.browser.1.7", SourceSHA256: apiDigest("3"), ReviewSHA256: apiDigest("4")},
	}
	transaction.Provenance.ResultVersion = browsertransaction.ResultAuthenticatedAuthoringV2
	transaction.CredentialBindings = []browsertransaction.CredentialBinding{}
	transaction.Session = "account_session"
	snapshot := transactionengine.Snapshot{Version: transactionengine.Version, Revision: apiDigest("9"), Transaction: &transaction, AllowedOperations: []transactionengine.Operation{transactionengine.OperationObserve}}
	resource := browserTransactionResource(snapshot)
	if resource.Review == nil || resource.Review.Composition != "BAP+BCP" || resource.Review.Session != "account_session" || resource.Review.RegistrationAuthoring != nil || resource.Review.CredentialBindings == nil {
		t.Fatalf("BAP+BCP review = %#v", resource.Review)
	}
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "accessibility_labels") || !strings.Contains(string(data), `"credential_bindings":[]`) {
		t.Fatalf("BAP+BCP resource changed kind-specific disclosure: %s", data)
	}
}

func TestBrowserTransactionTransportBoundsAndTypedErrors(t *testing.T) {
	fake := newFakeBrowserTransactions()
	handler := newBrowserTransactionHandler(t, fake)
	if response := doRequest(handler, http.MethodGet, "/api/v4/browser-transactions/current", "", "", false); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized current = %d %s", response.Code, response.Body.String())
	}
	if response := doRequest(handler, http.MethodGet, "/api/v4/browser-transactions/start", "", "", true); response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("wrong method = %d %s", response.Code, response.Body.String())
	}
	for name, body := range map[string]string{
		"unknown":   `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true,"private_root":"/secret"}`,
		"duplicate": `{"revision":"sha256:r","revision":"sha256:changed","transaction_sha256":"sha256:t","human_approved":true}`,
		"deep":      `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true,"extra":` + strings.Repeat("[", MaxJSONDepth+2) + "0" + strings.Repeat("]", MaxJSONDepth+2) + `}`,
	} {
		t.Run(name, func(t *testing.T) {
			response := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/review", body, "application/json", true)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"malformed_request"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
	if fake.calls() != 0 {
		t.Fatalf("malformed requests invoked engine %d times", fake.calls())
	}
	if response := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/review", `{}`, "text/plain", true); response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported media = %d %s", response.Code, response.Body.String())
	}
	oversized := httptest.NewRequest(http.MethodPost, "http://"+testAuthority+"/api/v4/browser-transactions/review", bytes.NewReader(bytes.Repeat([]byte("x"), MaxRequestBytes+1)))
	oversized.Host = testAuthority
	oversized.Header.Set("Authorization", "Bearer "+testToken)
	oversized.Header.Set("Content-Type", "application/json")
	oversizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(oversizedResponse, oversized)
	if oversizedResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized = %d %s", oversizedResponse.Code, oversizedResponse.Body.String())
	}

	origin := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/api/v4/browser-transactions/current", nil)
	origin.Host = testAuthority
	origin.Header.Set("Authorization", "Bearer "+testToken)
	origin.Header.Set("Origin", "https://attacker.example")
	originResponse := httptest.NewRecorder()
	handler.ServeHTTP(originResponse, origin)
	if originResponse.Code != http.StatusForbidden {
		t.Fatalf("foreign origin = %d %s", originResponse.Code, originResponse.Body.String())
	}

	fake.setFailure(&transactionengine.Error{Class: browsertransaction.FailureConflict, Code: transactionengine.ErrorStaleRevision, Operation: transactionengine.OperationReview, Retryable: true})
	typed := doRequest(handler, http.MethodPost, "/api/v4/browser-transactions/review", `{"revision":"sha256:r","transaction_sha256":"sha256:t","human_approved":true}`, "application/json", true)
	if typed.Code != http.StatusConflict || !strings.Contains(typed.Body.String(), `"code":"stale_revision"`) || !strings.Contains(typed.Body.String(), `"retryable":true`) {
		t.Fatalf("typed error = %d %s", typed.Code, typed.Body.String())
	}
}

type fakeBrowserTransactions struct {
	mu        sync.Mutex
	snapshot  transactionengine.Snapshot
	operation transactionengine.Operation
	count     int
	failure   error
}

func newFakeBrowserTransactions() *fakeBrowserTransactions {
	return &fakeBrowserTransactions{snapshot: transactionengine.Snapshot{
		Version: transactionengine.Version, Revision: "sha256:r",
		AllowedOperations: []transactionengine.Operation{transactionengine.OperationObserve}, RuntimeExecutionSupported: false,
	}}
}

func (fake *fakeBrowserTransactions) record(operation transactionengine.Operation) (transactionengine.Snapshot, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.operation = operation
	fake.count++
	return fake.snapshot, fake.failure
}
func (fake *fakeBrowserTransactions) Observe(context.Context) (transactionengine.Snapshot, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.snapshot, nil
}
func (fake *fakeBrowserTransactions) Start(context.Context, transactionengine.StartRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationStart)
}
func (fake *fakeBrowserTransactions) Review(context.Context, transactionengine.ReviewRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationReview)
}
func (fake *fakeBrowserTransactions) Prepare(context.Context, transactionengine.PrepareRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationPrepare)
}
func (fake *fakeBrowserTransactions) Promote(context.Context, transactionengine.PromoteRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationPromote)
}
func (fake *fakeBrowserTransactions) Cancel(context.Context, transactionengine.CancelRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationCancel)
}
func (fake *fakeBrowserTransactions) InspectRecovery(context.Context, transactionengine.InspectRecoveryRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationInspectRecovery)
}
func (fake *fakeBrowserTransactions) Recover(context.Context, transactionengine.RecoverRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationRecover)
}
func (fake *fakeBrowserTransactions) InspectSelected(context.Context, transactionengine.InspectSelectedRequest) (transactionengine.Snapshot, error) {
	return fake.record(transactionengine.OperationInspectSelected)
}
func (fake *fakeBrowserTransactions) lastOperation() transactionengine.Operation {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.operation
}
func (fake *fakeBrowserTransactions) calls() int {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	return fake.count
}
func (fake *fakeBrowserTransactions) setFailure(err error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.failure = err
}

func newBrowserTransactionHandler(t *testing.T, transactions BrowserTransactionEngine) http.Handler {
	t.Helper()
	handler, err := NewHandler(HandlerConfig{
		Engine: &fakeEngine{}, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
		BrowserTransactions: transactions,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

var apiTransactionTime = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func apiRegistrationTransaction() browsertransaction.Transaction {
	return browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "registration-api", Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates:         []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: apiDigest("1"), ReviewSHA256: apiDigest("2")}},
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1, ResultSHA256: apiDigest("3"), ObservedAt: apiTransactionTime.Format(time.RFC3339Nano), ExpiresAt: apiTransactionTime.Add(time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://example.test"}},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "email", Binding: "account_email"}},
	}
}

func apiDigest(character string) string { return "sha256:" + strings.Repeat(character, 64) }
