package udonrunner

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/brokerhandoff"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/executablefile"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

func ValidConfigVersion(c Config) bool {
	return c.Version == RunConfigVersion && c.Broker == nil || c.Version == BrokerRunConfigVersion && c.Broker != nil
}

func validateBrokerConfig(c Config, opts Options) error {
	a := c.Broker
	if a == nil {
		if opts.BrokerConfigPath != "" {
			return errors.New("private broker reference requires explicit broker authority")
		}
		return nil
	}
	now := time.Now()
	if opts.Now != nil {
		now = opts.Now()
	}
	if a.ValidateAt(now) != nil || a.RunID != c.RunID || a.PackageSHA256 != c.PackageSHA256 || a.HandoffSHA256 != c.HandoffSHA256 || c.Browser != nil || c.ExecutorReportVersion != udonreport.VersionV5 || len(c.DataFiles) != 0 || (c.Tier != "sandbox" && c.Tier != "production") {
		return errors.New("broker config does not match current exact HTTP authority")
	}
	names := map[string]bool{}
	for _, op := range a.Operations {
		if c.Tier == "sandbox" && synthesize.ProductionEndpointURL(op.Origin) {
			return errors.New("sandbox broker authority contains a production destination")
		}
		for _, b := range op.Bindings {
			names[b.Name] = true
		}
	}
	var bindings []string
	for name := range names {
		bindings = append(bindings, name)
	}
	sort.Strings(bindings)
	declared := append([]string(nil), c.CredentialBindings...)
	sort.Strings(declared)
	if !reflect.DeepEqual(bindings, declared) {
		return errors.New("broker bindings differ from declared credentials")
	}
	_, err := brokerhandoff.ReadPrivate(opts.BrokerConfigPath, *a)
	return err
}

// BrokerPlan describes exact input and operation bytes, without granting a run
// or selecting current host credentials. The host adds revisions and concrete
// occurrence authority. Inputs are compiled into the reviewed workflow; broker
// mode accepts no external runtime data file.
type BrokerPlan struct {
	InputsSHA256 string                    `json:"inputs_sha256"`
	Operations   []brokerhandoff.Operation `json:"operations"`
	Requests     []BrokerRequest           `json:"requests"`
}

// BrokerRequest exposes reviewed source metadata and the exact canonical UWS
// operation/step constraints. The host applies its own finite policy to static
// inputs and explicitly approved dynamic inputs; this record grants no I/O.
type BrokerRequest struct {
	StepID            string          `json:"step_id"`
	SourceOperationID string          `json:"source_operation_id"`
	ServerURL         string          `json:"server_url"`
	PathTemplate      string          `json:"path_template"`
	Constraints       json.RawMessage `json:"constraints"`
}

// InspectBrokerPlan uses reviewed, digest-checked staged sources and the existing
// APItools adapter. It does not invoke an executor or make a network request.
func InspectBrokerPlan(stage, workflow, format string, apiPaths []string, inventory udonreport.InventoryV5) (BrokerPlan, error) {
	data, _, err := evidencefile.ReadRegular(workflow, evidencefile.DefaultMaxBytes)
	if err != nil {
		return BrokerPlan{}, err
	}
	var d uws1.Document
	switch format {
	case "uws-yaml", "yaml":
		err = convert.UnmarshalYAML(data, &d)
	case "uws-json", "json":
		err = convert.UnmarshalJSON(data, &d)
	case "uws-hcl", "hcl":
		err = convert.UnmarshalHCL(data, &d)
	default:
		return BrokerPlan{}, errors.New("unsupported broker workflow format")
	}
	if err != nil || len(d.Workflows) != 1 {
		return BrokerPlan{}, errors.New("invalid broker workflow")
	}
	allowed := map[string]bool{}
	for _, p := range apiPaths {
		allowed[filepath.ToSlash(p)] = true
	}
	specs := map[string]*workflowintent.OpenAPISpec{}
	for _, src := range d.SourceDescriptions {
		if src == nil || !allowed[src.URL] {
			return BrokerPlan{}, errors.New("broker source is outside reviewed API inventory")
		}
		spec, err := workflowintent.LoadOpenAPISpec(filepath.Join(stage, filepath.FromSlash(src.URL)))
		if err != nil {
			return BrokerPlan{}, errors.New("invalid reviewed broker source")
		}
		specs[src.Name] = spec
	}
	operations := map[string]*uws1.Operation{}
	for _, op := range d.Operations {
		if op != nil {
			operations[op.OperationID] = op
		}
	}
	plan := BrokerPlan{InputsSHA256: evidencefile.SHA256(data)}
	for i, step := range inventory.Steps {
		op := operations[step.OperationID]
		if op == nil || i >= len(d.Workflows[0].Steps) {
			return BrokerPlan{}, errors.New("broker operation inventory mismatch")
		}
		spec := specs[op.SourceDescription]
		if spec == nil {
			return BrokerPlan{}, errors.New("missing broker API source")
		}
		sourceID := op.SourceOperationID
		if sourceID == "" {
			sourceID = op.OpenAPIOperationID
		}
		var apiOp *workflowintent.OperationInfo
		for _, candidate := range spec.Operations {
			if candidate.OperationID == sourceID {
				if apiOp != nil {
					return BrokerPlan{}, errors.New("ambiguous broker API operation")
				}
				apiOp = candidate
			}
		}
		origin, err := url.Parse(spec.ServerURL)
		if apiOp == nil || err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Hostname() == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || strings.Contains(spec.ServerURL, "{") {
			return BrokerPlan{}, errors.New("broker requires a fixed reviewed API server and operation ID")
		}
		// Server overrides are deliberately unsupported in this first profile.
		// The generic executor may support them; the host must never review one
		// origin while the worker selects a different one.
		if workflowintent.HasServerOverrides(spec) {
			return BrokerPlan{}, errors.New("broker API server overrides are unsupported")
		}
		constraints, err := json.Marshal(struct {
			Operation *uws1.Operation `json:"operation"`
			Step      *uws1.Step      `json:"step"`
		}{op, d.Workflows[0].Steps[i]})
		if err != nil {
			return BrokerPlan{}, err
		}
		reviewed := brokerhandoff.Operation{StepID: step.StepID, OperationID: step.OperationID, InvocationID: step.InvocationID, Method: strings.ToUpper(apiOp.Method), Origin: origin.Scheme + "://" + origin.Host, ConstraintsSHA256: evidencefile.SHA256(constraints)}
		if len(apiOp.SecurityRequirementSets) > 1 {
			return BrokerPlan{}, errors.New("broker requires one fixed security requirement")
		}
		if len(apiOp.SecurityRequirementSets) == 1 {
			for _, security := range apiOp.SecurityRequirementSets[0].Requirements {
				if len(security.Scopes) != 0 {
					return BrokerPlan{}, errors.New("scoped broker authentication is unsupported")
				}
				binding := brokerhandoff.Binding{Name: security.Name}
				switch {
				case security.Type == "apiKey":
					binding.Kind, binding.In, binding.Parameter = "api_key", security.In, security.ParameterName
				case security.Type == "http" && strings.EqualFold(security.Scheme, "bearer"):
					binding.Kind, binding.In, binding.Parameter = "bearer", "header", "Authorization"
				default:
					return BrokerPlan{}, errors.New("unsupported broker security scheme")
				}
				if binding.In == "header" {
					binding.Parameter = http.CanonicalHeaderKey(binding.Parameter)
				}
				reviewed.Bindings = append(reviewed.Bindings, binding)
			}
		}
		plan.Operations = append(plan.Operations, reviewed)
		plan.Requests = append(plan.Requests, BrokerRequest{StepID: step.StepID, SourceOperationID: sourceID, ServerURL: spec.ServerURL, PathTemplate: apiOp.Path, Constraints: constraints})
	}
	return plan, nil
}

func validateBrokerPlan(c Config, result Result) error {
	plan, err := InspectBrokerPlan(result.StagePath, result.WorkflowPath, c.WorkflowFormat, result.APISourcePaths, *result.InventoryV5)
	if err != nil {
		return err
	}
	return CheckBrokerPlan(plan, *c.Broker)
}

// CheckBrokerPlan binds concrete authority to the inspected exact package.
// The host must separately validate current grant and credential revisions.
func CheckBrokerPlan(plan BrokerPlan, authority brokerhandoff.Authority) error {
	if authority.Validate() != nil || plan.InputsSHA256 != authority.InputsSHA256 || len(plan.Operations) != len(authority.Operations) {
		return errors.New("broker reviewed input/inventory mismatch")
	}
	for i, expected := range plan.Operations {
		got := authority.Operations[i]
		if len(expected.Bindings) != len(got.Bindings) {
			return errors.New("broker credential scheme inventory mismatch")
		}
		for j := range expected.Bindings {
			expected.Bindings[j].Revision = got.Bindings[j].Revision
		} // Current revisions remain host-owned.
		if !reflect.DeepEqual(expected, got) {
			return errors.New("broker reviewed operation constraints mismatch")
		}
	}
	return nil
}

func brokerExecutor(c Config, env map[string]string, result Result, copyBinary bool) (string, error) {
	path := env["OPENUDON_EXECUTOR"]
	if path == "" {
		path = env["OPENUDON_UDON_BIN"]
	}
	if !filepath.IsAbs(path) || !executablefile.Is(path) {
		return "", errors.New("broker requires an explicit absolute pinned executor binary")
	}
	data, _, err := evidencefile.ReadRegular(path, 256<<20)
	if err != nil || evidencefile.SHA256(data) != c.Broker.ExecutorSHA256 {
		return "", errors.New("broker executor digest mismatch")
	}
	if !copyBinary {
		return path, nil
	}
	// Execute a private copy of the checked bytes, preventing a later mutation
	// of the selected path from selecting a different executor.
	snapshot := filepath.Join(result.StagePath, ".broker-executor")
	if err := atomicfile.WriteNew(snapshot, data, 0o500); err != nil {
		return "", err
	}
	return snapshot, nil
}

func claimBrokerAttempt(c Config) error {
	// Kept outside the package/staging tree and never removed on an uncertain
	// result. The host additionally claims across worker instances/workdirs.
	if err := os.MkdirAll(c.WorkDir, 0o700); err != nil {
		return err
	}
	return atomicfile.WriteNew(filepath.Join(c.WorkDir, ".broker-attempt-"+c.RunID+".json"), []byte(c.Broker.PolicySHA256+"\n"), 0o600)
}

func snapshotBrokerTransport(c Config, opts Options, stage string) (string, error) {
	private, err := brokerhandoff.ReadPrivate(opts.BrokerConfigPath, *c.Broker)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(private)
	if err != nil {
		return "", errors.New("cannot encode private broker transport")
	}
	path := filepath.Join(stage, ".broker-transport.json")
	if err := atomicfile.WriteNew(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
