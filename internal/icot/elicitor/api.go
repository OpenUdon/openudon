package elicitor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
		out = append(out, "body")
	}
	return dedupeStrings(out)
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
