package workflowintent

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/uws/uws1"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const IntentPath = "workflows/intent.hcl"

type Intent struct {
	Source    string            `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI   string            `hcl:"openapi,optional" json:"openapi,omitempty"`
	ServerURL string            `hcl:"server_url,optional" json:"server_url,omitempty"`
	Workflow  *WorkflowMeta     `hcl:"workflow,block" json:"workflow,omitempty"`
	Inputs    []*Input          `hcl:"input,block" json:"inputs,omitempty"`
	Triggers  []*TriggerIntent  `hcl:"trigger,block" json:"triggers,omitempty"`
	Steps     []*Step           `hcl:"step,block" json:"steps,omitempty"`
	Security  []*SecurityIntent `hcl:"security,block" json:"security,omitempty"`
	Outputs   []*Output         `hcl:"output,block" json:"outputs,omitempty"`
	Locals    map[string]string `hcl:"locals,optional" json:"locals,omitempty"`
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
	Name            string                `hcl:"name,label" json:"name,omitempty"`
	Type            string                `hcl:"type,optional" json:"type,omitempty"`
	Do              string                `hcl:"do,optional" json:"do,omitempty"`
	Using           string                `hcl:"using,optional" json:"using,omitempty"`
	Set             string                `hcl:"set,optional" json:"set,omitempty"`
	When            string                `hcl:"when,optional" json:"when,omitempty"`
	ForEach         string                `hcl:"for_each,optional" json:"for_each,omitempty"`
	DependsOn       []string              `hcl:"depends_on,optional" json:"depends_on,omitempty"`
	With            map[string]string     `hcl:"with,optional" json:"with,omitempty"`
	Provider        string                `hcl:"provider,optional" json:"provider,omitempty"`
	Source          string                `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI         string                `hcl:"openapi,optional" json:"openapi,omitempty"`
	Operation       string                `hcl:"operation,optional" json:"operation,omitempty"`
	Timeout         *float64              `hcl:"timeout,optional" json:"timeout,omitempty"`
	Binds           []*StepBind           `hcl:"bind,block" json:"bind,omitempty"`
	Items           string                `hcl:"items,optional" json:"items,omitempty"`
	Mode            string                `hcl:"mode,optional" json:"mode,omitempty"`
	BatchSize       string                `hcl:"batch_size,optional" json:"batch_size,omitempty"`
	SuccessCriteria []*uws1.Criterion     `hcl:"successCriteria,block" json:"successCriteria,omitempty"`
	OnFailure       []*uws1.FailureAction `hcl:"onFailure,block" json:"onFailure,omitempty"`
	OnSuccess       []*uws1.SuccessAction `hcl:"onSuccess,block" json:"onSuccess,omitempty"`
	Steps           []*Step               `hcl:"step,block" json:"steps,omitempty"`
	Cases           []*StepCase           `hcl:"case,block" json:"cases,omitempty"`
	Default         *StepDefault          `hcl:"default,block" json:"default,omitempty"`
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
	data, err := os.ReadFile(path)
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
	file, diags := parser.ParseHCL(rewriteIntentHCLCompatibility(data), path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding HCL: %s", diags.Error())
	}
	var raw hclIntent
	diags = gohcl.DecodeBody(file.Body, nil, &raw)
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
	Source    string            `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI   string            `hcl:"openapi,optional" json:"openapi,omitempty"`
	ServerURL string            `hcl:"server_url,optional" json:"server_url,omitempty"`
	Workflow  *hclWorkflowMeta  `hcl:"workflow,block" json:"workflow,omitempty"`
	Inputs    []*Input          `hcl:"input,block" json:"inputs,omitempty"`
	Triggers  []*TriggerIntent  `hcl:"trigger,block" json:"triggers,omitempty"`
	Steps     []*hclStep        `hcl:"step,block" json:"steps,omitempty"`
	Security  []*SecurityIntent `hcl:"security,block" json:"security,omitempty"`
	Outputs   []*Output         `hcl:"output,block" json:"outputs,omitempty"`
	Locals    map[string]string `hcl:"locals,optional" json:"locals,omitempty"`
	Remain    hcl.Body          `hcl:",remain" json:"-"`
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
	Name            string              `hcl:"name,label" json:"name,omitempty"`
	Type            string              `hcl:"type,optional" json:"type,omitempty"`
	Do              string              `hcl:"do,optional" json:"do,omitempty"`
	Using           string              `hcl:"using,optional" json:"using,omitempty"`
	Set             string              `hcl:"set,optional" json:"set,omitempty"`
	When            string              `hcl:"when,optional" json:"when,omitempty"`
	ForEach         string              `hcl:"for_each,optional" json:"for_each,omitempty"`
	DependsOn       []string            `hcl:"depends_on,optional" json:"depends_on,omitempty"`
	With            map[string]string   `hcl:"with,optional" json:"with,omitempty"`
	Provider        string              `hcl:"provider,optional" json:"provider,omitempty"`
	Source          string              `hcl:"source,optional" json:"source,omitempty"`
	OpenAPI         string              `hcl:"openapi,optional" json:"openapi,omitempty"`
	Operation       string              `hcl:"operation,optional" json:"operation,omitempty"`
	Timeout         *float64            `hcl:"timeout,optional" json:"timeout,omitempty"`
	Binds           []*StepBind         `hcl:"bind,block" json:"bind,omitempty"`
	Items           string              `hcl:"items,optional" json:"items,omitempty"`
	Mode            string              `hcl:"mode,optional" json:"mode,omitempty"`
	BatchSize       string              `hcl:"batch_size,optional" json:"batch_size,omitempty"`
	SuccessCriteria []*hclCriterion     `hcl:"successCriteria,block" json:"successCriteria,omitempty"`
	OnFailure       []*hclFailureAction `hcl:"onFailure,block" json:"onFailure,omitempty"`
	OnSuccess       []*hclSuccessAction `hcl:"onSuccess,block" json:"onSuccess,omitempty"`
	Steps           []*hclStep          `hcl:"step,block" json:"steps,omitempty"`
	Cases           []*hclStepCase      `hcl:"case,block" json:"cases,omitempty"`
	Default         *hclStepDefault     `hcl:"default,block" json:"default,omitempty"`
	Remain          hcl.Body            `hcl:",remain" json:"-"`
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
	return nil
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

func validateTimeout(value *float64, path string) error {
	if value != nil && *value <= 0 {
		return fmt.Errorf("%s must be positive", path)
	}
	return nil
}

var labelBindPattern = regexp.MustCompile(`(?m)^([ \t]*)bind\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_-]*))\s*\{\s*$`)
var idempotencyAttrPattern = regexp.MustCompile(`(?m)^([ \t]*)idempotency\s*=\s*\{\s*$`)

func rewriteIntentHCLCompatibility(data []byte) []byte {
	return rewriteIdempotencyAttributeSyntax(rewriteLabelBindSyntax(data))
}

func rewriteLabelBindSyntax(data []byte) []byte {
	input := string(data)
	if !strings.Contains(input, "bind ") {
		return data
	}
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
		return fmt.Sprintf("%sbind {\n%s  from = %q", indent, indent, label)
	})
	return []byte(rewritten)
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
	setAttrMap(sb, "with", step.With, false)
	setAttrString(sb, "provider", step.Provider)
	if strings.TrimSpace(step.Source) != "" {
		setAttrString(sb, "source", step.Source)
	} else {
		setAttrString(sb, "openapi", step.OpenAPI)
	}
	setAttrString(sb, "operation", step.Operation)
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
	setAttrMap(bb, "fields", bind.Fields, false)
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

func FormatHCL(content string) (string, error) {
	if err := ValidateHCL(content); err != nil {
		return "", err
	}
	return string(hclwrite.Format([]byte(content))), nil
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
	if kind != "" && kind != "http" && kind != "openapi" {
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

func (intent *Intent) NormalizedForGeneration() *Intent {
	clone := intent.Clone()
	if clone == nil {
		return nil
	}
	if strings.TrimSpace(clone.Source) != "" && strings.TrimSpace(clone.OpenAPI) == "" {
		clone.OpenAPI = strings.TrimSpace(clone.Source)
	}
	for _, step := range clone.Steps {
		normalizeStepForGeneration(step)
	}
	clone.EnsureActionDescriptions()
	return clone
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

func (intent *Intent) Clone() *Intent {
	if intent == nil {
		return nil
	}
	data, _ := json.Marshal(intent)
	var clone Intent
	_ = json.Unmarshal(data, &clone)
	return &clone
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
		for target, source := range bind.Fields {
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

var _ = hcl.DiagError
