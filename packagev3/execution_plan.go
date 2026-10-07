package packagev3

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"reflect"
	"strings"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/uws/binding"
	"github.com/OpenUdon/uws/expressions"
	"github.com/OpenUdon/uws/runtimes"
	"github.com/OpenUdon/uws/uws1"
)

const ExecutionPlanVersion = "openudon.execution-plan.v1"

type WorkerIdentity struct {
	BinarySHA256    string `json:"binary_sha256"`
	ClosureSHA256   string `json:"closure_sha256"`
	RuntimeRevision string `json:"runtime_revision"`
}

func (w WorkerIdentity) valid() bool {
	if !digestValid(w.BinarySHA256) || !digestValid(w.ClosureSHA256) || len(w.RuntimeRevision) != 40 || strings.ToLower(w.RuntimeRevision) != w.RuntimeRevision {
		return false
	}
	_, err := hex.DecodeString(w.RuntimeRevision)
	return err == nil
}

type RuntimeAdmissionRequest struct {
	RuntimeRevision string
	WorkflowYAML    []byte
	DataJSON        []byte
	Sources         map[string][]byte
}

// RuntimeAdmission calls the selected implementing runtime's non-effectful
// Compile/CheckSupported admission in the isolated consuming worker. It must
// bind the actual runtime revision/closure, use only supplied snapshots and
// refuse unsupported profiles; it cannot grant host egress or credentials.
type RuntimeAdmission func(context.Context, RuntimeAdmissionRequest) error

type ExecutionOptions struct {
	Worker           WorkerIdentity
	RuntimeAdmission RuntimeAdmission
}

type ExecutionOperation struct {
	StepID            string           `json:"step_id"`
	OperationID       string           `json:"operation_id"`
	InvocationID      string           `json:"invocation_id"`
	Kind              string           `json:"kind"`
	Source            binding.Source   `json:"source"`
	Selector          binding.Selector `json:"selector"`
	Method            string           `json:"method"`
	Origin            string           `json:"origin"`
	PathTemplate      string           `json:"path_template"`
	ConstraintsSHA256 string           `json:"constraints_sha256"`
	MetadataComplete  bool             `json:"metadata_complete"`
	Security          binding.Security `json:"security"`
}

// ExecutionPlan is exact review metadata for a bounded sequence. Its digest
// binds inputs, full package/handoff, every selected operation and worker.
// MetadataComplete never supplies permission; current host grants remain
// independently checked. No credential value or workflow value is returned.
type ExecutionPlan struct {
	Version           string               `json:"version"`
	Scope             string               `json:"scope"`
	PackageSHA256     string               `json:"package_sha256"`
	HandoffSHA256     string               `json:"handoff_sha256"`
	InputsSHA256      string               `json:"inputs_sha256"`
	Worker            WorkerIdentity       `json:"worker"`
	AssessmentOutcome string               `json:"assessment_outcome"`
	Operations        []ExecutionOperation `json:"operations"`
	PlanSHA256        string               `json:"plan_sha256"`
}

func (p ExecutionPlan) Digest() string {
	p.PlanSHA256 = ""
	data, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return hashBytes(data)
}

func flatStep(step *uws1.Step) bool {
	return step != nil && step.OperationRef != "" && step.Pending == nil && step.Type == "" && len(step.Steps)+len(step.Cases)+len(step.Default) == 0 && step.Workflow == "" && step.When == "" && step.ForEach == "" && step.Wait == "" && step.Timeout == nil && step.ParallelGroup == "" && step.Items == "" && step.Mode == "" && step.BatchSize == ""
}
func flatOperation(op *uws1.Operation) bool {
	return op != nil && len(op.DependsOn) == 0 && op.When == "" && op.ForEach == "" && op.Wait == "" && op.Timeout == nil && op.ParallelGroup == "" && len(op.OnFailure)+len(op.OnSuccess) == 0
}

func functionName(op *uws1.Operation) (string, error) {
	if !op.IsExtensionOwned() || op.ExtensionProfile() != runtimes.ProfileName {
		return "", ErrPackage
	}
	value, ok := op.Extensions[runtimes.ExtensionRuntime]
	if !ok {
		return "", ErrPackage
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", ErrPackage
	}
	// Use the public neutral profile fields, with lossless decoding and a closed
	// key set; legacy aliases or runtime overrides cannot hide an extra effect.
	var profile runtimes.OperationRuntime
	if decodeProfile(encoded, &profile) != nil || profile.Type != runtimes.RuntimeTypeFnct || profile.Function == "" || profile.Command != "" || profile.WorkingDir != "" || profile.Workflow != "" {
		return "", ErrPackage
	}
	for key := range op.Extensions {
		if key != uws1.ExtensionOperationProfile && key != runtimes.ExtensionRuntime {
			return "", ErrPackage
		}
	}
	return profile.Function, nil
}

func fixedHTTP(shape binding.OperationShape) (string, error) {
	if shape.Protocol != "http" || !shape.Complete || !shape.Security.Known || len(shape.Servers) != 1 || !strings.HasPrefix(shape.Path, "/") || strings.ContainsAny(shape.Path, "\r\n?#") || strings.HasPrefix(shape.Path, "//") {
		return "", ErrPackage
	}
	server, err := url.Parse(shape.Servers[0])
	if err != nil || (server.Scheme != "http" && server.Scheme != "https") || server.Hostname() == "" || server.User != nil || server.RawQuery != "" || server.ForceQuery || server.Fragment != "" || server.RawPath != "" || server.Path != "" && server.Path != "/" {
		return "", ErrPackage
	}
	switch shape.Method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
	default:
		return "", ErrPackage
	}
	return server.Scheme + "://" + server.Host, nil
}

func DeriveExecutionPlan(ctx context.Context, verified VerifiedPackage, options ExecutionOptions) (ExecutionPlan, error) {
	if ctx == nil || verified.files == nil || !digestValid(verified.sha256) || !options.Worker.valid() || verified.assessment.Outcome == "incompatible" {
		return ExecutionPlan{}, ErrPackage
	}
	if ctx.Err() != nil {
		return ExecutionPlan{}, ctx.Err()
	}
	doc, raw, err := DecodeWorkflow(ctx, verified.files[WorkflowPath])
	if err != nil || validateWorkflowSchema(raw) != nil || doc.Validate() != nil || doc.ValidateExecutable() != nil || doc.ValidateExecutionEntrypoint() != nil || len(expressions.CheckPortability(doc)) != 0 || len(doc.Workflows) != 1 || len(doc.Triggers)+len(doc.Results) != 0 || len(doc.Extensions) != 0 {
		return ExecutionPlan{}, ErrPackage
	}
	workflow := doc.Workflows[0]
	if workflow.Type != uws1.WorkflowTypeSequence || len(workflow.Steps) == 0 || len(workflow.Steps) > authority.MaxOperations || len(workflow.Cases)+len(workflow.Default)+len(workflow.DependsOn) != 0 || workflow.When != "" || workflow.ForEach != "" || workflow.Wait != "" || workflow.Timeout != nil || workflow.Items != "" || workflow.Mode != "" || workflow.BatchSize != "" || len(workflow.Extensions) != 0 {
		return ExecutionPlan{}, ErrPackage
	}
	table, err := binding.ParseTable(verified.files[ShapesPath])
	if err != nil {
		return ExecutionPlan{}, ErrPackage
	}
	resolver, err := binding.NewResolver(table)
	if err != nil {
		return ExecutionPlan{}, ErrPackage
	}
	operations := map[string]*uws1.Operation{}
	rawOperations := map[string]any{}
	for _, op := range doc.Operations {
		if !flatOperation(op) || !authority.Identifier(op.OperationID) {
			return ExecutionPlan{}, ErrPackage
		}
		operations[op.OperationID] = op
	}
	for _, value := range raw["operations"].([]any) {
		object := value.(map[string]any)
		id := object["operationId"].(string)
		rawOperations[id] = object
	}
	rawSteps := raw["workflows"].([]any)[0].(map[string]any)["steps"].([]any)
	plan := ExecutionPlan{Version: ExecutionPlanVersion, Scope: verified.manifest.Scope, PackageSHA256: verified.sha256, HandoffSHA256: hashBytes(verified.files[HandoffPath]), InputsSHA256: verified.assessment.InputsSHA256, Worker: options.Worker, AssessmentOutcome: verified.assessment.Outcome, Operations: []ExecutionOperation{}}
	used := map[string]bool{}
	needsRuntime := false
	for index, step := range workflow.Steps {
		if ctx.Err() != nil {
			return ExecutionPlan{}, ctx.Err()
		}
		if !flatStep(step) || !authority.Identifier(step.StepID) || len(step.Extensions) != 0 || used[step.OperationRef] {
			return ExecutionPlan{}, ErrPackage
		}
		for _, dependency := range step.DependsOn {
			found := false
			for _, prior := range plan.Operations {
				found = found || prior.StepID == dependency
			}
			if !found {
				return ExecutionPlan{}, ErrPackage
			}
		}
		op := operations[step.OperationRef]
		if op == nil {
			return ExecutionPlan{}, ErrPackage
		}
		used[step.OperationRef] = true
		constraints, err := json.Marshal(struct {
			Operation  any            `json:"operation"`
			Step       any            `json:"step"`
			DataSHA256 string         `json:"data_sha256"`
			Worker     WorkerIdentity `json:"worker"`
		}{rawOperations[op.OperationID], rawSteps[index], verified.manifest.Data.SHA256, options.Worker})
		if err != nil {
			return ExecutionPlan{}, ErrPackage
		}
		leaf := ExecutionOperation{StepID: step.StepID, OperationID: op.OperationID, InvocationID: step.StepID, ConstraintsSHA256: hashBytes(constraints)}
		var selected binding.Binding
		if op.HasSourceBinding() {
			if len(op.Extensions) != 0 {
				return ExecutionPlan{}, ErrPackage
			}
			selected, err = operationBinding(op, verified.manifest.Sources)
			if err != nil {
				return ExecutionPlan{}, ErrPackage
			}
		} else {
			name, err := functionName(op)
			if err != nil {
				return ExecutionPlan{}, ErrPackage
			}
			var source *Source
			for i := range verified.manifest.Sources {
				if verified.manifest.Sources[i].Kind == RuntimeSourceKind {
					source = &verified.manifest.Sources[i]
				}
			}
			if source == nil {
				return ExecutionPlan{}, ErrPackage
			}
			revision, err := runtimeRevision(verified.files[source.Artifact.Path])
			if err != nil || revision != options.Worker.RuntimeRevision {
				return ExecutionPlan{}, ErrPackage
			}
			selected = binding.Binding{Source: binding.Source{ID: source.ID, Kind: source.Kind, SHA256: source.Artifact.SHA256}, SelectorKind: "id", SelectorValue: name}
			needsRuntime = true
		}
		resolution, err := resolver.Resolve(ctx, selected)
		if err != nil || resolution.Status != binding.Resolved || resolution.Shape == nil {
			return ExecutionPlan{}, ErrPackage
		}
		shape := resolution.Shape
		leaf.Source, leaf.Selector, leaf.Security, leaf.MetadataComplete = shape.Source, shape.Selector, shape.Security, shape.Complete
		if shape.Protocol == "http" {
			leaf.Kind = "http"
			leaf.Method = shape.Method
			leaf.Origin, err = fixedHTTP(*shape)
			leaf.PathTemplate = shape.Path
			if err != nil {
				return ExecutionPlan{}, err
			}
		} else if shape.Protocol == "fnct" && shape.Source.Kind == RuntimeSourceKind {
			leaf.Kind = "fnct"
		} else {
			return ExecutionPlan{}, ErrPackage
		}
		plan.Operations = append(plan.Operations, leaf)
	}
	if len(used) != len(operations) {
		return ExecutionPlan{}, ErrPackage
	}
	// Native function invocation semantics are owned by the selected runtime,
	// whose independently checked catalog remains explicit partial metadata.
	if needsRuntime {
		if options.RuntimeAdmission == nil {
			return ExecutionPlan{}, ErrPackage
		}
		request := RuntimeAdmissionRequest{RuntimeRevision: options.Worker.RuntimeRevision, WorkflowYAML: append([]byte(nil), verified.files[WorkflowPath]...), DataJSON: append([]byte(nil), verified.files[DataPath]...), Sources: map[string][]byte{}}
		for _, source := range verified.manifest.Sources {
			request.Sources[source.Artifact.Path] = append([]byte(nil), verified.files[source.Artifact.Path]...)
		}
		if options.RuntimeAdmission(ctx, request) != nil {
			if ctx.Err() != nil {
				return ExecutionPlan{}, ctx.Err()
			}
			return ExecutionPlan{}, ErrPackage
		}
	}
	plan.PlanSHA256 = plan.Digest()
	return plan, ctx.Err()
}

// CheckExecutionPlan reproduces constraints from the retained snapshot and
// selected worker. The trusted host separately checks grant custody/revocation.
func CheckExecutionPlan(ctx context.Context, verified VerifiedPackage, options ExecutionOptions, expected ExecutionPlan) error {
	actual, err := DeriveExecutionPlan(ctx, verified, options)
	if err != nil {
		return err
	}
	if !digestValid(expected.PlanSHA256) || expected.PlanSHA256 != expected.Digest() || !reflect.DeepEqual(actual, expected) {
		return ErrPackage
	}
	return nil
}
