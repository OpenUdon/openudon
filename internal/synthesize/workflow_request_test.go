package synthesize

import (
	"testing"

	"github.com/OpenUdon/apitools"
)

func TestRequestFieldPlacementsExposeBearerAuthorization(t *testing.T) {
	fields, err := requestFieldPlacements(apitools.OperationSummary{
		OperationID: "getCustomer",
		Security: []apitools.SecuritySummary{{
			Name:   "BearerAuth",
			Type:   "http",
			Scheme: "bearer",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	placement, ok := fields["Authorization"]
	if !ok {
		t.Fatalf("Authorization placement missing: %#v", fields)
	}
	if placement.Section != "header" || placement.Name != "Authorization" {
		t.Fatalf("Authorization placement = %#v", placement)
	}
}

func TestRequestFieldPlacementsKeepAmbiguousAliasesBlocked(t *testing.T) {
	fields, err := requestFieldPlacements(apitools.OperationSummary{
		OperationID: "updateMessage",
		Parameters: []apitools.ParameterSummary{
			{Name: "id", In: "path"},
			{Name: "id", In: "query"},
		},
		RequestBody: &apitools.RequestBodySummary{Fields: []apitools.RequestFieldSummary{{Path: "id"}}},
		Security: []apitools.SecuritySummary{{
			Name:          "id",
			Type:          "apiKey",
			In:            "header",
			ParameterName: "id",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if placement, ok := fields["id"]; ok {
		t.Fatalf("ambiguous id alias was reintroduced as %#v in %#v", placement, fields)
	}
}
