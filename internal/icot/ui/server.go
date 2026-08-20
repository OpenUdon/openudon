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
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/engine"
)

const (
	APIVersion       = "openudon.icot-ui-api.v2"
	SessionCookie    = "openudon_icot_ui"
	MaxRequestBytes  = 1 << 20
	humanInputSource = "user"
	instancePrefix   = "/.icot-ui/"
)

// AuthoringEngine is the engine contract used by the local transport.
type AuthoringEngine interface {
	ApplyRound(context.Context, []authoring.RoundAnswer) (engine.Snapshot, error)
	ApproveAndWrite(context.Context, engine.Approval) (engine.ApprovalResult, error)
	WorkspaceStatus(context.Context) (engine.WorkspaceStatus, error)
}

// HandlerConfig configures one server handler after its loopback listener is
// active. Authority must be the listener's exact host:port value.
type HandlerConfig struct {
	Engine     AuthoringEngine
	Snapshot   engine.Snapshot
	ExampleDir string
	Token      string
	AccessCode string
	Authority  string
	ErrOut     io.Writer
	Now        func() time.Time
}

// Workspace identifies the selected example and its optimistic ownership
// status.
type Workspace struct {
	ExampleDir         string `json:"example_dir"`
	ExternallyModified bool   `json:"externally_modified"`
}

// Response is returned by every successful API request.
type Response struct {
	Version     string              `json:"version"`
	Revision    string              `json:"revision"`
	Completed   bool                `json:"completed"`
	Workspace   Workspace           `json:"workspace"`
	Snapshot    engine.Snapshot     `json:"snapshot"`
	WriteResult *engine.WriteResult `json:"write_result,omitempty"`
}

type errorEnvelope struct {
	Version  string       `json:"version"`
	Revision string       `json:"revision,omitempty"`
	Error    errorPayload `json:"error"`
}

type errorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	RequestID string `json:"request_id"`
}

type roundRequest struct {
	Revision string        `json:"revision"`
	Answers  []roundAnswer `json:"answers"`
}

type roundAnswer struct {
	QuestionID string `json:"question_id"`
	Value      string `json:"value"`
}

type approveRequest struct {
	Revision          string `json:"revision"`
	HumanApproved     bool   `json:"human_approved"`
	AllowOverwrite    bool   `json:"allow_overwrite"`
	ApproveIncomplete bool   `json:"approve_incomplete"`
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

	engine            AuthoringEngine
	snapshot          engine.Snapshot
	exampleDir        string
	token             string
	accessCodeDigest  [sha256.Size]byte
	accessCodeExpires time.Time
	accessCodeUsed    bool
	accessFailures    []time.Time
	now               func() time.Time
	authority         string
	origin            string
	basePath          string
	revision          string
	completed         bool
	writeResult       *engine.WriteResult
	workspace         engine.WorkspaceStatus
	errOut            io.Writer
}

var fallbackRequestID atomic.Uint64

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
	if len(accessCode) != 12 {
		return nil, errors.New("UI access code must contain 12 Crockford Base32 characters")
	}
	for _, ch := range accessCode {
		if !strings.ContainsRune("0123456789ABCDEFGHJKMNPQRSTVWXYZ", ch) {
			return nil, errors.New("UI access code must contain 12 Crockford Base32 characters")
		}
	}
	authority := strings.TrimSpace(config.Authority)
	if !validLoopbackAuthority(authority) {
		return nil, errors.New("UI authority must be an active 127.0.0.1 listener")
	}
	errOut := config.ErrOut
	if errOut == nil {
		errOut = io.Discard
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}
	s := &Server{
		engine: config.Engine, snapshot: config.Snapshot,
		exampleDir: config.ExampleDir, token: config.Token, authority: authority,
		origin: "http://" + authority, basePath: instanceBasePath(config.Token), errOut: errOut,
		accessCodeDigest: sha256.Sum256([]byte(accessCode)), accessCodeExpires: now().Add(5 * time.Minute), now: now,
	}
	revision, err := revisionFor(s.snapshot, false, nil, s.workspace)
	if err != nil {
		return nil, err
	}
	s.revision = revision
	return s, nil
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
	case "/api/v2/snapshot":
		s.serveSnapshot(w, r, cookieScoped, requestID)
	case "/api/v2/round":
		s.serveRound(w, r, cookieScoped, requestID)
	case "/api/v2/approve":
		s.serveApprove(w, r, cookieScoped, requestID)
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
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>OpenUdon iCoT</title></head><body><main><h1>OpenUdon iCoT</h1><p>Enter the 12-character access code shown in the terminal.</p><form method="post" action="/"><label>Access code <input name="code" inputmode="text" autocomplete="one-time-code" maxlength="12" required></label><button type="submit">Continue</button></form></main></body></html>`)
	case http.MethodPost:
		s.exchangeAccessCode(w, r, requestID)
	default:
		s.methodNotAllowed(w, "GET, POST", requestID)
	}
}

func (s *Server) exchangeAccessCode(w http.ResponseWriter, r *http.Request, requestID string) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "access-code form is invalid", false, requestID, "")
		return
	}
	if len(r.Form) != 1 || len(r.Form["code"]) != 1 {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "exactly one access code is required", false, requestID, "")
		return
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-time.Minute)
	kept := s.accessFailures[:0]
	for _, failedAt := range s.accessFailures {
		if failedAt.After(cutoff) {
			kept = append(kept, failedAt)
		}
	}
	s.accessFailures = kept
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
		s.writeInternalError(w, r, requestID, "/api/v2/snapshot", "workspace_inspection", err, true)
		return
	}
	setETag(w, s.revision)
	if matchesETag(r.Header.Get("If-None-Match"), s.revision) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
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
		if strings.TrimSpace(answer.QuestionID) == "" {
			s.writeError(w, http.StatusBadRequest, "malformed_request", "every answer requires question_id", false, requestID, s.currentRevision())
			return
		}
		answers[i] = authoring.RoundAnswer{QuestionID: answer.QuestionID, Value: answer.Value, Source: humanInputSource}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.beginMutation(w, r, requestID, "/api/v2/round", request.Revision) {
		return
	}
	snapshot, err := s.engine.ApplyRound(r.Context(), answers)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v2/round", "apply_round", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v2/round", "revision", err, true)
		return
	}
	setETag(w, s.revision)
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
	if !s.beginMutation(w, r, requestID, "/api/v2/approve", request.Revision) {
		return
	}
	result, err := s.engine.ApproveAndWrite(r.Context(), engine.Approval{
		HumanApproved: request.HumanApproved, AllowOverwrite: request.AllowOverwrite, ApproveIncomplete: request.ApproveIncomplete,
	})
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		s.writeEngineError(w, r, requestID, "/api/v2/approve", "approve", err)
		return
	}
	s.completed = true
	s.snapshot = result.Snapshot
	s.writeResult = &result.WriteResult
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v2/approve", "revision", err, true)
		return
	}
	setETag(w, s.revision)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
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
	if requestRevision != s.revision {
		s.writeError(w, http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, requestID, s.revision)
		return false
	}
	if s.completed {
		s.writeError(w, http.StatusConflict, "session_frozen", "the approved authoring session is frozen", false, requestID, s.revision)
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
	revision, err := revisionFor(s.snapshot, s.completed, s.writeResult, s.workspace)
	if err != nil {
		return err
	}
	s.revision = revision
	return nil
}

func (s *Server) responseLocked() Response {
	return Response{
		Version: APIVersion, Revision: s.revision, Completed: s.completed,
		Workspace: Workspace{ExampleDir: s.exampleDir, ExternallyModified: s.workspace.ExternallyModified},
		Snapshot:  s.snapshot, WriteResult: s.writeResult,
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
		s.writeError(w, http.StatusUnprocessableEntity, "engine_rejected", safeMessage(err), false, requestID, s.revision)
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

func revisionFor(snapshot engine.Snapshot, completed bool, result *engine.WriteResult, status engine.WorkspaceStatus) (string, error) {
	payload := struct {
		Snapshot    engine.Snapshot        `json:"snapshot"`
		Completed   bool                   `json:"completed"`
		WriteResult *engine.WriteResult    `json:"write_result,omitempty"`
		Workspace   engine.WorkspaceStatus `json:"workspace"`
	}{Snapshot: snapshot, Completed: completed, WriteResult: result, Workspace: status}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal UI revision state: %w", err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
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
	if err := scanJSONValue(decoder); err != nil {
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

func scanJSONValue(decoder *json.Decoder) error {
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
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
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
	s.writeJSON(w, status, errorEnvelope{Version: APIVersion, Revision: revision, Error: errorPayload{
		Code: code, Message: message, Retryable: retryable, RequestID: requestID,
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
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}
