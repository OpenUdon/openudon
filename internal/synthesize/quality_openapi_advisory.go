package synthesize

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/apitools/googlediscovery"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"gopkg.in/yaml.v3"
)

type securityOperationRef struct {
	Aliases []string
	Method  string
	Path    string
	Tags    []string
}

func operationRefsForSecuritySource(rel, path string) []securityOperationRef {
	sourceType := sourceDescriptionTypeForPath(rel)
	if sourceType != "openapi" {
		return nativeOperationRefsForSecurity(path, string(sourceType))
	}
	spec, err := rollout.LoadOpenAPISpec(path)
	if err != nil {
		return nil
	}
	var out []securityOperationRef
	for _, op := range spec.Operations {
		if op == nil || strings.TrimSpace(op.OperationID) == "" {
			continue
		}
		out = append(out, securityOperationRef{
			Aliases: []string{strings.TrimSpace(op.OperationID)},
			Method:  op.Method,
			Path:    op.Path,
			Tags:    append([]string(nil), op.Tags...),
		})
	}
	return out
}

func nativeOperationRefsForSecurity(path string, sourceType string) []securityOperationRef {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	switch sourceType {
	case "google-discovery":
		model, err := googlediscovery.Parse(data)
		if err != nil {
			return nil
		}
		var out []securityOperationRef
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			out = append(out, securityOperationRef{
				Aliases: operationIDAliases(op.OperationID, op.ID, op.Name),
				Method:  op.HTTPMethod,
				Path:    op.Path,
				Tags:    append([]string(nil), op.Tags...),
			})
		}
		return out
	case "aws-smithy":
		model, err := awssmithy.Parse(data)
		if err != nil {
			return nil
		}
		var out []securityOperationRef
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.Name) == "" {
				continue
			}
			out = append(out, securityOperationRef{
				Aliases: operationIDAliases(op.Name, op.ID),
				Method:  op.Method,
				Path:    firstNonEmpty(op.Path, op.URI),
			})
		}
		return out
	default:
		return nil
	}
}

func matchingOperationAliases(match catalog.OperationMatch, ops []securityOperationRef) [][]string {
	var out [][]string
	for _, op := range ops {
		if operationSecurityMatches(match, op) {
			out = append(out, op.Aliases)
		}
	}
	return out
}

func operationSecurityMatches(match catalog.OperationMatch, op securityOperationRef) bool {
	if strings.TrimSpace(match.OperationID) != "" {
		for _, alias := range op.Aliases {
			if strings.EqualFold(strings.TrimSpace(match.OperationID), alias) {
				return true
			}
		}
		return false
	}
	if strings.TrimSpace(match.Method) != "" && !strings.EqualFold(strings.TrimSpace(match.Method), strings.TrimSpace(op.Method)) {
		return false
	}
	if strings.TrimSpace(match.Path) != "" && strings.TrimSpace(match.Path) != strings.TrimSpace(op.Path) {
		return false
	}
	if len(match.Tags) > 0 {
		for _, want := range match.Tags {
			if !containsFold(op.Tags, want) {
				return false
			}
		}
	}
	return strings.TrimSpace(match.Method) != "" || strings.TrimSpace(match.Path) != "" || len(match.Tags) > 0
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func advisorySecuritySchemes(schemes []catalog.SecurityScheme) map[string]openAPISecurityRequirement {
	out := map[string]openAPISecurityRequirement{}
	for _, scheme := range schemes {
		name := strings.TrimSpace(scheme.Name)
		if name == "" {
			continue
		}
		out[name] = openAPISecurityRequirement{
			Scheme: name,
			Name:   strings.TrimSpace(scheme.ParameterName),
			In:     string(scheme.In),
			Type:   string(scheme.Type),
		}
	}
	return out
}

func advisorySecurityRequirements(requirements []catalog.SecurityRequirement, sets []catalog.SecurityRequirementSet, schemes map[string]openAPISecurityRequirement) []openAPISecurityRequirement {
	var out []openAPISecurityRequirement
	for _, req := range requirements {
		out = append(out, advisorySecurityRequirement(req, schemes))
	}
	for _, set := range sets {
		for _, req := range set.Requirements {
			out = append(out, advisorySecurityRequirement(req, schemes))
		}
	}
	return sortedSecurityRequirements(out)
}

func advisorySecurityRequirement(req catalog.SecurityRequirement, schemes map[string]openAPISecurityRequirement) openAPISecurityRequirement {
	name := strings.TrimSpace(req.Scheme)
	if scheme, ok := schemes[name]; ok {
		return scheme
	}
	return openAPISecurityRequirement{Scheme: name}
}

func readOpenAPISecurityDocument(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return asMap(raw), nil
}

func openAPISecuritySchemes(doc map[string]any) map[string]openAPISecurityRequirement {
	out := map[string]openAPISecurityRequirement{}
	components := asMap(doc["components"])
	schemes := asMap(components["securitySchemes"])
	if len(schemes) == 0 {
		schemes = asMap(doc["securityDefinitions"])
	}
	for name, raw := range schemes {
		scheme := asMap(raw)
		out[name] = openAPISecurityRequirement{
			Scheme: name,
			Name:   asString(scheme["name"]),
			In:     asString(scheme["in"]),
			Type:   asString(scheme["type"]),
		}
	}
	return out
}

func openAPISecurityRequirements(raw map[string]any, schemes map[string]openAPISecurityRequirement) []openAPISecurityRequirement {
	var out []openAPISecurityRequirement
	for _, item := range asSlice(raw) {
		req := asMap(item)
		for name := range req {
			if scheme, ok := schemes[name]; ok {
				out = append(out, scheme)
				continue
			}
			out = append(out, openAPISecurityRequirement{Scheme: name})
		}
	}
	return sortedSecurityRequirements(out)
}

func sortedSecurityRequirements(values []openAPISecurityRequirement) []openAPISecurityRequirement {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Scheme != values[j].Scheme {
			return values[i].Scheme < values[j].Scheme
		}
		if values[i].Name != values[j].Name {
			return values[i].Name < values[j].Name
		}
		if values[i].In != values[j].In {
			return values[i].In < values[j].In
		}
		return values[i].Type < values[j].Type
	})
	out := values[:0]
	seen := map[string]bool{}
	for _, value := range values {
		key := strings.Join([]string{
			strings.TrimSpace(value.Scheme),
			strings.TrimSpace(value.Name),
			strings.TrimSpace(value.In),
			strings.TrimSpace(value.Type),
		}, "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func asMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		out := map[string]any{}
		for key, val := range typed {
			out[fmt.Sprint(key)] = val
		}
		return out
	case []any:
		out := map[string]any{}
		for i, item := range typed {
			out[fmt.Sprint(i)] = item
		}
		return out
	default:
		return nil
	}
}

func asSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		if len(typed) == 0 {
			return nil
		}
		out := make([]any, 0, len(typed))
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			out = append(out, typed[key])
		}
		return out
	default:
		return nil
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func operationKey(specPath, operation string) string {
	return strings.TrimSpace(specPath) + "\x00" + strings.TrimSpace(operation)
}

func intentStepOpenAPIPath(intent *rollout.Intent, step *rollout.Step, primary string) string {
	if step != nil {
		if source := strings.TrimSpace(step.Source); source != "" {
			return source
		}
		if openapi := strings.TrimSpace(step.OpenAPI); openapi != "" {
			return openapi
		}
	}
	if intent != nil {
		if source := strings.TrimSpace(intent.Source); source != "" {
			return source
		}
		if openapi := strings.TrimSpace(intent.OpenAPI); openapi != "" {
			return openapi
		}
	}
	return strings.TrimSpace(primary)
}

func intentInputNames(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	for _, input := range intent.Inputs {
		if input != nil && strings.TrimSpace(input.Name) != "" {
			out[strings.TrimSpace(input.Name)] = true
		}
	}
	return out
}

func credentialLikeParam(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, token := range []string{"key", "token", "secret", "password", "appid", "api_key", "apikey", "authorization"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func stepSatisfiesParam(step *rollout.Step, param *rollout.ParameterInfo, inputs map[string]bool) bool {
	if step == nil || param == nil {
		return false
	}
	names := paramTargetNames(param)
	for _, name := range names {
		if step.With[name] != "" {
			return true
		}
		for _, bind := range step.Binds {
			if bind != nil && bind.Fields[name] != "" {
				return true
			}
		}
	}
	if inputs[param.Name] {
		return true
	}
	for _, value := range step.With {
		if referencesInputName(value, param.Name) {
			return true
		}
	}
	return false
}

func paramTargetNames(param *rollout.ParameterInfo) []string {
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil
	}
	var out []string
	out = append(out, name)
	if param.In != "" {
		out = append(out, strings.TrimSpace(param.In)+"."+name)
	}
	if param.In == "query" {
		out = append(out, "query_pars."+name)
	}
	if param.In == "path" {
		out = append(out, "path_pars."+name)
	}
	return out
}

func referencesInputName(value, name string) bool {
	value = strings.TrimSpace(value)
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return value == name || strings.Contains(value, "inputs."+name) || strings.Contains(value, "variables.inputs."+name) || strings.Contains(value, "input."+name)
}

func validateIntentDataFlowSources(intent *rollout.Intent) error {
	if intent == nil {
		return nil
	}
	stepNames := intentStepNameSet(intent)
	inputs := intentInputNames(intent)
	var unresolved []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for _, dep := range step.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep != "" && !stepNames[dep] {
				unresolved = append(unresolved, fmt.Sprintf("%s depends_on %q", name, dep))
			}
		}
		for target, source := range step.With {
			for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
				unresolved = append(unresolved, fmt.Sprintf("%s.%s references %q", name, target, ref))
			}
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			from := strings.TrimSpace(bind.From)
			if from != "" && !stepNames[from] {
				unresolved = append(unresolved, fmt.Sprintf("%s bind.from %q", name, from))
			}
			for target, source := range bind.Fields {
				for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
					unresolved = append(unresolved, fmt.Sprintf("%s.%s references %q", name, target, ref))
				}
			}
		}
		for label, source := range map[string]string{
			"when":       step.When,
			"for_each":   step.ForEach,
			"items":      step.Items,
			"batch_size": step.BatchSize,
		} {
			for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
				unresolved = append(unresolved, fmt.Sprintf("%s %s references %q", name, label, ref))
			}
		}
	})
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		for _, ref := range unresolvedDataFlowReferences(output.From, stepNames, inputs) {
			name := strings.TrimSpace(output.Name)
			if name == "" {
				name = "<unnamed>"
			}
			unresolved = append(unresolved, fmt.Sprintf("output %s references %q", name, ref))
		}
	}
	if len(unresolved) > 0 {
		return fmt.Errorf("%s. Use declared step names, inputs, or credential binding names only.", strings.Join(sortedCopy(unresolved), "; "))
	}
	return nil
}

func unresolvedDataFlowReferences(source string, stepNames, inputs map[string]bool) []string {
	var out []string
	for _, ref := range dataFlowReferencePrefixes(source) {
		lower := strings.ToLower(ref)
		if stepNames[ref] || inputs[ref] ||
			lower == "input" || lower == "inputs" || lower == "var" || lower == "vars" ||
			lower == "each" ||
			lower == "workflow" || lower == "trigger" || lower == "security" || lower == "credentials" ||
			lower == "body" || lower == "received_body" || lower == "request" || lower == "response" {
			continue
		}
		out = append(out, ref)
	}
	return sortedUnique(out)
}

func dataFlowReferencePrefixes(source string) []string {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	re := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_-]*)\s*\.`)
	matches := re.FindAllStringSubmatchIndex(source, -1)
	var out []string
	for _, match := range matches {
		if len(match) < 4 || dataFlowReferenceIsLiteralDomain(source, match[0]) {
			continue
		}
		out = append(out, source[match[2]:match[3]])
	}
	return sortedUnique(out)
}

func dataFlowReferenceIsLiteralDomain(source string, start int) bool {
	if start <= 0 || start > len(source) {
		return false
	}
	switch source[start-1] {
	case '@', '/', ':', '.':
		return true
	default:
		return false
	}
}

func intentStepNameSet(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			out[strings.TrimSpace(step.Name)] = true
		}
	})
	return out
}

type responsePathValidation struct {
	Failures []string
	Warnings []string
}

func validateIntentResponsePaths(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string) responsePathValidation {
	var result responsePathValidation
	if intent == nil {
		return result
	}
	ops := openAPIOperationIndex(candidates)
	for key, op := range localNativeOperationIndex(exampleDir) {
		ops[key] = op
	}
	stepOps := map[string]*rollout.OperationInfo{}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Name) == "" || strings.TrimSpace(step.Operation) == "" {
			return
		}
		op := ops[operationKey(intentStepOpenAPIPath(intent, step, primary), step.Operation)]
		if op != nil {
			stepOps[strings.TrimSpace(step.Name)] = op
		}
	})
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for target, source := range step.With {
			result.addResponsePathChecks(fmt.Sprintf("%s.%s", name, target), source, stepOps)
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for target, source := range bind.Fields {
				checkSource := strings.TrimSpace(source)
				from := strings.TrimSpace(bind.From)
				if from != "" && (strings.HasPrefix(checkSource, "body") || strings.HasPrefix(checkSource, "received_body")) {
					checkSource = from + "." + checkSource
				}
				result.addResponsePathChecks(fmt.Sprintf("%s.%s", name, target), checkSource, stepOps)
			}
		}
	})
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		name := strings.TrimSpace(output.Name)
		if name == "" {
			name = "<unnamed>"
		}
		result.addResponsePathChecks("output "+name, output.From, stepOps)
	}
	return result
}

func (r *responsePathValidation) addResponsePathChecks(label, source string, stepOps map[string]*rollout.OperationInfo) {
	for _, ref := range responsePathReferences(source) {
		op := stepOps[ref.Step]
		if op == nil {
			continue
		}
		switch responsePathStatus(op, ref.Path) {
		case "missing":
			r.Failures = append(r.Failures, fmt.Sprintf("%s references missing response path %s.%s", label, ref.Step, ref.Path))
		case "opaque":
			r.Warnings = append(r.Warnings, fmt.Sprintf("%s references unverified response path %s.%s", label, ref.Step, ref.Path))
		}
	}
}

type responsePathReference struct {
	Step string
	Path string
}

func responsePathReferences(source string) []responsePathReference {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	matches := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_-]*)\.(?:received_body|body)([A-Za-z0-9_\.\[\]-]*)`).FindAllStringSubmatch(source, -1)
	var out []responsePathReference
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		path := strings.TrimPrefix(match[2], ".")
		if path != "" {
			out = append(out, responsePathReference{Step: match[1], Path: path})
		}
	}
	return out
}

func responsePathStatus(op *rollout.OperationInfo, path string) string {
	schema := preferredResponseSchema(op)
	if len(schema) == 0 {
		return "opaque"
	}
	return schemaPathStatus(schema, responsePathTokens(path))
}

func preferredResponseSchema(op *rollout.OperationInfo) map[string]any {
	if op == nil {
		return nil
	}
	for _, code := range []string{"200", "201", "202", "204", "default"} {
		if response := op.Responses[code]; response != nil && len(response.Schema) > 0 {
			return response.Schema
		}
	}
	codes := make([]string, 0, len(op.Responses))
	for code := range op.Responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if response := op.Responses[code]; response != nil && len(response.Schema) > 0 {
			return response.Schema
		}
	}
	return nil
}

func responsePathTokens(path string) []string {
	path = strings.Trim(strings.TrimSpace(path), ".")
	if path == "" {
		return nil
	}
	path = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(path, "")
	path = strings.Trim(path, ".")
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func schemaPathStatus(schema map[string]any, tokens []string) string {
	if len(tokens) == 0 {
		return "present"
	}
	if len(schema) == 0 {
		return "missing"
	}
	if strings.TrimSpace(asString(schema["$ref"])) != "" {
		return "opaque"
	}
	if strings.EqualFold(asString(schema["type"]), "array") {
		return schemaPathStatus(asMap(schema["items"]), tokens)
	}
	props := asMap(schema["properties"])
	if len(props) > 0 {
		if next, ok := props[tokens[0]]; ok {
			return schemaPathStatus(asMap(next), tokens[1:])
		}
	}
	if additional := asMap(schema["additionalProperties"]); len(additional) > 0 {
		return schemaPathStatus(additional, tokens[1:])
	}
	return "missing"
}
