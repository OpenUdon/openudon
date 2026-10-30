package packagev3_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

const compositionAPI = `openapi: 3.1.0
info: {title: Synthetic, version: '1'}
servers: [{url: 'https://fixture.example.test'}]
security: []
paths:
  /read:
    get:
      operationId: read
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {type: object, properties: {heading: {const: Hello}}, required: [heading], additionalProperties: false}
  /consume:
    get:
      operationId: consume
      parameters: [{name: message, in: query, required: true, schema: {type: string}}]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {type: string}
`
const compositionWorkflow = `uws: 1.13.0
info: {title: Synthetic, version: '1'}
sourceDescriptions: [{name: api, type: openapi, url: sources/openapi/api.yaml}]
operations:
 - operationId: read
   sourceDescription: api
   sourceOperationId: read
   effect: read
 - operationId: format
   effect: read
   x-uws-operation-profile: uws.runtime.1.0
   x-uws-runtime: {type: fnct, function: message.format}
   request: {body: {heading: '$steps.read.outputs.heading', body: '$variables.body'}}
   outputs: {formatted: '$response.body'}
 - operationId: consume
   sourceDescription: api
   sourceOperationId: consume
   effect: read
   request: {query: {message: '$steps.format.outputs.formatted'}}
workflows:
 - workflowId: main
   type: sequence
   steps:
    - {stepId: read, operationRef: read, outputs: {heading: '$response.body.heading'}}
    - {stepId: format, operationRef: format, outputs: {formatted: '$response.body'}}
    - {stepId: consume, operationRef: consume}
   outputs: {result: '$steps.format.outputs.formatted'}
`

func compositionOptions(t *testing.T) packagev3.BuildOptions {
	o := bodyOptions(t)
	o.WorkflowYAML = []byte(compositionWorkflow)
	o.DataJSON = []byte(`{"body":"World"}`)
	o.Sources = append(o.Sources, packagev3.SourceInput{ID: "api", Kind: "openapi", Path: "sources/openapi/api.yaml", Bytes: []byte(compositionAPI)})
	return o
}

func checkAssessmentParity(t *testing.T, o packagev3.BuildOptions, want string) {
	t.Helper()
	ctx := context.Background()
	p, err := packagev3.Build(ctx, o)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != want {
		t.Fatalf("assessment=%s findings=%+v", p.Assessment.Outcome, p.Assessment.Findings)
	}
	a, err := packagev3.Assess(ctx, p.Manifest, p.Files, o.RuntimeVerifier)
	if err != nil || !reflect.DeepEqual(a, p.Assessment) {
		t.Fatalf("reassessment differs: %+v %v", a, err)
	}
	v, err := packagev3.Verify(ctx, packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files, RuntimeVerifier: o.RuntimeVerifier})
	if err != nil || !reflect.DeepEqual(v.Assessment(), p.Assessment) {
		t.Fatalf("verification differs: %v", err)
	}
}

func TestRuntimeCompositionParity(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"API to formatter to API", "", "", "compatible"},
		{"missing producer output", "$steps.read.outputs.heading", "$steps.read.outputs.missing", "incompatible"},
		{"missing formatter output", "$steps.format.outputs.formatted", "$steps.format.outputs.missing", "incompatible"},
		{"runtime output invalid projection", "$response.body'", "$response.body.missing'", "incompatible"},
		{"runtime header reference", "$response.body'", "$response.headers.x'", "incompatible"},
		{"future formatter reference", "'$steps.read.outputs.heading'", "'$steps.consume.outputs.message'", "incompatible"},
		{"incomplete body", "heading: '$steps.read.outputs.heading', ", "", "indeterminate"},
		{"extra body field", "body: '$variables.body'", "body: '$variables.body', extra: x", "incompatible"},
		{"selector missing", "function: message.format", "function: missing", "indeterminate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := compositionOptions(t)
			if tc.from != "" {
				o.WorkflowYAML = []byte(strings.ReplaceAll(string(o.WorkflowYAML), tc.from, tc.to))
			}
			checkAssessmentParity(t, o, tc.want)
		})
	}
}

func TestRuntimeMultipleOutputDiagnosticsDeterministic(t *testing.T) {
	ctx := context.Background()
	o := bodyOptions(t)
	o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "outputs: {formatted: '$response.body'}", "outputs: {a: '$response.body.missing', b: '$response.headers.x'}", 1))
	var first packagev3.Package
	for i := 0; i < 24; i++ {
		p, err := packagev3.Build(ctx, o)
		if err != nil {
			t.Fatal(err)
		}
		if p.Assessment.Outcome != "incompatible" {
			t.Fatalf("outcome: %+v", p.Assessment)
		}
		codes := map[string]bool{}
		for _, finding := range p.Assessment.Findings {
			codes[finding.Code] = true
		}
		if !codes["binding.output_field_missing"] || !codes["binding.output_missing"] {
			t.Fatalf("distinct output diagnostics absent: %+v", p.Assessment.Findings)
		}
		if i == 0 {
			first = p
		} else if p.SHA256 != first.SHA256 || !reflect.DeepEqual(p.Assessment, first.Assessment) {
			t.Fatalf("build %d changed assessment", i)
		}
		a, err := packagev3.Assess(ctx, p.Manifest, p.Files, o.RuntimeVerifier)
		if err != nil || !reflect.DeepEqual(a, p.Assessment) {
			t.Fatalf("assessment %d differs: %v", i, err)
		}
		v, err := packagev3.Verify(ctx, packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files, RuntimeVerifier: o.RuntimeVerifier})
		if err != nil || !reflect.DeepEqual(v.Assessment(), p.Assessment) {
			t.Fatalf("verification %d differs: %v", i, err)
		}
	}
}

func TestRuntimeEvidenceDriftRefused(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		change func(*packagev3.BuildOptions)
	}{
		{"stale revision", func(o *packagev3.BuildOptions) {
			o.Sources[0].Bytes = []byte(strings.Replace(string(o.Sources[0].Bytes), "6ebff379dc91fd7bb818ba8905a5e64eab5c490e", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1))
		}},
		{"changed catalog", func(o *packagev3.BuildOptions) { o.Sources[0].Bytes = append(o.Sources[0].Bytes, ' ') }},
		{"missing verifier", func(o *packagev3.BuildOptions) { o.RuntimeVerifier = nil }},
		{"changed source identity", func(o *packagev3.BuildOptions) { o.Sources[0].ID = "other-runtime" }},
		{"changed selector", func(o *packagev3.BuildOptions) {
			mutateRuntimeShape(t, o, func(s *binding.OperationShape) {
				s.Selector.Value = "message.forged"
				s.Selector.Key = "message.forged"
			})
		}},
		{"changed input identity", func(o *packagev3.BuildOptions) {
			mutateRuntimeShape(t, o, func(s *binding.OperationShape) { s.Inputs[0].Name = "body" })
		}},
		{"changed output identity", func(o *packagev3.BuildOptions) {
			mutateRuntimeShape(t, o, func(s *binding.OperationShape) { s.Outputs[0].Name = "body" })
		}},
		{"changed shape", func(o *packagev3.BuildOptions) {
			mutateRuntimeShape(t, o, func(s *binding.OperationShape) { s.Inputs[0].Schema.JSON = json.RawMessage(`true`) })
		}},
		{"incomplete shape", func(o *packagev3.BuildOptions) {
			mutateRuntimeShape(t, o, func(s *binding.OperationShape) { s.Complete = false })
		}},
		{"malformed shapes", func(o *packagev3.BuildOptions) { o.Sources[0].ShapesJSON = []byte(`{"version":`) }},
		{"malformed data", func(o *packagev3.BuildOptions) { o.DataJSON = []byte(`{"body":`) }},
		{"malformed workflow", func(o *packagev3.BuildOptions) { o.WorkflowYAML = []byte(`uws: [`) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := bodyOptions(t)
			tc.change(&o)
			if _, err := packagev3.Build(ctx, o); err == nil {
				t.Fatal("unverified changed evidence accepted")
			}
		})
	}
	// Rebind the altered source digest throughout the shape table so that the
	// independent verifier, rather than a superficial hash mismatch, rejects it.
	o := bodyOptions(t)
	o.Sources[0].Bytes = []byte(strings.Replace(string(o.Sources[0].Bytes), "6ebff379dc91fd7bb818ba8905a5e64eab5c490e", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1))
	table, err := binding.ParseTable(o.Sources[0].ShapesJSON)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(o.Sources[0].Bytes)
	identity := hex.EncodeToString(sum[:])
	table.Sources[0].SHA256 = identity
	for i := range table.Operations {
		table.Operations[i].Source.SHA256 = identity
	}
	o.Sources[0].ShapesJSON, err = table.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	verifier := o.RuntimeVerifier
	called := false
	o.RuntimeVerifier = func(ctx context.Context, revision string, data []byte, table binding.ShapeTable) error {
		called = true
		return verifier(ctx, revision, data, table)
	}
	if _, err := packagev3.Build(ctx, o); err == nil || !called {
		t.Fatal("coherent stale catalog bypassed independent verifier")
	}
	p, err := packagev3.Build(ctx, bodyOptions(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		verifier packagev3.RuntimeVerifier
	}{
		{"absent", nil}, {"wrong revision", func(_ context.Context, revision string, _ []byte, _ binding.ShapeTable) error {
			if revision != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
				return packagev3.ErrPackage
			}
			return nil
		}},
	} {
		t.Run("reassessment "+tc.name, func(t *testing.T) {
			if _, err := packagev3.Assess(ctx, p.Manifest, p.Files, tc.verifier); err == nil {
				t.Fatal("assessment accepted verifier refusal")
			}
			if _, err := packagev3.Verify(ctx, packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files, RuntimeVerifier: tc.verifier}); err == nil {
				t.Fatal("verification accepted verifier refusal")
			}
		})
	}
	// A rehashed altered table still cannot evade independent reproduction.
	table, err = binding.ParseTable(p.Files[packagev3.ShapesPath])
	if err != nil {
		t.Fatal(err)
	}
	for i := range table.Operations {
		if table.Operations[i].Selector.Value == "message.format" {
			table.Operations[i].Complete = false
		}
	}
	altered, err := table.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string][]byte, len(p.Files))
	for k, v := range p.Files {
		files[k] = v
	}
	files[packagev3.ShapesPath] = altered
	manifest := p.Manifest
	hash := sha256.Sum256(altered)
	manifest.Shapes.SHA256 = hex.EncodeToString(hash[:])
	if _, err := packagev3.Assess(ctx, manifest, files, bodyOptions(t).RuntimeVerifier); err == nil {
		t.Fatal("rehashed forged shape accepted")
	}
}

func mutateRuntimeShape(t *testing.T, o *packagev3.BuildOptions, change func(*binding.OperationShape)) {
	t.Helper()
	table, err := binding.ParseTable(o.Sources[0].ShapesJSON)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range table.Operations {
		if table.Operations[i].Selector.Value == "message.format" {
			change(&table.Operations[i])
			found = true
		}
	}
	if !found {
		t.Fatal("missing formatter")
	}
	o.Sources[0].ShapesJSON, err = table.Marshal()
	if err != nil {
		t.Fatal(err)
	}
}
