package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/schemas"
	"github.com/OpenUdon/uws/uws1"
	"github.com/OpenUdon/uws/validation"
)

// BrowserScenarioWorkflowRequest is a narrow deterministic synthesis seam for
// the release scenario suite. All source files must already be staged beneath
// ExampleDir by the normal authenticated-authoring import path.
type BrowserScenarioWorkflowRequest struct {
	ExampleDir             string
	AuthenticationPath     string
	CapabilityPath         string
	AuthenticationFlow     string
	CapabilityAction       string
	Session                string
	CredentialSlotBindings map[string]string
	Inputs                 []BrowserScenarioInput
	Actions                []BrowserScenarioAction
}

type BrowserScenarioInput struct {
	Name     string
	Type     string
	Required bool
}

type BrowserScenarioAction struct {
	Name      string
	Operation string
	With      map[string]string
}

type BrowserScenarioWorkflowResult struct {
	Path       string
	UWSVersion string
	Bindings   []string
}

func WriteBrowserScenarioWorkflow(request BrowserScenarioWorkflowRequest) (BrowserScenarioWorkflowResult, error) {
	root, err := filepath.Abs(request.ExampleDir)
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	authentication, err := scenarioRelativeSource(root, request.AuthenticationPath)
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	capability, err := scenarioRelativeSource(root, request.CapabilityPath)
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	if strings.TrimSpace(request.AuthenticationFlow) == "" || strings.TrimSpace(request.Session) == "" || (strings.TrimSpace(request.CapabilityAction) == "" && len(request.Actions) == 0) {
		return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario workflow identity is incomplete")
	}
	bindings := make(map[string]string, len(request.CredentialSlotBindings))
	var bindingNames []string
	for slot, binding := range request.CredentialSlotBindings {
		if strings.TrimSpace(slot) == "" || strings.TrimSpace(binding) == "" {
			return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario credential binding is invalid")
		}
		bindings[slot] = binding
		bindingNames = append(bindingNames, binding)
	}
	sort.Strings(bindingNames)
	timeout := 120.0
	inputs := make([]*rollout.Input, 0, len(request.Inputs))
	seenInputs := map[string]bool{}
	for _, input := range request.Inputs {
		if input.Name != strings.TrimSpace(input.Name) || input.Type != strings.TrimSpace(input.Type) || input.Name == "" || input.Type == "" || seenInputs[input.Name] {
			return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario workflow input is invalid")
		}
		seenInputs[input.Name] = true
		inputs = append(inputs, &rollout.Input{Name: input.Name, Type: input.Type, Required: input.Required})
	}
	actions := append([]BrowserScenarioAction(nil), request.Actions...)
	if len(actions) == 0 {
		actions = []BrowserScenarioAction{{Name: "read", Operation: request.CapabilityAction}}
	}
	if err := validateBrowserScenarioActions(root, capability, actions); err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	steps := []*rollout.Step{{Name: "authenticate", Type: "browser_authentication", Source: authentication, AuthenticationFlow: request.AuthenticationFlow, BrowserSession: request.Session, CredentialBindings: bindings, Timeout: &timeout}}
	previous := "authenticate"
	stepNames := make([]string, 0, len(actions))
	seenSteps := map[string]bool{"authenticate": true}
	for _, action := range actions {
		if action.Name != strings.TrimSpace(action.Name) || action.Operation != strings.TrimSpace(action.Operation) || action.Name == "" || action.Operation == "" || seenSteps[action.Name] {
			return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario action is invalid")
		}
		seenSteps[action.Name] = true
		with := make(map[string]string, len(action.With))
		for parameter, input := range action.With {
			if parameter != strings.TrimSpace(parameter) || input != strings.TrimSpace(input) || parameter == "" || input == "" || !seenInputs[input] {
				return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario action input binding is invalid")
			}
			with["body."+parameter] = "inputs." + input
		}
		steps = append(steps, &rollout.Step{Name: action.Name, Type: "browser", Source: capability, Operation: action.Operation, BrowserSession: request.Session, DependsOn: []string{previous}, With: with})
		stepNames = append(stepNames, action.Name)
		previous = action.Name
	}
	finalStep := stepNames[len(stepNames)-1]
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "browser_scenario", Description: "Deterministic browser scenario replay."},
		Inputs:   inputs, Steps: steps,
		Outputs: []*rollout.Output{{Name: "scenario_result", From: finalStep + ".received_body"}},
	}
	if err := validateIntentRequiredParameters(intent, root, nil, ""); err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	document, err := generateWorkflowDocument(Result{ExampleDir: root}, intent)
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	if err := wireBrowserScenarioOutputs(document, stepNames); err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	normalizeUWSStepsForSchema(document)
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	data = append(data, '\n')
	path := filepath.Join(root, "workflow.uws.json")
	if err := atomicfile.Write(path, data, 0o600); err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	if err := validation.ValidateFile(schemas.PathForVersion(root, document.UWS), path); err != nil {
		return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario UWS synthesis: %w", err)
	}
	return BrowserScenarioWorkflowResult{Path: path, UWSVersion: document.UWS, Bindings: bindingNames}, nil
}

func validateBrowserScenarioActions(root, capability string, actions []BrowserScenarioAction) error {
	value, err := loadBrowserProfile(filepath.Join(root, filepath.FromSlash(capability)))
	if err != nil {
		return err
	}
	for _, action := range actions {
		contract, ok := value.Actions[action.Operation]
		if !ok {
			return fmt.Errorf("browser scenario action %q is not declared", action.Operation)
		}
		properties, _ := contract.Parameters["properties"].(map[string]any)
		for parameter := range action.With {
			if _, ok := properties[parameter]; !ok {
				return fmt.Errorf("browser scenario action %q parameter %q is not declared", action.Operation, parameter)
			}
		}
		for _, parameter := range browserScenarioRequiredParameters(contract.Parameters["required"]) {
			if _, ok := action.With[parameter]; !ok {
				return fmt.Errorf("browser scenario action %q required parameter %q is not bound", action.Operation, parameter)
			}
		}
	}
	return nil
}

func browserScenarioRequiredParameters(raw any) []string {
	values, _ := raw.([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if name, ok := value.(string); ok {
			result = append(result, name)
		}
	}
	return result
}

func wireBrowserScenarioOutputs(document *uws1.Document, requestedStepNames ...[]string) error {
	if document == nil || len(document.Workflows) != 1 || document.Workflows[0] == nil {
		return fmt.Errorf("browser scenario workflow is missing")
	}
	stepNames := []string{"read"}
	if len(requestedStepNames) == 1 {
		stepNames = requestedStepNames[0]
	} else if len(requestedStepNames) > 1 {
		return fmt.Errorf("browser scenario action selection is invalid")
	}
	want := make(map[string]bool, len(stepNames))
	for _, name := range stepNames {
		want[name] = true
	}
	operationFound, stepFound := 0, 0
	for _, operation := range document.Operations {
		if operation != nil && want[operation.OperationID] {
			operation.Outputs = map[string]string{"received_body": "$response.body"}
			operationFound++
		}
	}
	for _, step := range document.Workflows[0].Steps {
		if step != nil && want[step.StepID] && step.OperationRef == step.StepID {
			step.Outputs = map[string]string{"received_body": "$response.body"}
			stepFound++
		}
	}
	if operationFound != len(stepNames) || stepFound != len(stepNames) {
		return fmt.Errorf("browser scenario action operation is missing")
	}
	finalStep := stepNames[len(stepNames)-1]
	document.Workflows[0].Outputs = map[string]string{
		"scenario_result": "$steps." + finalStep + ".outputs.received_body",
	}
	return nil
}

func scenarioRelativeSource(root, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, abs)
	if err != nil || relative == "." || relative == "" || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("browser scenario source is outside its example")
	}
	info, err := os.Lstat(abs)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("browser scenario source is not a regular staged file")
	}
	return filepath.ToSlash(relative), nil
}
