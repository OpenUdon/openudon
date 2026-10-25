package authoringcard_test

import (
	"encoding/json"
	"testing"

	"github.com/OpenUdon/openudon/authoringcard"
	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/OpenUdon/openudon/wire"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestAdditiveSchemaAndValueStates(t *testing.T) {
	data, err := schemas.AuthoringCardResources.ReadFile("openudon.authoring-card.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema any
	if err := wire.DecodeStrictNumbers(data, &schema); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	const location = "https://openudon.dev/schemas/openudon.authoring-card.v1.schema.json"
	if err := compiler.AddResource(location, schema); err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.Compile(location)
	if err != nil {
		t.Fatal(err)
	}
	card := authoringcard.Card{
		Version: authoringcard.Version,
		Intent:  authoringcard.TextValue{State: authoringcard.Known, Text: "Read the selected item"},
		Outcome: authoringcard.TextValue{State: authoringcard.Missing, Detail: "Owner must name the outcome"},
		Outline: []authoringcard.Step{{ID: "read", Title: "Read item", Operation: authoringcard.Selection{State: authoringcard.Unresolved, Detail: "Two admitted source candidates"}, Mappings: []authoringcard.Mapping{
			{Location: "query", Name: "nullable", Value: authoringcard.Value{State: authoringcard.Known, Literal: json.RawMessage(`null`)}},
			{Location: "query", Name: "count", Value: authoringcard.Value{State: authoringcard.Invalid, Literal: json.RawMessage(`-1`), Detail: "Must be positive"}},
			{Location: "path", Name: "id", Value: authoringcard.Value{State: authoringcard.Unresolved, Binding: "$inputs.id", Detail: "Input not supplied"}},
		}}},
		Validation: authoringcard.Validation{Outcome: "unvalidated", Findings: []authoringcard.Finding{}},
	}
	encoded, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := wire.DecodeStrictNumbers(encoded, &value); err != nil {
		t.Fatal(err)
	}
	if err := compiled.Validate(value); err != nil {
		t.Fatal(err)
	}
	var decoded authoringcard.Card
	if err := wire.DecodeStrictNumbers(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if string(decoded.Outline[0].Mappings[0].Value.Literal) != "null" || decoded.Outcome.State != authoringcard.Missing {
		t.Fatal("literal null or explicit missing state lost")
	}
	object := value.(map[string]any)
	object["approved"] = true
	if compiled.Validate(value) == nil {
		t.Fatal("authoring schema accepted an approval field")
	}
	delete(object, "approved")
	object["version"] = "openudon.step-authoring.v1"
	if compiled.Validate(value) == nil {
		t.Fatal("historical wire version accepted as new card")
	}
}
