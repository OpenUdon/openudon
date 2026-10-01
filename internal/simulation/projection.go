package simulation

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/OpenUdon/uws/mockruntime"
	"github.com/OpenUdon/uws/uws1"
)

var portableID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,127}$`)
var legacyOutput = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\.received_body(?:\.([A-Za-z0-9_.-]+))?$`)
var credentialReference = regexp.MustCompile(`\bcredentials\.([A-Za-z_][A-Za-z0-9_-]*)`)

type stepBinding struct {
	id, operation, effect string
	pending, browser      bool
}

type projection struct {
	document          *uws1.Document
	steps             []stepBinding
	pendingOperations map[string]string // synthetic operation -> original step
	definitions       map[string]ResponseDefinition
	fixtures          *mockruntime.FixtureSet
}

func project(document *uws1.Document, options Options) (projection, error) {
	result := projection{document: document, pendingOperations: map[string]string{}, definitions: map[string]ResponseDefinition{}}
	for id, definition := range options.Responses {
		result.definitions[id] = definition
	}
	occupied := map[string]bool{}
	for _, op := range document.Operations {
		if op != nil {
			if !portableID.MatchString(op.OperationID) {
				return result, errors.New("unsupported operation identity")
			}
			occupied[op.OperationID] = true
		}
	}
	for _, wf := range document.Workflows {
		if wf != nil {
			occupied[wf.WorkflowID] = true
		}
	}
	index := 0
	stepIDs := map[string]bool{}
	err := walkSteps(document, func(step *uws1.Step) error {
		if !portableID.MatchString(step.StepID) {
			return errors.New("unsupported step identity")
		}
		if stepIDs[step.StepID] {
			return errors.New("ambiguous step identity across workflows")
		}
		stepIDs[step.StepID] = true
		if len(stepIDs) > 256 {
			return errors.New("simulation step inventory exceeds bound")
		}
		occupied[step.StepID] = true
		return nil
	})
	if err != nil {
		return result, err
	}
	err = walkSteps(document, func(step *uws1.Step) error {
		binding := stepBinding{id: step.StepID, operation: step.OperationRef, effect: "unknown"}
		if step.Pending != nil {
			pending := step.Pending
			var id string
			for {
				id = fmt.Sprintf("__openudon_pending_%d", index)
				index++
				if !occupied[id] {
					break
				}
			}
			occupied[id] = true
			result.pendingOperations[id] = step.StepID
			definition, explicit := result.definitions[step.StepID]
			if !explicit {
				definition.Schema = &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{"body": pending.Outputs}}
			}
			result.definitions[id] = definition
			document.Operations = append(document.Operations, &uws1.Operation{OperationID: id, Effect: pending.Effect, Extensions: map[string]any{uws1.ExtensionOperationProfile: "openudon.pending-simulation.1"}})
			step.OperationRef = id
			step.Pending = nil
			binding.operation = id
			binding.pending = true
			binding.effect = string(pending.Effect)
		}
		if step.OperationRef != "" {
			// Private legacy received_body bindings are converted to the public
			// outputs vocabulary only in this projection, with evaluation delegated.
			if step.Outputs == nil {
				step.Outputs = map[string]string{}
			}
			if _, exists := step.Outputs["received_body"]; !exists {
				step.Outputs["received_body"] = "$response.body"
			}
			for _, op := range document.Operations {
				if op != nil && op.OperationID == step.OperationRef {
					if op.Effect != "" {
						binding.effect = string(op.Effect)
					}
					binding.browser = strings.HasPrefix(op.ExtensionProfile(), "uws.browser")
				}
			}
		}
		result.steps = append(result.steps, binding)
		return nil
	})
	if err != nil {
		return result, err
	}
	if len(result.steps) > 256 {
		return result, errors.New("simulation step inventory exceeds bound")
	}
	// Adapt expression wrappers only in operation requests. Variable values and
	// response schemas/examples remain literal data, never expression authority.
	for _, op := range document.Operations {
		if op == nil || op.Request == nil {
			continue
		}
		adapted, err := adaptValues(op.Request, 0)
		if err != nil {
			return result, err
		}
		request, ok := adapted.(map[string]any)
		if !ok {
			return result, errors.New("operation request must remain an object")
		}
		op.Request = request
	}
	adaptControlFields(document)
	if document.Variables == nil {
		document.Variables = map[string]any{}
		if document.Components != nil {
			for k, v := range document.Components.Variables {
				document.Variables[k] = v
			}
		}
	}
	if options.Inputs != nil {
		document.Variables["inputs"] = options.Inputs
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return result, err
	}
	credentials := map[string]any{}
	for _, match := range credentialReference.FindAllSubmatch(encoded, -1) {
		credentials[string(match[1])] = "<credential-reference>"
	}
	// Never resolve inline, environment or account credential values.
	document.Variables["credentials"] = credentials
	if document.Components != nil && document.Components.Variables != nil {
		document.Components.Variables["credentials"] = credentials
	}
	if options.Fixtures != nil {
		encoded, err := mockruntime.EncodeFixtures(options.Fixtures)
		if err != nil {
			return result, err
		}
		result.fixtures, err = mockruntime.DecodeFixtures(encoded)
		if err != nil {
			return result, err
		}
		for i := range result.fixtures.Fixtures {
			for synthetic, original := range result.pendingOperations {
				if result.fixtures.Fixtures[i].OperationID == original {
					result.fixtures.Fixtures[i].OperationID = synthetic
					break
				}
			}
		}
	}
	return result, nil
}

func walkSteps(doc *uws1.Document, visit func(*uws1.Step) error) error {
	var steps func([]*uws1.Step) error
	steps = func(list []*uws1.Step) error {
		for _, step := range list {
			if step == nil {
				continue
			}
			if err := visit(step); err != nil {
				return err
			}
			if err := steps(step.Steps); err != nil {
				return err
			}
			for _, branch := range step.Cases {
				if branch != nil {
					if err := steps(branch.Steps); err != nil {
						return err
					}
				}
			}
			if err := steps(step.Default); err != nil {
				return err
			}
		}
		return nil
	}
	for _, wf := range doc.Workflows {
		if wf == nil {
			continue
		}
		if err := steps(wf.Steps); err != nil {
			return err
		}
		for _, branch := range wf.Cases {
			if branch != nil {
				if err := steps(branch.Steps); err != nil {
					return err
				}
			}
		}
		if err := steps(wf.Default); err != nil {
			return err
		}
	}
	return nil
}

func adaptValues(value any, depth int) (any, error) {
	if depth > 64 {
		return nil, errors.New("simulation expression projection exceeds depth")
	}
	switch v := value.(type) {
	case map[string]any:
		if expression, ok := v["$expr"]; ok && len(v) == 1 {
			text, ok := expression.(string)
			if !ok {
				return nil, errors.New("invalid expression wrapper")
			}
			return adaptExpression(text), nil
		}
		for k, child := range v {
			var err error
			v[k], err = adaptValues(child, depth+1)
			if err != nil {
				return nil, err
			}
		}
	case []any:
		for i, child := range v {
			var err error
			v[i], err = adaptValues(child, depth+1)
			if err != nil {
				return nil, err
			}
		}
	}
	return value, nil
}

func adaptExpression(text string) string {
	if strings.HasPrefix(text, "variables.") {
		return "$" + text
	}
	if strings.HasPrefix(text, "inputs.") {
		return "$variables." + text
	}
	if strings.HasPrefix(text, "credentials.") {
		return "$variables." + text
	}
	if match := legacyOutput.FindStringSubmatch(text); match != nil {
		result := "$steps." + match[1] + ".outputs.received_body"
		if match[2] != "" {
			result += "." + match[2]
		}
		return result
	}
	return text
}

// Only expression-bearing fields are adapted; literal request string values
// and schemas are left intact. The public mock runtime evaluates each result.
func adaptControlFields(document *uws1.Document) {
	outputs := func(values map[string]string) {
		for k, v := range values {
			values[k] = adaptExpression(v)
		}
	}
	for _, op := range document.Operations {
		if op == nil {
			continue
		}
		op.When = adaptExpression(op.When)
		op.ForEach = adaptExpression(op.ForEach)
		outputs(op.Outputs)
		for _, criterion := range op.SuccessCriteria {
			if criterion != nil {
				criterion.Condition = adaptExpression(criterion.Condition)
			}
		}
	}
	walkSteps(document, func(step *uws1.Step) error {
		step.When = adaptExpression(step.When)
		step.ForEach = adaptExpression(step.ForEach)
		step.Items = adaptExpression(step.Items)
		outputs(step.Outputs)
		for _, branch := range step.Cases {
			if branch != nil {
				branch.When = adaptExpression(branch.When)
			}
		}
		return nil
	})
	for _, wf := range document.Workflows {
		if wf == nil {
			continue
		}
		wf.When = adaptExpression(wf.When)
		wf.ForEach = adaptExpression(wf.ForEach)
		wf.Items = adaptExpression(wf.Items)
		outputs(wf.Outputs)
		for _, branch := range wf.Cases {
			if branch != nil {
				branch.When = adaptExpression(branch.When)
			}
		}
	}
}
