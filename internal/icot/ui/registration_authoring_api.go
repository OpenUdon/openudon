package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	icotengine "github.com/OpenUdon/openudon/internal/icot/engine"
)

type registrationAuthoringStartRequest struct {
	Revision             string                            `json:"revision"`
	RegistrationRevision string                            `json:"registration_revision"`
	ProfileID            string                            `json:"profile_id"`
	URL                  string                            `json:"url"`
	Origins              []string                          `json:"origins"`
	Bounds               *registrationauthorsession.Bounds `json:"bounds,omitempty"`
}

type registrationAuthoringCommandRequest struct {
	Revision             string                    `json:"revision"`
	RegistrationRevision string                    `json:"registration_revision"`
	Type                 string                    `json:"type"`
	Method               string                    `json:"method,omitempty"`
	URL                  string                    `json:"url,omitempty"`
	Confirmed            bool                      `json:"confirmed,omitempty"`
	Draft                *registrationDraftRequest `json:"draft,omitempty"`
}

type registrationAuthoringCancelRequest struct {
	RegistrationRevision string `json:"registration_revision"`
}

func (s *Server) serveRegistrationAuthoringStart(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request registrationAuthoringStartRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	request.Revision = strings.TrimSpace(request.Revision)
	request.RegistrationRevision = strings.TrimSpace(request.RegistrationRevision)
	request.ProfileID = strings.TrimSpace(request.ProfileID)
	request.URL = strings.TrimSpace(request.URL)
	if request.ProfileID == "" || request.URL == "" || len(request.Origins) == 0 {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "registration authoring start requires profile_id, url, and origins", false, requestID, s.currentRevision())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(r.Context()); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/registration-authoring/start", "workspace_inspection", err, true)
		return
	}
	if request.Revision != s.revision || request.RegistrationRevision != s.registrationRevision {
		s.writeError(w, http.StatusConflict, "stale_revision", "authoring or registration-authoring revision is stale", true, requestID, s.revision)
		return
	}
	if s.lifecycle != lifecycleAuthoring || s.workspace.ExternallyModified {
		s.writeError(w, http.StatusConflict, "session_frozen", "registration authoring is unavailable in the current authoring state", false, requestID, s.revision)
		return
	}
	if s.captureContainmentFailed || s.registrationContainmentFailed {
		s.writeError(w, http.StatusConflict, "browser_teardown_failed", "a prior browser process tree did not confirm teardown; restart iCoT before another browser operation", false, requestID, s.revision)
		return
	}
	if captureActive(s.capture) || registrationAuthoringActive(s.registrationAuthoring) {
		s.writeError(w, http.StatusConflict, "browser_authoring_active", "only one browser authoring session may run at a time", true, requestID, s.revision)
		return
	}
	if s.registrationAuthoring != nil && s.registrationAuthoring.State == "review_ready" && s.registrationCandidate != nil {
		s.writeError(w, http.StatusConflict, "registration_candidate_pending", "the adopted registration candidate must be reviewed or canceled before another authoring session", false, requestID, s.revision)
		return
	}
	if s.privateRoot == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "private_root_required", "registration authoring requires icot ui --private-root", false, requestID, s.revision)
		return
	}
	if s.browserTransactions == nil || s.browserTransaction == nil {
		s.writeError(w, http.StatusUnprocessableEntity, "browser_transactions_required", "registration authoring requires package scope, restrictive scratch, and generation-store configuration", false, requestID, s.revision)
		return
	}
	if s.browserTransaction.Transaction != nil {
		s.writeError(w, http.StatusConflict, "browser_transaction_active", "finish or cancel the current browser transaction before registration authoring", false, requestID, s.revision)
		return
	}

	startedAt := s.now().UTC()
	s.registrationAuthoring = &RegistrationAuthoringState{
		State: "launching", Message: "Launching an isolated headed Chromium registration-authoring session.",
		StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: startedAt.Format(time.RFC3339),
	}
	s.registrationCandidate = nil
	s.clearRegistrationDraftLocked()
	privateStart := request
	privateStart.Revision, privateStart.RegistrationRevision = "", ""
	s.registrationStart = privateStart
	digest := sha256.Sum256([]byte(s.registrationRevision + "\x00" + request.ProfileID))
	session, err := s.startRegistration(s.captureContext, browserauthor.RegistrationConfig{
		PrivateRoot: s.privateRoot, DriverDir: s.driverDir, TransactionID: "registration-" + hex.EncodeToString(digest[:8]),
		Protocol: registrationauthorsession.ProtocolV2,
	})
	if err != nil {
		s.registrationAuthoring = &RegistrationAuthoringState{
			State: "failed", Message: "The isolated Chromium registration worker could not start.",
			StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
		}
		_ = s.updateRevisionLocked()
		s.writeError(w, http.StatusUnprocessableEntity, "registration_authoring_failed", "registration authoring failed before launch", false, requestID, s.revision)
		return
	}
	s.registrationSession = session
	go s.consumeRegistrationAuthoring(session, browserauthor.RegistrationCommand{
		Type: "start", ProfileID: request.ProfileID, URL: request.URL,
		Origins: append([]string(nil), request.Origins...), Bounds: cloneRegistrationAuthoringBounds(request.Bounds),
	}, startedAt)
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/registration-authoring/start", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusAccepted, s.responseLocked(), requestID)
}

func (s *Server) consumeRegistrationAuthoring(session RegistrationAuthoringSession, start browserauthor.RegistrationCommand, startedAt time.Time) {
	terminalState, terminalCode := "", ""
	resultReady := false
	for event := range session.Events() {
		s.mu.Lock()
		if s.registrationSession != session {
			s.mu.Unlock()
			continue
		}
		if s.registrationAuthoring != nil && s.registrationAuthoring.State == "canceling" && event.State != "failed" && event.State != "canceled" {
			s.mu.Unlock()
			continue
		}
		state := &RegistrationAuthoringState{
			State: event.State, Phase: event.Phase, Bounds: cloneRegistrationAuthoringBounds(event.Bounds),
			Observation: cloneRegistrationAuthoringObservation(event.Observation),
			Draft:       cloneRegistrationDraftDisclosure(s.registrationAuthoring.Draft),
			StartedAt:   startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
		}
		if event.ErrorCode != "" {
			state.Message = registrationAuthoringErrorMessage(event.ErrorCode)
		}
		if event.State == "ready" {
			state.State = "starting"
			state.Message = "The isolated worker is ready; opening the reviewed initial navigation."
			s.registrationAuthoring = state
			_ = s.updateRevisionLocked()
			s.mu.Unlock()
			ctx, cancel := context.WithTimeout(s.captureContext, 5*time.Second)
			err := session.Send(ctx, start)
			cancel()
			if err != nil {
				session.Cancel()
				s.mu.Lock()
				if s.registrationSession == session {
					s.registrationAuthoring = &RegistrationAuthoringState{State: "canceling", Message: "The registration worker rejected its fixed start command; waiting for teardown.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339)}
					terminalState, terminalCode = "failed", "start_rejected"
					_ = s.updateRevisionLocked()
				}
				s.mu.Unlock()
			}
			continue
		}
		if event.Candidate != nil {
			s.registrationCandidate = event.Candidate
			resultReady = true
			state.State = "teardown"
			state.Message = "The reviewed candidate was reconstructed; waiting for the worker event stream to close."
		}
		if event.State == "closed" {
			state.State = "teardown"
			state.Message = "The authoring protocol is closed; verifying clean worker teardown and result reconstruction."
		}
		if event.State == "failed" || event.State == "canceled" {
			terminalState, terminalCode = event.State, event.ErrorCode
			state.State = "canceling"
			state.Message = "The registration worker stopped; waiting for process-tree teardown to complete."
		}
		s.registrationAuthoring = state
		_ = s.updateRevisionLocked()
		s.mu.Unlock()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.registrationSession != session {
		return
	}
	s.registrationSession = nil
	updatedAt := s.now().UTC().Format(time.RFC3339)
	if terminalState != "" || s.registrationAuthoring != nil && s.registrationAuthoring.State == "canceling" {
		state := terminalState
		if state == "" {
			state = "canceled"
		}
		message := "Registration authoring was canceled after the worker and all descendants stopped; no candidate was adopted."
		if state == "failed" {
			message = "Registration authoring failed closed after the worker and all descendants stopped; no candidate is available."
		}
		containmentFailed := terminalCode == "worker_teardown"
		if containmentFailed {
			s.registrationContainmentFailed = true
			message += " Restart iCoT before starting another browser operation."
		}
		s.registrationAuthoring = &RegistrationAuthoringState{State: state, Message: message, ContainmentFailed: containmentFailed, StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt}
		s.registrationCandidate = nil
		s.clearRegistrationDraftLocked()
		_ = s.updateRevisionLocked()
		return
	}
	if resultReady && s.registrationCandidate != nil {
		draft := cloneRegistrationDraftDisclosure(s.registrationAuthoring.Draft)
		transactionSnapshot, err := s.startRegistrationTransactionLocked(s.captureContext, s.registrationCandidate)
		if err != nil {
			s.registrationAuthoring = &RegistrationAuthoringState{State: "failed", Message: "The adopted registration candidate could not enter the exact browser transaction lifecycle.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt}
			s.registrationCandidate = nil
			s.clearRegistrationDraftLocked()
			_ = s.updateRevisionLocked()
			return
		}
		s.browserTransaction = &transactionSnapshot
		s.registrationAuthoring = &RegistrationAuthoringState{
			State: "transaction_review", Message: "The worker stopped cleanly and the exact transaction-v2 candidate is ready for separate browser transaction review.",
			ResultReady: true, Draft: draft, StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt,
		}
		s.clearRegistrationDraftLocked()
		_ = s.updateRevisionLocked()
		return
	}
	s.registrationAuthoring = &RegistrationAuthoringState{State: "failed", Message: "The registration worker ended without a promotable candidate.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt}
	s.registrationCandidate = nil
	s.clearRegistrationDraftLocked()
	_ = s.updateRevisionLocked()
}

func (s *Server) startRegistrationTransactionLocked(ctx context.Context, candidate *browsercandidate.Registration) (transactionengine.Snapshot, error) {
	if s.browserTransactions == nil || candidate == nil {
		return transactionengine.Snapshot{}, errors.New("registration transaction lifecycle is unavailable")
	}
	transaction := candidate.Transaction()
	data, err := browsertransaction.CanonicalBytes(transaction)
	if err != nil {
		return transactionengine.Snapshot{}, err
	}
	digest, err := browsertransaction.Digest(transaction)
	if err != nil {
		return transactionengine.Snapshot{}, err
	}
	observeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	initial, err := s.browserTransactions.Observe(observeCtx)
	if err != nil || initial.Transaction != nil {
		return initial, errors.New("browser transaction lifecycle is not inactive")
	}
	return s.browserTransactions.Start(observeCtx, transactionengine.StartRequest{
		ExpectedRevision: initial.Revision, ExpectedTransactionSHA256: digest, TransactionJSON: data,
	})
}

func (s *Server) adoptReviewedRegistrationLocked(ctx context.Context, transactionSnapshot transactionengine.Snapshot) error {
	if s.registrationCandidate == nil || s.registrationAuthoring == nil || s.registrationAuthoring.State != "transaction_review" || transactionSnapshot.Transaction == nil {
		return errors.New("registration candidate is not pending transaction review")
	}
	expected, err := s.registrationCandidate.ReviewedTransaction()
	if err != nil {
		return err
	}
	expectedDigest, err := browsertransaction.Digest(expected)
	if err != nil {
		return err
	}
	actualDigest, err := browsertransaction.Digest(*transactionSnapshot.Transaction)
	if err != nil || expectedDigest != actualDigest || actualDigest != transactionSnapshot.TransactionSHA256 || transactionSnapshot.Transaction.State != browsertransaction.StateReviewed {
		return errors.New("reviewed browser transaction does not match the private registration candidate")
	}
	authoringEngine, ok := s.engine.(registrationVirtualBrowserEngine)
	if !ok {
		return errors.New("authoring engine does not support virtual browser sources")
	}
	input, err := icotengine.RegistrationVirtualBrowserTransaction(s.registrationCandidate, true)
	if err != nil {
		return err
	}
	replaced, err := authoringEngine.ReplaceVirtualBrowserSources(ctx, s.snapshot.SourceCandidates.VirtualBrowser.Generation, []elicitor.VirtualBrowserTransactionInput{input})
	if err != nil {
		return err
	}
	candidateID := ""
	for _, candidate := range replaced.SourceCandidates.VirtualBrowser.Candidates {
		if candidate.TransactionID == expected.ID && candidate.Kind == browsertransaction.CandidateRegistration {
			if candidateID != "" {
				return errors.New("registration virtual source is ambiguous")
			}
			candidateID = candidate.ID
		}
	}
	if candidateID == "" {
		return errors.New("registration virtual source is unavailable")
	}
	selected, err := authoringEngine.SelectVirtualBrowserSources(ctx, replaced.SourceCandidates.VirtualBrowser.Generation, []string{candidateID})
	if err != nil {
		return err
	}
	draft := cloneRegistrationDraftDisclosure(s.registrationAuthoring.Draft)
	s.snapshot = selected
	s.registrationAuthoring = &RegistrationAuthoringState{
		State: "adopted", Message: "The explicitly reviewed transaction-v2 candidate is selected as a virtual source. Complete authoring, write, and build the package before preparation.",
		Draft: draft, StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	}
	s.registrationCandidate = nil
	return nil
}

func (s *Server) serveRegistrationAuthoringCommand(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request registrationAuthoringCommandRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	command, ok := registrationAuthoringWireCommand(request)
	if !ok {
		s.writeError(w, http.StatusBadRequest, "malformed_request", "registration authoring command is not in the closed command union", false, requestID, s.currentRevision())
		return
	}
	if strings.TrimSpace(request.Type) == "draft" {
		s.serveRegistrationAuthoringDraft(w, r, requestID, request)
		return
	}

	s.mu.Lock()
	if strings.TrimSpace(request.Revision) != s.revision || strings.TrimSpace(request.RegistrationRevision) != s.registrationRevision || s.registrationSession == nil || !registrationAuthoringCommandAllowed(s.registrationAuthoring, request.Type) {
		s.writeError(w, http.StatusConflict, "stale_registration_revision", "registration-authoring revision is stale or the command is not pending", true, requestID, s.revision)
		s.mu.Unlock()
		return
	}
	session := s.registrationSession
	if strings.TrimSpace(request.Type) == "review" {
		if len(s.registrationDraft) == 0 || len(s.registrationDraftCandidates) == 0 || len(s.registrationDraftBindings) == 0 || s.registrationDraftFlow == "" || s.registrationDraftCleanup == "" {
			s.writeError(w, http.StatusConflict, "registration_draft_missing", "the canonical registration draft is unavailable", false, requestID, s.revision)
			s.mu.Unlock()
			return
		}
		command.Profile = append([]byte(nil), s.registrationDraft...)
		command.CandidateIDs = append([]string(nil), s.registrationDraftCandidates...)
		command.CredentialBindings = append([]browsertransaction.CredentialBinding(nil), s.registrationDraftBindings...)
		command.Flow = s.registrationDraftFlow
		command.CleanupDisposition = s.registrationDraftCleanup
	}
	previous := *s.registrationAuthoring
	s.registrationAuthoring = &RegistrationAuthoringState{
		State: "commanding", Message: "The typed command was accepted; waiting for the isolated worker.", Phase: previous.Phase,
		StartedAt: previous.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
		Draft: cloneRegistrationDraftDisclosure(previous.Draft),
	}
	if err := s.updateRevisionLocked(); err != nil {
		s.registrationAuthoring = &previous
		s.writeInternalError(w, r, requestID, "/api/v4/registration-authoring/command", "revision", err, true)
		s.mu.Unlock()
		return
	}
	reservedRevision := s.registrationRevision
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := session.Send(ctx, command); err != nil {
		s.mu.Lock()
		if s.registrationSession == session && s.registrationRevision == reservedRevision {
			s.registrationAuthoring = &previous
			_ = s.updateRevisionLocked()
		}
		revision := s.revision
		s.mu.Unlock()
		s.writeError(w, http.StatusUnprocessableEntity, "registration_command_rejected", "the typed command is not valid for the current registration-authoring phase", false, requestID, revision)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusAccepted, s.responseLocked(), requestID)
}

func (s *Server) serveRegistrationAuthoringCancel(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	var request registrationAuthoringCancelRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		s.writeRequestError(w, err, requestID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(request.RegistrationRevision) != s.registrationRevision || s.registrationSession == nil || !registrationAuthoringActive(s.registrationAuthoring) || s.registrationAuthoring.State == "canceling" {
		s.writeError(w, http.StatusConflict, "stale_registration_revision", "registration-authoring revision is stale or no session is active", true, requestID, s.revision)
		return
	}
	s.registrationSession.Cancel()
	s.registrationCandidate = nil
	s.clearRegistrationDraftLocked()
	s.registrationAuthoring = &RegistrationAuthoringState{
		State: "canceling", Message: "Cancellation was requested; waiting for the isolated worker and all descendants to stop.",
		StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	}
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/registration-authoring/cancel", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusAccepted, s.responseLocked(), requestID)
}

func registrationAuthoringWireCommand(request registrationAuthoringCommandRequest) (browserauthor.RegistrationCommand, bool) {
	typeName := strings.TrimSpace(request.Type)
	method := strings.TrimSpace(request.Method)
	url := strings.TrimSpace(request.URL)
	switch typeName {
	case "observe":
		if method != "" || url != "" || request.Confirmed || request.Draft != nil {
			return browserauthor.RegistrationCommand{}, false
		}
		return browserauthor.RegistrationCommand{Type: typeName}, true
	case "navigate":
		if method != http.MethodGet && method != http.MethodHead || url == "" || request.Confirmed || request.Draft != nil {
			return browserauthor.RegistrationCommand{}, false
		}
		return browserauthor.RegistrationCommand{Type: typeName, Method: method, URL: url}, true
	case "draft":
		if method != "" || url != "" || request.Confirmed || request.Draft == nil {
			return browserauthor.RegistrationCommand{}, false
		}
		return browserauthor.RegistrationCommand{Type: typeName}, true
	case "review":
		if method != "" || url != "" || !request.Confirmed || request.Draft != nil {
			return browserauthor.RegistrationCommand{}, false
		}
		return browserauthor.RegistrationCommand{
			Type: "review", Confirmed: true,
		}, true
	case "finish":
		if method != "" || url != "" || !request.Confirmed || request.Draft != nil {
			return browserauthor.RegistrationCommand{}, false
		}
		return browserauthor.RegistrationCommand{Type: "finish", Confirmed: true}, true
	default:
		return browserauthor.RegistrationCommand{}, false
	}
}

func registrationAuthoringCommandAllowed(state *RegistrationAuthoringState, command string) bool {
	if state == nil {
		return false
	}
	switch strings.TrimSpace(command) {
	case "observe":
		return state.State == "observing"
	case "navigate":
		return state.State == "observation"
	case "review":
		return state.State == "draft_review"
	case "finish":
		return state.State == "reviewed"
	default:
		return false
	}
}

func (s *Server) serveRegistrationAuthoringDraft(w http.ResponseWriter, r *http.Request, requestID string, request registrationAuthoringCommandRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(request.Revision) != s.revision || strings.TrimSpace(request.RegistrationRevision) != s.registrationRevision ||
		s.registrationSession == nil || s.registrationAuthoring == nil || s.registrationAuthoring.State != "observation" || s.registrationAuthoring.Observation == nil {
		s.writeError(w, http.StatusConflict, "stale_registration_revision", "registration-authoring revision is stale or no observation is ready for drafting", true, requestID, s.revision)
		return
	}
	profile, candidates, bindings, disclosure, err := buildRegistrationDraft(*request.Draft, s.registrationStart, *s.registrationAuthoring.Observation, s.now().UTC())
	if err != nil {
		message := "the structured registration draft is invalid"
		if errors.Is(err, errRegistrationDraftBindingsInvalid) {
			message = "credential bindings must be unique lowercase descriptive environment symbol names and must not contain recognized credential formats"
		}
		s.writeError(w, http.StatusUnprocessableEntity, "registration_draft_rejected", message, false, requestID, s.revision)
		return
	}
	s.registrationDraft = append([]byte(nil), profile...)
	s.registrationDraftCandidates = append([]string(nil), candidates...)
	s.registrationDraftBindings = append([]browsertransaction.CredentialBinding(nil), bindings...)
	s.registrationDraftFlow = request.Draft.Flow.Name
	s.registrationDraftCleanup = request.Draft.CallControls.CleanupDisposition
	s.registrationAuthoring = &RegistrationAuthoringState{
		State: "draft_review", Message: "Review the exact canonical credential-free BRP, retained queries, symbolic bindings, effects, and fixed call controls before confirming.",
		Phase: s.registrationAuthoring.Phase, Observation: cloneRegistrationAuthoringObservation(s.registrationAuthoring.Observation), Draft: disclosure,
		StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	}
	if err := s.updateRevisionLocked(); err != nil {
		s.writeInternalError(w, r, requestID, "/api/v4/registration-authoring/command", "revision", err, true)
		return
	}
	setETag(w, s.etag)
	s.writeJSON(w, http.StatusOK, s.responseLocked(), requestID)
}

func (s *Server) clearRegistrationDraftLocked() {
	s.registrationDraft = nil
	s.registrationDraftCandidates = nil
	s.registrationDraftBindings = nil
	s.registrationDraftFlow = ""
	s.registrationDraftCleanup = ""
}

func cloneRegistrationDraftDisclosure(value *RegistrationDraftDisclosure) *RegistrationDraftDisclosure {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Canonical = append(json.RawMessage(nil), value.Canonical...)
	copy.CredentialBindings = append([]browsertransaction.CredentialBinding(nil), value.CredentialBindings...)
	copy.RetainedQueries = append([]RetainedQueryDisclosure(nil), value.RetainedQueries...)
	for index := range copy.RetainedQueries {
		copy.RetainedQueries[index].Parameters = append([]RetainedQueryParameter(nil), value.RetainedQueries[index].Parameters...)
	}
	return &copy
}

func registrationAuthoringErrorMessage(code string) string {
	switch code {
	case "operator_idle_timeout":
		return "Registration authoring reached its operator idle limit."
	case "worker_teardown":
		return "The registration worker process tree did not confirm teardown."
	case "candidate_rejected", "review_missing":
		return "The private registration result did not match the reviewed authoring session."
	default:
		return "The registration worker failed closed."
	}
}

func cloneRegistrationAuthoringBounds(value *registrationauthorsession.Bounds) *registrationauthorsession.Bounds {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneRegistrationAuthoringObservation(value *registrationauthorsession.Observation) *registrationauthorsession.Observation {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Candidates = append([]registrationauthorsession.Candidate(nil), value.Candidates...)
	copy.Diagnostics = append([]string(nil), value.Diagnostics...)
	return &copy
}
