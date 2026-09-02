package synthesize

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	uwstrust "github.com/OpenUdon/uws/contenttrust"
	"github.com/OpenUdon/uws/uws1"
	"golang.org/x/mod/modfile"
)

const (
	qualificationPrivateContent = "E12_RUNTIME_PRIVATE_CONTENT"
	qualificationResolverDetail = "E12_RESOLVER_PRIVATE_DETAIL"
)

type qualificationResolverFunc func(context.Context, *uws1.Document, *uws1.Operation) (bool, uwstrust.OperationContract, error)

func (f qualificationResolverFunc) ResolveOperation(ctx context.Context, doc *uws1.Document, operation *uws1.Operation) (bool, uwstrust.OperationContract, error) {
	return f(ctx, doc, operation)
}

func TestContentTrustQualificationMatrix(t *testing.T) {
	type fixture struct {
		name       string
		document   func() *uws1.Document
		resolvers  func() []uwstrust.Resolver
		want       []string
		forbid     []string
		assertions func(*testing.T, *uwstrust.Report)
	}
	fixtures := []fixture{
		{
			name:     "mail_to_llm_data_to_authority",
			document: qualificationPipelineDocument,
			resolvers: func() []uwstrust.Resolver {
				return []uwstrust.Resolver{qualificationPipelineResolver(uwstrust.ChannelData, uwstrust.CapabilityFreeText, uwstrust.CapabilityFreeText)}
			},
			want:   []string{uwstrust.CodeUntrustedAuthority},
			forbid: []string{uwstrust.CodeUntrustedInstruction},
			assertions: func(t *testing.T, report *uwstrust.Report) {
				assertQualificationEdge(t, report, "workflows[0].steps[0].outputs.body", "workflows[0].steps[1].inputs.prompt", uws1.ContentTrustUntrusted, uwstrust.CapabilityFreeText)
				assertQualificationEdge(t, report, "workflows[0].steps[1].outputs.summary", "workflows[0].steps[2].inputs.target", uws1.ContentTrustUntrusted, uwstrust.CapabilityFreeText)
			},
		},
		{
			name:     "untrusted_llm_instruction",
			document: qualificationPipelineDocument,
			resolvers: func() []uwstrust.Resolver {
				return []uwstrust.Resolver{qualificationPipelineResolver(uwstrust.ChannelInstruction, uwstrust.CapabilityFreeText, uwstrust.CapabilityFreeText)}
			},
			want: []string{uwstrust.CodeUntrustedInstruction, uwstrust.CodeUntrustedAuthority},
		},
		{
			name: "constrained_scalar_control_retains_provenance",
			document: func() *uws1.Document {
				doc := qualificationPipelineDocument()
				doc.Workflows[0].Steps[2].When = "$steps.model_step.outputs.summary"
				return doc
			},
			resolvers: func() []uwstrust.Resolver {
				return []uwstrust.Resolver{qualificationPipelineResolver(uwstrust.ChannelInstruction, uwstrust.CapabilityConstrainedScalar, uwstrust.CapabilityConstrainedScalar)}
			},
			want:   []string{uwstrust.CodeUntrustedAuthority, uwstrust.CodeUntrustedControl},
			forbid: []string{uwstrust.CodeUntrustedInstruction},
			assertions: func(t *testing.T, report *uwstrust.Report) {
				assertQualificationEdge(t, report, "workflows[0].steps[1].outputs.summary", "workflows[0].steps[2].inputs.target", uws1.ContentTrustUntrusted, uwstrust.CapabilityConstrainedScalar)
			},
		},
		{
			name: "trigger_payload_defaults_untrusted",
			document: func() *uws1.Document {
				doc := qualificationPipelineDocument()
				doc.Workflows[0].Steps[1].Inputs["prompt"] = "$trigger.body"
				return doc
			},
			resolvers: func() []uwstrust.Resolver {
				return []uwstrust.Resolver{qualificationPipelineResolver(uwstrust.ChannelInstruction, uwstrust.CapabilityFreeText, uwstrust.CapabilityFreeText)}
			},
			want: []string{uwstrust.CodeUntrustedInstruction, uwstrust.CodeUntrustedAuthority},
			assertions: func(t *testing.T, report *uwstrust.Report) {
				assertQualificationEdge(t, report, "$trigger", "workflows[0].steps[1].inputs.prompt", uws1.ContentTrustUntrusted, uwstrust.CapabilityUnknown)
			},
		},
		{
			name: "unknown_entry_and_opaque_extension",
			document: func() *uws1.Document {
				doc := qualificationPipelineDocument()
				doc.Workflows[0].Inputs = &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{"external": {Type: "string"}}}
				doc.Workflows[0].When = `$inputs.external == "enabled"`
				doc.Operations[1].Request["body"] = map[string]any{"prompt": "prefix ${trigger.body}"}
				return doc
			},
			resolvers: func() []uwstrust.Resolver {
				return []uwstrust.Resolver{qualificationPipelineResolver(uwstrust.ChannelInstruction, uwstrust.CapabilityFreeText, uwstrust.CapabilityFreeText)}
			},
			want: []string{uwstrust.CodeUnknownProvenance, uwstrust.CodeOpaqueExpression},
		},
		{
			name:     "resolver_failure",
			document: qualificationPipelineDocument,
			resolvers: func() []uwstrust.Resolver {
				return []uwstrust.Resolver{qualificationResolverFunc(func(context.Context, *uws1.Document, *uws1.Operation) (bool, uwstrust.OperationContract, error) {
					return false, uwstrust.OperationContract{}, errors.New(qualificationResolverDetail)
				})}
			},
			want: []string{uwstrust.CodeResolverFailure},
		},
		{
			name:     "resolver_conflict",
			document: qualificationPipelineDocument,
			resolvers: func() []uwstrust.Resolver {
				claim := qualificationResolverFunc(func(context.Context, *uws1.Document, *uws1.Operation) (bool, uwstrust.OperationContract, error) {
					return true, uwstrust.OperationContract{}, nil
				})
				return []uwstrust.Resolver{claim, claim}
			},
			want: []string{uwstrust.CodeResolverConflict},
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			doc := fixture.document()
			resolvers := fixture.resolvers()
			report, err := uwstrust.Analyze(context.Background(), doc, resolvers...)
			if err != nil {
				t.Fatal(err)
			}
			second, err := uwstrust.Analyze(context.Background(), doc, resolvers...)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(report, second) {
				t.Fatalf("qualification report is not deterministic:\n%#v\n%#v", report, second)
			}
			for _, code := range fixture.want {
				if !hasContentTrustFinding(report, code) {
					t.Fatalf("missing %s: %#v", code, report.Findings)
				}
			}
			for _, code := range fixture.forbid {
				if hasContentTrustFinding(report, code) {
					t.Fatalf("unexpected %s: %#v", code, report.Findings)
				}
			}
			if fixture.assertions != nil {
				fixture.assertions(t, report)
			}
			assertQualificationEvidence(t, report)
		})
	}
}

func TestContentTrustQualificationLegacyPackageIsUnchanged(t *testing.T) {
	doc := browserContentTrustDocument("browser-profiles/status.json")
	doc.UWS = "1.9.0"
	doc.ContentTrust = nil
	report, err := analyzePackageContentTrust(context.Background(), t.TempDir(), doc)
	if err != nil {
		t.Fatal(err)
	}
	quality := &QualityReport{}
	if len(report.Edges) != 0 || len(report.Findings) != 0 || len(quality.Checks) != 0 {
		t.Fatalf("legacy report or quality changed: %#v / %#v", report, quality.Checks)
	}
	var review strings.Builder
	writeContentTrustReview(&review, reviewBuildState{})
	if review.Len() != 0 {
		t.Fatalf("legacy review changed: %q", review.String())
	}
}

func TestContentTrustQualificationUsesPublishedUWSAndBrowsertools(t *testing.T) {
	const (
		wantUWS          = "v0.0.0-20260826233246-9e676eaa469e"
		wantBrowsertools = "v0.0.0-20260902183222-ce06b13bfef8"
	)
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	module, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		t.Fatal(err)
	}
	versions := make(map[string]string, len(module.Require))
	for _, dependency := range module.Require {
		versions[dependency.Mod.Path] = dependency.Mod.Version
	}
	if versions["github.com/OpenUdon/uws"] != wantUWS {
		t.Fatalf("UWS version = %q, want %q", versions["github.com/OpenUdon/uws"], wantUWS)
	}
	if versions["github.com/OpenUdon/browsertools"] != wantBrowsertools {
		t.Fatalf("Browsertools version = %q, want %q", versions["github.com/OpenUdon/browsertools"], wantBrowsertools)
	}
}

func qualificationPipelineDocument() *uws1.Document {
	profile := func() map[string]any { return map[string]any{uws1.ExtensionOperationProfile: "e12.test.1"} }
	return &uws1.Document{
		UWS:       "1.9.1",
		Info:      &uws1.Info{Title: "E12 content-trust qualification", Version: "1.0.0"},
		Variables: map[string]any{"private_runtime_content": qualificationPrivateContent},
		Operations: []*uws1.Operation{
			{OperationID: "read_message", Outputs: map[string]string{"body": "$response.body"}, Extensions: profile()},
			{OperationID: "summarize", Request: map[string]any{"body": map[string]any{"prompt": "$inputs.prompt"}}, Outputs: map[string]string{"summary": "$response.body"}, Extensions: profile()},
			{OperationID: "send_message", Request: map[string]any{"body": map[string]any{"target": "$inputs.target"}}, Extensions: profile()},
		},
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main", Type: uws1.WorkflowTypeSequence,
			Steps: []*uws1.Step{
				{StepID: "read_step", OperationRef: "read_message", Outputs: map[string]string{"body": "$outputs.body"}},
				{StepID: "model_step", OperationRef: "summarize", Inputs: map[string]any{"prompt": "$steps.read_step.outputs.body"}, Outputs: map[string]string{"summary": "$outputs.summary"}},
				{StepID: "send_step", OperationRef: "send_message", Inputs: map[string]any{"target": "$steps.model_step.outputs.summary"}},
			},
		}},
		ContentTrust: &uws1.ContentTrust{Operations: map[string]*uws1.OperationContentTrust{
			"read_message": {Outputs: map[string]uws1.ContentTrustLevel{"body": uws1.ContentTrustUntrusted}},
		}},
	}
}

func qualificationPipelineResolver(modelKind uwstrust.ChannelKind, readCapability, modelOutputCapability uwstrust.ValueCapability) uwstrust.Resolver {
	return qualificationResolverFunc(func(_ context.Context, _ *uws1.Document, operation *uws1.Operation) (bool, uwstrust.OperationContract, error) {
		switch operation.OperationID {
		case "read_message":
			return true, uwstrust.OperationContract{Outputs: map[string]uwstrust.ValueContract{"body": {Capability: readCapability}}}, nil
		case "summarize":
			return true, uwstrust.OperationContract{
				Inputs:                  []uwstrust.InputChannel{{Path: "/request/body/prompt", Kind: modelKind}},
				Outputs:                 map[string]uwstrust.ValueContract{"summary": {Capability: modelOutputCapability}},
				InheritsInputProvenance: true,
			}, nil
		case "send_message":
			return true, uwstrust.OperationContract{Inputs: []uwstrust.InputChannel{{Path: "/request/body/target", Kind: uwstrust.ChannelAuthority}}}, nil
		default:
			return false, uwstrust.OperationContract{}, nil
		}
	})
}

func assertQualificationEvidence(t *testing.T, report *uwstrust.Report) {
	t.Helper()
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	quality := &QualityReport{}
	addContentTrustReportChecks(quality, report)
	quality.finalize()
	if quality.Status != "pass" {
		t.Fatalf("advisory findings failed quality: %#v", quality.Checks)
	}
	for _, check := range quality.Checks {
		if check.Status != "warn" {
			t.Fatalf("finding check status = %q: %#v", check.Status, check)
		}
	}
	var review strings.Builder
	writeContentTrustReview(&review, reviewBuildState{contentTrustDeclared: true, contentTrust: report})
	reviewText := review.String()
	for _, finding := range report.Findings {
		for _, field := range []string{finding.Code, string(finding.Severity), finding.Path, finding.Message} {
			if field == "" || !strings.Contains(reviewText, field) {
				t.Fatalf("review omitted stable finding field %q:\n%s", field, reviewText)
			}
		}
	}
	for _, private := range []string{qualificationPrivateContent, qualificationResolverDetail} {
		if strings.Contains(string(encoded), private) || strings.Contains(reviewText, private) {
			t.Fatalf("qualification evidence leaked private content %q", private)
		}
	}
}

func assertQualificationEdge(t *testing.T, report *uwstrust.Report, from, to string, provenance uws1.ContentTrustLevel, capability uwstrust.ValueCapability) {
	t.Helper()
	for _, edge := range report.Edges {
		if edge.From == from && edge.To == to && edge.Provenance == provenance && edge.Capability == capability {
			return
		}
	}
	t.Fatalf("missing edge %s -> %s (%s, %s): %#v", from, to, provenance, capability, report.Edges)
}
