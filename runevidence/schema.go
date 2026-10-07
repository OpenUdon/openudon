package runevidence

import (
	"bytes"
	"sync"

	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var brokerSchemaOnce sync.Once
var brokerSchema *jsonschema.Schema
var brokerSchemaError error

type closedSchemaLoader struct{}

func (closedSchemaLoader) Load(string) (any, error) { return nil, ErrInvalidEvidence }

// The existing published v4 resources are compiled from immutable embedded
// bytes. An unresolved reference fails closed without filesystem/network I/O.
func validateBrokerWire(data []byte) error {
	brokerSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.UseLoader(closedSchemaLoader{})
		compiler.AssertFormat()
		for _, name := range []string{"openudon.broker-authority.v1.schema.json", "openudon.run-evidence.v4.schema.json"} {
			content, err := schemas.BrokerHandoffResources.ReadFile(name)
			if err != nil {
				brokerSchemaError = err
				return
			}
			resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(content))
			if err != nil {
				brokerSchemaError = err
				return
			}
			if err := compiler.AddResource("https://openudon.org/schemas/"+name, resource); err != nil {
				brokerSchemaError = err
				return
			}
		}
		brokerSchema, brokerSchemaError = compiler.Compile("https://openudon.org/schemas/openudon.run-evidence.v4.schema.json")
	})
	if brokerSchemaError != nil {
		return ErrInvalidEvidence
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil || brokerSchema.Validate(value) != nil {
		return ErrInvalidEvidence
	}
	return nil
}
