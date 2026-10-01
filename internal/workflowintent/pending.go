package workflowintent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OpenUdon/uws/uws1"
)

// Intent HCL stores schemas as JSON strings so the public schema model keeps
// every nested construct and extension, without a second HCL schema dialect.
type hclPending struct {
	Purpose string `hcl:"purpose"`
	Inputs  string `hcl:"inputs"`
	Outputs string `hcl:"outputs"`
	Effect  string `hcl:"effect"`
}

func (p hclPending) MarshalJSON() ([]byte, error) {
	var inputs, outputs uws1.ParamSchema
	if json.Unmarshal([]byte(p.Inputs), &inputs) != nil || json.Unmarshal([]byte(p.Outputs), &outputs) != nil {
		return nil, fmt.Errorf("pending schemas must be public JSON field sets")
	}
	return json.Marshal(uws1.PendingStep{Purpose: p.Purpose, Inputs: &inputs, Outputs: &outputs, Effect: uws1.OperationEffect(p.Effect)})
}

func validatePendingIntent(step *Step) error {
	if step.Pending == nil || step.Type != "pending" {
		return fmt.Errorf("pending intent requires type pending and its contract")
	}
	if len(step.Pending.Extensions) != 0 {
		return fmt.Errorf("intent pending extensions are unsupported; retain them in the source UWS document")
	}
	// An unresolved contract has no executable binding or structural children.
	if step.Operation != "" || step.Source != "" || step.OpenAPI != "" || step.Provider != "" || step.Using != "" || step.Set != "" || len(step.With) != 0 || len(step.Binds) != 0 || len(step.Steps) != 0 || len(step.Cases) != 0 || step.Default != nil || step.Items != "" || step.Mode != "" || step.BatchSize != "" || len(step.SuccessCriteria) != 0 || len(step.OnFailure) != 0 || len(step.OnSuccess) != 0 || step.Effect != "" || step.When != "" || step.ForEach != "" || step.Timeout != nil || step.AuthenticationFlow != "" || step.BrowserSession != "" || len(step.CredentialBindings) != 0 || hasBrowserRegistrationFields(step) {
		return fmt.Errorf("pending intent cannot carry executable or structural fields")
	}
	doc := &uws1.Document{UWS: "1.12.0", Info: &uws1.Info{Title: "Pending contract", Version: "1.0.0"}, Workflows: []*uws1.Workflow{{WorkflowID: "main", Type: "sequence", Steps: []*uws1.Step{{StepID: step.Name, Pending: step.Pending}}}}}
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("pending contract fails public UWS validation")
	}
	return nil
}

func validIntentEffect(effect string) bool {
	return effect == "" || effect == "read" || effect == "write" || effect == "unknown"
}

func isPendingIntent(step *Step) bool {
	return step != nil && (step.Pending != nil || strings.EqualFold(step.Type, "pending"))
}

// OnlyPendingSteps allows publishing unresolved contracts before API sources
// exist. Structural parents are allowed; any bound leaf keeps normal discovery.
func (intent *Intent) OnlyPendingSteps() bool {
	if intent == nil {
		return false
	}
	count, all := 0, true
	walkSteps(intent.Steps, func(step *Step) {
		if step == nil || isStructuralIntentStep(step) {
			return
		}
		count++
		if step.Pending == nil {
			all = false
		}
	})
	return count > 0 && all
}
