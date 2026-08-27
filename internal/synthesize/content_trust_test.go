package synthesize

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	uwstrust "github.com/OpenUdon/uws/contenttrust"
	uwsconvert "github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

func TestAnalyzePackageContentTrustUsesBrowserResolver(t *testing.T) {
	example := t.TempDir()
	relative := "browser-profiles/status.json"
	writeWorkflowBrowserFixture(t, example, relative, synthesizeBrowserProfileFixture(false, false, "query"))
	doc := browserContentTrustDocument(relative)
	report, err := analyzePackageContentTrust(context.Background(), example, doc)
	if err != nil {
		t.Fatal(err)
	}
	if !hasContentTrustFinding(report, uwstrust.CodeUntrustedControl) {
		t.Fatalf("browser output did not retain untrusted provenance: %#v", report)
	}
	second, err := analyzePackageContentTrust(context.Background(), example, doc)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, second) {
		t.Fatalf("reports are not deterministic:\n%#v\n%#v", report, second)
	}
}

func TestAssessContentTrustAddsWarningChecksWithoutFailingQuality(t *testing.T) {
	example := t.TempDir()
	relative := "browser-profiles/status.json"
	writeWorkflowBrowserFixture(t, example, relative, synthesizeBrowserProfileFixture(false, false, "query"))
	doc := browserContentTrustDocument(relative)
	data, err := uwsconvert.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{ExampleDir: example, UWSPath: filepath.Join(example, "workflows", "workflow.uws.yaml")}
	if err := os.MkdirAll(filepath.Dir(result.UWSPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.UWSPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	quality := &QualityReport{Status: "pass"}
	if err := assessContentTrust(context.Background(), quality, result); err != nil {
		t.Fatal(err)
	}
	quality.finalize()
	check := qualityContentTrustCheck(quality, uwstrust.CodeUntrustedControl)
	if quality.Status != "pass" || check == nil || check.Status != "warn" {
		t.Fatalf("quality checks/status = %#v / %s", quality.Checks, quality.Status)
	}
	if !strings.Contains(check.Detail, "severity=warning; path=") {
		t.Fatalf("quality detail = %q", check.Detail)
	}
	var review strings.Builder
	writeContentTrustReview(&review, reviewBuildState{contentTrustDeclared: true, contentTrust: &uwstrust.Report{Findings: []uwstrust.Finding{{
		Code: uwstrust.CodeUntrustedControl, Severity: uwstrust.SeverityWarning, Path: "workflows[0].steps[1].when", Message: "untrusted content can influence control flow",
	}}}})
	text := review.String()
	for _, want := range []string{"## Content-Trust Analysis", uwstrust.CodeUntrustedControl, "workflows[0].steps[1].when", "do not change quality status"} {
		if !strings.Contains(text, want) {
			t.Fatalf("review omitted %q:\n%s", want, text)
		}
	}
}

func qualityContentTrustCheck(report *QualityReport, code string) *QualityCheck {
	if report == nil {
		return nil
	}
	for i := range report.Checks {
		if report.Checks[i].Code == code {
			return &report.Checks[i]
		}
	}
	return nil
}

func TestAssessContentTrustLeavesLegacyQualityUnchanged(t *testing.T) {
	example := t.TempDir()
	doc := browserContentTrustDocument("browser-profiles/status.json")
	doc.UWS = "1.9.0"
	doc.ContentTrust = nil
	data, err := uwsconvert.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{ExampleDir: example, UWSPath: filepath.Join(example, "workflow.uws.yaml")}
	if err := os.WriteFile(result.UWSPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	quality := &QualityReport{}
	if err := assessContentTrust(context.Background(), quality, result); err != nil {
		t.Fatal(err)
	}
	if len(quality.Checks) != 0 {
		t.Fatalf("legacy checks = %#v", quality.Checks)
	}
}

func TestAnalyzePackageContentTrustTurnsBrowserLoadFailureIntoFinding(t *testing.T) {
	example := t.TempDir()
	relative := "browser-profiles/status.json"
	path := filepath.Join(example, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"profile":"uws.browser.1.7"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := analyzePackageContentTrust(context.Background(), example, browserContentTrustDocument(relative))
	if err != nil {
		t.Fatal(err)
	}
	if !hasContentTrustFinding(report, uwstrust.CodeResolverFailure) {
		t.Fatalf("missing resolver failure: %#v", report)
	}
	for _, finding := range report.Findings {
		if finding.Code == uwstrust.CodeResolverFailure && finding.Message != "operation resolver failed while describing the operation" {
			t.Fatalf("resolver failure leaked implementation detail: %#v", finding)
		}
	}
}

func TestAnalyzePackageContentTrustLegacyAndCancellation(t *testing.T) {
	legacy := browserContentTrustDocument("browser-profiles/status.json")
	legacy.UWS = "1.9.0"
	legacy.ContentTrust = nil
	report, err := analyzePackageContentTrust(context.Background(), t.TempDir(), legacy)
	if err != nil || len(report.Edges) != 0 || len(report.Findings) != 0 {
		t.Fatalf("legacy analysis = %#v, %v", report, err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if report, err := analyzePackageContentTrust(canceled, t.TempDir(), legacy); report != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled analysis = %#v, %v", report, err)
	}
}

func browserContentTrustDocument(relative string) *uws1.Document {
	return &uws1.Document{
		UWS:  "1.9.1",
		Info: &uws1.Info{Title: "Browser trust", Version: "1.0.0"},
		SourceDescriptions: []*uws1.SourceDescription{{
			Name: "browser", URL: relative, Type: uws1.SourceDescriptionTypeBrowserProfile,
		}},
		Operations: []*uws1.Operation{{
			OperationID: "read", SourceDescription: "browser", SourceOperationID: "read_status",
			Outputs: map[string]string{"status": "$response.body.status"},
		}, {
			OperationID: "noop", Extensions: map[string]any{uws1.ExtensionOperationProfile: "test.noop.1"},
		}},
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main", Type: uws1.WorkflowTypeSequence,
			Steps: []*uws1.Step{
				{StepID: "read_step", OperationRef: "read", Outputs: map[string]string{"status": "$outputs.status"}},
				{StepID: "gate", OperationRef: "noop", StepExecutionFields: uws1.StepExecutionFields{When: "$steps.read_step.outputs.status"}},
			},
		}},
		ContentTrust: &uws1.ContentTrust{Workflows: map[string]*uws1.WorkflowContentTrust{
			"main": {Default: uws1.ContentTrustUnknown},
		}},
	}
}

func hasContentTrustFinding(report *uwstrust.Report, code string) bool {
	if report == nil {
		return false
	}
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
