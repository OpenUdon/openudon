package elicitor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
)

const (
	maxRequestBodyFieldDepth = 6
	maxRequestBodyFields     = 60
)

type APIDocument struct {
	Path         string
	RelativePath string
	Title        string
	Description  string
	Operations   []*rollout.OperationInfo
}

type operationPromptContext struct {
	OperationID    string                    `json:"operationId"`
	Method         string                    `json:"method"`
	Path           string                    `json:"path"`
	Summary        string                    `json:"summary,omitempty"`
	Description    string                    `json:"description,omitempty"`
	RequiredFields []string                  `json:"required_fields,omitempty"`
	Parameters     []parameterPromptContext  `json:"parameters,omitempty"`
	RequestBody    *requestBodyPromptContext `json:"request_body,omitempty"`
	Security       securityPromptContext     `json:"security,omitempty"`
}

type parameterPromptContext struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type requestBodyPromptContext struct {
	Required           bool                      `json:"required"`
	ContentType        string                    `json:"content_type,omitempty"`
	Type               string                    `json:"type,omitempty"`
	Description        string                    `json:"description,omitempty"`
	Default            any                       `json:"default,omitempty"`
	Example            any                       `json:"example,omitempty"`
	Enum               []any                     `json:"enum,omitempty"`
	Fields             []requestBodyFieldContext `json:"fields,omitempty"`
	RequiredFieldPaths []string                  `json:"required_field_paths,omitempty"`
}

type requestBodyFieldContext struct {
	Path        string `json:"path"`
	Required    bool   `json:"required"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
	Example     any    `json:"example,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
}

type securityPromptContext struct {
	Schemes          []string `json:"schemes,omitempty"`
	CredentialFields []string `json:"credential_fields,omitempty"`
}

func DiscoverLocalAPIs(exampleDir, projectText string) ([]APIDocument, error) {
	openAPIDir := filepath.Join(exampleDir, "openapi")
	if _, err := os.Stat(openAPIDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	candidates, err := openapidisco.LocalFiles(openAPIDir, exampleDir, projectText)
	if err != nil {
		return nil, err
	}
	var docs []APIDocument
	for _, candidate := range candidates {
		spec, err := rollout.LoadOpenAPISpec(candidate.Path)
		if err != nil {
			continue
		}
		doc := APIDocument{
			Path:         candidate.Path,
			RelativePath: candidate.RelativePath,
			Title:        firstNonEmpty(candidate.Title, spec.Title),
			Description:  firstNonEmpty(candidate.Description, spec.Description),
			Operations:   spec.Operations,
		}
		sort.Slice(doc.Operations, func(i, j int) bool {
			return operationLabel(doc.Operations[i]) < operationLabel(doc.Operations[j])
		})
		docs = append(docs, doc)
	}
	sort.Slice(docs, func(i, j int) bool {
		return docs[i].RelativePath < docs[j].RelativePath
	})
	return docs, nil
}

func operationLabel(op *rollout.OperationInfo) string {
	if op == nil {
		return ""
	}
	id := op.OperationID
	if id == "" {
		id = strings.ToLower(op.Method) + "_" + strings.Trim(strings.ReplaceAll(op.Path, "/", "_"), "_")
	}
	text := fmt.Sprintf("%s %s %s", op.Method, op.Path, id)
	if op.Summary != "" {
		text += " - " + op.Summary
	}
	return text
}

func operationPrompt(op *rollout.OperationInfo) operationPromptContext {
	if op == nil {
		return operationPromptContext{}
	}
	return operationPromptContext{
		OperationID:    op.OperationID,
		Method:         op.Method,
		Path:           op.Path,
		Summary:        op.Summary,
		Description:    op.Description,
		RequiredFields: requiredFields(op),
		Parameters:     parameterPromptContexts(op.Parameters),
		RequestBody:    requestBodyPrompt(op.RequestBody),
		Security:       securityPrompt(op.Security),
	}
}

func parameterPromptContexts(parameters []*rollout.ParameterInfo) []parameterPromptContext {
	if len(parameters) == 0 {
		return nil
	}
	out := make([]parameterPromptContext, 0, len(parameters))
	for _, parameter := range parameters {
		if parameter == nil {
			continue
		}
		out = append(out, parameterPromptContext{
			Name:        parameter.Name,
			Location:    parameter.In,
			Required:    parameter.Required,
			Type:        parameter.Type,
			Description: parameter.Description,
		})
	}
	return out
}

func requestBodyPrompt(body *rollout.RequestBodyInfo) *requestBodyPromptContext {
	if body == nil {
		return nil
	}
	schema := asSchemaMap(body.Schema)
	out := &requestBodyPromptContext{
		Required:    body.Required,
		ContentType: body.ContentType,
		Type:        schemaType(schema),
		Description: schemaString(schema, "description"),
		Default:     schemaValue(schema, "default"),
		Example:     schemaValue(schema, "example"),
		Enum:        schemaEnum(schema),
		Fields:      flattenRequestBodyFields(body),
	}
	for _, field := range out.Fields {
		if field.Required {
			out.RequiredFieldPaths = append(out.RequiredFieldPaths, field.Path)
		}
	}
	return out
}

func flattenRequestBodyFields(body *rollout.RequestBodyInfo) []requestBodyFieldContext {
	if body == nil {
		return nil
	}
	var out []requestBodyFieldContext
	schema := asSchemaMap(body.Schema)
	flattenSchemaFields(schema, "", body.Required, 0, &out)
	if len(out) == 0 {
		out = append(out, requestBodyField("body", body.Required, schema))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	if len(out) > maxRequestBodyFields {
		out = out[:maxRequestBodyFields]
	}
	return out
}

func flattenSchemaFields(schema map[string]any, path string, required bool, depth int, out *[]requestBodyFieldContext) {
	if len(*out) >= maxRequestBodyFields || depth > maxRequestBodyFieldDepth {
		return
	}
	if len(schema) == 0 {
		if path != "" {
			*out = append(*out, requestBodyField(path, required, nil))
		}
		return
	}
	if path != "" {
		*out = append(*out, requestBodyField(path, required, schema))
		if len(*out) >= maxRequestBodyFields || depth == maxRequestBodyFieldDepth {
			return
		}
	}
	properties := schemaProperties(schema)
	if len(properties) > 0 {
		requiredNames := requiredNameSet(schema)
		names := make([]string, 0, len(properties))
		for name := range properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			childPath := name
			if path != "" {
				childPath = path + "." + name
			}
			flattenSchemaFields(asSchemaMap(properties[name]), childPath, required && requiredNames[name], depth+1, out)
			if len(*out) >= maxRequestBodyFields {
				return
			}
		}
		return
	}
	if items := asSchemaMap(schema["items"]); len(items) > 0 {
		itemPath := "body[]"
		if path != "" {
			itemPath = path + "[]"
		}
		flattenSchemaFields(items, itemPath, required, depth+1, out)
	}
}

func requestBodyField(path string, required bool, schema map[string]any) requestBodyFieldContext {
	return requestBodyFieldContext{
		Path:        path,
		Required:    required,
		Type:        schemaType(schema),
		Description: schemaString(schema, "description"),
		Default:     schemaValue(schema, "default"),
		Example:     schemaValue(schema, "example"),
		Enum:        schemaEnum(schema),
	}
}

func schemaProperties(schema map[string]any) map[string]any {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) > 0 {
		return properties
	}
	return nil
}

func requiredNameSet(schema map[string]any) map[string]bool {
	out := map[string]bool{}
	for _, name := range stringSliceFromAny(schema["required"]) {
		out[name] = true
	}
	return out
}

func schemaType(schema map[string]any) string {
	if value := schemaString(schema, "type"); value != "" {
		return value
	}
	if len(schemaProperties(schema)) > 0 {
		return "object"
	}
	if items := asSchemaMap(schema["items"]); len(items) > 0 {
		return "array"
	}
	return ""
}

func schemaString(schema map[string]any, key string) string {
	if schema == nil {
		return ""
	}
	value, _ := schema[key].(string)
	return strings.TrimSpace(value)
}

func schemaValue(schema map[string]any, key string) any {
	if schema == nil {
		return nil
	}
	return schema[key]
}

func schemaEnum(schema map[string]any) []any {
	if schema == nil {
		return nil
	}
	switch typed := schema["enum"].(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, value := range typed {
			out = append(out, value)
		}
		return out
	default:
		return nil
	}
}

func asSchemaMap(value any) map[string]any {
	switch typed := value.(type) {
	case nil:
		return nil
	case map[string]any:
		return typed
	default:
		return nil
	}
}

func securityPrompt(security []string) securityPromptContext {
	if len(security) == 0 {
		return securityPromptContext{}
	}
	schemes := append([]string(nil), security...)
	sort.Strings(schemes)
	var fields []string
	for _, scheme := range schemes {
		if field := securityFieldName(scheme); field != "" {
			fields = append(fields, field)
		}
	}
	return securityPromptContext{
		Schemes:          dedupeStrings(schemes),
		CredentialFields: dedupeStrings(fields),
	}
}

func requiredFields(op *rollout.OperationInfo) []string {
	if op == nil {
		return nil
	}
	var out []string
	for _, parameter := range op.Parameters {
		if parameter == nil || !parameter.Required {
			continue
		}
		out = append(out, parameter.Name)
	}
	if op.RequestBody != nil && op.RequestBody.Required {
		bodyFields := requiredRequestBodyFields(op.RequestBody)
		if len(bodyFields) == 0 {
			out = append(out, "body")
		} else {
			out = append(out, bodyFields...)
		}
	}
	for _, security := range op.Security {
		if field := securityFieldName(security); field != "" {
			out = append(out, field)
		}
	}
	return dedupeStrings(out)
}

func requiredRequestBodyFields(body *rollout.RequestBodyInfo) []string {
	if body == nil || len(body.Schema) == 0 {
		return nil
	}
	required := stringSliceFromAny(body.Schema["required"])
	properties, _ := body.Schema["properties"].(map[string]any)
	if len(required) == 0 {
		var fields []string
		for field := range properties {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		return fields
	}
	if len(properties) == 0 {
		return required
	}
	var out []string
	for _, field := range required {
		if _, ok := properties[field]; ok {
			out = append(out, field)
		}
	}
	return out
}

func stringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func securityFieldName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "api") && strings.Contains(lower, "key") {
		return camelToSnake(name)
	}
	if strings.Contains(lower, "bearer") || strings.Contains(lower, "auth") || strings.Contains(lower, "token") {
		return "Authorization"
	}
	return camelToSnake(name)
}

func camelToSnake(value string) string {
	var out []rune
	var prev rune
	for i, r := range value {
		if r == '-' || r == ' ' {
			r = '_'
		}
		if unicode.IsUpper(r) {
			if i > 0 && prev != '_' && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				out = append(out, '_')
			}
			r = unicode.ToLower(r)
		}
		out = append(out, r)
		prev = r
	}
	return slugIdent(string(out))
}

func operationByID(docs []APIDocument, docPath, operationID string) (*rollout.OperationInfo, bool) {
	for _, doc := range docs {
		if doc.RelativePath != docPath {
			continue
		}
		for _, op := range doc.Operations {
			if op != nil && op.OperationID == operationID {
				return op, true
			}
		}
	}
	return nil, false
}
