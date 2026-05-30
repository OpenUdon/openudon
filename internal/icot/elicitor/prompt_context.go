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
	credentialSeen := map[string]bool{}
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
			credentials := credentialBindingsFromSecurity(op.Security, credentialSeen, &ctx)
			ctx.Operations = append(ctx.Operations, promptcontext.OperationCandidate{
				ID:                 operationContextID,
				SourceID:           sourceID,
				OperationID:        operationID,
				Name:               operationLabel(op),
				Verb:               op.Method,
				Path:               op.Path,
				Summary:            firstNonEmpty(op.Summary, op.Description),
				CredentialBindings: credentials,
				Tags:               append([]string(nil), op.Tags...),
				Confidence:         promptContextConfidence(op),
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

func credentialBindingsFromSecurity(security []apitools.SecuritySummary, seen map[string]bool, ctx *promptcontext.Context) []string {
	var names []string
	for _, item := range security {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		if seen[name] {
			continue
		}
		seen[name] = true
		ctx.Credentials = append(ctx.Credentials, promptcontext.CredentialBinding{
			Name:     name,
			Kind:     item.Type,
			Scope:    firstNonEmpty(item.In, item.Scheme),
			Required: true,
			Summary:  item.Description,
		})
	}
	return names
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
