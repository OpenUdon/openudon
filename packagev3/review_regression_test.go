package packagev3_test

import (
	"context"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/packagev3"
)

func TestMissingFlowReferenceCannotQualifyAuthority(t *testing.T) {
	options := buildOptions()
	options.WorkflowYAML = []byte(strings.Replace(yamlFixture, "    type: sequence", "    type: sequence\n    outputs: {result: '$steps.missing.outputs.value'}", 1))
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome == "compatible" {
		t.Fatalf("unproved flow references qualified review: %+v", p.Assessment.Findings)
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, brokerOptions()); err == nil {
		t.Fatal("unproved flow reference qualified concrete authority")
	}
}

func TestExactReviewedVariableInputCanQualifyBinding(t *testing.T) {
	options := buildOptions()
	options.WorkflowYAML = []byte(strings.Replace(yamlFixture, "n: 9007199254740993", "n: '$variables.inputs.n'", 1))
	options.DataJSON = []byte(`{"inputs":{"n":9007199254740993}}`)
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != "compatible" {
		t.Fatalf("exact reviewed input contract not derived: %+v", p.Assessment.Findings)
	}
}

func TestAllCoreResponseFormsCheckSourceContracts(t *testing.T) {
	cases := []struct{ name, expr, spec, want string }{
		{"dot missing closed", "$response.body.missing", strings.Replace(strings.Replace(apiFixture, "openapi: 3.0.3", "openapi: 3.1.0", 1), "properties: {ok: {type: boolean}}", "properties: {ok: {type: boolean}}, additionalProperties: false", 1), "incompatible"},
		{"header missing", "$response.headers.X-Missing", apiFixture, "incompatible"},
		{"optional field", "$response.body.ok", apiFixture, "indeterminate"},
		{"required field", "$response.body.ok", strings.Replace(apiFixture, "properties: {ok: {type: boolean}}", "properties: {ok: {type: boolean}}, required: [ok]", 1), "compatible"},
		{"required nullable object", "$response.body.ok", strings.Replace(apiFixture, "type: object, properties: {ok: {type: boolean}}", "type: [object, 'null'], properties: {ok: {type: boolean}}, required: [ok]", 1), "indeterminate"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := buildOptions()
			o.Sources[0].Bytes = []byte(c.spec)
			o.WorkflowYAML = []byte(strings.Replace(yamlFixture, "    effect: read", "    effect: read\n    outputs: {value: '"+c.expr+"'}", 1))
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if p.Assessment.Outcome != c.want {
				t.Fatalf("%s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
			}
		})
	}
}
func TestSourceBackedChainedInputTypesRemainExact(t *testing.T) {
	spec := strings.Replace(apiFixture, "schema: {type: object, properties: {ok: {type: boolean}}}", "schema: {type: object, properties: {n: {type: integer, minimum: 9007199254740993}}, required: [n]}", 1)
	o := buildOptions()
	o.Sources[0].Bytes = []byte(spec)
	yaml := strings.Replace(yamlFixture, "    effect: read", "    effect: read\n    outputs: {n: '$response.body.n'}", 1)
	second := `  - operationId: second
    sourceDescription: api
    sourceOperationId: fetch
    effect: read
    request: {query: {n: '$steps.fetch.outputs.n'}}
`
	yaml = strings.Replace(yaml, "workflows:\n", second+"workflows:\n", 1)
	yaml = strings.Replace(yaml, "steps: [{stepId: fetch, operationRef: fetch}]", "steps: [{stepId: fetch, operationRef: fetch}, {stepId: second, operationRef: second}]", 1)
	o.WorkflowYAML = []byte(yaml)
	p, err := packagev3.Build(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != "compatible" {
		t.Fatalf("source-backed chain %s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
	}
	// Reversing the dependency cannot acquire a record from a future step.
	o.WorkflowYAML = []byte(strings.Replace(yaml, "steps: [{stepId: fetch, operationRef: fetch}, {stepId: second, operationRef: second}]", "steps: [{stepId: second, operationRef: second}, {stepId: fetch, operationRef: fetch}]", 1))
	p, err = packagev3.Build(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != "incompatible" {
		t.Fatal("future step qualified", p.Assessment.Findings)
	}
}
