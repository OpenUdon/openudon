package projectdoc

import "testing"

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
