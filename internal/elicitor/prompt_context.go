package elicitor

import (
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/authoring/promptcontext"
)

// PromptContextFromAPIDocuments translates OpenUdon/API-source metadata into
// Authoring's product-neutral prompt-safe context shape.
func PromptContextFromAPIDocuments(docs []APIDocument) promptcontext.Context {
	var ctx promptcontext.Context
	credentialIndexes := map[string]int{}
	for _, doc := range docs {
		sourceID := sourceDocumentID(doc)
		if sourceID == "" {
			continue
		}
		ctx.Sources = append(ctx.Sources, promptcontext.SourceDocument{
			ID:      sourceID,
			Kind:    sourceDocumentKind(doc),
			Title:   doc.Title,
			URI:     firstNonEmpty(doc.RelativePath, doc.Path),
			Summary: doc.Description,
		})
		for _, op := range doc.Operations {
			operationID := strings.TrimSpace(op.OperationID)
			if operationID == "" {
				continue
			}
			operationContextID := sourceID + "#" + operationID
			credentialSets := credentialBindingSetsFromSecurity(op.SecurityRequirementSets, credentialIndexes, &ctx)
			ctx.Operations = append(ctx.Operations, promptcontext.OperationCandidate{
				ID:                    operationContextID,
				SourceID:              sourceID,
				OperationID:           operationID,
				Name:                  operationLabel(op),
				Verb:                  op.Method,
				Path:                  op.Path,
				Summary:               firstNonEmpty(op.Summary, op.Description),
				CredentialBindingSets: credentialSets,
				Tags:                  append([]string(nil), op.Tags...),
				Confidence:            promptContextConfidence(op),
				Metadata: map[string]string{
					"provenance": op.Provenance,
				},
			})
			if schema := requestSchemaHint(operationContextID, op); schema.ID != "" {
				ctx.Schemas = append(ctx.Schemas, schema)
			}
		}
	}
	return promptcontext.Normalize(ctx)
}

func sourceDocumentID(doc APIDocument) string {
	return firstNonEmpty(doc.ID, doc.RelativePath, doc.Path)
}

func sourceDocumentKind(doc APIDocument) string {
	ref := strings.ToLower(strings.TrimSpace(firstNonEmpty(doc.RelativePath, doc.Path)))
	if i := strings.Index(ref, "/"); i > 0 {
		return ref[:i]
	}
	return ""
}

func credentialBindingSetsFromSecurity(sets []apitools.SecurityRequirementSetSummary, indexes map[string]int, ctx *promptcontext.Context) []promptcontext.CredentialBindingSet {
	if len(sets) == 0 {
		return nil
	}
	out := make([]promptcontext.CredentialBindingSet, 0, len(sets))
	for _, set := range sets {
		bindings := promptcontext.CredentialBindingSet{Bindings: []string{}}
		for _, item := range set.Requirements {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				continue
			}
			bindings.Bindings = append(bindings.Bindings, name)
			required := securityNameRequiredInAllAlternatives(name, sets)
			if index, ok := indexes[name]; ok {
				ctx.Credentials[index].Required = ctx.Credentials[index].Required && required
				continue
			}
			indexes[name] = len(ctx.Credentials)
			ctx.Credentials = append(ctx.Credentials, promptcontext.CredentialBinding{
				Name:     name,
				Kind:     item.Type,
				Scope:    firstNonEmpty(item.In, item.Scheme),
				Required: required,
				Summary:  item.Description,
			})
		}
		out = append(out, bindings)
	}
	return out
}

func securityNameRequiredInAllAlternatives(name string, sets []apitools.SecurityRequirementSetSummary) bool {
	if len(sets) == 0 {
		return false
	}
	for _, set := range sets {
		found := false
		for _, requirement := range set.Requirements {
			if strings.TrimSpace(requirement.Name) == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func promptContextConfidence(op apitools.OperationSummary) string {
	if op.Score > 0 {
		return "ranked"
	}
	return ""
}

func requestSchemaHint(operationContextID string, op apitools.OperationSummary) promptcontext.SchemaHint {
	if op.RequestBody == nil || len(op.RequestBody.Fields) == 0 {
		return promptcontext.SchemaHint{}
	}
	hint := promptcontext.SchemaHint{
		ID:        operationContextID + ":request",
		Name:      "request",
		Purpose:   "request",
		MediaType: strings.Join(op.RequestBody.ContentTypes, ","),
		Summary:   op.RequestBody.Description,
		Required:  append([]string(nil), op.RequestBody.RequiredFieldPaths...),
	}
	for _, field := range op.RequestBody.Fields {
		hint.Fields = append(hint.Fields, promptcontext.FieldHint{
			Name:     field.Path,
			Type:     field.Type,
			Required: field.Required,
			Summary:  field.Description,
		})
	}
	return hint
}
