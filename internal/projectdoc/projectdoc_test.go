package projectdoc

import (
	"strings"
	"testing"
)

func TestSectionNormalizesAndExtractsMarkdownHeadings(t *testing.T) {
	doc := "# Example\n\n## Credentials & Secrets\n\n- Use `support_api_token`.\n\n## Runtime Policy\n\n- `cmd` is not allowed unless explicitly approved here.\n"
	if got := Title(doc); got != "Example" {
		t.Fatalf("Title() = %q, want Example", got)
	}
	if got := Section(doc, "Credentials and Secrets"); got != "- Use `support_api_token`." {
		t.Fatalf("Section() = %q", got)
	}
}

func TestRuntimeExplicitlyAllowedRejectsDefaultDisallow(t *testing.T) {
	section := "- `cmd` is not allowed unless explicitly approved here.\n- `ssh` is explicitly approved for this project.\n"
	if RuntimeExplicitlyAllowed(section, "cmd") {
		t.Fatalf("cmd should not be allowed by default disallow text")
	}
	if !RuntimeExplicitlyAllowed(section, "ssh") {
		t.Fatalf("ssh should be allowed by explicit approval text")
	}
}

func TestNoOpenAPIRequiredRecognizesCanonicalPhrase(t *testing.T) {
	doc := "# Runtime Only\n\n## External Systems and OpenAPI\n\nOpenAPI: none required\n"
	if !NoOpenAPIRequired(doc) {
		t.Fatalf("NoOpenAPIRequired did not recognize canonical phrase")
	}
}

func TestCandidateWorkflowsRenderAndParseDeterministically(t *testing.T) {
	candidates := []CandidateWorkflow{
		{Title: "Send Alert", Outcome: "Notify the team", DeferralReason: "not in the active boundary", PromotionTrigger: "the reporting workflow is approved"},
		{Title: "Archive Report", Outcome: "Store the report", DeferralReason: "storage is undecided", PromotionTrigger: "an archive provider is selected"},
	}
	rendered := RenderCandidateWorkflows(candidates)
	if !strings.Contains(rendered, "openudon:candidate-workflows:v1; non-executable") || strings.Index(rendered, "Archive Report") > strings.Index(rendered, "Send Alert") {
		t.Fatalf("rendered candidates are not marked and sorted:\n%s", rendered)
	}
	parsed, err := ParseCandidateWorkflows("# Example\n\n" + rendered + "## Inputs\n\n- none\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 || parsed[0].Title != "Archive Report" || parsed[1].PromotionTrigger != "the reporting workflow is approved" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestCandidateWorkflowsRejectMalformedSection(t *testing.T) {
	doc := "# Example\n\n## Candidate Workflows\n\n<!-- openudon:candidate-workflows:v1; non-executable -->\n\n- **Send Alert**\n  - Outcome: Notify the team\n"
	if _, err := ParseCandidateWorkflows(doc); err == nil || !strings.Contains(err.Error(), "promotion trigger") {
		t.Fatalf("ParseCandidateWorkflows error = %v", err)
	}
}
