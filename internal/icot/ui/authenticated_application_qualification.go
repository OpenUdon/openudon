package ui

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	transactionengine "github.com/OpenUdon/openudon/internal/browsertransaction/engine"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

func (q *applicationQualification) command(operation string, request any) (Response, error) {
	frame := registrationControlFrame{Version: ApplicationControlVersion, Operation: operation}
	if request != nil {
		frame.Request, _ = json.Marshal(request)
	}
	if q.encoder.Encode(frame) != nil {
		return Response{}, errors.New("application_input")
	}
	return q.read()
}

// RunAuthenticatedApplicationQualification is a closed synthetic fixture
// journey. It drives the actual command/worker and all package decisions, with
// synthetic human responses only at the fixed loopback fixture's checkpoints.
func RunAuthenticatedApplicationQualification(ctx context.Context, options RegistrationQualificationOptions) (result transactionengine.Snapshot, resultErr error) {
	if !qualificationLoopbackURL(options.Origin, options.InitialURL) || options.InitialURL != options.Origin+"/login" || options.ApplicationExecutable == "" {
		return result, errors.New("fixture_authority")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	q, err := newApplicationQualification(ctx, options)
	if err != nil {
		return result, err
	}
	defer func() {
		if err := q.Close(); err != nil {
			resultErr = err
		}
	}()
	current := *q.initial
	q.initial = nil
	current, err = q.command("author.journey", journeyRequest{Revision: current.Revision, Starter: "authenticated_action", Goal: "Authenticate and prove campaign list presence"})
	if err != nil {
		return result, errors.New("authoring_journey")
	}

	current, err = q.command("browser.preflight", captureMutationRequest{Revision: current.Revision, CaptureRevision: current.CaptureRevision})
	if err != nil {
		return result, errors.New("capture_preflight")
	}
	current, err = q.command("capture.start", captureStartRequest{
		Revision: current.Revision, CaptureRevision: current.CaptureRevision, ProfileID: options.ProfileID,
		URL: options.InitialURL, DashboardURL: options.Origin + "/campaigns", Goal: "Prove campaign list presence", Origins: []string{options.Origin},
		GoalOrigin: options.Origin, GoalPath: "/campaigns", GoalContext: "main", GoalRole: "heading", GoalLabel: "Campaigns",
	})
	if err != nil {
		return result, errors.New("capture_start")
	}
	credentials := map[string]bool{}
	submitted := false
	for count := 0; count < 3000; count++ {
		if ctx.Err() != nil {
			return result, errors.New("capture_deadline")
		}
		capture := current.Capture
		if capture == nil {
			return result, errors.New("capture_state")
		}
		if capture.State == "stage_review" {
			break
		}
		if capture.State == "failed" || capture.State == "canceled" {
			return result, errors.New("capture_failed")
		}
		var response *browserauthor.Response
		switch {
		case capture.Approval != nil:
			response = &browserauthor.Response{Kind: "approve", ApprovalID: capture.Approval.ID}
		case capture.Checkpoint != nil:
			cp := capture.Checkpoint
			if cp.Kind == "credential" {
				if cp.InputKind != "identifier" && cp.InputKind != "password" {
					return result, errors.New("capture_checkpoint")
				}
				credentials[cp.InputKind] = true
				response = &browserauthor.Response{Kind: "continue", CandidateID: cp.CandidateID}
			} else if cp.Kind == "completion" && capture.Observation != nil {
				id, err := applicationFixtureCandidate(*capture.Observation, "status", "Campaign list")
				if err != nil {
					return result, err
				}
				response = &browserauthor.Response{Kind: "confirm", Confirmed: true, Outputs: []authorsession.OutputRequest{{CandidateID: id, Key: "campaign_list_present", Type: "presence", LocatorMode: "exact_name"}}}
			} else {
				return result, errors.New("capture_checkpoint")
			}
		case capture.Observation != nil:
			role, label, kind := "textbox", "Email address", "focus_human_input"
			budget := 0
			if credentials["identifier"] {
				label = "Password"
			}
			if credentials["password"] {
				role, label, kind, budget = "button", "Sign in", "click", 1
				if submitted {
					return result, errors.New("duplicate_login")
				}
				submitted = true
			}
			id, err := applicationFixtureCandidate(*capture.Observation, role, label)
			if err != nil {
				return result, err
			}
			response = &browserauthor.Response{Kind: kind, CandidateID: id, POSTBudget: budget}
		}
		if response != nil {
			current, err = q.command("capture.respond", captureRespondRequest{CaptureRevision: current.CaptureRevision, Response: *response})
		} else {
			select {
			case <-ctx.Done():
				return result, errors.New("capture_deadline")
			case <-time.After(20 * time.Millisecond):
			}
			current, err = q.command("snapshot", nil)
		}
		if err != nil {
			return result, errors.New("capture_response")
		}
	}
	if current.Capture == nil || current.Capture.State != "stage_review" {
		return result, errors.New("capture_readiness")
	}
	current, err = q.command("capture.stage", captureMutationRequest{Revision: current.Revision, CaptureRevision: current.CaptureRevision})
	if err != nil {
		return result, errors.New("capture_stage")
	}
	if current.BrowserTransaction == nil {
		return result, errors.New("capture_transaction")
	}
	tx := current.BrowserTransaction
	current, err = q.command("transaction.review", transactionengine.ReviewRequest{Authority: transactionengine.Authority{ExpectedRevision: tx.Revision, ExpectedTransactionSHA256: tx.TransactionSHA256, HumanApproved: true}})
	if err != nil {
		return result, errors.New("capture_transaction_review")
	}
	for round := 0; !current.Snapshot.ApprovalRequired && round < 24; round++ {
		if len(current.Snapshot.Frontier) == 0 {
			return result, errors.New("authoring_stalled")
		}
		request := roundRequest{Revision: current.Revision}
		for _, question := range current.Snapshot.Frontier {
			value := strings.TrimSpace(question.Recommendation)
			if value == "" {
				value = strings.TrimSpace(question.SuggestedAnswer)
			}
			if value == "" {
				switch question.ID {
				case "boundary.outcome", "readiness.missing_goal.workflow_description":
					value = "Authenticate and prove campaign list presence"
				case "workflow.steps":
					value = "reach_authenticated_goal"
				case "boundary.name":
					value = "campaign_list_presence"
				default:
					return result, errors.New("authoring_fixture_answer:" + question.ID)
				}
			}
			request.Answers = append(request.Answers, roundAnswer{QuestionID: question.ID, Value: value})
		}
		current, err = q.command("author.round", request)
		if err != nil {
			return result, errors.New("authoring_round:" + err.Error())
		}
	}
	if !current.Snapshot.Ready || !current.Snapshot.ApprovalRequired {
		return result, errors.New("authoring_not_ready")
	}
	current, err = q.command("author.approve", approveRequest{Revision: current.Revision, HumanApproved: true})
	if err != nil {
		return result, errors.New("authoring_approval")
	}
	current, err = q.command("package.build", buildRequest{Revision: current.Revision, Confirmed: true})
	if err != nil || current.Lifecycle != lifecycleHandoffReady {
		return result, errors.New("package_build")
	}
	tx = current.BrowserTransaction
	current, err = q.command("transaction.prepare", transactionengine.PrepareRequest{Authority: transactionengine.Authority{ExpectedRevision: tx.Revision, ExpectedTransactionSHA256: tx.TransactionSHA256, HumanApproved: true}})
	if err != nil || current.BrowserTransaction == nil || current.BrowserTransaction.Preparation == nil {
		return result, errors.New("package_prepare")
	}
	tx = current.BrowserTransaction
	current, err = q.command("transaction.promote", transactionengine.PromoteRequest{Authority: transactionengine.Authority{ExpectedRevision: tx.Revision, ExpectedTransactionSHA256: tx.TransactionSHA256, HumanApproved: true}, ExpectedPreparationSHA256: tx.Preparation.PreparationSHA256, ExpectedQualificationSHA256: tx.Preparation.QualificationSHA256})
	if err != nil || current.BrowserTransaction == nil || current.BrowserTransaction.Promotion == nil {
		return result, errors.New("package_promote")
	}
	return current.BrowserTransaction.Snapshot, nil
}
func applicationFixtureCandidate(observation authorsession.Observation, role, label string) (string, error) {
	id := ""
	for _, c := range observation.Candidates {
		if c.Role == role && c.Label == label {
			if c.Matches != 1 || id != "" {
				return "", errors.New("ambiguous_fixture_candidate")
			}
			id = c.ID
		}
	}
	if id == "" {
		return "", errors.New("missing_fixture_candidate")
	}
	return id, nil
}
