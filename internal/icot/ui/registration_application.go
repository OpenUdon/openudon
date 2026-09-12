package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

// RegistrationApplication owns registration decisions independently of transport.
// UI and supervised control share the same revision, one-attempt latch, worker,
// draft, transaction and containment state. No method grants runtime authority.
type RegistrationApplication struct{ server *Server }

type registrationResult struct {
	Revision string
	Status   int
	Snapshot Response
	ETag     string
	Failure  *errorPayload
}

func registrationFailure(status int, code, message string, retryable bool, revision string) registrationResult {
	return registrationResult{Status: status, Failure: &errorPayload{Code: code, Message: message, Retryable: retryable}, Revision: revision}
}

func (s *Server) writeRegistrationResult(w http.ResponseWriter, requestID string, result registrationResult) {
	if result.Failure != nil {
		e := result.Failure
		s.writeError(w, result.Status, e.Code, e.Message, e.Retryable, requestID, result.Revision)
		return
	}
	setETag(w, result.ETag)
	s.writeJSON(w, result.Status, result.Snapshot, requestID)
}

func (app RegistrationApplication) Start(ctx context.Context, request registrationAuthoringStartRequest) (result registrationResult) {
	if request.ProfileVersion != "" && request.ProfileVersion != "1.0" && request.ProfileVersion != "1.1" {
		return registrationFailure(http.StatusBadRequest, "malformed_request", "unsupported registration profile version", false, "")
	}
	s := app.server
	request.Revision = strings.TrimSpace(request.Revision)
	request.RegistrationRevision = strings.TrimSpace(request.RegistrationRevision)
	request.ProfileID = strings.TrimSpace(request.ProfileID)
	request.URL = strings.TrimSpace(request.URL)
	if request.ProfileID == "" || request.URL == "" || len(request.Origins) == 0 {
		result = registrationFailure(http.StatusBadRequest, "malformed_request", "registration authoring start requires profile_id, url, and origins", false, s.currentRevision())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.registrationAttemptConsumed {
		result = registrationFailure(http.StatusConflict, "registration_authorization_consumed", "this iCoT process already consumed its one registration-authoring attempt; a later session requires a fresh preflight, authorization, and process", false, s.revision)
		return
	}
	if err := s.refreshWorkspaceLocked(ctx); err != nil {
		result = registrationFailure(http.StatusInternalServerError, "workspace_inspection", "registration application failure", true, s.revision)
		return
	}
	if request.Revision != s.revision || request.RegistrationRevision != s.registrationRevision {
		result = registrationFailure(http.StatusConflict, "stale_revision", "authoring or registration-authoring revision is stale", true, s.revision)
		return
	}
	if s.lifecycle != lifecycleAuthoring || s.workspace.ExternallyModified {
		result = registrationFailure(http.StatusConflict, "session_frozen", "registration authoring is unavailable in the current authoring state", false, s.revision)
		return
	}
	if s.browserContainmentFailedLocked() {
		result = registrationFailure(http.StatusConflict, "browser_teardown_failed", "a prior browser process tree did not confirm teardown; restart iCoT before another browser operation", false, s.revision)
		return
	}
	if captureActive(s.capture) || registrationAuthoringActive(s.registrationAuthoring) {
		result = registrationFailure(http.StatusConflict, "browser_authoring_active", "only one browser authoring session may run at a time", true, s.revision)
		return
	}
	if s.registrationAuthoring != nil && s.registrationAuthoring.State == "review_ready" && s.registrationCandidate != nil {
		result = registrationFailure(http.StatusConflict, "registration_candidate_pending", "the adopted registration candidate must be reviewed or canceled before another authoring session", false, s.revision)
		return
	}
	if s.privateRoot == "" {
		result = registrationFailure(http.StatusUnprocessableEntity, "private_root_required", "registration authoring requires icot ui --private-root", false, s.revision)
		return
	}
	if s.browserTransactions == nil || s.browserTransaction == nil {
		result = registrationFailure(http.StatusUnprocessableEntity, "browser_transactions_required", "registration authoring requires package scope, restrictive scratch, and generation-store configuration", false, s.revision)
		return
	}
	if s.browserTransaction.Transaction != nil {
		result = registrationFailure(http.StatusConflict, "browser_transaction_active", "finish or cancel the current browser transaction before registration authoring", false, s.revision)
		return
	}

	startedAt := s.now().UTC()
	if !s.registrationAuthority.allowsStart(request, startedAt) {
		return registrationFailure(http.StatusForbidden, "registration_authority", "registration start is outside the consumer's fixed authority", false, s.revision)
	}
	s.registrationAttemptConsumed = true
	s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
		State: "launching", Message: "Launching an isolated headed Chromium registration-authoring session.",
		StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: startedAt.Format(time.RFC3339),
	})
	s.registrationCandidate = nil
	s.clearRegistrationDraftLocked()
	privateStart := request
	privateStart.Revision, privateStart.RegistrationRevision = "", ""
	s.registrationStart = privateStart
	digest := sha256.Sum256([]byte(s.registrationRevision + "\x00" + request.ProfileID))
	protocol := registrationauthorsession.ProtocolV2
	if request.ProfileVersion == "1.1" {
		protocol = registrationauthorsession.ProtocolV3
	}
	session, err := s.startRegistration(s.captureContext, browserauthor.RegistrationConfig{
		PrivateRoot: s.privateRoot, DriverDir: s.driverDir, TransactionID: "registration-" + hex.EncodeToString(digest[:8]),
		Protocol: protocol,
	})
	if err != nil {
		s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
			State: "failed", FailureCode: "worker_start_failed", Message: "The isolated Chromium registration worker could not start. This iCoT process has consumed its registration-authoring attempt; a later session requires a fresh preflight, authorization, and process.",
			StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
		})
		_ = s.updateRevisionLocked()
		result = registrationFailure(http.StatusUnprocessableEntity, "registration_authoring_failed", "registration authoring failed before launch", false, s.revision)
		return
	}
	s.registrationSession = session
	go s.consumeRegistrationAuthoring(session, browserauthor.RegistrationCommand{
		Type: "start", ProfileID: request.ProfileID, URL: request.URL,
		Origins: append([]string(nil), request.Origins...), Bounds: cloneRegistrationAuthoringBounds(request.Bounds),
	}, startedAt)
	if err := s.updateRevisionLocked(); err != nil {
		result = registrationFailure(http.StatusInternalServerError, "revision", "registration application failure", true, s.revision)
		return
	}
	result = registrationResult{Status: http.StatusAccepted, Snapshot: s.responseLocked(), ETag: s.etag}
	return
}

func (app RegistrationApplication) Command(ctx context.Context, request registrationAuthoringCommandRequest) (result registrationResult) {
	s := app.server
	command, ok := registrationAuthoringWireCommand(request)
	if !ok {
		result = registrationFailure(http.StatusBadRequest, "malformed_request", "registration authoring command is not in the closed command union", false, s.currentRevision())
		return
	}
	if strings.TrimSpace(request.Type) == "draft" {
		result = app.Draft(ctx, request)
		return
	}

	s.mu.Lock()
	if strings.TrimSpace(request.Revision) != s.revision || strings.TrimSpace(request.RegistrationRevision) != s.registrationRevision || s.registrationSession == nil || !registrationAuthoringCommandAllowed(s.registrationAuthoring, request.Type) {
		result = registrationFailure(http.StatusConflict, "stale_registration_revision", "registration-authoring revision is stale or the command is not pending", true, s.revision)
		s.mu.Unlock()
		return
	}
	session := s.registrationSession
	if s.registrationAuthority.validate(s.now()) != nil || command.Type == "navigate" && !s.registrationAuthority.allowsNavigation(command.URL, s.now()) {
		s.mu.Unlock()
		return registrationFailure(http.StatusForbidden, "registration_authority", "registration command is outside fixed authority", false, "")
	}
	if strings.TrimSpace(request.Type) == "review" {
		if len(s.registrationDraft) == 0 || len(s.registrationDraftCandidates) == 0 || len(s.registrationDraftBindings) == 0 || s.registrationDraftFlow == "" || s.registrationDraftCleanup == "" {
			result = registrationFailure(http.StatusConflict, "registration_draft_missing", "the canonical registration draft is unavailable", false, s.revision)
			s.mu.Unlock()
			return
		}
		command.Profile = append([]byte(nil), s.registrationDraft...)
		command.CandidateIDs = append([]string(nil), s.registrationDraftCandidates...)
		command.CredentialBindings = append([]browsertransaction.CredentialBinding(nil), s.registrationDraftBindings...)
		command.Flow = s.registrationDraftFlow
		command.CleanupDisposition = s.registrationDraftCleanup
		command.StepCandidates = append([]string(nil), s.registrationAuthoring.Draft.StepCandidates...)
	}
	previous := *s.registrationAuthoring
	s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
		State: "commanding", Message: "The typed command was accepted; waiting for the isolated worker.", Phase: previous.Phase,
		StartedAt: previous.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
		Draft: cloneRegistrationDraftDisclosure(previous.Draft),
	})
	if err := s.updateRevisionLocked(); err != nil {
		s.setRegistrationAuthoringLocked(&previous)
		result = registrationFailure(http.StatusInternalServerError, "revision", "registration application failure", true, s.revision)
		s.mu.Unlock()
		return
	}
	reservedRevision := s.registrationRevision
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := session.Send(ctx, command); err != nil {
		s.mu.Lock()
		if s.registrationSession == session && s.registrationRevision == reservedRevision {
			s.setRegistrationAuthoringLocked(&previous)
			_ = s.updateRevisionLocked()
		}
		revision := s.revision
		s.mu.Unlock()
		result = registrationFailure(http.StatusUnprocessableEntity, "registration_command_rejected", "the typed command is not valid for the current registration-authoring phase", false, revision)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	result = registrationResult{Status: http.StatusAccepted, Snapshot: s.responseLocked(), ETag: s.etag}
	return
}

func (app RegistrationApplication) Cancel(ctx context.Context, request registrationAuthoringCancelRequest) (result registrationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(request.RegistrationRevision) != s.registrationRevision || s.registrationSession == nil || !registrationAuthoringActive(s.registrationAuthoring) || s.registrationAuthoring.State == "canceling" {
		result = registrationFailure(http.StatusConflict, "stale_registration_revision", "registration-authoring revision is stale or no session is active", true, s.revision)
		return
	}
	s.registrationSession.Cancel()
	s.registrationCandidate = nil
	s.clearRegistrationDraftLocked()
	s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
		State: "canceling", Message: "Cancellation was requested; waiting for the isolated worker and all descendants to stop.",
		StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	})
	if err := s.updateRevisionLocked(); err != nil {
		result = registrationFailure(http.StatusInternalServerError, "revision", "registration application failure", true, s.revision)
		return
	}
	result = registrationResult{Status: http.StatusAccepted, Snapshot: s.responseLocked(), ETag: s.etag}
	return
}

func (app RegistrationApplication) Draft(ctx context.Context, request registrationAuthoringCommandRequest) (result registrationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.Draft == nil {
		return registrationFailure(http.StatusBadRequest, "malformed_request", "registration draft is required", false, s.revision)
	}
	for _, step := range request.Draft.Flow.Steps {
		if step.Type == "navigate" && !s.registrationAuthority.allowsNavigation(step.Navigate, s.now()) {
			return registrationFailure(http.StatusForbidden, "registration_authority", "draft navigation is outside fixed authority", false, s.revision)
		}
	}
	if strings.TrimSpace(request.Revision) != s.revision || strings.TrimSpace(request.RegistrationRevision) != s.registrationRevision ||
		s.registrationSession == nil || s.registrationAuthoring == nil || s.registrationAuthoring.State != "observation" || s.registrationAuthoring.Observation == nil {
		result = registrationFailure(http.StatusConflict, "stale_registration_revision", "registration-authoring revision is stale or no observation is ready for drafting", true, s.revision)
		return
	}
	profile, candidates, bindings, disclosure, err := buildRegistrationDraftHistory(*request.Draft, s.registrationStart, *s.registrationAuthoring.Observation, s.registrationAuthoring.History, s.registrationAuthoring.Previews, s.now().UTC())
	if err != nil {
		// Draft validation returns fixed local messages, never page or field values.
		message := err.Error()
		if errors.Is(err, errRegistrationDraftBindingsInvalid) {
			message = "credential bindings must be unique lowercase environment symbol names; entropy-like names must use reviewed descriptive terms and must not contain recognized credential formats"
		}
		result = registrationFailure(http.StatusUnprocessableEntity, "registration_draft_rejected", message, false, s.revision)
		return
	}
	s.registrationDraft = append([]byte(nil), profile...)
	s.registrationDraftCandidates = append([]string(nil), candidates...)
	s.registrationDraftBindings = append([]browsertransaction.CredentialBinding(nil), bindings...)
	s.registrationDraftFlow = request.Draft.Flow.Name
	s.registrationDraftCleanup = request.Draft.CallControls.CleanupDisposition
	s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
		State: "draft_review", Message: "Review the exact canonical credential-free BRP, retained queries, symbolic bindings, effects, and fixed call controls before confirming.",
		Phase: s.registrationAuthoring.Phase, Observation: cloneRegistrationAuthoringObservation(s.registrationAuthoring.Observation), Draft: disclosure,
		StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	})
	if err := s.updateRevisionLocked(); err != nil {
		result = registrationFailure(http.StatusInternalServerError, "revision", "registration application failure", true, s.revision)
		return
	}
	result = registrationResult{Status: http.StatusOK, Snapshot: s.responseLocked(), ETag: s.etag}
	return
}
