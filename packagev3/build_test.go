package packagev3_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

const apiFixture = `openapi: 3.0.3
info: {title: Fixture, version: '1'}
servers: [{url: 'https://fixture.example.test'}]
security: []
paths:
  /value:
    get:
      operationId: fetch
      parameters:
        - {name: n, in: query, required: true, schema: {type: integer, minimum: 9007199254740993}}
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {type: object, properties: {ok: {type: boolean}}}
`
const yamlFixture = `uws: 1.13.0
info: {title: Fixture, version: '1'}
sourceDescriptions: [{name: api, type: openapi, url: sources/openapi/api.yaml}]
operations:
  - operationId: fetch
    sourceDescription: api
    sourceOperationId: fetch
    effect: read
    request: {query: {n: 9007199254740993}}
workflows:
  - workflowId: main
    type: sequence
    steps: [{stepId: fetch, operationRef: fetch}]
`

func buildOptions() packagev3.BuildOptions {
	return packagev3.BuildOptions{Scope: "workflows/W01-fixture", WorkflowYAML: []byte(yamlFixture), DataJSON: []byte(`{"precise":9007199254740993}`), Sources: []packagev3.SourceInput{{ID: "api", Kind: "openapi", Path: "sources/openapi/api.yaml", Bytes: []byte(apiFixture)}}}
}
func TestBuildExactReviewedBytesAndIndependentSnapshot(t *testing.T) {
	options := buildOptions()
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != "compatible" {
		t.Fatalf("assessment %s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
	}
	if !bytes.Equal(p.Files[packagev3.WorkflowPath], options.WorkflowYAML) || len(p.Files) != 7 || p.Handoff.ReviewState != "review_required" || len(p.SHA256) != 64 {
		t.Fatal("exact snapshot or identity")
	}
	options.WorkflowYAML[0] = '!'
	options.Sources[0].Bytes[0] = '!'
	if p.Files[packagev3.WorkflowPath][0] != 'u' || p.Files["sources/openapi/api.yaml"][0] != 'o' {
		t.Fatal("aliased input")
	}
	doc, raw, err := packagev3.DecodeWorkflow(context.Background(), p.Files[packagev3.WorkflowPath])
	if err != nil {
		t.Fatal(err)
	}
	value := doc.Operations[0].Request["query"].(map[string]any)["n"]
	if value != json.Number("9007199254740993") {
		t.Fatal("rounded model number")
	}
	if raw["operations"].([]any)[0].(map[string]any)["request"].(map[string]any)["query"].(map[string]any)["n"] != value {
		t.Fatal("projection diverged")
	}
	again, err := packagev3.Build(context.Background(), buildOptions())
	if err != nil || again.SHA256 != p.SHA256 {
		t.Fatal("non-deterministic package")
	}
}
func TestBuildAssessmentKeepsConcreteMismatchPendingAndUnknown(t *testing.T) {
	cases := []struct{ name, yaml, outcome, code string }{
		{"literal mismatch", strings.Replace(yamlFixture, "n: 9007199254740993", "n: 9007199254740992", 1), "incompatible", "binding.literal_schema_mismatch"},
		{"output missing", strings.Replace(yamlFixture, "    effect: read", "    effect: read\n    outputs: {ok: '$response.body#/missing'}", 1), "indeterminate", "binding.output_field_indeterminate"},
		{"source path", strings.Replace(yamlFixture, "sources/openapi/api.yaml", "https://remote.test/api.yaml", 1), "incompatible", "source.inventory"},
		{"nonportable", strings.Replace(yamlFixture, "n: 9007199254740993", "n: '$inputs.x + 1'", 1), "incompatible", "workflow.nonportable"},
		{"unknown expression", strings.Replace(yamlFixture, "n: 9007199254740993", "n: '$inputs.x'", 1), "indeterminate", ""},
		{"pending", `uws: 1.13.0
info: {title: Pending, version: '1'}
operations: []
workflows:
 - workflowId: main
   type: sequence
   steps:
    - stepId: waiting
      pending: {purpose: unresolved, inputs: {type: object}, outputs: {type: object}, effect: read}
`, "incompatible", "workflow.not_executable"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := buildOptions()
			o.WorkflowYAML = []byte(c.yaml)
			if c.name == "pending" {
				o.Sources = nil
			}
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if p.Assessment.Outcome != c.outcome {
				t.Fatalf("%s %+v", p.Assessment.Outcome, p.Assessment.Findings)
			}
			if c.code != "" {
				found := false
				for _, f := range p.Assessment.Findings {
					found = found || f.Code == c.code
				}
				if !found {
					t.Fatalf("missing %s: %+v", c.code, p.Assessment.Findings)
				}
			}
		})
	}
}
func TestBuildRefusesUnsafeBytesAndPreservesCancellation(t *testing.T) {
	for _, mutate := range []func(*packagev3.BuildOptions){
		func(o *packagev3.BuildOptions) { o.WorkflowYAML = []byte("uws: 1.13.0\nuws: 1.12.0") },
		func(o *packagev3.BuildOptions) {
			o.WorkflowYAML = append(o.WorkflowYAML, []byte("\n---\nuws: 1.13.0")...)
		},
		func(o *packagev3.BuildOptions) {
			o.WorkflowYAML = append(o.WorkflowYAML, []byte("\nvariables: &anchor {x: 1}\n")...)
		},
		func(o *packagev3.BuildOptions) { o.DataJSON = []byte(`{"x":1,"x":2}`) },
		func(o *packagev3.BuildOptions) {
			o.DataJSON = []byte(`{"client_secret":"m8Z-pQ4_R2x7N1cV9bK3sD6fH0jL5wT2"}`)
		},
		func(o *packagev3.BuildOptions) { o.Sources[0].Path = "sources/openapi/intent.hcl" },
		func(o *packagev3.BuildOptions) { o.Sources = append(o.Sources, o.Sources[0]) },
		func(o *packagev3.BuildOptions) {
			o.Private = func(b []byte) bool { return bytes.Contains(b, []byte("Fixture")) }
		},
		func(o *packagev3.BuildOptions) { o.WorkflowYAML = make([]byte, packagev3.MaxFileBytes+1) },
	} {
		o := buildOptions()
		mutate(&o)
		p, err := packagev3.Build(context.Background(), o)
		if !errors.Is(err, packagev3.ErrPackage) || p.Files != nil {
			t.Fatal("partial or unsafe package admitted", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := packagev3.Build(ctx, buildOptions()); !errors.Is(err, context.Canceled) {
		t.Fatal("lost cancellation")
	}
}
func TestAssessIndependentlyRefusesForgedOrStaleShapes(t *testing.T) {
	for _, mutate := range []func(*packagev3.Package){
		func(p *packagev3.Package) { p.Files["sources/openapi/api.yaml"][0] = '!' },
		func(p *packagev3.Package) {
			table, err := binding.ParseTable(p.Files[packagev3.ShapesPath])
			if err != nil {
				t.Fatal(err)
			}
			table.Operations[0].Method = "DELETE"
			data, err := table.Marshal()
			if err != nil {
				t.Fatal(err)
			}
			p.Files[packagev3.ShapesPath] = data
			p.Manifest.Shapes.SHA256 = sha(data)
		},
	} {
		p, err := packagev3.Build(context.Background(), buildOptions())
		if err != nil {
			t.Fatal(err)
		}
		mutate(&p)
		if _, err := packagev3.Assess(context.Background(), p.Manifest, p.Files); !errors.Is(err, packagev3.ErrPackage) {
			t.Fatal("forged source shape admitted", err)
		}
	}
}

func sha(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func TestBuildPreservesEightNativeSourceFamilies(t *testing.T) {
	names := []string{"openapi.yaml", "google-discovery.json", "aws-smithy.json", "asyncapi.yaml", "graphql.graphql", "openrpc.json", "grpc.proto", "odata.xml"}
	kinds := []string{"openapi", "google-discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata"}
	options := packagev3.BuildOptions{Scope: "workflows/W01-families", DataJSON: []byte(`{}`)}
	descriptions := []any{}
	for i, kind := range kinds {
		data, err := os.ReadFile(filepath.Join("testdata", "sources", names[i]))
		if err != nil {
			t.Fatal(err)
		}
		path := "sources/" + kind + "/" + names[i]
		options.Sources = append(options.Sources, packagev3.SourceInput{ID: kind, Kind: kind, Path: path, Bytes: data})
		descriptions = append(descriptions, map[string]any{"name": kind, "type": kind, "url": path})
	}
	raw := map[string]any{"uws": "1.13.0", "info": map[string]any{"title": "Families", "version": "1"}, "sourceDescriptions": descriptions, "operations": []any{}, "workflows": []any{map[string]any{"workflowId": "main", "type": "sequence", "steps": []any{map[string]any{"stepId": "waiting", "pending": map[string]any{"purpose": "Review all source contracts", "inputs": map[string]any{"type": "object"}, "outputs": map[string]any{"type": "object"}, "effect": "read"}}}}}}
	var err error
	options.WorkflowYAML, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile("testdata/eight-families.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p.Files[packagev3.ShapesPath], bytes.TrimSuffix(golden, []byte("\n"))) {
		t.Fatal("native shape/security wire differs from accepted APItools fixture")
	}
	if p.Assessment.Outcome != "incompatible" {
		t.Fatal("pending became executable")
	}
}
