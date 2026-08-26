package browserscenario

import (
	"context"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

func TestValidateBAPBCPQualificationEvidenceRejectsMissingDuplicateAndMalformedDigests(t *testing.T) {
	valid := func(character string) string { return "sha256:" + strings.Repeat(character, 64) }
	evidence := BAPBCPQualificationEvidence{
		ProducerResultSHA256: valid("1"), TransactionSHA256: valid("2"), PreparationSHA256: valid("3"),
		QualificationSHA256: valid("4"), GenerationSHA256: valid("5"), SelectionSHA256: valid("6"),
		PackageSHA256: valid("7"), HandoffSHA256: valid("8"), WorkflowSHA256: valid("9"), EvidenceCount: 9,
	}
	if err := ValidateBAPBCPQualificationEvidence(evidence); err != nil {
		t.Fatal(err)
	}
	evidence.WorkflowSHA256 = evidence.HandoffSHA256
	if err := ValidateBAPBCPQualificationEvidence(evidence); err == nil {
		t.Fatal("duplicate qualification digest was accepted")
	}
	evidence.WorkflowSHA256 = "private/result/path"
	if err := ValidateBAPBCPQualificationEvidence(evidence); err == nil {
		t.Fatal("non-digest qualification evidence was accepted")
	}
}

func TestTaggedQualificationSHA256NormalizesOnlyValidatedDigests(t *testing.T) {
	bare := strings.Repeat("A", 64)
	if got, want := taggedQualificationSHA256(bare), "sha256:"+strings.ToLower(bare); got != want {
		t.Fatalf("normalized bare digest = %q, want %q", got, want)
	}
	tagged := "sha256:" + strings.Repeat("b", 64)
	if got := taggedQualificationSHA256(tagged); got != tagged {
		t.Fatalf("normalized tagged digest = %q, want %q", got, tagged)
	}
	for _, value := range []string{"private/result/path", " " + bare, strings.Repeat("c", 63)} {
		if got := taggedQualificationSHA256(value); got != "" {
			t.Fatalf("invalid digest %q normalized to %q", value, got)
		}
	}
}

func TestClosedPackageLifecycleFailureUsesTypedCodes(t *testing.T) {
	_, qualificationErr := packagepipeline.Qualify(context.Background(), packagepipeline.Prepared{}, packagepipeline.QualifyOptions{})
	if got := closedPackageLifecycleFailure(qualificationErr); got != "qualification_invalid_preparation" {
		t.Fatalf("qualification failure = %q", got)
	}
	_, promotionErr := packagepipeline.Promote(context.Background(), packagepipeline.Qualified{}, packagepipeline.PromotionOptions{})
	if got := closedPackageLifecycleFailure(promotionErr); got != "promotion_invalid_qualification" {
		t.Fatalf("promotion failure = %q", got)
	}
}

func TestClosedReplayFailureCodeRejectsPathBearingValues(t *testing.T) {
	if got := closedReplayFailureCode("invalid_context"); got != "invalid_context" {
		t.Fatalf("closed replay failure = %q", got)
	}
	if got := closedReplayFailureCode("private/result/path"); got != "unclassified" {
		t.Fatalf("path-bearing replay failure = %q", got)
	}
}

func TestClassifyExecutionFailureSummaryReturnsOnlyFixedCategories(t *testing.T) {
	for summary, want := range map[string]string{
		"operation read browser-profile source failed": "browser_profile_contract",
		"open /private/result: no such file":           "artifact_missing",
		"workflow main has depends_on cycle":           "dependency_cycle",
		"uws1: unsupported workflow type custom":       "workflow_type",
		"workflow not found":                           "lookup_failed",
		"unrecognized private failure":                 "unclassified",
	} {
		if got := classifyExecutionFailureSummary(summary); got != want {
			t.Fatalf("execution failure category = %q, want %q", got, want)
		}
	}
	if got := classifyExecutionFailureSummary("workflow alpha gamma"); got != "workflow_keywords_workflow" {
		t.Fatalf("allowlisted execution failure keywords = %q", got)
	}
}

func TestClosedQualityFailureIDsRetainsOnlyFixedCodes(t *testing.T) {
	report := &synthesize.QualityReport{Checks: []synthesize.QualityCheck{
		{Code: "openapi.discovery", Status: "warn"},
		{Code: "browser.authentication.sources", Status: "fail"},
		{Code: "path /private/result", Status: "fail"},
		{Code: "workflow.present", Status: "pass"},
	}}
	if got := closedQualityFailureIDs(report); got != "browser.authentication.sources" {
		t.Fatalf("closed quality failures = %q", got)
	}
	if got := closedQualityFailureIDs(nil); got != "unclassified" {
		t.Fatalf("nil quality failures = %q", got)
	}
}
