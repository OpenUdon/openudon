package simulation

import (
	"bytes"
	"errors"
	"sync"

	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const MaxInputBytes = 256 << 10

var inputSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	// Only the embedded resources may satisfy references. There is no file,
	// HTTP or credential-dependent schema loader.
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	for _, name := range []string{"openudon.step-authoring.v1", InputVersion} {
		data, err := schemas.SimulationInputResources.ReadFile(name + ".schema.json")
		if err != nil {
			return nil, err
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if err := compiler.AddResource("https://openudon.dev/schemas/"+name+".schema.json", document); err != nil {
			return nil, err
		}
	}
	return compiler.Compile("https://openudon.dev/schemas/" + InputVersion + ".schema.json")
})

// DecodeInput enforces the published envelope before package capture while
// retaining exact JSON numbers for public fixture matching.
func DecodeInput(data []byte) (Input, error) {
	var input Input
	if len(data) > MaxInputBytes || evidencefile.DecodeStrictNumbers(data, &input) != nil {
		return input, errors.New("invalid simulation input")
	}
	schema, err := inputSchema()
	if err != nil {
		return input, err
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return input, err
	}
	if err := schema.Validate(instance); err != nil {
		return input, err
	}
	return input, nil
}
