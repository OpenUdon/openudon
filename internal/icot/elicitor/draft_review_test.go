package elicitor

import "testing"

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
