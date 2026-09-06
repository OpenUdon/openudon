package ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	icotengine "github.com/OpenUdon/openudon/internal/icot/engine"
)

func decodeApplicationTransaction(raw json.RawMessage, target any, reply *applicationResult, snapshot *transactionengine.Snapshot) bool {
	if evidencefile.DecodeStrict(raw, target) == nil {
		return true
	}
	revision := ""
	if snapshot != nil {
		revision = snapshot.Revision
	}
	reply.fail(http.StatusBadRequest, "malformed_request", "browser transaction request is malformed", false, revision)
	return false
}
func (reply *applicationResult) transactionError(snapshot transactionengine.Snapshot, cause error) {
	class, code, _, retryable, ok := transactionengine.ErrorDetails(cause)
	if !ok {
		reply.fail(http.StatusInternalServerError, "browser_transaction_failed", "browser transaction operation failed", false, snapshot.Revision)
		return
	}
	status := http.StatusUnprocessableEntity
	if class == browsertransaction.FailureConflict || class == browsertransaction.FailureIndeterminate {
		status = http.StatusConflict
	} else if class == browsertransaction.FailureOperational {
		status = http.StatusInternalServerError
	}
	if code == transactionengine.ErrorCanceled {
		status = http.StatusRequestTimeout
	}
	reply.fail(status, string(code), cause.Error(), retryable, snapshot.Revision)
}

// Transaction preserves the UI's registration candidate adoption and package
// readiness checks in the same process that owns the private candidate.
func (app Application) Transaction(ctx context.Context, route string, raw json.RawMessage) (reply applicationResult) {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.browserTransactions == nil {
		reply.fail(http.StatusServiceUnavailable, "browser_transactions_unavailable", "browser transaction resources are not configured", false, "")
		return
	}
	if s.browserContainmentFailedLocked() {
		reply.fail(http.StatusConflict, "capture_teardown_failed", "browser process-tree teardown was not confirmed; restart iCoT before changing a browser transaction", false, s.revision)
		return
	}
	captureReviewOperation := s.capture != nil && s.capture.State == "transaction_review" &&
		(route == "/api/v4/browser-transactions/review" || route == "/api/v4/browser-transactions/cancel")
	if captureActive(s.capture) && !captureReviewOperation {
		reply.fail(http.StatusConflict, "capture_active", "browser transaction mutations are blocked while capture is active", true, s.revision)
		return
	}
	if route == "/api/v4/browser-transactions/prepare" && s.capture != nil && s.capture.State == "staged" && s.lifecycle != lifecycleHandoffReady {
		reply.fail(http.StatusConflict, "capture_package_not_ready", "write and build the reviewed capture package before transaction preparation", false, s.revision)
		return
	}
	registrationReviewOperation := s.registrationAuthoring != nil && s.registrationAuthoring.State == "transaction_review" &&
		(route == "/api/v4/browser-transactions/review" || route == "/api/v4/browser-transactions/cancel")
	if registrationAuthoringActive(s.registrationAuthoring) && !registrationReviewOperation {
		reply.fail(http.StatusConflict, "registration_authoring_active", "browser transaction mutations are blocked while registration authoring is active", true, s.revision)
		return
	}
	if route == "/api/v4/browser-transactions/prepare" && s.registrationAuthoring != nil && s.registrationAuthoring.State == "adopted" && s.lifecycle != lifecycleHandoffReady {
		reply.fail(http.StatusConflict, "registration_package_not_ready", "write and build the reviewed registration package before transaction preparation", false, s.revision)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, browserTransactionTimeout(route))
	defer cancel()
	var snapshot transactionengine.Snapshot
	var err error
	switch route {
	case "/api/v4/browser-transactions/start":
		var request transactionengine.StartRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.Start(ctx, request)
	case "/api/v4/browser-transactions/review":
		var request transactionengine.ReviewRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.Review(ctx, request)
	case "/api/v4/browser-transactions/prepare":
		var request transactionengine.PrepareRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.Prepare(ctx, request)
	case "/api/v4/browser-transactions/promote":
		var request transactionengine.PromoteRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.Promote(ctx, request)
	case "/api/v4/browser-transactions/cancel":
		var request transactionengine.CancelRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.Cancel(ctx, request)
	case "/api/v4/browser-transactions/recovery/inspect":
		var request transactionengine.InspectRecoveryRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.InspectRecovery(ctx, request)
	case "/api/v4/browser-transactions/recovery/reconcile":
		var request transactionengine.RecoverRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.Recover(ctx, request)
	case "/api/v4/browser-transactions/selected/inspect":
		var request transactionengine.InspectSelectedRequest
		if !decodeApplicationTransaction(raw, &request, &reply, s.browserTransaction) {
			return
		}
		snapshot, err = s.browserTransactions.InspectSelected(ctx, request)
	default:
		reply.fail(http.StatusNotFound, "not_found", "route not found", false, "")
		return
	}
	if snapshot.Version != "" {
		s.browserTransaction = &snapshot
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			reply.fail(http.StatusInternalServerError, "internal_error", "browser transaction resource could not be versioned", true, snapshot.Revision)
			return
		}
	}
	if err != nil {
		reply.transactionError(snapshot, err)
		return
	}
	if route == "/api/v4/browser-transactions/review" && s.captureCandidate != nil {
		if err := s.adoptReviewedCaptureLocked(ctx, snapshot); err != nil {
			s.capture = &CaptureState{State: "failed", Message: "The reviewed authentication candidate could not be adopted."}
			s.captureCandidate = nil
			_ = s.updateRevisionLocked()
			reply.fail(http.StatusConflict, "capture_candidate_adoption_failed", "reviewed capture could not enter authoring", false, s.revision)
			return
		}
		if err := s.updateRevisionLocked(); err != nil {
			reply.internal(s.revision, route, "revision", err, true)
			return
		}
	}
	if route == "/api/v4/browser-transactions/cancel" && s.captureCandidate != nil {
		s.captureCandidate = nil
		s.capture = &CaptureState{State: "canceled", Message: "The pending capture transaction was canceled."}
		if err := s.updateRevisionLocked(); err != nil {
			reply.internal(s.revision, route, "revision", err, true)
			return
		}
	}
	if route == "/api/v4/browser-transactions/review" && s.registrationCandidate != nil {
		adoptionCtx, cancelAdoption := context.WithTimeout(ctx, 15*time.Second)
		adoptionErr := s.adoptReviewedRegistrationLocked(adoptionCtx, snapshot)
		cancelAdoption()
		if adoptionErr != nil {
			s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: "failed", FailureCode: "candidate_adoption_failed", Message: "The reviewed transaction could not be adopted into the authoring source catalog. This iCoT process has consumed its registration-authoring attempt; a later session requires a fresh preflight, authorization, and process.", StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)})
			s.registrationCandidate = nil
			_ = s.updateRevisionLocked()
			reply.fail(http.StatusConflict, "registration_candidate_adoption_failed", "the reviewed registration candidate could not enter the authoring source catalog", false, s.revision)
			return
		}
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			reply.fail(http.StatusInternalServerError, "internal_error", "registration adoption could not be versioned", true, s.revision)
			return
		}
	}
	if route == "/api/v4/browser-transactions/cancel" && s.registrationAuthoring != nil && s.registrationAuthoring.State == "transaction_review" {
		s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: "canceled", Message: "The pending reviewed candidate and browser transaction were canceled. This iCoT process has consumed its registration-authoring attempt; a later session requires a fresh preflight, authorization, and process.", StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)})
		s.registrationCandidate = nil
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			reply.fail(http.StatusInternalServerError, "internal_error", "registration cancellation could not be versioned", true, s.revision)
			return
		}
	}
	if route == "/api/v4/browser-transactions/promote" && s.registrationAuthoring != nil && s.registrationAuthoring.State == "adopted" {
		s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: "promoted", Message: "The exact qualified registration package generation was promoted without runtime execution.", StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339)})
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			reply.fail(http.StatusInternalServerError, "internal_error", "registration promotion could not be versioned", true, s.revision)
			return
		}
	}
	reply.ETag = snapshot.Revision
	reply.respond(http.StatusOK, browserTransactionResponse{Version: APIVersion, Transaction: browserTransactionResource(snapshot)})
	return
}

func (s *Server) adoptReviewedCaptureLocked(ctx context.Context, snapshot transactionengine.Snapshot) error {
	if s.captureCandidate == nil || snapshot.Transaction == nil {
		return errors.New("capture candidate missing")
	}
	expected, err := s.captureCandidate.ReviewedTransaction()
	if err != nil {
		return err
	}
	digest, err := browsertransaction.Digest(expected)
	if err != nil || digest != snapshot.TransactionSHA256 {
		return errors.New("capture candidate changed")
	}
	actual, err := browsertransaction.Digest(*snapshot.Transaction)
	if err != nil || actual != digest {
		return errors.New("capture transaction changed")
	}
	author, ok := s.engine.(registrationVirtualBrowserEngine)
	if !ok {
		return errors.New("virtual sources unavailable")
	}
	input, err := icotengine.AuthenticationCapabilityVirtualBrowserTransaction(s.captureCandidate, true)
	if err != nil {
		return err
	}
	replaced, err := author.ReplaceVirtualBrowserSources(ctx, s.snapshot.SourceCandidates.VirtualBrowser.Generation, []elicitor.VirtualBrowserTransactionInput{input})
	if err != nil {
		return err
	}
	var ids []string
	for _, candidate := range replaced.SourceCandidates.VirtualBrowser.Candidates {
		if candidate.TransactionID == expected.ID && (candidate.Kind == browsertransaction.CandidateAuthentication || candidate.Kind == browsertransaction.CandidateCapability) {
			ids = append(ids, candidate.ID)
		}
	}
	if len(ids) != 2 {
		return errors.New("capture source composition")
	}
	selected, err := author.SelectVirtualBrowserSources(ctx, replaced.SourceCandidates.VirtualBrowser.Generation, ids)
	if err != nil {
		return err
	}
	s.snapshot = selected
	s.captureCandidate = nil
	s.capture = &CaptureState{State: "staged", Message: "The reviewed authentication and capability pair is selected for ordinary authoring."}
	return nil
}
