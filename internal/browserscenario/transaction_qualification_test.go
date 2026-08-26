package browserscenario

import (
	"strings"
	"testing"

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

func TestClosedQualityFailureIDsRetainsOnlyFixedCodes(t *testing.T) {
	report := &synthesize.QualityReport{Checks: []synthesize.QualityCheck{
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
