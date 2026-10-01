package browsercapture

import (
	"bytes"
	"errors"
	"sync"

	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var wireSchemas = sync.OnceValues(func() (map[string]*jsonschema.Schema, error) {
	data, err := schemas.BrowserCaptureResources.ReadFile(Version + ".schema.json")
	if err != nil {
		return nil, err
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	url := "https://openudon.dev/schemas/" + Version + ".schema.json"
	if err := compiler.AddResource(url, document); err != nil {
		return nil, err
	}
	result := make(map[string]*jsonschema.Schema)
	for _, name := range []string{"Proposal", "Decision", "Cancellation", "Event", "Command", "View"} {
		compiled, err := compiler.Compile(url + "#/$defs/browsercapture" + name)
		if err != nil {
			return nil, err
		}
		result[name] = compiled
	}
	return result, nil
})

func validateSchema(data []byte, target any) error {
	name := ""
	switch target.(type) {
	case *Proposal:
		name = "Proposal"
	case *Decision:
		name = "Decision"
	case *Cancellation:
		name = "Cancellation"
	case *Event:
		name = "Event"
	case *Command:
		name = "Command"
	case *View:
		name = "View"
	default:
		return errors.New("unsupported capture wire record")
	}
	compiled, err := wireSchemas()
	if err != nil {
		return errors.New("capture schema unavailable")
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil || compiled[name].Validate(instance) != nil {
		return errors.New("capture record does not match its published schema")
	}
	return nil
}
