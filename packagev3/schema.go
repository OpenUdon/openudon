package packagev3

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"io"

	"github.com/OpenUdon/openudon/wire"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Exact core schema bytes from accepted UWS M08 c0b19385a3b0; no browser
// supplement or ambient schema path is admitted. Tests compare every entry.
//
//go:embed uws-schemas.zip
var schemaArchive []byte

type closedSchemaLoader struct{}

func (closedSchemaLoader) Load(string) (any, error) {
	return nil, errors.New("external schema resources unavailable")
}
func validateWorkflowSchema(root map[string]any) error {
	version, ok := root["uws"].(string)
	if !ok {
		return ErrRecord
	}
	archive, err := zip.NewReader(bytes.NewReader(schemaArchive), int64(len(schemaArchive)))
	if err != nil {
		return ErrRecord
	}
	for _, file := range archive.File {
		if file.Name != version+".json" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return ErrRecord
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, MaxFileBytes+1))
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil || len(data) > MaxFileBytes {
			return ErrRecord
		}
		var schema any
		if wire.DecodeStrictNumbers(data, &schema) != nil {
			return ErrRecord
		}
		compiler := jsonschema.NewCompiler()
		compiler.UseLoader(closedSchemaLoader{})
		compiler.AssertFormat()
		const location = "https://openudon.invalid/schema/uws.json"
		if compiler.AddResource(location, schema) != nil {
			return ErrRecord
		}
		compiled, err := compiler.Compile(location)
		if err != nil || compiled.Validate(root) != nil {
			return ErrRecord
		}
		return nil
	}
	return ErrRecord
}
