package workflowintent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/OpenUdon/apitools"
	"gopkg.in/yaml.v3"
)

type OpenAPISpec struct {
	Title       string
	Version     string
	Description string
	ServerURL   string
	Operations  []*OperationInfo
	Security    []string
	RawSpec     map[string]any
}

type OpenAPISpecContext struct {
	Path         string
	ResolvedPath string
	Provider     string
	StepNames    []string
	Spec         *OpenAPISpec
}

type OperationInfo struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
	Description string
	Parameters  []*ParameterInfo
	RequestBody *RequestBodyInfo
	Responses   map[string]*ResponseInfo
	Security    []string
	Tags        []string
}

type ParameterInfo struct {
	Name        string
	In          string
	Required    bool
	Type        string
	Description string
}

type RequestBodyInfo struct {
	Required    bool
	ContentType string
	Schema      map[string]any
}

type ResponseInfo struct {
	Description string
	ContentType string
	Schema      map[string]any
}

func LoadOpenAPISpec(path string) (*OpenAPISpec, error) {
	index, err := apitools.LoadOperationIndex(path)
	if err != nil {
		return nil, err
	}
	spec := &OpenAPISpec{Operations: []*OperationInfo{}}
	if len(index.Inventory.Documents) > 0 {
		doc := index.Inventory.Documents[0]
		spec.Title = doc.Title
		spec.Description = doc.Description
		spec.Version = firstNonEmpty(doc.OpenAPI, doc.Swagger)
	}
	for _, op := range index.Inventory.Operations {
		info := &OperationInfo{
			OperationID: op.OperationID,
			Method:      strings.ToUpper(op.Method),
			Path:        op.Path,
			Summary:     op.Summary,
			Description: op.Description,
			Responses:   map[string]*ResponseInfo{},
			Tags:        append([]string(nil), op.Tags...),
		}
		for _, param := range op.Parameters {
			info.Parameters = append(info.Parameters, &ParameterInfo{
				Name:        param.Name,
				In:          param.In,
				Required:    param.Required,
				Type:        param.Type,
				Description: param.Description,
			})
		}
		if op.RequestBody != nil {
			info.RequestBody = &RequestBodyInfo{
				Required:    op.RequestBody.Required,
				ContentType: firstNonEmpty(op.RequestBody.ContentTypes...),
				Schema:      schemaSummaryToMap(op.RequestBody.Schema),
			}
		}
		for _, security := range op.Security {
			if security.Name != "" {
				info.Security = append(info.Security, security.Name)
			}
		}
		spec.Operations = append(spec.Operations, info)
	}
	augmentOpenAPIResponses(path, spec)
	return spec, nil
}

func augmentOpenAPIResponses(path string, spec *OpenAPISpec) {
	if spec == nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}
	root := normalizeYAMLMap(raw)
	rootMap := asMap(root)
	if spec.ServerURL == "" {
		if servers, ok := rootMap["servers"].([]any); ok && len(servers) > 0 {
			spec.ServerURL = asString(asMap(servers[0])["url"])
		}
		if spec.ServerURL == "" {
			spec.ServerURL = asString(rootMap["host"])
		}
	}
	paths := asMap(rootMap["paths"])
	if len(paths) == 0 {
		return
	}
	byID := map[string]*OperationInfo{}
	for _, op := range spec.Operations {
		if op != nil && strings.TrimSpace(op.OperationID) != "" {
			byID[strings.TrimSpace(op.OperationID)] = op
		}
	}
	for _, rawPathItem := range paths {
		pathItem := asMap(rawPathItem)
		for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
			rawOp := asMap(pathItem[method])
			if len(rawOp) == 0 {
				continue
			}
			op := byID[asString(rawOp["operationId"])]
			if op == nil {
				continue
			}
			if op.Responses == nil {
				op.Responses = map[string]*ResponseInfo{}
			}
			for code, rawResp := range asMap(rawOp["responses"]) {
				resp := asMap(rawResp)
				info := &ResponseInfo{Description: asString(resp["description"])}
				if schema := firstResponseSchema(resp); len(schema) > 0 {
					info.Schema = schema
					info.ContentType = "application/json"
				}
				op.Responses[fmt.Sprint(code)] = info
			}
		}
	}
}

func firstResponseSchema(resp map[string]any) map[string]any {
	if schema := asMap(resp["schema"]); len(schema) > 0 {
		return schema
	}
	content := asMap(resp["content"])
	if len(content) == 0 {
		return nil
	}
	if jsonMedia := asMap(content["application/json"]); len(jsonMedia) > 0 {
		return asMap(jsonMedia["schema"])
	}
	for _, media := range content {
		if schema := asMap(asMap(media)["schema"]); len(schema) > 0 {
			return schema
		}
	}
	return nil
}

func normalizeYAMLMap(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = normalizeYAMLMap(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[fmt.Sprint(key)] = normalizeYAMLMap(child)
		}
		return out
	case []any:
		for i, child := range typed {
			typed[i] = normalizeYAMLMap(child)
		}
		return typed
	default:
		return value
	}
}

func asMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func schemaSummaryToMap(schema *apitools.SchemaSummary) map[string]any {
	if schema == nil {
		return nil
	}
	out := map[string]any{}
	if schema.Type != "" {
		out["type"] = schema.Type
	}
	if schema.Format != "" {
		out["format"] = schema.Format
	}
	if schema.Ref != "" {
		out["$ref"] = schema.Ref
	}
	if schema.Description != "" {
		out["description"] = schema.Description
	}
	if len(schema.Properties) > 0 {
		props := map[string]any{}
		for _, prop := range schema.Properties {
			props[prop.Name] = map[string]any{"type": prop.Type, "description": prop.Description}
		}
		out["properties"] = props
	}
	return out
}

func documentSourceName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = regexp.MustCompile(`[^A-Za-z0-9_]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "openapi"
	}
	return name
}
