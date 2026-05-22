package uwsexec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/runtimes"
	"github.com/OpenUdon/uws/uws1"
)

const (
	DocumentFormatAuto = "auto"
	DocumentFormatYAML = "yaml"
	DocumentFormatJSON = "json"
	DocumentFormatHCL  = "hcl"
	ProfileName        = runtimes.ProfileName
)

type OperationRuntime = runtimes.OperationRuntime

func SetOperationExtension(dst *map[string]any, value *OperationRuntime) error {
	return runtimes.SetOperationExtension(dst, value)
}

func ReadOperationRuntime(extensions map[string]any) (*OperationRuntime, bool, error) {
	return runtimes.ReadOperationExtension(extensions)
}

func LoadDocumentFile(path, format string) (*uws1.Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc uws1.Document
	switch resolveFormat(path, format) {
	case DocumentFormatYAML:
		err = convert.UnmarshalYAML(data, &doc)
	case DocumentFormatJSON:
		err = convert.UnmarshalJSON(data, &doc)
	case DocumentFormatHCL:
		err = convert.UnmarshalHCL(data, &doc)
	default:
		err = fmt.Errorf("unsupported UWS document format %q", format)
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func MarshalDocument(doc *uws1.Document, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", DocumentFormatYAML:
		return convert.MarshalYAML(doc)
	case DocumentFormatJSON:
		return json.MarshalIndent(doc, "", "  ")
	case DocumentFormatHCL:
		return convert.MarshalHCL(doc)
	default:
		return nil, fmt.Errorf("unsupported UWS document format %q", format)
	}
}

func ValidateForExecution(doc *uws1.Document) error {
	if doc == nil {
		return fmt.Errorf("UWS document is required")
	}
	if err := doc.Validate(); err != nil {
		return err
	}
	sourceDescriptions := map[string]bool{}
	for _, source := range doc.SourceDescriptions {
		if source != nil && strings.TrimSpace(source.Name) != "" {
			sourceDescriptions[strings.TrimSpace(source.Name)] = true
		}
	}
	for _, op := range doc.Operations {
		if op == nil {
			continue
		}
		for key := range op.Extensions {
			switch key {
			case "x-udon-runtime", "x-udon-runtime-config":
				return fmt.Errorf("operation %s uses legacy %s extension", op.OperationID, key)
			}
		}
		runtime, hasRuntime, err := runtimes.ReadOperationExtension(op.Extensions)
		if err != nil {
			return fmt.Errorf("operation %s has invalid %s: %w", op.OperationID, runtimes.ExtensionRuntime, err)
		}
		if hasRuntime {
			if strings.EqualFold(strings.TrimSpace(runtime.Type), "http") {
				return fmt.Errorf("operation %s uses invalid x-uws-runtime type http; use OpenAPI binding fields", op.OperationID)
			}
			if !runtimes.IsRuntimeType(runtime.Type) {
				return fmt.Errorf("operation %s uses unsupported runtime type %q", op.OperationID, runtime.Type)
			}
			if strings.TrimSpace(op.ExtensionProfile()) != ProfileName {
				return fmt.Errorf("operation %s uses x-uws-runtime without %s profile", op.OperationID, ProfileName)
			}
		}
		if op.HasSourceBinding() {
			if strings.TrimSpace(op.SourceDescription) == "" {
				return fmt.Errorf("operation %s has API source binding without sourceDescription", op.OperationID)
			}
			if len(sourceDescriptions) > 0 && !sourceDescriptions[strings.TrimSpace(op.SourceDescription)] {
				return fmt.Errorf("operation %s references undeclared sourceDescription %q", op.OperationID, op.SourceDescription)
			}
		}
	}
	return nil
}

func resolveFormat(path, format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "" && format != DocumentFormatAuto {
		return format
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return DocumentFormatYAML
	case ".json":
		return DocumentFormatJSON
	case ".hcl":
		return DocumentFormatHCL
	default:
		return DocumentFormatYAML
	}
}
