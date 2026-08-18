// Package ui provides the loopback-only HTTP transport for one iCoT engine.
// The API is experimental and intentionally internal to OpenUdon.
package ui

import (
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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/engine"
)

const (
	APIVersion       = "openudon.icot-ui-api.v1"
	SessionCookie    = "openudon_icot_ui"
	MaxRequestBytes  = 1 << 20
	humanInputSource = "user"
	stateSyncTimeout = 2 * time.Second
	instancePrefix   = "/.icot-ui/"
)

// AuthoringEngine is the subset of the A07 engine used by the HTTP transport.
type AuthoringEngine interface {
	ApplyRound(context.Context, []authoring.RoundAnswer) (engine.Snapshot, error)
	ApproveAndWrite(context.Context, engine.Approval) (engine.WriteResult, error)
	Snapshot(context.Context) (engine.Snapshot, error)
}

// HandlerConfig configures one server handler after its loopback listener is
// active. Authority must be the listener's exact host:port value.
type HandlerConfig struct {
	Engine     AuthoringEngine
	Snapshot   engine.Snapshot
	ExampleDir string
	Token      string
	Authority  string
}

// Workspace identifies the single explicitly selected example.
type Workspace struct {
	ExampleDir string `json:"example_dir"`
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
	Version string       `json:"version"`
	Error   errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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

// Server serializes revision checks and all A07 engine mutations.
type Server struct {
	mu sync.Mutex

	engine       AuthoringEngine
	snapshot     engine.Snapshot
	exampleDir   string
	token        string
	authority    string
	origin       string
	basePath     string
	revision     string
	completed    bool
	writeResult  *engine.WriteResult
	synchronized bool
}

// GenerateToken returns a 256-bit URL-safe per-process capability token.
func GenerateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate UI capability token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
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
	authority := strings.TrimSpace(config.Authority)
	if !validLoopbackAuthority(authority) {
		return nil, errors.New("UI authority must be an active 127.0.0.1 listener")
	}
	s := &Server{
		engine: config.Engine, snapshot: config.Snapshot,
		exampleDir: config.ExampleDir, token: config.Token, authority: authority,
		origin: "http://" + authority, basePath: instanceBasePath(config.Token), synchronized: true,
	}
	revision, err := revisionFor(s.snapshot, false, nil)
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
	digest := sha256.Sum256([]byte("openudon.icot-ui.instance-path.v1\x00" + token))
	return instancePrefix + hex.EncodeToString(digest[:]) + "/"
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)
	if r.Host != s.authority {
		writeError(w, http.StatusForbidden, "forbidden", "request Host is not the active loopback listener")
		return
	}
	origins := r.Header.Values("Origin")
	if len(origins) > 1 || (len(origins) == 1 && origins[0] != s.origin) {
		writeError(w, http.StatusForbidden, "forbidden", "request Origin is not the active loopback origin")
		return
	}

	routePath, cookieScoped := s.routePath(r.URL.Path)
	switch routePath {
	case "/healthz":
		if cookieScoped {
			s.serveUnknown(w, r, cookieScoped)
			return
		}
		s.serveHealth(w, r)
	case "/":
		s.serveShell(w, r, cookieScoped)
	case "/assets/app.js":
		s.serveAsset(w, r, cookieScoped, "assets/app.js", "text/javascript; charset=utf-8")
	case "/assets/style.css":
		s.serveAsset(w, r, cookieScoped, "assets/style.css", "text/css; charset=utf-8")
	case "/api/v1/snapshot":
		s.serveSnapshot(w, r, cookieScoped)
	case "/api/v1/round":
		s.serveRound(w, r, cookieScoped)
	case "/api/v1/approve":
		s.serveApprove(w, r, cookieScoped)
	default:
		s.serveUnknown(w, r, cookieScoped)
	}
}

func (s *Server) routePath(path string) (string, bool) {
	if !strings.HasPrefix(path, s.basePath) {
		return path, false
	}
	relative := strings.TrimPrefix(path, s.basePath)
	return "/" + relative, true
}

func (s *Server) serveUnknown(w http.ResponseWriter, r *http.Request, cookieScoped bool) {
	if !s.authenticated(r, cookieScoped) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "route not found")
}

func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) serveShell(w http.ResponseWriter, r *http.Request, cookieScoped bool) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if query, present := r.URL.Query()["token"]; present {
		if len(r.URL.Query()) != 1 || len(query) != 1 || !secureEqual(query[0], s.token) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: s.token, Path: s.basePath, HttpOnly: true, SameSite: http.SameSiteStrictMode})
		http.Redirect(w, r, s.basePath, http.StatusSeeOther)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
		return
	}
	serveEmbedded(w, "assets/index.html", "text/html; charset=utf-8")
}

func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request, cookieScoped bool, name, contentType string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
		return
	}
	serveEmbedded(w, name, contentType)
}

func (s *Server) serveSnapshot(w http.ResponseWriter, r *http.Request, cookieScoped bool) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
		return
	}
	s.mu.Lock()
	if !s.synchronized {
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, "internal_error", "authoring state is temporarily unavailable")
		return
	}
	response := s.responseLocked()
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) serveRound(w http.ResponseWriter, r *http.Request, cookieScoped bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
		return
	}
	var request roundRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", safeMessage(err))
		return
	}
	if strings.TrimSpace(request.Revision) == "" {
		writeError(w, http.StatusBadRequest, "malformed_request", "revision is required")
		return
	}
	if request.Answers == nil {
		writeError(w, http.StatusBadRequest, "malformed_request", "answers is required")
		return
	}
	answers := make([]authoring.RoundAnswer, len(request.Answers))
	for i, answer := range request.Answers {
		if strings.TrimSpace(answer.QuestionID) == "" {
			writeError(w, http.StatusBadRequest, "malformed_request", "every answer requires question_id")
			return
		}
		answers[i] = authoring.RoundAnswer{QuestionID: answer.QuestionID, Value: answer.Value, Source: humanInputSource}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.synchronized {
		writeError(w, http.StatusInternalServerError, "internal_error", "authoring state is temporarily unavailable")
		return
	}
	if request.Revision != s.revision {
		writeError(w, http.StatusConflict, "stale_revision", "request revision does not match the current snapshot")
		return
	}
	if s.completed {
		writeError(w, http.StatusConflict, "session_frozen", "the approved authoring session is frozen")
		return
	}
	snapshot, err := s.engine.ApplyRound(r.Context(), answers)
	if err != nil {
		if syncErr := s.synchronizeLocked(); syncErr != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to synchronize authoring state after the rejected round")
			return
		}
		writeEngineError(w, err)
		return
	}
	revision, err := revisionFor(snapshot, false, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to cache the updated authoring state")
		return
	}
	s.snapshot = snapshot
	s.revision = revision
	writeJSON(w, http.StatusOK, s.responseLocked())
}

func (s *Server) serveApprove(w http.ResponseWriter, r *http.Request, cookieScoped bool) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required")
		return
	}
	var request approveRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "malformed_request", safeMessage(err))
		return
	}
	if strings.TrimSpace(request.Revision) == "" {
		writeError(w, http.StatusBadRequest, "malformed_request", "revision is required")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.synchronized {
		writeError(w, http.StatusInternalServerError, "internal_error", "authoring state is temporarily unavailable")
		return
	}
	if request.Revision != s.revision {
		writeError(w, http.StatusConflict, "stale_revision", "request revision does not match the current snapshot")
		return
	}
	if s.completed {
		writeError(w, http.StatusConflict, "session_frozen", "the approved authoring session is frozen")
		return
	}
	result, err := s.engine.ApproveAndWrite(r.Context(), engine.Approval{
		HumanApproved: request.HumanApproved, AllowOverwrite: request.AllowOverwrite, ApproveIncomplete: request.ApproveIncomplete,
	})
	if err != nil {
		// Best-effort synchronization captures any refreshed engine state. A
		// domain rejection can also make snapshot construction repeat that same
		// rejection, so retain the last known-good cache in that case. An
		// operational failure must synchronize or fail closed.
		if syncErr := s.synchronizeLocked(); syncErr != nil {
			if operationalEngineError(err) {
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to synchronize authoring state after the rejected approval")
				return
			}
			s.synchronized = true
		}
		writeEngineError(w, err)
		return
	}
	// A successful write is the terminal state even if a best-effort inspection
	// refresh fails after the transaction has committed.
	s.completed = true
	s.writeResult = &result
	if err := s.synchronizeLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to cache the approved authoring state")
		return
	}
	writeJSON(w, http.StatusOK, s.responseLocked())
}

func (s *Server) synchronizeLocked() error {
	ctx, cancel := context.WithTimeout(context.Background(), stateSyncTimeout)
	defer cancel()
	snapshot, err := s.engine.Snapshot(ctx)
	if err != nil {
		s.synchronized = false
		return err
	}
	revision, err := revisionFor(snapshot, s.completed, s.writeResult)
	if err != nil {
		s.synchronized = false
		return err
	}
	s.snapshot = snapshot
	s.revision = revision
	s.synchronized = true
	return nil
}

func (s *Server) responseLocked() Response {
	return Response{
		Version: APIVersion, Revision: s.revision, Completed: s.completed,
		Workspace: Workspace{ExampleDir: s.exampleDir}, Snapshot: s.snapshot, WriteResult: s.writeResult,
	}
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

func writeEngineError(w http.ResponseWriter, err error) {
	if operationalEngineError(err) {
		writeError(w, http.StatusInternalServerError, "internal_error", "authoring engine operation failed")
		return
	}
	writeError(w, http.StatusUnprocessableEntity, "engine_rejected", safeMessage(err))
}

func operationalEngineError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, os.ErrPermission) {
		return true
	}
	var pathErr *os.PathError
	var linkErr *os.LinkError
	var syscallErr *os.SyscallError
	return errors.As(err, &pathErr) || errors.As(err, &linkErr) || errors.As(err, &syscallErr)
}

func secureEqual(left, right string) bool {
	return len(left) == len(right) && subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func revisionFor(snapshot engine.Snapshot, completed bool, result *engine.WriteResult) (string, error) {
	payload := struct {
		Snapshot    engine.Snapshot     `json:"snapshot"`
		Completed   bool                `json:"completed"`
		WriteResult *engine.WriteResult `json:"write_result,omitempty"`
	}{Snapshot: snapshot, Completed: completed, WriteResult: result}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal UI revision state: %w", err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, target any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return errors.New("Content-Type must be application/json")
	}
	if r.ContentLength > MaxRequestBytes {
		return fmt.Errorf("request body exceeds %d bytes", MaxRequestBytes)
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return fmt.Errorf("request body exceeds %d bytes", MaxRequestBytes)
		}
		return fmt.Errorf("decode JSON request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON document")
		}
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
	return nil
}

func methodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "request method is not supported for this route")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Version: APIVersion, Error: errorPayload{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		data = []byte(`{"version":"openudon.icot-ui-api.v1","error":{"code":"internal_error","message":"failed to encode response"}}`)
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

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; object-src 'none'")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}
