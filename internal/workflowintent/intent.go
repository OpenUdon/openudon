package workflowintent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/uws/uws1"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const IntentPath = "workflows/intent.hcl"

type Intent struct {
	Source       string              `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI      string              `hcl:"openapi,optional" json:"openapi,omitempty"`
	ServerURL    string              `hcl:"server_url,optional" json:"server_url,omitempty"`
	Workflow     *WorkflowMeta       `hcl:"workflow,block" json:"workflow,omitempty"`
	ContentTrust *ContentTrustIntent `hcl:"content_trust,block" json:"contentTrust,omitempty"`
	Inputs       []*Input            `hcl:"input,block" json:"inputs,omitempty"`
	Triggers     []*TriggerIntent    `hcl:"trigger,block" json:"triggers,omitempty"`
	Steps        []*Step             `hcl:"step,block" json:"steps,omitempty"`
	Security     []*SecurityIntent   `hcl:"security,block" json:"security,omitempty"`
	Outputs      []*Output           `hcl:"output,block" json:"outputs,omitempty"`
	Locals       map[string]string   `hcl:"locals,optional" json:"locals,omitempty"`
}

// ContentTrustIntent is the operator-authored form of UWS contentTrust.
// Source declarations use package-relative source paths; synthesis resolves
// those paths to the generated sourceDescription identifiers.
type ContentTrustIntent struct {
	SourceDescriptions []*SourceDescriptionContentTrustIntent `hcl:"source_description,block" json:"sourceDescriptions,omitempty"`
	Operations         []*OperationContentTrustIntent         `hcl:"operation,block" json:"operations,omitempty"`
	Triggers           []*TriggerContentTrustIntent           `hcl:"trigger,block" json:"triggers,omitempty"`
	Workflows          []*WorkflowContentTrustIntent          `hcl:"workflow,block" json:"workflows,omitempty"`
}

type SourceDescriptionContentTrustIntent struct {
	Source string `hcl:"source,label" json:"source"`
	Level  string `hcl:"level" json:"level"`
}

type OperationContentTrustIntent struct {
	Operation string            `hcl:"operation,label" json:"operation"`
	Default   string            `hcl:"default,optional" json:"default,omitempty"`
	Outputs   map[string]string `hcl:"outputs,optional" json:"outputs,omitempty"`
}

type TriggerContentTrustIntent struct {
	Trigger string `hcl:"trigger,label" json:"trigger"`
	Level   string `hcl:"level" json:"level"`
}

type WorkflowContentTrustIntent struct {
	Workflow string            `hcl:"workflow,label" json:"workflow"`
	Default  string            `hcl:"default,optional" json:"default,omitempty"`
	Inputs   map[string]string `hcl:"inputs,optional" json:"inputs,omitempty"`
}

type WorkflowMeta struct {
	Name        string            `hcl:"name,optional" json:"name,omitempty"`
	Description string            `hcl:"description,optional" json:"description,omitempty"`
	Timeout     *float64          `hcl:"timeout,optional" json:"timeout,omitempty"`
	Idempotency *uws1.Idempotency `hcl:"idempotency,block" json:"idempotency,omitempty"`
}

type Input struct {
	Name        string `hcl:"name,label" json:"name,omitempty"`
	Type        string `hcl:"type,optional" json:"type,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
	Required    bool   `hcl:"required,optional" json:"required,omitempty"`
	Sensitive   bool   `hcl:"sensitive,optional" json:"sensitive,omitempty"`
	Default     string `hcl:"default,optional" json:"default,omitempty"`
}

type Step struct {
	Name                 string                `hcl:"name,label" json:"name,omitempty"`
	Type                 string                `hcl:"type,optional" json:"type,omitempty"`
	Do                   string                `hcl:"do,optional" json:"do,omitempty"`
	Using                string                `hcl:"using,optional" json:"using,omitempty"`
	Set                  string                `hcl:"set,optional" json:"set,omitempty"`
	When                 string                `hcl:"when,optional" json:"when,omitempty"`
	ForEach              string                `hcl:"for_each,optional" json:"for_each,omitempty"`
	DependsOn            []string              `hcl:"depends_on,optional" json:"depends_on,omitempty"`
	With                 map[string]string     `hcl:"with,optional" json:"with,omitempty"`
	Provider             string                `hcl:"provider,optional" json:"provider,omitempty"`
	Source               string                `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI              string                `hcl:"openapi,optional" json:"openapi,omitempty"`
	Operation            string                `hcl:"operation,optional" json:"operation,omitempty"`
	AuthenticationFlow   string                `hcl:"authentication_flow,optional" json:"authentication_flow,omitempty"`
	RegistrationFlow     string                `hcl:"registration_flow,optional" json:"registration_flow,omitempty"`
	RegistrationApproval string                `hcl:"registration_approval,optional" json:"registration_approval,omitempty"`
	DuplicatePrevention  string                `hcl:"duplicate_prevention,optional" json:"duplicate_prevention,omitempty"`
	OnDuplicate          string                `hcl:"on_duplicate,optional" json:"on_duplicate,omitempty"`
	AmbiguousOutcome     string                `hcl:"ambiguous_outcome,optional" json:"ambiguous_outcome,omitempty"`
	CleanupDisposition   string                `hcl:"cleanup_disposition,optional" json:"cleanup_disposition,omitempty"`
	BrowserSession       string                `hcl:"browser_session,optional" json:"browser_session,omitempty"`
	CredentialBindings   map[string]string     `hcl:"credential_bindings,optional" json:"credential_bindings,omitempty"`
	Timeout              *float64              `hcl:"timeout,optional" json:"timeout,omitempty"`
	Binds                []*StepBind           `hcl:"bind,block" json:"bind,omitempty"`
	Items                string                `hcl:"items,optional" json:"items,omitempty"`
	Mode                 string                `hcl:"mode,optional" json:"mode,omitempty"`
	BatchSize            string                `hcl:"batch_size,optional" json:"batch_size,omitempty"`
	SuccessCriteria      []*uws1.Criterion     `hcl:"successCriteria,block" json:"successCriteria,omitempty"`
	OnFailure            []*uws1.FailureAction `hcl:"onFailure,block" json:"onFailure,omitempty"`
	OnSuccess            []*uws1.SuccessAction `hcl:"onSuccess,block" json:"onSuccess,omitempty"`
	Steps                []*Step               `hcl:"step,block" json:"steps,omitempty"`
	Cases                []*StepCase           `hcl:"case,block" json:"cases,omitempty"`
	Default              *StepDefault          `hcl:"default,block" json:"default,omitempty"`
}

func (s *Step) UnmarshalJSON(data []byte) error {
	type stepAlias Step
	var raw struct {
		*stepAlias
		With json.RawMessage `json:"with,omitempty"`
	}
	raw.stepAlias = (*stepAlias)(s)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.With) > 0 {
		with, err := decodeStringMapOrPairs(raw.With, "with")
		if err != nil {
			return err
		}
		s.With = with
	}
	return nil
}

type StepBind struct {
	From   string            `hcl:"from" json:"from,omitempty"`
	Fields map[string]string `hcl:"fields,optional" json:"fields,omitempty"`
}

func (b *StepBind) UnmarshalJSON(data []byte) error {
	type bindAlias StepBind
	var raw struct {
		*bindAlias
		Fields json.RawMessage `json:"fields,omitempty"`
	}
	raw.bindAlias = (*bindAlias)(b)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Fields) > 0 {
		fields, err := decodeStringMapOrPairs(raw.Fields, "bind.fields")
		if err != nil {
			return err
		}
		b.Fields = fields
	}
	return nil
}

func decodeStringMapOrPairs(data []byte, fieldName string) (map[string]string, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var object map[string]string
	if err := json.Unmarshal(data, &object); err == nil {
		if len(object) == 0 {
			return nil, nil
		}
		return object, nil
	}
	var pairs []struct {
		Field  string `json:"field"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(data, &pairs); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", fieldName, err)
	}
	out := map[string]string{}
	for _, pair := range pairs {
		field := strings.TrimSpace(pair.Field)
		source := strings.TrimSpace(pair.Source)
		if field == "" || source == "" {
			continue
		}
		out[field] = source
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

type StepCase struct {
	Name  string  `hcl:"name,label" json:"name,omitempty"`
	When  string  `hcl:"when,optional" json:"when,omitempty"`
	Steps []*Step `hcl:"step,block" json:"steps,omitempty"`
}

type StepDefault struct {
	Steps []*Step `hcl:"step,block" json:"steps,omitempty"`
}

type TriggerIntent struct {
	Name           string                `hcl:"name,label" json:"name,omitempty"`
	Path           string                `hcl:"path,optional" json:"path,omitempty"`
	Authentication string                `hcl:"authentication,optional" json:"authentication,omitempty"`
	Methods        []string              `hcl:"methods,optional" json:"methods,omitempty"`
	Options        map[string]string     `hcl:"options,optional" json:"options,omitempty"`
	Outputs        []string              `hcl:"outputs,optional" json:"outputs,omitempty"`
	Routes         []*TriggerRouteIntent `hcl:"route,block" json:"routes,omitempty"`
}

type TriggerRouteIntent struct {
	Output string   `hcl:"output,label" json:"output,omitempty"`
	To     []string `hcl:"to,optional" json:"to,omitempty"`
}

type SecurityIntent struct {
	Name        string `hcl:"name,label" json:"name,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
	TokenFrom   string `hcl:"token_from,optional" json:"token_from,omitempty"`
}

type Output struct {
	Name        string `hcl:"name,label" json:"name,omitempty"`
	From        string `hcl:"from" json:"from,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
}

func ParseIntentFile(path string) (*Intent, error) {
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return ParseIntent(data, path)
}

func ParseIntent(data []byte, path string) (*Intent, error) {
	if strings.TrimSpace(path) == "" {
		path = IntentPath
	}
	parser := hclparse.NewParser()
	rewritten, insertedLines := rewriteIntentHCLCompatibility(data)
	file, diags := parser.ParseHCL(rewritten, path)
	remapIntentHCLDiagnostics(diags, insertedLines)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding HCL: %s", diags.Error())
	}
	var raw hclIntent
	diags = gohcl.DecodeBody(file.Body, nil, &raw)
	remapIntentHCLDiagnostics(diags, insertedLines)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding HCL: %s", diags.Error())
	}
	intent, err := raw.toIntent()
	if err != nil {
		return nil, err
	}
	if err := validateIntent(&intent); err != nil {
		return nil, err
	}
	return &intent, nil
}

type hclIntent struct {
	Source       string              `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI      string              `hcl:"openapi,optional" json:"openapi,omitempty"`
	ServerURL    string              `hcl:"server_url,optional" json:"server_url,omitempty"`
	Workflow     *hclWorkflowMeta    `hcl:"workflow,block" json:"workflow,omitempty"`
	ContentTrust *ContentTrustIntent `hcl:"content_trust,block" json:"contentTrust,omitempty"`
	Inputs       []*Input            `hcl:"input,block" json:"inputs,omitempty"`
	Triggers     []*TriggerIntent    `hcl:"trigger,block" json:"triggers,omitempty"`
	Steps        []*hclStep          `hcl:"step,block" json:"steps,omitempty"`
	Security     []*SecurityIntent   `hcl:"security,block" json:"security,omitempty"`
	Outputs      []*Output           `hcl:"output,block" json:"outputs,omitempty"`
	Locals       map[string]string   `hcl:"locals,optional" json:"locals,omitempty"`
}

type hclWorkflowMeta struct {
	Name        string          `hcl:"name,optional" json:"name,omitempty"`
	Description string          `hcl:"description,optional" json:"description,omitempty"`
	Timeout     *float64        `hcl:"timeout,optional" json:"timeout,omitempty"`
	Idempotency *hclIdempotency `hcl:"idempotency,block" json:"idempotency,omitempty"`
}

type hclIdempotency struct {
	Key        string   `hcl:"key" json:"key,omitempty"`
	OnConflict string   `hcl:"onConflict,optional" json:"onConflict,omitempty"`
	TTL        *float64 `hcl:"ttl,optional" json:"ttl,omitempty"`
}

type hclStep struct {
	Name                 string              `hcl:"name,label" json:"name,omitempty"`
	Type                 string              `hcl:"type,optional" json:"type,omitempty"`
	Do                   string              `hcl:"do,optional" json:"do,omitempty"`
	Using                string              `hcl:"using,optional" json:"using,omitempty"`
	Set                  string              `hcl:"set,optional" json:"set,omitempty"`
	When                 string              `hcl:"when,optional" json:"when,omitempty"`
	ForEach              string              `hcl:"for_each,optional" json:"for_each,omitempty"`
	DependsOn            []string            `hcl:"depends_on,optional" json:"depends_on,omitempty"`
	With                 map[string]string   `hcl:"with,optional" json:"with,omitempty"`
	Provider             string              `hcl:"provider,optional" json:"provider,omitempty"`
	Source               string              `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI              string              `hcl:"openapi,optional" json:"openapi,omitempty"`
	Operation            string              `hcl:"operation,optional" json:"operation,omitempty"`
	AuthenticationFlow   string              `hcl:"authentication_flow,optional" json:"authentication_flow,omitempty"`
	RegistrationFlow     string              `hcl:"registration_flow,optional" json:"registration_flow,omitempty"`
	RegistrationApproval string              `hcl:"registration_approval,optional" json:"registration_approval,omitempty"`
	DuplicatePrevention  string              `hcl:"duplicate_prevention,optional" json:"duplicate_prevention,omitempty"`
	OnDuplicate          string              `hcl:"on_duplicate,optional" json:"on_duplicate,omitempty"`
	AmbiguousOutcome     string              `hcl:"ambiguous_outcome,optional" json:"ambiguous_outcome,omitempty"`
	CleanupDisposition   string              `hcl:"cleanup_disposition,optional" json:"cleanup_disposition,omitempty"`
	BrowserSession       string              `hcl:"browser_session,optional" json:"browser_session,omitempty"`
	CredentialBindings   map[string]string   `hcl:"credential_bindings,optional" json:"credential_bindings,omitempty"`
	Timeout              *float64            `hcl:"timeout,optional" json:"timeout,omitempty"`
	Binds                []*StepBind         `hcl:"bind,block" json:"bind,omitempty"`
	Items                string              `hcl:"items,optional" json:"items,omitempty"`
	Mode                 string              `hcl:"mode,optional" json:"mode,omitempty"`
	BatchSize            string              `hcl:"batch_size,optional" json:"batch_size,omitempty"`
	SuccessCriteria      []*hclCriterion     `hcl:"successCriteria,block" json:"successCriteria,omitempty"`
	OnFailure            []*hclFailureAction `hcl:"onFailure,block" json:"onFailure,omitempty"`
	OnSuccess            []*hclSuccessAction `hcl:"onSuccess,block" json:"onSuccess,omitempty"`
	Steps                []*hclStep          `hcl:"step,block" json:"steps,omitempty"`
	Cases                []*hclStepCase      `hcl:"case,block" json:"cases,omitempty"`
	Default              *hclStepDefault     `hcl:"default,block" json:"default,omitempty"`
}

type hclStepCase struct {
	Name  string     `hcl:"name,label" json:"name,omitempty"`
	When  string     `hcl:"when,optional" json:"when,omitempty"`
	Steps []*hclStep `hcl:"step,block" json:"steps,omitempty"`
}

type hclStepDefault struct {
	Steps []*hclStep `hcl:"step,block" json:"steps,omitempty"`
}

type hclCriterion struct {
	Condition string `hcl:"condition" json:"condition,omitempty"`
	Type      string `hcl:"type,optional" json:"type,omitempty"`
	Context   string `hcl:"context,optional" json:"context,omitempty"`
}

type hclFailureAction struct {
	Name       string          `hcl:"name,label" json:"name,omitempty"`
	Type       string          `hcl:"type" json:"type,omitempty"`
	WorkflowID string          `hcl:"workflowId,optional" json:"workflowId,omitempty"`
	StepID     string          `hcl:"stepId,optional" json:"stepId,omitempty"`
	RetryAfter float64         `hcl:"retryAfter,optional" json:"retryAfter,omitempty"`
	RetryLimit int             `hcl:"retryLimit,optional" json:"retryLimit,omitempty"`
	Criteria   []*hclCriterion `hcl:"criterion,block" json:"criteria,omitempty"`
}

type hclSuccessAction struct {
	Name       string          `hcl:"name,label" json:"name,omitempty"`
	Type       string          `hcl:"type" json:"type,omitempty"`
	WorkflowID string          `hcl:"workflowId,optional" json:"workflowId,omitempty"`
	StepID     string          `hcl:"stepId,optional" json:"stepId,omitempty"`
	Criteria   []*hclCriterion `hcl:"criterion,block" json:"criteria,omitempty"`
}

func (raw hclIntent) toIntent() (Intent, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return Intent{}, err
	}
	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return Intent{}, err
	}
	return intent, nil
}

func validateIntent(intent *Intent) error {
	if intent == nil {
		return fmt.Errorf("intent is required")
	}
	if intent.Workflow != nil {
		if err := validateTimeout(intent.Workflow.Timeout, "workflow.timeout"); err != nil {
			return err
		}
		if idem := intent.Workflow.Idempotency; idem != nil {
			if strings.TrimSpace(idem.Key) == "" {
				return fmt.Errorf("workflow.idempotency.key is required")
			}
			switch idem.OnConflict {
			case "", "reject", "returnPrevious":
			default:
				return fmt.Errorf("workflow.idempotency.onConflict must be reject or returnPrevious")
			}
			if idem.TTL != nil && *idem.TTL <= 0 {
				return fmt.Errorf("workflow.idempotency.ttl must be positive")
			}
		}
	}
	if len(intent.Steps) == 0 && len(intent.Triggers) == 0 {
		return fmt.Errorf("at least one step or trigger is required")
	}
	for i, step := range intent.Steps {
		if err := validateStep(step, fmt.Sprintf("step %d", i)); err != nil {
			return err
		}
	}
	for i, trigger := range intent.Triggers {
		if trigger == nil {
			continue
		}
		if strings.TrimSpace(trigger.Name) == "" {
			return fmt.Errorf("trigger %d: name label is required", i)
		}
		for routeIndex, route := range trigger.Routes {
			if route != nil && strings.TrimSpace(route.Output) == "" {
				return fmt.Errorf("trigger %d (%s) route %d: output label is required", i, trigger.Name, routeIndex)
			}
		}
	}
	if err := validateContentTrustIntent(intent); err != nil {
		return err
	}
	return nil
}

func validateContentTrustIntent(intent *Intent) error {
	trust := intent.ContentTrust
	if trust == nil {
		return nil
	}
	if len(trust.SourceDescriptions) == 0 && len(trust.Operations) == 0 && len(trust.Triggers) == 0 && len(trust.Workflows) == 0 {
		return fmt.Errorf("content_trust must contain at least one declaration")
	}
	sources := intentSourceReferences(intent)
	operations := map[string]bool{}
	walkSteps(intent.Steps, func(step *Step) {
		if step != nil && !isStructuralIntentStep(step) {
			operations[strings.TrimSpace(step.Name)] = true
		}
	})
	triggers := map[string]bool{}
	for _, trigger := range intent.Triggers {
		if trigger != nil {
			triggers[strings.TrimSpace(trigger.Name)] = true
		}
	}
	inputs := map[string]bool{}
	for _, input := range intent.Inputs {
		if input != nil {
			inputs[strings.TrimSpace(input.Name)] = true
		}
	}

	seen := map[string]bool{}
	for i, declaration := range trust.SourceDescriptions {
		path := fmt.Sprintf("content_trust.source_description %d", i)
		if declaration == nil {
			return fmt.Errorf("%s is required", path)
		}
		source := filepathSlash(strings.TrimSpace(declaration.Source))
		if source == "" {
			return fmt.Errorf("%s source label is required", path)
		}
		if seen[source] {
			return fmt.Errorf("content_trust has duplicate source_description %q", source)
		}
		seen[source] = true
		if !sources[source] {
			return fmt.Errorf("content_trust.source_description %q references an undeclared source", source)
		}
		if err := validateContentTrustLevel(declaration.Level, path+".level"); err != nil {
			return err
		}
	}
	seen = map[string]bool{}
	for i, declaration := range trust.Operations {
		path := fmt.Sprintf("content_trust.operation %d", i)
		if declaration == nil {
			return fmt.Errorf("%s is required", path)
		}
		operation := strings.TrimSpace(declaration.Operation)
		if operation == "" {
			return fmt.Errorf("%s operation label is required", path)
		}
		if seen[operation] {
			return fmt.Errorf("content_trust has duplicate operation %q", operation)
		}
		seen[operation] = true
		if !operations[operation] {
			return fmt.Errorf("content_trust.operation %q references an undeclared leaf step", operation)
		}
		if strings.TrimSpace(declaration.Default) == "" && len(declaration.Outputs) == 0 {
			return fmt.Errorf("content_trust.operation %q must declare default or outputs", operation)
		}
		if declaration.Default != "" {
			if err := validateContentTrustLevel(declaration.Default, path+".default"); err != nil {
				return err
			}
		}
		if err := validateContentTrustLevelMap(declaration.Outputs, path+".outputs"); err != nil {
			return err
		}
	}
	seen = map[string]bool{}
	for i, declaration := range trust.Triggers {
		path := fmt.Sprintf("content_trust.trigger %d", i)
		if declaration == nil {
			return fmt.Errorf("%s is required", path)
		}
		trigger := strings.TrimSpace(declaration.Trigger)
		if trigger == "" {
			return fmt.Errorf("%s trigger label is required", path)
		}
		if seen[trigger] {
			return fmt.Errorf("content_trust has duplicate trigger %q", trigger)
		}
		seen[trigger] = true
		if !triggers[trigger] {
			return fmt.Errorf("content_trust.trigger %q references an undeclared trigger", trigger)
		}
		if err := validateContentTrustLevel(declaration.Level, path+".level"); err != nil {
			return err
		}
	}
	seen = map[string]bool{}
	for i, declaration := range trust.Workflows {
		path := fmt.Sprintf("content_trust.workflow %d", i)
		if declaration == nil {
			return fmt.Errorf("%s is required", path)
		}
		workflow := strings.TrimSpace(declaration.Workflow)
		if workflow == "" {
			return fmt.Errorf("%s workflow label is required", path)
		}
		if workflow != "main" {
			return fmt.Errorf("content_trust.workflow %q references an undeclared workflow (only main is generated)", workflow)
		}
		if seen[workflow] {
			return fmt.Errorf("content_trust has duplicate workflow %q", workflow)
		}
		seen[workflow] = true
		if strings.TrimSpace(declaration.Default) == "" && len(declaration.Inputs) == 0 {
			return fmt.Errorf("content_trust.workflow %q must declare default or inputs", workflow)
		}
		if declaration.Default != "" {
			if err := validateContentTrustLevel(declaration.Default, path+".default"); err != nil {
				return err
			}
		}
		for input := range declaration.Inputs {
			if !inputs[strings.TrimSpace(input)] {
				return fmt.Errorf("content_trust.workflow %q references undeclared input %q", workflow, input)
			}
		}
		if err := validateContentTrustLevelMap(declaration.Inputs, path+".inputs"); err != nil {
			return err
		}
	}
	return nil
}

func isStructuralIntentStep(step *Step) bool {
	if step == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(step.Type)) {
	case "sequence", "parallel", "switch", "merge", "loop", "await":
		return true
	default:
		return false
	}
}

func intentSourceReferences(intent *Intent) map[string]bool {
	result := map[string]bool{}
	add := func(value string) {
		if value = filepathSlash(strings.TrimSpace(value)); value != "" {
			result[value] = true
		}
	}
	add(intent.Source)
	add(intent.OpenAPI)
	walkSteps(intent.Steps, func(step *Step) {
		if step != nil {
			add(step.Source)
			add(step.OpenAPI)
		}
	})
	return result
}

func filepathSlash(value string) string {
	return strings.ReplaceAll(value, `\`, "/")
}

func validateContentTrustLevelMap(values map[string]string, path string) error {
	for key, level := range values {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("%s keys must be non-empty", path)
		}
		if err := validateContentTrustLevel(level, path+"."+key); err != nil {
			return err
		}
	}
	return nil
}

func validateContentTrustLevel(level, path string) error {
	switch strings.TrimSpace(level) {
	case string(uws1.ContentTrustUnknown), string(uws1.ContentTrustTrusted), string(uws1.ContentTrustUntrusted):
		return nil
	default:
		return fmt.Errorf("%s must be unknown, trusted, or untrusted", path)
	}
}

func validateStep(step *Step, label string) error {
	if step == nil {
		return nil
	}
	if strings.TrimSpace(step.Name) == "" {
		return fmt.Errorf("%s: name label is required", label)
	}
	if err := validateTimeout(step.Timeout, label+".timeout"); err != nil {
		return err
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind == "browser" || kind == "browser_authentication" || kind == "browser_registration" {
		if !browserBindingPattern.MatchString(strings.TrimSpace(step.Name)) {
			return fmt.Errorf("%s name must be a portable browser runtime identifier", label)
		}
	}
	if kind == "browser_authentication" {
		if strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI)) == "" {
			return fmt.Errorf("%s.source is required for browser authentication", label)
		}
		if strings.TrimSpace(step.AuthenticationFlow) == "" {
			return fmt.Errorf("%s.authentication_flow is required for browser authentication", label)
		}
		if strings.TrimSpace(step.BrowserSession) == "" {
			return fmt.Errorf("%s.browser_session is required for browser authentication", label)
		}
		if step.Timeout == nil || *step.Timeout > 600 {
			return fmt.Errorf("%s.timeout must be set and no greater than 600 seconds for browser authentication", label)
		}
		if len(step.With) != 0 {
			return fmt.Errorf("%s.with is not supported for browser authentication", label)
		}
		for slot, binding := range step.CredentialBindings {
			if !browserBindingPattern.MatchString(strings.TrimSpace(slot)) || !browserBindingPattern.MatchString(strings.TrimSpace(binding)) || authoring.ContainsLikelyCredentialValue([]byte(binding)) {
				return fmt.Errorf("%s.credential_bindings must map symbolic slot and binding names", label)
			}
		}
		if hasBrowserRegistrationFields(step) {
			return fmt.Errorf("%s carries browser-registration-only fields", label)
		}
	} else if kind == "browser_registration" {
		if strings.TrimSpace(step.Source) == "" {
			return fmt.Errorf("%s.source is required for browser registration", label)
		}
		if strings.TrimSpace(step.OpenAPI) != "" {
			return fmt.Errorf("%s.openapi is not supported for browser registration", label)
		}
		if !browserBindingPattern.MatchString(strings.TrimSpace(step.RegistrationFlow)) {
			return fmt.Errorf("%s.registration_flow must be a portable symbolic name", label)
		}
		if !browserBindingPattern.MatchString(strings.TrimSpace(step.RegistrationApproval)) {
			return fmt.Errorf("%s.registration_approval must be a portable symbolic name", label)
		}
		if step.DuplicatePrevention != "operator_attestation" || step.OnDuplicate != "fail" || step.AmbiguousOutcome != "stop_without_retry" {
			return fmt.Errorf("%s registration duplicate and ambiguity policy must be operator_attestation, fail, and stop_without_retry", label)
		}
		if step.CleanupDisposition != "delete_separately" && step.CleanupDisposition != "retain_dedicated_test_identity" {
			return fmt.Errorf("%s.cleanup_disposition must be delete_separately or retain_dedicated_test_identity", label)
		}
		if step.Timeout == nil || *step.Timeout > 600 {
			return fmt.Errorf("%s.timeout must be set and no greater than 600 seconds for browser registration", label)
		}
		if strings.TrimSpace(step.BrowserSession) != "" {
			return fmt.Errorf("%s.browser_session is not supported for browser registration", label)
		}
		if strings.TrimSpace(step.AuthenticationFlow) != "" || strings.TrimSpace(step.Operation) != "" || len(step.With) != 0 {
			return fmt.Errorf("%s carries fields not supported for browser registration", label)
		}
		if len(step.CredentialBindings) == 0 {
			return fmt.Errorf("%s.credential_bindings is required for browser registration", label)
		}
		for slot, binding := range step.CredentialBindings {
			if !browserBindingPattern.MatchString(strings.TrimSpace(slot)) || !browserBindingPattern.MatchString(strings.TrimSpace(binding)) || authoring.ContainsLikelyCredentialValue([]byte(binding)) {
				return fmt.Errorf("%s.credential_bindings must map symbolic slot and binding names", label)
			}
		}
	} else if kind == "browser" {
		if strings.TrimSpace(step.AuthenticationFlow) != "" || len(step.CredentialBindings) != 0 {
			return fmt.Errorf("%s carries browser-authentication-only fields", label)
		}
		if hasBrowserRegistrationFields(step) {
			return fmt.Errorf("%s carries browser-registration-only fields", label)
		}
	} else if strings.TrimSpace(step.AuthenticationFlow) != "" || strings.TrimSpace(step.BrowserSession) != "" || len(step.CredentialBindings) != 0 || hasBrowserRegistrationFields(step) {
		return fmt.Errorf("%s carries browser-only fields on type %q", label, kind)
	}
	for i, nested := range step.Steps {
		if err := validateStep(nested, fmt.Sprintf("%s.step %d", label, i)); err != nil {
			return err
		}
	}
	for i, branch := range step.Cases {
		if branch == nil {
			continue
		}
		if strings.TrimSpace(branch.Name) == "" {
			return fmt.Errorf("%s.case %d: name label is required", label, i)
		}
		for j, nested := range branch.Steps {
			if err := validateStep(nested, fmt.Sprintf("%s.case %s.step %d", label, branch.Name, j)); err != nil {
				return err
			}
		}
	}
	if step.Default != nil {
		for i, nested := range step.Default.Steps {
			if err := validateStep(nested, fmt.Sprintf("%s.default.step %d", label, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func hasBrowserRegistrationFields(step *Step) bool {
	return step != nil && (strings.TrimSpace(step.RegistrationFlow) != "" ||
		strings.TrimSpace(step.RegistrationApproval) != "" ||
		strings.TrimSpace(step.DuplicatePrevention) != "" ||
		strings.TrimSpace(step.OnDuplicate) != "" ||
		strings.TrimSpace(step.AmbiguousOutcome) != "" ||
		strings.TrimSpace(step.CleanupDisposition) != "")
}

func validateTimeout(value *float64, path string) error {
	if value != nil && *value <= 0 {
		return fmt.Errorf("%s must be positive", path)
	}
	return nil
}

var labelBindPattern = regexp.MustCompile(`(?m)^([ \t]*)bind\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_-]*))\s*\{\s*$`)
var idempotencyAttrPattern = regexp.MustCompile(`(?m)^([ \t]*)idempotency\s*=\s*\{\s*$`)
var browserBindingPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func rewriteIntentHCLCompatibility(data []byte) ([]byte, []int) {
	rewritten, insertedLines := rewriteLabelBindSyntax(data)
	return rewriteIdempotencyAttributeSyntax(rewritten), insertedLines
}

func rewriteLabelBindSyntax(data []byte) ([]byte, []int) {
	input := string(data)
	if !strings.Contains(input, "bind ") {
		return data, nil
	}
	insertedLines := []int{}
	inserted := 0
	searchOffset := 0
	rewritten := labelBindPattern.ReplaceAllStringFunc(input, func(line string) string {
		match := labelBindPattern.FindStringSubmatch(line)
		if len(match) < 4 {
			return line
		}
		indent := match[1]
		label := strings.TrimSpace(match[2])
		if label == "" {
			label = strings.TrimSpace(match[3])
		}
		if label == "" {
			return line
		}
		relativeOffset := strings.Index(input[searchOffset:], line)
		if relativeOffset < 0 {
			return line
		}
		lineOffset := searchOffset + relativeOffset
		searchOffset = lineOffset + len(line)
		originalLine := strings.Count(input[:lineOffset], "\n") + 1
		inserted++
		insertedLines = append(insertedLines, originalLine+inserted)
		return fmt.Sprintf("%sbind {\n%s  from = %q", indent, indent, label)
	})
	return []byte(rewritten), insertedLines
}

func remapIntentHCLDiagnostics(diags hcl.Diagnostics, insertedLines []int) {
	mapRange := func(value *hcl.Range) {
		if value == nil {
			return
		}
		value.Start.Line = originalIntentHCLLine(value.Start.Line, insertedLines)
		value.End.Line = originalIntentHCLLine(value.End.Line, insertedLines)
	}
	for _, diagnostic := range diags {
		if diagnostic != nil {
			mapRange(diagnostic.Subject)
			mapRange(diagnostic.Context)
		}
	}
}

func originalIntentHCLLine(line int, insertedLines []int) int {
	transformedLine := line
	for _, insertedLine := range insertedLines {
		if transformedLine >= insertedLine {
			line--
		}
	}
	return line
}

func rewriteIdempotencyAttributeSyntax(data []byte) []byte {
	input := string(data)
	if !strings.Contains(input, "idempotency") {
		return data
	}
	return []byte(idempotencyAttrPattern.ReplaceAllString(input, `${1}idempotency {`))
}

func RenderIntentHCL(intent *Intent) (string, error) {
	if intent == nil {
		return "", fmt.Errorf("intent is required")
	}
	file := hclwrite.NewEmptyFile()
	body := file.Body()
	if strings.TrimSpace(intent.Source) != "" {
		setAttrString(body, "source", intent.Source)
	} else {
		setAttrString(body, "openapi", intent.OpenAPI)
	}
	setAttrString(body, "server_url", intent.ServerURL)
	if len(intent.Locals) > 0 {
		setAttrMap(body, "locals", intent.Locals, true)
	}
	if intent.Workflow != nil {
		block := body.AppendNewBlock("workflow", nil)
		wb := block.Body()
		setAttrString(wb, "name", intent.Workflow.Name)
		setAttrString(wb, "description", intent.Workflow.Description)
		setAttrFloatPtr(wb, "timeout", intent.Workflow.Timeout)
		if intent.Workflow.Idempotency != nil {
			addIdempotencyBlock(wb, intent.Workflow.Idempotency)
		}
	}
	addContentTrustBlock(body, intent.ContentTrust)
	for _, input := range intent.Inputs {
		if input == nil {
			continue
		}
		block := body.AppendNewBlock("input", []string{input.Name})
		ib := block.Body()
		setAttrString(ib, "type", input.Type)
		setAttrString(ib, "description", input.Description)
		setAttrBool(ib, "required", input.Required)
		setAttrBool(ib, "sensitive", input.Sensitive)
		setAttrString(ib, "default", input.Default)
	}
	for _, trigger := range intent.Triggers {
		addTriggerBlock(body, trigger)
	}
	for _, step := range intent.Steps {
		addStepBlock(body, step)
	}
	for _, sec := range intent.Security {
		if sec == nil {
			continue
		}
		block := body.AppendNewBlock("security", []string{sec.Name})
		sb := block.Body()
		setAttrString(sb, "description", sec.Description)
		setAttrString(sb, "token_from", sec.TokenFrom)
	}
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		block := body.AppendNewBlock("output", []string{output.Name})
		ob := block.Body()
		setAttrString(ob, "from", output.From)
		setAttrString(ob, "description", output.Description)
	}
	data := hclwrite.Format(file.Bytes())
	if _, err := ParseIntent(data, IntentPath); err != nil {
		return "", err
	}
	return string(data), nil
}

func addContentTrustBlock(body *hclwrite.Body, trust *ContentTrustIntent) {
	if trust == nil {
		return
	}
	block := body.AppendNewBlock("content_trust", nil)
	b := block.Body()
	for _, declaration := range trust.SourceDescriptions {
		if declaration == nil {
			continue
		}
		db := b.AppendNewBlock("source_description", []string{declaration.Source}).Body()
		setAttrString(db, "level", declaration.Level)
	}
	for _, declaration := range trust.Operations {
		if declaration == nil {
			continue
		}
		db := b.AppendNewBlock("operation", []string{declaration.Operation}).Body()
		setAttrString(db, "default", declaration.Default)
		setAttrMap(db, "outputs", declaration.Outputs, true)
	}
	for _, declaration := range trust.Triggers {
		if declaration == nil {
			continue
		}
		db := b.AppendNewBlock("trigger", []string{declaration.Trigger}).Body()
		setAttrString(db, "level", declaration.Level)
	}
	for _, declaration := range trust.Workflows {
		if declaration == nil {
			continue
		}
		db := b.AppendNewBlock("workflow", []string{declaration.Workflow}).Body()
		setAttrString(db, "default", declaration.Default)
		setAttrMap(db, "inputs", declaration.Inputs, true)
	}
}

func addTriggerBlock(body *hclwrite.Body, trigger *TriggerIntent) {
	if trigger == nil {
		return
	}
	block := body.AppendNewBlock("trigger", []string{trigger.Name})
	tb := block.Body()
	setAttrString(tb, "path", trigger.Path)
	setAttrString(tb, "authentication", trigger.Authentication)
	setAttrList(tb, "methods", trigger.Methods)
	setAttrMap(tb, "options", trigger.Options, true)
	setAttrList(tb, "outputs", trigger.Outputs)
	for _, route := range trigger.Routes {
		if route == nil {
			continue
		}
		rb := tb.AppendNewBlock("route", []string{route.Output})
		setAttrList(rb.Body(), "to", route.To)
	}
}

func addStepBlock(body *hclwrite.Body, step *Step) {
	if step == nil {
		return
	}
	block := body.AppendNewBlock("step", []string{step.Name})
	sb := block.Body()
	setAttrString(sb, "type", step.Type)
	setAttrString(sb, "do", step.Do)
	setAttrString(sb, "using", step.Using)
	setAttrString(sb, "set", step.Set)
	setAttrString(sb, "when", step.When)
	setAttrString(sb, "for_each", step.ForEach)
	setAttrList(sb, "depends_on", step.DependsOn)
	setAttrMap(sb, "with", step.With, true)
	setAttrString(sb, "provider", step.Provider)
	if strings.TrimSpace(step.Source) != "" {
		setAttrString(sb, "source", step.Source)
	} else {
		setAttrString(sb, "openapi", step.OpenAPI)
	}
	setAttrString(sb, "operation", step.Operation)
	setAttrString(sb, "authentication_flow", step.AuthenticationFlow)
	setAttrString(sb, "registration_flow", step.RegistrationFlow)
	setAttrString(sb, "registration_approval", step.RegistrationApproval)
	setAttrString(sb, "duplicate_prevention", step.DuplicatePrevention)
	setAttrString(sb, "on_duplicate", step.OnDuplicate)
	setAttrString(sb, "ambiguous_outcome", step.AmbiguousOutcome)
	setAttrString(sb, "cleanup_disposition", step.CleanupDisposition)
	setAttrString(sb, "browser_session", step.BrowserSession)
	setAttrMap(sb, "credential_bindings", step.CredentialBindings, true)
	setAttrFloatPtr(sb, "timeout", step.Timeout)
	setAttrString(sb, "items", step.Items)
	setAttrString(sb, "mode", step.Mode)
	setAttrString(sb, "batch_size", step.BatchSize)
	for _, bind := range step.Binds {
		addBindBlock(sb, bind)
	}
	for _, criterion := range step.SuccessCriteria {
		if criterion != nil {
			gohcl.EncodeIntoBody(criterion, sb.AppendNewBlock("successCriteria", nil).Body())
		}
	}
	for _, action := range step.OnFailure {
		if action != nil {
			gohcl.EncodeIntoBody(action, sb.AppendNewBlock("onFailure", nil).Body())
		}
	}
	for _, action := range step.OnSuccess {
		if action != nil {
			gohcl.EncodeIntoBody(action, sb.AppendNewBlock("onSuccess", nil).Body())
		}
	}
	for _, nested := range step.Steps {
		addStepBlock(sb, nested)
	}
	for _, branch := range step.Cases {
		if branch == nil {
			continue
		}
		cb := sb.AppendNewBlock("case", []string{branch.Name})
		setAttrString(cb.Body(), "when", branch.When)
		for _, nested := range branch.Steps {
			addStepBlock(cb.Body(), nested)
		}
	}
	if step.Default != nil {
		db := sb.AppendNewBlock("default", nil)
		for _, nested := range step.Default.Steps {
			addStepBlock(db.Body(), nested)
		}
	}
}

func addBindBlock(body *hclwrite.Body, bind *StepBind) {
	if bind == nil {
		return
	}
	block := body.AppendNewBlock("bind", nil)
	bb := block.Body()
	setAttrString(bb, "from", bind.From)
	setAttrMap(bb, "fields", bind.Fields, true)
}

func addIdempotencyBlock(body *hclwrite.Body, idem *uws1.Idempotency) {
	block := body.AppendNewBlock("idempotency", nil)
	ib := block.Body()
	setAttrString(ib, "key", idem.Key)
	setAttrString(ib, "onConflict", idem.OnConflict)
	setAttrFloatPtr(ib, "ttl", idem.TTL)
}

func setAttrString(body *hclwrite.Body, name, value string) {
	if strings.TrimSpace(value) != "" {
		body.SetAttributeValue(name, cty.StringVal(value))
	}
}

func setAttrBool(body *hclwrite.Body, name string, value bool) {
	if value {
		body.SetAttributeValue(name, cty.BoolVal(value))
	}
}

func setAttrFloatPtr(body *hclwrite.Body, name string, value *float64) {
	if value != nil {
		body.SetAttributeValue(name, cty.NumberFloatVal(*value))
	}
}

func setAttrList(body *hclwrite.Body, name string, values []string) {
	var out []cty.Value
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, cty.StringVal(value))
		}
	}
	if len(out) > 0 {
		body.SetAttributeValue(name, cty.ListVal(out))
	}
}

func setAttrMap(body *hclwrite.Body, name string, values map[string]string, sortKeys bool) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return
	}
	if sortKeys {
		sort.Strings(keys)
	}
	out := make(map[string]cty.Value, len(keys))
	for _, key := range keys {
		out[key] = cty.StringVal(values[key])
	}
	body.SetAttributeValue(name, cty.ObjectVal(out))
}

func ValidateHCL(content string) error {
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL([]byte(content), "workflow.hcl")
	if diags.HasErrors() {
		return fmt.Errorf("HCL validation error: %s", diags.Error())
	}
	return nil
}

func (intent *Intent) MissingSlots() []string {
	var missing []string
	if intent.missingDefaultOpenAPIContext() {
		missing = append(missing, "API source document URL or content")
	}
	if len(intent.Steps) == 0 && len(intent.Triggers) == 0 {
		missing = append(missing, "At least one workflow step")
	}
	for i, step := range intent.Steps {
		if step != nil && stepRequiresDo(step) && step.Do == "" {
			missing = append(missing, fmt.Sprintf("Description for step %d", i+1))
		}
	}
	return missing
}

func (intent *Intent) RequiresOpenAPI() bool {
	if intent == nil {
		return false
	}
	if strings.TrimSpace(intent.Source) != "" || strings.TrimSpace(intent.OpenAPI) != "" {
		return true
	}
	required := false
	walkSteps(intent.Steps, func(step *Step) {
		if step != nil && !required && stepUsesAPISource(step) {
			required = true
		}
	})
	return required
}

func (intent *Intent) missingDefaultOpenAPIContext() bool {
	if intent == nil || strings.TrimSpace(intent.Source) != "" || strings.TrimSpace(intent.OpenAPI) != "" {
		return false
	}
	missing := false
	walkSteps(intent.Steps, func(step *Step) {
		if step != nil && !missing && stepUsesAPISource(step) && strings.TrimSpace(step.Source) == "" && strings.TrimSpace(step.OpenAPI) == "" {
			missing = true
		}
	})
	return missing
}

func stepUsesAPISource(step *Step) bool {
	if step == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind != "" && kind != "http" && kind != "openapi" && kind != "browser" && kind != "browser_authentication" && kind != "browser_registration" {
		return false
	}
	return strings.TrimSpace(step.Source) != "" || strings.TrimSpace(step.OpenAPI) != "" || strings.TrimSpace(step.Operation) != ""
}

func stepRequiresDo(step *Step) bool {
	if step == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(step.Type)) {
	case "sequence", "parallel", "switch", "merge", "loop", "await":
		return false
	default:
		return strings.TrimSpace(step.Operation) == ""
	}
}

func walkSteps(steps []*Step, fn func(*Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		fn(step)
		walkSteps(step.Steps, fn)
		for _, branch := range step.Cases {
			if branch != nil {
				walkSteps(branch.Steps, fn)
			}
		}
		if step.Default != nil {
			walkSteps(step.Default.Steps, fn)
		}
	}
}

func (intent *Intent) NormalizedForGeneration() (*Intent, error) {
	clone, err := intent.Clone()
	if err != nil || clone == nil {
		return clone, err
	}
	if strings.TrimSpace(clone.Source) != "" && strings.TrimSpace(clone.OpenAPI) == "" {
		clone.OpenAPI = strings.TrimSpace(clone.Source)
	}
	for _, step := range clone.Steps {
		normalizeStepForGeneration(step)
	}
	clone.EnsureActionDescriptions()
	return clone, nil
}

func (intent *Intent) EnsureActionDescriptions() {
	if intent == nil {
		return
	}
	ensureStepActionDescriptions(intent.Steps)
}

func ensureStepActionDescriptions(steps []*Step) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.Do) == "" {
			if op := strings.TrimSpace(step.Operation); op != "" {
				step.Do = "Run operation " + op + "."
			} else if typ := strings.TrimSpace(step.Type); typ != "" {
				step.Do = "Run " + typ + " step."
			}
		}
		ensureStepActionDescriptions(step.Steps)
		for _, branch := range step.Cases {
			if branch != nil {
				ensureStepActionDescriptions(branch.Steps)
			}
		}
		if step.Default != nil {
			ensureStepActionDescriptions(step.Default.Steps)
		}
	}
}

func (intent *Intent) ToPromptContext() string {
	var result string
	if intent == nil {
		return result
	}
	if intent.Workflow != nil {
		result += fmt.Sprintf("Workflow: %s\n", intent.Workflow.Name)
		if intent.Workflow.Description != "" {
			result += fmt.Sprintf("Description: %s\n", intent.Workflow.Description)
		}
		result += "\n"
	}
	if len(intent.Inputs) > 0 {
		result += "Inputs:\n"
		for _, input := range intent.Inputs {
			if input == nil {
				continue
			}
			req := ""
			if input.Required {
				req = " (required)"
			}
			result += fmt.Sprintf("  - %s: %s%s\n", input.Name, input.Type, req)
		}
		result += "\n"
	}
	result += "Steps:\n"
	for _, step := range intent.Steps {
		appendStepPrompt(&result, step, "  ")
	}
	if len(intent.Outputs) > 0 {
		result += "\nOutputs:\n"
		for _, out := range intent.Outputs {
			if out != nil {
				result += fmt.Sprintf("  - %s: from %s\n", out.Name, out.From)
			}
		}
	}
	return result
}

func appendStepPrompt(result *string, step *Step, indent string) {
	if step == nil {
		return
	}
	*result += fmt.Sprintf("%s- %s", indent, step.Name)
	if step.Type != "" {
		*result += fmt.Sprintf(" (%s)", step.Type)
	}
	if step.Do != "" {
		*result += fmt.Sprintf(": %s", step.Do)
	}
	*result += "\n"
	for _, nested := range step.Steps {
		appendStepPrompt(result, nested, indent+"  ")
	}
	for _, branch := range step.Cases {
		if branch != nil {
			for _, nested := range branch.Steps {
				appendStepPrompt(result, nested, indent+"  ")
			}
		}
	}
	if step.Default != nil {
		for _, nested := range step.Default.Steps {
			appendStepPrompt(result, nested, indent+"  ")
		}
	}
}

func (intent *Intent) Clone() (*Intent, error) {
	if intent == nil {
		return nil, nil
	}
	data, err := json.Marshal(intent)
	if err != nil {
		return nil, fmt.Errorf("clone intent: %w", err)
	}
	var clone Intent
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, fmt.Errorf("clone intent: %w", err)
	}
	return &clone, nil
}

func normalizeStepForGeneration(step *Step) {
	if step == nil {
		return
	}
	step.Type = normalizeIntentStepType(step.Type)
	if strings.TrimSpace(step.Source) != "" && strings.TrimSpace(step.OpenAPI) == "" {
		step.OpenAPI = strings.TrimSpace(step.Source)
	}
	applyStepBindHints(step)
	for _, nested := range step.Steps {
		normalizeStepForGeneration(nested)
	}
	for _, branch := range step.Cases {
		if branch != nil {
			for _, nested := range branch.Steps {
				normalizeStepForGeneration(nested)
			}
		}
	}
	if step.Default != nil {
		for _, nested := range step.Default.Steps {
			normalizeStepForGeneration(nested)
		}
	}
}

func normalizeIntentStepType(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "format", "formatter", "formatting", "process", "processing", "transform", "transformer", "mapping", "compose", "composition":
		return "fnct"
	default:
		return kind
	}
}

func applyStepBindHints(step *Step) {
	if step == nil || len(step.Binds) == 0 {
		return
	}
	if step.With == nil {
		step.With = map[string]string{}
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		from := strings.TrimSpace(bind.From)
		if from == "" {
			continue
		}
		step.DependsOn = appendUnique(step.DependsOn, from)
		keys := make([]string, 0, len(bind.Fields))
		for target := range bind.Fields {
			keys = append(keys, target)
		}
		sort.Strings(keys)
		for _, target := range keys {
			source := bind.Fields[target]
			target = normalizeRequestFieldTarget(target)
			if target == "" {
				continue
			}
			if _, exists := step.With[target]; !exists {
				step.With[target] = bindFieldReference(from, target, source)
			}
		}
	}
}

func normalizeRequestFieldTarget(target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "payload.") {
		return strings.TrimPrefix(target, "payload.")
	}
	return target
}

func bindFieldReference(from, target, source string) string {
	source = strings.TrimSpace(source)
	if source == "" || source == "received_body" {
		return from + ".received_body." + leafName(target)
	}
	if strings.HasPrefix(source, from+".") {
		return source
	}
	return from + "." + source
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.TrimSpace(existing) == value {
			return values
		}
	}
	return append(values, value)
}

func leafName(path string) string {
	path = strings.Trim(path, ".")
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
