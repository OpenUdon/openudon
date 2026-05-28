package synthesize

import (
	"reflect"
	"strings"
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

func TestRequestFieldPlacementsMapUWS14SourceParameters(t *testing.T) {
	fields, err := requestFieldPlacements(apitools.OperationSummary{
		OperationID: "mixed",
		Parameters: []apitools.ParameterSummary{
			{Name: "episode", In: "graphql-variable"},
			{Name: "a", In: "json-rpc"},
			{Name: "$top", In: "odata-query-option"},
			{Name: "customerId", In: "odata-parameter"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for field, want := range map[string]requestFieldPlacement{
		"episode":               {Section: "body", Name: "variables.episode"},
		"variables.episode":     {Section: "body", Name: "variables.episode"},
		"a":                     {Section: "body", Name: "params.a"},
		"params.a":              {Section: "body", Name: "params.a"},
		"$top":                  {Section: "query", Name: "$top"},
		"top":                   {Section: "query", Name: "$top"},
		"query.$top":            {Section: "query", Name: "$top"},
		"query.top":             {Section: "query", Name: "$top"},
		"customerId":            {Section: "body", Name: "parameters.customerId"},
		"parameters.customerId": {Section: "body", Name: "parameters.customerId"},
	} {
		got, ok := fields[field]
		if !ok {
			t.Fatalf("missing placement %q in %#v", field, fields)
		}
		if got.Section != want.Section || got.Name != want.Name {
			t.Fatalf("placement %q = %#v, want %#v", field, got, want)
		}
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

func TestRequestBindingLookupMissingFieldReportsKnownFields(t *testing.T) {
	mapper := &requestBindingMapper{cache: map[string]map[string]map[string]requestFieldPlacement{
		"openapi/things.yaml": {
			"updateThing": {
				"thingId":       {Original: "thingId", Section: "path", Name: "thingId"},
				"path.thingId":  {Original: "path.thingId", Section: "path", Name: "thingId"},
				"verbose":       {Original: "verbose", Section: "query", Name: "verbose"},
				"query.verbose": {Original: "query.verbose", Section: "query", Name: "verbose"},
			},
		},
	}}

	_, err := mapper.lookup("openapi/things.yaml", "updateThing", "undeclared")
	if err == nil {
		t.Fatal("expected missing field error")
	}
	got := err.Error()
	for _, want := range []string{
		`source path "openapi/things.yaml"`,
		`operationId "updateThing"`,
		`request field "undeclared"`,
		"known request fields: path.thingId",
		"query.verbose",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in error:\n%s", want, got)
		}
	}
}

func TestRequestBindingLookupMissingOperationReportsAvailableOperationIDs(t *testing.T) {
	mapper := &requestBindingMapper{cache: map[string]map[string]map[string]requestFieldPlacement{
		"openapi/things.yaml": {
			"createThing": {},
			"updateThing": {},
		},
	}}

	_, err := mapper.lookup("openapi/things.yaml", "missingThing", "thingId")
	if err == nil {
		t.Fatal("expected missing operation error")
	}
	got := err.Error()
	for _, want := range []string{
		`source path "openapi/things.yaml"`,
		`operationId "missingThing"`,
		"available operationIds: createThing, updateThing",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in error:\n%s", want, got)
		}
	}
}

func TestRequestBindingLookupAmbiguousAliasSuggestsQualifiedSections(t *testing.T) {
	mapper := &requestBindingMapper{
		cache: map[string]map[string]map[string]requestFieldPlacement{
			"openapi/things.yaml": {
				"updateThing": {
					"path.id":  {Original: "path.id", Section: "path", Name: "id"},
					"query.id": {Original: "query.id", Section: "query", Name: "id"},
					"body.id":  {Original: "body.id", Section: "body", Name: "id"},
				},
			},
		},
		ambiguous: map[string]map[string]map[string]bool{
			"openapi/things.yaml": {
				"updateThing": {"id": true},
			},
		},
	}

	_, err := mapper.lookup("openapi/things.yaml", "updateThing", "id")
	if err == nil {
		t.Fatal("expected ambiguous alias error")
	}
	got := err.Error()
	for _, want := range []string{
		`ambiguous request field "id"`,
		`source path "openapi/things.yaml"`,
		`operationId "updateThing"`,
		"qualify fields as path.<name>, query.<name>, header.<name>, or body.<name>",
		"known request fields: body.id",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in error:\n%s", want, got)
		}
	}
}
