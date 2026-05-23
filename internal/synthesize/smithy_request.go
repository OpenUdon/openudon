package synthesize

import (
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/awssmithy"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func smithyOperationSummary(op *awssmithy.Operation) apitools.OperationSummary {
	if op == nil {
		return apitools.OperationSummary{}
	}
	summary := apitools.OperationSummary{
		OperationID: op.Name,
		Method:      op.Method,
		Path:        op.URI,
		Parameters:  smithyRequestParameters(op),
	}
	if body := smithyRequestBodySummary(op); body != nil {
		summary.RequestBody = body
	}
	return summary
}

func smithyRequestParameters(op *awssmithy.Operation) []apitools.ParameterSummary {
	if op == nil {
		return nil
	}
	var out []apitools.ParameterSummary
	for _, binding := range op.InputBindings {
		location, ok := smithyNonBodyLocation(binding)
		if !ok {
			continue
		}
		out = append(out, apitools.ParameterSummary{
			Name:     smithyBindingRequestName(binding),
			In:       location,
			Required: binding.Required,
		})
	}
	return out
}

func smithyRequestBodySummary(op *awssmithy.Operation) *apitools.RequestBodySummary {
	if op == nil || (op.Payload == nil && len(op.UnboundInput) == 0 && len(op.StaticPayload) == 0) {
		return nil
	}
	body := &apitools.RequestBodySummary{
		ContentTypes: []string{firstNonEmpty(op.RequestMediaType, "application/json")},
		Ref:          op.Input,
	}
	if op.Payload != nil {
		field := smithyRequestBodyField(op.Payload)
		body.Fields = append(body.Fields, field)
		body.Required = body.Required || field.Required
	}
	for _, binding := range op.UnboundInput {
		field := smithyRequestBodyField(binding)
		if strings.TrimSpace(field.Path) == "" {
			continue
		}
		body.Fields = append(body.Fields, field)
		body.Required = body.Required || field.Required
	}
	for _, field := range body.Fields {
		if field.Required {
			body.RequiredFieldPaths = append(body.RequiredFieldPaths, field.Path)
		}
	}
	body.RequiredFieldPaths = sortedUnique(body.RequiredFieldPaths)
	return body
}

func smithyRequestBodyField(binding *awssmithy.MemberBinding) apitools.RequestFieldSummary {
	if binding == nil {
		return apitools.RequestFieldSummary{}
	}
	return apitools.RequestFieldSummary{
		Path:     smithyBindingMemberName(binding),
		Required: binding.Required,
		Ref:      binding.Target,
	}
}

func smithyOperationParameters(op *awssmithy.Operation) []*rollout.ParameterInfo {
	if op == nil {
		return nil
	}
	var out []*rollout.ParameterInfo
	for _, binding := range op.InputBindings {
		location, ok := smithyNonBodyLocation(binding)
		if !ok {
			continue
		}
		out = append(out, &rollout.ParameterInfo{
			Name:     smithyBindingRequestName(binding),
			In:       location,
			Required: binding.Required,
		})
	}
	for _, binding := range smithyBodyBindings(op) {
		if binding == nil || strings.TrimSpace(smithyBindingMemberName(binding)) == "" {
			continue
		}
		out = append(out, &rollout.ParameterInfo{
			Name:     smithyBindingMemberName(binding),
			In:       "body",
			Required: binding.Required,
		})
	}
	return out
}

func smithyBodyBindings(op *awssmithy.Operation) []*awssmithy.MemberBinding {
	if op == nil {
		return nil
	}
	var out []*awssmithy.MemberBinding
	if op.Payload != nil {
		out = append(out, op.Payload)
	}
	out = append(out, op.UnboundInput...)
	return out
}

func smithyNonBodyLocation(binding *awssmithy.MemberBinding) (string, bool) {
	if binding == nil {
		return "", false
	}
	switch strings.TrimSpace(binding.Location) {
	case "label", "path":
		return "path", true
	case "query", "queryParams":
		return "query", true
	case "header", "prefixHeaders":
		return "header", true
	default:
		return "", false
	}
}

func smithyBindingRequestName(binding *awssmithy.MemberBinding) string {
	if binding == nil {
		return ""
	}
	return firstNonEmpty(binding.WireName, binding.MemberName)
}

func smithyBindingMemberName(binding *awssmithy.MemberBinding) string {
	if binding == nil {
		return ""
	}
	return strings.TrimSpace(binding.MemberName)
}
