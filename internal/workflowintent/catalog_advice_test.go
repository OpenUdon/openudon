package workflowintent

import (
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
)

func TestCatalogAdviceForIntentMatchesProvider(t *testing.T) {
	report, err := CatalogAdviceForIntent(&Intent{
		Steps: []*Step{{
			Name:     "create_issue",
			Type:     "http",
			Provider: "github",
		}},
	}, CatalogAdviceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Enabled || len(report.Providers) != 1 {
		t.Fatalf("report = %#v, want one enabled provider advice", report)
	}
	provider := report.Providers[0]
	if provider.ProviderID != "github" || provider.MatchSource != "step.create_issue.provider" {
		t.Fatalf("provider advice = %#v", provider)
	}
	if provider.SpecRef == nil || provider.SpecRef.ID != "github-rest-api-openapi" {
		t.Fatalf("spec ref = %#v, want github-rest-api-openapi", provider.SpecRef)
	}
	if provider.AuthStatus != catalog.AuthStatusOverlayRequired {
		t.Fatalf("auth status = %q, want %q", provider.AuthStatus, catalog.AuthStatusOverlayRequired)
	}
	if !containsString(provider.OverlayIDs, "github-rest-api-auth-overlay") {
		t.Fatalf("overlay ids = %#v, want github-rest-api-auth-overlay", provider.OverlayIDs)
	}
}

func TestCatalogAdviceForIntentRecordsExplicitOpenAPIPrecedence(t *testing.T) {
	report, err := CatalogAdviceForIntent(&Intent{
		OpenAPI: "openapi/github.yaml",
		Steps: []*Step{{
			Name:     "list_repos",
			Type:     "http",
			Provider: "github",
		}},
	}, CatalogAdviceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 1 {
		t.Fatalf("providers = %#v, want github", report.Providers)
	}
	provider := report.Providers[0]
	if !provider.ExplicitOpenAPIOverride {
		t.Fatalf("expected explicit OpenAPI override in %#v", provider)
	}
	if !containsString(provider.ExplicitOpenAPIInputs, "openapi/github.yaml") {
		t.Fatalf("explicit inputs = %#v, want openapi/github.yaml", provider.ExplicitOpenAPIInputs)
	}
	markdown := RenderCatalogAdviceMarkdown(report)
	for _, expected := range []string{
		"Catalog metadata is advisory",
		"Explicit OpenAPI input overrides built-in catalog spec",
		"github-rest-api-openapi",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("markdown missing %q:\n%s", expected, markdown)
		}
	}
}

func TestCatalogAdviceForIntentNoMatchRendersEmptyMarkdown(t *testing.T) {
	report, err := CatalogAdviceForIntent(&Intent{
		Steps: []*Step{{
			Name:     "lookup",
			Type:     "http",
			Provider: "private-crm",
		}},
	}, CatalogAdviceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 0 {
		t.Fatalf("providers = %#v, want none", report.Providers)
	}
	if got := RenderCatalogAdviceMarkdown(report); got != "" {
		t.Fatalf("markdown = %q, want empty", got)
	}
}

func TestCatalogAdviceForIntentDisabled(t *testing.T) {
	report, err := CatalogAdviceForIntent(&Intent{
		Steps: []*Step{{Name: "send", Type: "http", Provider: "slack"}},
	}, CatalogAdviceOptions{Disabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Enabled || len(report.Providers) != 0 {
		t.Fatalf("report = %#v, want disabled empty report", report)
	}
	if got := RenderCatalogAdviceMarkdown(report); got != "" {
		t.Fatalf("markdown = %q, want empty", got)
	}
}

func TestCatalogAdviceForIntentMatchesOpenAPIFileName(t *testing.T) {
	report, err := CatalogAdviceForIntent(&Intent{
		OpenAPI: "openapi/slack.yaml",
		Steps: []*Step{{
			Name:      "post_message",
			Type:      "http",
			Operation: "chat_postMessage",
		}},
	}, CatalogAdviceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 1 || report.Providers[0].ProviderID != "slack" {
		t.Fatalf("providers = %#v, want slack", report.Providers)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
