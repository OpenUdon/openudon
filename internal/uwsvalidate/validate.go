package uwsvalidate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// ValidateFile validates one UWS JSON or YAML document against the UWS schema.
func ValidateFile(schemaPath, documentPath string) error {
	schema, err := compileSchema(schemaPath)
	if err != nil {
		return err
	}

	value, err := loadDocument(documentPath)
	if err != nil {
		return err
	}

	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("%s: %w", documentPath, err)
	}
	return nil
}

func compileSchema(path string) (*jsonschema.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", path, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse schema %s: %w", path, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(path, doc); err != nil {
		return nil, fmt.Errorf("add schema resource %s: %w", path, err)
	}
	schema, err := compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("compile schema %s: %w", path, err)
	}
	return schema, nil
}

func loadDocument(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read document %s: %w", path, err)
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		value, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("parse JSON document %s: %w", path, err)
		}
		return value, nil
	case ".yaml", ".yml":
		var value any
		if err := yaml.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("parse YAML document %s: %w", path, err)
		}
		return normalizeYAML(value), nil
	default:
		return nil, fmt.Errorf("unsupported document extension %q", filepath.Ext(path))
	}
}

func normalizeYAML(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, val := range typed {
			out[key] = normalizeYAML(val)
		}
		return out
	case []any:
		for i, val := range typed {
			typed[i] = normalizeYAML(val)
		}
		return typed
	case json.Number:
		return typed
	default:
		return typed
	}
}
