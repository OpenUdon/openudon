package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

// Application is the transport-independent lifecycle shared by UI and private
// command adapters. It retains one Server-owned state and never invokes runtime.
type Application struct{ server *Server }

// applicationResult carries the existing review resource or reduced failure.
// Only the HTTP adapter emits request diagnostics; private control exposes codes.
type applicationResult struct {
	Status       int
	Data         any
	Discovery    json.RawMessage
	ETag         string
	Revision     string
	Failure      *errorPayload
	Route, Stage string
	Cause        error
}

func (r *applicationResult) respond(status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		r.fail(http.StatusInternalServerError, "internal_error", "failed to encode response", false, "")
		return
	}
	r.Status, r.Data = status, json.RawMessage(data)
}
func (r *applicationResult) fail(status int, code, message string, retry bool, revision string) {
	r.Status, r.Revision = status, revision
	r.Failure = &errorPayload{Code: code, Message: message, Retryable: retry}
}
func (r *applicationResult) question(status int, code, message string, retry bool, revision, question string) {
	r.fail(status, code, message, retry, revision)
	r.Failure.QuestionID = question
}
func (r *applicationResult) internal(revision, route, stage string, cause error, retry bool) {
	r.fail(http.StatusInternalServerError, "internal_error", "authoring engine operation failed", retry, revision)
	r.Route, r.Stage, r.Cause = route, stage, cause
}
func (r *applicationResult) engineError(revision, route, stage string, err error) {
	class, _ := engine.FailureDetails(err)
	switch class {
	case engine.FailureRejected:
		r.question(http.StatusUnprocessableEntity, "engine_rejected", safeMessage(err), false, revision, engine.FailureQuestionID(err))
	case engine.FailureConflict:
		r.fail(http.StatusConflict, "workspace_changed", "the authoring workspace changed outside this process; restart is required", false, revision)
	case engine.FailureIndeterminate:
		r.internal(revision, route, stage, err, false)
	default:
		r.internal(revision, route, stage, err, true)
	}
}
func (s *Server) writeApplicationResult(w http.ResponseWriter, request *http.Request, requestID string, reply applicationResult) {
	if reply.Cause != nil {
		fmt.Fprintf(s.errOut, "icot ui: request_id=%s route=%s stage=%s cause=%s\n", requestID, reply.Route, reply.Stage, sanitizeLogCause(reply.Cause))
	}
	if e := reply.Failure; e != nil {
		s.writeQuestionError(w, reply.Status, e.Code, e.Message, e.Retryable, requestID, reply.Revision, e.QuestionID)
		return
	}
	setETag(w, reply.ETag)
	s.writeJSON(w, reply.Status, reply.Data, requestID)
}

func (app Application) BrowserPreflight(ctx context.Context, request captureMutationRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	if !app.beginMutation(ctx, &reply, "/api/v4/browser/preflight", strings.TrimSpace(request.Revision)) {
		s.mu.Unlock()
		return
	}
	if strings.TrimSpace(request.CaptureRevision) != s.captureRevision {
		reply.fail(http.StatusConflict, "stale_capture_revision", "capture revision is stale", true, s.revision)
		s.mu.Unlock()
		return
	}
	if s.privateRoot == "" {
		reply.fail(http.StatusUnprocessableEntity, "private_root_required", "browser capture requires icot ui --private-root", false, s.revision)
		s.mu.Unlock()
		return
	}
	previousCapture, previousRevision, previousCaptureRevision, previousETag := s.capture, s.revision, s.captureRevision, s.etag
	s.capture = &CaptureState{State: "preflight", Message: "Checking the installed Playwright driver and Chromium runtime.", UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	if err := s.updateRevisionLocked(); err != nil {
		s.capture, s.revision, s.captureRevision, s.etag = previousCapture, previousRevision, previousCaptureRevision, previousETag
		reply.internal(s.revision, "/api/v4/browser/preflight", "revision", err, true)
		s.mu.Unlock()
		return
	}
	preflightRevision := s.captureRevision
	preflightContext, preflightCancel := context.WithCancel(ctx)
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
		reply.fail(http.StatusConflict, "capture_canceled", "browser preflight was canceled", false, s.revision)
		return
	}
	if s.captureRevision != preflightRevision || s.capture == nil || s.capture.State != "preflight" {
		preflightCancel()
		reply.fail(http.StatusConflict, "stale_capture_revision", "browser preflight state changed before the readiness check completed", true, s.revision)
		return
	}
	preflightCancel()
	s.captureCancel = nil
	if browserauthor.TeardownFailed(err) {
		s.captureContainmentFailed = true
		s.capture = &CaptureState{State: "failed", Message: "The Chromium readiness worker did not confirm process-tree teardown. Restart iCoT before another browser capture.", ContainmentFailed: true, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		_ = s.updateRevisionLocked()
		reply.fail(http.StatusUnprocessableEntity, "capture_teardown_failed", "Chromium readiness process teardown was not confirmed; restart iCoT", false, s.revision)
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
		reply.fail(http.StatusUnprocessableEntity, "browser_unavailable", "Browsertools could not verify Chromium readiness", false, s.revision)
		return
	}
	s.capture = &CaptureState{State: "configuring", Message: "Chromium is ready. Review the exact capture authority before launch.", UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/browser/preflight", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) CaptureStart(ctx context.Context, request captureStartRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(ctx); err != nil {
		reply.internal(s.revision, "/api/v4/capture/start", "workspace_inspection", err, true)
		return
	}
	if strings.TrimSpace(request.Revision) != s.revision || strings.TrimSpace(request.CaptureRevision) != s.captureRevision {
		reply.fail(http.StatusConflict, "stale_revision", "authoring or capture revision is stale", true, s.revision)
		return
	}
	if s.lifecycle != lifecycleAuthoring || s.workspace.ExternallyModified {
		reply.fail(http.StatusConflict, "session_frozen", "browser capture is unavailable in the current authoring state", false, s.revision)
		return
	}
	if s.browserContainmentFailedLocked() {
		reply.fail(http.StatusConflict, "capture_teardown_failed", "a prior browser process tree did not confirm teardown; restart iCoT before another capture", false, s.revision)
		return
	}
	if captureActive(s.capture) && s.capture.State != "configuring" {
		reply.fail(http.StatusConflict, "capture_active", "only one browser capture may run at a time", true, s.revision)
		return
	}
	if registrationAuthoringActive(s.registrationAuthoring) {
		reply.fail(http.StatusConflict, "registration_authoring_active", "browser capture is blocked while registration authoring is active", true, s.revision)
		return
	}
	if s.privateRoot == "" {
		reply.fail(http.StatusUnprocessableEntity, "private_root_required", "browser capture requires icot ui --private-root", false, s.revision)
		return
	}
	if s.capture == nil || s.capture.State != "configuring" || s.doctorReport == nil || !s.doctorReport.DriverReady || !s.doctorReport.BrowserReady {
		reply.fail(http.StatusConflict, "browser_preflight_required", "a passing Chromium preflight is required before browser capture launch", false, s.revision)
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
		reply.fail(http.StatusUnprocessableEntity, "capture_failed", "browser capture failed before launch", false, s.revision)
		return
	}
	s.captureSession = session
	go s.consumeCapture(session, startedAt)
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/capture/start", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusAccepted, s.responseLocked())
	return
}

func (app Application) CaptureRespond(ctx context.Context, request captureRespondRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	if request.CaptureRevision != s.captureRevision || !captureResponsePending(s.capture) || s.captureSession == nil {
		reply.fail(http.StatusConflict, "stale_capture_revision", "capture revision is stale or no capture response is pending", true, s.revision)
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
		reply.internal(s.revision, "/api/v4/capture/respond", "revision", err, true)
		s.mu.Unlock()
		return
	}
	reservedRevision := s.captureRevision
	s.mu.Unlock()
	responseCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := session.Respond(responseCtx, request.Response); err != nil {
		s.mu.Lock()
		if s.captureSession == session && s.captureRevision == reservedRevision {
			s.capture = &pending
			if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
				reply.internal(s.revision, "/api/v4/capture/respond", "revision", revisionErr, true)
				s.mu.Unlock()
				return
			}
		}
		currentRevision := s.revision
		s.mu.Unlock()
		reply.fail(http.StatusUnprocessableEntity, "capture_response_rejected", "the typed response is not valid for the current browser checkpoint", false, currentRevision)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	reply.ETag = s.etag
	reply.respond(http.StatusAccepted, s.responseLocked())
	return
}

func (app Application) CaptureCancel(ctx context.Context, request captureMutationRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.CaptureRevision != s.captureRevision || !captureActive(s.capture) || s.capture.State == "canceling" || s.captureSession == nil && s.captureCancel == nil {
		reply.fail(http.StatusConflict, "stale_capture_revision", "capture revision is stale or no capture is active", true, s.revision)
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
		reply.internal(s.revision, "/api/v4/capture/cancel", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusAccepted, s.responseLocked())
	return
}

func (app Application) CaptureStage(ctx context.Context, request captureMutationRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.Revision != s.revision || request.CaptureRevision != s.captureRevision {
		reply.fail(http.StatusConflict, "stale_revision", "authoring or capture revision is stale", true, s.revision)
		return
	}
	if s.browserContainmentFailedLocked() {
		reply.fail(http.StatusConflict, "capture_teardown_failed", "browser process-tree teardown was not confirmed; restart iCoT before staging a capture", false, s.revision)
		return
	}
	if s.capture == nil || !s.capture.ResultReady || s.captureResult == nil || s.captureAttestation == nil || s.prepareCapture == nil {
		reply.fail(http.StatusConflict, "capture_not_ready", "a completed reviewed capture is required before staging", false, s.revision)
		return
	}
	stager, ok := s.engine.(browserCaptureEngine)
	if !ok {
		reply.fail(http.StatusNotImplemented, "unsupported", "browser capture staging is unavailable", false, s.revision)
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
		reply.fail(http.StatusUnprocessableEntity, "capture_validation_failed", "completed browser capture was rejected", false, s.revision)
		return
	}
	if prepared.Candidate != nil && s.browserTransactions != nil {
		transaction, err := s.startCandidateTransactionLocked(ctx, prepared.Candidate.Transaction())
		if err != nil {
			s.capture = &CaptureState{State: "failed", Message: "The capture transaction could not be initialized.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
			s.captureResult, s.captureAttestation = nil, nil
			_ = s.updateRevisionLocked()
			reply.fail(http.StatusConflict, "capture_transaction_failed", "capture transaction could not be initialized", false, s.revision)
			return
		}
		s.captureCandidate = prepared.Candidate
		s.browserTransaction = &transaction
		s.capture = &CaptureState{State: "transaction_review", Message: "The isolated worker stopped. Review the exact authentication and capability transaction before authoring.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		s.captureResult = nil
		s.captureAttestation = nil
		if err := s.updateRevisionLocked(); err != nil {
			reply.internal(s.revision, "/api/v4/capture/stage", "revision", err, true)
			return
		}
		reply.ETag = s.etag
		reply.respond(http.StatusOK, s.responseLocked())
		return
	}
	snapshot, err := stager.StageBrowserCapture(ctx, prepared)
	if err != nil {
		s.capture = &CaptureState{State: "failed", Message: "The reviewed capture could not be staged; no partial capture was adopted.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
		s.captureResult = nil
		s.captureAttestation = nil
		s.captureSession = nil
		s.refreshWorkspaceAfterFailure()
		_ = s.updateRevisionLocked()
		reply.engineError(s.revision, "/api/v4/capture/stage", "stage_capture", err)
		return
	}

	s.snapshot = snapshot
	s.capture = &CaptureState{State: "staged", Message: "The canonical profile pair and safe capture review were staged. Continue normal authoring review.", StartedAt: startedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)}
	s.captureResult = nil
	s.captureAttestation = nil
	s.captureSession = nil
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/capture/stage", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) Round(ctx context.Context, request roundRequest) (reply applicationResult) {
	s := app.server
	if strings.TrimSpace(request.Revision) == "" {
		reply.fail(http.StatusBadRequest, "malformed_request", "revision is required", false, s.currentRevision())
		return
	}
	if request.Answers == nil {
		reply.fail(http.StatusBadRequest, "malformed_request", "answers is required", false, s.currentRevision())
		return
	}
	answers := make([]authoring.RoundAnswer, len(request.Answers))
	for i, answer := range request.Answers {
		questionID := strings.TrimSpace(answer.QuestionID)
		if questionID == "" {
			reply.fail(http.StatusBadRequest, "malformed_request", "every answer requires question_id", false, s.currentRevision())
			return
		}
		value := answer.Value
		if answer.Deferral != nil {
			var err error
			value, err = encodeRoundDeferral(answer.Value, *answer.Deferral)
			if err != nil {
				reply.question(http.StatusBadRequest, "malformed_request", err.Error(), false, s.currentRevision(), questionID)
				return
			}
		}
		answers[i] = authoring.RoundAnswer{QuestionID: questionID, Value: value, Source: humanInputSource}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !app.beginMutation(ctx, &reply, "/api/v4/round", request.Revision) {
		return
	}
	snapshot, err := s.engine.ApplyRound(ctx, answers)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		reply.engineError(s.revision, "/api/v4/round", "apply_round", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/round", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) Reopen(ctx context.Context, request reopenRequest) (reply applicationResult) {
	s := app.server
	request.Revision = strings.TrimSpace(request.Revision)
	request.QuestionID = strings.TrimSpace(request.QuestionID)
	if request.Revision == "" {
		reply.fail(http.StatusBadRequest, "malformed_request", "revision is required", false, s.currentRevision())
		return
	}
	if request.QuestionID == "" {
		reply.fail(http.StatusBadRequest, "malformed_request", "question_id is required", false, s.currentRevision())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !app.beginMutation(ctx, &reply, "/api/v4/reopen", request.Revision) {
		return
	}
	snapshot, err := s.engine.ReopenDecision(ctx, request.QuestionID)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		reply.engineError(s.revision, "/api/v4/reopen", "reopen_decision", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/reopen", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) Approve(ctx context.Context, request approveRequest) (reply applicationResult) {
	s := app.server
	if strings.TrimSpace(request.Revision) == "" {
		reply.fail(http.StatusBadRequest, "malformed_request", "revision is required", false, s.currentRevision())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !app.beginMutation(ctx, &reply, "/api/v4/author/approve", request.Revision) {
		return
	}
	result, err := s.engine.ApproveAndWrite(ctx, engine.Approval{
		HumanApproved: request.HumanApproved, AllowOverwrite: request.AllowOverwrite, ApproveIncomplete: request.ApproveIncomplete,
	})
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		reply.engineError(s.revision, "/api/v4/author/approve", "approve", err)
		return
	}
	s.lifecycle = lifecycleAuthored
	s.completed = false
	s.snapshot = result.Snapshot
	s.writeResult = &result.WriteResult
	s.packageState = nil
	s.artifactPaths = map[string]string{}
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/author/approve", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) Resume(ctx context.Context, request revisionRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(ctx); err != nil {
		reply.internal(s.revision, "/api/v4/author/resume", "workspace_inspection", err, true)
		return
	}
	if strings.TrimSpace(request.Revision) != s.revision {
		reply.fail(http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, s.revision)
		return
	}
	if s.browserContainmentFailedLocked() {
		reply.fail(http.StatusConflict, "capture_teardown_failed", "browser process-tree teardown was not confirmed; restart iCoT before resuming authoring", false, s.revision)
		return
	}
	if s.lifecycle != lifecyclePackageFail {
		reply.fail(http.StatusConflict, "invalid_lifecycle", "authoring can resume only after a package quality failure", false, s.revision)
		return
	}
	resumer, ok := s.engine.(resumeEngine)
	if !ok {
		reply.fail(http.StatusNotImplemented, "unsupported", "authoring resume is unavailable", false, s.revision)
		return
	}
	snapshot, err := resumer.ResumeAuthoring(ctx)
	if err != nil {
		reply.engineError(s.revision, "/api/v4/author/resume", "resume_authoring", err)
		return
	}
	s.snapshot = snapshot
	s.lifecycle = lifecycleAuthoring
	s.completed = false
	s.writeResult = nil
	s.packageState = nil
	s.artifactPaths = map[string]string{}
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/author/resume", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) PackageBuild(ctx context.Context, request buildRequest) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(ctx); err != nil {
		reply.internal(s.revision, "/api/v4/package/build", "workspace_inspection", err, true)
		return
	}
	if strings.TrimSpace(request.Revision) != s.revision {
		reply.fail(http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, s.revision)
		return
	}
	if captureActive(s.capture) {
		reply.fail(http.StatusConflict, "capture_active", "package build is blocked while browser capture is active", true, s.revision)
		return
	}
	if s.browserContainmentFailedLocked() {
		reply.fail(http.StatusConflict, "capture_teardown_failed", "package build is blocked because browser process-tree teardown was not confirmed; restart iCoT", false, s.revision)
		return
	}
	if s.lifecycle != lifecycleAuthored {
		reply.fail(http.StatusConflict, "invalid_lifecycle", "a separately approved authored state is required before package build", false, s.revision)
		return
	}
	if !request.Confirmed {
		reply.fail(http.StatusUnprocessableEntity, "confirmation_required", "explicit package-build confirmation is required", false, s.revision)
		return
	}
	buildCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	result, buildReport, err := s.buildPackage(buildCtx, synthesize.Options{ExampleDir: s.exampleDir, LocalOnlyDiscovery: true})
	if err != nil {
		reply.internal(s.revision, "/api/v4/package/build", "deterministic_build", err, true)
		return
	}
	if buildReport != nil && !buildReport.Passed() {
		s.lifecycle = lifecyclePackageFail
		s.completed = false
		s.packageState = &PackageState{
			Status:      "failed",
			Quality:     &PackageQuality{Status: buildReport.Status, Checks: append([]synthesize.QualityCheck(nil), buildReport.Checks...)},
			Remediation: packageRemediation(buildReport),
		}
		s.artifactPaths = map[string]string{}
		if err := s.updateRevisionLocked(); err != nil {
			reply.internal(s.revision, "/api/v4/package/build", "revision", err, true)
			return
		}
		reply.ETag = s.etag
		reply.respond(http.StatusOK, s.responseLocked())
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
		reply.internal(s.revision, "/api/v4/package/build", "current_state_assessment", assessmentErr, true)
		return
	}
	if report == nil {
		if inspectionErr == nil {
			inspectionErr = errors.New("assessment returned no quality report")
		}
		reply.internal(s.revision, "/api/v4/package/build", "current_state_assessment", inspectionErr, true)
		return
	}
	quality := &PackageQuality{Status: report.Status, Checks: append([]synthesize.QualityCheck(nil), report.Checks...)}
	if !report.Passed() {
		s.lifecycle = lifecyclePackageFail
		s.completed = false
		s.packageState = &PackageState{Status: "failed", Quality: quality, Remediation: packageRemediation(report)}
		s.artifactPaths = map[string]string{}
		if err := s.updateRevisionLocked(); err != nil {
			reply.internal(s.revision, "/api/v4/package/build", "revision", err, true)
			return
		}
		reply.ETag = s.etag
		reply.respond(http.StatusOK, s.responseLocked())
		return
	}
	if inspectionErr != nil {
		reply.internal(s.revision, "/api/v4/package/build", "package_inspection", inspectionErr, true)
		return
	}
	artifacts, paths, err := inspectAllowedArtifacts(s.exampleDir, result)
	if err != nil {
		reply.internal(s.revision, "/api/v4/package/build", "artifact_allowlist", err, true)
		return
	}
	if err := s.revalidatePackage(buildCtx, trustedrunner.TemplateOptions{RepoRoot: s.repoRoot, ExampleDir: s.exampleDir}, inspection); err != nil {
		reply.internal(s.revision, "/api/v4/package/build", "package_freeze_revalidation", err, true)
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
		reply.internal(s.revision, "/api/v4/package/build", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) Journey(ctx context.Context, request journeyRequest) (reply applicationResult) {
	s := app.server
	selected, ok := s.engine.(journeyEngine)
	if !ok {
		reply.fail(http.StatusNotImplemented, "unsupported", "journey selection is unavailable", false, s.currentRevision())
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !app.beginMutation(ctx, &reply, "/api/v4/journey", strings.TrimSpace(request.Revision)) {
		return
	}
	snapshot, err := selected.SelectJourney(ctx, request.Starter, request.Goal)
	if err != nil {
		s.refreshWorkspaceAfterFailure()
		reply.engineError(s.revision, "/api/v4/journey", "select_journey", err)
		return
	}
	s.snapshot = snapshot
	if err := s.updateRevisionLocked(); err != nil {
		reply.internal(s.revision, "/api/v4/journey", "revision", err, true)
		return
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}

func (app Application) beginMutation(ctx context.Context, reply *applicationResult, route, requestRevision string) bool {
	s := app.server
	if err := s.refreshWorkspaceLocked(ctx); err != nil {
		reply.internal(s.revision, route, "workspace_inspection", err, true)
		return false
	}
	if s.workspace.ExternallyModified {
		reply.fail(http.StatusConflict, "workspace_changed", "the authoring workspace changed outside this process; restart is required", false, s.revision)
		return false
	}
	if s.browserContainmentFailedLocked() {
		reply.fail(http.StatusConflict, "capture_teardown_failed", "browser process-tree teardown was not confirmed; restart iCoT before authoring continues", false, s.revision)
		return false
	}
	if captureActive(s.capture) {
		reply.fail(http.StatusConflict, "capture_active", "authoring mutations are blocked while browser capture is active", true, s.revision)
		return false
	}
	if registrationAuthoringActive(s.registrationAuthoring) {
		reply.fail(http.StatusConflict, "registration_authoring_active", "authoring mutations are blocked while browser registration authoring is active", true, s.revision)
		return false
	}
	if s.lifecycle != lifecycleAuthoring {
		reply.fail(http.StatusConflict, "session_frozen", "authoring is not mutable in the current lifecycle state", false, s.revision)
		return false
	}
	if requestRevision != s.revision {
		reply.fail(http.StatusConflict, "stale_revision", "request revision does not match the current snapshot", true, s.revision)
		return false
	}
	return true
}

// Snapshot refreshes workspace and immutable package evidence for either adapter.
func (app Application) Snapshot(ctx context.Context) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.refreshWorkspaceLocked(ctx); err != nil {
		reply.internal(s.revision, "/api/v4/snapshot", "workspace_inspection", err, true)
		return
	}
	if s.lifecycle == lifecycleHandoffReady {
		if err := s.validateFrozenArtifactsLocked(ctx); err != nil {
			s.invalidateHandoffLocked()
			if err := s.updateRevisionLocked(); err != nil {
				reply.internal(s.revision, "/api/v4/snapshot", "revision", err, true)
				return
			}
		}
	}
	reply.ETag = s.etag
	reply.respond(http.StatusOK, s.responseLocked())
	return
}
