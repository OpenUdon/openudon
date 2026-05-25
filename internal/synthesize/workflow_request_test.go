package synthesize

import (
	"reflect"
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

func TestIntentRequestMapAllowsPathParameterNamedPath(t *testing.T) {
	mapper := &requestBindingMapper{cache: map[string]map[string]map[string]requestFieldPlacement{
		"openapi/github.yaml": {
			"getContent": {
				"path": {Original: "path", Section: "path", Name: "path"},
			},
		},
	}}
	got, err := intentRequestMap(map[string]string{"path": "render_backup_file.received_body.path"}, "http", "openapi/github.yaml", "getContent", mapper)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"path": map[string]any{
			"path": map[string]any{"$expr": "render_backup_file.received_body.path"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request map = %#v, want %#v", got, want)
	}
}
