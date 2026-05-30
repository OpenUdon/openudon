package elicitor

import (
	"testing"

	"github.com/OpenUdon/apitools"
)

func TestPromptContextFromAPIDocumentsMapsOperationAndCredentials(t *testing.T) {
	ctx := PromptContextFromAPIDocuments([]APIDocument{{
		RelativePath: "openapi/support.yaml",
		Title:        "Support API",
		Description:  "Ticket operations.",
		Operations: []apitools.OperationSummary{{
			OperationID: "getTicket",
			Method:      "GET",
			Path:        "/tickets/{ticketId}",
			Summary:     "Get a support ticket.",
			Tags:        []string{"tickets"},
			Score:       42,
			Provenance:  "openapi",
			Security: []apitools.SecuritySummary{{
				Name:        "BearerAuth",
				Type:        "http",
				Scheme:      "bearer",
				Description: "Bearer token binding.",
			}},
			RequestBody: &apitools.RequestBodySummary{
				ContentTypes:       []string{"application/json"},
				RequiredFieldPaths: []string{"comment.body"},
				Fields: []apitools.RequestFieldSummary{{
					Path:        "comment.body",
					Type:        "string",
					Required:    true,
					Description: "Comment body.",
				}},
			},
		}},
	}})

	if ctx.Version == "" || len(ctx.Sources) != 1 || ctx.Sources[0].ID != "openapi/support.yaml" || ctx.Sources[0].Kind != "openapi" {
		t.Fatalf("sources = %#v version=%q", ctx.Sources, ctx.Version)
	}
	if len(ctx.Operations) != 1 {
		t.Fatalf("operations = %#v", ctx.Operations)
	}
	op := ctx.Operations[0]
	if op.ID != "openapi/support.yaml#getTicket" || op.SourceID != "openapi/support.yaml" || op.OperationID != "getTicket" || op.Verb != "GET" || op.Confidence != "ranked" {
		t.Fatalf("operation = %#v", op)
	}
	if len(op.CredentialBindings) != 1 || op.CredentialBindings[0] != "BearerAuth" {
		t.Fatalf("operation credentials = %#v", op.CredentialBindings)
	}
	if len(ctx.Credentials) != 1 || ctx.Credentials[0].Name != "BearerAuth" || ctx.Credentials[0].Kind != "http" {
		t.Fatalf("credentials = %#v", ctx.Credentials)
	}
	if len(ctx.Schemas) != 1 || ctx.Schemas[0].ID != "openapi/support.yaml#getTicket:request" || len(ctx.Schemas[0].Fields) != 1 {
		t.Fatalf("schemas = %#v", ctx.Schemas)
	}
}
