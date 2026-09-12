package browsertransaction

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRegistrationV3RequiresExactProfileAndProducerLineage(t *testing.T) {
	data, err := os.ReadFile("../../docs/schemas/openudon.browser-profile-transaction.v3.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("v3.json", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("v3.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"valid", "profile", "producer", "transaction", "session", "kind"} {
		t.Run(change, func(t *testing.T) {
			value := validRegistrationV2()
			value.Version = VersionV3
			value.Provenance.ResultVersion = ResultRegistrationAuthoringV3
			value.Candidates[0].Schema = "uws.browser-registration.1.1"
			switch change {
			case "profile":
				value.Candidates[0].Schema = "uws.browser-registration.1.0"
			case "producer":
				value.Provenance.ResultVersion = ResultRegistrationAuthoringV2
			case "transaction":
				value.Version = VersionV2
			case "session":
				value.Session = "private_session"
			case "kind":
				value.Kind = KindAuthenticationCapability
			}
			data, _ := json.Marshal(value)
			instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if (value.Validate() == nil) != (change == "valid") || (schema.Validate(instance) == nil) != (change == "valid") {
				t.Fatal("Go and published schema disagree with version boundary")
			}
		})
	}
}
