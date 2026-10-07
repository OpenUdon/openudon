package packagev3

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/binding"
	"github.com/OpenUdon/uws/expressions"
	"github.com/OpenUdon/uws/uws1"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type expressionScope struct {
	workflow  *uws1.Workflow
	step      *uws1.Step
	operation *uws1.Operation
	index     int
}
type expressionContracts struct {
	ctx        context.Context
	doc        *uws1.Document
	data       map[string]any
	sources    []Source
	resolver   *binding.TableResolver
	operations map[string]*uws1.Operation
	scopes     map[string][]expressionScope
}

func newExpressionContracts(ctx context.Context, doc *uws1.Document, data map[string]any, sources []Source, resolver *binding.TableResolver) *expressionContracts {
	e := &expressionContracts{ctx: ctx, doc: doc, data: data, sources: sources, resolver: resolver, operations: map[string]*uws1.Operation{}, scopes: map[string][]expressionScope{}}
	for _, op := range doc.Operations {
		e.operations[op.OperationID] = op
	}
	for _, workflow := range doc.Workflows {
		for index, step := range workflow.Steps {
			if step != nil && step.OperationRef != "" {
				e.scopes[step.OperationRef] = append(e.scopes[step.OperationRef], expressionScope{workflow: workflow, step: step, operation: e.operations[step.OperationRef], index: index})
			}
		}
	}
	return e
}
func (e *expressionContracts) operationScope(op *uws1.Operation) expressionScope {
	scopes := e.scopes[op.OperationID]
	if len(scopes) == 1 {
		return scopes[0]
	}
	return expressionScope{operation: op, index: -1}
}
func (e *expressionContracts) shape(op *uws1.Operation) *binding.OperationShape {
	if op == nil || !op.HasSourceBinding() {
		return nil
	}
	b, err := operationBinding(op, e.sources)
	if err != nil {
		return nil
	}
	resolution, err := e.resolver.Resolve(e.ctx, b)
	if err != nil || resolution.Status != binding.Resolved {
		return nil
	}
	return resolution.Shape
}
func literalContract(value any) binding.Schema {
	// Runtime references in reviewed data are not literal guarantees. The shared
	// grammar validates syntax separately; no evaluator or environment is called.
	var dynamic func(any) bool
	dynamic = func(value any) bool {
		switch v := value.(type) {
		case string:
			return strings.HasPrefix(v, "$") || strings.HasPrefix(strings.TrimSpace(v), "expr(")
		case map[string]any:
			for _, child := range v {
				if dynamic(child) {
					return true
				}
			}
		case []any:
			for _, child := range v {
				if dynamic(child) {
					return true
				}
			}
		}
		return false
	}
	if dynamic(value) {
		return binding.Schema{}
	}
	data, err := json.Marshal(map[string]any{"const": value})
	if err != nil || len(data) > binding.MaxSchemaBytes {
		return binding.Schema{}
	}
	return binding.Schema{Known: true, JSON: data}
}
func lookupLiteral(value any, parts []string) (any, bool) {
	for _, part := range parts {
		switch v := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = v[part]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || strconv.Itoa(index) != part || index < 0 || index >= len(v) {
				return nil, false
			}
			value = v[index]
		default:
			return nil, false
		}
	}
	return value, true
}

// childContract projects only proved presence. Optional/open/array paths retain
// unknownness; constraints aren't erased to make an expression fit its target.
func childContract(schema binding.Schema, parts []string) binding.Schema {
	if !schema.Known {
		return binding.Schema{}
	}
	if len(parts) == 0 {
		return schema
	}
	var root any
	if wire.DecodeStrictNumbers(schema.JSON, &root) != nil {
		return binding.Schema{}
	}
	for index, part := range parts {
		object, ok := root.(map[string]any)
		if !ok {
			return binding.Schema{}
		}
		if constant, exists := object["const"]; exists {
			compiler := jsonschema.NewCompiler()
			compiler.UseLoader(closedSchemaLoader{})
			const uri = "https://openudon.invalid/const-source"
			if compiler.AddResource(uri, object) != nil {
				return binding.Schema{}
			}
			compiled, err := compiler.Compile(uri)
			if err != nil || compiled.Validate(constant) != nil {
				return binding.Schema{}
			}

			value, ok := lookupLiteral(constant, parts[index:])
			if !ok {
				return binding.Schema{Known: true, JSON: json.RawMessage(`false`)}
			}
			return literalContract(value)
		}
		// A child schema alone cannot represent additional restricting parent
		// keywords. Preserve unknownness instead of erasing those restrictions.
		for keyword := range object {
			switch keyword {
			case "type", "properties", "required", "additionalProperties":
			default:
				return binding.Schema{}
			}
		}
		kind, ok := object["type"].(string)
		if !ok || kind != "object" {
			return binding.Schema{}
		}
		properties, ok := object["properties"].(map[string]any)
		if !ok {
			return binding.Schema{}
		}
		child, exists := properties[part]
		if !exists {
			if object["additionalProperties"] == false {
				return binding.Schema{Known: true, JSON: json.RawMessage(`false`)}
			}
			return binding.Schema{}
		}
		required := false
		for _, name := range listValues(object["required"]) {
			required = required || name == part
		}
		if !required {
			return binding.Schema{}
		}
		root = child
	}
	data, err := json.Marshal(root)
	if err != nil {
		return binding.Schema{}
	}
	return binding.Schema{Known: true, JSON: data}
}
func listValues(value any) []string {
	items, _ := value.([]any)
	out := []string{}
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func responseReference(source string) (binding.OutputReference, bool) {
	switch {
	case source == "$response.body":
		return binding.OutputReference{Location: "body", Name: "body"}, true
	case strings.HasPrefix(source, "$response.body#"):
		return binding.OutputReference{Location: "body", Name: "body", Pointer: strings.TrimPrefix(source, "$response.body")}, true
	case strings.HasPrefix(source, "$response.body."):
		parts := strings.Split(strings.TrimPrefix(source, "$response.body."), ".")
		for i := range parts {
			parts[i] = strings.ReplaceAll(strings.ReplaceAll(parts[i], "~", "~0"), "/", "~1")
		}
		return binding.OutputReference{Location: "body", Name: "body", Pointer: "#/" + strings.Join(parts, "/")}, true
	case strings.HasPrefix(source, "$response.headers."):
		return binding.OutputReference{Location: "header", Name: strings.TrimPrefix(source, "$response.headers.")}, true
	}
	return binding.OutputReference{}, false
}
func (e *expressionContracts) contract(text string, scope expressionScope, active map[string]bool, depth int) (binding.Schema, string) {
	if depth > 64 || e.ctx.Err() != nil {
		return binding.Schema{}, "indeterminate"
	}
	parsed, err := expressions.Parse(text, expressions.Context{Version: e.doc.UWS, Field: expressions.Value, InLoop: true})
	if err != nil {
		return binding.Schema{}, "incompatible"
	}
	if parsed.Operator() != "" {
		return binding.Schema{}, "indeterminate"
	}
	source := parsed.Source()
	switch {
	case strings.HasPrefix(source, "$variables."):
		parts := strings.Split(strings.TrimPrefix(source, "$variables."), ".")
		name := parts[0]
		value, exists := e.doc.Variables[name]
		if overlay, ok := e.data[name]; ok {
			value, exists = overlay, true
		}
		// The implementing legacy lowerer and reference evaluator have differing
		// collision precedence. A collision cannot prove one runtime value here.
		if e.doc.Components != nil {
			if component, ok := e.doc.Components.Variables[name]; ok {
				if exists {
					return binding.Schema{}, "indeterminate"
				}
				value, exists = component, true
			}
		}
		if !exists {
			return binding.Schema{}, "incompatible"
		}
		value, exists = lookupLiteral(value, parts[1:])
		if !exists {
			return binding.Schema{}, "incompatible"
		}
		schema := literalContract(value)
		if !schema.Known {
			return schema, "indeterminate"
		}
		return schema, "compatible"
	case strings.HasPrefix(source, "$steps."):
		parts := strings.Split(strings.TrimPrefix(source, "$steps."), ".")
		if len(parts) < 3 {
			return binding.Schema{}, "incompatible"
		}
		if scope.workflow == nil {
			return binding.Schema{}, "indeterminate"
		}
		var target *uws1.Step
		index := -1
		for i, step := range scope.workflow.Steps {
			if step != nil && step.StepID == parts[0] {
				if target != nil {
					return binding.Schema{}, "incompatible"
				}
				target, index = step, i
			}
		}
		if target == nil {
			return binding.Schema{}, "incompatible"
		}
		if scope.workflow.Type != uws1.WorkflowTypeSequence || !flatStep(target) || target.When != "" {
			return binding.Schema{}, "indeterminate"
		}
		if scope.step != nil && index >= scope.index {
			return binding.Schema{}, "incompatible"
		}
		targetScope := expressionScope{workflow: scope.workflow, step: target, operation: e.operations[target.OperationRef], index: index}
		schema, status := e.output(parts[2], targetScope, active, depth+1)
		if status != "compatible" {
			return schema, status
		}
		schema = childContract(schema, parts[3:])
		if !schema.Known {
			return schema, "indeterminate"
		}
		return schema, "compatible"
	case strings.HasPrefix(source, "$outputs."):
		parts := strings.Split(strings.TrimPrefix(source, "$outputs."), ".")
		schema, status := e.operationOutput(parts[0], scope, active, depth+1)
		if status != "compatible" {
			return schema, status
		}
		schema = childContract(schema, parts[1:])
		if !schema.Known {
			return schema, "indeterminate"
		}
		return schema, "compatible"
	case source == "$response.statusCode":
		shape := e.shape(scope.operation)
		if shape == nil || shape.Protocol != "http" {
			return binding.Schema{}, "indeterminate"
		}
		return binding.Schema{Known: true, JSON: json.RawMessage(`{"type":"integer"}`)}, "compatible"
	}
	if ref, ok := responseReference(source); ok {
		shape := e.shape(scope.operation)
		if shape == nil {
			return binding.Schema{}, "indeterminate"
		}
		for _, output := range shape.Outputs {
			if output.Location == ref.Location && output.Name == ref.Name {
				schema := output.Schema
				if ref.Pointer != "" {
					parts, err := expressions.Pointer(ref.Pointer)
					if err != nil {
						return binding.Schema{}, "incompatible"
					}
					schema = childContract(schema, parts)
				}
				if !schema.Known {
					return schema, "indeterminate"
				}
				if string(schema.JSON) == "false" {
					return schema, "incompatible"
				}
				return schema, "compatible"
			}
		}
		if shape.Complete {
			return binding.Schema{}, "incompatible"
		}
		return binding.Schema{}, "indeterminate"
	}
	return binding.Schema{}, "indeterminate"
}
func (e *expressionContracts) operationOutput(name string, scope expressionScope, active map[string]bool, depth int) (binding.Schema, string) {
	if scope.operation == nil {
		return binding.Schema{}, "indeterminate"
	}
	text, ok := scope.operation.Outputs[name]
	if !ok {
		return binding.Schema{}, "incompatible"
	}
	key := "operation/" + scope.operation.OperationID + "/" + name
	if active[key] {
		return binding.Schema{}, "incompatible"
	}
	active[key] = true
	defer delete(active, key)
	return e.contract(text, scope, active, depth+1)
}
func (e *expressionContracts) output(name string, scope expressionScope, active map[string]bool, depth int) (binding.Schema, string) {
	if scope.step == nil {
		return e.operationOutput(name, scope, active, depth)
	}
	if text, ok := scope.step.Outputs[name]; ok {
		key := "step/" + scope.step.StepID + "/" + name
		if active[key] {
			return binding.Schema{}, "incompatible"
		}
		active[key] = true
		defer delete(active, key)
		return e.contract(text, scope, active, depth+1)
	}
	return e.operationOutput(name, scope, active, depth)
}
func (e *expressionContracts) types(value any, scope expressionScope) map[string]binding.Schema {
	out := map[string]binding.Schema{}
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "$") {
				schema, status := e.contract(v, scope, map[string]bool{}, 0)
				if status == "compatible" && schema.Known {
					out[v] = schema
				}
			}
		case map[string]any:
			for _, child := range v {
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(value)
	return out
}
func (e *expressionContracts) checkOutputs(values map[string]string, scope expressionScope, add func(string, string)) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, status := e.contract(values[key], scope, map[string]bool{}, 0)
		if status != "compatible" {
			add("binding.output_reference_"+status, status)
		}
	}
}

func (e *expressionContracts) checkExpressionValues(value any, scope expressionScope, add func(string, string)) {
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "$") {
				_, status := e.contract(v, scope, map[string]bool{}, 0)
				if status == "incompatible" {
					add("binding.expression_reference_incompatible", status)
				}
			}
		case map[string]any:
			keys := make([]string, 0, len(v))
			for key := range v {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				walk(v[key])
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(value)
}
