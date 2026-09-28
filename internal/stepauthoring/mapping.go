package stepauthoring

import (
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

type fieldEvidence struct {
	typeName      string
	format        string
	required      bool
	requiredKnown bool
	nullable      bool
	found         bool
}

func operationCapability(candidate apitools.OperationCandidate, dimension string) apitools.OperationCapabilityStatus {
	for _, capability := range candidate.Capabilities {
		if capability.Dimension == dimension {
			return capability.Status
		}
	}
	return apitools.OperationCapabilityUnsupported
}

func addMappingCompatibilityCheck(check *checkAccumulator, code, dimension, status string, pointer ...string) {
	location := ""
	if len(pointer) > 0 {
		location = pointer[0]
	}
	switch status {
	case "compatible":
		check.add(code, "pass", "Source metadata is compatible with the declared step "+dimension+" contract.")
	case "incompatible":
		check.addAt(code, "fail", "Source metadata conflicts with the declared step "+dimension+" contract.", location)
	default:
		check.addAt(code, "indeterminate", "Source metadata cannot establish compatibility with the declared step "+dimension+" contract.", location)
	}
}

func aggregateMappingStatus(current, next string) string {
	if current == "incompatible" || next == "incompatible" {
		return "incompatible"
	}
	if current == "indeterminate" || next == "indeterminate" {
		return "indeterminate"
	}
	return "compatible"
}

func combineStepMappings(step *workflowintent.Step) map[string]string {
	mappings := map[string]string{}
	if step == nil {
		return mappings
	}
	for key, value := range step.With {
		mappings[key] = value
	}
	for _, binding := range step.Binds {
		if binding == nil {
			continue
		}
		for key, value := range binding.Fields {
			mappings[key] = value
		}
	}
	return mappings
}

func inputContractPath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "inputs.") {
		return "", false
	}
	path := strings.TrimPrefix(value, "inputs.")
	if path == "" || strings.ContainsAny(path, "[]\" ") || strings.Contains(path, "..") {
		return "", false
	}
	return path, true
}

func mappingPointer(base, path string) string {
	parts := strings.Split(path, ".")
	for index, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		part = strings.ReplaceAll(part, "/", "~1")
		parts[index] = part
	}
	if path == "" {
		return base
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.Join(parts, "/")
}

func escapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}

func contractSchemaField(schema *uws1.ParamSchema, path string) (*uws1.ParamSchema, bool, bool) {
	if schema == nil || path == "" {
		return nil, false, false
	}
	current := schema
	allRequired := true
	for _, part := range strings.Split(path, ".") {
		if current == nil || current.Type != "" && current.Type != "object" {
			return nil, false, false
		}
		child, ok := current.Properties[part]
		if !ok || child == nil {
			return nil, false, false
		}
		allRequired = allRequired && schemaRequired(current, part)
		current = child
	}
	return current, allRequired, true
}

func contractRootField(schema *uws1.ParamSchema, path string) *uws1.ParamSchema {
	if schema == nil || path == "" {
		return nil
	}
	return schema.Properties[strings.Split(path, ".")[0]]
}

func requiredContractPaths(schema *uws1.ParamSchema) []string {
	if schema == nil {
		return nil
	}
	var paths []string
	var walk func(*uws1.ParamSchema, string, bool)
	walk = func(node *uws1.ParamSchema, prefix string, ancestorsRequired bool) {
		if node == nil {
			return
		}
		names := make([]string, 0, len(node.Properties))
		for name := range node.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			path := name
			if prefix != "" {
				path = prefix + "." + name
			}
			required := ancestorsRequired && schemaRequired(node, name)
			if required {
				paths = append(paths, path)
			}
			walk(node.Properties[name], path, required)
		}
	}
	walk(schema, "", true)
	return paths
}

func pathRelated(left, right string) bool {
	return left == right || strings.HasPrefix(left, right+".") || strings.HasPrefix(right, left+".")
}

func contractInputMapped(mappings map[string]string, path string) bool {
	for _, value := range mappings {
		mappedPath, ok := inputContractPath(value)
		if ok && pathRelated(mappedPath, path) {
			return true
		}
	}
	return false
}

func hasLocatedInputMapping(mappings map[string]string, locations map[string]map[string]bool, section, name string) bool {
	for key, value := range mappings {
		if strings.TrimSpace(value) == "" {
			continue
		}
		gotSection, gotName, qualified := requestMappingLocation(key)
		if !qualified {
			if len(locations[gotName]) != 1 {
				continue
			}
			for location := range locations[gotName] {
				gotSection = location
			}
		}
		if gotSection == section && pathRelated(gotName, name) {
			return true
		}
	}
	return false
}

func requestFieldEvidence(operation apitools.OperationSummary, section, name string) fieldEvidence {
	for _, parameter := range operation.Parameters {
		if parameter.Name == name && sourceParameterSection(parameter.In) == section {
			return fieldEvidence{typeName: parameter.Type, format: parameter.Format, required: parameter.Required, requiredKnown: true, found: true}
		}
	}
	if section != "body" || operation.RequestBody == nil {
		return fieldEvidence{}
	}
	for _, field := range operation.RequestBody.Fields {
		if strings.TrimPrefix(strings.TrimSpace(field.Path), "$") == name {
			return fieldEvidence{typeName: field.Type, format: field.Format, required: field.Required, requiredKnown: true, nullable: field.Nullable, found: true}
		}
	}
	for _, path := range operation.RequestBody.RequiredFieldPaths {
		if path == name {
			return fieldEvidence{required: true, requiredKnown: true, found: true}
		}
	}
	return fieldEvidence{}
}

func responseFieldEvidence(operation apitools.OperationSummary, path string) fieldEvidence {
	if operation.ResponseBody == nil {
		return fieldEvidence{}
	}
	if path == "" {
		if operation.ResponseBody.Schema == nil {
			return fieldEvidence{}
		}
		return fieldEvidence{
			typeName: operation.ResponseBody.Schema.Type,
			format:   operation.ResponseBody.Schema.Format,
			nullable: operation.ResponseBody.Schema.Nullable,
			found:    true,
		}
	}
	path = strings.TrimPrefix(path, "$")
	for _, field := range operation.ResponseBody.Fields {
		if strings.TrimPrefix(strings.TrimSpace(field.Path), "$") == path {
			return fieldEvidence{typeName: field.Type, format: field.Format, required: field.Required, requiredKnown: true, nullable: field.Nullable, found: true}
		}
	}
	return fieldEvidence{}
}

func checkRequestMappingCompatibility(step *workflowintent.Step, contract StepContract, operation apitools.OperationSummary, available bool) (string, string) {
	status := "compatible"
	pointer := ""
	locations := declaredRequestLocations(operation)
	mappings := combineStepMappings(step)
	keys := make([]string, 0, len(mappings))
	for key := range mappings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := mappings[key]
		section, name, qualified := requestMappingLocation(strings.TrimSpace(key))
		if !qualified {
			if len(locations[name]) != 1 {
				if _, isInput := inputContractPath(value); isInput {
					status = aggregateMappingStatus(status, "indeterminate")
					if pointer == "" {
						pointer = "/request_mappings/" + escapeJSONPointer(key)
					}
				}
				continue
			}
			for section = range locations[name] {
			}
		}
		path, isInput := inputContractPath(value)
		if !isInput {
			if !validCredentialReference(strings.TrimSpace(value)) && requestFieldEvidence(operation, section, name).found {
				status = aggregateMappingStatus(status, "indeterminate")
				if pointer == "" {
					pointer = "/request_mappings/" + escapeJSONPointer(key)
				}
			}
			continue
		}
		contractField, contractRequired, contractFound := contractSchemaField(contract.Inputs, path)
		if !contractFound {
			status = aggregateMappingStatus(status, "indeterminate")
			if pointer == "" {
				pointer = mappingPointer("/contract/inputs", path)
			}
			continue
		}
		source := requestFieldEvidence(operation, section, name)
		fieldStatus := fieldTypeStatus(contractField, source, true)
		if source.requiredKnown && source.required && !contractRequired {
			fieldStatus = "incompatible"
		}
		if fieldStatus != "compatible" && (status == "compatible" || fieldStatus == "incompatible" && status != "incompatible") {
			pointer = mappingPointer("/contract/inputs", path)
		}
		status = aggregateMappingStatus(status, fieldStatus)
	}
	if !available {
		status = aggregateMappingStatus(status, "indeterminate")
		if pointer == "" {
			pointer = "/operation/capabilities/inputs"
		}
	}
	return status, pointer
}

func fieldTypeStatus(contract *uws1.ParamSchema, source fieldEvidence, input bool) string {
	if contract == nil || !source.found || contract.Type == "" || source.typeName == "" {
		return "indeterminate"
	}
	actual, expected := source.typeName, contract.Type
	if input {
		actual, expected = contract.Type, source.typeName
	}
	compatible, known := workflowInputTypeCompatible(actual, expected)
	if !known {
		return "indeterminate"
	}
	if !compatible {
		return "incompatible"
	}
	if input {
		if source.format != "" && contract.Format == "" {
			return "indeterminate"
		}
		if source.format != "" && contract.Format != source.format {
			return "incompatible"
		}
	} else {
		if contract.Format != "" && source.format == "" {
			return "indeterminate"
		}
		if contract.Format != "" && source.format != contract.Format {
			return "incompatible"
		}
	}
	return "compatible"
}

func effectiveOutputMappings(contract *uws1.ParamSchema, mappings map[string]string) map[string]string {
	if mappings != nil {
		copy := make(map[string]string, len(mappings))
		for path, target := range mappings {
			copy[path] = target
		}
		return copy
	}
	identity := make(map[string]string)
	for path := range contractOutputPaths(contract.Properties) {
		identity[path] = "received_body." + path
	}
	return identity
}

func effectiveOutputTarget(path string, mappings map[string]string) (string, bool) {
	if target, ok := mappings[path]; ok {
		return target, true
	}
	ancestor := path
	for {
		index := strings.LastIndex(ancestor, ".")
		if index < 0 {
			return "", false
		}
		ancestor = ancestor[:index]
		if target, ok := mappings[ancestor]; ok {
			suffix := strings.TrimPrefix(path, ancestor)
			return target + suffix, true
		}
	}
}

func responseReferencePath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "received_body" {
		return "", true
	}
	if !strings.HasPrefix(value, "received_body.") {
		return "", false
	}
	return strings.TrimPrefix(value, "received_body."), true
}

func outputPathCoveredByContract(field string, schema *uws1.ParamSchema, mappings map[string]string) bool {
	for contractPath := range contractOutputPaths(schema.Properties) {
		target, ok := effectiveOutputTarget(contractPath, mappings)
		if !ok {
			continue
		}
		responsePath, ok := responseReferencePath(target)
		if !ok {
			continue
		}
		contractField := contractPath
		if responsePath == field {
			if _, _, exists := contractSchemaField(schema, contractField); exists {
				return true
			}
		}
		if responsePath != "" && strings.HasPrefix(field, responsePath+".") {
			suffix := strings.TrimPrefix(field, responsePath+".")
			mappedContractPath := contractField + "." + suffix
			if _, _, exists := contractSchemaField(schema, mappedContractPath); exists {
				mappedTarget, mapped := effectiveOutputTarget(mappedContractPath, mappings)
				if mapped && mappedTarget == "received_body."+field {
					return true
				}
			}
		}
	}
	return false
}
