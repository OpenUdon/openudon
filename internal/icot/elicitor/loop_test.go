package elicitor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestRunFillsRuntimeOnlyIntent(t *testing.T) {
	example := t.TempDir()
	input := strings.Join([]string{
		"Render a local summary report from a runtime input.",
		"runtime_only_render",
		"Render a local summary report.",
		"",
		"",
		"no",
		"summary:string",
		"render_report",
		"fnct",
		"Render the summary report.",
		"",
		"summary",
		"",
		"",
		"report",
		"render_report.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop if the summary input is missing",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if intent.Workflow.Name != "runtime_only_render" || len(intent.Steps) != 1 {
		t.Fatalf("unexpected intent: %#v", intent)
	}
	if got := intent.Steps[0].With["summary"]; got != "inputs.summary" {
		t.Fatalf("summary binding = %q", got)
	}
}

func TestRunFillsOpenAPIRequiredParams(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"support_ticket_lookup",
		"Fetch a support ticket by runtime id.",
		"",
		"",
		"yes",
		"1",
		"ticketId:string",
		"get_ticket",
		"http",
		"Fetch the ticket.",
		"",
		"1",
		"1",
		"",
		"",
		"",
		"ticket",
		"get_ticket.received_body",
		"sandbox-only",
		"support_api_token",
		"Sandbox proof runs only",
		"Stop if the support API is unavailable",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if intent.OpenAPI != "openapi/support.yaml" {
		t.Fatalf("openapi = %q", intent.OpenAPI)
	}
	step := intent.Steps[0]
	if step.Operation != "getTicket" {
		t.Fatalf("operation = %q", step.Operation)
	}
	if got := step.With["ticketId"]; got != "inputs.ticketId" {
		t.Fatalf("ticketId binding = %q", got)
	}
}

func TestRunUsesAIDraftDefaults(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"getTicket",
		"ticketId=inputs.ticketId",
		"save",
	}, "\n") + "\n"
	extractor := draftExtractor{session: Session{
		Intent: rollout.Intent{
			OpenAPI: "openapi/support.yaml",
			Inputs:  []*rollout.Input{{Name: "ticketId", Type: "string", Required: true}},
			Steps: []*rollout.Step{{
				Name:      "get_ticket",
				Type:      "http",
				Do:        "Fetch the ticket.",
				Operation: "getTicket",
				With:      map[string]string{"ticketId": "inputs.ticketId"},
			}},
			Outputs: []*rollout.Output{{Name: "ticket", From: "get_ticket.received_body"}},
		},
		Assumptions: []Assumption{{
			ID:                   "op_get_ticket",
			Slot:                 "steps.get_ticket.operation",
			Value:                "getTicket",
			Reason:               "Only support ticket lookup operation matched the brief.",
			Evidence:             "Support API getTicket",
			Risk:                 "low",
			RequiresConfirmation: true,
		}},
	}}
	var out strings.Builder
	artifacts, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if len(intent.Steps) != 1 || intent.Steps[0].Operation != "getTicket" {
		t.Fatalf("drafted step not preserved: %#v", intent.Steps)
	}
	if !strings.Contains(out.String(), "Assumptions to confirm") || !strings.Contains(out.String(), "op_get_ticket") {
		t.Fatalf("final review missing assumptions:\n%s", out.String())
	}
}

func TestFastPromptModeSuppressesDraftErrorAndAssumptions(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	seed := supportTicketDraft(true)
	var out strings.Builder
	_, err := Run(context.Background(), strings.NewReader(""), &out, seed, Options{
		ExampleDir:     example,
		NoLLM:          false,
		Extractor:      invalidJSONDraftExtractor{},
		DefaultMode:    authoring.PromptDefaultsSilent,
		DisableAIDraft: false,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	text := out.String()
	for _, unexpected := range []string{
		"icot: AI draft skipped",
		"model returned invalid draft JSON",
		"Assumptions to confirm:",
		"Saving confirms these assumptions.",
		"op_get_ticket",
	} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("fast prompt output included %q:\n%s", unexpected, text)
		}
	}
	if !strings.Contains(text, "----- current draft -----") {
		t.Fatalf("fast prompt output should still show final draft summary:\n%s", text)
	}
}

func TestDiscoverLocalAPIsIncludesGoogleDiscoveryOperations(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir discovery: %v", err)
	}
	data := `{
	  "kind": "discovery#restDescription",
	  "name": "gmail",
	  "title": "Gmail API",
	  "version": "v1",
	  "rootUrl": "https://gmail.googleapis.com/",
	  "servicePath": "",
	  "resources": {
	    "users": {
	      "resources": {
	        "messages": {
	          "methods": {
	            "send": {
	              "id": "gmail.users.messages.send",
	              "path": "gmail/v1/users/{userId}/messages/send",
	              "httpMethod": "POST",
	              "description": "Sends the specified message to the recipients.",
	              "parameters": {
	                "userId": {
	                  "type": "string",
	                  "location": "path",
	                  "required": true,
	                  "description": "The user's email address."
	                }
	              },
	              "request": {"$ref": "Message"}
	            }
	          }
	        }
	      }
	    }
	  }
	}`
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write discovery: %v", err)
	}

	docs, err := DiscoverLocalAPIs(example, "gmail send report")
	if err != nil {
		t.Fatalf("DiscoverLocalAPIs failed: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs = %#v, want one discovery doc", docs)
	}
	if docs[0].RelativePath != "discovery/gmail.json" || len(docs[0].Operations) != 1 {
		t.Fatalf("unexpected discovery doc: %#v", docs[0])
	}
	op := docs[0].Operations[0]
	if op.OperationID != "gmail_users_messages_send" || op.Description == "" {
		t.Fatalf("operation = %#v", op)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "userId" {
		t.Fatalf("parameters = %#v", op.Parameters)
	}
}

func TestDiscoverLocalAPIsIncludesAWSSmithyOperations(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "aws-smithy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir aws-smithy: %v", err)
	}
	data := `{
	  "smithy": "2.0",
	  "shapes": {
	    "com.amazonaws.lambda#Lambda": {
	      "type": "service",
	      "version": "2015-03-31",
	      "operations": [{"target": "com.amazonaws.lambda#GetFunction"}],
	      "traits": {
	        "aws.api#service": {
	          "sdkId": "Lambda",
	          "arnNamespace": "lambda",
	          "endpointPrefix": "lambda"
	        },
	        "aws.auth#sigv4": {"name": "lambda"},
	        "aws.protocols#restJson1": {}
	      }
	    },
	    "com.amazonaws.lambda#GetFunction": {
	      "type": "operation",
	      "input": {"target": "com.amazonaws.lambda#GetFunctionRequest"},
	      "traits": {
	        "smithy.api#http": {"method": "GET", "uri": "/2015-03-31/functions/{FunctionName}", "code": 200}
	      }
	    },
	    "com.amazonaws.lambda#GetFunctionRequest": {
	      "type": "structure",
	      "members": {
	        "FunctionName": {
	          "target": "smithy.api#String",
	          "traits": {
	            "smithy.api#required": {},
	            "smithy.api#httpLabel": {}
	          }
	        }
	      }
	    }
	  }
	}`
	if err := os.WriteFile(filepath.Join(dir, "lambda.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write smithy: %v", err)
	}

	docs, err := DiscoverLocalAPIs(example, "lambda get function")
	if err != nil {
		t.Fatalf("DiscoverLocalAPIs failed: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs = %#v, want one smithy doc", docs)
	}
	if docs[0].RelativePath != "aws-smithy/lambda.json" || len(docs[0].Operations) != 1 {
		t.Fatalf("unexpected smithy doc: %#v", docs[0])
	}
	op := docs[0].Operations[0]
	if op.OperationID != "GetFunction" || op.Method != "GET" {
		t.Fatalf("operation = %#v", op)
	}
	if len(op.Parameters) != 1 || op.Parameters[0].Name != "FunctionName" || op.Parameters[0].In != "path" {
		t.Fatalf("parameters = %#v", op.Parameters)
	}
}

func TestRunCreatesStepBindFromPriorOutput(t *testing.T) {
	example := t.TempDir()
	input := strings.Join([]string{
		"Fetch a customer and write a draft.",
		"customer_draft",
		"Fetch a customer and write a draft.",
		"",
		"",
		"no",
		"customerId:string",
		"get_customer",
		"fnct",
		"Fetch the customer.",
		"",
		"customerId",
		"",
		"write_draft",
		"fnct",
		"Write the draft.",
		"",
		"customerId",
		"get_customer.received_body.id",
		"",
		"draft",
		"write_draft.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop on missing customer data",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	step := intent.Steps[1]
	if len(step.DependsOn) != 1 || step.DependsOn[0] != "get_customer" {
		t.Fatalf("depends_on = %#v", step.DependsOn)
	}
	if len(step.Binds) != 1 || step.Binds[0].Fields["customerId"] != "received_body.id" {
		t.Fatalf("binds = %#v", step.Binds)
	}
}

func TestRunFillsTimeoutAndIdempotencyControls(t *testing.T) {
	example := t.TempDir()
	input := strings.Join([]string{
		"Submit one controlled local function call.",
		"timeout_idempotency_controls",
		"Submit one controlled local function call.",
		"120",
		"inputs.request_id",
		"returnPrevious",
		"86400",
		"no",
		"request_id:string, payload:string",
		"call_api",
		"fnct",
		"Submit the payload through the approved local function.",
		"10",
		"payload",
		"",
		"",
		"result",
		"call_api.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop on missing payload",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if intent.Workflow.Timeout == nil || *intent.Workflow.Timeout != 120 {
		t.Fatalf("workflow timeout = %#v", intent.Workflow.Timeout)
	}
	if intent.Workflow.Idempotency == nil || intent.Workflow.Idempotency.Key != "inputs.request_id" || intent.Workflow.Idempotency.OnConflict != "returnPrevious" {
		t.Fatalf("workflow idempotency = %#v", intent.Workflow.Idempotency)
	}
	if intent.Workflow.Idempotency.TTL == nil || *intent.Workflow.Idempotency.TTL != 86400 {
		t.Fatalf("workflow idempotency ttl = %#v", intent.Workflow.Idempotency)
	}
	if len(intent.Steps) != 1 || intent.Steps[0].Timeout == nil || *intent.Steps[0].Timeout != 10 {
		t.Fatalf("step timeout = %#v", intent.Steps)
	}
}

type draftExtractor struct {
	noopExtractor
	session Session
}

func (e draftExtractor) Draft(context.Context, DraftRequest) (Session, error) {
	return e.session, nil
}

type invalidJSONDraftExtractor struct {
	noopExtractor
}

func (invalidJSONDraftExtractor) Draft(context.Context, DraftRequest) (Session, error) {
	return Session{}, errors.New("invalid character 'x' looking for beginning of value")
}

type sequenceDraftExtractor struct {
	noopExtractor
	drafts                 []Session
	calls                  []DraftRequest
	requestMappingResponse RequestMappingResponse
	requestMappingRequests []RequestMappingRequest
	draftReviewResponse    DraftReviewResponse
	draftReviewRequests    []DraftReviewRequest
}

func (e *sequenceDraftExtractor) Draft(_ context.Context, request DraftRequest) (Session, error) {
	e.calls = append(e.calls, request)
	if len(e.drafts) == 0 {
		return Session{}, nil
	}
	draft := e.drafts[0]
	if len(e.drafts) > 1 {
		e.drafts = e.drafts[1:]
	}
	return draft, nil
}

func (e *sequenceDraftExtractor) RequestMappings(_ context.Context, request RequestMappingRequest) (RequestMappingResponse, error) {
	e.requestMappingRequests = append(e.requestMappingRequests, request)
	return e.requestMappingResponse, nil
}

func (e *sequenceDraftExtractor) ReviewDraft(_ context.Context, request DraftReviewRequest) (DraftReviewResponse, error) {
	e.draftReviewRequests = append(e.draftReviewRequests, request)
	return e.draftReviewResponse, nil
}

type catalogPlanningExtractor struct {
	noopExtractor
}

func (catalogPlanningExtractor) CatalogPlan(_ context.Context, request CatalogPlanRequest) (CatalogPlanResponse, error) {
	for _, candidate := range request.Candidates {
		if candidate.ProviderID == "gmail" {
			return CatalogPlanResponse{
				SelectedArtifacts: []CatalogPlanArtifactSelection{{
					ProviderID:  candidate.ProviderID,
					ArtifactKey: candidate.ArtifactKey,
				}},
				ProposedSteps: []CatalogPlanStep{{
					Name:     "send_gmail",
					Type:     "http",
					Provider: "gmail",
					OpenAPI:  candidate.RelativePath,
					Do:       "Send the report through Gmail.",
				}},
			}, nil
		}
	}
	return CatalogPlanResponse{Blockers: []string{"gmail artifact not listed"}}, nil
}

func TestProgressiveOneAnswerJumpsToConfirmation(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(true)}}
	input := strings.Join([]string{
		"Fetch a support ticket by runtime id.",
		"getTicket",
		"save",
	}, "\n") + "\n"
	var out strings.Builder
	artifacts, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "Workflow name") || strings.Contains(out.String(), "Use OpenAPI/API steps?") {
		t.Fatalf("progressive path fell back to manual prompts:\n%s", out.String())
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if len(intent.Steps) != 1 || intent.Steps[0].Operation != "getTicket" {
		t.Fatalf("operation = %#v", intent.Steps)
	}
	if len(extractor.calls) != 1 || len(extractor.calls[0].TranscriptTurns) == 0 {
		t.Fatalf("draft request did not include transcript turns: %#v", extractor.calls)
	}
}

func TestProgressiveTwoQuestionPathUsesReadinessFeedback(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	first := supportTicketDraft(false)
	first.Intent.Inputs = nil
	extractor := &sequenceDraftExtractor{
		drafts: []Session{first},
		requestMappingResponse: RequestMappingResponse{Steps: []RequestMappingStepResponse{{
			Name: "get_ticket",
			With: map[string]string{"ticketId": "inputs.ticketId"},
		}}},
	}
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"getTicket",
		"save",
	}, "\n") + "\n"
	var out strings.Builder
	_, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "What values should the required request fields use?") {
		t.Fatalf("required-field question was shown before LLM mapping draft:\n%s", out.String())
	}
	if len(extractor.requestMappingRequests) != 1 || len(extractor.requestMappingRequests[0].ReadinessIssues) == 0 {
		t.Fatalf("request-mapping draft did not receive readiness feedback: %#v", extractor.requestMappingRequests)
	}
}

func TestProgressivePreFinalReviewAddsCrossStepWarning(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	transcriptPath := filepath.Join(example, ".icot", "transcript.json")
	extractor := &sequenceDraftExtractor{
		drafts: []Session{supportTicketDraft(true)},
		draftReviewResponse: DraftReviewResponse{Issues: []DraftReviewIssue{{
			Severity: "warning",
			Code:     "output_transport_response",
			Slot:     "intent.outputs.result",
			Message:  "The output returns the API transport response instead of the report content requested by the goal.",
			Evidence: "result=get_ticket.received_body",
		}}},
	}
	var out strings.Builder
	artifacts, err := Run(context.Background(), strings.NewReader("Fetch a support ticket by runtime id.\ngetTicket\nexplain op_get_ticket\nsave\n"), &out, Session{}, Options{
		ExampleDir:     example,
		NoLLM:          false,
		Extractor:      extractor,
		TranscriptPath: transcriptPath,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if len(extractor.draftReviewRequests) != 1 {
		t.Fatalf("draft review requests = %d, want 1", len(extractor.draftReviewRequests))
	}
	if !strings.Contains(out.String(), "llm_flow_review_output_transport_response") ||
		!strings.Contains(out.String(), "The output returns the API transport response") {
		t.Fatalf("final review missing LLM flow warning:\n%s", out.String())
	}
	for _, expected := range []string{
		"# iCoT flow review warning (llm_flow_review_output_transport_response)",
		"# Evidence: result=get_ticket.received_body",
		`output "ticket" {`,
	} {
		if !strings.Contains(artifacts.IntentHCL, expected) {
			t.Fatalf("annotated intent missing %q:\n%s", expected, artifacts.IntentHCL)
		}
	}
	if _, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl"); err != nil {
		t.Fatalf("parse annotated intent: %v\n%s", err, artifacts.IntentHCL)
	}
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"draft_flow_review_result", "result=get_ticket.received_body"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("transcript missing %q:\n%s", expected, text)
		}
	}
}

func TestProgressivePreFinalReviewRunsAfterFinalBlockingRepair(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	docs, err := DiscoverLocalAPIs(example, "")
	if err != nil {
		t.Fatalf("discover APIs: %v", err)
	}
	session := supportTicketDraft(true)
	session.Intent.Steps[0].Operation = ""
	var out strings.Builder
	prompts := authoring.NewPromptSession(strings.NewReader("\nsave\n"), &out)
	reviewCalls := 0
	review := func(_ context.Context, artifacts Artifacts, _ []ReadinessIssue) []DraftReviewIssue {
		reviewCalls++
		if got := artifacts.Session.Intent.Steps[0].Operation; got != "getTicket" {
			t.Fatalf("review saw operation %q, want repaired getTicket", got)
		}
		return nil
	}
	if _, err := finalProgressiveConfirmationLoop(context.Background(), &out, &prompter{PromptSession: prompts, out: &out}, &session, docs, "", nil, true, review); err != nil {
		t.Fatalf("final confirmation failed: %v\n%s", err, out.String())
	}
	if reviewCalls != 1 {
		t.Fatalf("review calls = %d, want 1", reviewCalls)
	}
}

func TestProgressivePreFinalReviewRunsAgainAfterFinalEdit(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	extractor := &sequenceDraftExtractor{
		drafts: []Session{supportTicketDraft(true)},
	}
	input := strings.Join([]string{
		"Fetch a support ticket by runtime id.",
		"getTicket",
		"edit goal",
		"",
		"Fetch a support ticket after final edit.",
		"save",
	}, "\n") + "\n"
	var out strings.Builder
	if _, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	}); err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if len(extractor.draftReviewRequests) != 2 {
		t.Fatalf("draft review requests = %d, want 2\n%s", len(extractor.draftReviewRequests), out.String())
	}
	if got := extractor.draftReviewRequests[1].Workflow; got != "Fetch a support ticket after final edit." {
		t.Fatalf("second review workflow = %q", got)
	}
}

func TestFastPromptModeSuppressesRequestMappingStatus(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	first := supportTicketDraft(false)
	first.Intent.Inputs = nil
	extractor := &sequenceDraftExtractor{
		drafts: []Session{first},
		requestMappingResponse: RequestMappingResponse{Steps: []RequestMappingStepResponse{{
			Name: "get_ticket",
			With: map[string]string{"ticketId": "inputs.ticketId"},
		}}},
	}
	var out strings.Builder
	_, err := Run(context.Background(), strings.NewReader("Fetch a support ticket.\n"), &out, Session{}, Options{
		ExampleDir:  example,
		NoLLM:       false,
		Extractor:   extractor,
		DefaultMode: authoring.PromptDefaultsSilent,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "icot: drafted request mappings") {
		t.Fatalf("fast prompt output included request-mapping status:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "----- current draft -----") {
		t.Fatalf("fast prompt output should still show final draft summary:\n%s", out.String())
	}
}

func TestProgressiveQuestionDraftFillsRequiredMappingsBeforePrompt(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	first := Session{}
	extractor := &sequenceDraftExtractor{
		drafts: []Session{first},
		requestMappingResponse: RequestMappingResponse{Steps: []RequestMappingStepResponse{{
			Name: "get_ticket",
			With: map[string]string{"ticketId": "inputs.ticketId"},
		}}},
	}
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"getTicket",
		"save",
	}, "\n") + "\n"
	var out strings.Builder
	artifacts, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "What values should the required request fields use?") {
		t.Fatalf("required-field prompt was shown despite question draft:\n%s", out.String())
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if got := intent.Steps[0].With["ticketId"]; got != "inputs.ticketId" {
		t.Fatalf("ticketId mapping = %q", got)
	}
	if len(extractor.calls) != 1 {
		t.Fatalf("draft calls = %d, want initial draft only", len(extractor.calls))
	}
	if len(extractor.requestMappingRequests) != 1 {
		t.Fatalf("request mapping calls = %d, want 1", len(extractor.requestMappingRequests))
	}
}

func TestProgressiveDeterministicPrefillSkipsRequiredFieldQuestion(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(false)}}
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"getTicket",
		"save",
	}, "\n") + "\n"
	var out strings.Builder
	artifacts, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "What values should the required request fields use?") {
		t.Fatalf("deterministic prefill still asked for required fields:\n%s", out.String())
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if got := intent.Steps[0].With["ticketId"]; got != "inputs.ticketId" {
		t.Fatalf("ticketId prefill = %q", got)
	}
	if !hasAssumption(artifacts.Session.Assumptions, "deterministic_prefill_steps_get_ticket_with_ticketId") {
		t.Fatalf("missing deterministic prefill assumption: %#v", artifacts.Session.Assumptions)
	}
	if !hasClassification(artifacts.Session.Classifications, "steps.get_ticket.with.ticketId", "inputs.ticketId", mappingSourceDeterministic, mappingConfidenceHigh) {
		t.Fatalf("missing deterministic prefill classification: %#v", artifacts.Session.Classifications)
	}
}

func TestDeterministicPrefillUsesSingleCredentialBinding(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"support_api_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{"ticketId": "inputs.ticketId"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
		Security:    securitySummaries("BearerAuth"),
	}}}}

	if !deterministicPrefill(&session, docs) {
		t.Fatalf("deterministic prefill did not change session")
	}
	if got := session.Intent.Steps[0].With["Authorization"]; got != "credentials.support_api_token" {
		t.Fatalf("Authorization prefill = %q", got)
	}
	if !hasClassification(session.Classifications, "steps.get_ticket.with.Authorization", "credentials.support_api_token", mappingSourceDeterministic, mappingConfidenceHigh) {
		t.Fatalf("missing credential classification: %#v", session.Classifications)
	}
}

func TestDeterministicPrefillLeavesAmbiguousCredentialUnfilled(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"first_token", "second_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{"ticketId": "inputs.ticketId"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
		Security:    securitySummaries("BearerAuth"),
	}}}}

	deterministicPrefill(&session, docs)
	if got := session.Intent.Steps[0].With["Authorization"]; got != "" {
		t.Fatalf("ambiguous credential was prefilled: %q", got)
	}
	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "missing_required_request_values") {
		t.Fatalf("missing credential field was not reported: %#v", issues)
	}
}

func TestDeterministicPrefillLeavesNonMatchingInputUnfilled(t *testing.T) {
	session := supportTicketDraft(false)
	session.Intent.Inputs = []*rollout.Input{{Name: "requestId", Type: "string", Required: true}}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}}

	deterministicPrefill(&session, docs)
	if got := session.Intent.Steps[0].With["ticketId"]; got != "" {
		t.Fatalf("non-matching input was prefilled: %q", got)
	}
	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "missing_required_request_values") {
		t.Fatalf("missing input field was not reported: %#v", issues)
	}
}

func TestDeterministicPrefillAddsSingleStepOutput(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Outputs = nil
	if !deterministicPrefill(&session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{OperationID: "getTicket"}}}}) {
		t.Fatalf("deterministic prefill did not change session")
	}
	if len(session.Intent.Outputs) != 1 || session.Intent.Outputs[0].Name != "result" || session.Intent.Outputs[0].From != "get_ticket.received_body" {
		t.Fatalf("outputs = %#v", session.Intent.Outputs)
	}
	if !hasClassification(session.Classifications, "intent.outputs.result", "result=get_ticket.received_body", mappingSourceFallbackDefault, mappingConfidenceReview) {
		t.Fatalf("missing output fallback classification: %#v", session.Classifications)
	}
}

func TestApplyProgressiveAnswerRecordsUserHighClassification(t *testing.T) {
	session := supportTicketDraft(false)
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}}

	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps.get_ticket.with.ticketId"}}, "ticketId=inputs.ticketId", docs)

	if !hasClassification(session.Classifications, "steps.get_ticket.with.ticketId", "inputs.ticketId", mappingSourceUser, mappingConfidenceHigh) {
		t.Fatalf("missing user classification: %#v", session.Classifications)
	}
	if hasReadinessCode(CheckReadiness(session, docs), "low_confidence_mapping") {
		t.Fatalf("user high classification created low-confidence readiness issue: %#v", CheckReadiness(session, docs))
	}
}

func TestApplyProgressiveAnswerMapsOperationToSelectedStep(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{
		{Name: "openweathermap", Type: "http"},
		{Name: "gmail", Type: "http"},
	}}}
	docs := []APIDocument{
		{RelativePath: "openapi/weather.yaml", Operations: []apitools.OperationSummary{{OperationID: "getWeather", Summary: "Get current weather"}}},
		{RelativePath: "openapi/gmail.yaml", Operations: []apitools.OperationSummary{{OperationID: "sendMessage", Summary: "Send mail"}}},
	}

	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps.gmail.operation"}}, "sendMessage", docs)

	if got := session.Intent.Steps[0].Operation; got != "" {
		t.Fatalf("weather operation = %q, want unchanged", got)
	}
	if got := session.Intent.Steps[1].Operation; got != "sendMessage" {
		t.Fatalf("gmail operation = %q, want sendMessage", got)
	}
	if got := session.Intent.Steps[1].OpenAPI; got != "openapi/gmail.yaml" {
		t.Fatalf("gmail openapi = %q, want openapi/gmail.yaml", got)
	}
}

func TestApplyProgressiveAnswerMapsRequestFieldsToSelectedStep(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{
		{Name: "openweathermap", Type: "http", Provider: "openweathermap", Operation: "getOpenWeatherMapOneCall3"},
		{Name: "gmail", Type: "http", Provider: "gmail", Operation: "gmail_users_messages_send"},
	}}}

	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps.openweathermap.with"}}, "appid=inputs.appid, lat=inputs.lat, lon=inputs.lon", nil)

	if got := session.Intent.Steps[0].With["appid"]; got != "inputs.appid" {
		t.Fatalf("openweathermap appid = %q, want inputs.appid", got)
	}
	if len(session.Intent.Steps[1].With) != 0 {
		t.Fatalf("gmail fields were modified: %#v", session.Intent.Steps[1].With)
	}
}

func TestApplyProgressiveAnswerRejectsOperationFromWrongProvider(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{
		{Name: "openweathermap", Type: "http", Provider: "openweathermap"},
		{Name: "gmail", Type: "http", Provider: "gmail"},
	}}}
	docs := []APIDocument{
		{RelativePath: "discovery/gmail-discovery-v1.json", Title: "Gmail API", Operations: []apitools.OperationSummary{{OperationID: "gmail_users_getprofile", Summary: "Gets the user's Gmail profile."}}},
	}

	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps.openweathermap.operation"}}, "gmail_users_getprofile", docs)

	if got := session.Intent.Steps[0].Operation; got != "" {
		t.Fatalf("openweathermap operation = %q, want unchanged", got)
	}
	if got := session.Intent.Steps[0].OpenAPI; got != "" {
		t.Fatalf("openweathermap openapi = %q, want unchanged", got)
	}
}

func TestMissingLocalOpenAPIPathBlocksReadiness(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.OpenAPI = "openapi/missing.yaml"

	issues := CheckReadiness(session, nil)

	if !hasReadinessCode(issues, "missing_api_doc") {
		t.Fatalf("missing local OpenAPI path was not reported: %#v", issues)
	}
	if got := firstBlockingIssue(issues).Message; !strings.Contains(got, "openapi/missing.yaml") {
		t.Fatalf("missing API doc message = %q", got)
	}
}

func TestApplyCatalogDocumentAnswerRejectsMissingPath(t *testing.T) {
	session := supportTicketDraft(false)
	session.Intent.OpenAPI = "openapi/missing.yaml"
	var out strings.Builder

	handled, err := applyCatalogDocumentAnswer(&out, &session, QuestionPlan{Slots: []string{"intent.openapi"}, SuggestedAnswer: "openapi/missing.yaml"}, "", nil, t.TempDir())

	if err != nil {
		t.Fatalf("applyCatalogDocumentAnswer returned error: %v", err)
	}
	if !handled {
		t.Fatalf("missing path answer was not handled")
	}
	if session.Intent.OpenAPI != "" {
		t.Fatalf("missing path was retained as OpenAPI: %q", session.Intent.OpenAPI)
	}
	if !strings.Contains(out.String(), "local API document not found") {
		t.Fatalf("missing path message not printed:\n%s", out.String())
	}
}

func TestApplyCatalogDocumentAnswerClearsStaleMissingPath(t *testing.T) {
	session := supportTicketDraft(false)
	session.Intent.OpenAPI = "openapi/stale.yaml"
	var out strings.Builder

	handled, err := applyCatalogDocumentAnswer(&out, &session, QuestionPlan{Slots: []string{"intent.openapi"}, SuggestedAnswer: "openapi/api.yaml"}, "", nil, t.TempDir())

	if err != nil {
		t.Fatalf("applyCatalogDocumentAnswer returned error: %v", err)
	}
	if !handled {
		t.Fatalf("missing path answer was not handled")
	}
	if session.Intent.OpenAPI != "" {
		t.Fatalf("stale missing path was retained as OpenAPI: %q", session.Intent.OpenAPI)
	}
}

func TestShouldRetrieveCatalogArtifactsWhenProviderDocMissingButMigratable(t *testing.T) {
	overlay := filepath.Join(t.TempDir(), "openweathermap-one-call-3-overlay.json")
	if err := os.WriteFile(overlay, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write overlay: %v", err)
	}
	session := Session{Project: projectwizard.Answers{Goal: "get weather and gmail me"}}
	docs := []APIDocument{{RelativePath: "discovery/gmail-discovery-v1.json", Title: "Gmail API"}}
	hints := []CatalogHint{
		{Provider: catalog.Provider{ID: "gmail", DisplayName: "Gmail"}},
		{
			Provider:         catalog.Provider{ID: "openweathermap", DisplayName: "OpenWeatherMap"},
			OverlayArtifacts: []string{overlay},
		},
	}

	if !shouldRetrieveCatalogArtifactsForHints(session, docs, hints) {
		t.Fatalf("expected retrieval for missing OpenWeatherMap overlay when Gmail doc is already local")
	}
}

func TestSuggestedAPIDocAnswerReportsArtifactBlocker(t *testing.T) {
	session := Session{}
	session.Project.Goal = "get weather and gmail me"
	session.Intent.Workflow = &rollout.WorkflowMeta{Description: session.Project.Goal}
	docs := []APIDocument{{RelativePath: "discovery/gmail-discovery-v1.json", Title: "Gmail API"}}

	if got, want := suggestedAPIDocAnswer(session, docs), "Generate/provide the missing API artifact, then rerun iCoT."; got != want {
		t.Fatalf("suggested API doc answer = %q, want %q", got, want)
	}
	issues := CheckReadiness(session, docs)
	issue := readinessIssue(issues, "missing_api_doc")
	if !strings.Contains(issue.Message, "No first-class OpenAPI is available for OpenWeatherMap") {
		t.Fatalf("missing API doc message = %q", issue.Message)
	}
}

func TestOperationForStepRejectsWrongProviderOperation(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{
		Name:      "openweathermap",
		Type:      "http",
		Provider:  "openweathermap",
		Operation: "gmail_users_getprofile",
	}}}}
	docs := []APIDocument{{RelativePath: "discovery/gmail-discovery-v1.json", Title: "Gmail API", Operations: []apitools.OperationSummary{{OperationID: "gmail_users_getprofile"}}}}

	if _, ok := operationForStep(session, docs, session.Intent.Steps[0]); ok {
		t.Fatalf("gmail operation matched openweathermap step")
	}
	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "missing_operation") {
		t.Fatalf("wrong-provider operation did not produce missing_operation: %#v", issues)
	}
}

func TestSuggestedOperationAnswerRanksStepCandidates(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "get weather in Toronto and gmail me the report"},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather_report", Description: "get weather in Toronto and gmail me the report"},
			Steps: []*rollout.Step{{
				Name:     "gmail",
				Type:     "http",
				Provider: "gmail",
				Do:       "Send the weather report email.",
			}},
		},
	}
	docs := []APIDocument{{RelativePath: "discovery/gmail-discovery-v1.json", Title: "Gmail API", Operations: []apitools.OperationSummary{
		{OperationID: "gmail_users_getprofile", Summary: "Gets the user's Gmail profile."},
		{OperationID: "gmail_users_messages_send", Summary: "Sends the specified message to the recipients."},
	}}}

	if got, want := suggestedOperationAnswerForStep(session, docs, session.Intent.Steps[0]), "gmail_users_messages_send"; got != want {
		t.Fatalf("suggested operation = %q, want %q", got, want)
	}
	hint := operationChoiceHintForStep(session, docs, session.Intent.Steps[0])
	if !strings.Contains(hint, "gmail_users_messages_send") || !strings.Contains(hint, "gmail_users_getprofile") {
		t.Fatalf("operation hint did not list all candidates: %q", hint)
	}
}

func TestAdvisoryAPIDocumentTakesProviderPriority(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "get weather in Toronto"},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather", Description: "get weather in Toronto"},
			Steps: []*rollout.Step{{
				Name:     "openweathermap",
				Type:     "http",
				Provider: "openweathermap",
				Do:       "Get current weather.",
			}},
		},
	}
	docs := []APIDocument{
		{RelativePath: "openapi/openweathermap-original.json", Title: "OpenWeatherMap Original API", Operations: []apitools.OperationSummary{{OperationID: "getOriginalWeather", Summary: "Get weather from original API"}}},
		{RelativePath: "openapi/openweathermap-one-call-3-overlay.json", Title: "OpenWeatherMap One Call 3.0 Advisory Overlay", Operations: []apitools.OperationSummary{{OperationID: "getOpenWeatherMapOneCall3", Summary: "Get One Call API 3.0 weather data"}}},
	}

	filtered := filterDocsForStep(&session, docs, session.Intent.Steps[0])
	if len(filtered) != 2 || filtered[0].RelativePath != "openapi/openweathermap-one-call-3-overlay.json" {
		t.Fatalf("filtered docs priority = %#v", filtered)
	}
	if got, want := suggestedOperationAnswerForStep(session, docs, session.Intent.Steps[0]), "getOpenWeatherMapOneCall3"; got != want {
		t.Fatalf("suggested operation = %q, want %q", got, want)
	}
}

func TestProgressiveDraftErrorMessageHidesInvalidJSONDetails(t *testing.T) {
	message, isJSON := progressiveDraftErrorMessage(errors.New("json: cannot unmarshal object into Go struct field Answers.project.openapi of type string"))
	if !isJSON {
		t.Fatalf("expected invalid JSON classification")
	}
	if strings.Contains(message, "Go struct field") {
		t.Fatalf("message exposed decoder internals: %q", message)
	}
}

func TestMergeProgressiveSessionsRecordsLLMReviewClassifications(t *testing.T) {
	base := supportTicketDraft(false)
	base.Intent.Steps[0].Operation = ""
	base.Intent.Outputs = nil
	overlay := supportTicketDraft(true)
	overlay.Credentials = []string{"support_api_token"}
	overlay.CredentialsSet = true
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}}

	merged := mergeProgressiveSessions(base, overlay, docs)

	for _, want := range []struct {
		slot  string
		value string
	}{
		{"steps.get_ticket.operation", "getTicket"},
		{"steps.get_ticket.with.ticketId", "inputs.ticketId"},
		{"credentials", "support_api_token"},
		{"intent.outputs.ticket", "ticket=get_ticket.received_body"},
	} {
		if !hasClassification(merged.Classifications, want.slot, want.value, mappingSourceLLM, mappingConfidenceReview) {
			t.Fatalf("missing llm classification %s=%s: %#v", want.slot, want.value, merged.Classifications)
		}
	}
}

func TestDefaultSingleOpenAPIDocRecordsFallbackReviewClassification(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.OpenAPI = ""
	defaultSingleOpenAPIDoc(&session, []APIDocument{{RelativePath: "openapi/support.yaml"}})

	if !hasClassification(session.Classifications, "intent.source", "openapi/support.yaml", mappingSourceFallbackDefault, mappingConfidenceReview) {
		t.Fatalf("missing openapi fallback classification: %#v", session.Classifications)
	}
}

func TestMergeProgressiveSessionsPreservesSource(t *testing.T) {
	base := Session{}
	overlay := Session{Intent: rollout.Intent{Source: "google-discovery/gmail.json"}}

	merged := mergeProgressiveSessions(base, overlay, nil)

	if merged.Intent.Source != "google-discovery/gmail.json" {
		t.Fatalf("source = %q", merged.Intent.Source)
	}
}

func TestApplyCatalogDocumentAnswerSetsNativeSource(t *testing.T) {
	session := Session{}
	docs := []APIDocument{{RelativePath: "google-discovery/gmail.json"}}
	ok, err := applyCatalogDocumentAnswer(&strings.Builder{}, &session, QuestionPlan{Slots: []string{"intent.source"}}, "google-discovery/gmail.json", docs, t.TempDir())
	if err != nil || !ok {
		t.Fatalf("applyCatalogDocumentAnswer ok=%v err=%v", ok, err)
	}
	if session.Intent.Source != "google-discovery/gmail.json" {
		t.Fatalf("source = %q", session.Intent.Source)
	}
	if session.Intent.OpenAPI != "" {
		t.Fatalf("openapi alias should be empty for native source, got %q", session.Intent.OpenAPI)
	}
}

func TestReadinessFlagsConflictingMappingClassifications(t *testing.T) {
	session := supportTicketDraft(true)
	addMappingClassification(&session, MappingClassification{
		Slot:       "steps.get_ticket.with.ticketId",
		Value:      "inputs.ticketId",
		Source:     mappingSourceDeterministic,
		Confidence: mappingConfidenceHigh,
	})
	addMappingClassification(&session, MappingClassification{
		Slot:       "steps.get_ticket.with.ticketId",
		Value:      "literal-ticket",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceReview,
	})

	issues := CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}})
	if !hasReadinessCode(issues, "conflicting_mapping") {
		t.Fatalf("missing conflicting mapping issue: %#v", issues)
	}
}

func TestGroupedDefaultsUseExactRuntimeInputMatch(t *testing.T) {
	session := supportTicketDraft(false)
	issues := CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}})
	issue := readinessIssue(issues, "missing_required_request_values")
	if issue.SuggestedAnswer != "ticketId=inputs.ticketId" {
		t.Fatalf("suggested answer = %q, issues=%#v", issue.SuggestedAnswer, issues)
	}
}

func TestGroupedDefaultsUseLeafRuntimeInputMatch(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].Operation = "createOrder"
	session.Intent.Steps[0].With = nil
	session.Intent.Inputs = []*rollout.Input{{Name: "email", Type: "string", Required: true}}
	docs := []APIDocument{{RelativePath: "openapi/orders.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "createOrder",
		RequestBody: &apitools.RequestBodySummary{
			Required:           true,
			Fields:             []apitools.RequestFieldSummary{{Path: "customer.email", Required: true, Type: "string", Description: "Customer email address"}},
			RequiredFieldPaths: []string{"customer.email"},
		},
	}}}}

	issue := readinessIssue(CheckReadiness(session, docs), "missing_required_request_values")
	if issue.SuggestedAnswer != "customer.email=inputs.email" {
		t.Fatalf("suggested answer = %q", issue.SuggestedAnswer)
	}
}

func TestGroupedDefaultsUseKnownCredentialBinding(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"support_api_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{"ticketId": "inputs.ticketId"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Title: "Support API", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
		Security:    securitySummaries("BearerAuth"),
	}}}}

	issue := readinessIssue(CheckReadiness(session, docs), "missing_required_request_values")
	if issue.SuggestedAnswer != "Authorization=credentials.support_api_token" {
		t.Fatalf("suggested answer = %q", issue.SuggestedAnswer)
	}
}

func TestGroupedDefaultsDeriveCredentialBindingAndAcceptAddsIt(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = nil
	session.CredentialsSet = false
	session.Intent.Steps[0].With = map[string]string{"ticketId": "inputs.ticketId"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Title: "Support API", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
		Security:    securitySummaries("BearerAuth"),
	}}}}

	issue := readinessIssue(CheckReadiness(session, docs), "missing_required_request_values")
	if issue.SuggestedAnswer != "Authorization=credentials.support_api_token" {
		t.Fatalf("suggested answer = %q", issue.SuggestedAnswer)
	}
	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps.get_ticket.with"}}, issue.SuggestedAnswer, docs)
	if len(session.Credentials) != 1 || session.Credentials[0] != "support_api_token" {
		t.Fatalf("credentials not added from accepted mapping: %#v", session.Credentials)
	}
	if !hasClassification(session.Classifications, "credentials", "support_api_token", mappingSourceUser, mappingConfidenceHigh) {
		t.Fatalf("missing credential classification: %#v", session.Classifications)
	}
}

func TestGroupedDefaultsUseSafeRequestBodyLiterals(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].Operation = "createOrder"
	session.Intent.Steps[0].With = nil
	session.Intent.Inputs = nil
	docs := []APIDocument{{RelativePath: "openapi/orders.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "createOrder",
		RequestBody: &apitools.RequestBodySummary{
			Required: true,
			Fields: []apitools.RequestFieldSummary{
				{Path: "active", Required: true, Type: "boolean"},
				{Path: "quantity", Required: true, Type: "integer"},
				{Path: "status", Required: true, Type: "string"},
			},
			RequiredFieldPaths: []string{"active", "quantity", "status"},
		},
	}}}}

	issue := readinessIssue(CheckReadiness(session, docs), "missing_required_request_values")
	for _, expected := range []string{"active=inputs.active", "quantity=inputs.quantity", "status=inputs.status"} {
		if !strings.Contains(issue.SuggestedAnswer, expected) {
			t.Fatalf("suggested answer missing %q: %q", expected, issue.SuggestedAnswer)
		}
	}
}

func TestGroupedDefaultsDoNotUseSecretLikeLiteralDefaults(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].Operation = "createOrder"
	session.Intent.Steps[0].With = nil
	session.Intent.Inputs = nil
	docs := []APIDocument{{RelativePath: "openapi/orders.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "createOrder",
		RequestBody: &apitools.RequestBodySummary{
			Required:           true,
			Fields:             []apitools.RequestFieldSummary{{Path: "apiKey", Required: true, Type: "string"}},
			RequiredFieldPaths: []string{"apiKey"},
		},
	}}}}

	issue := readinessIssue(CheckReadiness(session, docs), "missing_required_request_values")
	if issue.SuggestedAnswer != "apiKey=inputs.apiKey" {
		t.Fatalf("suggested answer = %q", issue.SuggestedAnswer)
	}
}

func TestReadinessFlagsUndeclaredCredentialReference(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = nil
	session.Intent.Steps[0].With = map[string]string{
		"ticketId":      "inputs.ticketId",
		"Authorization": "credentials.missing_token",
	}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
		Security:    securitySummaries("BearerAuth"),
	}}}}

	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "undeclared_credential_reference") {
		t.Fatalf("missing undeclared credential issue: %#v", issues)
	}
}

func TestReadinessFlagsInventedRequestField(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].With["extra"] = "inputs.ticketId"
	issues := CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}})
	if !hasReadinessCode(issues, "invented_request_field") {
		t.Fatalf("missing invented request field issue: %#v", issues)
	}
}

func TestReadinessAcceptsKnownOpenAPIRequestFields(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"support_api_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{
		"ticketId":      "inputs.ticketId",
		"include":       "summary",
		"X-Trace-ID":    "inputs.ticketId",
		"Authorization": "credentials.support_api_token",
	}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters: []apitools.ParameterSummary{
			{Name: "ticketId", In: "path", Required: true, Type: "string"},
			{Name: "include", In: "query", Type: "string"},
			{Name: "X-Trace-ID", In: "header", Type: "string"},
		},
		Security: securitySummaries("BearerAuth"),
	}}}}

	issues := CheckReadiness(session, docs)
	for _, code := range []string{"invented_request_field", "undeclared_credential_reference", "incompatible_request_value_type"} {
		if hasReadinessCode(issues, code) {
			t.Fatalf("unexpected %s issue: %#v", code, issues)
		}
	}
}

func TestReadinessAcceptsRequiredSecurityCredentialField(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"openweathermap_one_call_3_0_advisory_overlay_api_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].Provider = "openweathermap"
	session.Intent.Steps[0].Operation = "getOpenWeatherMapOneCall3"
	session.Intent.Steps[0].With = map[string]string{
		"appid":                  "inputs.appid",
		"lat":                    "inputs.lat",
		"lon":                    "inputs.lon",
		"open_weather_a_p_i_key": "credentials.openweathermap_one_call_3_0_advisory_overlay_api_token",
	}
	docs := []APIDocument{{RelativePath: "openapi/openweathermap-one-call-3-overlay.json", Title: "OpenWeatherMap One Call 3.0 Advisory Overlay", Operations: []apitools.OperationSummary{{
		OperationID: "getOpenWeatherMapOneCall3",
		Parameters: []apitools.ParameterSummary{
			{Name: "appid", In: "query", Required: true, Type: "string"},
			{Name: "lat", In: "query", Required: true, Type: "number"},
			{Name: "lon", In: "query", Required: true, Type: "number"},
		},
		Security: securitySummaries("OpenWeatherAPIKey"),
	}}}}

	issues := CheckReadiness(session, docs)
	if hasReadinessCode(issues, "invented_request_field") {
		t.Fatalf("security credential field was treated as invented: %#v", issues)
	}
}

func TestDeterministicPrefillAddsOpenWeatherMapGeocodePrework(t *testing.T) {
	session := Session{Intent: rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Description: "get weather of Toronto, Canada, and then gmail me the report"},
		Steps: []*rollout.Step{{
			Name:      "openweathermap",
			Type:      "http",
			Provider:  "openweathermap",
			OpenAPI:   "openapi/openweathermap-one-call-3-overlay.json",
			Operation: "getOpenWeatherMapOneCall3",
			With:      map[string]string{},
		}},
	}}
	docs := []APIDocument{{RelativePath: "openapi/openweathermap-one-call-3-overlay.json", Title: "OpenWeatherMap One Call 3.0 and Geocoding Advisory Overlay", Operations: []apitools.OperationSummary{
		{
			OperationID: "getOpenWeatherMapOneCall3",
			Parameters: []apitools.ParameterSummary{
				{Name: "lat", In: "query", Required: true, Type: "number"},
				{Name: "lon", In: "query", Required: true, Type: "number"},
				{Name: "appid", In: "query", Required: true, Type: "string"},
			},
			Security: []apitools.SecuritySummary{{Name: "openWeatherAPIKey", Type: "apiKey", In: "query", ParameterName: "appid"}},
		},
		{
			OperationID: "geocodeOpenWeatherMapLocationName",
			Parameters: []apitools.ParameterSummary{
				{Name: "q", In: "query", Required: true, Type: "string"},
				{Name: "appid", In: "query", Required: true, Type: "string"},
			},
			Security: []apitools.SecuritySummary{{Name: "openWeatherAPIKey", Type: "apiKey", In: "query", ParameterName: "appid"}},
		},
	}}}

	if !deterministicPrefill(&session, docs) {
		t.Fatal("deterministic prefill did not add geocode prework")
	}
	if len(session.Intent.Steps) != 2 {
		t.Fatalf("steps = %#v, want geocode plus weather", session.Intent.Steps)
	}
	geocode := session.Intent.Steps[0]
	weather := session.Intent.Steps[1]
	if geocode.Operation != "geocodeOpenWeatherMapLocationName" || geocode.With["q"] != "Toronto, Canada" {
		t.Fatalf("geocode step = %#v", geocode)
	}
	if got := weather.DependsOn; len(got) != 1 || got[0] != geocode.Name {
		t.Fatalf("weather depends_on = %#v, want %s", got, geocode.Name)
	}
	if len(weather.Binds) != 1 || weather.Binds[0].From != geocode.Name || weather.Binds[0].Fields["lat"] != "received_body[0].lat" || weather.Binds[0].Fields["lon"] != "received_body[0].lon" {
		t.Fatalf("weather binds = %#v", weather.Binds)
	}
	if weather.With["appid"] == "" || geocode.With["appid"] == "" {
		t.Fatalf("credential parameter mappings missing: geocode=%#v weather=%#v", geocode.With, weather.With)
	}
	issues := CheckReadiness(session, docs)
	for _, issue := range issues {
		if issue.Code == "missing_required_request_values" && strings.Contains(issue.Slot, "openweathermap") {
			t.Fatalf("weather still missing request mappings: %#v", issues)
		}
	}
}

func TestReadinessValidatesRequestBodyPaths(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].Operation = "createOrder"
	session.Intent.Steps[0].With = map[string]string{
		"customer.email": "inputs.ticketId",
		"items[].sku":    "inputs.ticketId",
		"customer.phone": "inputs.ticketId",
	}
	docs := []APIDocument{{RelativePath: "openapi/orders.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "createOrder",
		RequestBody: &apitools.RequestBodySummary{
			Required: true,
			Fields: []apitools.RequestFieldSummary{
				{Path: "customer.email", Required: true, Type: "string"},
				{Path: "items[].sku", Required: true, Type: "string"},
			},
			RequiredFieldPaths: []string{"customer.email", "items[].sku"},
		},
	}}}}

	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "invalid_request_body_path") {
		t.Fatalf("missing invalid body path issue: %#v", issues)
	}
	if hasReadinessCode(issues, "missing_required_request_values") {
		t.Fatalf("valid required body paths were treated as missing: %#v", issues)
	}
}

func TestReadinessFlagsIncompatibleLiteralTypes(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].With = map[string]string{
		"ticketId": "123",
		"page":     "abc",
		"active":   "yes",
	}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters: []apitools.ParameterSummary{
			{Name: "ticketId", Required: true, Type: "string"},
			{Name: "page", Type: "integer"},
			{Name: "active", Type: "boolean"},
		},
	}}}}

	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "incompatible_request_value_type") {
		t.Fatalf("missing incompatible type issue: %#v", issues)
	}
}

func hasAssumption(assumptions []Assumption, id string) bool {
	for _, assumption := range assumptions {
		if assumption.ID == id {
			return true
		}
	}
	return false
}

func hasReadinessCode(issues []ReadinessIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func readinessIssue(issues []ReadinessIssue, code string) ReadinessIssue {
	for _, issue := range issues {
		if issue.Code == code {
			return issue
		}
	}
	return ReadinessIssue{}
}

func hasClassification(classifications []MappingClassification, slot, value, source, confidence string) bool {
	for _, classification := range classifications {
		if classification.Slot == slot && classification.Value == value && classification.Source == source && classification.Confidence == confidence {
			return true
		}
	}
	return false
}

func TestProgressiveAmbiguousOperationAsksBeforeFieldMapping(t *testing.T) {
	example := t.TempDir()
	writeMultiOperationOpenAPI(t, example)
	first := supportTicketDraft(false)
	first.Intent.Steps[0].Operation = ""
	second := supportTicketDraft(true)
	extractor := &sequenceDraftExtractor{drafts: []Session{first, second}}
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"getTicket",
		"ticketId=inputs.ticketId",
		"save",
	}, "\n") + "\n"
	var out strings.Builder
	_, err := Run(context.Background(), strings.NewReader(input), &out, Session{}, Options{
		ExampleDir: example,
		NoLLM:      false,
		Extractor:  extractor,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	operationPrompt := strings.Index(out.String(), "Which API action or workflow step should run first?")
	fieldPrompt := strings.Index(out.String(), "What values should the required request fields use?")
	if operationPrompt < 0 {
		t.Fatalf("missing operation question:\n%s", out.String())
	}
	if fieldPrompt >= 0 && fieldPrompt < operationPrompt {
		t.Fatalf("field mapping was asked before operation choice:\n%s", out.String())
	}
	if len(extractor.calls) == 0 || len(extractor.calls[0].Session.Intent.Steps) == 0 || extractor.calls[0].Session.Intent.Steps[0].Operation != "getTicket" {
		t.Fatalf("draft ran before selected operation context: %#v", extractor.calls)
	}
}

func TestProgressiveResumeStaysOnProgressivePipeline(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	seed := supportTicketDraft(false)
	seed.Intent.Steps[0].Operation = ""
	seed.Intent.Steps[0].With = nil
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(true)}}
	input := strings.Join([]string{
		"save",
	}, "\n") + "\n"
	var out strings.Builder

	_, err := Run(context.Background(), strings.NewReader(input), &out, seed, Options{
		ExampleDir:     example,
		NoLLM:          false,
		Extractor:      extractor,
		DisableAIDraft: true,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "Workflow name") || strings.Contains(out.String(), "Use OpenAPI/API steps?") {
		t.Fatalf("resume fell back to manual prompts:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "op_get_ticket") {
		t.Fatalf("resume did not repair operation through progressive readiness:\n%s", out.String())
	}
}

func TestPlanNextQuestionPriorityAndGrouping(t *testing.T) {
	session := supportTicketDraft(false)
	session.Intent.Workflow.Description = ""
	session.Project.Goal = ""
	issues := CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}})
	plan := PlanNextQuestion(session, nil, issues)
	if got := plan.Slots[0]; got != "workflow.description" {
		t.Fatalf("first slot = %q, issues=%#v", got, issues)
	}

	session.Intent.Workflow.Description = "Fetch ticket"
	issues = CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true}},
	}}}})
	plan = PlanNextQuestion(session, nil, issues)
	if !plan.Grouped || !strings.Contains(plan.Prompt, "required request fields") {
		t.Fatalf("required fields were not grouped: %#v issues=%#v", plan, issues)
	}
}

func TestGuidedSaaSOperationQuestionIncludesListedOperationIDs(t *testing.T) {
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "support", Description: "Work a support ticket."},
		},
	}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{
		{OperationID: "getTicket"},
		{OperationID: "searchTickets"},
	}}}
	issues := CheckReadiness(session, docs)
	issue := readinessIssue(issues, "missing_operation")
	if !strings.Contains(issue.Message, "Available operationIds") || !strings.Contains(issue.Message, "getTicket") {
		t.Fatalf("operation issue did not include operation choices: %#v", issue)
	}
	plan := PlanNextQuestion(session, docs, issues)
	if !strings.Contains(plan.Prompt, "listed operationId") {
		t.Fatalf("operation prompt missing listed operationId guidance: %#v", plan)
	}
}

func TestOperationQuestionOmitsDocumentPathForSingleArtifact(t *testing.T) {
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather", Description: "Get weather in Toronto."},
			Source:   "openapi/openweathermap-one-call-3-overlay.json",
			Steps: []*rollout.Step{{
				Name:     "openweathermap",
				Type:     "http",
				Provider: "openweathermap",
				Source:   "openapi/openweathermap-one-call-3-overlay.json",
			}},
		},
	}
	docs := []APIDocument{{RelativePath: "openapi/openweathermap-one-call-3-overlay.json", Operations: []apitools.OperationSummary{
		{OperationID: "getOpenWeatherMapOneCall3", Summary: "Get One Call API 3.0 weather data"},
		{OperationID: "geocodeOpenWeatherMapLocationName", Summary: "Geocode location name"},
	}}}
	issues := CheckReadiness(session, docs)
	plan := PlanNextQuestion(session, docs, issues)
	if !strings.Contains(plan.Prompt, "getOpenWeatherMapOneCall3") {
		t.Fatalf("operation prompt missing operationId: %#v", plan)
	}
	if strings.Contains(plan.Prompt, "[openapi/openweathermap-one-call-3-overlay.json]") {
		t.Fatalf("operation prompt included redundant single-document path: %s", plan.Prompt)
	}
}

func TestOperationQuestionKeepsDocumentPathForMultipleArtifacts(t *testing.T) {
	hint := operationChoicesHint([]rankedOperationChoice{
		{
			Doc: APIDocument{RelativePath: "openapi/support.yaml"},
			Op:  apitools.OperationSummary{OperationID: "getTicket"},
		},
		{
			Doc: APIDocument{RelativePath: "openapi/tickets.yaml"},
			Op:  apitools.OperationSummary{OperationID: "getTicket"},
		},
	})
	if !strings.Contains(hint, "[openapi/support.yaml]") || !strings.Contains(hint, "[openapi/tickets.yaml]") {
		t.Fatalf("operation hint should keep paths for multi-document choices: %s", hint)
	}
}

func TestCatalogMatchedWorkflowBlocksOnMissingOpenAPI(t *testing.T) {
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{
				Name:        "weather_gmail",
				Description: "get weather in Toronto, and gmail the report to me",
			},
		},
	}
	issues := CheckReadiness(session, nil)
	issue := readinessIssue(issues, "missing_api_doc")
	if issue.Code == "" {
		t.Fatalf("missing_api_doc was not reported: %#v", issues)
	}
	if !strings.Contains(issue.Message, "OpenWeatherMap -> Gmail") || !strings.Contains(issue.Message, "cannot continue to operation selection") {
		t.Fatalf("missing_api_doc did not describe catalog blocker: %#v", issue)
	}
	plan := PlanNextQuestion(session, nil, issues)
	if !strings.Contains(plan.Prompt, "OpenWeatherMap -> Gmail") || strings.Contains(plan.Prompt, "operationId") {
		t.Fatalf("catalog blocker prompt = %#v", plan)
	}
	var out strings.Builder
	printSummary(&out, session)
	if !strings.Contains(out.String(), "API documents: not local yet; catalog providers matched OpenWeatherMap -> Gmail") {
		t.Fatalf("summary did not report unresolved catalog OpenAPI:\n%s", out.String())
	}
}

func TestSuggestedPolicyAnswerDistinguishesReadOnlyAndSideEffects(t *testing.T) {
	readOnly := supportTicketDraft(true)
	readOnly.Safety = ""
	readOnly.SafetySet = false
	readOnly.SideEffectScope = ""
	if got := suggestedPolicyAnswer(readOnly); got != projectwizard.SideEffectReadOnly {
		t.Fatalf("read-only policy suggestion = %q", got)
	}

	readBlogPost := readOnly
	readBlogPost.Intent.Steps[0].Operation = "getPost"
	readBlogPost.Intent.Steps[0].Do = "Fetch a blog post."
	if got := suggestedPolicyAnswer(readBlogPost); got != projectwizard.SideEffectReadOnly {
		t.Fatalf("read-only getPost policy suggestion = %q", got)
	}

	write := readOnly
	write.Intent.Steps[0].Operation = "sendMessage"
	write.Intent.Steps[0].Do = "Send a customer message."
	if got := suggestedPolicyAnswer(write); got != projectwizard.SideEffectSandboxOnly {
		t.Fatalf("write policy suggestion = %q", got)
	}
}

func TestProgressiveTranscriptIncludesEvents(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	path := filepath.Join(example, ".icot", "transcript.json")
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(true)}}
	input := "Fetch a support ticket.\ngetTicket\nsave\n"
	if _, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir:     example,
		NoLLM:          false,
		Extractor:      extractor,
		TranscriptPath: path,
	}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"model_draft_call", "readiness_decision", "next_question_decision", "final_generated_artifacts"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("transcript missing %q:\n%s", expected, text)
		}
	}
}

func TestProgressiveWeatherGmailDraftAddsGeocodingOperation(t *testing.T) {
	example := t.TempDir()
	writeWeatherGmailOpenAPI(t, example)
	path := filepath.Join(example, ".icot", "transcript.json")
	chat := &sequenceChat{responses: []string{
		`{"requested_operation_ids":["geocodeCity"],"detail_request_reason":"The selected weather operation requires latitude and longitude, but the user gave Toronto."}`,
		`{
  "intent": {
    "openapi": "openapi/weather.yaml",
    "workflow": {"name":"weather_toronto_gmail","description":"Get weather for Toronto, Canada, and Gmail me the report."},
    "inputs": [],
    "steps": [
      {"name":"geocode_city","type":"http","openapi":"openapi/weather.yaml","operation":"geocodeCity","with":{"q":"Toronto,CA"}},
      {"name":"get_weather","type":"http","openapi":"openapi/weather.yaml","operation":"getWeatherByLatLon","with":{"lat":"geocode_city.received_body.lat","lon":"geocode_city.received_body.lon","appid":"credentials.weather_api_key"}},
      {"name":"send_gmail","type":"http","openapi":"openapi/gmail.yaml","operation":"gmail_users_messages_send","with":{"userId":"me","raw":"get_weather.received_body"}}
    ],
    "outputs": [{"name":"sent_message","from":"send_gmail.received_body"}]
  },
  "credentials": ["weather_api_key","gmail_oauth_token"],
  "credentials_set": true,
  "safety": "Send only after approval with reviewed recipients.",
  "safety_set": true,
  "side_effect_scope": "after-approval",
  "assumptions": [
    {"id":"op_geocode_city","slot":"steps.geocode_city.operation","value":"geocodeCity","reason":"Toronto needs coordinate resolution before weather-by-lat/lon.","evidence":"geocodeCity was listed in the local catalog and details were fetched.","risk":"review","requires_confirmation":true},
    {"id":"op_send_gmail","slot":"steps.send_gmail.operation","value":"gmail_users_messages_send","reason":"The brief asks to Gmail the report.","evidence":"selected Gmail operation details.","risk":"review","requires_confirmation":true}
  ]
}`,
	}}
	seed := weatherGmailDraftRequest().Session
	var out strings.Builder
	artifacts, err := Run(context.Background(), strings.NewReader("geocodeCity\nsave\n"), &out, seed, Options{
		ExampleDir:     example,
		NoLLM:          false,
		Extractor:      NewChatExtractor(chat, nil),
		TranscriptPath: path,
	})
	if err != nil {
		t.Fatalf("Run failed: %v\n%s", err, out.String())
	}
	if len(artifacts.Session.Intent.Steps) != 3 || !sessionHasOperation(artifacts.Session, "geocodeCity") {
		t.Fatalf("draft did not add geocode step: %#v", artifacts.Session.Intent.Steps)
	}
	if !strings.Contains(artifacts.IntentHCL, `operation = "geocodeCity"`) || !strings.Contains(artifacts.IntentHCL, `operation = "gmail_users_messages_send"`) {
		t.Fatalf("intent missing expected operations:\n%s", artifacts.IntentHCL)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"geocodeCity", "gmail_users_messages_send"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("transcript missing %q:\n%s", expected, text)
		}
	}
}

func TestProgressiveTranscriptIncludesOperationDetailEvents(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	path := filepath.Join(example, ".icot", "transcript.json")
	draft := supportTicketDraft(true)
	draft.DraftEvents = []TranscriptEvent{
		{Kind: "operation_detail_request", Data: map[string]any{"operation_ids": []string{"getTicket"}}},
		{Kind: "operation_detail_fulfilled", Data: map[string]any{"operation_ids": []string{"getTicket"}}},
	}
	extractor := &sequenceDraftExtractor{drafts: []Session{draft}}
	input := "Fetch a support ticket.\ngetTicket\nsave\n"
	if _, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir:     example,
		NoLLM:          false,
		Extractor:      extractor,
		TranscriptPath: path,
	}); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"operation_detail_request", "operation_detail_fulfilled", "getTicket"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("transcript missing %q:\n%s", expected, text)
		}
	}
}

func TestProgressiveTranscriptIncludesCatalogPlanEvents(t *testing.T) {
	example := t.TempDir()
	cacheRoot := t.TempDir()
	writeGmailDiscoveryCatalogArtifact(t, cacheRoot)
	path := filepath.Join(example, ".icot", "transcript.json")
	input := strings.Join([]string{
		"Email me a report with Gmail.",
		"",
		"userId=me, raw=inputs.raw",
		"after-approval",
		"save",
	}, "\n") + "\n"

	_, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir:         example,
		NoLLM:              false,
		Extractor:          catalogPlanningExtractor{},
		TranscriptPath:     path,
		CatalogHintOptions: CatalogHintOptions{CacheRoot: cacheRoot},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"catalog_plan_call", "catalog_plan_result", "gmail:google-discovery/gmail-discovery-v1.json"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("transcript missing %q:\n%s", expected, text)
		}
	}
}

func writeGmailDiscoveryCatalogArtifact(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "google-discovery", "gmail-discovery-v1.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir discovery artifact dir: %v", err)
	}
	data := `{
	  "kind": "discovery#restDescription",
	  "name": "gmail",
	  "title": "Gmail API",
	  "version": "v1",
	  "rootUrl": "https://gmail.googleapis.com/",
	  "servicePath": "",
	  "resources": {
	    "users": {
	      "resources": {
	        "messages": {
	          "methods": {
	            "send": {
	              "id": "gmail.users.messages.send",
	              "path": "gmail/v1/users/{userId}/messages/send",
	              "httpMethod": "POST",
	              "description": "Sends the specified message to the recipients.",
	              "parameters": {
	                "userId": {"type": "string", "location": "path", "required": true}
	              },
	              "request": {"$ref": "Message"}
	            }
	          }
	        }
	      }
	    }
	  }
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write discovery artifact: %v", err)
	}
}

func supportTicketDraft(withField bool) Session {
	step := &rollout.Step{
		Name:      "get_ticket",
		Type:      "http",
		Do:        "Fetch the ticket.",
		Operation: "getTicket",
	}
	if withField {
		step.With = map[string]string{"ticketId": "inputs.ticketId"}
	}
	return Session{
		Intent: rollout.Intent{
			OpenAPI:  "openapi/support.yaml",
			Workflow: &rollout.WorkflowMeta{Name: "support_ticket_lookup", Description: "Fetch a support ticket by runtime id."},
			Inputs:   []*rollout.Input{{Name: "ticketId", Type: "string", Required: true}},
			Steps:    []*rollout.Step{step},
			Outputs:  []*rollout.Output{{Name: "ticket", From: "get_ticket.received_body"}},
		},
		Safety:          "Sandbox proof runs only.",
		SafetySet:       true,
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Assumptions: []Assumption{{
			ID:                   "op_get_ticket",
			Slot:                 "steps.get_ticket.operation",
			Value:                "getTicket",
			Reason:               "The support ticket lookup operation matched the brief.",
			Evidence:             "Support API getTicket",
			Risk:                 "low",
			RequiresConfirmation: true,
		}},
	}
}

func sessionHasOperation(session Session, operationID string) bool {
	found := false
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step != nil && step.Operation == operationID {
			found = true
		}
	})
	return found
}

func TestSessionNormalizeExplicitPolicyMarkersReplaceSeededProjectValues(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{
			ProjectName: "Policy",
			Credentials: []string{"old_token"},
			Safety:      "Old safety note",
			Fallback:    "Old fallback note",
		},
		Intent:         rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "policy", Description: "Test policy edits"}},
		Credentials:    []string{"new_token"},
		CredentialsSet: true,
		Safety:         "",
		SafetySet:      true,
		Fallback:       "New fallback note",
		FallbackSet:    true,
	}
	session.Normalize()
	if len(session.Project.Credentials) != 1 || session.Project.Credentials[0] != "new_token" {
		t.Fatalf("credentials were not replaced: %#v", session.Project.Credentials)
	}
	if session.Project.Safety != "" || session.Safety != "" {
		t.Fatalf("safety was not cleared: project=%q top=%q", session.Project.Safety, session.Safety)
	}
	if session.Project.Fallback != "New fallback note" || session.Fallback != "New fallback note" {
		t.Fatalf("fallback was not replaced: project=%q top=%q", session.Project.Fallback, session.Fallback)
	}
}

func writeOpenAPI(t *testing.T, example string) {
	t.Helper()
	dir := filepath.Join(example, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir openapi: %v", err)
	}
	data := `openapi: 3.0.0
info:
  title: Support API
  version: "1.0"
paths:
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      summary: Get a support ticket
      parameters:
        - name: ticketId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(filepath.Join(dir, "support.yaml"), []byte(data), 0o644); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
}

func writeMultiOperationOpenAPI(t *testing.T, example string) {
	t.Helper()
	dir := filepath.Join(example, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir openapi: %v", err)
	}
	data := `openapi: 3.0.0
info:
  title: Support API
  version: "1.0"
paths:
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      summary: Get a support ticket
      parameters:
        - name: ticketId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
  /tickets/search:
    get:
      operationId: searchTickets
      summary: Search support tickets
      parameters:
        - name: query
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(filepath.Join(dir, "support.yaml"), []byte(data), 0o644); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
}

func writeWeatherGmailOpenAPI(t *testing.T, example string) {
	t.Helper()
	dir := filepath.Join(example, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir openapi: %v", err)
	}
	weather := `openapi: 3.0.0
info:
  title: Weather API
  version: "1.0"
paths:
  /weather:
    get:
      operationId: getWeatherByLatLon
      summary: Get current weather by coordinates
      parameters:
        - name: lat
          in: query
          required: true
          schema: {type: number}
        - name: lon
          in: query
          required: true
          schema: {type: number}
        - name: appid
          in: query
          required: true
          schema: {type: string}
      responses:
        "200":
          description: ok
  /geo/1.0/direct:
    get:
      operationId: geocodeCity
      summary: Resolve a city to coordinates
      parameters:
        - name: q
          in: query
          required: true
          schema: {type: string}
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(filepath.Join(dir, "weather.yaml"), []byte(weather), 0o644); err != nil {
		t.Fatalf("write weather openapi: %v", err)
	}
	gmail := `openapi: 3.0.0
info:
  title: Gmail API
  version: "1.0"
paths:
  /gmail/v1/users/{userId}/messages/send:
    post:
      operationId: gmail_users_messages_send
      summary: Send a Gmail message
      parameters:
        - name: userId
          in: path
          required: true
          schema: {type: string}
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [raw]
              properties:
                raw:
                  type: string
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(filepath.Join(dir, "gmail.yaml"), []byte(gmail), 0o644); err != nil {
		t.Fatalf("write gmail openapi: %v", err)
	}
}
