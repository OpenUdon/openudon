package elicitor

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestDraftReviewIssuesAreAdvisoryWarnings(t *testing.T) {
	issues := draftReviewReadinessIssues(DraftReviewResponse{Issues: []DraftReviewIssue{{
		Severity: "blocking",
		Code:     "disconnected_report",
		Message:  "Email step does not consume report content.",
		Slot:     "steps.gmail.with.raw",
	}}})
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want one issue", issues)
	}
	if issues[0].Severity != readinessWarning {
		t.Fatalf("severity = %q, want warning", issues[0].Severity)
	}
	if issues[0].Code != "llm_flow_review_disconnected_report" {
		t.Fatalf("code = %q", issues[0].Code)
	}
}

func TestSortReadinessIssuesKeepsUnknownWarningsAfterKnownWarnings(t *testing.T) {
	issues := sortReadinessIssues([]ReadinessIssue{
		{Severity: readinessWarning, Code: "llm_flow_review_disconnected_report", Message: "review"},
		{Severity: readinessWarning, Code: "missing_side_effect_policy", Message: "policy"},
	})
	if issues[0].Code != "missing_side_effect_policy" {
		t.Fatalf("first issue = %q, want known deterministic warning first", issues[0].Code)
	}
}

func TestSanitizeDraftReviewResponseFiltersPrefixesAndCaps(t *testing.T) {
	response := DraftReviewResponse{Issues: []DraftReviewIssue{
		{Code: "missing_message"},
		{Severity: "blocking", Code: "bad raw", Message: strings.Repeat("x", 300), Slot: "steps.gmail.with.raw", SuggestedAnswer: "raw=render.body", Evidence: "gmail.raw missing report"},
		{Code: "second", Message: "second"},
		{Code: "third", Message: "third"},
		{Code: "fourth", Message: "fourth"},
		{Code: "fifth", Message: "fifth"},
		{Code: "sixth", Message: "sixth"},
	}}

	sanitized := sanitizeDraftReviewResponse(response)
	if len(sanitized.Issues) != 5 {
		t.Fatalf("issues = %d, want cap of 5: %#v", len(sanitized.Issues), sanitized.Issues)
	}
	first := sanitized.Issues[0]
	if first.Severity != readinessWarning || first.Code != "llm_flow_review_bad_raw" {
		t.Fatalf("first issue = %#v", first)
	}
	if len(first.Message) > 240 || first.Evidence != "gmail.raw missing report" {
		t.Fatalf("sanitized first issue = %#v", first)
	}
}

func TestDraftReviewIssueClassificationFallbacks(t *testing.T) {
	tests := []struct {
		name   string
		issue  DraftReviewIssue
		kind   string
		action string
	}{
		{
			name:   "missing transform",
			issue:  DraftReviewIssue{Code: "output_transport_response", Message: "The output returns raw transport data instead of the requested receipt report.", Slot: "intent.outputs.result"},
			kind:   flowGapMissingTransformStep,
			action: remediationProposeFnctStep,
		},
		{
			name:   "api prework",
			issue:  DraftReviewIssue{Code: "missing_lookup", Message: "Resolve the customer id before this request.", Slot: "steps.update.with.customerId"},
			kind:   flowGapMissingAPIPrework,
			action: remediationProposeAPIPrework,
		},
		{
			name:   "notification",
			issue:  DraftReviewIssue{Code: "slack_disconnected", Message: "Slack message does not consume the generated summary.", Slot: "steps.slack.with.text"},
			kind:   flowGapDisconnectedNotification,
			action: remediationCommentOnly,
		},
		{
			name:   "ambiguous output",
			issue:  DraftReviewIssue{Code: "ambiguous_output", Message: "It is unclear which output should replace the raw transport response.", Slot: "intent.outputs.result"},
			kind:   flowGapAmbiguousOutput,
			action: remediationAskUser,
		},
		{
			name:   "operation mismatch",
			issue:  DraftReviewIssue{Code: "wrong_operation", Message: "The selected operation does not match the goal.", Slot: "steps.send.operation"},
			kind:   flowGapOperationMismatch,
			action: remediationCommentOnly,
		},
		{
			name:   "unavailable source",
			issue:  DraftReviewIssue{Code: "missing_artifact", Message: "The required API artifact is not available locally.", Slot: "intent.source"},
			kind:   flowGapUnavailableSource,
			action: remediationCommentOnly,
		},
		{
			name:   "unclear intent",
			issue:  DraftReviewIssue{Code: "unclear", Message: "The user intent is underspecified.", Slot: "workflow.description"},
			kind:   flowGapUnclearIntent,
			action: remediationCommentOnly,
		},
		{
			name:   "narrow repair",
			issue:  DraftReviewIssue{Code: "missing_dep", Message: "Add dependency.", Slot: "steps.send.depends_on", SuggestedAnswer: "depends_on=render_report"},
			kind:   flowGapNarrowRepair,
			action: remediationApplyNarrowRepair,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeDraftReviewResponse(DraftReviewResponse{Issues: []DraftReviewIssue{tt.issue}})
			if len(got.Issues) != 1 {
				t.Fatalf("issues = %#v", got.Issues)
			}
			if got.Issues[0].GapKind != tt.kind || got.Issues[0].RemediationAction != tt.action {
				t.Fatalf("remediation = %s/%s, want %s/%s", got.Issues[0].GapKind, got.Issues[0].RemediationAction, tt.kind, tt.action)
			}
		})
	}
}

func TestDraftReviewRejectsInvalidSuppliedRemediation(t *testing.T) {
	got := sanitizeDraftReviewResponse(DraftReviewResponse{Issues: []DraftReviewIssue{{
		Code:              "wrong_operation",
		Message:           "The selected operation does not match the goal.",
		Slot:              "steps.send.operation",
		GapKind:           "missing_transform_step",
		RemediationAction: "rewrite_operation",
	}}})
	if len(got.Issues) != 1 {
		t.Fatalf("issues = %#v", got.Issues)
	}
	if got.Issues[0].GapKind != flowGapOperationMismatch || got.Issues[0].RemediationAction != remediationCommentOnly {
		t.Fatalf("invalid remediation was not reclassified: %#v", got.Issues[0])
	}
}

func TestDraftReviewRejectsInvalidGapActionPair(t *testing.T) {
	tests := []struct {
		name   string
		issue  DraftReviewIssue
		kind   string
		action string
	}{
		{
			name: "operation mismatch cannot narrow repair",
			issue: DraftReviewIssue{
				Code:              "wrong_operation",
				Message:           "The selected operation does not match the goal.",
				Slot:              "steps.send.operation",
				GapKind:           flowGapOperationMismatch,
				RemediationAction: remediationApplyNarrowRepair,
			},
			kind:   flowGapOperationMismatch,
			action: remediationCommentOnly,
		},
		{
			name: "unavailable source cannot fnct repair",
			issue: DraftReviewIssue{
				Code:              "missing_artifact",
				Message:           "The required API source is not available locally.",
				Slot:              "intent.source",
				GapKind:           flowGapUnavailableSource,
				RemediationAction: remediationProposeFnctStep,
			},
			kind:   flowGapUnavailableSource,
			action: remediationCommentOnly,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeDraftReviewResponse(DraftReviewResponse{Issues: []DraftReviewIssue{tt.issue}})
			if len(got.Issues) != 1 {
				t.Fatalf("issues = %#v", got.Issues)
			}
			if got.Issues[0].GapKind != tt.kind || got.Issues[0].RemediationAction != tt.action {
				t.Fatalf("remediation = %s/%s, want %s/%s", got.Issues[0].GapKind, got.Issues[0].RemediationAction, tt.kind, tt.action)
			}
		})
	}
}

func TestAnnotateIntentHCLWithFlowReviewWarningsAnchorsToSlots(t *testing.T) {
	intent := rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "report", Description: "Send a report."},
		Steps: []*rollout.Step{{
			Name: "gmail",
			Type: "http",
			Do:   "Send the report.",
		}},
		Outputs: []*rollout.Output{{Name: "result", From: "gmail.received_body"}},
	}
	hcl, err := rollout.RenderIntentHCL(&intent)
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	annotated := annotateIntentHCLWithFlowReviewWarnings(hcl, []DraftReviewIssue{{
		Severity:        readinessWarning,
		Code:            "llm_flow_review_disconnected_report",
		Message:         "Gmail raw body does not use the report.",
		Slot:            "steps.gmail.with.raw",
		Evidence:        "raw missing",
		SuggestedAnswer: "Bind raw to render_report output.",
	}})
	if _, err := rollout.ParseIntent([]byte(annotated), "intent.hcl"); err != nil {
		t.Fatalf("parse annotated intent: %v\n%s", err, annotated)
	}
	comment := "# iCoT flow review warning (llm_flow_review_disconnected_report)"
	step := `step "gmail" {`
	if !strings.Contains(annotated, comment) || !strings.Contains(annotated, "# Evidence: raw missing") {
		t.Fatalf("missing review comment:\n%s", annotated)
	}
	if strings.Index(annotated, comment) > strings.Index(annotated, step) {
		t.Fatalf("review comment not anchored before step:\n%s", annotated)
	}
}

func TestReviewFinalDraftSkipsNoSourceCapabilityGapFallback(t *testing.T) {
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "gap", Description: "Stop and render a capability gap report."},
			Inputs: []*rollout.Input{
				{Name: "provider", Type: "string", Required: true},
				{Name: "action", Type: "string", Required: true},
			},
			Steps: []*rollout.Step{{
				Name: "render_capability_gap",
				Type: "fnct",
				Do:   "Render a capability gap report for the missing provider action.",
				With: map[string]string{"provider": "inputs.provider", "action": "inputs.action"},
			}},
			Outputs: []*rollout.Output{{Name: "gap_report", From: "render_capability_gap.received_body"}},
		},
		DecisionEvidence: []DecisionEvidence{{
			Stage:  decisionStageCatalogPlan,
			Slot:   "intent.steps.render_capability_gap",
			Value:  "no-source capability gap fallback",
			Source: mappingSourceDeterministic,
		}},
	}
	extractor := &sequenceDraftExtractor{draftReviewResponse: DraftReviewResponse{Issues: []DraftReviewIssue{{
		Code:              "ambiguous_output",
		Message:           "Do not surface this model false positive.",
		Slot:              "intent.outputs.gap_report",
		GapKind:           flowGapAmbiguousOutput,
		RemediationAction: remediationAskUser,
	}}}}
	var out strings.Builder
	var events []TranscriptEvent
	issues := reviewFinalDraft(context.Background(), &out, extractor, &session, nil, nil, &events)
	if len(issues) != 0 {
		t.Fatalf("review issues = %#v", issues)
	}
	if len(extractor.draftReviewRequests) != 0 {
		t.Fatalf("review extractor was called for deterministic fallback")
	}
	if !hasDraftReviewSkippedEvent(events) {
		t.Fatalf("missing skipped review event: %#v", events)
	}
}

func TestReviewFinalDraftSuppressesLocalFnctTransportOutputFalsePositive(t *testing.T) {
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "audit_receipt", Description: "Send a Gmail message and render an audit receipt."},
			Steps: []*rollout.Step{
				{
					Name:      "send_message",
					Type:      "http",
					OpenAPI:   "openapi/gmail.yaml",
					Operation: "sendMessage",
				},
				{
					Name:      "render_audit_receipt",
					Type:      "fnct",
					Do:        "Locally render audit receipt from send_message output for review.",
					DependsOn: []string{"send_message"},
					Binds: []*rollout.StepBind{{
						From: "send_message",
						Fields: map[string]string{
							"id":       "received_body.id",
							"threadId": "received_body.threadId",
						},
					}},
				},
			},
			Outputs: []*rollout.Output{{Name: "audit_receipt", From: "render_audit_receipt.received_body"}},
		},
	}
	extractor := &sequenceDraftExtractor{draftReviewResponse: DraftReviewResponse{Issues: []DraftReviewIssue{{
		Code:              "ambiguous_output",
		Message:           "The workflow output is wired to render_audit_receipt.received_body, which looks like a transport-style body rather than the local audit receipt content requested by the goal.",
		Slot:              "intent.outputs.audit_receipt",
		Evidence:          "Goal asks to render a local audit receipt, but the declared output comes from the fnct step's received_body instead of an explicit rendered receipt result.",
		GapKind:           flowGapAmbiguousOutput,
		RemediationAction: remediationApplyNarrowRepair,
	}}}}
	var out strings.Builder
	var events []TranscriptEvent
	issues := reviewFinalDraft(context.Background(), &out, extractor, &session, nil, nil, &events)
	if len(issues) != 0 {
		t.Fatalf("review issues = %#v", issues)
	}
	if len(extractor.draftReviewRequests) != 1 {
		t.Fatalf("review extractor calls = %d, want 1", len(extractor.draftReviewRequests))
	}
	if len(session.DecisionEvidence) != 0 {
		t.Fatalf("false-positive review issue added decision evidence: %#v", session.DecisionEvidence)
	}
}

func TestReviewFinalDraftAddsLocalMissingRenderedBodyIssueWithoutLLM(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "Fetch weather and email me a report."},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather_email", Description: "Fetch weather and email me a report."},
			Steps: []*rollout.Step{
				{Name: "get_weather", Type: "http", OpenAPI: "openapi/weather.yaml", Operation: "getWeather"},
				{Name: "send_email", Type: "http", OpenAPI: "openapi/email.yaml", Operation: "sendEmail", With: map[string]string{"userId": "me"}},
			},
		},
	}
	docs := []APIDocument{{
		RelativePath: "openapi/email.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "sendEmail",
			RequestBody: &apitools.RequestBodySummary{
				Required: true,
				Fields:   []apitools.RequestFieldSummary{{Path: "raw", Required: true, Type: "string"}},
			},
		}},
	}}
	issues := reviewFinalDraft(context.Background(), io.Discard, nil, &session, docs, nil, nil)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want one local issue", issues)
	}
	if issues[0].Slot != "steps.send_email.with.raw" || issues[0].RemediationAction != remediationProposeFnctStep {
		t.Fatalf("issue = %#v", issues[0])
	}
}
