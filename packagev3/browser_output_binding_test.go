package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/binding"
)

func countBrowserOptions(t *testing.T) packagev3.BuildOptions {
	t.Helper()
	for _, vector := range browserVectors(t) {
		if vector.Source == "action-1.10.json" {
			if vector.SHA256 != "56b48d53a4f1e6834bccab3bb4c1027b852b46fdd5b48986642c1791890bed87" {
				t.Fatal("P11 source identity changed")
			}
			return browserOptions(t, vector)
		}
	}
	t.Fatal("missing frozen count browser source")
	return packagev3.BuildOptions{}
}

// The exact published M33 declarations and C10 validation are the producer
// control. Do not rename the source outputs or manufacture a body wrapper.
func TestBrowserNamedOutputPublishedShapeControl(t *testing.T) {
	o := countBrowserOptions(t)
	opts := browsertools.BrowserShapeOptions{Sources: []browsertools.BrowserShapeSourceInput{{ID: "browser", Content: o.Sources[0].Bytes}}}
	table, err := browsertools.BuildBrowserShapeTable(context.Background(), opts)
	if err != nil || len(table.Operations) != 1 {
		t.Fatal("M33 shape construction", err)
	}
	if err := browsertools.VerifyBrowserShapeTable(context.Background(), opts, table); err != nil {
		t.Fatal("independent M33 shape reproduction", err)
	}
	var source struct {
		Actions map[string]struct {
			Outputs map[string]json.RawMessage `json:"outputs"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(o.Sources[0].Bytes, &source); err != nil {
		t.Fatal(err)
	}
	shape := table.Operations[0]
	if len(shape.Outputs) != 2 || shape.Outputs[0].Name != "all_result_count" || shape.Outputs[1].Name != "result_count" {
		t.Fatal("named outputs changed", shape.Outputs)
	}
	for _, output := range shape.Outputs {
		declaration, err := browsercontract.CanonicalJSON(source.Actions["inspect_results"].Outputs[output.Name])
		if err != nil || output.Location != "body" || !output.Schema.Known || !bytes.Equal(output.Schema.JSON, declaration) {
			t.Fatal("full native declaration not retained", output.Name, err)
		}
	}
	var count map[string]any
	if err := wire.DecodeStrictNumbers(shape.Outputs[1].Schema.JSON, &count); err != nil {
		t.Fatal(err)
	}
	if count["validation"].(map[string]any)["maximum"] != json.Number("9007199254740991") {
		t.Fatal("native numeric spelling changed")
	}
	resolver, err := binding.NewResolver(table)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, output, outcome, code string
	}{
		{"rendered count", "result_count", "compatible", ""},
		{"all count", "all_result_count", "compatible", ""},
		{"missing name", "missing", "incompatible", "binding.output_missing"},
		{"invented body wrapper", "body", "incompatible", "binding.output_missing"},
	} {
		t.Run(c.name, func(t *testing.T) {
			report, err := binding.ValidateBinding(context.Background(), resolver, binding.Request{
				Binding:          binding.Binding{Source: shape.Source, SelectorKind: "id", SelectorValue: "inspect_results"},
				OutputReferences: []binding.OutputReference{{Location: "body", Name: c.output}},
			})
			if err != nil || string(report.Outcome) != c.outcome {
				t.Fatal("C10 control", report, err)
			}
			if c.code != "" && (len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != c.code) {
				t.Fatal("C10 control diagnostic", report)
			}
		})
	}
}

func browserOutputOptions(t *testing.T, operation, step, workflow, consumer, consumerType string, reversed bool) packagev3.BuildOptions {
	t.Helper()
	o := countBrowserOptions(t)
	yaml := string(o.WorkflowYAML)
	if operation != "" {
		yaml = strings.Replace(yaml, "    effect: read", "    effect: read\n    outputs: {"+operation+"}", 1)
	}
	first := "{stepId: leaf, operationRef: op"
	if step != "" {
		first += ", outputs: {" + step + "}"
	}
	first += "}"
	steps := first
	if consumer != "" {
		// Mixed review packages are supported; this is source-backed assessment,
		// never execution-plan admission or an HTTP/browser runtime.
		api := strings.Replace(apiFixture, "type: integer, minimum: 9007199254740993", "type: "+consumerType, 1)
		o.Sources = append(o.Sources, packagev3.SourceInput{ID: "api", Kind: "openapi", Path: "sources/openapi/api.yaml", Bytes: []byte(api)})
		yaml = strings.Replace(yaml, "sourceDescriptions: [{name: browser, type: browser-profile, url: sources/browser-profile/action-1.10.json}]", "sourceDescriptions: [{name: browser, type: browser-profile, url: sources/browser-profile/action-1.10.json}, {name: api, type: openapi, url: sources/openapi/api.yaml}]", 1)
		yaml = strings.Replace(yaml, "workflows:\n", "  - operationId: consumer\n    sourceDescription: api\n    sourceOperationId: fetch\n    effect: read\n    request: {query: {n: '"+consumer+"'}}\nworkflows:\n", 1)
		steps += ", {stepId: consumer, operationRef: consumer}"
		if reversed {
			steps = "{stepId: consumer, operationRef: consumer}, " + first
		}
	}
	yaml = strings.Replace(yaml, "steps: [{stepId: leaf, operationRef: op}]", "steps: ["+steps+"]", 1)
	if workflow != "" {
		yaml = strings.Replace(yaml, "    type: sequence", "    type: sequence\n    outputs: {"+workflow+"}", 1)
	}
	o.WorkflowYAML = []byte(yaml)
	return o
}

// P11.1 intentionally characterizes the accepted P10 defect. P11.2 must change
// the valid cases to compatible after correcting native expression resolution;
// these incompatible expectations are not acceptance of delivered support.
func TestBrowserNamedOutputP11Characterization(t *testing.T) {
	for _, c := range []struct {
		name, operation, step, workflow, consumer, consumerType, outcome string
		codes                                                            []string
		reversed                                                         bool
	}{
		{name: "no output control", outcome: "compatible"},
		{name: "operation pointer", operation: "count: '$response.body#/result_count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "operation dot", operation: "count: '$response.body.result_count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "other named output", operation: "count: '$response.body#/all_result_count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "step pointer", step: "count: '$response.body#/result_count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible"}},
		{name: "operation and step", operation: "count: '$response.body#/result_count'", step: "count: '$response.body#/result_count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "step preceding output", step: "a: '$response.body#/result_count', count: '$outputs.a'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible"}},
		{name: "workflow export", step: "count: '$response.body#/result_count'", workflow: "count: '$steps.leaf.outputs.count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible"}},
		{name: "integer consumer", step: "count: '$response.body#/result_count'", consumer: "$steps.leaf.outputs.count", consumerType: "integer", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.expression_reference_incompatible"}},
		{name: "string consumer", step: "count: '$response.body#/result_count'", consumer: "$steps.leaf.outputs.count", consumerType: "string", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.expression_reference_incompatible"}},
		{name: "operation does not populate step", operation: "count: '$response.body#/result_count'", consumer: "$steps.leaf.outputs.count", consumerType: "integer", outcome: "incompatible", codes: []string{"binding.expression_reference_incompatible"}},
		{name: "future step", step: "count: '$response.body#/result_count'", consumer: "$steps.leaf.outputs.count", consumerType: "integer", reversed: true, outcome: "incompatible", codes: []string{"binding.expression_reference_incompatible"}},
		{name: "missing declared name", operation: "count: '$response.body#/missing'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "whole body has no declaration", operation: "count: '$response.body'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "escaped slash names missing", operation: "count: '$response.body#/result~1count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "escaped tilde names missing", operation: "count: '$response.body#/result~0count'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "nested scalar path", operation: "count: '$response.body#/result_count/nested'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible", "binding.output_missing"}},
		{name: "nested dot scalar path", step: "count: '$response.body.result_count.nested'", outcome: "incompatible", codes: []string{"binding.output_reference_incompatible"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := browserOutputOptions(t, c.operation, c.step, c.workflow, c.consumer, c.consumerType, c.reversed)
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal("build", err)
			}
			if p.Assessment.Outcome != c.outcome {
				t.Fatalf("current P10 outcome %s, want %s: %+v", p.Assessment.Outcome, c.outcome, p.Assessment.Findings)
			}
			for _, code := range c.codes {
				found := false
				for _, finding := range p.Assessment.Findings {
					found = found || finding.Code == code
				}
				if !found {
					t.Fatalf("missing diagnostic %s: %+v", code, p.Assessment.Findings)
				}
			}
			assessment, err := packagev3.Assess(context.Background(), p.Manifest, p.Files)
			if err != nil || !reflect.DeepEqual(assessment, p.Assessment) {
				t.Fatal("independent Assess changed proof", err)
			}
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil || !reflect.DeepEqual(v.Assessment(), p.Assessment) {
				t.Fatal("independent Verify changed proof", err)
			}
			if !bytes.Equal(p.Files[o.Sources[0].Path], o.Sources[0].Bytes) || !bytes.Equal(p.Files[packagev3.WorkflowPath], o.WorkflowYAML) {
				t.Fatal("reviewed source/workflow bytes changed")
			}
		})
	}
}

func TestBrowserNamedOutputMalformedReferencesRefuse(t *testing.T) {
	for _, expression := range []string{
		"$response.body#/result_count/~2",
		"$response.body#/result_count/~",
		"$response.body#result_count",
	} {
		t.Run(expression, func(t *testing.T) {
			o := browserOutputOptions(t, "count: '"+expression+"'", "", "", "", "", false)
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal("malformed-expression review package", err)
			}
			if p.Assessment.Outcome != "incompatible" {
				t.Fatal("malformed reference qualified", p.Assessment)
			}
			found := false
			for _, finding := range p.Assessment.Findings {
				found = found || finding.Code == "binding.output_reference_incompatible"
			}
			if !found {
				t.Fatal("malformed reference lost native diagnostic", p.Assessment.Findings)
			}
		})
	}
}

func TestBrowserOutputCorrectionHTTPProjectionControls(t *testing.T) {
	for _, c := range []struct {
		name, expression, schema, outcome string
	}{
		{"whole body", "$response.body", "{type: object, properties: {ok: {type: boolean}}}", "compatible"},
		{"escaped nested pointer", "$response.body#/a~1b/~0key", "{type: object, properties: {'a/b': {type: object, properties: {'~key': {" + finiteInteger + "}}, required: ['~key']}}, required: ['a/b']}", "compatible"},
		{"missing escaped child", "$response.body#/a~1b/~0missing", "{type: object, properties: {'a/b': {type: object, properties: {'~key': {type: integer}}, required: ['~key'], additionalProperties: false}}, required: ['a/b'], additionalProperties: false}", "incompatible"},
		{"optional nested pointer", "$response.body#/a~1b/~0key", "{type: object, properties: {'a/b': {type: object, properties: {'~key': {type: integer}}, required: ['~key']}}}", "indeterminate"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := buildOptions()
			o.Sources[0].Bytes = []byte(strings.Replace(strings.Replace(apiFixture, "openapi: 3.0.3", "openapi: 3.1.0", 1), "{type: object, properties: {ok: {type: boolean}}}", c.schema, 1))
			o.WorkflowYAML = []byte(strings.Replace(yamlFixture, "    effect: read", "    effect: read\n    outputs: {value: '"+c.expression+"'}", 1))
			p, err := packagev3.Build(context.Background(), o)
			if err != nil || p.Assessment.Outcome != c.outcome {
				t.Fatal("HTTP projection control", p.Assessment, err)
			}
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil || !reflect.DeepEqual(v.Assessment(), p.Assessment) {
				t.Fatal("HTTP source reproduction", err)
			}
		})
	}
}
