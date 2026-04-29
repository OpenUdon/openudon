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

type APIDocument struct {
	Path         string
	RelativePath string
	Title        string
	Description  string
	Operations   []*rollout.OperationInfo
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
