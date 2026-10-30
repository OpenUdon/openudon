package packagev3_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

const bodyWorkflow = `uws: 1.13.0
info: {title: Format, version: '1'}
operations:
 - operationId: format
   effect: read
   x-uws-operation-profile: uws.runtime.1.0
   x-uws-runtime: {type: fnct, function: message.format}
   request: {body: {heading: Hello, body: World}}
   outputs: {formatted: '$response.body'}
workflows:
 - workflowId: main
   type: sequence
   steps: [{stepId: format, operationRef: format, outputs: {formatted: '$response.body'}}]
   outputs: {result: '$steps.format.outputs.formatted'}
`

func bodyOptions(t *testing.T) packagev3.BuildOptions {
	t.Helper()
	catalog, err := os.ReadFile("testdata/runtime-p17/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	shapes, err := os.ReadFile("testdata/runtime-p17/shapes.json")
	if err != nil {
		t.Fatal(err)
	}
	verifier := func(ctx context.Context, revision string, data []byte, table binding.ShapeTable) error {
		encoded, err := table.Marshal()
		if err != nil || revision != "6ebff379dc91fd7bb818ba8905a5e64eab5c490e" || !bytes.Equal(data, catalog) || !bytes.Equal(encoded, shapes) {
			return errors.New("different runtime")
		}
		return ctx.Err()
	}
	return packagev3.BuildOptions{Scope: "workflows/W01-format", WorkflowYAML: []byte(bodyWorkflow), DataJSON: []byte(`{}`), Sources: []packagev3.SourceInput{{ID: packagev3.RuntimeSourceID, Kind: packagev3.RuntimeSourceKind, Path: "sources/runtime-function/catalog.json", Bytes: catalog, ShapesJSON: shapes}}, RuntimeVerifier: verifier}
}

func TestRuntimeBodyProof(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct{ name, from, to, want string }{
		{"complete", "", "", "compatible"},
		{"missing field", "heading: Hello, ", "", "incompatible"},
		{"wrong type", "heading: Hello", "heading: 3", "incompatible"},
		{"whole body expression", "{heading: Hello, body: World}", "'$variables.payload'", "indeterminate"},
		{"field expression", "heading: Hello", "heading: '$variables.heading'", "compatible"},
		{"array body", "{heading: Hello, body: World}", "[Hello, World]", "indeterminate"},
		{"unknown function", "message.format", "unknown", "indeterminate"},
		{"missing body", "request: {body: {heading: Hello, body: World}}", "request: {}", "indeterminate"},
		{"source bound spoof", "x-uws-operation-profile: uws.runtime.1.0", "sourceDescription: udon-runtime-functions\n   sourceOperationId: message.format\n   x-uws-operation-profile: uws.runtime.1.0", "incompatible"},
		{"unsupported argument", "function: message.format}", "function: message.format, arguments: [x]}", "indeterminate"},
		{"wrong output", "$response.body'", "$response.headers.foo'", "incompatible"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := bodyOptions(t)
			o.DataJSON = []byte(`{"payload":{"heading":"Hello","body":"World"},"heading":"Hello"}`)
			table, e := binding.ParseTable(o.Sources[0].ShapesJSON)
			if e != nil {
				t.Fatalf("table: %v", e)
			}
			if e = o.RuntimeVerifier(ctx, "6ebff379dc91fd7bb818ba8905a5e64eab5c490e", o.Sources[0].Bytes, table); e != nil {
				t.Fatalf("fixture verifier: %v", e)
			}
			if tc.from != "" {
				o.WorkflowYAML = []byte(strings.ReplaceAll(string(o.WorkflowYAML), tc.from, tc.to))
			}
			if _, _, err := packagev3.DecodeWorkflow(ctx, o.WorkflowYAML); err != nil {
				t.Fatalf("workflow: %v", err)
			}
			p, err := packagev3.Build(ctx, o)
			if err != nil {
				t.Fatal(err)
			}
			if p.Assessment.Outcome != tc.want {
				t.Fatalf("got %s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
			}
			if _, err = packagev3.Verify(ctx, packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files, RuntimeVerifier: o.RuntimeVerifier}); err != nil {
				t.Fatal(err)
			}
		})
	}
	o := bodyOptions(t)
	v := verifiedPackage(t, o)
	execution := executionOptions()
	execution.Worker.RuntimeRevision = "6ebff379dc91fd7bb818ba8905a5e64eab5c490e"
	if _, err := packagev3.DeriveExecutionPlan(ctx, v, execution); err == nil {
		t.Fatal("compatible assessment admitted a runtime without independent admission")
	}
	for _, change := range []func(*packagev3.BuildOptions){
		func(o *packagev3.BuildOptions) { o.RuntimeVerifier = nil },
		func(o *packagev3.BuildOptions) { o.Sources[0].Bytes = append(o.Sources[0].Bytes, ' ') },
		func(o *packagev3.BuildOptions) {
			table, _ := binding.ParseTable(o.Sources[0].ShapesJSON)
			table.Operations[4].Complete = false
			o.Sources[0].ShapesJSON, _ = table.Marshal()
		},
	} {
		o := bodyOptions(t)
		change(&o)
		if _, err := packagev3.Build(ctx, o); err == nil {
			t.Fatal("forged or absent evidence accepted")
		}
	}
}
