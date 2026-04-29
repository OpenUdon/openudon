package elicitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
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
		"support_ticket_lookup",
		"Fetch a support ticket by runtime id.",
		"",
		"",
		"yes",
		"1",
		"sandbox-only",
		"",
		"",
		"",
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

type sequenceDraftExtractor struct {
	noopExtractor
	drafts []Session
	calls  []DraftRequest
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

func TestProgressiveOneAnswerJumpsToConfirmation(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(true)}}
	input := strings.Join([]string{
		"Fetch a support ticket by runtime id.",
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
	second := supportTicketDraft(true)
	extractor := &sequenceDraftExtractor{drafts: []Session{first, second}}
	input := strings.Join([]string{
		"Fetch a support ticket.",
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
	if !strings.Contains(out.String(), "What values should the required request fields use?") {
		t.Fatalf("missing grouped required-field question:\n%s", out.String())
	}
	if len(extractor.calls) < 2 || len(extractor.calls[1].ReadinessFeedback) == 0 {
		t.Fatalf("second draft did not receive readiness feedback: %#v", extractor.calls)
	}
}

func TestProgressiveDeterministicPrefillSkipsRequiredFieldQuestion(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(false)}}
	input := strings.Join([]string{
		"Fetch a support ticket.",
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
}

func TestDeterministicPrefillUsesSingleCredentialBinding(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"support_api_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{"ticketId": "inputs.ticketId"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
		Security:    []string{"BearerAuth"},
	}}}}

	if !deterministicPrefill(&session, docs) {
		t.Fatalf("deterministic prefill did not change session")
	}
	if got := session.Intent.Steps[0].With["Authorization"]; got != "credentials.support_api_token" {
		t.Fatalf("Authorization prefill = %q", got)
	}
}

func TestDeterministicPrefillLeavesAmbiguousCredentialUnfilled(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = []string{"first_token", "second_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{"ticketId": "inputs.ticketId"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
		Security:    []string{"BearerAuth"},
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
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
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
	if !deterministicPrefill(&session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{OperationID: "getTicket"}}}}) {
		t.Fatalf("deterministic prefill did not change session")
	}
	if len(session.Intent.Outputs) != 1 || session.Intent.Outputs[0].Name != "result" || session.Intent.Outputs[0].From != "get_ticket.received_body" {
		t.Fatalf("outputs = %#v", session.Intent.Outputs)
	}
}

func TestReadinessFlagsUndeclaredCredentialReference(t *testing.T) {
	session := supportTicketDraft(true)
	session.Credentials = nil
	session.Intent.Steps[0].With = map[string]string{
		"ticketId":      "inputs.ticketId",
		"Authorization": "credentials.missing_token",
	}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
		Security:    []string{"BearerAuth"},
	}}}}

	issues := CheckReadiness(session, docs)
	if !hasReadinessCode(issues, "undeclared_credential_reference") {
		t.Fatalf("missing undeclared credential issue: %#v", issues)
	}
}

func TestReadinessFlagsInventedRequestField(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps[0].With["extra"] = "inputs.ticketId"
	issues := CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
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
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters: []*rollout.ParameterInfo{
			{Name: "ticketId", In: "path", Required: true, Type: "string"},
			{Name: "include", In: "query", Type: "string"},
			{Name: "X-Trace-ID", In: "header", Type: "string"},
		},
		Security: []string{"BearerAuth"},
	}}}}

	issues := CheckReadiness(session, docs)
	for _, code := range []string{"invented_request_field", "undeclared_credential_reference", "incompatible_request_value_type"} {
		if hasReadinessCode(issues, code) {
			t.Fatalf("unexpected %s issue: %#v", code, issues)
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
	docs := []APIDocument{{RelativePath: "openapi/orders.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "createOrder",
		RequestBody: &rollout.RequestBodyInfo{
			Required: true,
			Schema: map[string]any{
				"type":     "object",
				"required": []any{"customer", "items"},
				"properties": map[string]any{
					"customer": map[string]any{
						"type":     "object",
						"required": []any{"email"},
						"properties": map[string]any{
							"email": map[string]any{"type": "string"},
						},
					},
					"items": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":     "object",
							"required": []any{"sku"},
							"properties": map[string]any{
								"sku": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
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
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters: []*rollout.ParameterInfo{
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
}

func TestPlanNextQuestionPriorityAndGrouping(t *testing.T) {
	session := supportTicketDraft(false)
	session.Intent.Workflow.Description = ""
	session.Project.Goal = ""
	issues := CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
	}}}})
	plan := PlanNextQuestion(session, nil, issues)
	if got := plan.Slots[0]; got != "workflow.description" {
		t.Fatalf("first slot = %q, issues=%#v", got, issues)
	}

	session.Intent.Workflow.Description = "Fetch ticket"
	issues = CheckReadiness(session, []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []*rollout.OperationInfo{{
		OperationID: "getTicket",
		Parameters:  []*rollout.ParameterInfo{{Name: "ticketId", Required: true}},
	}}}})
	plan = PlanNextQuestion(session, nil, issues)
	if !plan.Grouped || !strings.Contains(plan.Prompt, "required request fields") {
		t.Fatalf("required fields were not grouped: %#v issues=%#v", plan, issues)
	}
}

func TestProgressiveTranscriptIncludesEvents(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	path := filepath.Join(example, ".icot", "transcript.json")
	extractor := &sequenceDraftExtractor{drafts: []Session{supportTicketDraft(true)}}
	input := "Fetch a support ticket.\nsave\n"
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
