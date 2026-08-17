package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	if strings.TrimSpace(request.AuthenticationFlow) == "" || strings.TrimSpace(request.CapabilityAction) == "" || strings.TrimSpace(request.Session) == "" {
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
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "browser_scenario", Description: "Deterministic browser scenario replay."},
		Steps: []*rollout.Step{
			{Name: "authenticate", Type: "browser_authentication", Source: authentication, AuthenticationFlow: request.AuthenticationFlow, BrowserSession: request.Session, CredentialBindings: bindings, Timeout: &timeout},
			{Name: "read", Type: "browser", Source: capability, Operation: request.CapabilityAction, BrowserSession: request.Session, DependsOn: []string{"authenticate"}},
		},
		Outputs: []*rollout.Output{{Name: "scenario_result", From: "read.received_body"}},
	}
	document, err := generateWorkflowDocument(Result{ExampleDir: root}, intent)
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	if err := wireBrowserScenarioOutputs(document); err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	normalizeUWSStepsForSchema(document)
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	data = append(data, '\n')
	path := filepath.Join(root, "workflow.uws.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return BrowserScenarioWorkflowResult{}, err
	}
	if err := validation.ValidateFile(schemas.PathForVersion(root, document.UWS), path); err != nil {
		return BrowserScenarioWorkflowResult{}, fmt.Errorf("browser scenario UWS synthesis: %w", err)
	}
	return BrowserScenarioWorkflowResult{Path: path, UWSVersion: document.UWS, Bindings: bindingNames}, nil
}

func wireBrowserScenarioOutputs(document *uws1.Document) error {
	if document == nil || len(document.Workflows) != 1 || document.Workflows[0] == nil {
		return fmt.Errorf("browser scenario workflow is missing")
	}
	operationFound, stepFound := false, false
	for _, operation := range document.Operations {
		if operation != nil && operation.OperationID == "read" {
			operation.Outputs = map[string]string{"received_body": "$response.body"}
			operationFound = true
		}
	}
	for _, step := range document.Workflows[0].Steps {
		if step != nil && step.StepID == "read" && step.OperationRef == "read" {
			step.Outputs = map[string]string{"received_body": "$response.body"}
			stepFound = true
		}
	}
	if !operationFound || !stepFound {
		return fmt.Errorf("browser scenario read operation is missing")
	}
	document.Workflows[0].Outputs = map[string]string{
		"scenario_result": "$steps.read.outputs.received_body",
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
