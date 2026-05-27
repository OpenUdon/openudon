package synthesize

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/googlediscovery"
	apitoolshelper "github.com/OpenUdon/apitools/helper"
	"github.com/OpenUdon/apitools/helper/fnctspec"
	"github.com/OpenUdon/openudon/internal/uwsvalidate"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/runtimes"
	"github.com/OpenUdon/uws/uws1"
	"gopkg.in/yaml.v3"
)

func promoteWorkflow(result Result, schemaPath string) error {
	doc, err := loadUWSDocumentFile(result.WorkflowPath)
	if err != nil {
		return fmt.Errorf("load workflow UWS document: %w", err)
	}
	normalizeUWSStepsForSchema(doc)
	if intent, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		addStructuralResultsFromIntent(doc, intent)
	}
	uwsBytes, err := convert.MarshalYAML(doc)
	if err != nil {
		return fmt.Errorf("marshal UWS: %w", err)
	}
	uwsBytes, err = pruneEmptyUWSStepTypes(uwsBytes)
	if err != nil {
		return fmt.Errorf("normalize UWS YAML: %w", err)
	}
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	if err := os.WriteFile(result.UWSPath, uwsBytes, 0o644); err != nil {
		return err
	}

	schemaPath = strings.TrimSpace(schemaPath)
	if schemaPath == "" {
		schemaPath = defaultSchemaPathForVersion(result.ExampleDir, doc.UWS)
	}
	if err := uwsvalidate.ValidateFile(schemaPath, result.UWSPath); err != nil {
		return fmt.Errorf("validate exported UWS: %w", err)
	}
	return nil
}

func loadUWSDocumentFile(path string) (*uws1.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc uws1.Document
	switch strings.ToLower(filepath.Ext(path)) {
	case ".hcl":
		err = convert.UnmarshalHCL(data, &doc)
	case ".json":
		err = convert.UnmarshalJSON(data, &doc)
	default:
		err = convert.UnmarshalYAML(data, &doc)
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func generateWorkflowDocument(result Result, intent *rollout.Intent) (*uws1.Document, error) {
	if intent == nil {
		return nil, fmt.Errorf("intent is required")
	}
	normalized := intent.NormalizedForGeneration()
	title := "OpenUdon workflow"
	description := ""
	timeout := (*float64)(nil)
	var idempotency *uws1.Idempotency
	if normalized.Workflow != nil {
		if strings.TrimSpace(normalized.Workflow.Name) != "" {
			title = normalized.Workflow.Name
		}
		description = normalized.Workflow.Description
		timeout = normalized.Workflow.Timeout
		idempotency = normalized.Workflow.Idempotency
	}
	doc := &uws1.Document{
		UWS: uwsVersionForIntent(normalized),
		Info: &uws1.Info{
			Title:       title,
			Description: description,
			Version:     "1.0.0",
		},
		Operations: []*uws1.Operation{},
		Workflows: []*uws1.Workflow{{
			WorkflowID:       "main",
			Type:             uws1.WorkflowTypeSequence,
			Description:      description,
			Idempotency:      idempotency,
			Steps:            []*uws1.Step{},
			Outputs:          workflowOutputs(normalized.Outputs),
			StructuralFields: uws1.StructuralFields{},
		}},
	}
	doc.Workflows[0].Timeout = timeout
	sourceNames := map[string]string{}
	requestMapper := newRequestBindingMapper(result.ExampleDir)
	ensureSourceDescription := func(rel string) string {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			return ""
		}
		if name := sourceNames[rel]; name != "" {
			return name
		}
		name := sourceDescriptionName(rel, sourceNames)
		sourceNames[rel] = name
		doc.SourceDescriptions = append(doc.SourceDescriptions, &uws1.SourceDescription{
			Name: name,
			URL:  filepath.ToSlash(rel),
			Type: sourceDescriptionTypeForPath(rel),
		})
		return name
	}
	defaultOpenAPI := firstNonEmpty(normalized.Source, normalized.OpenAPI, result.PrimaryOpenAPI)
	steps, ops, err := buildUWSSteps(normalized.Steps, defaultOpenAPI, ensureSourceDescription, requestMapper)
	if err != nil {
		return nil, err
	}
	doc.Workflows[0].Steps = steps
	doc.Operations = append(doc.Operations, ops...)
	addTriggers(doc, normalized.Triggers)
	addStructuralResultsFromIntent(doc, normalized)
	if len(doc.Operations) == 0 && len(doc.Workflows[0].Steps) == 0 {
		return nil, fmt.Errorf("intent produced no UWS operations or steps")
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return doc, nil
}

func buildUWSSteps(steps []*rollout.Step, defaultOpenAPI string, sourceFor func(string) string, requestMapper *requestBindingMapper) ([]*uws1.Step, []*uws1.Operation, error) {
	var outSteps []*uws1.Step
	var outOps []*uws1.Operation
	for _, step := range steps {
		if step == nil {
			continue
		}
		uwsStep, ops, err := buildUWSStep(step, defaultOpenAPI, sourceFor, requestMapper)
		if err != nil {
			return nil, nil, err
		}
		if uwsStep != nil {
			outSteps = append(outSteps, uwsStep)
		}
		outOps = append(outOps, ops...)
	}
	return outSteps, outOps, nil
}

func buildUWSStep(step *rollout.Step, defaultOpenAPI string, sourceFor func(string) string, requestMapper *requestBindingMapper) (*uws1.Step, []*uws1.Operation, error) {
	name := sanitizeIdentifier(firstNonEmpty(step.Name, "step"))
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind == "" {
		if strings.TrimSpace(step.Operation) != "" {
			kind = "http"
		} else {
			kind = "fnct"
		}
	}
	uwsStep := &uws1.Step{
		StepID:      name,
		Description: strings.TrimSpace(step.Do),
		StepExecutionFields: uws1.StepExecutionFields{
			DependsOn: uniqueStrings(step.DependsOn),
			When:      runtimeInputExecutionExpression(strings.TrimSpace(step.When)),
			ForEach:   runtimeInputExecutionExpression(strings.TrimSpace(step.ForEach)),
			Timeout:   step.Timeout,
		},
		StructuralFields: uws1.StructuralFields{
			Items:     runtimeInputExecutionExpression(strings.TrimSpace(step.Items)),
			Mode:      strings.TrimSpace(step.Mode),
			BatchSize: strings.TrimSpace(step.BatchSize),
		},
	}
	var ops []*uws1.Operation
	if isIntentStructuralType(kind) {
		uwsStep.Type = uwsWorkflowType(kind)
		nested, nestedOps, err := buildUWSSteps(step.Steps, firstNonEmpty(step.Source, step.OpenAPI, defaultOpenAPI), sourceFor, requestMapper)
		if err != nil {
			return nil, nil, err
		}
		uwsStep.Steps = nested
		ops = append(ops, nestedOps...)
		for _, branch := range step.Cases {
			if branch == nil {
				continue
			}
			caseSteps, caseOps, err := buildUWSSteps(branch.Steps, firstNonEmpty(step.Source, step.OpenAPI, defaultOpenAPI), sourceFor, requestMapper)
			if err != nil {
				return nil, nil, err
			}
			uwsStep.Cases = append(uwsStep.Cases, &uws1.Case{
				CaseFields: uws1.CaseFields{Name: branch.Name, When: runtimeInputExecutionExpression(branch.When)},
				Steps:      caseSteps,
			})
			ops = append(ops, caseOps...)
		}
		if step.Default != nil {
			defaultSteps, defaultOps, err := buildUWSSteps(step.Default.Steps, firstNonEmpty(step.Source, step.OpenAPI, defaultOpenAPI), sourceFor, requestMapper)
			if err != nil {
				return nil, nil, err
			}
			uwsStep.Default = defaultSteps
			ops = append(ops, defaultOps...)
		}
		return uwsStep, ops, nil
	}
	openAPIPath := firstNonEmpty(step.Source, step.OpenAPI, defaultOpenAPI)
	request, err := intentRequestMap(step.With, kind, openAPIPath, step.Operation, requestMapper)
	if err != nil {
		return nil, nil, fmt.Errorf("step %s request bindings: %w", name, err)
	}
	op := &uws1.Operation{
		OperationID:              name,
		Description:              strings.TrimSpace(step.Do),
		Request:                  request,
		SuccessCriteria:          step.SuccessCriteria,
		OnFailure:                step.OnFailure,
		OnSuccess:                step.OnSuccess,
		OperationExecutionFields: uws1.OperationExecutionFields{DependsOn: uniqueStrings(step.DependsOn), When: runtimeInputExecutionExpression(step.When), ForEach: runtimeInputExecutionExpression(step.ForEach), Timeout: step.Timeout},
	}
	switch kind {
	case "http", "openapi":
		source := sourceFor(openAPIPath)
		if source == "" {
			return nil, nil, fmt.Errorf("step %s references OpenAPI operation without source document", name)
		}
		op.SourceDescription = source
		sourceType := sourceDescriptionTypeForPath(openAPIPath)
		if sourceType == uws1.SourceDescriptionTypeOpenAPI {
			op.OpenAPIOperationID = strings.TrimSpace(step.Operation)
		} else if strings.HasPrefix(strings.TrimSpace(step.Operation), "#/") {
			op.SourceOperationRef = strings.TrimSpace(step.Operation)
		} else {
			op.SourceOperationID = strings.TrimSpace(step.Operation)
		}
	default:
		if op.Extensions == nil {
			op.Extensions = map[string]any{}
		}
		op.Extensions[uws1.ExtensionOperationProfile] = runtimes.ProfileName
		runtime := &runtimes.OperationRuntime{Type: kind}
		switch kind {
		case "cmd":
			runtime.Command = firstNonEmpty(step.With["command"], step.Do)
		case "fnct":
			runtime.Function = firstNonEmpty(step.Provider, step.Operation, name)
		}
		if !fnctUsesRequestBodyObject(runtime.Function) {
			var args []any
			for _, key := range sortedStringMapKeys(step.With) {
				args = append(args, map[string]any{"name": key, "value": step.With[key]})
			}
			runtime.Arguments = args
		}
		if err := runtimes.SetOperationExtension(&op.Extensions, runtime); err != nil {
			return nil, nil, err
		}
	}
	uwsStep.OperationRef = op.OperationID
	return uwsStep, []*uws1.Operation{op}, nil
}

func fnctUsesRequestBodyObject(function string) bool {
	function = strings.TrimSpace(function)
	if function == "" {
		return false
	}
	for _, spec := range apitoolshelper.FunctionSpecs() {
		if spec.Name == function && spec.InvocationMode == fnctspec.InvocationRequestBodyObject {
			return true
		}
	}
	return false
}

func workflowOutputs(outputs []*rollout.Output) map[string]string {
	out := map[string]string{}
	for _, output := range outputs {
		if output == nil || strings.TrimSpace(output.Name) == "" || strings.TrimSpace(output.From) == "" {
			continue
		}
		out[output.Name] = runtimeInputExecutionExpression(output.From)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func intentRequestMap(values map[string]string, kind, openAPIPath, operationID string, mapper *requestBindingMapper) (map[string]any, error) {
	if len(values) == 0 {
		return nil, nil
	}
	root := map[string]any{}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || strings.TrimSpace(value) == "" {
			continue
		}
		requestValue := requestBindingValue(value)
		parts := strings.Split(key, ".")
		if len(parts) == 1 && isStandardRequestSection(parts[0]) {
			if kind == "http" || kind == "openapi" {
				if mapper == nil {
					return nil, fmt.Errorf("API source request metadata is unavailable for %q", key)
				}
				placement, err := mapper.lookup(openAPIPath, operationID, key)
				if err == nil {
					assignRequestPlacement(root, placement, requestValue)
					continue
				}
			}
			if parts[0] != "body" {
				return nil, fmt.Errorf("request section %q requires a field name", parts[0])
			}
			root["body"] = requestValue
			continue
		}
		if len(parts) > 1 && isStandardRequestSection(parts[0]) {
			if parts[0] == "body" {
				assignNestedMap(root, parts, requestValue)
			} else {
				child, _ := root[parts[0]].(map[string]any)
				if child == nil {
					child = map[string]any{}
					root[parts[0]] = child
				}
				assignNested(child, parts[1:], requestValue)
			}
			continue
		}
		placement := requestFieldPlacement{Section: "body", Name: key}
		if kind == "http" || kind == "openapi" {
			if mapper == nil {
				return nil, fmt.Errorf("API source request metadata is unavailable for %q", key)
			}
			var err error
			placement, err = mapper.lookup(openAPIPath, operationID, key)
			if err != nil {
				return nil, err
			}
		}
		assignRequestPlacement(root, placement, requestValue)
	}
	if len(root) == 0 {
		return nil, nil
	}
	return root, nil
}

func isStandardRequestSection(section string) bool {
	switch section {
	case "path", "query", "header", "cookie", "body":
		return true
	default:
		return false
	}
}

func assignRequestPlacement(root map[string]any, placement requestFieldPlacement, value any) {
	section := strings.TrimSpace(placement.Section)
	if section == "" {
		section = "body"
	}
	name := strings.TrimSpace(placement.Name)
	if name == "" {
		name = strings.TrimSpace(placement.Original)
	}
	if section == "body" {
		parts := []string{"body"}
		if name != "" && name != "body" {
			parts = append(parts, strings.Split(name, ".")...)
		}
		assignNestedMap(root, parts, value)
		return
	}
	child, _ := root[section].(map[string]any)
	if child == nil {
		child = map[string]any{}
		root[section] = child
	}
	child[name] = value
}

func assignNestedMap(root map[string]any, parts []string, value any) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		root[parts[0]] = value
		return
	}
	child, _ := root[parts[0]].(map[string]any)
	if child == nil {
		child = map[string]any{}
		root[parts[0]] = child
	}
	assignNested(child, parts[1:], value)
}

func assignNested(root map[string]any, parts []string, value any) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		root[parts[0]] = value
		return
	}
	child, _ := root[parts[0]].(map[string]any)
	if child == nil {
		child = map[string]any{}
		root[parts[0]] = child
	}
	assignNested(child, parts[1:], value)
}

func requestBindingValue(value string) any {
	value = strings.TrimSpace(value)
	if value == "" || !looksLikeRuntimeExpression(value) {
		return value
	}
	value = runtimeInputExecutionExpression(value)
	return map[string]any{"$expr": value}
}

func looksLikeRuntimeExpression(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, prefix := range []string{
		"inputs.", "input.", "json.", "nodes.", "node_items.", "node_binaries.",
		"variables.inputs.",
		"current_ref.", "input_refs.", "current_lineage.", "lineage.",
		"workflow.", "execution.", "env.", "input_item.",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	if value == "run_index" {
		return true
	}
	return strings.Contains(value, ".received_body") ||
		strings.Contains(value, ".received_raw") ||
		strings.Contains(value, ".received_status_code") ||
		strings.Contains(value, ".received_response_header")
}

type requestFieldPlacement struct {
	Original string
	Section  string
	Name     string
}

var errAPISourceOperationNotFound = errors.New("api source operation not found")

type requestBindingMapper struct {
	exampleDir string
	cache      map[string]map[string]map[string]requestFieldPlacement
	ambiguous  map[string]map[string]map[string]bool
}

func newRequestBindingMapper(exampleDir string) *requestBindingMapper {
	return &requestBindingMapper{
		exampleDir: exampleDir,
		cache:      map[string]map[string]map[string]requestFieldPlacement{},
		ambiguous:  map[string]map[string]map[string]bool{},
	}
}

func (mapper *requestBindingMapper) lookup(openAPIPath, operationID, field string) (requestFieldPlacement, error) {
	openAPIPath = strings.TrimSpace(openAPIPath)
	operationID = strings.TrimSpace(operationID)
	field = strings.TrimSpace(field)
	if openAPIPath == "" || operationID == "" {
		return requestFieldPlacement{}, fmt.Errorf("cannot infer %q without API source and operationId", field)
	}
	operations, err := mapper.load(openAPIPath)
	if err != nil {
		return requestFieldPlacement{}, err
	}
	fields, ok := operations[operationID]
	if !ok {
		return requestFieldPlacement{}, fmt.Errorf("%w: source path %q does not declare operationId %q; available operationIds: %s", errAPISourceOperationNotFound, openAPIPath, operationID, formatKnownValues(sortedRequestOperationIDs(operations)))
	}
	placement, ok := fields[field]
	if !ok {
		if mapper.isAmbiguous(openAPIPath, operationID, field) {
			return requestFieldPlacement{}, fmt.Errorf("ambiguous request field %q for source path %q operationId %q; qualify fields as path.<name>, query.<name>, header.<name>, or body.<name>; known request fields: %s", field, openAPIPath, operationID, formatKnownValues(knownRequestFields(fields)))
		}
		return requestFieldPlacement{}, fmt.Errorf("request field %q is not declared by source path %q operationId %q; known request fields: %s", field, openAPIPath, operationID, formatKnownValues(knownRequestFields(fields)))
	}
	return placement, nil
}

func (mapper *requestBindingMapper) isAmbiguous(openAPIPath, operationID, field string) bool {
	if mapper == nil || mapper.ambiguous == nil {
		return false
	}
	sourceAmbiguous := mapper.ambiguous[filepath.ToSlash(strings.TrimSpace(openAPIPath))]
	if sourceAmbiguous == nil {
		return false
	}
	operationAmbiguous := sourceAmbiguous[strings.TrimSpace(operationID)]
	if operationAmbiguous == nil {
		return false
	}
	return operationAmbiguous[strings.TrimSpace(field)]
}

func (mapper *requestBindingMapper) load(openAPIPath string) (map[string]map[string]requestFieldPlacement, error) {
	if mapper == nil {
		return nil, fmt.Errorf("API source request metadata is unavailable")
	}
	key := filepath.ToSlash(strings.TrimSpace(openAPIPath))
	if cached, ok := mapper.cache[key]; ok {
		return cached, nil
	}
	path := key
	if !filepath.IsAbs(path) {
		path = filepath.Join(mapper.exampleDir, filepath.FromSlash(path))
	}
	if sourceDescriptionTypeForPath(key) != uws1.SourceDescriptionTypeOpenAPI {
		operations, err := mapper.loadNative(path, key, sourceDescriptionTypeForPath(key))
		if err != nil {
			return nil, err
		}
		mapper.cache[key] = operations
		return operations, nil
	}
	index, err := apitools.LoadOperationIndex(path)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI request metadata %s: %w", openAPIPath, err)
	}
	operations := map[string]map[string]requestFieldPlacement{}
	ambiguous := map[string]map[string]bool{}
	for _, operation := range index.OperationIDs {
		fields, blocked, err := requestFieldPlacementMetadata(operation)
		if err != nil {
			return nil, fmt.Errorf("operation %s: %w", operation.OperationID, err)
		}
		operations[operation.OperationID] = fields
		ambiguous[operation.OperationID] = blocked
	}
	mapper.cache[key] = operations
	if mapper.ambiguous == nil {
		mapper.ambiguous = map[string]map[string]map[string]bool{}
	}
	mapper.ambiguous[key] = ambiguous
	return operations, nil
}

func (mapper *requestBindingMapper) loadNative(path, sourceRef string, sourceType uws1.SourceDescriptionType) (map[string]map[string]requestFieldPlacement, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load API source request metadata %s: %w", sourceRef, err)
	}
	operations := map[string]map[string]requestFieldPlacement{}
	ambiguous := map[string]map[string]bool{}
	add := func(ids []string, operation apitools.OperationSummary) error {
		fields, blocked, err := requestFieldPlacementMetadata(operation)
		if err != nil {
			return fmt.Errorf("operation %s: %w", operation.OperationID, err)
		}
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				operations[id] = fields
				ambiguous[id] = blocked
			}
		}
		return nil
	}
	switch sourceType {
	case uws1.SourceDescriptionTypeGoogleDiscovery:
		model, parseErr := googlediscovery.Parse(data)
		if parseErr != nil {
			return nil, fmt.Errorf("load Google Discovery request metadata %s: %w", sourceRef, parseErr)
		}
		for _, op := range model.Operations {
			if op == nil {
				continue
			}
			summary := apitools.OperationSummary{
				OperationID: op.OperationID,
				Parameters:  discoveryRequestParameters(op.Parameters),
			}
			if op.RequestRef != "" || op.RequestMediaType != "" {
				summary.RequestBody = googleDiscoveryRequestBodySummary(model, op)
			}
			if err := add(operationIDAliases(op.OperationID, op.ID, op.Name), summary); err != nil {
				return nil, err
			}
		}
	case uws1.SourceDescriptionTypeAWSSmithy:
		model, parseErr := awssmithy.Parse(data)
		if parseErr != nil {
			return nil, fmt.Errorf("load AWS Smithy request metadata %s: %w", sourceRef, parseErr)
		}
		for _, op := range model.Operations {
			if op == nil {
				continue
			}
			summary := smithyOperationSummary(op)
			if err := add(operationIDAliases(op.Name, op.ID), summary); err != nil {
				return nil, err
			}
		}
	case uws1.SourceDescriptionTypeAsyncAPI:
		doc, parseErr := parseAsyncAPIDocument(data)
		if parseErr != nil {
			return nil, fmt.Errorf("load AsyncAPI request metadata %s: %w", sourceRef, parseErr)
		}
		for _, op := range asyncAPIOperationSummaries(sourceRef, doc) {
			if err := add([]string{op.OperationID}, op); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("unsupported API source type %q for %s", sourceType, sourceRef)
	}
	if mapper.ambiguous == nil {
		mapper.ambiguous = map[string]map[string]map[string]bool{}
	}
	mapper.ambiguous[filepath.ToSlash(strings.TrimSpace(sourceRef))] = ambiguous
	return operations, nil
}

func discoveryRequestParameters(params []*googlediscovery.Parameter) []apitools.ParameterSummary {
	var out []apitools.ParameterSummary
	for _, param := range params {
		if param == nil {
			continue
		}
		out = append(out, apitools.ParameterSummary{
			Name: param.Name,
			In:   param.Location,
			Type: stringFromAnyMap(param.Schema, "type"),
		})
	}
	return out
}

func googleDiscoveryRequestBodySummary(model *googlediscovery.Model, op *googlediscovery.Operation) *apitools.RequestBodySummary {
	if op == nil {
		return nil
	}
	body := &apitools.RequestBodySummary{
		Required:     strings.TrimSpace(op.RequestRef) != "",
		Ref:          op.RequestRef,
		ContentTypes: []string{firstNonEmpty(op.RequestMediaType, "application/json")},
	}
	if model == nil || strings.TrimSpace(op.RequestRef) == "" {
		return body
	}
	schema := model.Schemas[strings.TrimPrefix(op.RequestRef, "#/components/schemas/")]
	if len(schema) == 0 {
		return body
	}
	body.Description = asString(schema["description"])
	body.Fields = googleDiscoveryRequestFields(schema, op.OperationID)
	for _, field := range body.Fields {
		if field.Required {
			body.RequiredFieldPaths = append(body.RequiredFieldPaths, field.Path)
		}
	}
	return body
}

func googleDiscoveryRequestFields(schema map[string]any, operationID string) []apitools.RequestFieldSummary {
	props := asMap(schema["properties"])
	if len(props) == 0 {
		return nil
	}
	required := stringSet(asStringSlice(schema["required"]))
	var out []apitools.RequestFieldSummary
	for _, name := range sortedMapKeys(props) {
		prop := asMap(props[name])
		if len(prop) == 0 {
			continue
		}
		out = append(out, apitools.RequestFieldSummary{
			Path:        name,
			Required:    required[name] || googleDiscoveryPropertyRequiredForOperation(prop, operationID),
			Type:        asString(prop["type"]),
			Format:      asString(prop["format"]),
			Ref:         asString(prop["$ref"]),
			Description: asString(prop["description"]),
		})
	}
	return out
}

func googleDiscoveryPropertyRequiredForOperation(prop map[string]any, operationID string) bool {
	annotations := asMap(prop["annotations"])
	for _, required := range asStringSlice(annotations["required"]) {
		if googleDiscoveryOperationIDMatches(required, operationID) {
			return true
		}
	}
	return false
}

func googleDiscoveryOperationIDMatches(candidate, operationID string) bool {
	candidate = strings.TrimSpace(candidate)
	operationID = strings.TrimSpace(operationID)
	if candidate == "" || operationID == "" {
		return false
	}
	return candidate == operationID || operationIDAlias(candidate) == operationIDAlias(operationID)
}

func asStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(asString(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out[strings.TrimSpace(value)] = true
		}
	}
	return out
}

func stringFromAnyMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func requestFieldPlacements(operation apitools.OperationSummary) (map[string]requestFieldPlacement, error) {
	fields, _, err := requestFieldPlacementMetadata(operation)
	return fields, err
}

func requestFieldPlacementMetadata(operation apitools.OperationSummary) (map[string]requestFieldPlacement, map[string]bool, error) {
	out := map[string]requestFieldPlacement{}
	blocked := map[string]bool{}
	add := func(key, section, name string) error {
		key = strings.TrimSpace(key)
		section = strings.TrimSpace(section)
		name = strings.TrimSpace(name)
		if key == "" || section == "" || name == "" {
			return nil
		}
		placement := requestFieldPlacement{Original: key, Section: section, Name: name}
		qualified := section + "." + name
		if qualified != "body.body" {
			out[qualified] = requestFieldPlacement{Original: qualified, Section: section, Name: name}
		}
		if blocked[key] {
			return nil
		}
		if existing, ok := out[key]; ok && (existing.Section != placement.Section || existing.Name != placement.Name) {
			delete(out, key)
			blocked[key] = true
			return nil
		}
		out[key] = placement
		if alias := camelToSnake(key); alias != key {
			if blocked[alias] {
				return nil
			}
			if existing, ok := out[alias]; ok && (existing.Section != placement.Section || existing.Name != placement.Name) {
				delete(out, alias)
				blocked[alias] = true
				return nil
			}
			out[alias] = requestFieldPlacement{Original: alias, Section: section, Name: name}
		}
		return nil
	}
	for _, parameter := range operation.Parameters {
		section := strings.TrimSpace(parameter.In)
		if !isStandardRequestSection(section) || section == "body" {
			continue
		}
		if err := add(parameter.Name, section, parameter.Name); err != nil {
			return nil, nil, err
		}
	}
	if operation.RequestBody != nil {
		if err := add("body", "body", "body"); err != nil {
			return nil, nil, err
		}
		for _, field := range operation.RequestBody.Fields {
			if err := add(field.Path, "body", field.Path); err != nil {
				return nil, nil, err
			}
		}
	}
	for _, security := range operation.Security {
		section, target, ok := securityRequestPlacement(security)
		if !ok {
			continue
		}
		for _, alias := range []string{security.Name, security.ParameterName, apitools.SecurityCredentialFieldName(security)} {
			if err := add(alias, section, target); err != nil {
				return nil, nil, err
			}
		}
	}
	return out, blocked, nil
}

func knownRequestFields(fields map[string]requestFieldPlacement) []string {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedRequestOperationIDs(operations map[string]map[string]requestFieldPlacement) []string {
	if len(operations) == 0 {
		return nil
	}
	keys := make([]string, 0, len(operations))
	for key := range operations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func formatKnownValues(values []string) string {
	values = uniqueStrings(values)
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func securityRequestPlacement(security apitools.SecuritySummary) (string, string, bool) {
	section := strings.TrimSpace(security.In)
	target := firstNonEmpty(security.ParameterName, security.Name)
	if isStandardRequestSection(section) && section != "body" && target != "" {
		return section, target, true
	}
	if securitySummaryUsesBearerAuthorization(security) {
		return "header", "Authorization", true
	}
	return "", "", false
}

func securitySummaryUsesBearerAuthorization(security apitools.SecuritySummary) bool {
	if strings.EqualFold(strings.TrimSpace(security.Type), "http") {
		return true
	}
	scheme := strings.ToLower(strings.TrimSpace(security.Scheme))
	name := strings.ToLower(strings.TrimSpace(firstNonEmpty(security.Name, security.ParameterName)))
	return scheme == "bearer" || strings.Contains(scheme, "bearer") || strings.Contains(name, "bearer")
}

func camelToSnake(value string) string {
	var b strings.Builder
	for i, r := range value {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func addTriggers(doc *uws1.Document, triggers []*rollout.TriggerIntent) {
	for _, trigger := range triggers {
		if trigger == nil || strings.TrimSpace(trigger.Name) == "" {
			continue
		}
		routes := make([]*uws1.TriggerRoute, 0, len(trigger.Routes))
		for _, route := range trigger.Routes {
			if route == nil {
				continue
			}
			routes = append(routes, &uws1.TriggerRoute{
				TriggerRouteFields: uws1.TriggerRouteFields{
					Output: route.Output,
					To:     uniqueStrings(route.To),
				},
			})
		}
		outputs := uniqueStrings(trigger.Outputs)
		if len(outputs) == 0 && len(routes) > 0 {
			for _, route := range routes {
				outputs = append(outputs, route.Output)
			}
			outputs = uniqueStrings(outputs)
		}
		doc.Triggers = append(doc.Triggers, &uws1.Trigger{
			TriggerID: trigger.Name,
			TriggerFields: uws1.TriggerFields{
				Path:           trigger.Path,
				Authentication: trigger.Authentication,
				Methods:        trigger.Methods,
			},
			Options: stringMapToAny(trigger.Options),
			Outputs: outputs,
			Routes:  routes,
		})
	}
}

func stringMapToAny(values map[string]string) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for _, key := range sortedStringMapKeys(values) {
		if strings.TrimSpace(key) != "" {
			out[key] = values[key]
		}
	}
	return out
}

func uwsVersionForIntent(intent *rollout.Intent) string {
	if intentRequiresUWS13(intent) {
		return "1.3.0"
	}
	if intentRequiresUWS12(intent) {
		return "1.2.0"
	}
	if intentRequiresUWS11(intent) {
		return "1.1.0"
	}
	return "1.0.0"
}

func intentRequiresUWS13(intent *rollout.Intent) bool {
	if intent == nil {
		return false
	}
	if sourceDescriptionTypeForPath(firstNonEmpty(intent.Source, intent.OpenAPI)) == uws1.SourceDescriptionTypeAsyncAPI {
		return true
	}
	requires := false
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && !requires && sourceDescriptionTypeForPath(firstNonEmpty(step.Source, step.OpenAPI)) == uws1.SourceDescriptionTypeAsyncAPI {
			requires = true
		}
	})
	return requires
}

func intentRequiresUWS12(intent *rollout.Intent) bool {
	if intent == nil {
		return false
	}
	if sourceDescriptionTypeForPath(firstNonEmpty(intent.Source, intent.OpenAPI)) != uws1.SourceDescriptionTypeOpenAPI {
		return true
	}
	requires := false
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && !requires && sourceDescriptionTypeForPath(firstNonEmpty(step.Source, step.OpenAPI)) != uws1.SourceDescriptionTypeOpenAPI {
			requires = true
		}
	})
	return requires
}

func intentRequiresUWS11(intent *rollout.Intent) bool {
	if intent == nil {
		return false
	}
	if intent.Workflow != nil && (intent.Workflow.Timeout != nil || intent.Workflow.Idempotency != nil) {
		return true
	}
	var walk func([]*rollout.Step) bool
	walk = func(steps []*rollout.Step) bool {
		for _, step := range steps {
			if step == nil {
				continue
			}
			if step.Timeout != nil {
				return true
			}
			if walk(step.Steps) {
				return true
			}
			for _, branch := range step.Cases {
				if branch != nil && walk(branch.Steps) {
					return true
				}
			}
			if step.Default != nil && walk(step.Default.Steps) {
				return true
			}
		}
		return false
	}
	return walk(intent.Steps)
}

func writeWorkflowHCL(result Result, doc *uws1.Document, intent *rollout.Intent) error {
	data, err := convert.MarshalHCL(doc)
	if err != nil {
		return err
	}
	compat := workflowCompatibilityComments(intent)
	if len(compat) > 0 {
		data = append(append(compat, '\n'), data...)
	}
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	return os.WriteFile(result.WorkflowPath, data, 0o644)
}

func workflowCompatibilityComments(intent *rollout.Intent) []byte {
	if intent == nil {
		return nil
	}
	var lines []string
	if strings.TrimSpace(intent.OpenAPI) != "" {
		lines = append(lines, fmt.Sprintf("# openapi = %q", intent.OpenAPI))
	}
	var walk func([]*rollout.Step)
	walk = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			kind := strings.ToLower(strings.TrimSpace(step.Type))
			if kind == "" {
				kind = "fnct"
			}
			lines = append(lines, fmt.Sprintf("# %s %q", kind, step.Name))
			if step.Items != "" {
				lines = append(lines, "# items = "+step.Items)
			}
			if step.BatchSize != "" {
				lines = append(lines, fmt.Sprintf("# batch_size = %q", step.BatchSize))
			}
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(intent.Steps)
	if len(lines) == 0 {
		return nil
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func addStructuralResultsFromIntent(doc *uws1.Document, intent *rollout.Intent) {
	if doc == nil || intent == nil {
		return
	}
	for _, result := range structuralPlanResults(intent) {
		if structuralResultExists(doc.Results, result.Name) {
			continue
		}
		doc.Results = append(doc.Results, &uws1.StructuralResult{
			Name:  result.Name,
			Kind:  result.Kind,
			From:  result.From,
			Value: result.Value,
		})
	}
}

func structuralResultExists(results []*uws1.StructuralResult, name string) bool {
	name = strings.TrimSpace(name)
	for _, result := range results {
		if result != nil && strings.TrimSpace(result.Name) == name {
			return true
		}
	}
	return false
}

func pruneEmptyUWSStepTypes(data []byte) ([]byte, error) {
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	pruneEmptyTypeFields(value)
	return yaml.Marshal(value)
}

func pruneEmptyTypeFields(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if raw, ok := typed["type"]; ok && strings.TrimSpace(fmt.Sprint(raw)) == "" {
			delete(typed, "type")
		}
		for _, child := range typed {
			pruneEmptyTypeFields(child)
		}
	case []any:
		for _, child := range typed {
			pruneEmptyTypeFields(child)
		}
	}
}

func normalizeUWSStepsForSchema(doc *uws1.Document) {
	if doc == nil {
		return
	}
	operationIDs := make(map[string]bool, len(doc.Operations))
	for _, op := range doc.Operations {
		if op != nil && strings.TrimSpace(op.OperationID) != "" {
			operationIDs[strings.TrimSpace(op.OperationID)] = true
		}
	}
	for _, workflow := range doc.Workflows {
		if workflow == nil {
			continue
		}
		normalizeUWSStepList(workflow.Steps, operationIDs)
	}
}

func normalizeUWSStepList(steps []*uws1.Step, operationIDs map[string]bool) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.OperationRef) == "" && !isUWSStructuralStepType(step.Type) && operationIDs[strings.TrimSpace(step.StepID)] {
			step.OperationRef = strings.TrimSpace(step.StepID)
		}
		if strings.TrimSpace(step.OperationRef) != "" && !isUWSStructuralStepType(step.Type) {
			step.Type = ""
		}
		normalizeUWSStepList(step.Steps, operationIDs)
		for _, branch := range step.Cases {
			if branch != nil {
				normalizeUWSStepList(branch.Steps, operationIDs)
			}
		}
		normalizeUWSStepList(step.Default, operationIDs)
	}
}

func isUWSStructuralStepType(value string) bool {
	switch strings.TrimSpace(value) {
	case "", uws1.WorkflowTypeSequence, uws1.WorkflowTypeParallel, uws1.WorkflowTypeSwitch,
		uws1.WorkflowTypeMerge, uws1.WorkflowTypeLoop, uws1.WorkflowTypeAwait:
		return true
	default:
		return false
	}
}

func isIntentStructuralType(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "sequence", "parallel", "switch", "merge", "loop", "await":
		return true
	default:
		return false
	}
}

func uwsWorkflowType(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "parallel":
		return uws1.WorkflowTypeParallel
	case "switch":
		return uws1.WorkflowTypeSwitch
	case "merge":
		return uws1.WorkflowTypeMerge
	case "loop":
		return uws1.WorkflowTypeLoop
	case "await":
		return uws1.WorkflowTypeAwait
	default:
		return uws1.WorkflowTypeSequence
	}
}

func sourceDescriptionName(path string, existing map[string]string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	name = sanitizeIdentifier(name)
	if name == "" {
		name = "openapi"
	}
	used := map[string]bool{}
	for _, value := range existing {
		used[value] = true
	}
	if !used[name] {
		return name
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", name, i)
		if !used[candidate] {
			return candidate
		}
	}
}

func sanitizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	value = regexp.MustCompile(`[^A-Za-z0-9_]+`).ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return ""
	}
	if value[0] >= '0' && value[0] <= '9' {
		value = "_" + value
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func sortedStringMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
