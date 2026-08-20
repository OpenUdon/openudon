package ui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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
	testAuthority  = "127.0.0.1:43123"
	testToken      = "test-capability-token"
	testAccessCode = "0123456789AB"
)

type fakeEngine struct {
	mu sync.Mutex

	snapshot               engine.Snapshot
	rounds                 int
	reopens                int
	approvals              int
	seen                   []authoring.RoundAnswer
	reopenedQuestion       string
	approval               engine.Approval
	roundErr               error
	reopenErr              error
	writeErr               error
	snapshotErr            error
	workspace              engine.WorkspaceStatus
	result                 *engine.WriteResult
	delay                  time.Duration
	mutateBeforeRoundError bool
}

func (f *fakeEngine) ReopenDecision(_ context.Context, questionID string) (engine.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reopens++
	f.reopenedQuestion = questionID
	if f.reopenErr != nil {
		return engine.Snapshot{}, f.reopenErr
	}
	f.snapshot.Frontier = []elicitor.QuestionPlan{{ID: questionID, Prompt: "Revise this decision", Required: true}}
	f.snapshot.RevisableDecisions = nil
	f.snapshot.ApprovalRequired = false
	f.snapshot.Ready = false
	return f.snapshot, nil
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

func (f *fakeEngine) ApproveAndWrite(_ context.Context, approval engine.Approval) (engine.ApprovalResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.approvals++
	f.approval = approval
	if !approval.HumanApproved {
		return engine.ApprovalResult{}, engineRejected(errors.New("explicit human approval is required before writing authoring artifacts"))
	}
	if f.writeErr != nil {
		return engine.ApprovalResult{}, f.writeErr
	}
	result := engine.WriteResult{
		Written: []string{"project.md", "workflows/intent.hcl"},
		Preview: engine.Preview{ProjectMD: "project", IntentHCL: "intent", ProjectPath: "project.md", IntentPath: "workflows/intent.hcl"},
	}
	if f.result != nil {
		result = *f.result
	}
	return engine.ApprovalResult{Snapshot: f.snapshot, WriteResult: result}, nil
}

func (f *fakeEngine) Snapshot(context.Context) (engine.Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshotErr != nil {
		return engine.Snapshot{}, f.snapshotErr
	}
	return f.snapshot, nil
}

func (f *fakeEngine) WorkspaceStatus(context.Context) (engine.WorkspaceStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.snapshotErr != nil {
		return engine.WorkspaceStatus{}, f.snapshotErr
	}
	return f.workspace, nil
}

func engineRejected(err error) error {
	return &engine.Failure{Class: engine.FailureRejected, Code: "engine_rejected", Cause: err}
}

func (f *fakeEngine) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rounds, f.approvals
}

func (f *fakeEngine) reopenRecord() (int, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reopens, f.reopenedQuestion
}

func (f *fakeEngine) setWorkspace(status engine.WorkspaceStatus) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.workspace = status
}

func newFakeHandler(t *testing.T, fake *fakeEngine) http.Handler {
	t.Helper()
	handler, err := NewHandler(HandlerConfig{
		Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
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
	response := doRequest(handler, http.MethodGet, "/api/v2/snapshot", "", "", true)
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
	unauthorized := doRequest(handler, http.MethodGet, "/api/v2/snapshot", "", "", false)
	if unauthorized.Code != http.StatusUnauthorized || strings.Contains(unauthorized.Body.String(), testToken) {
		t.Fatalf("unauthorized response = %d %s", unauthorized.Code, unauthorized.Body.String())
	}

	bootstrap := doRequest(handler, http.MethodPost, "/", "code="+url.QueryEscape(testAccessCode), "application/x-www-form-urlencoded", false)
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
	lostSession := doRequest(handler, http.MethodGet, basePath, "", "", false)
	if lostSession.Code != http.StatusSeeOther || lostSession.Header().Get("Location") != "/" {
		t.Fatalf("lost browser session = %d location %q", lostSession.Code, lostSession.Header().Get("Location"))
	}
	scopedAPI := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+basePath+"api/v2/snapshot", nil)
	scopedAPI.Host = testAuthority
	scopedAPI.AddCookie(cookies[0])
	scopedResponse := httptest.NewRecorder()
	handler.ServeHTTP(scopedResponse, scopedAPI)
	if scopedResponse.Code != http.StatusOK {
		t.Fatalf("scoped cookie API response = %d %s", scopedResponse.Code, scopedResponse.Body.String())
	}
	globalAPI := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/api/v2/snapshot", nil)
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

	badBootstrap := doRequest(handler, http.MethodPost, "/", "code=WRONGCODE000", "application/x-www-form-urlencoded", false)
	if badBootstrap.Code != http.StatusUnauthorized || len(badBootstrap.Result().Cookies()) != 0 {
		t.Fatalf("bad bootstrap = %d %#v", badBootstrap.Code, badBootstrap.Result().Cookies())
	}
	globalBootstrap := doRequest(handler, http.MethodGet, "/?token="+testToken, "", "", false)
	if globalBootstrap.Code != http.StatusNotFound || len(globalBootstrap.Result().Cookies()) != 0 {
		t.Fatalf("global bootstrap = %d %#v", globalBootstrap.Code, globalBootstrap.Result().Cookies())
	}
	v1 := doRequest(handler, http.MethodGet, "/api/v1/snapshot", "", "", true)
	if v1.Code != http.StatusNotFound {
		t.Fatalf("retired v1 route = %d %s", v1.Code, v1.Body.String())
	}
}

func TestAccessCodeExpiresIsSingleUseAndRateLimited(t *testing.T) {
	start := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	now := start
	newHandler := func() http.Handler {
		handler, err := NewHandler(HandlerConfig{
			Engine: &fakeEngine{}, ExampleDir: "/tmp/example", Token: testToken,
			AccessCode: testAccessCode, Authority: testAuthority, Now: func() time.Time { return now },
		})
		if err != nil {
			t.Fatal(err)
		}
		return handler
	}

	handler := newHandler()
	first := doRequest(handler, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first exchange = %d", first.Code)
	}
	reuse := doRequest(handler, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("reused exchange = %d", reuse.Code)
	}

	now = start
	expiring := newHandler()
	now = start.Add(5 * time.Minute)
	expired := doRequest(expiring, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if expired.Code != http.StatusUnauthorized {
		t.Fatalf("expired exchange = %d", expired.Code)
	}

	now = start
	limited := newHandler()
	for i := 0; i < 5; i++ {
		response := doRequest(limited, http.MethodPost, "/", "code=WRONGCODE000", "application/x-www-form-urlencoded", false)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("failed attempt %d = %d", i+1, response.Code)
		}
	}
	response := doRequest(limited, http.MethodPost, "/", "code=WRONGCODE000", "application/x-www-form-urlencoded", false)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("throttled attempt = %d", response.Code)
	}
	now = start.Add(time.Minute + time.Nanosecond)
	response = doRequest(limited, http.MethodPost, "/", "code=WRONGCODE000", "application/x-www-form-urlencoded", false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("post-window attempt = %d", response.Code)
	}
}

func TestAccessCodeRecoveryRotatesOnlyThroughTerminal(t *testing.T) {
	var terminal bytes.Buffer
	handler, err := NewHandler(HandlerConfig{
		Engine: &fakeEngine{}, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
		AccessCodeOut: &terminal, GenerateAccessCode: func() (string, error) { return "ABCDEFGHJKM2", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	active := doRequest(handler, http.MethodPost, "/", "recover=1", "application/x-www-form-urlencoded", false)
	if active.Code != http.StatusConflict || terminal.Len() != 0 || !strings.Contains(active.Body.String(), "access_code_active") {
		t.Fatalf("active-code recovery = %d %q terminal %q", active.Code, active.Body.String(), terminal.String())
	}
	used := doRequest(handler, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if used.Code != http.StatusSeeOther {
		t.Fatalf("initial exchange = %d %s", used.Code, used.Body.String())
	}
	recovered := doRequest(handler, http.MethodPost, "/", "recover=1", "application/x-www-form-urlencoded", false)
	if recovered.Code != http.StatusOK || !strings.Contains(recovered.Body.String(), "printed in the terminal") || strings.Contains(recovered.Body.String(), "ABCDEFGHJKM2") {
		t.Fatalf("recovery response = %d %s", recovered.Code, recovered.Body.String())
	}
	if terminal.String() != "icot ui replacement access code: ABCDEFGHJKM2\n" {
		t.Fatalf("terminal recovery output = %q", terminal.String())
	}
	oldCode := doRequest(handler, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if oldCode.Code != http.StatusUnauthorized {
		t.Fatalf("rotated code remained valid = %d", oldCode.Code)
	}
	freshCode := doRequest(handler, http.MethodPost, "/", "code=ABCDEFGHJKM2", "application/x-www-form-urlencoded", false)
	if freshCode.Code != http.StatusSeeOther {
		t.Fatalf("replacement exchange = %d %s", freshCode.Code, freshCode.Body.String())
	}
}

func TestAccessCodeRecoveryIsRateLimited(t *testing.T) {
	handler, err := NewHandler(HandlerConfig{
		Engine: &fakeEngine{}, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
		AccessCodeOut: io.Discard, GenerateAccessCode: func() (string, error) { return "ABCDEFGHJKM2", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	server := handler.(*Server)
	for attempt := 0; attempt < 5; attempt++ {
		server.mu.Lock()
		server.accessCodeUsed = true
		server.mu.Unlock()
		response := doRequest(handler, http.MethodPost, "/", "recover=1", "application/x-www-form-urlencoded", false)
		if response.Code != http.StatusOK {
			t.Fatalf("recovery %d = %d %s", attempt+1, response.Code, response.Body.String())
		}
	}
	server.mu.Lock()
	server.accessCodeUsed = true
	server.mu.Unlock()
	response := doRequest(handler, http.MethodPost, "/", "recover=1", "application/x-www-form-urlencoded", false)
	if response.Code != http.StatusTooManyRequests || !strings.Contains(response.Body.String(), "rate_limited") {
		t.Fatalf("recovery throttle = %d %s", response.Code, response.Body.String())
	}
}

func TestAccessCodeRecoveryWithoutTerminalFailsWithoutRotation(t *testing.T) {
	handler, err := NewHandler(HandlerConfig{
		Engine: &fakeEngine{}, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
		GenerateAccessCode: func() (string, error) { return "ABCDEFGHJKM2", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	used := doRequest(handler, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if used.Code != http.StatusSeeOther {
		t.Fatalf("initial exchange = %d %s", used.Code, used.Body.String())
	}
	recovery := doRequest(handler, http.MethodPost, "/", "recover=1", "application/x-www-form-urlencoded", false)
	if recovery.Code != http.StatusConflict || !strings.Contains(recovery.Body.String(), "access_code_recovery_unavailable") {
		t.Fatalf("missing-terminal recovery = %d %s", recovery.Code, recovery.Body.String())
	}
	reuse := doRequest(handler, http.MethodPost, "/", "code="+testAccessCode, "application/x-www-form-urlencoded", false)
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("used code was unexpectedly rotated or reusable = %d", reuse.Code)
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
		"Cross-Origin-Opener-Policy": true, "Cross-Origin-Resource-Policy": true,
		"Referrer-Policy": true, "X-Content-Type-Options": true, "X-Frame-Options": true,
	} {
		if exactOriginResponse.Header().Get(name) == "" {
			t.Errorf("missing security header %s", name)
		}
	}
	csp := exactOriginResponse.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "form-action 'self'") || strings.Contains(csp, "form-action 'none'") {
		t.Fatalf("bootstrap-blocking content security policy = %q", csp)
	}
	if policy := exactOriginResponse.Header().Get("Referrer-Policy"); policy != "same-origin" {
		t.Fatalf("origin-breaking referrer policy = %q", policy)
	}
	for name := range exactOriginResponse.Header() {
		if strings.HasPrefix(strings.ToLower(name), "access-control-") {
			t.Errorf("unexpected CORS header %s", name)
		}
	}

	unsupported := doRequest(handler, http.MethodGet, "/api/v2/round", "", "", true)
	if unsupported.Code != http.StatusMethodNotAllowed || unsupported.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("unsupported method = %d allow %q", unsupported.Code, unsupported.Header().Get("Allow"))
	}
	unsupportedReopen := doRequest(handler, http.MethodGet, "/api/v2/reopen", "", "", true)
	if unsupportedReopen.Code != http.StatusMethodNotAllowed || unsupportedReopen.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("unsupported reopen method = %d allow %q", unsupportedReopen.Code, unsupportedReopen.Header().Get("Allow"))
	}
	preflight := httptest.NewRequest(http.MethodOptions, "http://"+testAuthority+"/api/v2/round", nil)
	preflight.Host = testAuthority
	preflight.Header.Set("Origin", "https://attacker.invalid")
	preflightResponse := httptest.NewRecorder()
	handler.ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusForbidden || preflightResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("preflight response = %d %#v", preflightResponse.Code, preflightResponse.Header())
	}
}

func TestConditionalSnapshotAndWorkspaceRevision(t *testing.T) {
	fake := &fakeEngine{snapshot: engine.Snapshot{Boundary: elicitor.WorkflowBoundary{Outcome: "cached"}}}
	handler := newFakeHandler(t, fake)
	initial := currentResponse(t, handler)
	request := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/api/v2/snapshot", nil)
	request.Host = testAuthority
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("If-None-Match", strconv.Quote(initial.Revision))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotModified || response.Body.Len() != 0 || response.Header().Get("ETag") != strconv.Quote(initial.Revision) {
		t.Fatalf("conditional response = %d etag=%q body=%q", response.Code, response.Header().Get("ETag"), response.Body.String())
	}
	wildcard := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/api/v2/snapshot", nil)
	wildcard.Host = testAuthority
	wildcard.Header.Set("Authorization", "Bearer "+testToken)
	wildcard.Header.Set("If-None-Match", "*")
	wildcardResponse := httptest.NewRecorder()
	handler.ServeHTTP(wildcardResponse, wildcard)
	if wildcardResponse.Code != http.StatusNotModified || wildcardResponse.Body.Len() != 0 || wildcardResponse.Header().Get("ETag") != strconv.Quote(initial.Revision) {
		t.Fatalf("wildcard conditional response = %d etag=%q body=%q", wildcardResponse.Code, wildcardResponse.Header().Get("ETag"), wildcardResponse.Body.String())
	}

	fake.setWorkspace(engine.WorkspaceStatus{ExternallyModified: true})
	changed := doRequest(handler, http.MethodGet, "/api/v2/snapshot", "", "", true)
	if changed.Code != http.StatusOK {
		t.Fatalf("changed snapshot = %d %s", changed.Code, changed.Body.String())
	}
	payload := decodeResponse(t, changed)
	if !payload.Workspace.ExternallyModified || payload.Revision == initial.Revision {
		t.Fatalf("workspace change did not revise response: %#v", payload)
	}
	mutation := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true,"allow_overwrite":true}`, payload.Revision), "application/json", true)
	if mutation.Code != http.StatusConflict || !strings.Contains(mutation.Body.String(), `"code":"workspace_changed"`) || !strings.Contains(mutation.Body.String(), payload.Revision) {
		t.Fatalf("workspace mutation = %d %s", mutation.Code, mutation.Body.String())
	}
}

func TestWriteConflictsAreExposedAndRevisionBound(t *testing.T) {
	conflict := engine.WriteConflict{Code: "overwrite_required", Action: "write", Path: "/tmp/example/project.md"}
	withConflict := newFakeHandler(t, &fakeEngine{snapshot: engine.Snapshot{WriteConflicts: []engine.WriteConflict{conflict}}})
	conflicted := currentResponse(t, withConflict)
	if !reflect.DeepEqual(conflicted.Snapshot.WriteConflicts, []engine.WriteConflict{conflict}) {
		t.Fatalf("API write conflicts = %#v", conflicted.Snapshot.WriteConflicts)
	}

	withoutConflict := newFakeHandler(t, &fakeEngine{snapshot: engine.Snapshot{WriteConflicts: []engine.WriteConflict{}}})
	clear := currentResponse(t, withoutConflict)
	if conflicted.Revision == clear.Revision {
		t.Fatalf("write conflict did not affect revision: %s", conflicted.Revision)
	}
	conditionalRequest := httptest.NewRequest(http.MethodGet, "http://"+testAuthority+"/api/v2/snapshot", nil)
	conditionalRequest.Host = testAuthority
	conditionalRequest.Header.Set("Authorization", "Bearer "+testToken)
	conditionalRequest.Header.Set("If-None-Match", strconv.Quote(conflicted.Revision))
	conditional := httptest.NewRecorder()
	withConflict.ServeHTTP(conditional, conditionalRequest)
	if conditional.Code != http.StatusNotModified {
		t.Fatalf("conflict ETag response = %d %s", conditional.Code, conditional.Body.String())
	}
}

func TestStrictJSONAndRequestLimit(t *testing.T) {
	handler := newFakeHandler(t, &fakeEngine{})
	tests := []struct {
		name        string
		body        string
		contentType string
		status      int
		code        string
	}{
		{name: "missing content type", body: `{}`, status: http.StatusUnsupportedMediaType, code: "unsupported_media_type"},
		{name: "wrong content type", body: `{}`, contentType: "text/plain", status: http.StatusUnsupportedMediaType, code: "unsupported_media_type"},
		{name: "missing answers", body: `{"revision":"sha256:no"}`, contentType: "application/json", status: http.StatusBadRequest, code: "malformed_request"},
		{name: "unknown field", body: `{"revision":"sha256:no","answers":[],"slots":[]}`, contentType: "application/json", status: http.StatusBadRequest, code: "malformed_request"},
		{name: "trailing document", body: `{"revision":"sha256:no","answers":[]} {}`, contentType: "application/json", status: http.StatusBadRequest, code: "malformed_request"},
		{name: "malformed", body: `{"revision":`, contentType: "application/json", status: http.StatusBadRequest, code: "malformed_request"},
		{name: "oversized", body: `{"revision":"sha256:no","answers":[],"padding":"` + strings.Repeat("x", MaxRequestBytes) + `"}`, contentType: "application/json", status: http.StatusRequestEntityTooLarge, code: "payload_too_large"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := doRequest(handler, http.MethodPost, "/api/v2/round", test.body, test.contentType, true)
			if response.Code != test.status || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) || strings.Contains(response.Body.String(), testToken) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
	if rounds, approvals := handler.(*Server).engine.(*fakeEngine).counts(); rounds != 0 || approvals != 0 {
		t.Fatalf("malformed requests mutated engine: rounds %d approvals %d", rounds, approvals)
	}

	for name, body := range map[string][]byte{
		"recursive duplicate": []byte(`{"revision":"sha256:no","answers":[{"question_id":"goal","value":"one","value":"two"}]}`),
		"invalid utf8":        append([]byte(`{"revision":"sha256:no","answers":[{"question_id":"goal","value":"`), 0xff, '"', '}', ']', '}'),
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://"+testAuthority+"/api/v2/round", bytes.NewReader(body))
			request.Host = testAuthority
			request.Header.Set("Authorization", "Bearer "+testToken)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"malformed_request"`) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
	unsupportedCharset := doRequest(handler, http.MethodPost, "/api/v2/round", `{}`, "application/json; charset=iso-8859-1", true)
	if unsupportedCharset.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unsupported charset = %d %s", unsupportedCharset.Code, unsupportedCharset.Body.String())
	}
	chunked := httptest.NewRequest(http.MethodPost, "http://"+testAuthority+"/api/v2/round", strings.NewReader(strings.Repeat("x", MaxRequestBytes+1)))
	chunked.Host = testAuthority
	chunked.ContentLength = -1
	chunked.TransferEncoding = []string{"chunked"}
	chunked.Header.Set("Authorization", "Bearer "+testToken)
	chunked.Header.Set("Content-Type", "application/json")
	chunkedResponse := httptest.NewRecorder()
	handler.ServeHTTP(chunkedResponse, chunked)
	if chunkedResponse.Code != http.StatusRequestEntityTooLarge || !strings.Contains(chunkedResponse.Body.String(), `"code":"payload_too_large"`) {
		t.Fatalf("chunked oversized body = %d %s", chunkedResponse.Code, chunkedResponse.Body.String())
	}
}

func TestErrorEnvelopeIncludesRequestIDRetryAndRevision(t *testing.T) {
	handler := newFakeHandler(t, &fakeEngine{})
	current := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v2/round", `{"revision":"sha256:stale","answers":[]}`, "application/json", true)
	var envelope errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusConflict || envelope.Revision != current.Revision || !envelope.Error.Retryable || len(envelope.Error.RequestID) != 32 {
		t.Fatalf("error envelope = %d %#v", response.Code, envelope)
	}
}

func TestServerLogsOnlySanitizedInternalFailures(t *testing.T) {
	var logs bytes.Buffer
	fake := &fakeEngine{
		snapshot: engine.Snapshot{Ready: true},
		writeErr: &engine.Failure{Class: engine.FailureOperational, Code: "engine_operation_failed", Cause: errors.New("disk failed token=super-secret")},
	}
	handler, err := NewHandler(HandlerConfig{
		Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, ErrOut: &logs,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("internal response = %d %s", response.Code, response.Body.String())
	}
	line := logs.String()
	for _, expected := range []string{"request_id=", "route=/api/v2/approve", "stage=approve", "cause=redacted operational failure"} {
		if !strings.Contains(line, expected) {
			t.Fatalf("safe log missing %q: %q", expected, line)
		}
	}
	if strings.Contains(line, "super-secret") || strings.Contains(line, testToken) {
		t.Fatalf("safe log leaked a capability: %q", line)
	}
	beforeLength := logs.Len()
	fake.writeErr = engineRejected(errors.New("review rejected"))
	response = doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if response.Code != http.StatusUnprocessableEntity || logs.Len() != beforeLength {
		t.Fatalf("domain rejection was logged: status=%d logs=%q", response.Code, logs.String())
	}
}

func TestRealLoopbackListenerBootstrapBearerApprovalAndFreeze(t *testing.T) {
	example := filepath.Join(t.TempDir(), "listener")
	eng, snapshot, err := engine.Open(context.Background(), engine.Config{
		ExampleDir: example, FromExample: filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render"), NetworkPolicy: "never",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	authority := listener.Addr().String()
	handler, err := NewHandler(HandlerConfig{Engine: eng, Snapshot: snapshot, ExampleDir: example, Token: token, AccessCode: testAccessCode, Authority: authority})
	if err != nil {
		t.Fatal(err)
	}
	server := newHTTPServer(handler)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	defer func() {
		_ = server.Shutdown(context.Background())
		<-served
	}()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	baseURL := "http://" + authority
	bootstrap, err := client.PostForm(baseURL+"/", url.Values{"code": []string{testAccessCode}})
	if err != nil {
		t.Fatal(err)
	}
	_ = bootstrap.Body.Close()
	if bootstrap.StatusCode != http.StatusOK || bootstrap.Request.URL.Path != instanceBasePath(token) {
		t.Fatalf("bootstrap final response = %d %s", bootstrap.StatusCode, bootstrap.Request.URL)
	}
	request, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v2/snapshot", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var current Response
	if err := json.NewDecoder(response.Body).Decode(&current); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	approvalBody := fmt.Sprintf(`{"revision":%q,"human_approved":true}`, current.Revision)
	request, _ = http.NewRequest(http.MethodPost, baseURL+"/api/v2/approve", strings.NewReader(approvalBody))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var approved Response
	if err := json.NewDecoder(response.Body).Decode(&approved); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !approved.Completed || approved.WriteResult == nil {
		t.Fatalf("approval response = %d %#v", response.StatusCode, approved)
	}
	request, _ = http.NewRequest(http.MethodPost, baseURL+"/api/v2/approve", strings.NewReader(fmt.Sprintf(`{"revision":%q,"human_approved":true}`, approved.Revision)))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("frozen response = %d", response.StatusCode)
	}
}

func TestRealLoopbackListenerRound(t *testing.T) {
	fake := &fakeEngine{snapshot: engine.Snapshot{Boundary: elicitor.WorkflowBoundary{Outcome: "before"}}}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	authority := listener.Addr().String()
	handler, err := NewHandler(HandlerConfig{
		Engine: fake, Snapshot: fake.snapshot, ExampleDir: t.TempDir(), Token: testToken, AccessCode: testAccessCode, Authority: authority,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := newHTTPServer(handler)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	defer func() {
		_ = server.Shutdown(context.Background())
		<-served
	}()
	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := "http://" + authority
	request, _ := http.NewRequest(http.MethodGet, baseURL+"/api/v2/snapshot", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var current Response
	if err := json.NewDecoder(response.Body).Decode(&current); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"outcome","value":"after"}]}`, current.Revision)
	request, _ = http.NewRequest(http.MethodPost, baseURL+"/api/v2/round", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var applied Response
	if err := json.NewDecoder(response.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || applied.Snapshot.Boundary.Outcome != "after" || applied.Revision == current.Revision {
		t.Fatalf("round response = %d %#v", response.StatusCode, applied)
	}
}

func TestRealListenerRejectsSlowBody(t *testing.T) {
	fake := &fakeEngine{}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	authority := listener.Addr().String()
	handler, err := NewHandler(HandlerConfig{
		Engine: fake, Snapshot: fake.snapshot, ExampleDir: t.TempDir(), Token: testToken, AccessCode: testAccessCode, Authority: authority,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := newHTTPServer(handler)
	server.ReadTimeout = 100 * time.Millisecond
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	defer func() {
		_ = server.Close()
		<-served
	}()
	connection, err := net.DialTimeout("tcp4", authority, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := fmt.Fprintf(connection, "POST /api/v2/round HTTP/1.1\r\nHost: %s\r\nAuthorization: Bearer %s\r\nContent-Type: application/json\r\nContent-Length: 100\r\nConnection: close\r\n\r\n{", authority, testToken); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("slow body status = %d", response.StatusCode)
	}
}

func TestRealListenerRejectsOversizedHeader(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := newHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	defer func() {
		_ = server.Close()
		<-served
	}()
	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := fmt.Fprintf(connection, "GET / HTTP/1.1\r\nHost: %s\r\nX-Oversized: %s\r\n\r\n", listener.Addr().String(), strings.Repeat("x", 64<<10)); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("oversized header status = %d", response.StatusCode)
	}
}

func TestRevisionRoundApprovalAndFrozenInspection(t *testing.T) {
	fake := &fakeEngine{
		snapshot: engine.Snapshot{Ready: false},
		result:   &engine.WriteResult{CleanupWarnings: []string{"temporary backup cleanup incomplete"}},
	}
	handler := newFakeHandler(t, fake)
	initial := currentResponse(t, handler)

	malformed := doRequest(handler, http.MethodPost, "/api/v2/round", `{"revision":"`+initial.Revision+`","answers":[{"question_id":"","value":"x"}]}`, "application/json", true)
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("malformed status = %d", malformed.Code)
	}
	stale := doRequest(handler, http.MethodPost, "/api/v2/round", `{"revision":"sha256:stale","answers":[{"question_id":"goal","value":"new"}]}`, "application/json", true)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale status = %d body %s", stale.Code, stale.Body.String())
	}
	if rounds, _ := fake.counts(); rounds != 0 {
		t.Fatalf("stale request called engine %d times", rounds)
	}

	round := doRequest(handler, http.MethodPost, "/api/v2/round", `{"revision":"`+initial.Revision+`","answers":[{"question_id":"goal","value":"new outcome"}]}`, "application/json; charset=utf-8", true)
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
	declined := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":false}`, afterRound.Revision), "application/json", true)
	if declined.Code != http.StatusUnprocessableEntity || !strings.Contains(declined.Body.String(), "explicit human approval") {
		t.Fatalf("declined approval = %d %s", declined.Code, declined.Body.String())
	}
	if current := currentResponse(t, handler); current.Revision != afterRound.Revision || current.Completed {
		t.Fatalf("declined approval changed state: %#v", current)
	}

	approve := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true,"allow_overwrite":true,"approve_incomplete":true}`, afterRound.Revision), "application/json", true)
	if approve.Code != http.StatusOK {
		t.Fatalf("approve status = %d body %s", approve.Code, approve.Body.String())
	}
	approved := decodeResponse(t, approve)
	if !approved.Completed || approved.WriteResult == nil || approved.Revision == afterRound.Revision || len(approved.WriteResult.CleanupWarnings) != 1 {
		t.Fatalf("approved response = %#v", approved)
	}
	fake.mu.Lock()
	if !fake.approval.HumanApproved || !fake.approval.AllowOverwrite || !fake.approval.ApproveIncomplete {
		t.Fatalf("approval flags = %#v", fake.approval)
	}
	fake.mu.Unlock()

	frozen := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, approved.Revision), "application/json", true)
	if frozen.Code != http.StatusConflict || !strings.Contains(frozen.Body.String(), "session_frozen") {
		t.Fatalf("frozen response = %d %s", frozen.Code, frozen.Body.String())
	}
	frozenWithStaleRevision := doRequest(handler, http.MethodPost, "/api/v2/round", `{"revision":"sha256:stale","answers":[]}`, "application/json", true)
	if frozenWithStaleRevision.Code != http.StatusConflict || !strings.Contains(frozenWithStaleRevision.Body.String(), "session_frozen") || strings.Contains(frozenWithStaleRevision.Body.String(), "stale_revision") {
		t.Fatalf("frozen stale response = %d %s", frozenWithStaleRevision.Code, frozenWithStaleRevision.Body.String())
	}
	inspection := currentResponse(t, handler)
	if !inspection.Completed || inspection.Revision != approved.Revision || inspection.WriteResult == nil || len(inspection.WriteResult.CleanupWarnings) != 1 {
		t.Fatalf("frozen inspection = %#v", inspection)
	}
	if rounds, approvals := fake.counts(); rounds != 1 || approvals != 2 {
		t.Fatalf("engine counts = rounds %d approvals %d", rounds, approvals)
	}
}

func TestReopenRequiresExactMutableRevision(t *testing.T) {
	fake := &fakeEngine{snapshot: engine.Snapshot{
		Ready: true, ApprovalRequired: true,
		RevisableDecisions: []elicitor.RevisableDecision{{QuestionID: "boundary.actor_trigger", Prompt: "Who starts it?", Value: "operator | on demand"}},
	}}
	handler := newFakeHandler(t, fake)
	initial := currentResponse(t, handler)

	missing := doRequest(handler, http.MethodPost, "/api/v2/reopen", fmt.Sprintf(`{"revision":%q}`, initial.Revision), "application/json", true)
	if missing.Code != http.StatusBadRequest || !strings.Contains(missing.Body.String(), "question_id is required") {
		t.Fatalf("missing question = %d %s", missing.Code, missing.Body.String())
	}
	unknown := doRequest(handler, http.MethodPost, "/api/v2/reopen", fmt.Sprintf(`{"revision":%q,"question_id":"boundary.actor_trigger","extra":true}`, initial.Revision), "application/json", true)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown reopen field = %d %s", unknown.Code, unknown.Body.String())
	}
	stale := doRequest(handler, http.MethodPost, "/api/v2/reopen", `{"revision":"sha256:stale","question_id":"boundary.actor_trigger"}`, "application/json", true)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "stale_revision") {
		t.Fatalf("stale reopen = %d %s", stale.Code, stale.Body.String())
	}
	if count, _ := fake.reopenRecord(); count != 0 {
		t.Fatalf("invalid requests called reopen %d times", count)
	}

	reopened := doRequest(handler, http.MethodPost, "/api/v2/reopen", fmt.Sprintf(`{"revision":%q,"question_id":"boundary.actor_trigger"}`, initial.Revision), "application/json", true)
	if reopened.Code != http.StatusOK {
		t.Fatalf("reopen = %d %s", reopened.Code, reopened.Body.String())
	}
	afterReopen := decodeResponse(t, reopened)
	if afterReopen.Revision == initial.Revision || len(afterReopen.Snapshot.Frontier) != 1 || afterReopen.Snapshot.Frontier[0].ID != "boundary.actor_trigger" || afterReopen.Snapshot.ApprovalRequired {
		t.Fatalf("reopen response = %#v", afterReopen)
	}
	if count, questionID := fake.reopenRecord(); count != 1 || questionID != "boundary.actor_trigger" {
		t.Fatalf("reopen record = %d %q", count, questionID)
	}

	approved := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, afterReopen.Revision), "application/json", true)
	if approved.Code != http.StatusOK {
		t.Fatalf("approval = %d %s", approved.Code, approved.Body.String())
	}
	frozenState := decodeResponse(t, approved)
	frozen := doRequest(handler, http.MethodPost, "/api/v2/reopen", fmt.Sprintf(`{"revision":%q,"question_id":"boundary.actor_trigger"}`, frozenState.Revision), "application/json", true)
	if frozen.Code != http.StatusConflict || !strings.Contains(frozen.Body.String(), "session_frozen") {
		t.Fatalf("frozen reopen = %d %s", frozen.Code, frozen.Body.String())
	}

	driftFake := &fakeEngine{snapshot: fake.snapshot, workspace: engine.WorkspaceStatus{ExternallyModified: true}}
	driftHandler := newFakeHandler(t, driftFake)
	driftState := currentResponse(t, driftHandler)
	drifted := doRequest(driftHandler, http.MethodPost, "/api/v2/reopen", fmt.Sprintf(`{"revision":%q,"question_id":"boundary.actor_trigger"}`, driftState.Revision), "application/json", true)
	if drifted.Code != http.StatusConflict || !strings.Contains(drifted.Body.String(), "workspace_changed") {
		t.Fatalf("drifted reopen = %d %s", drifted.Code, drifted.Body.String())
	}
}

func TestRejectedReopenIdentifiesDecision(t *testing.T) {
	fake := &fakeEngine{reopenErr: engineRejected(authoring.WithQuestionID("boundary.actor_trigger", errors.New("decision is no longer settled")))}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v2/reopen", fmt.Sprintf(`{"revision":%q,"question_id":"boundary.actor_trigger"}`, before.Revision), "application/json", true)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reopen rejection = %d %s", response.Code, response.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.QuestionID != "boundary.actor_trigger" || envelope.Error.Message != "decision is no longer settled" {
		t.Fatalf("reopen rejection envelope = %#v", envelope)
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
	response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true,"approve_incomplete":true}`, before.Revision), "application/json", true)
	if response.Code != http.StatusOK {
		t.Fatalf("incomplete approval = %d %s", response.Code, response.Body.String())
	}
	after := decodeResponse(t, response)
	if !after.Completed || after.WriteResult == nil || !after.WriteResult.Incomplete {
		t.Fatalf("incomplete response = %#v", after)
	}
	frozen := doRequest(handler, http.MethodPost, "/api/v2/round", fmt.Sprintf(`{"revision":%q,"answers":[]}`, after.Revision), "application/json", true)
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
			fake := &fakeEngine{snapshot: engine.Snapshot{Ready: true}, writeErr: engineRejected(errors.New(test.failure))}
			handler := newFakeHandler(t, fake)
			before := currentResponse(t, handler)
			response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
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

func TestCanceledRoundLeavesCachedRevisionAvailable(t *testing.T) {
	fake := &fakeEngine{
		snapshot: engine.Snapshot{Boundary: elicitor.WorkflowBoundary{Outcome: "before"}},
		roundErr: &engine.Failure{Class: engine.FailureOperational, Code: "engine_operation_failed", Cause: context.Canceled},
	}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"goal","value":"autosaved after cancellation"}]}`, before.Revision)
	request := httptest.NewRequest(http.MethodPost, "http://"+testAuthority+"/api/v2/round", strings.NewReader(body))
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
	if after.Revision != before.Revision || after.Snapshot.Boundary.Outcome != "before" {
		t.Fatalf("transactional rejection changed cached state: before %#v after %#v", before, after)
	}
	staleApproval := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if staleApproval.Code != http.StatusOK {
		t.Fatalf("approval after rejected round = %d %s", staleApproval.Code, staleApproval.Body.String())
	}
}

func TestFailedWorkspaceInspectionCanRecover(t *testing.T) {
	fake := &fakeEngine{
		snapshot: engine.Snapshot{Boundary: elicitor.WorkflowBoundary{Outcome: "before"}},
	}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	fake.snapshotErr = errors.New("workspace inspection unavailable")
	inspection := doRequest(handler, http.MethodGet, "/api/v2/snapshot", "", "", true)
	if inspection.Code != http.StatusInternalServerError {
		t.Fatalf("failed inspection response = %d %s", inspection.Code, inspection.Body.String())
	}
	fake.snapshotErr = nil
	after := currentResponse(t, handler)
	if after.Revision != before.Revision || after.Snapshot.Boundary.Outcome != "before" {
		t.Fatalf("recovered inspection changed state: before %#v after %#v", before, after)
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
			name: "round autosave", fake: &fakeEngine{roundErr: operationFailure}, endpoint: "/api/v2/round",
			body: func(revision string) string {
				return fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"goal","value":"new"}]}`, revision)
			},
		},
		{
			name: "approval write", fake: &fakeEngine{snapshot: engine.Snapshot{Ready: true}, writeErr: operationFailure}, endpoint: "/api/v2/approve",
			body: func(revision string) string {
				return fmt.Sprintf(`{"revision":%q,"human_approved":true}`, revision)
			},
		},
		{
			name: "reopen autosave", fake: &fakeEngine{reopenErr: operationFailure}, endpoint: "/api/v2/reopen",
			body: func(revision string) string {
				return fmt.Sprintf(`{"revision":%q,"question_id":"boundary.actor_trigger"}`, revision)
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

func TestRejectedRoundIdentifiesAuthoritativeQuestion(t *testing.T) {
	fake := &fakeEngine{roundErr: engineRejected(authoring.WithQuestionID("boundary.actor_trigger", errors.New("actor and trigger are invalid")))}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	response := doRequest(handler, http.MethodPost, "/api/v2/round", fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"boundary.actor_trigger","value":"invalid"}]}`, before.Revision), "application/json", true)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("rejection status = %d body %s", response.Code, response.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.QuestionID != "boundary.actor_trigger" || envelope.Error.Message != "actor and trigger are invalid" {
		t.Fatalf("rejection envelope = %#v", envelope)
	}
}

func TestStructuredDeferralIsBoundToTheQuestionAnswer(t *testing.T) {
	fake := &fakeEngine{}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"workflow.fallback","deferral":{"owner":"API owner","impact":"draft remains incomplete","unblock_condition":"provider publishes a spec","suggested_next_action":"add the reviewed source"}}]}`, before.Revision)
	response := doRequest(handler, http.MethodPost, "/api/v2/round", body, "application/json", true)
	if response.Code != http.StatusOK {
		t.Fatalf("structured deferral = %d %s", response.Code, response.Body.String())
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.seen) != 1 || fake.seen[0].QuestionID != "workflow.fallback" || fake.seen[0].Value != "defer:API owner | draft remains incomplete | provider publishes a spec | add the reviewed source" {
		t.Fatalf("encoded deferral = %#v", fake.seen)
	}
}

func TestMalformedStructuredDeferralIsQuestionAddressable(t *testing.T) {
	fake := &fakeEngine{}
	handler := newFakeHandler(t, fake)
	before := currentResponse(t, handler)
	body := fmt.Sprintf(`{"revision":%q,"answers":[{"question_id":"workflow.fallback","value":"also answer","deferral":{"owner":"API owner","impact":"blocked","unblock_condition":"spec exists","suggested_next_action":"retry"}}]}`, before.Revision)
	response := doRequest(handler, http.MethodPost, "/api/v2/round", body, "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("malformed deferral = %d %s", response.Code, response.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.QuestionID != "workflow.fallback" || !strings.Contains(envelope.Error.Message, "either value or deferral") {
		t.Fatalf("malformed deferral envelope = %#v", envelope)
	}
	if rounds, _ := fake.counts(); rounds != 0 {
		t.Fatalf("malformed deferral reached engine %d times", rounds)
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
	handler, err := NewHandler(HandlerConfig{
		Engine: authoringEngine, Snapshot: snapshot, ExampleDir: example, Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
	})
	if err != nil {
		t.Fatal(err)
	}
	before := currentResponse(t, handler)
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows"), []byte("blocks directory creation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("commit failure response = %d %s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("project.md exists after failed transaction: %v", err)
	}
	after := doRequest(handler, http.MethodGet, "/api/v2/snapshot", "", "", true)
	if after.Code != http.StatusInternalServerError {
		t.Fatalf("unsafe workspace inspection = %d %s", after.Code, after.Body.String())
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
		handler, err := NewHandler(HandlerConfig{Engine: authoringEngine, Snapshot: snapshot, ExampleDir: example, Token: testToken, AccessCode: testAccessCode, Authority: testAuthority})
		if err != nil {
			t.Fatal(err)
		}
		before := currentResponse(t, handler)
		report["checkedAt"] = now.Format(time.RFC3339Nano)
		writeTestJSON(t, reportPath, report)
		response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
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
		handler, err := NewHandler(HandlerConfig{Engine: authoringEngine, Snapshot: snapshot, ExampleDir: example, Token: testToken, AccessCode: testAccessCode, Authority: testAuthority})
		if err != nil {
			t.Fatal(err)
		}
		before := currentResponse(t, handler)
		if err := os.Rename(registryRoot, registryRoot+".unavailable"); err != nil {
			t.Fatal(err)
		}
		response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, before.Revision), "application/json", true)
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
			statuses <- doRequest(handler, http.MethodPost, "/api/v2/round", body, "application/json", true).Code
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
	handler, err := NewHandler(HandlerConfig{Engine: httpEngine, Snapshot: snapshot, ExampleDir: httpDir, Token: testToken, AccessCode: testAccessCode, Authority: testAuthority})
	if err != nil {
		t.Fatal(err)
	}
	revision := currentResponse(t, handler).Revision
	response := doRequest(handler, http.MethodPost, "/api/v2/approve", fmt.Sprintf(`{"revision":%q,"human_approved":true}`, revision), "application/json", true)
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

func TestEmbeddedShellContainsAccessiblePhaseCControls(t *testing.T) {
	html, err := assetFiles.ReadFile("assets/index.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`<main id="main-content"`,
		`<form id="round-form"`,
		`<div id="frontier-fields"`,
		`<form id="approval-form"`,
		`id="review-confirmed"`,
		`id="allow-overwrite"`,
		`data-approval="final"`,
		`data-approval="incomplete"`,
		`id="mutation-status"`,
		`id="completion-banner"`,
		`id="workspace-details"`,
		`id="review-status" class="state-pill"`,
		`id="readiness-panel" class="review-panel"`,
		`id="conflicts-panel" class="review-panel review-panel-wide"`,
		`id="candidate-workflows-list"`,
		`id="decision-evidence-list"`,
		`id="source-evidence-list"`,
		`id="review-heading" tabindex="-1"`,
		`role="alert"`,
		`aria-live="polite"`,
	} {
		if !bytes.Contains(html, []byte(required)) {
			t.Errorf("embedded shell missing %q", required)
		}
	}
	if bytes.Contains(html, []byte("onclick=")) || bytes.Contains(html, []byte("<style")) || bytes.Contains(html, []byte("<script>")) {
		t.Fatal("embedded shell contains inline script or style")
	}

	javascript, err := assetFiles.ReadFile("assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`input.dataset.questionId`,
		`control.input_kind === "choice"`,
		`deferral: Object.fromEntries`,
		`revision: state.renderedPayload.revision`,
		`human_approved: true`,
		`approve_incomplete: mode === "incomplete"`,
		`allow_overwrite: byID("allow-overwrite").checked`,
		`If-None-Match`,
		`document.hidden`,
		`maximumBackoff`,
		`pollGeneration`,
		`generation !== state.pollGeneration || state.pendingMutation`,
		`actionCell.dataset.label = "Action"`,
		`byID("review-section").dataset.state = reviewState`,
		`announceMutation("Round submitted. Continue with the next authoring question.")`,
		`successFocusID = "review-heading"`,
		`successFocusID = "completion-banner"`,
		`showQuestionError(error.question_id || "", failure.message)`,
		`renderCandidateWorkflows(snapshot)`,
		`renderDecisionEvidence(snapshot)`,
		`renderSourceEvidence(snapshot)`,
	} {
		if !bytes.Contains(javascript, []byte(required)) {
			t.Errorf("embedded client missing %q", required)
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
