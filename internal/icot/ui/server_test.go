package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/bundle"
	bevidence "github.com/OpenUdon/browsertools/evidence"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registry"
	"github.com/OpenUdon/browsertools/review"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	testAuthority = "127.0.0.1:43123"
	testToken     = "test-capability-token"
)

type fakeEngine struct {
	mu sync.Mutex

	snapshot               engine.Snapshot
	rounds                 int
	approvals              int
	seen                   []authoring.RoundAnswer
	approval               engine.Approval
	roundErr               error
	writeErr               error
	snapshotErr            error
	result                 *engine.WriteResult
	delay                  time.Duration
	mutateBeforeRoundError bool
}

func (f *fakeEngine) ApplyRound(_ context.Context, answers []authoring.RoundAnswer) (engine.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rounds++
	f.seen = append([]authoring.RoundAnswer(nil), answers...)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.roundErr != nil {
		if f.mutateBeforeRoundError && len(answers) > 0 {
			f.snapshot.Boundary.Outcome = answers[0].Value
		}
		return engine.Snapshot{}, f.roundErr
	}
	f.snapshot.Boundary.Outcome = answers[0].Value
	return f.snapshot, nil
}

func (f *fakeEngine) ApproveAndWrite(_ context.Context, approval engine.Approval) (engine.WriteResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.approvals++
	f.approval = approval
	if !approval.HumanApproved {
		return engine.WriteResult{}, errors.New("explicit human approval is required before writing authoring artifacts")
	}
	if f.writeErr != nil {
		return engine.WriteResult{}, f.writeErr
	}
	if f.result != nil {
		return *f.result, nil
	}
	return engine.WriteResult{
		Written: []string{"project.md", "workflows/intent.hcl"},
		Preview: engine.Preview{ProjectMD: "project", IntentHCL: "intent", ProjectPath: "project.md", IntentPath: "workflows/intent.hcl"},
	}, nil
}

func (f *fakeEngine) Snapshot(context.Context) (engine.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshotErr != nil {
		return engine.Snapshot{}, f.snapshotErr
	}
	return f.snapshot, nil
}

func (f *fakeEngine) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rounds, f.approvals
}

func newFakeHandler(t *testing.T, fake *fakeEngine) http.Handler {
	t.Helper()
	handler, err := NewHandler(HandlerConfig{
		Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, Authority: testAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func doRequest(handler http.Handler, method, path, body, contentType string, auth bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "http://"+testAuthority+path, strings.NewReader(body))
	request.Host = testAuthority
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if auth {
		request.Header.Set("Authorization", "Bearer "+testToken)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder) Response {
	t.Helper()
	var payload Response
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response %q: %v", response.Body.String(), err)
	}
	return payload
}

func currentResponse(t *testing.T, handler http.Handler) Response {
	t.Helper()
	response := doRequest(handler, http.MethodGet, "/api/v1/snapshot", "", "", true)
	if response.Code != http.StatusOK {
		t.Fatalf("snapshot status = %d, body = %s", response.Code, response.Body.String())
	}
	return decodeResponse(t, response)
}

func TestTokenBootstrapCookieAndBearerAuthentication(t *testing.T) {
	fake := &fakeEngine{}
	handler := newFakeHandler(t, fake)
	basePath := instanceBasePath(testToken)

	health := doRequest(handler, http.MethodGet, "/healthz", "", "", false)
	if health.Code != http.StatusOK || health.Body.String() != "ok\n" || strings.Contains(health.Body.String(), "revision") {
		t.Fatalf("health response = %d %q", health.Code, health.Body.String())
	}
	unauthorized := doRequest(handler, http.MethodGet, "/api/v1/snapshot", "", "", false)
	if unauthorized.Code != http.StatusUnauthorized || strings.Contains(unauthorized.Body.String(), testToken) {
		t.Fatalf("unauthorized response = %d %s", unauthorized.Code, unauthorized.Body.String())
	}

	bootstrap := doRequest(handler, http.MethodGet, basePath+"?token="+testToken, "", "", false)
	if bootstrap.Code != http.StatusSeeOther || bootstrap.Header().Get("Location") != basePath {
		t.Fatalf("bootstrap response = %d location %q", bootstrap.Code, bootstrap.Header().Get("Location"))
	}
	cookies := bootstrap.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != SessionCookie || cookies[0].Value != testToken || cookies[0].Path != basePath || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].MaxAge != 0 {
		t.Fatalf("bootstrap cookies = %#v", cookies)
	}
	request := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+basePath, nil)
	request.Host = testAuthority
	request.AddCookie(cookies[0])
	shell := httptest.NewRecorder()
	handler.ServeHTTP(shell, request)
	if shell.Code != http.StatusOK || !strings.Contains(shell.Body.String(), "iCoT workspace") || strings.Contains(shell.Body.String(), testToken) {
		t.Fatalf("shell response = %d %s", shell.Code, shell.Body.String())
	}
	scopedAPI := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+basePath+"api/v1/snapshot", nil)
	scopedAPI.Host = testAuthority
	scopedAPI.AddCookie(cookies[0])
	scopedResponse := httptest.NewRecorder()
	handler.ServeHTTP(scopedResponse, scopedAPI)
	if scopedResponse.Code != http.StatusOK {
		t.Fatalf("scoped cookie API response = %d %s", scopedResponse.Code, scopedResponse.Body.String())
	}
	globalAPI := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/api/v1/snapshot", nil)
	globalAPI.Host = testAuthority
	globalAPI.AddCookie(cookies[0])
	globalResponse := httptest.NewRecorder()
	handler.ServeHTTP(globalResponse, globalAPI)
	if globalResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unscoped cookie API response = %d %s", globalResponse.Code, globalResponse.Body.String())
	}
	bearer := currentResponse(t, handler)
	if bearer.Version != APIVersion || !strings.HasPrefix(bearer.Revision, "sha256:") || bearer.Workspace.ExampleDir != "/tmp/example" {
		t.Fatalf("bearer response = %#v", bearer)
	}

	badBootstrap := doRequest(handler, http.MethodGet, basePath+"?token=wrong", "", "", false)
	if badBootstrap.Code != http.StatusUnauthorized || len(badBootstrap.Result().Cookies()) != 0 {
		t.Fatalf("bad bootstrap = %d %#v", badBootstrap.Code, badBootstrap.Result().Cookies())
	}
}

func TestCapabilityCookieIsIsolatedFromSiblingLoopbackServices(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	firstBase := instanceBasePath(testToken)
	firstURL, err := url.Parse("http://" + testAuthority + firstBase)
	if err != nil {
		t.Fatal(err)
	}
	jar.SetCookies(firstURL, []*http.Cookie{{Name: SessionCookie, Value: testToken, Path: firstBase, HttpOnly: true, SameSite: http.SameSiteStrictMode}})

	siblingRoot, err := url.Parse("http://127.0.0.1:43124/")
	if err != nil {
		t.Fatal(err)
	}
	if cookies := jar.Cookies(siblingRoot); len(cookies) != 0 {
		t.Fatalf("capability cookie leaked to sibling root: %#v", cookies)
	}
	secondBase := instanceBasePath("second-process-token")
	secondURL, err := url.Parse("http://127.0.0.1:43124" + secondBase)
	if err != nil {
		t.Fatal(err)
	}
	if cookies := jar.Cookies(secondURL); len(cookies) != 0 {
		t.Fatalf("first capability cookie leaked to second instance path: %#v", cookies)
	}
	jar.SetCookies(secondURL, []*http.Cookie{{Name: SessionCookie, Value: "second-process-token", Path: secondBase, HttpOnly: true, SameSite: http.SameSiteStrictMode}})
	firstCookies := jar.Cookies(firstURL)
	if len(firstCookies) != 1 || firstCookies[0].Value != testToken {
		t.Fatalf("second process replaced first capability cookie: %#v", firstCookies)
	}
}

func TestExactHostOriginSecurityHeadersAndMethods(t *testing.T) {
	handler := newFakeHandler(t, &fakeEngine{})

	wrongHost := httptest.NewRequest(http.MethodGet, "http://localhost/healthz", nil)
	wrongHost.Host = "localhost:43123"
	wrongHostResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongHostResponse, wrongHost)
	if wrongHostResponse.Code != http.StatusForbidden {
		t.Fatalf("wrong Host status = %d", wrongHostResponse.Code)
	}

	wrongOrigin := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/healthz", nil)
	wrongOrigin.Host = testAuthority
	wrongOrigin.Header.Set("Origin", "http://127.0.0.1:43124")
	wrongOriginResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongOriginResponse, wrongOrigin)
	if wrongOriginResponse.Code != http.StatusForbidden {
		t.Fatalf("wrong Origin status = %d", wrongOriginResponse.Code)
	}
	multipleOrigins := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/healthz", nil)
	multipleOrigins.Host = testAuthority
	multipleOrigins.Header.Add("Origin", "http://"+testAuthority)
	multipleOrigins.Header.Add("Origin", "https://attacker.invalid")
	multipleOriginsResponse := httptest.NewRecorder()
	handler.ServeHTTP(multipleOriginsResponse, multipleOrigins)
	if multipleOriginsResponse.Code != http.StatusForbidden {
		t.Fatalf("multiple Origin status = %d", multipleOriginsResponse.Code)
	}

	exactOrigin := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/healthz", nil)
	exactOrigin.Host = testAuthority
	exactOrigin.Header.Set("Origin", "http://"+testAuthority)
	exactOriginResponse := httptest.NewRecorder()
	handler.ServeHTTP(exactOriginResponse, exactOrigin)
	if exactOriginResponse.Code != http.StatusOK {
		t.Fatalf("exact Origin status = %d", exactOriginResponse.Code)
	}
	for name := range map[string]bool{
		"Cache-Control": true, "Content-Security-Policy": true, "Permissions-Policy": true,
		"Referrer-Policy": true, "X-Content-Type-Options": true, "X-Frame-Options": true,
	} {
		if exactOriginResponse.Header().Get(name) == "" {
			t.Errorf("missing security header %s", name)
		}
	}
	for name := range exactOriginResponse.Header() {
		if strings.HasPrefix(strings.ToLower(name), "access-control-") {
			t.Errorf("unexpected CORS header %s", name)
		}
	}

	unsupported := doRequest(handler, http.MethodGet, "/api/v1/round", "", "", true)
	if unsupported.Code != http.StatusMethodNotAllowed || unsupported.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("unsupported method = %d allow %q", unsupported.Code, unsupported.Header().Get("Allow"))
	}
	preflight := httptest.NewRequest(http.MethodOptions, "http://"+testAuthority+"/api/v1/round", nil)
	preflight.Host = testAuthority
	preflight.Header.Set("Origin", "https://attacker.invalid")
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusForbidden || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("preflight response = %d %#v", preflightResponse.Code, preflightResponse.Header())
	}
}

func TestStrictJSONAndRequestLimit(t *testing.T) {
	handler := newFakeHandler(t, &fakeEngine{})
	tests := []struct {
		name        string
		body        string
		contentType string
	}{
		{name: "missing content type", body: `{}`},
		{name: "wrong content type", body: `{}`, contentType: "text/plain"},
		{name: "missing answers", body: `{"revision":"sha256:no"}`, contentType: "application/json"},
		{name: "unknown field", body: `{"revision":"sha256:no","answers":[],"slots":[]}`, contentType: "application/json"},
		{name: "trailing document", body: `{"revision":"sha256:no","answers":[]} {}`, contentType: "application/json"},
		{name: "malformed", body: `{"revision":`, contentType: "application/json"},
		{name: "oversized", body: `{"revision":"sha256:no","answers":[],"padding":"` + strings.Repeat("x", MaxRequestBytes) + `"}`, contentType: "application/json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := doRequest(handler, http.MethodPost, "/api/v1/round", test.body, test.contentType, true)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"malformed_request"`) || strings.Contains(response.Body.String(), testToken) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
	if rounds, approvals := handler.(*Server).engine.(*fakeEngine).counts(); rounds != 0 || approvals != 0 {
		t.Fatalf("malformed requests mutated engine: rounds %d approvals %d", rounds, approvals)
	}
}

func TestRevisionRoundApprovalAndFrozenInspection(t *testing.T) {
	fake := &fakeEngine{snapshot: engine.Snapshot{Ready: false}}
	handler := newFakeHandler(t, fake)
	initial := currentResponse(t, handler)

	malformed := doRequest(handler, http.MethodPost, "/api/v1/round", `{"revision":"`+initial.Revision+`","answers":[{"question_id":"","value":"x"}]}`, "application/json", true)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed status = %d", malformed.Code)
	}
	stale := doRequest(handler, http.MethodPost, "/api/v1/round", `{"revision":"sha256:stale","answers":[{"question_id":"goal","value":"new"}]}`, "application/json", true)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale status = %d body %s", stale.Code, stale.Body.String())
	}
	if rounds, _ := fake.counts(); rounds != 0 {
		t.Fatalf("stale request called engine %d times", rounds)
	}

	round := doRequest(handler, http.MethodPost, "/api/v1/round", `{"revision":"`+initial.Revision+`","answers":[{"question_id":"goal","value":"new outcome"}]}`, "application/json; charset=utf-8", true)
	if round.Code != http.StatusOK {
		t.Fatalf("round status = %d body %s", round.Code, round.Body.String())
	}
	afterRound := decodeResponse(t, round)
	if afterRound.Revision == initial.Revision || afterRound.Snapshot.Boundary.Outcome != "new outcome" {
		t.Fatalf("round response = %#v", afterRound)
	}
	fake.mu.Lock()
	if len(fake.seen) != 1 || fake.seen[0].Source != humanInputSource || len(fake.seen[0].Slots) != 0 {
		t.Fatalf("engine answers = %#v", fake.seen)
	}
	fake.mu.Unlock()
	declined := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":false}`, afterRound.Revision), "application/json", true)
	if declined.Code != http.StatusUnprocessableEntity || !strings.Contains(declined.Body.String(), "explicit human approval") {
		t.Fatalf("declined approval = %d %s", declined.Code, declined.Body.String())
	}
	if current := currentResponse(t, handler); current.Revision != afterRound.Revision || current.Completed {
		t.Fatalf("declined approval changed state: %#v", current)
	}

	approve := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true,"allow_overwrite":true,"approve_incomplete":true}`, afterRound.Revision), "application/json", true)
	if approve.Code != http.StatusOK {
		t.Fatalf("approve status = %d body %s", approve.Code, approve.Body.String())
	}
	approved := decodeResponse(t, approve)
	if !approved.Completed || approved.WriteResult == nil || approved.Revision == afterRound.Revision {
		t.Fatalf("approved response = %#v", approved)
	}
	fake.mu.Lock()
	if !fake.approval.HumanApproved || !fake.approval.AllowOverwrite || !fake.approval.ApproveIncomplete {
		t.Fatalf("approval flags = %#v", fake.approval)
	}
	fake.mu.Unlock()

	frozen := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, approved.Revision), "application/json", true)
	if frozen.Code != http.StatusConflict || !strings.Contains(frozen.Body.String(), "session_frozen") {
		t.Fatalf("frozen response = %d %s", frozen.Code, frozen.Body.String())
	}
	inspection := currentResponse(t, handler)
	if !inspection.Completed || inspection.Revision != approved.Revision || inspection.WriteResult == nil {
		t.Fatalf("frozen inspection = %#v", inspection)
	}
	if rounds, approvals := fake.counts(); rounds != 1 || approvals != 2 {
		t.Fatalf("engine counts = rounds %d approvals %d", rounds, approvals)
	}
}

func TestIncompleteWriteAlsoFreezesSession(t *testing.T) {
	result := engine.WriteResult{
		Written:    []string{"project.md", "workflows/intent.draft.hcl", ".icot/session.yaml", ".icot/readiness.json"},
		Incomplete: true,
		Preview:    engine.Preview{Incomplete: true, ProjectPath: "project.md", IntentPath: "workflows/intent.draft.hcl"},
	}
	fake := &fakeEngine{result: &result}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true,"approve_incomplete":true}`, before.Revision), "application/json", true)
	if response.Code != http.StatusOK {
		t.Fatalf("incomplete approval = %d %s", response.Code, response.Body.String())
	}
	after := decodeResponse(t, response)
	if !after.Completed || after.WriteResult == nil || !after.WriteResult.Incomplete {
		t.Fatalf("incomplete response = %#v", after)
	}
	frozen := doRequest(handler, http.MethodPost, "/api/v1/round", fmt.Sprintf(`{"revision":%q,"answers":[]}`, after.Revision), "application/json", true)
	if frozen.Code != http.StatusConflict || !strings.Contains(frozen.Body.String(), "session_frozen") {
		t.Fatalf("frozen response = %d %s", frozen.Code, frozen.Body.String())
	}
}

func TestEngineRejectionsPreserveRevisionAndState(t *testing.T) {
	for _, test := range []struct {
		name    string
		failure string
	}{
		{name: "replaced browser verification", failure: "browser verification report changed after review"},
		{name: "unavailable registry", failure: "selected browser registry profile could not be freshly revalidated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeEngine{snapshot: engine.Snapshot{Ready: true}, writeErr: errors.New(test.failure)}
			handler := newFakeHandler(t, fake)
			before := currentResponse(t, handler)
			response := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
			if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), test.failure) {
				t.Fatalf("rejection response = %d %s", response.Code, response.Body.String())
			}
			after := currentResponse(t, handler)
			if after.Revision != before.Revision || after.Completed || after.WriteResult != nil {
				t.Fatalf("state changed across rejection: before %#v after %#v", before, after)
			}
		})
	}
}

func TestCanceledRoundResynchronizesRevisionAfterEngineMutation(t *testing.T) {
	fake := &fakeEngine{
		snapshot:               engine.Snapshot{Boundary: elicitor.WorkflowBoundary{Outcome: "before"}},
		roundErr:               context.Canceled,
		mutateBeforeRoundError: true,
	}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"goal","value":"autosaved after cancellation"}]}`, before.Revision)
	request := httptest.NewRequest(http.MethodPost, "http://"+testAuthority+"/api/v1/round", strings.NewReader(body))
	request.Host = testAuthority
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	canceled, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(canceled)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("canceled round response = %d %s", response.Code, response.Body.String())
	}

	after := currentResponse(t, handler)
	if after.Revision == before.Revision || after.Snapshot.Boundary.Outcome != "autosaved after cancellation" {
		t.Fatalf("state was not resynchronized: before %#v after %#v", before, after)
	}
	staleApproval := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if staleApproval.Code != http.StatusConflict || !strings.Contains(staleApproval.Body.String(), "stale_revision") {
		t.Fatalf("stale approval response = %d %s", staleApproval.Code, staleApproval.Body.String())
	}
	if _, approvals := fake.counts(); approvals != 0 {
		t.Fatalf("stale approval called engine %d times", approvals)
	}
}

func TestFailedPostMutationSynchronizationDoesNotAdvertiseStaleRevision(t *testing.T) {
	fake := &fakeEngine{
		snapshot:               engine.Snapshot{Boundary: elicitor.WorkflowBoundary{Outcome: "before"}},
		roundErr:               context.Canceled,
		snapshotErr:            errors.New("snapshot unavailable"),
		mutateBeforeRoundError: true,
	}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"goal","value":"advanced"}]}`, before.Revision)
	response := doRequest(handler, http.MethodPost, "/api/v1/round", body, "application/json", true)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("round response = %d %s", response.Code, response.Body.String())
	}
	inspection := doRequest(handler, http.MethodGet, "/api/v1/snapshot", "", "", true)
	if inspection.Code != http.StatusInternalServerError || strings.Contains(inspection.Body.String(), before.Revision) {
		t.Fatalf("stale inspection response = %d %s", inspection.Code, inspection.Body.String())
	}
	approval := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if approval.Code != http.StatusInternalServerError {
		t.Fatalf("approval against unavailable state = %d %s", approval.Code, approval.Body.String())
	}
	if _, approvals := fake.counts(); approvals != 0 {
		t.Fatalf("approval called engine %d times", approvals)
	}
}

func TestOperationalEngineFailuresReturnServerErrors(t *testing.T) {
	operationFailure := &os.PathError{Op: "rename", Path: "/private/workflows/intent.hcl", Err: errors.New("disk full")}
	for _, test := range []struct {
		name     string
		fake     *fakeEngine
		endpoint string
		body     func(string) string
	}{
		{
			name: "round autosave", fake: &fakeEngine{roundErr: operationFailure}, endpoint: "/api/v1/round",
			body: func(revision string) string {
				return fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"goal","value":"new"}]}`, revision)
			},
		},
		{
			name: "approval write", fake: &fakeEngine{snapshot: engine.Snapshot{Ready: true}, writeErr: operationFailure}, endpoint: "/api/v1/approve",
			body: func(revision string) string {
				return fmt.Sprintf(`{"revision":%q,"human_approved":true}`, revision)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := newFakeHandler(t, test.fake)
			before := currentResponse(t, handler)
			response := doRequest(handler, http.MethodPost, test.endpoint, test.body(before.Revision), "application/json", true)
			if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
				t.Fatalf("operational failure response = %d %s", response.Code, response.Body.String())
			}
			if strings.Contains(response.Body.String(), operationFailure.Path) || strings.Contains(response.Body.String(), "engine_rejected") {
				t.Fatalf("operational details escaped in response: %s", response.Body.String())
			}
			after := currentResponse(t, handler)
			if after.Revision != before.Revision || after.Completed || after.WriteResult != nil {
				t.Fatalf("operational failure corrupted state: before %#v after %#v", before, after)
			}
		})
	}
}

func TestRealEngineCommitFailureReturnsServerErrorWithoutWrite(t *testing.T) {
	example := filepath.Join(t.TempDir(), "target")
	fixture := filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render")
	authoringEngine, snapshot, err := engine.Open(context.Background(), engine.Config{
		ExampleDir: example, FromExample: fixture, NetworkPolicy: "never",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows"), []byte("blocks directory creation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(HandlerConfig{
		Engine: authoringEngine, Snapshot: snapshot, ExampleDir: example, Token: testToken, Authority: testAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("commit failure response = %d %s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("project.md exists after failed transaction: %v", err)
	}
	after := currentResponse(t, handler)
	if after.Revision != before.Revision || after.Completed || after.WriteResult != nil {
		t.Fatalf("commit failure changed HTTP state: before %#v after %#v", before, after)
	}
}

func TestRealEngineBrowserRefreshFailuresPreserveHTTPRevision(t *testing.T) {
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	profileValue, err := profile.ParseJSON(profileData)
	if err != nil {
		t.Fatal(err)
	}
	profileDigest, err := browserverify.ProfileDigest(profileValue)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("replaced verification", func(t *testing.T) {
		report := map[string]any{
			"version": browserverify.LiveCheckVersion, "profileDigest": profileDigest,
			"checkedAt": now.Add(-time.Minute).Format(time.RFC3339Nano), "origin": "https://example.test",
			"actions": []string{"read_status"}, "ok": true,
			"checks": []map[string]any{{
				"kind": "output", "path": "actions.read_status.outputs.status", "ok": true, "matches": 1,
				"expectedType": "string", "observedType": "string", "message": "declared output source and JSON type matched",
			}, {
				"kind": "locator", "path": "actions.read_status.sequence[1].wait_for", "ok": true, "matches": 1,
				"message": "declared accessibility locator resolved exactly once",
			}},
		}
		reportPath := filepath.Join(t.TempDir(), "live-check.json")
		writeTestJSON(t, reportPath, report)
		example := filepath.Join(t.TempDir(), "target")
		seed := uiBrowserSeedSession()
		authoringEngine, snapshot, err := engine.Open(context.Background(), engine.Config{
			ExampleDir: example, Seed: &seed,
			BrowserSources:       []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}},
			BrowserVerifications: []string{reportPath}, NetworkPolicy: "never", Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		handler, err := NewHandler(HandlerConfig{Engine: authoringEngine, Snapshot: snapshot, ExampleDir: example, Token: testToken, Authority: testAuthority})
		if err != nil {
			t.Fatal(err)
		}
		before := currentResponse(t, handler)
		report["checkedAt"] = now.Format(time.RFC3339Nano)
		writeTestJSON(t, reportPath, report)
		response := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "changed after review") {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
		assertHTTPStateUnchanged(t, handler, before, example)
	})

	t.Run("unavailable registry", func(t *testing.T) {
		registryRoot := filepath.Join(t.TempDir(), "registry")
		if _, err := registry.PublishLocal(context.Background(), registry.PublishOptions{Root: registryRoot, Bundle: uiBrowserRegistryBundle(t, now), At: now}); err != nil {
			t.Fatal(err)
		}
		example := filepath.Join(t.TempDir(), "target")
		seed := uiBrowserSeedSession()
		seed.Boundary.Outcome = "status"
		authoringEngine, snapshot, err := engine.Open(context.Background(), engine.Config{
			ExampleDir: example, Seed: &seed, BrowserRegistries: []string{registryRoot}, NetworkPolicy: "never", Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.SelectedSources) != 1 || snapshot.SelectedSources[0].RegistryCoordinate != "status@1.0.0" {
			t.Fatalf("selected sources = %#v", snapshot.SelectedSources)
		}
		handler, err := NewHandler(HandlerConfig{Engine: authoringEngine, Snapshot: snapshot, ExampleDir: example, Token: testToken, Authority: testAuthority})
		if err != nil {
			t.Fatal(err)
		}
		before := currentResponse(t, handler)
		if err := os.Rename(registryRoot, registryRoot+".unavailable"); err != nil {
			t.Fatal(err)
		}
		response := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
		if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "freshly revalidated") {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
		assertHTTPStateUnchanged(t, handler, before, example)
	})
}

func TestConcurrentSameRevisionMutationHasOneWinner(t *testing.T) {
	fake := &fakeEngine{delay: 20 * time.Millisecond}
	handler := newFakeHandler(t, fake)
	revision := currentResponse(t, handler).Revision
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"goal","value":"winner"}]}`, revision)
	start := make(chan struct{})
	statuses := make(chan int, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			statuses <- doRequest(handler, http.MethodPost, "/api/v1/round", body, "application/json", true).Code
		}()
	}
	close(start)
	workers.Wait()
	close(statuses)
	counts := map[int]int{}
	for status := range statuses {
		counts[status]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("statuses = %#v", counts)
	}
	if rounds, _ := fake.counts(); rounds != 1 {
		t.Fatalf("engine round count = %d", rounds)
	}
}

func TestHTTPAndDirectEngineArtifactParity(t *testing.T) {
	fixture := filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render")
	httpDir := filepath.Join(t.TempDir(), "http")
	directDir := filepath.Join(t.TempDir(), "direct")

	httpEngine, snapshot, err := engine.Open(context.Background(), engine.Config{ExampleDir: httpDir, FromExample: fixture, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(HandlerConfig{Engine: httpEngine, Snapshot: snapshot, ExampleDir: httpDir, Token: testToken, Authority: testAuthority})
	if err != nil {
		t.Fatal(err)
	}
	revision := currentResponse(t, handler).Revision
	response := doRequest(handler, http.MethodPost, "/api/v1/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, revision), "application/json", true)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP approval = %d %s", response.Code, response.Body.String())
	}

	directEngine, _, err := engine.Open(context.Background(), engine.Config{ExampleDir: directDir, FromExample: fixture, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := directEngine.ApproveAndWrite(context.Background(), engine.Approval{HumanApproved: true}); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"project.md", "workflows/intent.hcl"} {
		httpData, err := os.ReadFile(filepath.Join(httpDir, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		directData, err := os.ReadFile(filepath.Join(directDir, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(httpData, directData) {
			t.Fatalf("%s differs\nHTTP:\n%s\ndirect:\n%s", relative, httpData, directData)
		}
	}
}

func TestEmbeddedAssetsAreSeparateAndAvailable(t *testing.T) {
	for _, name := range []string{"assets/index.html", "assets/app.js", "assets/style.css"} {
		data, err := assetFiles.ReadFile(name)
		if err != nil || len(data) == 0 {
			t.Fatalf("embedded %s = %d bytes, %v", name, len(data), err)
		}
	}
	handler := newFakeHandler(t, &fakeEngine{})
	for _, path := range []string{"/assets/app.js", "/assets/style.css"} {
		response := doRequest(handler, http.MethodGet, path, "", "", true)
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") == "" {
			t.Fatalf("asset %s = %d %q", path, response.Code, response.Header().Get("Content-Type"))
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func uiBrowserSeedSession() elicitor.Session {
	return elicitor.Session{
		Boundary: elicitor.WorkflowBoundary{Outcome: "Read browser status", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"status output is returned"}, Confirmed: true},
		Intent: rollout.Intent{
			Source: "browser-profiles/status.json", Workflow: &rollout.WorkflowMeta{Name: "browser_status", Description: "Read browser status"},
			Inputs:  []*rollout.Input{{Name: "item", Type: "string", Required: true}},
			Steps:   []*rollout.Step{{Name: "read", Type: "browser", Source: "browser-profiles/status.json", Operation: "read_status", With: map[string]string{"item": "inputs.item"}}},
			Outputs: []*rollout.Output{{Name: "status", From: "read.received_body.status"}},
		},
		BrowserRoute: "browser", BrowserSession: "none", Fallback: "stop cleanly", FallbackSet: true,
		SideEffectScope: "read-only", Safety: "read-only", SafetySet: true, CredentialsSet: true,
	}
}

func uiBrowserRegistryBundle(t *testing.T, now time.Time) *bundle.Bundle {
	t.Helper()
	profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	value, err := profile.ParseJSON(profileData)
	if err != nil {
		t.Fatal(err)
	}
	record, err := (&bevidence.RawRecord{Record: bevidence.Record{
		Origin: "https://example.test", ObservationKind: bevidence.ObservationA11ySnapshot,
		ObservedAt: now.Add(-time.Hour).Format(time.RFC3339), ActionHint: "read_status",
		CandidateLocators: []bevidence.CandidateLocator{{Role: "status", Name: "Ready"}},
		RedactionStatus:   bevidence.RedactionNotRequired, Provenance: bevidence.Provenance{Tool: "synthetic-ui-test", Version: "1"},
	}}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	reviewed, err := review.Build(value, []bevidence.Record{record}, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	result, err := bundle.Build(bundle.BuildOptions{
		ID: "status", Release: "1.0.0", Source: "reviewed_synthetic_fixture", License: "CC0-1.0",
		Authors: []string{"OpenUdon"}, Profile: value, Review: reviewed, Evidence: []bevidence.Record{record}, PublishedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertHTTPStateUnchanged(t *testing.T, handler http.Handler, before Response, example string) {
	t.Helper()
	after := currentResponse(t, handler)
	if after.Revision != before.Revision || after.Completed || after.WriteResult != nil {
		t.Fatalf("state changed across rejection: before %#v after %#v", before, after)
	}
	for _, relative := range []string{"project.md", "workflows/intent.hcl", "workflows/intent.draft.hcl"} {
		if _, err := os.Stat(filepath.Join(example, filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Fatalf("deliverable %s exists after rejected write: %v", relative, err)
		}
	}
}
