package ui

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/browsertransaction/presentation"
)

type BrowserTransactionSnapshot = transactionengine.Snapshot

// BrowserTransactionResource augments the exact engine snapshot only with
// deterministic, value-free review disclosures. It does not change the
// browser-profile transaction v1 wire nested inside the snapshot.
type BrowserTransactionResource = presentation.Resource
type BrowserTransactionReview = presentation.Review
type RegistrationAuthoringDisclosure = presentation.RegistrationAuthoringDisclosure

// BrowserTransactionEngine is the one shared lifecycle used by API and
// terminal adapters. No browser or runtime operation is part of the contract.
type BrowserTransactionEngine interface {
	Observe(context.Context) (transactionengine.Snapshot, error)
	Start(context.Context, transactionengine.StartRequest) (transactionengine.Snapshot, error)
	Review(context.Context, transactionengine.ReviewRequest) (transactionengine.Snapshot, error)
	Prepare(context.Context, transactionengine.PrepareRequest) (transactionengine.Snapshot, error)
	Promote(context.Context, transactionengine.PromoteRequest) (transactionengine.Snapshot, error)
	Cancel(context.Context, transactionengine.CancelRequest) (transactionengine.Snapshot, error)
	InspectRecovery(context.Context, transactionengine.InspectRecoveryRequest) (transactionengine.Snapshot, error)
	Recover(context.Context, transactionengine.RecoverRequest) (transactionengine.Snapshot, error)
	InspectSelected(context.Context, transactionengine.InspectSelectedRequest) (transactionengine.Snapshot, error)
}

type browserTransactionResponse struct {
	Version     string                     `json:"version"`
	Transaction BrowserTransactionResource `json:"browser_transaction"`
}

func (s *Server) serveBrowserTransactionCurrent(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID string) {
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
	if s.browserTransactions == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser_transactions_unavailable", "browser transaction resources are not configured", false, requestID, "")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	snapshot, err := s.browserTransactions.Observe(ctx)
	if err != nil {
		s.writeBrowserTransactionError(w, requestID, snapshot, err)
		return
	}
	s.browserTransaction = &snapshot
	if err := s.updateRevisionLocked(); err != nil {
		s.writeError(w, http.StatusInternalServerError, "internal_error", "browser transaction resource could not be versioned", true, requestID, snapshot.Revision)
		return
	}
	setETag(w, snapshot.Revision)
	s.writeJSON(w, http.StatusOK, browserTransactionResponse{Version: APIVersion, Transaction: browserTransactionResource(snapshot)}, requestID)
}

func (s *Server) serveBrowserTransactionMutation(w http.ResponseWriter, r *http.Request, cookieScoped bool, requestID, route string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w, http.MethodPost, requestID)
		return
	}
	if !s.authenticated(r, cookieScoped) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "a valid UI capability token is required", false, requestID, "")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.browserTransactions == nil {
		s.writeError(w, http.StatusServiceUnavailable, "browser_transactions_unavailable", "browser transaction resources are not configured", false, requestID, "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), browserTransactionTimeout(route))
	defer cancel()
	var snapshot transactionengine.Snapshot
	var err error
	switch route {
	case "/api/v4/browser-transactions/start":
		var request transactionengine.StartRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.Start(ctx, request)
	case "/api/v4/browser-transactions/review":
		var request transactionengine.ReviewRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.Review(ctx, request)
	case "/api/v4/browser-transactions/prepare":
		var request transactionengine.PrepareRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.Prepare(ctx, request)
	case "/api/v4/browser-transactions/promote":
		var request transactionengine.PromoteRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.Promote(ctx, request)
	case "/api/v4/browser-transactions/cancel":
		var request transactionengine.CancelRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.Cancel(ctx, request)
	case "/api/v4/browser-transactions/recovery/inspect":
		var request transactionengine.InspectRecoveryRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.InspectRecovery(ctx, request)
	case "/api/v4/browser-transactions/recovery/reconcile":
		var request transactionengine.RecoverRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.Recover(ctx, request)
	case "/api/v4/browser-transactions/selected/inspect":
		var request transactionengine.InspectSelectedRequest
		if !s.decodeBrowserTransactionRequest(w, r, requestID, &request) {
			return
		}
		snapshot, err = s.browserTransactions.InspectSelected(ctx, request)
	default:
		s.writeError(w, http.StatusNotFound, "not_found", "route not found", false, requestID, "")
		return
	}
	if snapshot.Version != "" {
		s.browserTransaction = &snapshot
		if revisionErr := s.updateRevisionLocked(); revisionErr != nil {
			s.writeError(w, http.StatusInternalServerError, "internal_error", "browser transaction resource could not be versioned", true, requestID, snapshot.Revision)
			return
		}
	}
	if err != nil {
		s.writeBrowserTransactionError(w, requestID, snapshot, err)
		return
	}
	setETag(w, snapshot.Revision)
	s.writeJSON(w, http.StatusOK, browserTransactionResponse{Version: APIVersion, Transaction: browserTransactionResource(snapshot)}, requestID)
}

func (s *Server) decodeBrowserTransactionRequest(w http.ResponseWriter, r *http.Request, requestID string, target any) bool {
	if err := decodeJSONRequest(w, r, target); err != nil {
		var requestErr *requestError
		if !errors.As(err, &requestErr) {
			requestErr = &requestError{status: http.StatusBadRequest, code: "malformed_request", text: "browser transaction request is malformed"}
		}
		revision := ""
		if s.browserTransaction != nil {
			revision = s.browserTransaction.Revision
		}
		s.writeError(w, requestErr.status, requestErr.code, requestErr.text, false, requestID, revision)
		return false
	}
	return true
}

func (s *Server) writeBrowserTransactionError(w http.ResponseWriter, requestID string, snapshot transactionengine.Snapshot, cause error) {
	class, code, _, retryable, ok := transactionengine.ErrorDetails(cause)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "browser_transaction_failed", "browser transaction operation failed", false, requestID, snapshot.Revision)
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
	s.writeError(w, status, string(code), cause.Error(), retryable, requestID, snapshot.Revision)
}

func browserTransactionTimeout(route string) time.Duration {
	switch route {
	case "/api/v4/browser-transactions/prepare", "/api/v4/browser-transactions/promote", "/api/v4/browser-transactions/recovery/reconcile":
		return 2 * time.Minute
	case "/api/v4/browser-transactions/recovery/inspect", "/api/v4/browser-transactions/selected/inspect":
		return 30 * time.Second
	default:
		return 5 * time.Second
	}
}

func (s *Server) browserTransactionResourceLocked() *BrowserTransactionResource {
	if s.browserTransaction == nil {
		return nil
	}
	resource := browserTransactionResource(*s.browserTransaction)
	return &resource
}

func browserTransactionResource(snapshot transactionengine.Snapshot) BrowserTransactionResource {
	return presentation.New(snapshot)
}
