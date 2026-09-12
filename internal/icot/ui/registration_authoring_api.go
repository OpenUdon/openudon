package ui

import (
	"context"
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
	ProfileVersion       string                            `json:"profile_version,omitempty"`
}

type registrationAuthoringCommandRequest struct {
	Revision             string                                    `json:"revision"`
	RegistrationRevision string                                    `json:"registration_revision"`
	Type                 string                                    `json:"type"`
	Method               string                                    `json:"method,omitempty"`
	URL                  string                                    `json:"url,omitempty"`
	Confirmed            bool                                      `json:"confirmed,omitempty"`
	Draft                *registrationDraftRequest                 `json:"draft,omitempty"`
	Preview              *registrationauthorsession.PreviewRequest `json:"preview,omitempty"`
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
	s.writeRegistrationResult(w, requestID, (RegistrationApplication{server: s}).Start(r.Context(), request))
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
			History:     cloneRegistrationPublic(event.History), Previews: cloneRegistrationPublic(event.Previews),
			Draft:     cloneRegistrationDraftDisclosure(s.registrationAuthoring.Draft),
			StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339),
		}
		if event.ErrorCode != "" {
			state.FailureCode = registrationAuthoringFailureCode(event.ErrorCode)
			state.Message = registrationAuthoringErrorMessage(state.FailureCode)
		}
		if event.State == "ready" {
			state.State = "starting"
			state.Message = "The isolated worker is ready; opening the reviewed initial navigation."
			s.setRegistrationAuthoringLocked(state)
			_ = s.updateRevisionLocked()
			s.mu.Unlock()
			ctx, cancel := context.WithTimeout(s.captureContext, 5*time.Second)
			err := session.Send(ctx, start)
			cancel()
			if err != nil {
				session.Cancel()
				s.mu.Lock()
				if s.registrationSession == session {
					s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: "canceling", FailureCode: "start_rejected", Message: "The registration worker rejected its fixed start command; waiting for teardown.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: s.now().UTC().Format(time.RFC3339)})
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
			terminalState, terminalCode = event.State, registrationAuthoringFailureCode(event.ErrorCode)
			state.State = "canceling"
			state.Message = "The registration worker stopped; waiting for process-tree teardown to complete."
		}
		s.setRegistrationAuthoringLocked(state)
		_ = s.updateRevisionLocked()
		s.mu.Unlock()
	}
	terminalEvent, terminalEventSet := session.TerminalEvent()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.registrationSession != session {
		return
	}
	if terminalEventSet {
		if terminalEvent.Candidate != nil {
			s.registrationCandidate = terminalEvent.Candidate
			resultReady = true
		}
		if terminalEvent.State == "failed" || terminalEvent.State == "canceled" {
			terminalState = terminalEvent.State
			terminalCode = registrationAuthoringFailureCode(terminalEvent.ErrorCode)
		}
	}
	s.registrationSession = nil
	updatedAt := s.now().UTC().Format(time.RFC3339)
	if terminalState != "" || s.registrationAuthoring != nil && s.registrationAuthoring.State == "canceling" {
		state := terminalState
		if state == "" {
			state = "canceled"
		}
		failureCode := registrationAuthoringFailureCode(terminalCode)
		message := registrationAuthoringTerminalMessage(state, failureCode)
		containmentFailed := failureCode == "worker_teardown"
		if containmentFailed {
			s.registrationContainmentFailed = true
		}
		s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: state, Message: message, FailureCode: failureCode, ContainmentFailed: containmentFailed, StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt})
		s.registrationCandidate = nil
		s.clearRegistrationDraftLocked()
		_ = s.updateRevisionLocked()
		return
	}
	if resultReady && s.registrationCandidate != nil {
		draft := cloneRegistrationDraftDisclosure(s.registrationAuthoring.Draft)
		transactionSnapshot, err := s.startRegistrationTransactionLocked(s.captureContext, s.registrationCandidate)
		if err != nil {
			s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: "failed", FailureCode: "transaction_start_failed", Message: "The adopted registration candidate could not enter the exact browser transaction lifecycle. This iCoT process has consumed its registration-authoring attempt; a later session requires a fresh preflight, authorization, and process.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt})
			s.registrationCandidate = nil
			s.clearRegistrationDraftLocked()
			_ = s.updateRevisionLocked()
			return
		}
		s.browserTransaction = &transactionSnapshot
		s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
			State: "transaction_review", Message: "The worker stopped cleanly and the exact transaction-v2 candidate is ready for separate browser transaction review.",
			ResultReady: true, Draft: draft, StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt,
		})
		s.clearRegistrationDraftLocked()
		_ = s.updateRevisionLocked()
		return
	}
	s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{State: "failed", FailureCode: "candidate_missing", Message: "The registration worker ended without a promotable candidate. This iCoT process has consumed its registration-authoring attempt; a later session requires a fresh preflight, authorization, and process.", StartedAt: startedAt.Format(time.RFC3339), UpdatedAt: updatedAt})
	s.registrationCandidate = nil
	s.clearRegistrationDraftLocked()
	_ = s.updateRevisionLocked()
}

func (s *Server) startRegistrationTransactionLocked(ctx context.Context, candidate *browsercandidate.Registration) (transactionengine.Snapshot, error) {
	if s.browserTransactions == nil || candidate == nil {
		return transactionengine.Snapshot{}, errors.New("registration transaction lifecycle is unavailable")
	}
	return s.startCandidateTransactionLocked(ctx, candidate.Transaction())
}

func (s *Server) startCandidateTransactionLocked(ctx context.Context, transaction browsertransaction.Transaction) (transactionengine.Snapshot, error) {
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
	s.setRegistrationAuthoringLocked(&RegistrationAuthoringState{
		State: "adopted", Message: "The explicitly reviewed transaction-v2 candidate is selected as a virtual source. Complete authoring, write, and build the package before preparation.",
		Draft: draft, StartedAt: s.registrationAuthoring.StartedAt, UpdatedAt: s.now().UTC().Format(time.RFC3339),
	})
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
	s.writeRegistrationResult(w, requestID, (RegistrationApplication{server: s}).Command(r.Context(), request))
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
	s.writeRegistrationResult(w, requestID, (RegistrationApplication{server: s}).Cancel(r.Context(), request))
}

func registrationAuthoringWireCommand(request registrationAuthoringCommandRequest) (browserauthor.RegistrationCommand, bool) {
	typeName := strings.TrimSpace(request.Type)
	method := strings.TrimSpace(request.Method)
	url := strings.TrimSpace(request.URL)
	if typeName != "preview" && request.Preview != nil {
		return browserauthor.RegistrationCommand{}, false
	}
	switch typeName {
	case "preview":
		if method != "" || url != "" || !request.Confirmed || request.Draft != nil || request.Preview == nil {
			return browserauthor.RegistrationCommand{}, false
		}
		return browserauthor.RegistrationCommand{Type: "preview", Confirmed: true, Preview: cloneRegistrationPublic(request.Preview)}, true
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
	case "navigate", "preview":
		return state.State == "observation"
	case "review":
		return state.State == "draft_review"
	case "finish":
		return state.State == "reviewed"
	default:
		return false
	}
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
	copy.StepCandidates = append([]string(nil), value.StepCandidates...)
	copy.CredentialBindings = append([]browsertransaction.CredentialBinding(nil), value.CredentialBindings...)
	copy.RetainedQueries = append([]RetainedQueryDisclosure(nil), value.RetainedQueries...)
	for index := range copy.RetainedQueries {
		copy.RetainedQueries[index].Parameters = append([]RetainedQueryParameter(nil), value.RetainedQueries[index].Parameters...)
	}
	return &copy
}

func (s *Server) setRegistrationAuthoringLocked(state *RegistrationAuthoringState) {
	if state != nil {
		if state.Observation != nil && len(state.History) != 0 {
			state.Suggestions = registrationauthorsession.SuggestFields(*state.Observation)
		}
		if s.registrationAuthoring != nil && registrationAuthoringActive(state) && state.History == nil {
			state.History = cloneRegistrationPublic(s.registrationAuthoring.History)
			state.Previews = cloneRegistrationPublic(s.registrationAuthoring.Previews)
		}
		state.FailureCode = registrationAuthoringFailureCode(state.FailureCode)
		state.AttemptConsumed = s.registrationAttemptConsumed
	}
	s.registrationAuthoring = state
}

func registrationAuthoringFailureCode(code string) string {
	switch strings.TrimSpace(code) {
	case "":
		return ""
	case "candidate_adoption_failed", "candidate_missing", "candidate_rejected",
		"invalid_response", "malformed_diagnostic", "operator_idle_timeout",
		"protocol_mismatch", "protocol_negotiation", "review_missing",
		"start_rejected", "transaction_start_failed", "worker_exit",
		"worker_failed", "worker_protocol", "worker_start_failed",
		"worker_teardown", "worker_write":
		return strings.TrimSpace(code)
	default:
		return "worker_failed"
	}
}

func registrationAuthoringTerminalMessage(state, code string) string {
	const consumed = " This iCoT process has consumed its registration-authoring attempt; a later session requires a fresh preflight, authorization, and process."
	if code == "worker_teardown" {
		return "Registration authoring failed closed because clean worker-descendant teardown could not be confirmed; no candidate is available." + consumed
	}
	if state == "canceled" {
		message := "Registration authoring was canceled after the worker and all descendants stopped; no candidate was adopted."
		if code != "" {
			message = registrationAuthoringErrorMessage(code) + " The worker and all descendants then stopped; no candidate was adopted."
		}
		return message + consumed
	}
	message := "Registration authoring failed closed after the worker and all descendants stopped; no candidate is available."
	if code != "" {
		message = registrationAuthoringErrorMessage(code) + " The worker and all descendants then stopped; no candidate is available."
	}
	return message + consumed
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
	return cloneRegistrationPublic(value)
}

// Clone nested public definitions so transport snapshots cannot mutate authority.
func cloneRegistrationPublic[T any](value T) T {
	data, _ := json.Marshal(value)
	var result T
	_ = json.Unmarshal(data, &result)
	return result
}
