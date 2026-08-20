package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/apitools/googlediscovery"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"gopkg.in/yaml.v3"
)

type openAPISecurityRequirement struct {
	Scheme string
	Name   string
	In     string
	Type   string
}

func (r openAPISecurityRequirement) label() string {
	if strings.TrimSpace(r.Scheme) != "" {
		return strings.TrimSpace(r.Scheme)
	}
	if strings.TrimSpace(r.Name) != "" {
		return strings.TrimSpace(r.Name)
	}
	return "security"
}

func (r openAPISecurityRequirement) fieldNames() []string {
	var out []string
	for _, name := range []string{r.Name, r.Scheme} {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
			if alias := camelToSnake(name); alias != name {
				out = append(out, alias)
			}
		}
	}
	if strings.EqualFold(r.Type, "http") || strings.EqualFold(r.Scheme, "bearer") || strings.Contains(strings.ToLower(r.Scheme), "bearer") {
		out = append(out, "Authorization", "authorization", "header.Authorization", "header.authorization", "header_pars.Authorization", "header_pars.authorization")
	}
	switch strings.ToLower(strings.TrimSpace(r.In)) {
	case "query":
		for _, name := range []string{r.Name, r.Scheme} {
			if strings.TrimSpace(name) != "" {
				out = append(out, "query."+name, "query_pars."+name)
			}
		}
	case "header":
		for _, name := range []string{r.Name, r.Scheme} {
			if strings.TrimSpace(name) != "" {
				out = append(out, "header."+name, "header_pars."+name)
			}
		}
	}
	return sortedUnique(out)
}

func (r openAPISecurityRequirement) bindingCandidates() []string {
	return sortedUnique([]string{r.Scheme, r.Name, strings.ReplaceAll(r.Name, "-", "_"), strings.ReplaceAll(r.Scheme, "-", "_")})
}

func openAPISecurityIndex(candidates []openapidisco.Candidate) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	for _, candidate := range candidates {
		doc, err := readOpenAPISecurityDocument(candidate.Path)
		if err != nil {
			continue
		}
		schemes := openAPISecuritySchemes(doc)
		global := openAPISecurityRequirements(asMap(doc["security"]), schemes)
		paths := asMap(doc["paths"])
		for path, rawPathItem := range paths {
			pathItem := asMap(rawPathItem)
			for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
				rawOp, ok := pathItem[method]
				if !ok {
					continue
				}
				op := asMap(rawOp)
				operationID := strings.TrimSpace(asString(op["operationId"]))
				if operationID == "" {
					continue
				}
				requirements := global
				if rawSecurity, ok := op["security"]; ok {
					requirements = openAPISecurityRequirements(asMap(rawSecurity), schemes)
				}
				if len(requirements) == 0 {
					continue
				}
				_ = path
				out[operationKey(candidate.RelativePath, operationID)] = requirements
			}
		}
	}
	return out
}

func nativeSecurityIndex(candidates []openapidisco.Candidate) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	for _, candidate := range candidates {
		sourceType := sourceDescriptionTypeForPath(candidate.RelativePath)
		if sourceType == "openapi" {
			continue
		}
		for key, reqs := range nativeSecurityForSource(candidate.RelativePath, candidate.Path, string(sourceType)) {
			out[key] = append(out[key], reqs...)
		}
	}
	for key, reqs := range out {
		out[key] = sortedSecurityRequirements(reqs)
	}
	return out
}

func localNativeSecurityIndex(exampleDir string) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	if strings.TrimSpace(exampleDir) == "" {
		return out
	}
	paths, err := packageartifacts.CollectAPISourcePaths(exampleDir)
	if err != nil {
		return out
	}
	for _, rel := range paths {
		sourceType := sourceDescriptionTypeForPath(rel)
		if sourceType == "openapi" {
			continue
		}
		path := filepath.Join(exampleDir, filepath.FromSlash(rel))
		for key, reqs := range nativeSecurityForSource(rel, path, string(sourceType)) {
			out[key] = append(out[key], reqs...)
		}
	}
	for key, reqs := range out {
		out[key] = sortedSecurityRequirements(reqs)
	}
	return out
}

func nativeSecurityForSource(rel, path string, sourceType string) map[string][]openAPISecurityRequirement {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	switch sourceType {
	case "google-discovery":
		model, err := googlediscovery.Parse(data)
		if err != nil {
			return nil
		}
		return googleDiscoverySecurityForSource(rel, model)
	case "aws-smithy":
		model, err := awssmithy.Parse(data)
		if err != nil {
			return nil
		}
		return smithySecurityForSource(rel, model)
	default:
		return nil
	}
}

func googleDiscoverySecurityForSource(rel string, model *googlediscovery.Model) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	if model == nil {
		return out
	}
	scheme := nativeCredentialScheme(model.Name, "oauth_token")
	if scheme == "" {
		return out
	}
	req := openAPISecurityRequirement{Scheme: scheme, Type: "oauth2"}
	for _, op := range model.Operations {
		if op == nil || len(op.Scopes) == 0 {
			continue
		}
		for _, alias := range operationIDAliases(op.OperationID, op.ID, op.Name) {
			out[operationKey(rel, alias)] = append(out[operationKey(rel, alias)], req)
		}
	}
	return out
}

func smithySecurityForSource(rel string, model *awssmithy.Model) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	if model == nil {
		return out
	}
	scheme := nativeCredentialScheme(model.SigningName, "sigv4")
	if scheme == "" {
		return out
	}
	req := openAPISecurityRequirement{
		Scheme: scheme,
		Name:   "Authorization",
		In:     "header",
		Type:   "http",
	}
	for _, op := range model.Operations {
		if op == nil || strings.TrimSpace(op.Name) == "" {
			continue
		}
		for _, alias := range operationIDAliases(op.Name, op.ID) {
			out[operationKey(rel, alias)] = append(out[operationKey(rel, alias)], req)
		}
	}
	return out
}

func nativeCredentialScheme(name, suffix string) string {
	name = operationIDAlias(strings.TrimSpace(name))
	suffix = strings.TrimSpace(suffix)
	if name == "" || suffix == "" {
		return ""
	}
	return name + "_" + suffix
}

func localAdvisorySecurityIndex(exampleDir string) (map[string][]openAPISecurityRequirement, []error) {
	out := map[string][]openAPISecurityRequirement{}
	if strings.TrimSpace(exampleDir) == "" {
		return out, nil
	}
	paths, err := packageartifacts.CollectAPISourcePaths(exampleDir)
	if err != nil {
		return out, []error{err}
	}
	var errs []error
	for _, rel := range paths {
		path := filepath.Join(exampleDir, filepath.FromSlash(rel))
		overlay, ok, err := readSecuritySidecar(path)
		if err != nil {
			errs = append(errs, err)
		}
		if !ok {
			continue
		}
		for key, reqs := range advisorySecurityForSource(rel, path, overlay) {
			out[key] = append(out[key], reqs...)
		}
	}
	for key, reqs := range out {
		out[key] = sortedSecurityRequirements(reqs)
	}
	return out, errs
}

func mergeAdvisorySecurityRequirements(base, advisory map[string][]openAPISecurityRequirement) {
	for key, reqs := range advisory {
		if advisoryOverridesNativeSecurity(key) {
			base[key] = append([]openAPISecurityRequirement(nil), reqs...)
			continue
		}
		base[key] = append(base[key], reqs...)
	}
}

func advisoryOverridesNativeSecurity(key string) bool {
	source, _, _ := strings.Cut(key, "\x00")
	return sourceDescriptionTypeForPath(source) != "openapi"
}

func readSecuritySidecar(sourcePath string) (catalog.SecurityMetadata, bool, error) {
	for _, path := range packageartifacts.AdvisorySecuritySidecarPathCandidates(sourcePath) {
		data, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return catalog.SecurityMetadata{}, true, fmt.Errorf("%s: %w", path, err)
			}
			continue
		}
		var metadata catalog.SecurityMetadata
		if err := json.Unmarshal(data, &metadata); err == nil && securityMetadataPresent(metadata) {
			return metadata, true, nil
		}
		if err := unmarshalYAMLThroughJSON(data, &metadata); err == nil && securityMetadataPresent(metadata) {
			return metadata, true, nil
		}
		var overlay catalog.SecurityOverlay
		if err := json.Unmarshal(data, &overlay); err == nil {
			metadata = catalog.SecurityMetadata{
				SecuritySchemes:   overlay.SecuritySchemes,
				RootSecurity:      overlay.RootSecurity,
				RootSecuritySets:  overlay.RootSecuritySets,
				OperationSecurity: overlay.OperationSecurity,
			}
			if securityMetadataPresent(metadata) {
				return metadata, true, nil
			}
		}
		if err := unmarshalYAMLThroughJSON(data, &overlay); err == nil {
			metadata = catalog.SecurityMetadata{
				SecuritySchemes:   overlay.SecuritySchemes,
				RootSecurity:      overlay.RootSecurity,
				RootSecuritySets:  overlay.RootSecuritySets,
				OperationSecurity: overlay.OperationSecurity,
			}
			if securityMetadataPresent(metadata) {
				return metadata, true, nil
			}
		}
		return catalog.SecurityMetadata{}, true, fmt.Errorf("%s: invalid or empty security metadata; expected catalog.SecurityMetadata or catalog.SecurityOverlay with security schemes or requirements", path)
	}
	return catalog.SecurityMetadata{}, false, nil
}

func unmarshalYAMLThroughJSON(data []byte, dst any) error {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	compatible := yamlToJSONCompatible(raw)
	jsonData, err := json.Marshal(compatible)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, dst)
}

func yamlToJSONCompatible(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = yamlToJSONCompatible(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[strings.TrimSpace(fmt.Sprint(key))] = yamlToJSONCompatible(child)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = yamlToJSONCompatible(child)
		}
		return out
	default:
		return value
	}
}

func securityMetadataPresent(metadata catalog.SecurityMetadata) bool {
	return len(metadata.SecuritySchemes) > 0 || len(metadata.RootSecurity) > 0 || len(metadata.RootSecuritySets) > 0 || len(metadata.OperationSecurity) > 0
}

func joinErrorMessages(errs []error) string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil && strings.TrimSpace(err.Error()) != "" {
			messages = append(messages, err.Error())
		}
	}
	sort.Strings(messages)
	return strings.Join(messages, "; ")
}

func advisorySecurityForSource(rel, path string, metadata catalog.SecurityMetadata) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	schemes := advisorySecuritySchemes(metadata.SecuritySchemes)
	ops := operationRefsForSecuritySource(rel, path)
	root := advisorySecurityRequirements(metadata.RootSecurity, metadata.RootSecuritySets, schemes)
	if len(root) > 0 {
		for _, op := range ops {
			for _, alias := range op.Aliases {
				out[operationKey(rel, alias)] = append(out[operationKey(rel, alias)], root...)
			}
		}
	}
	for _, opSecurity := range metadata.OperationSecurity {
		reqs := advisorySecurityRequirements(opSecurity.Security, opSecurity.SecuritySets, schemes)
		if len(reqs) == 0 {
			continue
		}
		for _, aliases := range matchingOperationAliases(opSecurity.Match, ops) {
			for _, alias := range aliases {
				out[operationKey(rel, alias)] = append(out[operationKey(rel, alias)], reqs...)
			}
		}
	}
	return out
}
