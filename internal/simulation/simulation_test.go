package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/mockruntime"
	"github.com/OpenUdon/uws/uws1"
)

func writePackage(t *testing.T, document *uws1.Document) (string, string) {
	t.Helper()
	repo := t.TempDir()
	root := filepath.Join(repo, "example")
	if err := document.Validate(); err != nil {
		t.Fatal(err)
	}
	hcl, err := convert.MarshalHCL(document)
	if err != nil {
		t.Fatal(err)
	}
	yaml, err := convert.MarshalYAML(document)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"project.md": []byte("# Disposable simulation package\n"), "workflows/intent.hcl": []byte("# Fixture intent\n"), "workflows/workflow.hcl": hcl, "workflows/workflow.uws.yaml": yaml, "expected/plan.json": []byte("{}\n"), "expected/quality.json": []byte(`{"status":"fail"}`), "expected/refinement.json": []byte("{}\n"), "expected/review.md": []byte("# Review fixture\n")}
	for name, data := range files {
		write(t, filepath.Join(root, filepath.FromSlash(name)), data)
	}
	paths, err := packageartifacts.RequiredPackagePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	inputs := make([]authoring.ReviewHandoffInput, 0, len(paths))
	for _, path := range paths {
		inputs = append(inputs, authoring.ReviewHandoffInput{Path: path, Purpose: "fixture package input", Required: true, SHA256: strings.Repeat("0", 64)})
	}
	manifest := authoring.NewReviewHandoff(authoring.ReviewHandoffOptions{HandoffInputs: inputs, OwnerSplit: authoring.ReviewOwnerSplit{"openudon": {"artifact validation"}, "external_review_orchestration": {"approval routing"}}})
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, packageartifacts.ReviewHandoffPath), data)
	return repo, root
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func pendingDocument() *uws1.Document {
	return &uws1.Document{UWS: "1.12.0", Info: &uws1.Info{Title: "Pending fixture", Version: "1.0.0"}, Workflows: []*uws1.Workflow{{WorkflowID: "main", Type: "sequence", Steps: []*uws1.Step{{StepID: "report", Pending: &uws1.PendingStep{Purpose: "Read a report", Inputs: &uws1.ParamSchema{Type: "object"}, Outputs: &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{"count": {Type: "integer"}, "text": {Type: "string"}}}, Effect: "read"}}}}}}
}

func TestPendingSimulationIsHypotheticalAndImmutable(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending-only", true: "nested-pending"}[nested], func(t *testing.T) {
			doc := pendingDocument()
			if nested {
				doc.Workflows[0].Steps = []*uws1.Step{{StepID: "group", Type: "sequence", Steps: doc.Workflows[0].Steps}}
			}
			repo, root := writePackage(t, doc)
			before, err := capture(context.Background(), repo, root)
			if err != nil {
				t.Fatal(err)
			}
			report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: root})
			data, _ := json.Marshal(report)
			if report.Status != "completed" || !report.Hypothetical || !report.PackageUnchanged {
				t.Fatalf("pending preview=%s", data)
			}
			index := 0
			if nested {
				index = 1
			}
			step := report.Steps[index]
			if step.StepID != "report" || step.Binding != "pending" || step.Outcome != "simulated" || !reflect.DeepEqual(step.ResponseProvenance, []string{"synthesized"}) {
				t.Fatalf("pending result=%+v", step)
			}
			if len(report.WouldBeRequests) != 1 || report.WouldBeRequests[0].EndpointStatus != "unresolved" || report.WouldBeRequests[0].OperationID != "" || len(report.WouldBeRequests[0].WouldBeRequest) != 0 {
				t.Fatalf("pending endpoint invented: %s", data)
			}
			after, err := capture(context.Background(), repo, root)
			if err != nil {
				t.Fatal(err)
			}
			if before.digest != after.digest || !reflect.DeepEqual(before.files, after.files) {
				t.Fatal("simulation changed package bytes/digest")
			}
			if doc.ValidateExecutable() == nil {
				t.Fatal("original pending contract became executable")
			}
		})
	}
}

func TestSimulationMocksWritesAndUnknownsWithNoNetworkOrExecutor(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests.Add(1) }))
	defer server.Close()
	marker := filepath.Join(t.TempDir(), "called")
	executor := filepath.Join(t.TempDir(), "executor")
	write(t, executor, []byte("#!/bin/sh\ntouch '"+marker+"'\n"))
	os.Chmod(executor, 0700)
	t.Setenv("OPENUDON_EXECUTOR", executor)
	t.Setenv("UDON_CREDENTIAL_SUPPORT_TOKEN", "ENV-PRIVATE-CANARY")
	doc := pendingDocument()
	doc.SourceDescriptions = []*uws1.SourceDescription{{Name: "api", Type: "openapi", URL: server.URL}}
	doc.Operations = []*uws1.Operation{{OperationID: "write", Effect: "write", SourceDescription: "api", OpenAPIOperationID: "post", Request: map[string]any{"header": map[string]any{"Authorization": map[string]any{"$expr": "variables.credentials.support_token"}}, "body": map[string]any{"private": map[string]any{"$expr": "variables.inputs.private"}}}}, {OperationID: "unknown", Effect: "unknown", Extensions: map[string]any{uws1.ExtensionOperationProfile: "test.mock.1"}, Request: map[string]any{"body": map[string]any{"result": map[string]any{"$expr": "write_step.received_body.text"}}}}}
	doc.Workflows[0].Steps = []*uws1.Step{{StepID: "write_step", OperationRef: "write"}, {StepID: "unknown_step", OperationRef: "unknown", StepExecutionFields: uws1.StepExecutionFields{DependsOn: []string{"write_step"}}}, doc.Workflows[0].Steps[0]}
	doc.Workflows[0].Outputs = map[string]string{"report": "report.received_body.text"}
	repo, root := writePackage(t, doc)
	report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: root, Inputs: map[string]any{"private": "INPUT-PRIVATE-CANARY"}, Responses: map[string]ResponseDefinition{"write": {Example: json.RawMessage(`{"statusCode":200,"body":{"text":"RESPONSE-PRIVATE-CANARY"}}`)}, "unknown": {Example: json.RawMessage(`{"body":{}}`)}}})
	encoded, _ := json.Marshal(report)
	if report.Status != "completed" || !report.PackageUnchanged || len(report.WouldBeRequests) != 3 {
		t.Fatalf("preview=%s", encoded)
	}
	if requests.Load() != 0 {
		t.Fatal("simulation contacted bound HTTP service")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("simulation invoked executor")
	}
	if strings.Contains(string(encoded), "CANARY") || strings.Contains(string(encoded), "credential-reference") || strings.Contains(string(encoded), "requestDigest") {
		t.Fatal("private simulation values/digests leaked")
	}
	for index, provenance := range []string{"example", "example", "synthesized"} {
		if !reflect.DeepEqual(report.Steps[index].ResponseProvenance, []string{provenance}) || report.Steps[index].Outcome != "simulated" {
			t.Fatalf("step provenance=%+v", report.Steps[index])
		}
	}
}

func TestFixturesExamplesSynthesisAndRefusalRemainDeterministic(t *testing.T) {
	doc := pendingDocument()
	repo, root := writePackage(t, doc)
	digest, err := mockruntime.RequestDigest(map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	fixtures := &mockruntime.FixtureSet{Format: mockruntime.FixtureFormatV1, Fixtures: []mockruntime.Fixture{{OperationID: "report", RequestDigest: digest, Provenance: mockruntime.FixtureProvenance{Kind: "example"}, Response: json.RawMessage(`{"body":{"count":7,"text":"fixture"}}`)}}}
	options := Options{RepoRoot: repo, ExampleDir: root, Fixtures: fixtures, Responses: map[string]ResponseDefinition{"report": {Example: json.RawMessage(`{"body":{"count":1,"text":"example"}}`)}}}
	first := Run(context.Background(), options)
	second := Run(context.Background(), options)
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if first.Status != "completed" || string(a) != string(b) || !reflect.DeepEqual(first.Steps[0].ResponseProvenance, []string{"fixture"}) {
		t.Fatalf("fixture nondeterminism: %s %s", a, b)
	}
	if fixtures.Fixtures[0].OperationID != "report" {
		t.Fatal("fixture set mutated")
	}
	miss := *fixtures
	miss.Fixtures = append([]mockruntime.Fixture(nil), fixtures.Fixtures...)
	miss.Fixtures[0].RequestDigest = "sha256:" + strings.Repeat("a", 64)
	options.Fixtures = &miss
	refused := Run(context.Background(), options)
	if refused.Status != "blocked" || !refused.PackageUnchanged {
		t.Fatalf("fixture miss silently synthesized: %+v", refused)
	}
	options.AllowGeneratedFallback = true
	generated := Run(context.Background(), options)
	if generated.Status != "completed" || !reflect.DeepEqual(generated.Steps[0].ResponseProvenance, []string{"example"}) {
		t.Fatalf("explicit example fallback=%+v", generated)
	}
}

func TestSimulationRefusesUnsafeInconsistentAndUnsupportedPackages(t *testing.T) {
	for _, kind := range []string{"mismatch", "symlink", "invalid-schema", "unsupported-schema", "missing-response", "cancelled", "outside-root"} {
		t.Run(kind, func(t *testing.T) {
			doc := pendingDocument()
			if kind == "unsupported-schema" {
				doc.Workflows[0].Steps[0].Pending.Outputs = &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{"timestamp": {Type: "string", Format: "date-time"}}}
			}
			if kind == "missing-response" {
				doc.Operations = []*uws1.Operation{{OperationID: "read", Extensions: map[string]any{uws1.ExtensionOperationProfile: "test.mock.1"}}}
				doc.Workflows[0].Steps = []*uws1.Step{{StepID: "read_step", OperationRef: "read"}}
			}
			repo, root := writePackage(t, doc)
			ctx := context.Background()
			switch kind {
			case "mismatch":
				other := pendingDocument()
				other.Workflows[0].Steps[0].Pending.Purpose = "PRIVATE-CANARY"
				hcl, err := convert.MarshalHCL(other)
				if err != nil {
					t.Fatal(err)
				}
				write(t, filepath.Join(root, "workflows/workflow.hcl"), hcl)
			case "symlink":
				p := filepath.Join(root, "workflows/workflow.uws.yaml")
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				outside := filepath.Join(t.TempDir(), "private.yaml")
				write(t, outside, data)
				os.Remove(p)
				if err := os.Symlink(outside, p); err != nil {
					t.Fatal(err)
				}
			case "invalid-schema":
				p := filepath.Join(root, "workflows/workflow.uws.yaml")
				data, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				write(t, p, []byte(strings.Replace(string(data), "integer", "PRIVATE-CANARY", 1)))
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "outside-root":
				repo = t.TempDir()
			}
			report := Run(ctx, Options{RepoRoot: repo, ExampleDir: root})
			encoded, _ := json.Marshal(report)
			if report.Status != "blocked" || strings.Contains(string(encoded), "PRIVATE-CANARY") {
				t.Fatalf("unsafe preview accepted or exposed data: %s", encoded)
			}
			if (kind == "unsupported-schema" || kind == "missing-response") && !report.PackageUnchanged {
				t.Fatal("failed runtime preview changed package")
			}
		})
	}
}

func TestPublicOrchestratorOwnsParallelBranchesAndLoop(t *testing.T) {
	for _, kind := range []string{"parallel", "switch", "loop"} {
		t.Run(kind, func(t *testing.T) {
			doc := pendingDocument()
			first := doc.Workflows[0].Steps[0]
			second := *first
			second.StepID = "unselected"
			expected := 1
			switch kind {
			case "parallel":
				doc.Workflows[0].Type = "parallel"
				doc.Workflows[0].Steps = append(doc.Workflows[0].Steps, &second)
				expected = 2
			case "switch":
				doc.Variables = map[string]any{"take": true, "skip": false}
				doc.Workflows[0].Type = "switch"
				doc.Workflows[0].Steps = nil
				doc.Workflows[0].Cases = []*uws1.Case{{CaseFields: uws1.CaseFields{Name: "take", When: "$variables.take"}, Steps: []*uws1.Step{first}}, {CaseFields: uws1.CaseFields{Name: "skip", When: "$variables.skip"}, Steps: []*uws1.Step{&second}}}
			case "loop":
				doc.Workflows[0].Type = "loop"
				doc.Workflows[0].Items = "$variables.items"
				doc.Variables = map[string]any{"items": []any{1, 2, 3}}
				expected = 3
			}
			repo, root := writePackage(t, doc)
			report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: root})
			encoded, _ := json.Marshal(report)
			if report.Status != "completed" || !report.PackageUnchanged || len(report.WouldBeRequests) != expected {
				t.Fatalf("%s preview=%s", kind, encoded)
			}
			if kind == "switch" && report.Steps[1].Outcome != "not_started" {
				t.Fatal("unselected branch fabricated execution")
			}
		})
	}
}

func TestSimulationRequestBoundsAndFieldNameRedaction(t *testing.T) {
	doc := pendingDocument()
	doc.Workflows[0].Type = "loop"
	doc.Workflows[0].Items = "$variables.items"
	items := make([]any, MaxRequests+1)
	for i := range items {
		items[i] = i
	}
	doc.Variables = map[string]any{"items": items}
	repo, root := writePackage(t, doc)
	report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: root})
	if report.Status != "blocked" || !report.PackageUnchanged || len(report.WouldBeRequests) != MaxRequests {
		t.Fatalf("unbounded preview: %+v", report)
	}
	preview, _ := redactedJSON(map[string]any{"PRIVATE-FIELD-CANARY": map[string]any{"PRIVATE-VALUE-CANARY": "PRIVATE-SECRET-CANARY"}}, 4096)
	if strings.Contains(string(preview), "CANARY") {
		t.Fatalf("private object name exposed: %s", preview)
	}
}

func TestProjectionPreservesLiteralVariablesAndUsesCredentialSymbols(t *testing.T) {
	doc := pendingDocument()
	doc.Variables = map[string]any{"literal": map[string]any{"$expr": "PRIVATE-LITERAL-CANARY"}, "credentials": map[string]any{"token": "PRIVATE-CREDENTIAL-CANARY"}}
	before, _ := json.Marshal(doc.Variables["literal"])
	projection, err := project(doc, Options{})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(projection.document.Variables["literal"])
	if string(before) != string(after) {
		t.Fatal("literal variable became expression authority")
	}
	if credentials := projection.document.Variables["credentials"].(map[string]any); len(credentials) != 0 {
		t.Fatal("inline credentials retained")
	}
}

func TestBrowserSimulationDoesNotClaimPageVerification(t *testing.T) {
	doc := pendingDocument()
	doc.Operations = []*uws1.Operation{{OperationID: "browser", Effect: "unknown", Extensions: map[string]any{uws1.ExtensionOperationProfile: "uws.browser.1.10"}}}
	doc.Workflows[0].Steps = []*uws1.Step{{StepID: "browser_step", OperationRef: "browser"}}
	repo, root := writePackage(t, doc)
	report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: root, Responses: map[string]ResponseDefinition{"browser": {Example: json.RawMessage(`{"body":{}}`)}}})
	if report.Status != "completed" || report.Steps[0].BrowserEvidence != "mocked contract only; page state not verified" {
		t.Fatalf("browser proof overstated: %+v", report)
	}
}

func TestChangedPackageInvalidatesPreview(t *testing.T) {
	repo, root := writePackage(t, pendingDocument())
	before, err := capture(context.Background(), repo, root)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "project.md"), []byte("# Changed during preview\n"))
	report := finish(context.Background(), Options{RepoRoot: repo, ExampleDir: root}, before, Report{Version: Version, Tier: "simulation", Status: "completed", PackageSHA256: before.digest})
	if report.Status != "blocked" || report.PackageUnchanged || report.Diagnostics[0].Code != "package.changed" {
		t.Fatalf("stale preview remained valid: %+v", report)
	}
}

func TestRequestPreviewTruncatesWithoutPrivateValues(t *testing.T) {
	doc := pendingDocument()
	request := map[string]any{}
	for i := 0; i < 600; i++ {
		request[fmt.Sprintf("PRIVATE-FIELD-CANARY-%d", i)] = "PRIVATE-VALUE-CANARY"
	}
	doc.Operations = []*uws1.Operation{{OperationID: "large", Effect: "write", Extensions: map[string]any{uws1.ExtensionOperationProfile: "test.mock.1"}, Request: map[string]any{"body": request}}}
	doc.Workflows[0].Steps = []*uws1.Step{{StepID: "write", OperationRef: "large"}}
	repo, root := writePackage(t, doc)
	report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: root, Responses: map[string]ResponseDefinition{"large": {Example: json.RawMessage(`{"body":{}}`)}}})
	data, _ := json.Marshal(report)
	if report.Status != "completed" || !report.WouldBeRequests[0].Truncated || strings.Contains(string(data), "CANARY") {
		t.Fatalf("oversized preview exposed or unbounded: %s", data)
	}
}
