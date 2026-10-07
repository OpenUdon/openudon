package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/OpenUdon/openudon/approval"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

const runtimeRevision = "da43e57be37af4e18e633558580f740c525f139d"
const functionYAML = `uws: 1.13.0
info: {title: Function, version: '1'}
operations:
 - operationId: one
   effect: read
   x-uws-operation-profile: uws.runtime.1.0
   x-uws-runtime: {type: fnct, function: identity, arguments: [literal]}
workflows:
 - workflowId: main
   type: sequence
   steps: [{stepId: one, operationRef: one}]
`

func runtimeOptions(t *testing.T) packagev3.BuildOptions {
	t.Helper()
	catalog, err := os.ReadFile("testdata/runtime/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	shapes, err := os.ReadFile("testdata/runtime/shapes.json")
	if err != nil {
		t.Fatal(err)
	}
	// Unit adapter stands in for the separately qualified private consumer.
	// Exact independent expected bytes are generated and verified by published
	// Udon M48, never reconstructed from the caller's claimed table.
	verifier := func(ctx context.Context, revision string, data []byte, table binding.ShapeTable) error {
		encoded, err := table.Marshal()
		if err != nil || revision != runtimeRevision || !bytes.Equal(data, catalog) || !bytes.Equal(encoded, shapes) {
			return errors.New("runtime metadata mismatch")
		}
		return ctx.Err()
	}
	return packagev3.BuildOptions{Scope: "workflows/W01-function", WorkflowYAML: []byte(functionYAML), DataJSON: []byte(`{}`), Sources: []packagev3.SourceInput{{ID: packagev3.RuntimeSourceID, Kind: packagev3.RuntimeSourceKind, Path: "sources/runtime-function/catalog.json", Bytes: catalog, ShapesJSON: shapes}}, RuntimeVerifier: verifier}
}
func TestRuntimeCatalogRequiresIndependentConsumerVerifier(t *testing.T) {
	options := runtimeOptions(t)
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != "indeterminate" {
		t.Fatal("partial native function metadata became complete")
	}
	if _, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files}); err == nil {
		t.Fatal("no independent runtime verifier")
	}
	for _, mutate := range []func(*packagev3.BuildOptions){
		func(o *packagev3.BuildOptions) { o.RuntimeVerifier = nil },
		func(o *packagev3.BuildOptions) { o.Sources[0].Bytes = append(o.Sources[0].Bytes, ' ') },
		func(o *packagev3.BuildOptions) {
			table, err := binding.ParseTable(o.Sources[0].ShapesJSON)
			if err != nil {
				t.Fatal(err)
			}
			table.Operations[0].Complete = true
			o.Sources[0].ShapesJSON, _ = table.Marshal()
		},
		func(o *packagev3.BuildOptions) {
			table, err := binding.ParseTable(o.Sources[0].ShapesJSON)
			if err != nil {
				t.Fatal(err)
			}
			table.Operations[0].Security.Known = false
			o.Sources[0].ShapesJSON, _ = table.Marshal()
		},
		func(o *packagev3.BuildOptions) { o.Sources[0].ID = "producer-functions" },
	} {
		o := runtimeOptions(t)
		mutate(&o)
		if _, err := packagev3.Build(context.Background(), o); err == nil {
			t.Fatal("forged runtime source/table accepted")
		}
	}
}
func TestFunctionConstraintsBindSelectedRuntimeAndRequireAdmission(t *testing.T) {
	options := runtimeOptions(t)
	v := verifiedPackage(t, options)
	execution := executionOptions()
	execution.Worker.RuntimeRevision = runtimeRevision
	if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil {
		t.Fatal("native admission omitted")
	}
	calls := 0
	execution.RuntimeAdmission = func(ctx context.Context, r packagev3.RuntimeAdmissionRequest) error {
		calls++
		if r.RuntimeRevision != runtimeRevision || !bytes.Equal(r.WorkflowYAML, []byte(functionYAML)) {
			return errors.New("mismatch")
		}
		r.WorkflowYAML[0] = '!'
		return ctx.Err()
	}
	plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(plan.Operations) != 1 || plan.Operations[0].Kind != "fnct" || plan.Operations[0].Method != "" || plan.Operations[0].Origin != "" || plan.Operations[0].MetadataComplete || plan.Operations[0].Selector.Value != "identity" {
		t.Fatal("fabricated function proof or HTTP")
	}
	if v.Snapshot()[packagev3.WorkflowPath][0] != 'u' {
		t.Fatal("admission mutated custody")
	}
	execution.Worker.RuntimeRevision = strings.Repeat("c", 40)
	if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil {
		t.Fatal("wrong runtime revision")
	}
	execution.Worker.RuntimeRevision = runtimeRevision
	execution.RuntimeAdmission = func(context.Context, packagev3.RuntimeAdmissionRequest) error {
		return errors.New("unsafe native profile")
	}
	if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil {
		t.Fatal("native refusal ignored")
	}
	broker := brokerOptions()
	broker.Execution = execution
	if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, broker); err == nil {
		t.Fatal("function became HTTP broker authority")
	}
}
func TestOpenRuntimeValuesStayLossless(t *testing.T) {
	yaml := strings.Replace(functionYAML, "arguments: [literal]", "arguments: [9007199254740993]", 1)
	doc, _, err := packagev3.DecodeWorkflow(context.Background(), []byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	arg := doc.Operations[0].Extensions["x-uws-runtime"].(map[string]any)["arguments"].([]any)[0]
	if arg != json.Number("9007199254740993") {
		t.Fatal("runtime extension value rounded")
	}
}

func TestFunctionApprovalRequiresExactConfirmedWorkerPlan(t *testing.T) {
	options := runtimeOptions(t)
	v := verifiedPackage(t, options)
	execution := executionOptions()
	execution.Worker.RuntimeRevision = runtimeRevision
	execution.RuntimeAdmission = func(ctx context.Context, r packagev3.RuntimeAdmissionRequest) error { return ctx.Err() }
	plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	check := packagev3.ExecutionApprovalOptions{Execution: execution, ExpectedPlanSHA256: plan.PlanSHA256, Now: now, Tier: "sandbox", Approval: approval.Approval{Version: approval.Version, Scope: "workflows/W01-function", State: approval.StateApprovedForSandbox, Reviewer: "owner", ApprovedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Minute).Format(time.RFC3339), PackageSHA256: v.SHA256()}}
	if err := packagev3.CheckExecutionApproval(context.Background(), v, check); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*packagev3.ExecutionApprovalOptions){
		func(o *packagev3.ExecutionApprovalOptions) { o.ExpectedPlanSHA256 = "" },
		func(o *packagev3.ExecutionApprovalOptions) {
			o.Execution.Worker.ClosureSHA256 = strings.Repeat("c", 64)
		},
		func(o *packagev3.ExecutionApprovalOptions) { o.Approval.PackageSHA256 = strings.Repeat("c", 64) },
		func(o *packagev3.ExecutionApprovalOptions) { o.Approval.ExpiresAt = "" },
		func(o *packagev3.ExecutionApprovalOptions) {
			o.Approval.ApprovedAt = now.Add(time.Minute).Format(time.RFC3339)
		},
		func(o *packagev3.ExecutionApprovalOptions) { o.Approval.Scope = "workflows/W02-other" },
		func(o *packagev3.ExecutionApprovalOptions) { o.Now = now.Add(time.Hour) },
	} {
		c := check
		mutate(&c)
		if packagev3.CheckExecutionApproval(context.Background(), v, c) == nil {
			t.Fatal("approval retained without exact confirmed current constraints")
		}
	}
}
