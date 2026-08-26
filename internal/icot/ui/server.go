// Package ui provides the loopback-only HTTP transport for one iCoT engine.
// The API is experimental and intentionally internal to OpenUdon.
package ui

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

const (
	APIVersion       = "openudon.icot-ui-api.v4"
	SessionCookie    = "openudon_icot_ui"
	MaxRequestBytes  = 1 << 20
	MaxArtifactBytes = 2 << 20
	MaxJSONDepth     = 32
	humanInputSource = "user"
	instancePrefix   = "/.icot-ui/"
)

// AuthoringEngine is the engine contract used by the local transport.
type AuthoringEngine interface {
	ApplyRound(context.Context, []authoring.RoundAnswer) (engine.Snapshot, error)
	ReopenDecision(context.Context, string) (engine.Snapshot, error)
	ApproveAndWrite(context.Context, engine.Approval) (engine.ApprovalResult, error)
	WorkspaceStatus(context.Context) (engine.WorkspaceStatus, error)
}

type journeyEngine interface {
	SelectJourney(context.Context, string, string) (engine.Snapshot, error)
}

type sourceEngine interface {
	UploadSource(context.Context, string, io.Reader) (engine.UploadedSource, engine.Snapshot, error)
	StageUploadedSource(context.Context, string) (engine.Snapshot, error)
	RemoveStagedSource(context.Context, string) (engine.Snapshot, error)
}

type resumeEngine interface {
	ResumeAuthoring(context.Context) (engine.Snapshot, error)
}

type browserCaptureEngine interface {
	StageBrowserCapture(context.Context, engine.BrowserCaptureStage) (engine.Snapshot, error)
}

type CaptureSession interface {
	// Events closes only after worker protocol handling and process-tree
	// teardown have completed or reported a containment failure.
	Events() <-chan browserauthor.Event
	Respond(context.Context, browserauthor.Response) error
	Cancel()
}

// HandlerConfig configures one server handler after its loopback listener is
// active. Authority must be the listener's exact host:port value.
type HandlerConfig struct {
	Context             context.Context
	Engine              AuthoringEngine
	Snapshot            engine.Snapshot
	ExampleDir          string
	Token               string
	AccessCode          string
	Authority           string
	ErrOut              io.Writer
	AccessCodeOut       io.Writer
	Now                 func() time.Time
	GenerateAccessCode  func() (string, error)
	RepoRoot            string
	BuildPackage        func(context.Context, synthesize.Options) (*synthesize.Result, *synthesize.QualityReport, error)
	AssessPackage       func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
	InspectPackage      func(context.Context, trustedrunner.TemplateOptions) (trustedrunner.PackageInspection, error)
	RevalidatePackage   func(context.Context, trustedrunner.TemplateOptions, trustedrunner.PackageInspection) error
	PrivateRoot         string
	DriverDir           string
	DoctorBrowser       func(context.Context, string, string) (browserauthor.DoctorReport, error)
	StartCapture        func(context.Context, browserauthor.Config) (CaptureSession, error)
	PrepareCapture      func(CaptureStageRequest) (engine.BrowserCaptureStage, error)
	BrowserTransactions BrowserTransactionEngine
}

// Workspace identifies the selected example and its optimistic ownership
// status.
type Workspace struct {
	ExampleDir         string `json:"example_dir"`
	ExternallyModified bool   `json:"externally_modified"`
}

// Response is returned by every successful API request.
type Response struct {
	ETag               string                        `json:"-"`
	Version            string                        `json:"version"`
	Revision           string                        `json:"revision"`
	CaptureRevision    string                        `json:"capture_revision"`
	Lifecycle          string                        `json:"lifecycle"`
	Completed          bool                          `json:"completed"`
	Workspace          Workspace                     `json:"workspace"`
	Snapshot           engine.Snapshot               `json:"snapshot"`
	Capture            *CaptureState                 `json:"capture,omitempty"`
	BrowserDoctor      *browserauthor.UIDoctorReport `json:"browser_doctor,omitempty"`
	WriteResult        *engine.WriteResult           `json:"write_result,omitempty"`
	Package            *PackageState                 `json:"package,omitempty"`
	BrowserTransaction *BrowserTransactionResource   `json:"browser_transaction,omitempty"`
}

const (
	lifecycleAuthoring    = "authoring"
	lifecycleAuthored     = "authored"
	lifecyclePackageFail  = "package_failed"
	lifecycleHandoffReady = "handoff_ready"
)

// CaptureState contains only reduced protocol observations. Credential and
// challenge values, cookies, storage, stderr, and raw worker output have no
// representation in this API.
type CaptureState struct {
	State             string                     `json:"state"`
	Message           string                     `json:"message,omitempty"`
	Phase             string                     `json:"phase,omitempty"`
	Observation       *authorsession.Observation `json:"observation,omitempty"`
	Approval          *authorsession.Approval    `json:"approval,omitempty"`
	Checkpoint        *authorsession.Checkpoint  `json:"checkpoint,omitempty"`
	ResultReady       bool                       `json:"result_ready,omitempty"`
	ContainmentFailed bool                       `json:"containment_failed,omitempty"`
	StartedAt         string                     `json:"started_at,omitempty"`
	UpdatedAt         string                     `json:"updated_at,omitempty"`
}

type ArtifactSummary struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
}

type PackageState struct {
	Status               string                           `json:"status"`
	Quality              *PackageQuality                  `json:"quality,omitempty"`
	Inspection           *trustedrunner.PackageInspection `json:"inspection,omitempty"`
	Artifacts            []ArtifactSummary                `json:"artifacts,omitempty"`
	ApprovalTemplateArgv []string                         `json:"approval_template_argv,omitempty"`
	Remediation          []string                         `json:"remediation,omitempty"`
}

type PackageQuality struct {
	Status string                    `json:"status"`
	Checks []synthesize.QualityCheck `json:"checks"`
}

type errorEnvelope struct {
	Version  string       `json:"version"`
	Revision string       `json:"revision,omitempty"`
	Error    errorPayload `json:"error"`
}

type errorPayload struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable"`
	RequestID  string `json:"request_id"`
	QuestionID string `json:"question_id,omitempty"`
}

type roundRequest struct {
	Revision string        `json:"revision"`
	Answers  []roundAnswer `json:"answers"`
}

type roundAnswer struct {
	QuestionID string         `json:"question_id"`
	Value      string         `json:"value"`
	Deferral   *roundDeferral `json:"deferral,omitempty"`
}

type roundDeferral struct {
	Owner               string `json:"owner"`
	Impact              string `json:"impact"`
	UnblockCondition    string `json:"unblock_condition"`
	SuggestedNextAction string `json:"suggested_next_action"`
}

type approveRequest struct {
	Revision          string `json:"revision"`
	HumanApproved     bool   `json:"human_approved"`
	AllowOverwrite    bool   `json:"allow_overwrite"`
	ApproveIncomplete bool   `json:"approve_incomplete"`
}

type reopenRequest struct {
	Revision   string `json:"revision"`
	QuestionID string `json:"question_id"`
}

type revisionRequest struct {
	Revision string `json:"revision"`
}

type journeyRequest struct {
	Revision string `json:"revision"`
	Starter  string `json:"starter"`
	Goal     string `json:"goal"`
}

type sourceMutationRequest struct {
	Revision string `json:"revision"`
	ID       string `json:"id"`
}

type buildRequest struct {
	Revision  string `json:"revision"`
	Confirmed bool   `json:"confirmed"`
}

type captureStartRequest struct {
	Revision        string   `json:"revision"`
	CaptureRevision string   `json:"capture_revision"`
	ProfileID       string   `json:"profile_id"`
	URL             string   `json:"url"`
	DashboardURL    string   `json:"dashboard_url"`
	Goal            string   `json:"goal"`
	Origins         []string `json:"origins"`
	GoalOrigin      string   `json:"goal_origin,omitempty"`
	GoalPath        string   `json:"goal_path,omitempty"`
	GoalContext     string   `json:"goal_context,omitempty"`
	GoalRole        string   `json:"goal_role,omitempty"`
	GoalLabel       string   `json:"goal_label,omitempty"`
}

type captureRespondRequest struct {
	CaptureRevision string                 `json:"capture_revision"`
	Response        browserauthor.Response `json:"response"`
}

type captureMutationRequest struct {
	Revision        string `json:"revision,omitempty"`
	CaptureRevision string `json:"capture_revision"`
}

// CaptureStageRequest remains process-private; the Browsertools result path is
// passed only to the OpenUdon validator and is never serialized to the UI.
type CaptureStageRequest struct {
	Start       captureStartRequest
	ExampleDir  string
	PrivateRoot string
	Result      authorsession.Result
	Attestation *browserauthor.Attestation
}

type requestError struct {
	status int
	code   string
	text   string
}

func (e *requestError) Error() string { return e.text }

// Server serializes revisions, workspace inspection, and engine mutations.
type Server struct {
	mu sync.Mutex

	engine                   AuthoringEngine
	snapshot                 engine.Snapshot
	exampleDir               string
	token                    string
	accessCodeDigest         [sha256.Size]byte
	accessCodeExpires        time.Time
	accessCodeUsed           bool
	accessFailures           []time.Time
	accessRecoveries         []time.Time
	accessCodeOut            io.Writer
	generateAccessCode       func() (string, error)
	now                      func() time.Time
	authority                string
	origin                   string
	basePath                 string
	revision                 string
	captureRevision          string
	etag                     string
	lifecycle                string
	completed                bool
	writeResult              *engine.WriteResult
	capture                  *CaptureState
	packageState             *PackageState
	artifactPaths            map[string]string
	repoRoot                 string
	buildPackage             func(context.Context, synthesize.Options) (*synthesize.Result, *synthesize.QualityReport, error)
	assessPackage            func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
	inspectPackage           func(context.Context, trustedrunner.TemplateOptions) (trustedrunner.PackageInspection, error)
	revalidatePackage        func(context.Context, trustedrunner.TemplateOptions, trustedrunner.PackageInspection) error
	privateRoot              string
	driverDir                string
	doctorBrowser            func(context.Context, string, string) (browserauthor.DoctorReport, error)
	startCapture             func(context.Context, browserauthor.Config) (CaptureSession, error)
	prepareCapture           func(CaptureStageRequest) (engine.BrowserCaptureStage, error)
	captureSession           CaptureSession
	captureCancel            context.CancelFunc
	captureResult            *authorsession.Result
	captureAttestation       *browserauthor.Attestation
	captureStart             captureStartRequest
	captureContainmentFailed bool
	doctorReport             *browserauthor.UIDoctorReport
	captureContext           context.Context
	workspace                engine.WorkspaceStatus
	errOut                   io.Writer
	browserTransactions      BrowserTransactionEngine
	browserTransaction       *BrowserTransactionSnapshot
}

var fallbackRequestID atomic.Uint64

var revisionDigest = digestRevision

// GenerateToken returns a 256-bit URL-safe per-process capability token.
func GenerateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate UI capability token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// GenerateAccessCode returns a terminal-only 12-character Crockford Base32
// bootstrap code. Ambiguous I, L, O, and U characters are never emitted.
func GenerateAccessCode() (string, error) {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	raw := make([]byte, 12)
	out := make([]byte, 12)
	for i := range out {
		for {
			if _, err := rand.Read(raw[i : i+1]); err != nil {
				return "", fmt.Errorf("generate UI access code: %w", err)
			}
			if raw[i] < 224 {
				out[i] = alphabet[int(raw[i])%len(alphabet)]
				break
			}
		}
	}
	return string(out), nil
}

// NewHandler builds the authenticated loopback handler for one engine.
func NewHandler(config HandlerConfig) (http.Handler, error) {
	if config.Engine == nil {
		return nil, errors.New("UI engine is required")
	}
	if strings.TrimSpace(config.ExampleDir) == "" {
		return nil, errors.New("UI example directory is required")
	}
	if strings.TrimSpace(config.Token) == "" {
		return nil, errors.New("UI capability token is required")
	}
	accessCode := strings.ToUpper(strings.TrimSpace(config.AccessCode))
	if !validAccessCode(accessCode) {
		return nil, errors.New("UI access code must contain 12 Crockford Base32 characters")
	}
	authority := strings.TrimSpace(config.Authority)
	if !validLoopbackAuthority(authority) {
		return nil, errors.New("UI authority must be an active 127.0.0.1 listener")
	}
	errOut := config.ErrOut
	if errOut == nil {
		errOut = io.Discard
	}
	accessCodeOut := config.AccessCodeOut
	generateAccessCode := config.GenerateAccessCode
	if generateAccessCode == nil {
		generateAccessCode = GenerateAccessCode
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	s := &Server{
		engine: config.Engine, snapshot: config.Snapshot,
		exampleDir: config.ExampleDir, token: config.Token, authority: authority,
		origin: "http://" + authority, basePath: instanceBasePath(config.Token), errOut: errOut,
		accessCodeOut: accessCodeOut, generateAccessCode: generateAccessCode,
		accessCodeDigest: sha256.Sum256([]byte(accessCode)), accessCodeExpires: now().Add(5 * time.Minute), now: now,
		lifecycle: lifecycleAuthoring, artifactPaths: map[string]string{}, repoRoot: strings.TrimSpace(config.RepoRoot),
		buildPackage: config.BuildPackage, assessPackage: config.AssessPackage, inspectPackage: config.InspectPackage, revalidatePackage: config.RevalidatePackage,
		privateRoot: strings.TrimSpace(config.PrivateRoot), driverDir: strings.TrimSpace(config.DriverDir),
		doctorBrowser: config.DoctorBrowser, startCapture: config.StartCapture, prepareCapture: config.PrepareCapture,
		browserTransactions: config.BrowserTransactions,
		captureContext:      config.Context,
	}
	if s.buildPackage == nil {
		s.buildPackage = synthesize.PackageFromIntent
	}
	if s.assessPackage == nil {
		s.assessPackage = synthesize.AssessCurrent
	}
	if s.inspectPackage == nil {
		s.inspectPackage = trustedrunner.InspectPackage
	}
	if s.revalidatePackage == nil {
		s.revalidatePackage = trustedrunner.RevalidatePackageBytes
	}
	if s.doctorBrowser == nil {
		s.doctorBrowser = browserauthor.Doctor
	}
	if s.startCapture == nil {
		s.startCapture = func(ctx context.Context, config browserauthor.Config) (CaptureSession, error) {
			return browserauthor.Start(ctx, config)
		}
	}
	if s.captureContext == nil {
		s.captureContext = context.Background()
	}
	if s.repoRoot == "" {
		s.repoRoot, _ = os.Getwd()
	}
	if s.browserTransactions != nil {
		transactionSnapshot, err := s.browserTransactions.Observe(config.Context)
		if err != nil {
			return nil, errors.New("initialize browser transaction resource")
		}
		s.browserTransaction = &transactionSnapshot
	}
	if err := s.updateRevisionLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

func validAccessCode(code string) bool {
	if len(code) != 12 {
		return false
	}
	for _, ch := range code {
		if !strings.ContainsRune("0123456789ABCDEFGHJKMNPQRSTVWXYZ", ch) {
			return false
		}
	}
	return true
}

func validLoopbackAuthority(authority string) bool {
	host, portText, err := net.SplitHostPort(authority)
	if err != nil || host != "127.0.0.1" {
		return false
	}
	port, err := strconv.Atoi(portText)
	return err == nil && port > 0 && port <= 65535
}

func instanceBasePath(token string) string {
	digest := sha256.Sum256([]byte("openudon.icot-ui.instance-path.v2\x00" + token))
	return instancePrefix + hex.EncodeToString(digest[:]) + "/"
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := newRequestID()
	setSecurityHeaders(w)
	routePath, cookieScoped := s.routePath(r.URL.Path)
	if r.Host != s.authority {
		s.writeError(w, http.StatusForbidden, "forbidden", "request Host is not the active loopback listener", false, requestID, "")
		return
	}
	origins := r.Header.Values("Origin")
	if len(origins) > 1 || (len(origins) == 1 && origins[0] != s.origin) {
		s.writeError(w, http.StatusForbidden, "forbidden", "request Origin is not the active loopback origin", false, requestID, "")
		return
	}

	switch routePath {
	case "/healthz":
		if cookieScoped {
			s.serveUnknown(w, r, cookieScoped, requestID)
			return
		}
		s.serveHealth(w, r, requestID)
	case "/":
		s.serveShell(w, r, cookieScoped, requestID)
	case "/assets/app.js":
		s.serveAsset(w, r, cookieScoped, requestID, "assets/app.js", "text/javascript; charset=utf-8")
	case "/assets/style.css":
		s.serveAsset(w, r, cookieScoped, requestID, "assets/style.css", "text/css; charset=utf-8")
	case "/api/v4/snapshot":
		s.serveSnapshot(w, r, cookieScoped, requestID)
	case "/api/v4/journey":
		s.serveJourney(w, r, cookieScoped, requestID)
	case "/api/v4/round":
		s.serveRound(w, r, cookieScoped, requestID)
	case "/api/v4/reopen":
		s.serveReopen(w, r, cookieScoped, requestID)
	case "/api/v4/source/upload":
		s.serveSourceUpload(w, r, cookieScoped, requestID)
	case "/api/v4/source/stage":
		s.serveSourceMutation(w, r, cookieScoped, requestID, true)
	case "/api/v4/source/remove":
		s.serveSourceMutation(w, r, cookieScoped, requestID, false)
	case "/api/v4/browser/preflight":
		s.serveBrowserPreflight(w, r, cookieScoped, requestID)
	case "/api/v4/capture/start":
		s.serveCaptureStart(w, r, cookieScoped, requestID)
	case "/api/v4/capture/respond":
		s.serveCaptureRespond(w, r, cookieScoped, requestID)
	case "/api/v4/capture/stage":
		s.serveCaptureStage(w, r, cookieScoped, requestID)
	case "/api/v4/capture/cancel":
		s.serveCaptureCancel(w, r, cookieScoped, requestID)
	case "/api/v4/author/approve":
		s.serveApprove(w, r, cookieScoped, requestID)
	case "/api/v4/author/resume":
		s.serveResume(w, r, cookieScoped, requestID)
	case "/api/v4/package/build":
		s.servePackageBuild(w, r, cookieScoped, requestID)
	case "/api/v4/artifact":
		s.serveArtifact(w, r, cookieScoped, requestID)
	case "/api/v4/browser-transactions/current":
		s.serveBrowserTransactionCurrent(w, r, cookieScoped, requestID)
	case "/api/v4/browser-transactions/start", "/api/v4/browser-transactions/review", "/api/v4/browser-transactions/prepare", "/api/v4/browser-transactions/promote", "/api/v4/browser-transactions/cancel", "/api/v4/browser-transactions/recovery/inspect", "/api/v4/browser-transactions/recovery/reconcile", "/api/v4/browser-transactions/selected/inspect":
		s.serveBrowserTransactionMutation(w, r, cookieScoped, requestID, routePath)
	default:
		s.serveUnknown(w, r, cookieScoped, requestID)
	}
}

func (s *Server) routePath(path string) (string, bool) {
	if !strings.HasPrefix(path, s.basePath) {
		return path, false
	}
	relative := strings.TrimPrefix(path, s.basePath)
	return "/" + relative, true
}

func (s *Server) serveUnknown(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	s.writeError(w, http.StatusNotFound, "not_found", "route not found", false, requestID, "")
}

func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request, requestID string) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, http.MethodGet, requestID)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) serveShell(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if !cookieScoped {
		s.serveBootstrap(w, r, requestID)
		return
	}
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, http.MethodGet, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := serveEmbedded(w, "assets/index.html", "text/html; charset=utf-8"); err != nil {
		s.writeInternalError(w, r, requestID, "/", "embedded_asset", err, true)
	}
}

func (s *Server) serveBootstrap(w http.ResponseWriter, r *http.Request, requestID string) {
	if r.URL.Path != "/" || len(r.URL.Query()) != 0 {
		s.writeError(w, http.StatusNotFound, "not_found", "route not found", false, requestID, "")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.writeBootstrapPage(w, "")
	case http.MethodPost:
		s.handleBootstrapPost(w, r, requestID)
	default:
		s.methodNotAllowed(w, "GET, POST", requestID)
	}
}

func (s *Server) writeBootstrapPage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>OpenUdon iCoT</title></head><body><main><h1>OpenUdon iCoT</h1>`)
	if message != "" {
		_, _ = io.WriteString(w, `<p role="status">`+html.EscapeString(message)+`</p>`)
	}
	_, _ = io.WriteString(w, `<p>Enter the 12-character access code shown in the terminal.</p><form method="post" action="/"><label>Access code <input name="code" inputmode="text" autocomplete="one-time-code" maxlength="12" required></label><button type="submit">Continue</button></form><hr><p>If the browser session was lost after the code was used, print a fresh single-use code in the terminal.</p><form method="post" action="/"><button type="submit" name="recover" value="1">Print a fresh terminal code</button></form></main></body></html>`)
}

func (s *Server) handleBootstrapPost(w http.ResponseWriter, r *http.Request, requestID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "access-code form is invalid", false, requestID, "")
		return
	}
	if len(r.Form) == 1 && len(r.Form["recover"]) == 1 && r.Form.Get("recover") == "1" {
		s.recoverAccessCode(w, r, requestID)
		return
	}
	if len(r.Form) != 1 || len(r.Form["code"]) != 1 {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "exactly one access code is required", false, requestID, "")
		return
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessFailures = retainRecentAttempts(s.accessFailures, now)
	if len(s.accessFailures) >= 5 {
		s.writeError(w, http.StatusTooManyRequests, "rate_limited", "too many failed access-code attempts; try again later", true, requestID, "")
		return
	}
	codeDigest := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(r.Form.Get("code")))))
	valid := !s.accessCodeUsed && now.Before(s.accessCodeExpires) && subtle.ConstantTimeCompare(codeDigest[:], s.accessCodeDigest[:]) == 1
	if !valid {
		s.accessFailures = append(s.accessFailures, now)
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "the access code is invalid, expired, or already used", false, requestID, "")
		return
	}
	s.accessCodeUsed = true
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: s.token, Path: s.basePath, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, s.basePath, http.StatusSeeOther)
}

func (s *Server) recoverAccessCode(w http.ResponseWriter, r *http.Request, requestID string) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessRecoveries = retainRecentAttempts(s.accessRecoveries, now)
	if len(s.accessRecoveries) >= 5 {
		s.writeError(w, http.StatusTooManyRequests, "rate_limited", "too many access-code recovery requests; try again later", true, requestID, "")
		return
	}
	if !s.accessCodeUsed && now.Before(s.accessCodeExpires) {
		s.writeError(w, http.StatusConflict, "access_code_active", "the current access code is still active; use the code already shown in the terminal", false, requestID, "")
		return
	}
	if s.accessCodeOut == nil {
		s.writeError(w, http.StatusConflict, "access_code_recovery_unavailable", "access-code recovery is unavailable; restart the UI server to print a new code", false, requestID, "")
		return
	}
	code, err := s.generateAccessCode()
	if err != nil {
		s.writeInternalError(w, r, requestID, "/", "access_code_generation", err, true)
		return
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if !validAccessCode(code) {
		s.writeInternalError(w, r, requestID, "/", "access_code_generation", errors.New("generated access code is invalid"), true)
		return
	}
	if _, err := fmt.Fprintf(s.accessCodeOut, "icot ui replacement access code: %s\n", code); err != nil {
		s.writeInternalError(w, r, requestID, "/", "access_code_terminal", err, true)
		return
	}
	s.accessCodeDigest = sha256.Sum256([]byte(code))
	s.accessCodeExpires = now.Add(5 * time.Minute)
	s.accessCodeUsed = false
	s.accessFailures = nil
	s.accessRecoveries = append(s.accessRecoveries, now)
	s.writeBootstrapPage(w, "A fresh single-use code was printed in the terminal. It expires in five minutes.")
}

func retainRecentAttempts(attempts []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-time.Minute)
	kept := attempts[:0]
	for _, attemptedAt := range attempts {
		if attemptedAt.After(cutoff) {
			kept = append(kept, attemptedAt)
		}
	}
	return kept
}

func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID, name, contentType string) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, http.MethodGet, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	if err := serveEmbedded(w, name, contentType); err != nil {
		s.writeInternalError(w, r, requestID, "/assets", "embedded_asset", err, true)
	}
}

func (s *Server) serveSnapshot(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, http.MethodGet, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(r.Context()); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/snapshot", "workspace_inspection", err, true)
		return
	}
	if s.lifecycle == lifecycleHandoffReady {
		if err := s.validateFrozenArtifactsLocked(r.Context()); err != nil {
			s.invalidateHandoffLocked()
			if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
				s.writeInternalError(w, r, requestID, "/api/v4/snapshot", "revision", revisionErr, true)
				return
			}
		}
	}
	setETag(w, s.etag)
	if matchesETag(r.Header.Get("If-None-Match"), s.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveJourney(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request journeyRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	selected, ok := s.engine.(journeyEngine)
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "unsupported", "journey selection is unavailable", false, requestID, s.currentRevision())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, "/api/v4/journey", strings.TrimSpace(request.Revision)) {
		return
	}
	snapshot, err := selected.SelectJourney(r.Context(), request.Starter, request.Goal)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v4/journey", "select_journey", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/journey", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveSourceUpload(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, engine.MaxUploadBytes+(64<<10))
	reader, err := r.MultipartReader()
	if err != nil {
		s.writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be multipart/form-data", false, requestID, s.currentRevision())
		return
	}
	var revision, filename string
	var sourceBytes []byte
	seen := map[string]bool{}
	for {
		part, partErr := reader.NextPart()
		if errors.Is(partErr, io.EOF) {
			break
		}
		if partErr != nil {
			s.writeError(w, http.StatusBadRequest, "malformed_request", "multipart upload is malformed", false, requestID, s.currentRevision())
			return
		}
		name := part.FormName()
		if seen[name] || name != "revision" && name != "source" {
			s.writeError(w, http.StatusBadRequest, "malformed_request", "upload requires exactly one revision and source part", false, requestID, s.currentRevision())
			return
		}
		seen[name] = true
		if name == "revision" {
			value, readErr := io.ReadAll(io.LimitReader(part, 1025))
			if readErr != nil || len(value) > 1024 || part.FileName() != "" {
				s.writeError(w, http.StatusBadRequest, "malformed_request", "revision upload part is invalid", false, requestID, s.currentRevision())
				return
			}
			revision = strings.TrimSpace(string(value))
		} else {
			filename = part.FileName()
			value, readErr := io.ReadAll(io.LimitReader(part, engine.MaxUploadBytes+1))
			if readErr != nil || int64(len(value)) > engine.MaxUploadBytes || filename == "" {
				s.writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "uploaded source is missing or exceeds 20 MiB", false, requestID, s.currentRevision())
				return
			}
			sourceBytes = value
		}
	}
	if revision == "" || !seen["source"] {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "upload requires exactly one revision and source part", false, requestID, s.currentRevision())
		return
	}
	uploader, ok := s.engine.(sourceEngine)
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "unsupported", "source upload is unavailable", false, requestID, s.currentRevision())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, "/api/v4/source/upload", revision) {
		return
	}
	_, snapshot, err := uploader.UploadSource(r.Context(), filename, bytes.NewReader(sourceBytes))
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v4/source/upload", "upload_source", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/source/upload", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveSourceMutation(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string, stage bool) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request sourceMutationRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	mutator, ok := s.engine.(sourceEngine)
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "unsupported", "source mutation is unavailable", false, requestID, s.currentRevision())
		return
	}
	route, operation := "/api/v4/source/remove", "remove_source"
	if stage {
		route, operation = "/api/v4/source/stage", "stage_source"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, route, strings.TrimSpace(request.Revision)) {
		return
	}
	var snapshot engine.Snapshot
	var err error
	if stage {
		snapshot, err = mutator.StageUploadedSource(r.Context(), request.ID)
	} else {
		snapshot, err = mutator.RemoveStagedSource(r.Context(), request.ID)
	}
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, route, operation, err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, route, "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveBrowserPreflight(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request captureMutationRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	if !s.beginMutation(w, r, requestID, "/api/v4/browser/preflight", strings.TrimSpace(request.Revision)) {
		s.mu.Unlock()
		return
	}
	if strings.TrimSpace(request.CaptureRevision) != s.captureRevision {
		s.writeError(w, http.StatusConflict, "stale_capture_revision", "capture revision is stale", true, requestID, s.revision)
		s.mu.Unlock()
		return
	}
	if s.privateRoot == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "private_root_required", "browser capture requires icot ui --private-root", false, requestID, s.revision)
		s.mu.Unlock()
		return
	}
	previousCapture, previousRevision, previousCaptureRevision, previousETag := s.capture, s.revision, s.captureRevision, s.etag
	s.capture = &CaptureState{State: "preflight", Message: "Checking the installed Playwright driver and Chromium runtime.", UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	if err := s.updateRevisionLocked(); err != nil {
		s.capture, s.revision, s.captureRevision, s.etag = previousCapture, previousRevision, previousCaptureRevision, previousETag
		s.writeInternalError(w, r, requestID, "/api/v4/browser/preflight", "revision", err, true)
		s.mu.Unlock()
		return
	}
	preflightRevision := s.captureRevision
	preflightContext, preflightCancel := context.WithCancel(r.Context())
	s.captureCancel = preflightCancel
	s.mu.Unlock()

	// Doctor may take up to 30 seconds. Keep the state lock free so polling can
	// continue to render the explicit preflight state while the isolated worker
	// performs the readiness check.
	report, err := s.doctorBrowser(preflightContext, s.privateRoot, s.driverDir)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.capture != nil && s.capture.State == "canceling" && s.captureCancel != nil {
		preflightCancel()
		s.captureCancel = nil
		state := "canceled"
		message := "Browser readiness checking was canceled after the isolated worker stopped."
		containmentFailed := browserauthor.TeardownFailed(err)
		if containmentFailed {
			state = "failed"
			message = "The Chromium readiness worker did not confirm process-tree teardown. Restart iCoT before another browser capture."
			s.captureContainmentFailed = true
		}
		s.capture = &CaptureState{State: state, Message: message, ContainmentFailed: containmentFailed, StartedAt: s.capture.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		_ = s.updateRevisionLocked()
		s.writeError(w, http.StatusConflict, "capture_canceled", "browser preflight was canceled", false, requestID, s.revision)
		return
	}
	if s.captureRevision != preflightRevision || s.capture == nil || s.capture.State != "preflight" {
		preflightCancel()
		s.writeError(w, http.StatusConflict, "stale_capture_revision", "browser preflight state changed before the readiness check completed", true, requestID, s.revision)
		return
	}
	preflightCancel()
	s.captureCancel = nil
	if browserauthor.TeardownFailed(err) {
		s.captureContainmentFailed = true
		s.capture = &CaptureState{State: "failed", Message: "The Chromium readiness worker did not confirm process-tree teardown. Restart iCoT before another browser capture.", ContainmentFailed: true, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		_ = s.updateRevisionLocked()
		s.writeError(w, http.StatusUnprocessableEntity, "capture_teardown_failed", "Chromium readiness process teardown was not confirmed; restart iCoT", false, requestID, s.revision)
		return
	}
	if report.Version == browserauthor.DoctorVersion && report.Engine == browserauthor.EngineChromium {
		reviewedReport := report.UI()
		if err != nil {
			reviewedReport.Error = "Chromium readiness check failed"
		}
		s.doctorReport = &reviewedReport
	}
	if err != nil {
		s.capture = &CaptureState{State: "failed", Message: "Chromium readiness check failed.", UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		_ = s.updateRevisionLocked()
		s.writeError(w, http.StatusUnprocessableEntity, "browser_unavailable", "Browsertools could not verify Chromium readiness", false, requestID, s.revision)
		return
	}
	s.capture = &CaptureState{State: "configuring", Message: "Chromium is ready. Review the exact capture authority before launch.", UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/browser/preflight", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveCaptureStart(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request captureStartRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(r.Context()); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/capture/start", "workspace_inspection", err, true)
		return
	}
	if strings.TrimSpace(request.Revision) != s.revision || strings.TrimSpace(request.CaptureRevision) != s.captureRevision {
		s.writeError(w, http.StatusConflict, "stale_revision", "authoring or capture revision is stale", true, requestID, s.revision)
		return
	}
	if s.lifecycle != lifecycleAuthoring || s.workspace.ExternallyModified {
		s.writeError(w, http.StatusConflict, "session_frozen", "browser capture is unavailable in the current authoring state", false, requestID, s.revision)
		return
	}
	if s.captureContainmentFailed {
		s.writeError(w, http.StatusConflict, "capture_teardown_failed", "a prior browser process tree did not confirm teardown; restart iCoT before another capture", false, requestID, s.revision)
		return
	}
	if captureActive(s.capture) && s.capture.State != "configuring" {
		s.writeError(w, http.StatusConflict, "capture_active", "only one browser capture may run at a time", true, requestID, s.revision)
		return
	}
	if s.privateRoot == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "private_root_required", "browser capture requires icot ui --private-root", false, requestID, s.revision)
		return
	}
	if s.capture == nil || s.capture.State != "configuring" || s.doctorReport == nil || !s.doctorReport.DriverReady || !s.doctorReport.BrowserReady {
		s.writeError(w, http.StatusConflict, "browser_preflight_required", "a passing Chromium preflight is required before browser capture launch", false, requestID, s.revision)
		return
	}
	goalOrigin := strings.TrimSpace(request.GoalOrigin)
	if goalOrigin == "" && len(request.Origins) > 0 {
		goalOrigin = request.Origins[len(request.Origins)-1]
	}
	goalPath := strings.TrimSpace(request.GoalPath)
	if goalPath == "" {
		goalPath = "/"
	}
	goalContext := strings.TrimSpace(request.GoalContext)
	if goalContext == "" {
		goalContext = "main"
	}
	goalRole := strings.TrimSpace(request.GoalRole)
	if goalRole == "" {
		goalRole = "heading"
	}
	goalLabel := strings.TrimSpace(request.GoalLabel)
	if goalLabel == "" {
		goalLabel = "Dashboard"
	}
	request.GoalOrigin, request.GoalPath, request.GoalContext, request.GoalRole, request.GoalLabel = goalOrigin, goalPath, goalContext, goalRole, goalLabel
	startedAt := s.now().UTC()
	s.capture = &CaptureState{State: "launching", Message: "Launching an isolated headed Chromium authoring session.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: startedAt.Format(time.RFC3339)}
	s.captureResult = nil
	s.captureAttestation = nil
	request.Revision, request.CaptureRevision = "", ""
	s.captureStart = request
	session, err := s.startCapture(s.captureContext, browserauthor.Config{
		PrivateRoot: s.privateRoot, DriverDir: s.driverDir, InitialURL: request.URL, DashboardURL: request.DashboardURL,
		Goal: request.Goal, Origins: append([]string(nil), request.Origins...), ProfileID: request.ProfileID,
		GoalPredicate: authorresult.GoalPredicate{Origin: goalOrigin, Path: goalPath, Context: goalContext, Role: goalRole, Label: goalLabel},
	})
	if err != nil {
		s.capture = &CaptureState{State: "failed", Message: "The isolated Chromium worker could not start.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		_ = s.updateRevisionLocked()
		s.writeError(w, http.StatusUnprocessableEntity, "capture_failed", "browser capture failed before launch", false, requestID, s.revision)
		return
	}
	s.captureSession = session
	go s.consumeCapture(session, startedAt)
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/capture/start", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusAccepted, s.responseLocked(), requestID)
}

func (s *Server) consumeCapture(session CaptureSession, startedAt time.Time) {
	sawTerminal := false
	resultReceived := false
	terminalState := ""
	terminalMessage := ""
	terminalErrorCode := ""
	for event := range session.Events() {
		s.mu.Lock()
		if s.captureSession != session {
			s.mu.Unlock()
			continue
		}
		// Cancellation is monotonic. A result or checkpoint that was already
		// queued when the operator canceled must never restore an actionable
		// state or become promotable while process-tree teardown is pending.
		if s.capture != nil && s.capture.State == "canceling" && event.State != "failed" && event.State != "canceled" {
			s.mu.Unlock()
			continue
		}
		state := &CaptureState{
			State: event.State, Phase: event.Phase, Observation: event.Observation, Approval: event.Approval, Checkpoint: event.Checkpoint,
			StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
		}
		if event.ErrorCode != "" {
			state.Message = captureErrorMessage(event.ErrorCode)
		}
		if event.Result != nil {
			copy := *event.Result
			s.captureResult = &copy
			s.captureAttestation = event.Attestation
			resultReceived = true
			state.State = "completion_review"
			state.ResultReady = false
			state.Message = "The reduced profiles are complete; waiting for isolated worker teardown before stage review."
		}
		if event.State == "failed" || event.State == "canceled" {
			sawTerminal = true
			if event.State == "failed" || terminalState == "" {
				terminalState = event.State
				terminalMessage = state.Message
				terminalErrorCode = event.ErrorCode
			}
			s.capture = &CaptureState{
				State: "canceling", Message: "The browser worker stopped; waiting for process-tree teardown to complete.",
				StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
			}
			_ = s.updateRevisionLocked()
			s.mu.Unlock()
			continue
		}
		s.capture = state
		if event.Result != nil {
			sawTerminal = true
		}
		_ = s.updateRevisionLocked()
		s.mu.Unlock()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.captureSession != session {
		return
	}
	s.captureSession = nil
	if terminalState != "" || s.capture != nil && s.capture.State == "canceling" {
		state := "canceled"
		message := "Browser capture was canceled after the isolated worker and its descendants stopped; no capture bytes were staged."
		if terminalState == "canceled" && terminalMessage != "" {
			message = terminalMessage + " The isolated worker and its descendants have stopped."
		}
		if terminalState == "failed" {
			state = "failed"
			message = terminalMessage
			if message == "" {
				message = "The browser worker failed during teardown; no capture bytes were staged."
			}
			if terminalErrorCode == "worker_teardown" {
				s.captureContainmentFailed = true
				message += " Restart iCoT before starting another browser capture."
			}
		}
		s.capture = &CaptureState{State: state, Message: message, ContainmentFailed: terminalErrorCode == "worker_teardown", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		s.captureResult = nil
		s.captureAttestation = nil
		_ = s.updateRevisionLocked()
		return
	}
	if resultReceived && s.captureResult != nil {
		s.capture = &CaptureState{
			State: "stage_review", Message: "The reduced profiles are complete and the isolated worker has stopped. Review and stage them explicitly.", ResultReady: true,
			StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
		}
		_ = s.updateRevisionLocked()
		return
	}
	if !sawTerminal {
		s.capture = &CaptureState{State: "failed", Message: "The browser worker ended without a promotable result.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		_ = s.updateRevisionLocked()
	}
}

func (s *Server) serveCaptureRespond(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request captureRespondRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	if request.CaptureRevision != s.captureRevision || !captureResponsePending(s.capture) || s.captureSession == nil {
		s.writeError(w, http.StatusConflict, "stale_capture_revision", "capture revision is stale or no capture response is pending", true, requestID, s.revision)
		s.mu.Unlock()
		return
	}
	session := s.captureSession
	pending := *s.capture
	s.capture = &CaptureState{
		State: pending.State, Phase: pending.Phase, Message: "Response accepted; waiting for the isolated browser worker.",
		StartedAt: pending.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	}
	if err := s.updateRevisionLocked(); err != nil {
		s.capture = &pending
		s.writeInternalError(w, r, requestID, "/api/v4/capture/respond", "revision", err, true)
		s.mu.Unlock()
		return
	}
	reservedRevision := s.captureRevision
	s.mu.Unlock()
	responseCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := session.Respond(responseCtx, request.Response); err != nil {
		s.mu.Lock()
		if s.captureSession == session && s.captureRevision == reservedRevision {
			s.capture = &pending
			if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
				s.writeInternalError(w, r, requestID, "/api/v4/capture/respond", "revision", revisionErr, true)
				s.mu.Unlock()
				return
			}
		}
		currentRevision := s.revision
		s.mu.Unlock()
		s.writeError(w, http.StatusUnprocessableEntity, "capture_response_rejected", "the typed response is not valid for the current browser checkpoint", false, requestID, currentRevision)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusAccepted, s.responseLocked(), requestID)
}

func (s *Server) serveCaptureCancel(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request captureMutationRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.CaptureRevision != s.captureRevision || !captureActive(s.capture) || s.capture.State == "canceling" || s.captureSession == nil && s.captureCancel == nil {
		s.writeError(w, http.StatusConflict, "stale_capture_revision", "capture revision is stale or no capture is active", true, requestID, s.revision)
		return
	}
	if s.captureSession != nil {
		s.captureSession.Cancel()
	}
	if s.captureCancel != nil {
		s.captureCancel()
	}
	s.capture = &CaptureState{State: "canceling", Message: "Cancellation was requested; waiting for the isolated worker and all descendants to stop.", StartedAt: s.capture.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	s.captureResult = nil
	s.captureAttestation = nil
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/capture/cancel", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusAccepted, s.responseLocked(), requestID)
}

func (s *Server) serveCaptureStage(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request captureMutationRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.Revision != s.revision || request.CaptureRevision != s.captureRevision {
		s.writeError(w, http.StatusConflict, "stale_revision", "authoring or capture revision is stale", true, requestID, s.revision)
		return
	}
	if s.capture == nil || !s.capture.ResultReady || s.captureResult == nil || s.captureAttestation == nil || s.prepareCapture == nil {
		s.writeError(w, http.StatusConflict, "capture_not_ready", "a completed reviewed capture is required before staging", false, requestID, s.revision)
		return
	}
	stager, ok := s.engine.(browserCaptureEngine)
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "unsupported", "browser capture staging is unavailable", false, requestID, s.revision)
		return
	}
	startedAt := s.capture.StartedAt
	s.capture = &CaptureState{State: "staging", Message: "Revalidating and atomically staging the reduced profile pair.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	prepared, err := s.prepareCapture(CaptureStageRequest{Start: s.captureStart, ExampleDir: s.exampleDir, PrivateRoot: s.privateRoot, Result: *s.captureResult, Attestation: s.captureAttestation})
	if err != nil {
		s.capture = &CaptureState{State: "failed", Message: "The completed capture failed independent OpenUdon validation; nothing was staged.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		s.captureResult = nil
		s.captureAttestation = nil
		_ = s.updateRevisionLocked()
		s.writeError(w, http.StatusUnprocessableEntity, "capture_validation_failed", "completed browser capture was rejected", false, requestID, s.revision)
		return
	}
	snapshot, err := stager.StageBrowserCapture(r.Context(), prepared)
	if err != nil {
		s.capture = &CaptureState{State: "failed", Message: "The reviewed capture could not be staged; no partial capture was adopted.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		s.captureResult = nil
		s.captureAttestation = nil
		s.captureSession = nil
		s.refreshWorkspaceAfterFailure()
		_ = s.updateRevisionLocked()
		s.writeEngineError(w, r, requestID, "/api/v4/capture/stage", "stage_capture", err)
		return
	}
	s.snapshot = snapshot
	s.capture = &CaptureState{State: "staged", Message: "The canonical profile pair and safe capture review were staged. Continue normal authoring review.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	s.captureResult = nil
	s.captureAttestation = nil
	s.captureSession = nil
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/capture/stage", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func captureErrorMessage(code string) string {
	switch code {
	case "operator_idle_timeout":
		return "Browser capture was canceled after 30 minutes without an operator response."
	case "absolute_timeout":
		return "Browser capture reached its two-hour absolute ceiling."
	case "worker_teardown":
		return "The browser process tree did not confirm teardown within the containment deadline."
	default:
		return "The browser worker failed closed; no capture bytes were staged."
	}
}

func (s *Server) serveRound(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request roundRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	if strings.TrimSpace(request.Revision) == "" {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "revision is required", false, requestID, s.currentRevision())
		return
	}
	if request.Answers == nil {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "answers is required", false, requestID, s.currentRevision())
		return
	}
	answers := make([]authoring.RoundAnswer, len(request.Answers))
	for i, answer := range request.Answers {
		questionID := strings.TrimSpace(answer.QuestionID)
		if questionID == "" {
			s.writeError(w, http.StatusBadRequest, "malformed_request", "every answer requires question_id", false, requestID, s.currentRevision())
			return
		}
		value := answer.Value
		if answer.Deferral != nil {
			var err error
			value, err = encodeRoundDeferral(answer.Value, *answer.Deferral)
			if err != nil {
				s.writeQuestionError(w, http.StatusBadRequest, "malformed_request", err.Error(), false, requestID, s.currentRevision(), questionID)
				return
			}
		}
		answers[i] = authoring.RoundAnswer{QuestionID: questionID, Value: value, Source: humanInputSource}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, "/api/v4/round", request.Revision) {
		return
	}
	snapshot, err := s.engine.ApplyRound(r.Context(), answers)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v4/round", "apply_round", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/round", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func encodeRoundDeferral(answerValue string, deferral roundDeferral) (string, error) {
	if strings.TrimSpace(answerValue) != "" {
		return "", errors.New("an answer must contain either value or deferral, not both")
	}
	parts := []string{deferral.Owner, deferral.Impact, deferral.UnblockCondition, deferral.SuggestedNextAction}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", errors.New("a deferral requires owner, impact, unblock condition, and suggested next action")
		}
		if strings.Contains(part, "|") {
			return "", errors.New("deferral fields may not contain the | character")
		}
	}
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return "defer:" + strings.Join(parts, " | "), nil
}

func (s *Server) serveReopen(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request reopenRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	request.Revision = strings.TrimSpace(request.Revision)
	request.QuestionID = strings.TrimSpace(request.QuestionID)
	if request.Revision == "" {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "revision is required", false, requestID, s.currentRevision())
		return
	}
	if request.QuestionID == "" {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "question_id is required", false, requestID, s.currentRevision())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, "/api/v4/reopen", request.Revision) {
		return
	}
	snapshot, err := s.engine.ReopenDecision(r.Context(), request.QuestionID)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v4/reopen", "reopen_decision", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/reopen", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveApprove(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request approveRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	if strings.TrimSpace(request.Revision) == "" {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "revision is required", false, requestID, s.currentRevision())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, "/api/v4/author/approve", request.Revision) {
		return
	}
	result, err := s.engine.ApproveAndWrite(r.Context(), engine.Approval{
		HumanApproved: request.HumanApproved, AllowOverwrite: request.AllowOverwrite, ApproveIncomplete: request.ApproveIncomplete,
	})
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v4/author/approve", "approve", err)
		return
	}
	s.lifecycle = lifecycleAuthored
	s.completed = false
	s.snapshot = result.Snapshot
	s.writeResult = &result.WriteResult
	s.packageState = nil
	s.artifactPaths = map[string]string{}
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/author/approve", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveResume(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request revisionRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(r.Context()); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/author/resume", "workspace_inspection", err, true)
		return
	}
	if strings.TrimSpace(request.Revision) != s.revision {
		s.writeError(w, http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, requestID, s.revision)
		return
	}
	if s.lifecycle != lifecyclePackageFail {
		s.writeError(w, http.StatusConflict, "invalid_lifecycle", "authoring can resume only after a package quality failure", false, requestID, s.revision)
		return
	}
	resumer, ok := s.engine.(resumeEngine)
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "unsupported", "authoring resume is unavailable", false, requestID, s.revision)
		return
	}
	snapshot, err := resumer.ResumeAuthoring(r.Context())
	if err != nil {
		s.writeEngineError(w, r, requestID, "/api/v4/author/resume", "resume_authoring", err)
		return
	}
	s.snapshot = snapshot
	s.lifecycle = lifecycleAuthoring
	s.completed = false
	s.writeResult = nil
	s.packageState = nil
	s.artifactPaths = map[string]string{}
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/author/resume", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) servePackageBuild(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request buildRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(r.Context()); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "workspace_inspection", err, true)
		return
	}
	if strings.TrimSpace(request.Revision) != s.revision {
		s.writeError(w, http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, requestID, s.revision)
		return
	}
	if captureActive(s.capture) {
		s.writeError(w, http.StatusConflict, "capture_active", "package build is blocked while browser capture is active", true, requestID, s.revision)
		return
	}
	if s.captureContainmentFailed {
		s.writeError(w, http.StatusConflict, "capture_teardown_failed", "package build is blocked because browser process-tree teardown was not confirmed; restart iCoT", false, requestID, s.revision)
		return
	}
	if s.lifecycle != lifecycleAuthored {
		s.writeError(w, http.StatusConflict, "invalid_lifecycle", "a separately approved authored state is required before package build", false, requestID, s.revision)
		return
	}
	if !request.Confirmed {
		s.writeError(w, http.StatusUnprocessableEntity, "confirmation_required", "explicit package-build confirmation is required", false, requestID, s.revision)
		return
	}
	buildCtx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	result, _, err := s.buildPackage(buildCtx, synthesize.Options{ExampleDir: s.exampleDir, LocalOnlyDiscovery: true})
	if err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "deterministic_build", err, true)
		return
	}
	var report *synthesize.QualityReport
	var assessmentErr error
	inspection, inspectionErr := s.inspectPackage(buildCtx, trustedrunner.TemplateOptions{
		RepoRoot: s.repoRoot, ExampleDir: s.exampleDir,
		Assess: func(ctx context.Context, opts synthesize.Options) (*synthesize.QualityReport, error) {
			opts.LocalOnlyDiscovery = true
			assessed, assessErr := s.assessPackage(ctx, opts)
			assessmentErr = assessErr
			if assessed != nil {
				copy := *assessed
				copy.Checks = append([]synthesize.QualityCheck(nil), assessed.Checks...)
				report = &copy
			}
			return assessed, assessErr
		},
	})
	if assessmentErr != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "current_state_assessment", assessmentErr, true)
		return
	}
	if report == nil {
		if inspectionErr == nil {
			inspectionErr = errors.New("assessment returned no quality report")
		}
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "current_state_assessment", inspectionErr, true)
		return
	}
	quality := &PackageQuality{Status: report.Status, Checks: append([]synthesize.QualityCheck(nil), report.Checks...)}
	if !report.Passed() {
		s.lifecycle = lifecyclePackageFail
		s.completed = false
		s.packageState = &PackageState{Status: "failed", Quality: quality, Remediation: packageRemediation(report)}
		s.artifactPaths = map[string]string{}
		if err := s.updateRevisionLocked(); err != nil {
			s.writeInternalError(w, r, requestID, "/api/v4/package/build", "revision", err, true)
			return
		}
		setETag(w, s.etag)
		s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
		return
	}
	if inspectionErr != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "package_inspection", inspectionErr, true)
		return
	}
	artifacts, paths, err := inspectAllowedArtifacts(s.exampleDir, result)
	if err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "artifact_allowlist", err, true)
		return
	}
	if err := s.revalidatePackage(buildCtx, trustedrunner.TemplateOptions{RepoRoot: s.repoRoot, ExampleDir: s.exampleDir}, inspection); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "package_freeze_revalidation", err, true)
		return
	}
	s.lifecycle = lifecycleHandoffReady
	s.completed = true
	s.artifactPaths = paths
	s.packageState = &PackageState{
		Status: "pass", Quality: quality, Inspection: &inspection, Artifacts: artifacts,
		ApprovalTemplateArgv: []string{"openudon", "approval-template", "--example", s.exampleDir, "--state", trustedrunner.StateApprovedForSandbox, "--reviewer", "REVIEWER"},
	}
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/package/build", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) serveArtifact(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w, http.MethodGet, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	if len(r.URL.Query()) != 1 || len(r.URL.Query()["name"]) != 1 {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "artifact inspection requires exactly one name", false, requestID, s.currentRevision())
		return
	}
	name := r.URL.Query().Get("name")
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lifecycle == lifecycleHandoffReady {
		if err := s.validateFrozenArtifactsLocked(r.Context()); err != nil {
			s.invalidateHandoffLocked()
			if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
				s.writeInternalError(w, r, requestID, "/api/v4/artifact", "revision", revisionErr, true)
				return
			}
			s.writeError(w, http.StatusConflict, "package_changed", "the reviewed package changed after handoff inspection; return to authoring and rebuild", false, requestID, s.revision)
			return
		}
	}
	if s.lifecycle != lifecycleHandoffReady {
		s.writeError(w, http.StatusConflict, "invalid_lifecycle", "artifacts are available only after a passing package build", false, requestID, s.revision)
		return
	}
	path, ok := s.artifactPaths[name]
	if !ok {
		s.writeError(w, http.StatusNotFound, "not_found", "artifact is not in the inspection allowlist", false, requestID, s.revision)
		return
	}
	data, err := readAllowedArtifact(s.exampleDir, path)
	if err != nil {
		s.invalidateHandoffLocked()
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			s.writeInternalError(w, r, requestID, "/api/v4/artifact", "revision", revisionErr, true)
			return
		}
		s.writeError(w, http.StatusConflict, "package_changed", "the reviewed package changed after handoff inspection; return to authoring and rebuild", false, requestID, s.revision)
		return
	}
	summary, ok := s.artifactSummaryLocked(name)
	digest := sha256.Sum256(data)
	if !ok || summary.Bytes != int64(len(data)) || summary.SHA256 != "sha256:"+hex.EncodeToString(digest[:]) {
		s.invalidateHandoffLocked()
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			s.writeInternalError(w, r, requestID, "/api/v4/artifact", "revision", revisionErr, true)
			return
		}
		s.writeError(w, http.StatusConflict, "package_changed", "the reviewed package changed after handoff inspection; return to authoring and rebuild", false, requestID, s.revision)
		return
	}
	w.Header().Set("Content-Type", artifactMediaType(path))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) beginMutation(w http.ResponseWriter, r *http.Request, requestID, route, requestRevision string) bool {
	if err := s.refreshWorkspaceLocked(r.Context()); err != nil {
		s.writeInternalError(w, r, requestID, route, "workspace_inspection", err, true)
		return false
	}
	if s.workspace.ExternallyModified {
		s.writeError(w, http.StatusConflict, "workspace_changed", "the authoring workspace changed outside this process; restart is required", false, requestID, s.revision)
		return false
	}
	if s.captureContainmentFailed {
		s.writeError(w, http.StatusConflict, "capture_teardown_failed", "browser process-tree teardown was not confirmed; restart iCoT before authoring continues", false, requestID, s.revision)
		return false
	}
	if captureActive(s.capture) {
		s.writeError(w, http.StatusConflict, "capture_active", "authoring mutations are blocked while browser capture is active", true, requestID, s.revision)
		return false
	}
	if s.lifecycle != lifecycleAuthoring {
		s.writeError(w, http.StatusConflict, "session_frozen", "authoring is not mutable in the current lifecycle state", false, requestID, s.revision)
		return false
	}
	if requestRevision != s.revision {
		s.writeError(w, http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, requestID, s.revision)
		return false
	}
	return true
}

func (s *Server) refreshWorkspaceLocked(ctx context.Context) error {
	status, err := s.engine.WorkspaceStatus(ctx)
	if err != nil {
		return err
	}
	if status != s.workspace {
		s.workspace = status
		return s.updateRevisionLocked()
	}
	return nil
}

func (s *Server) refreshWorkspaceAfterFailure() {
	status, err := s.engine.WorkspaceStatus(context.Background())
	if err != nil {
		return
	}
	if status != s.workspace {
		s.workspace = status
		_ = s.updateRevisionLocked()
	}
}

func (s *Server) updateRevisionLocked() error {
	revision, err := revisionDigest(struct {
		Snapshot           engine.Snapshot             `json:"snapshot"`
		Lifecycle          string                      `json:"lifecycle"`
		WriteResult        *engine.WriteResult         `json:"write_result,omitempty"`
		Package            *PackageState               `json:"package,omitempty"`
		BrowserTransaction *BrowserTransactionSnapshot `json:"browser_transaction,omitempty"`
		Workspace          engine.WorkspaceStatus      `json:"workspace"`
	}{Snapshot: s.snapshot, Lifecycle: s.lifecycle, WriteResult: s.writeResult, Package: s.packageState, BrowserTransaction: s.browserTransaction, Workspace: s.workspace})
	if err != nil {
		return err
	}
	s.revision = revision
	captureRevision, err := revisionDigest(struct {
		Capture *CaptureState                 `json:"capture,omitempty"`
		Doctor  *browserauthor.UIDoctorReport `json:"doctor,omitempty"`
	}{Capture: s.capture, Doctor: s.doctorReport})
	if err != nil {
		return err
	}
	s.captureRevision = captureRevision
	s.etag, err = revisionDigest(struct {
		Authoring string `json:"authoring"`
		Capture   string `json:"capture"`
	}{Authoring: s.revision, Capture: s.captureRevision})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) responseLocked() Response {
	return Response{
		Version: APIVersion, Revision: s.revision, CaptureRevision: s.captureRevision,
		Lifecycle: s.lifecycle, Completed: s.completed,
		Workspace: Workspace{ExampleDir: s.exampleDir, ExternallyModified: s.workspace.ExternallyModified},
		Snapshot:  s.snapshot, Capture: s.capture, BrowserDoctor: s.doctorReport, WriteResult: s.writeResult, Package: s.packageState,
		BrowserTransaction: s.browserTransactionResourceLocked(),
	}
}

func (s *Server) currentRevision() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision
}

func (s *Server) authenticated(r *http.Request, allowCookie bool) bool {
	values := r.Header.Values("Authorization")
	if len(values) == 1 {
		parts := strings.Split(values[0], " ")
		if len(parts) == 2 && parts[0] == "Bearer" && secureEqual(parts[1], s.token) {
			return true
		}
	}
	if !allowCookie {
		return false
	}
	cookie, err := r.Cookie(SessionCookie)
	return err == nil && secureEqual(cookie.Value, s.token)
}

func (s *Server) writeEngineError(w http.ResponseWriter, r *http.Request, requestID, route, stage string, err error) {
	class, _ := engine.FailureDetails(err)
	switch class {
	case engine.FailureRejected:
		s.writeQuestionError(w, http.StatusUnprocessableEntity, "engine_rejected", safeMessage(err), false, requestID, s.revision, engine.FailureQuestionID(err))
	case engine.FailureConflict:
		s.writeError(w, http.StatusConflict, "workspace_changed", "the authoring workspace changed outside this process; restart is required", false, requestID, s.revision)
	case engine.FailureIndeterminate:
		s.writeInternalError(w, r, requestID, route, stage, err, false)
	default:
		s.writeInternalError(w, r, requestID, route, stage, err, true)
	}
}

func secureEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func digestRevision(payload any) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal UI revision state: %w", err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func captureActive(capture *CaptureState) bool {
	if capture == nil {
		return false
	}
	switch capture.State {
	case "staged", "canceled", "failed":
		return false
	default:
		return true
	}
}

func captureResponsePending(capture *CaptureState) bool {
	return captureActive(capture) && (capture.Observation != nil || capture.Approval != nil || capture.Checkpoint != nil)
}

func packageRemediation(report *synthesize.QualityReport) []string {
	if report == nil {
		return []string{"Reopen authoring and rebuild the reviewed package."}
	}
	var result []string
	for _, check := range report.Checks {
		if strings.EqualFold(check.Status, "pass") {
			continue
		}
		message := strings.TrimSpace(check.Message)
		if detail := strings.TrimSpace(check.Detail); detail != "" {
			message = strings.TrimSpace(message + ": " + detail)
		}
		result = append(result, strings.TrimSpace(check.Code+": "+safeMessage(errors.New(message))))
	}
	if len(result) == 0 {
		result = append(result, "Reopen authoring, review the quality report, and rebuild the package.")
	}
	return result
}

func inspectAllowedArtifacts(exampleDir string, result *synthesize.Result) ([]ArtifactSummary, map[string]string, error) {
	if result == nil {
		return nil, nil, errors.New("package build returned no artifact paths")
	}
	candidates := []struct {
		name string
		path string
	}{
		{"project", result.ProjectPath}, {"intent", result.IntentPath}, {"workflow", result.WorkflowPath},
		{"uws", result.UWSPath}, {"plan_json", result.PlanJSONPath}, {"plan_md", result.PlanMDPath},
		{"review", result.ReviewPath}, {"review_handoff", result.ReviewHandoffPath},
		{"quality_json", result.QualityJSONPath}, {"quality_md", result.QualityMDPath},
	}
	root, err := filepath.Abs(exampleDir)
	if err != nil {
		return nil, nil, err
	}
	paths := make(map[string]string, len(candidates))
	artifacts := make([]ArtifactSummary, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.path) == "" {
			continue
		}
		absolute, err := filepath.Abs(candidate.path)
		if err != nil {
			return nil, nil, err
		}
		relative, err := filepath.Rel(root, absolute)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return nil, nil, fmt.Errorf("package artifact %s is outside the example", candidate.name)
		}
		data, err := readAllowedArtifact(root, absolute)
		if err != nil {
			return nil, nil, fmt.Errorf("inspect package artifact %s: %w", candidate.name, err)
		}
		digest := sha256.Sum256(data)
		paths[candidate.name] = absolute
		artifacts = append(artifacts, ArtifactSummary{
			Name: candidate.name, Path: filepath.ToSlash(relative), MediaType: artifactMediaType(absolute),
			Bytes: int64(len(data)), SHA256: "sha256:" + hex.EncodeToString(digest[:]),
		})
	}
	return artifacts, paths, nil
}

func (s *Server) validateFrozenArtifactsLocked(ctx context.Context) error {
	if s.packageState == nil || s.packageState.Status != "pass" || s.packageState.Inspection == nil || len(s.packageState.Artifacts) == 0 || len(s.artifactPaths) != len(s.packageState.Artifacts) {
		return errors.New("handoff artifact inventory is incomplete")
	}
	if err := s.revalidatePackage(ctx, trustedrunner.TemplateOptions{
		RepoRoot: s.repoRoot, ExampleDir: s.exampleDir,
	}, *s.packageState.Inspection); err != nil {
		return fmt.Errorf("revalidate complete handoff package: %w", err)
	}
	for _, summary := range s.packageState.Artifacts {
		path, ok := s.artifactPaths[summary.Name]
		if !ok {
			return errors.New("handoff artifact path is missing")
		}
		data, err := readAllowedArtifact(s.exampleDir, path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if summary.Bytes != int64(len(data)) || summary.SHA256 != "sha256:"+hex.EncodeToString(digest[:]) {
			return errors.New("handoff artifact digest changed")
		}
	}
	return nil
}

func (s *Server) artifactSummaryLocked(name string) (ArtifactSummary, bool) {
	if s.packageState == nil {
		return ArtifactSummary{}, false
	}
	for _, summary := range s.packageState.Artifacts {
		if summary.Name == name {
			return summary, true
		}
	}
	return ArtifactSummary{}, false
}

func (s *Server) invalidateHandoffLocked() {
	var quality *PackageQuality
	if s.packageState != nil {
		quality = s.packageState.Quality
	}
	s.lifecycle = lifecyclePackageFail
	s.completed = false
	s.packageState = &PackageState{
		Status: "failed", Quality: quality,
		Remediation: []string{"A reviewed package artifact changed after handoff inspection. Reopen authoring, repeat approval, and rebuild the package."},
	}
	s.artifactPaths = map[string]string{}
}

func readAllowedArtifact(exampleDir, path string) ([]byte, error) {
	root, err := filepath.Abs(exampleDir)
	if err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return nil, errors.New("artifact path is outside the example")
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("artifact is not a regular non-symlink file")
	}
	if info.Size() > MaxArtifactBytes {
		return nil, fmt.Errorf("artifact exceeds the %d-byte inspection limit", MaxArtifactBytes)
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(absolute)
	if err != nil || after.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, after) || after.Size() != int64(len(data)) {
		return nil, errors.New("artifact changed while it was inspected")
	}
	return data, nil
}

func artifactMediaType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "application/json; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return &requestError{status: http.StatusUnsupportedMediaType, code: "unsupported_media_type", text: "Content-Type must be application/json with no charset or UTF-8"}
	}
	for name, value := range params {
		if !strings.EqualFold(name, "charset") || !strings.EqualFold(value, "utf-8") {
			return &requestError{status: http.StatusUnsupportedMediaType, code: "unsupported_media_type", text: "Content-Type must be application/json with no charset or UTF-8"}
		}
	}
	if r.ContentLength > MaxRequestBytes {
		return &requestError{status: http.StatusRequestEntityTooLarge, code: "payload_too_large", text: fmt.Sprintf("request body exceeds %d bytes", MaxRequestBytes)}
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return &requestError{status: http.StatusRequestEntityTooLarge, code: "payload_too_large", text: fmt.Sprintf("request body exceeds %d bytes", MaxRequestBytes)}
		}
		return &requestError{status: http.StatusBadRequest, code: "malformed_request", text: "could not read JSON request body"}
	}
	if !utf8.Valid(data) {
		return &requestError{status: http.StatusBadRequest, code: "malformed_request", text: "JSON request body is not valid UTF-8"}
	}
	if err := rejectDuplicateJSONNames(data); err != nil {
		return &requestError{status: http.StatusBadRequest, code: "malformed_request", text: safeMessage(err)}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &requestError{status: http.StatusBadRequest, code: "malformed_request", text: safeMessage(fmt.Errorf("decode JSON request: %w", err))}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return &requestError{status: http.StatusBadRequest, code: "malformed_request", text: "request body must contain exactly one JSON document"}
	}
	return nil
}

func rejectDuplicateJSONNames(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON document")
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, depth int) error {
	if depth > MaxJSONDepth {
		return fmt.Errorf("JSON nesting exceeds %d levels", MaxJSONDepth)
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := nameToken.(string)
			if !ok {
				return errors.New("JSON object member name is not a string")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON object name %q", name)
			}
			seen[name] = true
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("JSON array is not closed")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

func (s *Server) methodNotAllowed(w http.ResponseWriter, allowed, requestID string) {
	w.Header().Set("Allow", allowed)
	s.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "request method is not supported for this route", false, requestID, "")
}

func (s *Server) writeRequestError(w http.ResponseWriter, err error, requestID string) {
	var requestErr *requestError
	if !errors.As(err, &requestErr) {
		requestErr = &requestError{status: http.StatusBadRequest, code: "malformed_request", text: safeMessage(err)}
	}
	s.writeError(w, requestErr.status, requestErr.code, requestErr.text, false, requestID, s.currentRevision())
}

func (s *Server) writeInternalError(w http.ResponseWriter, _ *http.Request, requestID, route, stage string, cause error, retryable bool) {
	fmt.Fprintf(s.errOut, "icot ui: request_id=%s route=%s stage=%s cause=%s\n", requestID, route, stage, sanitizeLogCause(cause))
	s.writeError(w, http.StatusInternalServerError, "internal_error", "authoring engine operation failed", retryable, requestID, s.revision)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string, retryable bool, requestID, revision string) {
	s.writeQuestionError(w, status, code, message, retryable, requestID, revision, "")
}

func (s *Server) writeQuestionError(w http.ResponseWriter, status int, code, message string, retryable bool, requestID, revision, questionID string) {
	s.writeJSON(w, status, errorEnvelope{Version: APIVersion, Revision: revision, Error: errorPayload{
		Code: code, Message: message, Retryable: retryable, RequestID: requestID, QuestionID: questionID,
	}}, requestID)
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, value any, requestID string) {
	data, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		data, _ = json.Marshal(errorEnvelope{Version: APIVersion, Error: errorPayload{
			Code: "internal_error", Message: "failed to encode response", Retryable: true, RequestID: requestID,
		}})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}

func safeMessage(err error) string {
	if err == nil {
		return "request rejected"
	}
	message := strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\t' || r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, err.Error())
	message = strings.TrimSpace(message)
	if len(message) > 1024 {
		message = message[:1024]
	}
	if message == "" {
		return "request rejected"
	}
	return message
}

func sanitizeLogCause(err error) string {
	message := safeMessage(err)
	lower := strings.ToLower(message)
	for _, marker := range []string{"token=", "authorization", "cookie", "bearer "} {
		if strings.Contains(lower, marker) {
			return "redacted operational failure"
		}
	}
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}

func newRequestID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err == nil {
		return hex.EncodeToString(raw)
	}
	value := fallbackRequestID.Add(1)
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%d", time.Now().UnixNano(), value)))
	return hex.EncodeToString(digest[:16])
}

func setETag(w http.ResponseWriter, revision string) {
	w.Header().Set("ETag", strconv.Quote(revision))
}

func matchesETag(header, revision string) bool {
	for _, value := range strings.Split(header, ",") {
		value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "W/"))
		if value == "*" {
			return true
		}
		if value == revision || value == strconv.Quote(revision) {
			return true
		}
	}
	return false
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; object-src 'none'")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}
