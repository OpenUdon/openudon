package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/authoring"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
	icotengine "github.com/OpenUdon/openudon/internal/icot/engine"
	"github.com/OpenUdon/openudon/internal/packagepipeline"
)

const (
	registrationQualificationAuthority = "127.0.0.1:43124"
	registrationQualificationToken     = "loopback-registration-qualification"
	registrationQualificationCode      = "0123456789AB"
)

// RegistrationQualificationOptions fixes the synthetic iCoT wizard and its
// package lifecycle. Browser contact remains confined to InitialURL/Origin.
type RegistrationQualificationOptions struct {
	RepoRoot               string
	BrowsertoolsExecutable string
	ExampleDir             string
	PrivateRoot            string
	ScratchParent          string
	StoreDir               string
	Scope                  string
	ProfileID              string
	InitialURL             string
	Origin                 string
	Now                    func() time.Time
}

// RegistrationQualificationResult retains only canonical public profile and
// value-free transaction/package evidence from the authenticated UI path.
type RegistrationQualificationResult struct {
	Snapshot         transactionengine.Snapshot
	CanonicalProfile json.RawMessage
	RetainedQuery    bool
}

// RunRegistrationQualification drives the same authenticated revision-bound
// endpoints as the guided iCoT browser-registration wizard. It does not grant
// runtime authority; the caller separately attests and invokes the promoted
// package after this function returns.
func RunRegistrationQualification(ctx context.Context, options RegistrationQualificationOptions) (RegistrationQualificationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	clock := func() time.Time { return time.Now().UTC().Truncate(time.Second) }
	if options.Now != nil {
		clock = func() time.Time { return options.Now().UTC().Truncate(time.Second) }
	}
	now := clock()
	if now.IsZero() || strings.TrimSpace(options.RepoRoot) == "" || strings.TrimSpace(options.BrowsertoolsExecutable) == "" ||
		strings.TrimSpace(options.ExampleDir) == "" || strings.TrimSpace(options.PrivateRoot) == "" || strings.TrimSpace(options.ScratchParent) == "" ||
		strings.TrimSpace(options.StoreDir) == "" || strings.TrimSpace(options.Scope) == "" || strings.TrimSpace(options.ProfileID) == "" ||
		strings.TrimSpace(options.InitialURL) == "" || strings.TrimSpace(options.Origin) == "" {
		return RegistrationQualificationResult{}, errors.New("registration qualification UI authority is invalid")
	}
	authoringEngine, authoringSnapshot, err := icotengine.Open(ctx, icotengine.Config{
		ExampleDir: options.ExampleDir, NetworkPolicy: "never", Now: clock,
	})
	if err != nil {
		return RegistrationQualificationResult{}, errors.New("open registration qualification authoring engine")
	}
	transactions, _, err := transactionengine.New(transactionengine.Config{
		Package: packagepipeline.CurrentOptions{
			ExampleDir: options.ExampleDir, Scope: options.Scope, ScratchParent: options.ScratchParent, StoreDir: options.StoreDir,
		},
		Now: clock,
	})
	if err != nil {
		return RegistrationQualificationResult{}, errors.New("open registration qualification transaction engine")
	}
	handler, err := NewHandler(HandlerConfig{
		Context: ctx, Engine: authoringEngine, Snapshot: authoringSnapshot, ExampleDir: options.ExampleDir,
		Token: registrationQualificationToken, AccessCode: registrationQualificationCode, Authority: registrationQualificationAuthority,
		PrivateRoot: options.PrivateRoot, Now: clock, BrowserTransactions: transactions,
		ErrOut: io.Discard, RepoRoot: options.RepoRoot,
		StartRegistration: func(startCtx context.Context, config browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return browserauthor.StartExternalRegistration(startCtx, config, options.BrowsertoolsExecutable)
		},
	})
	if err != nil {
		return RegistrationQualificationResult{}, errors.New("open registration qualification UI")
	}

	current, err := registrationQualificationCurrent(ctx, handler)
	if err != nil {
		return RegistrationQualificationResult{}, err
	}
	start := registrationAuthoringStartRequest{
		Revision: current.Revision, RegistrationRevision: current.RegistrationRevision,
		ProfileID: options.ProfileID, URL: options.InitialURL, Origins: []string{options.Origin},
	}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/start", start, http.StatusAccepted); err != nil {
		return RegistrationQualificationResult{}, errors.New("start registration qualification wizard")
	}
	observing, err := registrationQualificationWait(ctx, handler, "observing")
	if err != nil {
		return RegistrationQualificationResult{}, err
	}
	observe := registrationAuthoringCommandRequest{Revision: observing.Revision, RegistrationRevision: observing.RegistrationRevision, Type: "observe"}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", observe, http.StatusAccepted); err != nil {
		return RegistrationQualificationResult{}, errors.New("observe initial registration qualification page")
	}
	observed, err := registrationQualificationWait(ctx, handler, "observation")
	if err != nil {
		return RegistrationQualificationResult{}, err
	}
	head := registrationAuthoringCommandRequest{
		Revision: observed.Revision, RegistrationRevision: observed.RegistrationRevision,
		Type: "navigate", Method: http.MethodHead, URL: options.InitialURL,
	}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", head, http.StatusAccepted); err != nil {
		return RegistrationQualificationResult{}, errors.New("navigate registration qualification wizard")
	}
	observing, err = registrationQualificationWait(ctx, handler, "observing")
	if err != nil {
		return RegistrationQualificationResult{}, err
	}
	observe = registrationAuthoringCommandRequest{Revision: observing.Revision, RegistrationRevision: observing.RegistrationRevision, Type: "observe"}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", observe, http.StatusAccepted); err != nil {
		return RegistrationQualificationResult{}, errors.New("observe registration qualification wizard")
	}
	observed, err = registrationQualificationWait(ctx, handler, "observation")
	if err != nil || observed.RegistrationAuthoring == nil || observed.RegistrationAuthoring.Observation == nil {
		return RegistrationQualificationResult{}, errors.New("registration qualification observation is unavailable")
	}
	ids, err := registrationQualificationCandidateIDs(*observed.RegistrationAuthoring.Observation)
	if err != nil {
		return RegistrationQualificationResult{}, err
	}
	draft := registrationDraftRequest{
		Title: "Synthetic dedicated test registration", Provider: "Synthetic loopback", Confidence: "high", ExpiresAfter: "P30D",
		CredentialSlots: []registrationDraftSlot{
			{Slot: "identifier", Kind: "identifier", Binding: "registration_identifier"},
			{Slot: "password", Kind: "password", Binding: "reg_password"},
		},
		Flow: registrationDraftFlow{
			Name: "create_dedicated_test_user", Description: "Create one dedicated loopback test identity.",
			Steps: []registrationDraftStep{
				{Type: "navigate", Navigate: options.InitialURL},
				{Type: "type_credential", CandidateID: ids.email, Slot: "identifier"},
				{Type: "type_credential", CandidateID: ids.password, Slot: "password"},
				{Type: "submit", CandidateID: ids.submit},
				{Type: "wait_for", CandidateID: ids.success},
			},
			Effects: []string{"creates_account"}, ConfirmationPrompt: "Approve creation of one dedicated loopback test identity.",
			Success: registrationDraftSuccess{Origin: options.Origin, Path: "/registration-complete", CandidateID: ids.success},
		},
		CallControls: registrationDraftCallControls{
			Approval: "browser_registration_submit", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail",
			AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately",
		},
	}
	draftCommand := registrationAuthoringCommandRequest{
		Revision: observed.Revision, RegistrationRevision: observed.RegistrationRevision, Type: "draft", Draft: &draft,
	}
	drafted, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", draftCommand, http.StatusOK)
	if err != nil || drafted.RegistrationAuthoring == nil || drafted.RegistrationAuthoring.Draft == nil {
		return RegistrationQualificationResult{}, errors.New("build registration qualification draft")
	}
	disclosure := drafted.RegistrationAuthoring.Draft
	retained := len(disclosure.RetainedQueries) == 1 && len(disclosure.RetainedQueries[0].Parameters) == 1 &&
		disclosure.RetainedQueries[0].Parameters[0].Key == "action" && disclosure.RetainedQueries[0].Parameters[0].Value == "startnew"
	if !retained || len(disclosure.Canonical) == 0 {
		return RegistrationQualificationResult{}, errors.New("registration qualification retained-query review is invalid")
	}
	review := registrationAuthoringCommandRequest{
		Revision: drafted.Revision, RegistrationRevision: drafted.RegistrationRevision, Type: "review", Confirmed: true,
	}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", review, http.StatusAccepted); err != nil {
		return RegistrationQualificationResult{}, errors.New("review registration qualification draft")
	}
	reviewed, err := registrationQualificationWait(ctx, handler, "reviewed")
	if err != nil {
		return RegistrationQualificationResult{}, err
	}
	finish := registrationAuthoringCommandRequest{
		Revision: reviewed.Revision, RegistrationRevision: reviewed.RegistrationRevision, Type: "finish", Confirmed: true,
	}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", finish, http.StatusAccepted); err != nil {
		return RegistrationQualificationResult{}, errors.New("finish registration qualification worker")
	}
	pending, err := registrationQualificationWait(ctx, handler, "transaction_review")
	if err != nil || pending.BrowserTransaction == nil || pending.BrowserTransaction.Transaction == nil {
		return RegistrationQualificationResult{}, errors.New("registration qualification transaction is unavailable")
	}
	transactionSnapshot, err := transactions.Observe(ctx)
	if err != nil {
		return RegistrationQualificationResult{}, errors.New("observe registration qualification transaction")
	}
	transactionReview := transactionengine.ReviewRequest{Authority: transactionengine.Authority{
		ExpectedRevision: transactionSnapshot.Revision, ExpectedTransactionSHA256: transactionSnapshot.TransactionSHA256, HumanApproved: true,
	}}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/browser-transactions/review", transactionReview, http.StatusOK); err != nil {
		return RegistrationQualificationResult{}, errors.New("review registration qualification transaction")
	}

	authoringState, err := registrationQualificationCurrent(ctx, handler)
	if err != nil || authoringState.RegistrationAuthoring == nil || authoringState.RegistrationAuthoring.State != "adopted" {
		return RegistrationQualificationResult{}, errors.New("adopt registration qualification virtual source")
	}
	for round := 0; !authoringState.Snapshot.ApprovalRequired && round < 16; round++ {
		if len(authoringState.Snapshot.Frontier) == 0 {
			return RegistrationQualificationResult{}, errors.New("registration qualification authoring stalled")
		}
		answers := make([]authoring.RoundAnswer, 0, len(authoringState.Snapshot.Frontier))
		for _, question := range authoringState.Snapshot.Frontier {
			value := strings.TrimSpace(question.Recommendation)
			if value == "" {
				value = strings.TrimSpace(question.SuggestedAnswer)
			}
			if value == "" {
				value = "reviewed registration package"
			}
			answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Value: value})
		}
		response, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/round", map[string]any{
			"revision": authoringState.Revision, "answers": answers,
		}, http.StatusOK)
		if err != nil {
			return RegistrationQualificationResult{}, errors.New("advance registration qualification authoring")
		}
		authoringState = response
	}
	if !authoringState.Snapshot.ApprovalRequired || !authoringState.Snapshot.Ready {
		return RegistrationQualificationResult{}, errors.New("registration qualification package is not review-ready")
	}
	approved, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/author/approve", map[string]any{
		"revision": authoringState.Revision, "human_approved": true,
	}, http.StatusOK)
	if err != nil {
		return RegistrationQualificationResult{}, errors.New("approve registration qualification package")
	}
	built, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/package/build", map[string]any{
		"revision": approved.Revision, "confirmed": true,
	}, http.StatusOK)
	if err != nil || built.Lifecycle != lifecycleHandoffReady || built.Package == nil || built.Package.Status != "pass" {
		return RegistrationQualificationResult{}, errors.New("build registration qualification package")
	}
	transactionSnapshot, err = transactions.Observe(ctx)
	if err != nil {
		return RegistrationQualificationResult{}, errors.New("observe reviewed registration qualification transaction")
	}
	prepare := transactionengine.PrepareRequest{Authority: transactionengine.Authority{
		ExpectedRevision: transactionSnapshot.Revision, ExpectedTransactionSHA256: transactionSnapshot.TransactionSHA256, HumanApproved: true,
	}}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/browser-transactions/prepare", prepare, http.StatusOK); err != nil {
		return RegistrationQualificationResult{}, errors.New("prepare registration qualification transaction")
	}
	prepared, err := transactions.Observe(ctx)
	if err != nil || prepared.Preparation == nil {
		return RegistrationQualificationResult{}, errors.New("registration qualification preparation is unavailable")
	}
	promote := transactionengine.PromoteRequest{
		Authority: transactionengine.Authority{
			ExpectedRevision: prepared.Revision, ExpectedTransactionSHA256: prepared.TransactionSHA256, HumanApproved: true,
		},
		ExpectedPreparationSHA256: prepared.Preparation.PreparationSHA256, ExpectedQualificationSHA256: prepared.Preparation.QualificationSHA256,
	}
	if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/browser-transactions/promote", promote, http.StatusOK); err != nil {
		return RegistrationQualificationResult{}, errors.New("promote registration qualification transaction")
	}
	promoted, err := transactions.Observe(ctx)
	if err != nil || promoted.Transaction == nil || promoted.Preparation == nil || promoted.Promotion == nil {
		return RegistrationQualificationResult{}, errors.New("registration qualification promotion evidence is unavailable")
	}
	return RegistrationQualificationResult{
		Snapshot: promoted, CanonicalProfile: append(json.RawMessage(nil), disclosure.Canonical...), RetainedQuery: true,
	}, nil
}

type registrationQualificationIDs struct{ email, password, submit, success string }

func registrationQualificationCandidateIDs(observation registrationauthorsession.Observation) (registrationQualificationIDs, error) {
	result := registrationQualificationIDs{}
	for _, candidate := range observation.Candidates {
		if candidate.Matches != 1 {
			continue
		}
		switch {
		case candidate.Role == "textbox" && candidate.Label == "Email" && result.email == "":
			result.email = candidate.ID
		case candidate.Role == "textbox" && candidate.Label == "Password" && result.password == "":
			result.password = candidate.ID
		case candidate.Role == "button" && candidate.Label == "Register" && result.submit == "":
			result.submit = candidate.ID
		case candidate.Role == "status" && candidate.Label == "Registration complete" && result.success == "":
			result.success = candidate.ID
		}
	}
	if result.email == "" || result.password == "" || result.submit == "" || result.success == "" {
		return registrationQualificationIDs{}, errors.New("registration qualification accessibility locators are incomplete or ambiguous")
	}
	return result, nil
}

func registrationQualificationWait(ctx context.Context, handler http.Handler, state string) (Response, error) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, err := registrationQualificationCurrent(ctx, handler)
		if err != nil {
			return Response{}, err
		}
		if current.RegistrationAuthoring != nil && current.RegistrationAuthoring.State == state {
			return current, nil
		}
		if current.RegistrationAuthoring != nil && (current.RegistrationAuthoring.State == "failed" || current.RegistrationAuthoring.State == "canceled") {
			return Response{}, errors.New("registration qualification wizard failed closed")
		}
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func registrationQualificationCurrent(ctx context.Context, handler http.Handler) (Response, error) {
	return registrationQualificationJSON(ctx, handler, http.MethodGet, "/api/v4/snapshot", nil, http.StatusOK)
}

func registrationQualificationJSON(ctx context.Context, handler http.Handler, method, path string, body any, status int) (Response, error) {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return Response{}, err
		}
	}
	request := httptest.NewRequestWithContext(ctx, method, "http://"+registrationQualificationAuthority+path, bytes.NewReader(data))
	request.Host = registrationQualificationAuthority
	request.Header.Set("Authorization", "Bearer "+registrationQualificationToken)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != status {
		return Response{}, errors.New("registration qualification UI request failed")
	}
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		return Response{}, errors.New("registration qualification UI response is invalid")
	}
	return response, nil
}
