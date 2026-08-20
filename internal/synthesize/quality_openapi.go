package synthesize

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/googlediscovery"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

func validateIntentOpenAPIOperations(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	sourceRegistry, registryErr := newLocalAPISourceRegistry(exampleDir, candidates)
	if registryErr != nil && !os.IsNotExist(registryErr) {
		return fmt.Errorf("local API source registry could not be scanned: %w", registryErr)
	}
	var missing []string
	var omitted []string
	var invalid []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		specPath := intentStepOpenAPIPath(intent, step, primary)
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		if !intentStepRequiresOpenAPIOperation(intent, step, primary) {
			return
		}
		if entry, ok := sourceRegistry.get(specPath); ok && entry.Err != nil {
			invalid = append(invalid, fmt.Sprintf("%s in %q: %v", name, specPath, entry.Err))
			return
		}
		if sourceDescriptionTypeForPath(specPath) != "openapi" {
			if operation == "" {
				if intentStepRequiresOpenAPIOperation(intent, step, primary) {
					omitted = append(omitted, fmt.Sprintf("%s in %q", name, specPath))
				}
				return
			}
			entry, ok := sourceRegistry.get(specPath)
			if !ok {
				missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
				return
			}
			if len(entry.Operations) == 0 {
				omitted = append(omitted, fmt.Sprintf("%s in %q (no operations discovered)", name, specPath))
				return
			}
			if !entry.Operations[operation] {
				missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
			}
			return
		}
		if operation == "" {
			if intentStepRequiresOpenAPIOperation(intent, step, primary) {
				omitted = append(omitted, fmt.Sprintf("%s in %q", name, specPath))
			}
			return
		}
		if op := ops[operationKey(specPath, operation)]; op == nil {
			missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
		}
	})
	if len(invalid) > 0 || len(omitted) > 0 || len(missing) > 0 {
		sort.Strings(invalid)
		sort.Strings(omitted)
		sort.Strings(missing)
		var details []string
		for _, item := range invalid {
			details = append(details, "invalid API source "+item)
		}
		for _, item := range omitted {
			details = append(details, "missing operation for "+item)
		}
		for _, item := range missing {
			details = append(details, "missing API source operation "+item)
		}
		return fmt.Errorf("%s", strings.Join(details, "; "))
	}
	return nil
}

func intentStepRequiresOpenAPIOperation(intent *rollout.Intent, step *rollout.Step, primary string) bool {
	if step == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind != "" && kind != "http" && kind != "openapi" && kind != "browser" {
		return false
	}
	return strings.TrimSpace(intentStepOpenAPIPath(intent, step, primary)) != ""
}

func validateIntentRequiredParameters(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	for key, op := range localNativeOperationIndex(exampleDir) {
		ops[key] = op
	}
	inputs := intentInputNames(intent)
	var missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		if operation == "" {
			return
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, operation)]
		if op == nil {
			return
		}
		for _, param := range op.Parameters {
			if param == nil || !param.Required || credentialLikeParam(param.Name) {
				continue
			}
			if stepSatisfiesParam(step, param, inputs) {
				continue
			}
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			missing = append(missing, fmt.Sprintf("%s.%s requires %s parameter %q", name, operation, param.In, param.Name))
		}
	})
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s. Add literals, inputs, bind/with mappings, or import a complementary OpenAPI document that produces the missing values.", strings.Join(missing, "; "))
	}
	return nil
}

func validateIntentCredentialPolicy(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	for key, op := range localNativeOperationIndex(exampleDir) {
		ops[key] = op
	}
	inputs := intentInputNames(intent)
	var required, missingBinding []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		if operation == "" {
			return
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, operation)]
		if op == nil {
			return
		}
		for _, param := range op.Parameters {
			if param == nil || !param.Required || !credentialLikeParam(param.Name) {
				continue
			}
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			required = append(required, fmt.Sprintf("%s.%s requires credential-like parameter %q", name, operation, param.Name))
			if stepSatisfiesParam(step, param, inputs) {
				continue
			}
			if credentialDeclaredForParam(policy, param.Name) {
				continue
			}
			missingBinding = append(missingBinding, fmt.Sprintf("%s.%s has no auditable credential binding for %q", name, operation, param.Name))
		}
	})
	if len(required) == 0 {
		return nil
	}
	if strings.TrimSpace(policy.CredentialSection) == "" {
		sort.Strings(required)
		return fmt.Errorf("%s. Add a Credentials and Secrets section that names runtime credential bindings, never literal secrets.", strings.Join(required, "; "))
	}
	if len(missingBinding) > 0 {
		sort.Strings(missingBinding)
		return fmt.Errorf("%s. Add a with/bind request mapping or name a credential binding that includes the parameter name.", strings.Join(missingBinding, "; "))
	}
	return nil
}

func validateIntentOpenAPISecurity(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	security := openAPISecurityIndex(candidates)
	for key, reqs := range nativeSecurityIndex(candidates) {
		security[key] = append(security[key], reqs...)
	}
	for key, reqs := range localNativeSecurityIndex(exampleDir) {
		security[key] = append(security[key], reqs...)
	}
	advisory, advisoryErrs := localAdvisorySecurityIndex(exampleDir)
	if len(advisoryErrs) > 0 {
		return fmt.Errorf("advisory security sidecar metadata is invalid: %s", joinErrorMessages(advisoryErrs))
	}
	mergeAdvisorySecurityRequirements(security, advisory)
	for key, reqs := range security {
		security[key] = sortedSecurityRequirements(reqs)
	}
	var required, missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Operation) == "" {
			return
		}
		reqs := security[operationKey(intentStepOpenAPIPath(intent, step, primary), step.Operation)]
		if len(reqs) == 0 {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for _, req := range reqs {
			label := req.label()
			required = append(required, fmt.Sprintf("%s.%s requires API source security %q", name, step.Operation, label))
			if intentSecurityCoversRequirement(intent, req) || stepCoversSecurityRequirement(step, req, policy) || credentialDeclaredForSecurity(policy, req) {
				continue
			}
			missing = append(missing, fmt.Sprintf("%s.%s has no auditable credential binding for API source security %q", name, step.Operation, label))
		}
	})
	if len(required) == 0 {
		return nil
	}
	if strings.TrimSpace(policy.CredentialSection) == "" {
		return fmt.Errorf("%s. Add a Credentials and Secrets section that names security credential bindings, never literal secrets.", strings.Join(sortedCopy(required), "; "))
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s. Bind the security field by credential binding name or add a matching credential binding policy.", strings.Join(sortedCopy(missing), "; "))
	}
	return nil
}

func credentialDeclaredForParam(policy projectPolicy, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, binding := range credentialBindingNames(policy) {
		if strings.Contains(strings.ToLower(binding), name) {
			return true
		}
	}
	return false
}

func credentialDeclaredForSecurity(policy projectPolicy, req openAPISecurityRequirement) bool {
	for _, binding := range credentialBindingNames(policy) {
		if securityBindingMatches(binding, req) {
			return true
		}
	}
	return false
}

func securityBindingMatches(binding string, req openAPISecurityRequirement) bool {
	binding = strings.ToLower(strings.TrimSpace(binding))
	if binding == "" {
		return false
	}
	for _, candidate := range req.bindingCandidates() {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate != "" && (strings.Contains(binding, candidate) || strings.Contains(candidate, binding)) {
			return true
		}
	}
	return false
}

func intentSecurityCoversRequirement(intent *rollout.Intent, req openAPISecurityRequirement) bool {
	if intent == nil {
		return false
	}
	for _, security := range intent.Security {
		if security == nil {
			continue
		}
		for _, candidate := range []string{security.Name, security.TokenFrom} {
			if securityBindingMatches(candidate, req) {
				return true
			}
		}
	}
	return false
}

func stepCoversSecurityRequirement(step *rollout.Step, req openAPISecurityRequirement, policy projectPolicy) bool {
	if step == nil {
		return false
	}
	names := req.fieldNames()
	for _, name := range names {
		if source := strings.TrimSpace(step.With[name]); source != "" && securityCredentialSourceAllowed(source, req, policy) {
			return true
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			if source := strings.TrimSpace(bind.Fields[name]); source != "" && securityCredentialSourceAllowed(source, req, policy) {
				return true
			}
		}
	}
	return false
}

func securityCredentialSourceAllowed(source string, req openAPISecurityRequirement, policy projectPolicy) bool {
	source = strings.TrimSpace(source)
	if securityBindingMatches(source, req) {
		return true
	}
	credentialSource := strings.TrimPrefix(source, "credentials.")
	for _, binding := range credentialBindingNames(policy) {
		binding = strings.TrimSpace(binding)
		if strings.EqualFold(source, binding) || strings.EqualFold(credentialSource, binding) {
			return true
		}
	}
	return false
}

func openAPIOperationIndex(candidates []openapidisco.Candidate) map[string]*rollout.OperationInfo {
	out := map[string]*rollout.OperationInfo{}
	for _, candidate := range candidates {
		sourceType := sourceDescriptionTypeForPath(candidate.RelativePath)
		if sourceType != "openapi" {
			native, err := nativeOperationInfoIndex(candidate.Path, string(sourceType))
			if err != nil {
				continue
			}
			for alias, op := range native {
				out[operationKey(candidate.RelativePath, alias)] = op
			}
			continue
		}
		spec, err := rollout.LoadOpenAPISpec(candidate.Path)
		if err != nil {
			continue
		}
		for _, op := range spec.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			out[operationKey(candidate.RelativePath, op.OperationID)] = op
		}
	}
	return out
}

func localNativeOperationIndex(exampleDir string) map[string]*rollout.OperationInfo {
	out, _ := localNativeOperationIndexWithErrors(exampleDir)
	return out
}

func localNativeOperationIndexWithErrors(exampleDir string) (map[string]*rollout.OperationInfo, []error) {
	out := map[string]*rollout.OperationInfo{}
	var errs []error
	paths, err := packageartifacts.CollectExecutionSourcePaths(exampleDir)
	if err != nil {
		return out, []error{err}
	}
	for _, rel := range paths {
		sourceType := sourceDescriptionTypeForPath(rel)
		if sourceType == "openapi" {
			continue
		}
		path := filepath.Join(exampleDir, filepath.FromSlash(rel))
		native, err := nativeOperationInfoIndex(path, string(sourceType))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", rel, err))
			continue
		}
		for alias, op := range native {
			out[operationKey(rel, alias)] = op
		}
	}
	return out, errs
}

func nativeOperationInfoIndex(path string, sourceType string) (map[string]*rollout.OperationInfo, error) {
	out := map[string]*rollout.OperationInfo{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	switch sourceType {
	case "asyncapi":
		return asyncAPIOperationInfoIndex(path)
	case "google-discovery":
		model, err := googlediscovery.Parse(data)
		if err != nil {
			return out, fmt.Errorf("parse Google Discovery %s: %w", path, err)
		}
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			info := &rollout.OperationInfo{
				OperationID: op.OperationID,
				Method:      op.HTTPMethod,
				Path:        op.Path,
				Summary:     op.Summary,
				Description: op.Description,
				Responses:   googleDiscoveryResponses(model, op),
				Tags:        append([]string(nil), op.Tags...),
			}
			for _, param := range op.Parameters {
				if param == nil {
					continue
				}
				info.Parameters = append(info.Parameters, &rollout.ParameterInfo{
					Name:        param.Name,
					In:          param.Location,
					Required:    param.Required,
					Description: param.Description,
				})
			}
			if body := googleDiscoveryRequestBodySummary(model, op); body != nil && body.Required {
				for _, field := range requiredRequestBodyParameterNames(body) {
					info.Parameters = append(info.Parameters, &rollout.ParameterInfo{Name: field, In: "body", Required: true})
				}
			}
			for _, alias := range operationIDAliases(op.OperationID, op.ID, op.Name) {
				out[alias] = info
			}
		}
		return out, nil
	case "aws-smithy":
		model, err := awssmithy.Parse(data)
		if err != nil {
			return out, fmt.Errorf("parse AWS Smithy %s: %w", path, err)
		}
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.Name) == "" {
				continue
			}
			info := &rollout.OperationInfo{
				OperationID: op.Name,
				Method:      op.Method,
				Path:        op.URI,
				Responses:   smithyResponses(model, op),
			}
			for _, param := range smithyOperationParameters(op) {
				info.Parameters = append(info.Parameters, param)
			}
			for _, alias := range operationIDAliases(op.Name, op.ID) {
				out[alias] = info
			}
		}
		return out, nil
	case "graphql", "openrpc", "grpc-protobuf", "odata":
		uwsSourceType := uwsSourceTypeFromString(sourceType)
		ops, err := parseNativeOperationSummaries(data, path, uwsSourceType)
		if err != nil {
			return out, fmt.Errorf("parse %s %s: %w", sourceType, path, err)
		}
		for _, op := range ops {
			info := operationSummaryInfo(op)
			for _, alias := range nativeOperationIDAliases(uwsSourceType, op) {
				out[alias] = info
			}
		}
		return out, nil
	case "browser-profile":
		value, err := loadBrowserProfile(path)
		if err != nil {
			return out, fmt.Errorf("parse browser profile %s: %w", path, err)
		}
		for name, action := range value.Actions {
			info := &rollout.OperationInfo{
				OperationID: name,
				Method:      "BROWSER",
				Path:        "#/actions/" + name,
				Summary:     action.Description,
				Responses:   browserActionResponses(action),
			}
			for _, field := range browserRequiredParameterNames(action.Parameters) {
				info.Parameters = append(info.Parameters, &rollout.ParameterInfo{Name: field, In: "body", Required: true})
			}
			out[name] = info
		}
		return out, nil
	default:
		return out, nil
	}
}

func browserActionResponses(action profile.Action) map[string]*rollout.ResponseInfo {
	if len(action.Outputs) == 0 {
		return nil
	}
	properties := make(map[string]any, len(action.Outputs))
	for name, output := range action.Outputs {
		schema := map[string]any{"type": string(output.Type)}
		for key, value := range output.Validation {
			schema[key] = value
		}
		properties[name] = schema
	}
	return map[string]*rollout.ResponseInfo{
		"200": {Description: "Verified browser action outputs.", Schema: map[string]any{"type": "object", "properties": properties}},
	}
}

func browserRequiredParameterNames(schema profile.JSONSchema) []string {
	var names []string
	switch values := schema["required"].(type) {
	case []any:
		for _, value := range values {
			if name := strings.TrimSpace(fmt.Sprint(value)); name != "" {
				names = append(names, name)
			}
		}
	case []string:
		names = append(names, values...)
	}
	return sortedUnique(names)
}

func uwsSourceTypeFromString(sourceType string) uws1.SourceDescriptionType {
	return uws1.SourceDescriptionType(strings.TrimSpace(sourceType))
}

func operationSummaryInfo(op apitools.OperationSummary) *rollout.OperationInfo {
	info := &rollout.OperationInfo{
		OperationID:          op.OperationID,
		Method:               op.Method,
		Path:                 op.Path,
		Summary:              op.Summary,
		Description:          op.Description,
		Tags:                 append([]string(nil), op.Tags...),
		RequestBody:          operationSummaryRequestBodyInfo(op.RequestBody),
		Responses:            map[string]*rollout.ResponseInfo{},
		SecurityAlternatives: operationSummarySecurityAlternatives(op.SecurityRequirementSets),
	}
	for _, param := range op.Parameters {
		if strings.TrimSpace(param.Name) == "" {
			continue
		}
		info.Parameters = append(info.Parameters, &rollout.ParameterInfo{
			Name:        param.Name,
			In:          operationSummaryParameterLocation(param.In),
			Required:    param.Required,
			Description: param.Description,
			Type:        firstNonEmpty(param.Type, param.Ref),
		})
	}
	if op.RequestBody != nil && op.RequestBody.Required {
		for _, field := range requiredRequestBodyParameterNames(op.RequestBody) {
			info.Parameters = append(info.Parameters, &rollout.ParameterInfo{Name: field, In: "body", Required: true})
		}
	}
	if len(info.Responses) == 0 {
		info.Responses = nil
	}
	return info
}

func operationSummaryParameterLocation(location string) string {
	switch strings.TrimSpace(location) {
	case "graphql-variable", "json-rpc", "odata-parameter":
		return "body"
	case "odata-query-option":
		return "query"
	default:
		return strings.TrimSpace(location)
	}
}

func operationSummaryRequestBodyInfo(body *apitools.RequestBodySummary) *rollout.RequestBodyInfo {
	if body == nil {
		return nil
	}
	info := &rollout.RequestBodyInfo{
		Required:    body.Required,
		ContentType: firstNonEmpty(body.ContentTypes...),
	}
	if body.Schema != nil {
		info.Schema = schemaSummaryMap(body.Schema)
		return info
	}
	if len(body.Fields) == 0 {
		return info
	}
	properties := map[string]any{}
	var required []string
	for _, field := range body.Fields {
		name := strings.TrimSpace(field.Path)
		if name == "" {
			continue
		}
		properties[name] = map[string]any{"type": firstNonEmpty(field.Type, "string")}
		if field.Required {
			required = append(required, name)
		}
	}
	if len(properties) == 0 {
		return info
	}
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	info.Schema = schema
	return info
}

func schemaSummaryMap(summary *apitools.SchemaSummary) map[string]any {
	if summary == nil {
		return nil
	}
	schema := map[string]any{}
	if strings.TrimSpace(summary.Type) != "" {
		schema["type"] = strings.TrimSpace(summary.Type)
	}
	if strings.TrimSpace(summary.Format) != "" {
		schema["format"] = strings.TrimSpace(summary.Format)
	}
	if strings.TrimSpace(summary.Ref) != "" {
		schema["$ref"] = strings.TrimSpace(summary.Ref)
	}
	if len(summary.Required) > 0 {
		schema["required"] = append([]string(nil), summary.Required...)
	}
	if len(summary.Properties) > 0 {
		props := map[string]any{}
		for _, prop := range summary.Properties {
			if strings.TrimSpace(prop.Name) == "" {
				continue
			}
			props[prop.Name] = map[string]any{"type": firstNonEmpty(prop.Type, "string")}
		}
		if len(props) > 0 {
			schema["properties"] = props
		}
	}
	if len(schema) == 0 {
		return nil
	}
	return schema
}

func operationSummarySecurityAlternatives(sets []apitools.SecurityRequirementSetSummary) [][]string {
	if len(sets) == 0 {
		return nil
	}
	out := make([][]string, 0, len(sets))
	for _, set := range sets {
		names := []string{}
		for _, requirement := range set.Requirements {
			name := firstNonEmpty(requirement.Name, requirement.ParameterName, requirement.Scheme)
			if strings.TrimSpace(name) != "" {
				names = append(names, strings.TrimSpace(name))
			}
		}
		out = append(out, sortedUnique(names))
	}
	return out
}

func googleDiscoveryResponses(model *googlediscovery.Model, op *googlediscovery.Operation) map[string]*rollout.ResponseInfo {
	if model == nil || op == nil {
		return nil
	}
	ref := nativeSchemaRefName(op.ResponseRef)
	if ref == "" {
		return nil
	}
	schema := model.Schemas[ref]
	if len(schema) == 0 {
		return nil
	}
	return map[string]*rollout.ResponseInfo{
		"200": {
			ContentType: firstNonEmpty(op.ResponseMediaType, "application/json"),
			Schema:      schema,
		},
	}
}

func nativeSchemaRefName(ref string) string {
	ref = strings.TrimSpace(ref)
	for _, prefix := range []string{"#/components/schemas/", "#/schemas/", "#/definitions/"} {
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(ref, prefix))
		}
	}
	return ref
}

func smithyResponses(model *awssmithy.Model, op *awssmithy.Operation) map[string]*rollout.ResponseInfo {
	if model == nil || op == nil || len(op.OutputBindings) == 0 {
		return nil
	}
	props := map[string]any{}
	for _, binding := range op.OutputBindings {
		if binding == nil {
			continue
		}
		switch strings.TrimSpace(binding.Location) {
		case "", "payload":
		default:
			continue
		}
		name := strings.TrimSpace(binding.MemberName)
		if name == "" {
			continue
		}
		props[name] = smithyShapeSchema(model, binding.Target, nil)
	}
	if len(props) == 0 {
		return nil
	}
	return map[string]*rollout.ResponseInfo{
		"200": {
			ContentType: firstNonEmpty(op.ResponseMediaType, "application/json"),
			Schema: map[string]any{
				"type":       "object",
				"properties": props,
			},
		},
	}
}

func smithyShapeSchema(model *awssmithy.Model, target string, seen map[string]bool) map[string]any {
	target = strings.TrimSpace(target)
	if target == "" {
		return map[string]any{"$ref": "smithy:unknown"}
	}
	if schema := smithyPrimitiveSchema(target); len(schema) > 0 {
		return schema
	}
	if seen[target] {
		return map[string]any{"$ref": target}
	}
	nextSeen := make(map[string]bool, len(seen)+1)
	for key, value := range seen {
		nextSeen[key] = value
	}
	nextSeen[target] = true
	shape, ok := model.Shape(target)
	if !ok || shape == nil {
		return map[string]any{"$ref": target}
	}
	switch strings.TrimSpace(shape.Type) {
	case "structure", "union":
		members := asMap(shape.Raw["members"])
		if len(members) == 0 {
			return map[string]any{"type": "object"}
		}
		props := map[string]any{}
		var required []string
		for _, name := range sortedMapKeys(members) {
			member := asMap(members[name])
			if len(member) == 0 {
				continue
			}
			props[name] = smithyShapeSchema(model, asString(member["target"]), nextSeen)
			if _, ok := asMap(member["traits"])["smithy.api#required"]; ok {
				required = append(required, name)
			}
		}
		schema := map[string]any{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	case "list", "set":
		member := asMap(shape.Raw["member"])
		return map[string]any{
			"type":  "array",
			"items": smithyShapeSchema(model, asString(member["target"]), nextSeen),
		}
	case "map":
		value := asMap(shape.Raw["value"])
		return map[string]any{
			"type":                 "object",
			"additionalProperties": smithyShapeSchema(model, asString(value["target"]), nextSeen),
		}
	case "boolean":
		return map[string]any{"type": "boolean"}
	case "byte", "short", "integer", "long", "bigInteger":
		return map[string]any{"type": "integer"}
	case "float", "double", "bigDecimal":
		return map[string]any{"type": "number"}
	case "timestamp":
		return map[string]any{"type": "string", "format": "date-time"}
	case "blob":
		return map[string]any{"type": "string", "format": "byte"}
	case "document":
		return map[string]any{"$ref": target}
	default:
		return map[string]any{"type": "string"}
	}
}

func smithyPrimitiveSchema(target string) map[string]any {
	switch strings.TrimPrefix(strings.TrimSpace(target), "smithy.api#") {
	case "String", "PrimitiveString", "Enum", "IntEnum":
		return map[string]any{"type": "string"}
	case "Boolean", "PrimitiveBoolean":
		return map[string]any{"type": "boolean"}
	case "Byte", "Short", "Integer", "Long", "PrimitiveByte", "PrimitiveShort", "PrimitiveInteger", "PrimitiveLong", "BigInteger":
		return map[string]any{"type": "integer"}
	case "Float", "Double", "PrimitiveFloat", "PrimitiveDouble", "BigDecimal":
		return map[string]any{"type": "number"}
	case "Timestamp":
		return map[string]any{"type": "string", "format": "date-time"}
	case "Blob":
		return map[string]any{"type": "string", "format": "byte"}
	case "Document":
		return map[string]any{"$ref": target}
	default:
		return nil
	}
}

func requiredRequestBodyParameterNames(body *apitools.RequestBodySummary) []string {
	if body == nil || !body.Required {
		return nil
	}
	var out []string
	for _, field := range body.Fields {
		if strings.TrimSpace(field.Path) != "" && field.Required {
			out = append(out, strings.TrimSpace(field.Path))
		}
	}
	if len(out) == 0 && len(body.Fields) > 0 {
		for _, preferred := range []string{"raw", "message", "text", "body", "content", "html", "payload"} {
			for _, field := range body.Fields {
				if strings.EqualFold(strings.TrimSpace(field.Path), preferred) {
					return []string{strings.TrimSpace(field.Path)}
				}
			}
		}
	}
	if len(out) == 0 {
		out = append(out, "body")
	}
	return sortedUnique(out)
}

func operationIDAliases(ids ...string) []string {
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, id)
		alias := operationIDAlias(id)
		if alias != "" && alias != id {
			out = append(out, alias)
		}
	}
	return sortedUnique(out)
}

func nativeOperationIDAliases(sourceType uws1.SourceDescriptionType, op apitools.OperationSummary) []string {
	ids := []string{op.OperationID, op.ID, op.Extensions["source_operation_id"]}
	if sourceType == uws1.SourceDescriptionTypeOData {
		for _, id := range append([]string(nil), ids...) {
			if canonical := canonicalODataSourceOperationID(id); canonical != "" {
				ids = append(ids, canonical)
			}
		}
	}
	return operationIDAliases(ids...)
}

func sourceOperationIDForUWS(sourceType uws1.SourceDescriptionType, operationID string) string {
	operationID = strings.TrimSpace(operationID)
	if sourceType == uws1.SourceDescriptionTypeOData {
		if canonical := canonicalODataSourceOperationID(operationID); canonical != "" {
			return canonical
		}
	}
	return operationID
}

func canonicalODataSourceOperationID(operationID string) string {
	operationID = strings.TrimSpace(operationID)
	switch {
	case strings.HasPrefix(operationID, "entityset."):
		name := strings.TrimSpace(strings.TrimPrefix(operationID, "entityset."))
		if name != "" {
			return "entitySet." + name + ".query"
		}
	case strings.HasPrefix(operationID, "singleton.") && !strings.HasSuffix(operationID, ".read"):
		name := strings.TrimSpace(strings.TrimPrefix(operationID, "singleton."))
		if name != "" {
			return "singleton." + name + ".read"
		}
	}
	return operationID
}

func operationIDAlias(id string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}
