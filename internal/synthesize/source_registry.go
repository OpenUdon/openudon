package synthesize

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/googlediscovery"
	"github.com/OpenUdon/asyncapi"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/uws/uws1"
)

type localAPISourceRegistry struct {
	entries map[string]localAPISource
}

type localAPISource struct {
	RelativePath string
	Type         uws1.SourceDescriptionType
	Operations   map[string]bool
	Err          error
}

func newLocalAPISourceRegistry(exampleDir string, candidates []openapidisco.Candidate) (*localAPISourceRegistry, error) {
	registry := &localAPISourceRegistry{entries: map[string]localAPISource{}}
	for _, candidate := range candidates {
		rel := normalizeAPISourceRef(candidate.RelativePath)
		if rel == "" {
			continue
		}
		registry.add(exampleDir, localAPISource{RelativePath: rel, Type: uws1.SourceDescriptionTypeOpenAPI})
	}
	paths, err := packageartifacts.CollectAPISourcePaths(exampleDir)
	if err != nil {
		return registry, err
	}
	for _, path := range paths {
		rel := normalizeAPISourceRef(path)
		if rel == "" {
			continue
		}
		sourceType := sourceDescriptionTypeForPath(rel)
		entry := localAPISource{RelativePath: rel, Type: sourceType}
		abs := filepath.Join(exampleDir, filepath.FromSlash(rel))
		if sniffed, ok, sniffErr := sniffAPISourceType(abs); sniffErr != nil {
			entry.Err = sniffErr
		} else if ok && sniffed != sourceType {
			entry.Err = fmt.Errorf("source path implies %s but document looks like %s", sourceType, sniffed)
		}
		if entry.Err == nil && sourceType != uws1.SourceDescriptionTypeOpenAPI {
			entry.Operations, entry.Err = nativeAPISourceOperations(abs, sourceType)
		}
		registry.add(exampleDir, entry)
	}
	return registry, nil
}

func (registry *localAPISourceRegistry) add(exampleDir string, entry localAPISource) {
	if registry == nil || entry.RelativePath == "" {
		return
	}
	registry.entries[entry.RelativePath] = entry
	if exampleDir != "" {
		abs := normalizeAPISourceRef(filepath.Join(exampleDir, filepath.FromSlash(entry.RelativePath)))
		if abs != "" {
			registry.entries[abs] = entry
		}
	}
}

func (registry *localAPISourceRegistry) get(ref string) (localAPISource, bool) {
	if registry == nil {
		return localAPISource{}, false
	}
	ref = normalizeAPISourceRef(ref)
	if ref == "" {
		return localAPISource{}, false
	}
	entry, ok := registry.entries[ref]
	return entry, ok
}

func (registry *localAPISourceRegistry) nativePaths() []string {
	if registry == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, entry := range registry.entries {
		if entry.Type == uws1.SourceDescriptionTypeOpenAPI {
			continue
		}
		if seen[entry.RelativePath] {
			continue
		}
		seen[entry.RelativePath] = true
		if entry.Err != nil {
			out = append(out, fmt.Sprintf("%s (%s invalid: %v)", entry.RelativePath, entry.Type, entry.Err))
			continue
		}
		out = append(out, fmt.Sprintf("%s (%s)", entry.RelativePath, entry.Type))
	}
	sort.Strings(out)
	return out
}

func nativeAPISourceOperations(path string, sourceType uws1.SourceDescriptionType) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	operations := map[string]bool{}
	switch sourceType {
	case uws1.SourceDescriptionTypeGoogleDiscovery:
		model, parseErr := googlediscovery.Parse(data)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, op := range model.Operations {
			if op == nil {
				continue
			}
			for _, id := range operationIDAliases(op.OperationID, op.ID, op.Name) {
				if id = strings.TrimSpace(id); id != "" {
					operations[id] = true
				}
			}
		}
	case uws1.SourceDescriptionTypeAWSSmithy:
		model, parseErr := awssmithy.Parse(data)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, op := range model.Operations {
			if op == nil {
				continue
			}
			for _, id := range operationIDAliases(op.Name, op.ID) {
				if id = strings.TrimSpace(id); id != "" {
					operations[id] = true
				}
			}
		}
	case uws1.SourceDescriptionTypeAsyncAPI:
		doc, parseErr := asyncapi.Parse(data)
		if parseErr != nil {
			return nil, parseErr
		}
		for selector := range doc.SelectorAliases() {
			operations[selector] = true
		}
	default:
		return nil, nil
	}
	return operations, nil
}

func normalizeAPISourceRef(ref string) string {
	ref = filepath.ToSlash(strings.TrimSpace(ref))
	if ref == "" {
		return ""
	}
	for strings.HasPrefix(ref, "./") {
		ref = strings.TrimPrefix(ref, "./")
	}
	return filepath.ToSlash(filepath.Clean(ref))
}

func sourceDescriptionTypeForPath(path string) uws1.SourceDescriptionType {
	path = normalizeAPISourceRef(path)
	parts := strings.Split(path, "/")
	for _, part := range parts {
		switch part {
		case "google-discovery", "discovery":
			return uws1.SourceDescriptionTypeGoogleDiscovery
		case "aws-smithy":
			return uws1.SourceDescriptionTypeAWSSmithy
		case "asyncapi":
			return uws1.SourceDescriptionTypeAsyncAPI
		case "openapi":
			return uws1.SourceDescriptionTypeOpenAPI
		}
	}
	return uws1.SourceDescriptionTypeOpenAPI
}

func sniffAPISourceType(path string) (uws1.SourceDescriptionType, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 64*1024))
	if err != nil {
		return "", false, err
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", false, nil
	}
	if strings.HasPrefix(text, "{") {
		var root map[string]any
		if err := json.Unmarshal(data, &root); err == nil {
			if _, ok := root["smithy"]; ok {
				return uws1.SourceDescriptionTypeAWSSmithy, true, nil
			}
			if _, ok := root["discoveryVersion"]; ok {
				return uws1.SourceDescriptionTypeGoogleDiscovery, true, nil
			}
			if strings.EqualFold(strings.TrimSpace(stringValue(root["kind"])), "discovery#restDescription") {
				return uws1.SourceDescriptionTypeGoogleDiscovery, true, nil
			}
			if _, ok := root["openapi"]; ok {
				return uws1.SourceDescriptionTypeOpenAPI, true, nil
			}
			if _, ok := root["swagger"]; ok {
				return uws1.SourceDescriptionTypeOpenAPI, true, nil
			}
			if _, ok := root["asyncapi"]; ok {
				return uws1.SourceDescriptionTypeAsyncAPI, true, nil
			}
		}
	}
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, `"smithy"`):
		return uws1.SourceDescriptionTypeAWSSmithy, true, nil
	case strings.Contains(lower, "discovery#restdescription") || strings.Contains(lower, `"discoveryversion"`):
		return uws1.SourceDescriptionTypeGoogleDiscovery, true, nil
	case strings.HasPrefix(lower, "openapi:") || strings.Contains(lower, "\nopenapi:") ||
		strings.HasPrefix(lower, "swagger:") || strings.Contains(lower, "\nswagger:"):
		return uws1.SourceDescriptionTypeOpenAPI, true, nil
	case strings.HasPrefix(lower, "asyncapi:") || strings.Contains(lower, "\nasyncapi:"):
		return uws1.SourceDescriptionTypeAsyncAPI, true, nil
	default:
		return "", false, nil
	}
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
