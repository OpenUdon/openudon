package synthesize

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/asyncapi"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func asyncAPIOperationInfoIndex(path string) (map[string]*rollout.OperationInfo, error) {
	out := map[string]*rollout.OperationInfo{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("read AsyncAPI %s: %w", path, err)
	}
	doc, err := asyncapi.Parse(data)
	if err != nil {
		return out, fmt.Errorf("parse AsyncAPI %s: %w", path, err)
	}
	for _, summary := range apitools.AsyncAPIOperationSummaries("", doc) {
		info := &rollout.OperationInfo{
			OperationID: summary.OperationID,
			Method:      summary.Method,
			Path:        summary.Path,
			Summary:     summary.Summary,
			Description: summary.Description,
		}
		out[summary.OperationID] = info
		out["#/operations/"+escapeAsyncAPIJSONPointerToken(summary.OperationID)] = info
	}
	for selector := range doc.SelectorAliases() {
		if _, exists := out[selector]; exists || !strings.HasPrefix(selector, "#/") {
			continue
		}
		if target, ok := doc.ResolveSelector(selector); ok && target.Kind != "operation" {
			out[selector] = &rollout.OperationInfo{
				OperationID: selector,
				Path:        selector,
			}
		}
	}
	return out, nil
}

func escapeAsyncAPIJSONPointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return token
}
