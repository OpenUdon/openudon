package packagev3_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/binding"
	"github.com/OpenUdon/uws/mockruntime"
	"github.com/OpenUdon/uws/uws1"
)

// The native evaluator resolves the actual request reference; mock request
// records intentionally retain expressions rather than proving their values.
type exactChainRuntime struct {
	*mockruntime.Runtime
	secondValue any
}

func (r *exactChainRuntime) ExecuteLeafWithResult(ctx context.Context, op *uws1.Operation) (any, error) {
	if op.OperationID == "second" {
		text := op.Request["query"].(map[string]any)["n"].(string)
		value, err := r.Runtime.EvaluateExpression(ctx, text)
		if err != nil {
			return nil, err
		}
		r.secondValue = value
	}
	return r.Runtime.ExecuteLeafWithResult(ctx, op)
}

func chainOptions(leaf, parent, operationOutputs, stepOutputs string, reversed bool) packagev3.BuildOptions {
	o := buildOptions()
	o.Sources[0].Bytes = []byte(strings.Replace(apiFixture, "schema: {type: object, properties: {ok: {type: boolean}}}", "schema: {type: object, properties: {n: {"+leaf+"}}, "+parent+"}", 1))
	yaml := strings.Replace(yamlFixture, "    effect: read", "    effect: read\n    outputs: {"+operationOutputs+"}", 1)
	second := `  - operationId: second
    sourceDescription: api
    sourceOperationId: fetch
    effect: read
    request: {query: {n: '$steps.fetch.outputs.n'}}
`
	yaml = strings.Replace(yaml, "workflows:\n", second+"workflows:\n", 1)
	first := "{stepId: fetch, operationRef: fetch"
	if stepOutputs != "" {
		first += ", outputs: {" + stepOutputs + "}"
	}
	first += "}"
	steps := first + ", {stepId: second, operationRef: second}"
	if reversed {
		steps = "{stepId: second, operationRef: second}, " + first
	}
	o.WorkflowYAML = []byte(strings.Replace(yaml, "steps: [{stepId: fetch, operationRef: fetch}]", "steps: ["+steps+"]", 1))
	return o
}

const finiteInteger = "type: integer, minimum: 9007199254740993, enum: [9007199254740993, 9007199254740995]"

func TestDeclaredOutputOwnersAndTimingMatchNativeExecution(t *testing.T) {
	for _, c := range []struct {
		name, operation, step, workflow, outcome, failure string
		reversed, minimumOnly                             bool
	}{
		{name: "operation outputs do not populate step", operation: "n: '$response.body.n'", outcome: "incompatible", failure: "$steps.fetch.outputs.n is unresolved"},
		{name: "explicit step output", operation: "n: '$response.body.n'", step: "n: '$response.body.n'", outcome: "compatible"},
		{name: "step output cannot borrow operation current output", operation: "n: '$response.body.n'", step: "n: '$outputs.n'", outcome: "incompatible", failure: "$outputs.n is unresolved"},
		{name: "step preceding output", operation: "n: '$response.body.n'", step: "a: '$response.body.n', n: '$outputs.a'", outcome: "compatible"},
		{name: "operation preceding output", operation: "a: '$response.body.n', n: '$outputs.a'", step: "n: '$response.body.n'", outcome: "compatible"},
		{name: "workflow preceding output", operation: "n: '$response.body.n'", step: "n: '$response.body.n'", workflow: "a: '$steps.fetch.outputs.n', n: '$outputs.a'", outcome: "compatible"},
		{name: "workflow cannot borrow operation output", operation: "a: '$response.body.n'", step: "n: '$response.body.n'", workflow: "n: '$outputs.a'", outcome: "incompatible", failure: "$outputs.a is unresolved"},
		{name: "operation self output", operation: "n: '$outputs.n'", step: "n: '$response.body.n'", outcome: "incompatible", failure: "$outputs.n is unresolved"},
		{name: "step future output", operation: "n: '$response.body.n'", step: "a: '$outputs.n', n: '$response.body.n'", outcome: "incompatible", failure: "$outputs.n is unresolved"},
		{name: "step cyclic output", operation: "n: '$response.body.n'", step: "a: '$outputs.n', n: '$outputs.a'", outcome: "incompatible", failure: "$outputs.n is unresolved"},
		{name: "future step", operation: "n: '$response.body.n'", step: "n: '$response.body.n'", outcome: "incompatible", failure: "$steps.fetch has no execution record", reversed: true},
		{name: "minimum-only stays unproved", operation: "n: '$response.body.n'", step: "n: '$response.body.n'", outcome: "indeterminate", minimumOnly: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			leaf := finiteInteger
			if c.minimumOnly {
				leaf = "type: integer, minimum: 9007199254740993"
			}
			o := chainOptions(leaf, "required: [n]", c.operation, c.step, c.reversed)
			if c.workflow != "" {
				o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "    type: sequence", "    type: sequence\n    outputs: {"+c.workflow+"}", 1))
			}
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if p.Assessment.Outcome != c.outcome {
				t.Fatalf("Build %s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
			}
			if c.outcome == "compatible" {
				table, err := binding.ParseTable(p.Files[packagev3.ShapesPath])
				if err != nil {
					t.Fatal(err)
				}
				var schema map[string]any
				if wire.DecodeStrictNumbers(table.Operations[0].Outputs[0].Schema.JSON, &schema) != nil {
					t.Fatal("schema number decode")
				}
				child := schema["properties"].(map[string]any)["n"].(map[string]any)
				if child["minimum"] != json.Number("9007199254740993") || child["enum"].([]any)[0] != json.Number("9007199254740993") {
					t.Fatal("source schema numbers rounded")
				}
			}
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil || v.Assessment().Outcome != c.outcome {
				t.Fatal("Verify changed assessment", err)
			}
			_, err = packagev3.DeriveBrokerAuthority(context.Background(), v, brokerOptions())
			if (err == nil) != (c.outcome == "compatible") {
				t.Fatal("broker metadata did not retain refusal", err)
			}
			doc, _, err := packagev3.DecodeWorkflow(context.Background(), o.WorkflowYAML)
			if err != nil {
				t.Fatal(err)
			}
			if err := doc.ValidateExecutable(); err != nil {
				t.Fatal(err)
			}
			rt, err := mockruntime.NewRuntime(doc, mockruntime.Options{ResponseResolver: mockruntime.ResponseResolverFunc(func(context.Context, *uws1.Operation) (mockruntime.ResponseDefinition, error) {
				return mockruntime.ResponseDefinition{Example: json.RawMessage(`{"statusCode":200,"headers":{},"body":{"n":9007199254740993}}`)}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			observed := &exactChainRuntime{Runtime: rt}
			doc.SetRuntime(observed)
			err = doc.Execute(context.Background())
			if c.failure != "" {
				if err == nil || !strings.Contains(err.Error(), c.failure) {
					t.Fatalf("native missing/current/future output not refused: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if doc.ExecutionRecords()["step:fetch"].Outputs["n"] != json.Number("9007199254740993") || observed.secondValue != json.Number("9007199254740993") || len(rt.RequestRecords()) != 2 {
					t.Fatal("native chain lost exact output or input")
				}
			}
		})
	}
}

func TestOwnedResponseProjectionRetainsDialectNumbersAndParentConstraints(t *testing.T) {
	for _, c := range []struct{ name, leaf, parent, dialect, outcome string }{
		{"finite large integer", finiteInteger, "required: [n]", "3.0.3", "compatible"},
		{"original minimum only", "type: integer, minimum: 9007199254740993", "required: [n]", "3.0.3", "indeterminate"},
		{"3.0 const unproved", "type: integer, minimum: 9007199254740993, const: 9007199254740993", "required: [n]", "3.0.3", "indeterminate"},
		{"3.1 const", "type: integer, minimum: 9007199254740993, const: 9007199254740993", "required: [n]", "3.1.0", "compatible"},
		{"rounded value", "type: integer, minimum: 0, const: 9007199254740992", "required: [n]", "3.1.0", "incompatible"},
		{"boolean value", "type: boolean", "required: [n]", "3.0.3", "incompatible"},
		{"contradictory parent", finiteInteger, "required: [n], maxProperties: 0", "3.0.3", "incompatible"},
		{"unknown parent", finiteInteger, "required: [n], propertyNames: false", "3.1.0", "indeterminate"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := chainOptions(c.leaf, c.parent, "n: '$response.body.n'", "n: '$response.body.n'", false)
			o.Sources[0].Bytes = []byte(strings.Replace(string(o.Sources[0].Bytes), "openapi: 3.0.3", "openapi: "+c.dialect, 1))
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if p.Assessment.Outcome != c.outcome {
				t.Fatalf("%s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
			}
			v := verifiedPackage(t, o)
			if v.Assessment().Outcome != c.outcome {
				t.Fatal("source proof changed outcome")
			}
			if _, err = packagev3.DeriveBrokerAuthority(context.Background(), v, brokerOptions()); (err == nil) != (c.outcome == "compatible") {
				t.Fatal("unproved binding acquired metadata authority", err)
			}
		})
	}
}
