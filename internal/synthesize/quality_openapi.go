package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/awssmithy"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/apitools/googlediscovery"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"gopkg.in/yaml.v3"
)

func validateIntentOpenAPIOperations(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	sourceRegistry, registryErr := newLocalAPISourceRegistry(exampleDir, candidates)
	if registryErr != nil && !os.IsNotExist(registryErr) {
		return fmt.Errorf("local API source registry could not be scanned: %w", registryErr)
	}
	var missing []string
	var omitted []string
	var invalid []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		specPath := intentStepOpenAPIPath(intent, step, primary)
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		if entry, ok := sourceRegistry.get(specPath); ok && entry.Err != nil {
			invalid = append(invalid, fmt.Sprintf("%s in %q: %v", name, specPath, entry.Err))
			return
		}
		if sourceDescriptionTypeForPath(specPath) != "openapi" {
			if operation == "" {
				if intentStepRequiresOpenAPIOperation(intent, step, primary) {
					omitted = append(omitted, fmt.Sprintf("%s in %q", name, specPath))
				}
				return
			}
			entry, ok := sourceRegistry.get(specPath)
			if !ok {
				missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
				return
			}
			if len(entry.Operations) == 0 {
				omitted = append(omitted, fmt.Sprintf("%s in %q (no operations discovered)", name, specPath))
				return
			}
			if !entry.Operations[operation] {
				missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
			}
			return
		}
		if operation == "" {
			if intentStepRequiresOpenAPIOperation(intent, step, primary) {
				omitted = append(omitted, fmt.Sprintf("%s in %q", name, specPath))
			}
			return
		}
		if op := ops[operationKey(specPath, operation)]; op == nil {
			missing = append(missing, fmt.Sprintf("%s operation %q in %q", name, operation, specPath))
		}
	})
	if len(invalid) > 0 || len(omitted) > 0 || len(missing) > 0 {
		sort.Strings(invalid)
		sort.Strings(omitted)
		sort.Strings(missing)
		var details []string
		for _, item := range invalid {
			details = append(details, "invalid API source "+item)
		}
		for _, item := range omitted {
			details = append(details, "missing operation for "+item)
		}
		for _, item := range missing {
			details = append(details, "missing API source operation "+item)
		}
		return fmt.Errorf("%s", strings.Join(details, "; "))
	}
	return nil
}

func intentStepRequiresOpenAPIOperation(intent *rollout.Intent, step *rollout.Step, primary string) bool {
	if step == nil {
		return false
	}
	kind := strings.ToLower(strings.TrimSpace(step.Type))
	if kind != "" && kind != "http" && kind != "openapi" {
		return false
	}
	return strings.TrimSpace(intentStepOpenAPIPath(intent, step, primary)) != ""
}

func validateIntentRequiredParameters(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	for key, op := range localNativeOperationIndex(exampleDir) {
		ops[key] = op
	}
	inputs := intentInputNames(intent)
	var missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		if operation == "" {
			return
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, operation)]
		if op == nil {
			return
		}
		for _, param := range op.Parameters {
			if param == nil || !param.Required || credentialLikeParam(param.Name) {
				continue
			}
			if stepSatisfiesParam(step, param, inputs) {
				continue
			}
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			missing = append(missing, fmt.Sprintf("%s.%s requires %s parameter %q", name, operation, param.In, param.Name))
		}
	})
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s. Add literals, inputs, bind/with mappings, or import a complementary OpenAPI document that produces the missing values.", strings.Join(missing, "; "))
	}
	return nil
}

func validateIntentCredentialPolicy(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	ops := openAPIOperationIndex(candidates)
	for key, op := range localNativeOperationIndex(exampleDir) {
		ops[key] = op
	}
	inputs := intentInputNames(intent)
	var required, missingBinding []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		operation := strings.TrimSpace(step.Operation)
		if operation == "" {
			return
		}
		specPath := intentStepOpenAPIPath(intent, step, primary)
		op := ops[operationKey(specPath, operation)]
		if op == nil {
			return
		}
		for _, param := range op.Parameters {
			if param == nil || !param.Required || !credentialLikeParam(param.Name) {
				continue
			}
			name := strings.TrimSpace(step.Name)
			if name == "" {
				name = "<unnamed>"
			}
			required = append(required, fmt.Sprintf("%s.%s requires credential-like parameter %q", name, operation, param.Name))
			if stepSatisfiesParam(step, param, inputs) {
				continue
			}
			if credentialDeclaredForParam(policy, param.Name) {
				continue
			}
			missingBinding = append(missingBinding, fmt.Sprintf("%s.%s has no auditable credential binding for %q", name, operation, param.Name))
		}
	})
	if len(required) == 0 {
		return nil
	}
	if strings.TrimSpace(policy.CredentialSection) == "" {
		sort.Strings(required)
		return fmt.Errorf("%s. Add a Credentials and Secrets section that names runtime credential bindings, never literal secrets.", strings.Join(required, "; "))
	}
	if len(missingBinding) > 0 {
		sort.Strings(missingBinding)
		return fmt.Errorf("%s. Add a with/bind request mapping or name a credential binding that includes the parameter name.", strings.Join(missingBinding, "; "))
	}
	return nil
}

func validateIntentOpenAPISecurity(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	security := openAPISecurityIndex(candidates)
	for key, reqs := range localAdvisorySecurityIndex(exampleDir) {
		security[key] = append(security[key], reqs...)
	}
	var required, missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Operation) == "" {
			return
		}
		reqs := security[operationKey(intentStepOpenAPIPath(intent, step, primary), step.Operation)]
		if len(reqs) == 0 {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for _, req := range reqs {
			label := req.label()
			required = append(required, fmt.Sprintf("%s.%s requires OpenAPI security %q", name, step.Operation, label))
			if intentSecurityCoversRequirement(intent, req) || stepCoversSecurityRequirement(step, req, policy) || credentialDeclaredForSecurity(policy, req) {
				continue
			}
			missing = append(missing, fmt.Sprintf("%s.%s has no auditable credential binding for OpenAPI security %q", name, step.Operation, label))
		}
	})
	if len(required) == 0 {
		return nil
	}
	if strings.TrimSpace(policy.CredentialSection) == "" {
		return fmt.Errorf("%s. Add a Credentials and Secrets section that names security credential bindings, never literal secrets.", strings.Join(sortedCopy(required), "; "))
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s. Bind the security field by credential binding name or add a matching credential binding policy.", strings.Join(sortedCopy(missing), "; "))
	}
	return nil
}

func credentialDeclaredForParam(policy projectPolicy, name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, binding := range credentialBindingNames(policy) {
		if strings.Contains(strings.ToLower(binding), name) {
			return true
		}
	}
	return false
}

func credentialDeclaredForSecurity(policy projectPolicy, req openAPISecurityRequirement) bool {
	for _, binding := range credentialBindingNames(policy) {
		if securityBindingMatches(binding, req) {
			return true
		}
	}
	return false
}

func securityBindingMatches(binding string, req openAPISecurityRequirement) bool {
	binding = strings.ToLower(strings.TrimSpace(binding))
	if binding == "" {
		return false
	}
	for _, candidate := range req.bindingCandidates() {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate != "" && (strings.Contains(binding, candidate) || strings.Contains(candidate, binding)) {
			return true
		}
	}
	return false
}

func intentSecurityCoversRequirement(intent *rollout.Intent, req openAPISecurityRequirement) bool {
	if intent == nil {
		return false
	}
	for _, security := range intent.Security {
		if security == nil {
			continue
		}
		for _, candidate := range []string{security.Name, security.TokenFrom} {
			if securityBindingMatches(candidate, req) {
				return true
			}
		}
	}
	return false
}

func stepCoversSecurityRequirement(step *rollout.Step, req openAPISecurityRequirement, policy projectPolicy) bool {
	if step == nil {
		return false
	}
	names := req.fieldNames()
	for _, name := range names {
		if source := strings.TrimSpace(step.With[name]); source != "" && securityCredentialSourceAllowed(source, req, policy) {
			return true
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			if source := strings.TrimSpace(bind.Fields[name]); source != "" && securityCredentialSourceAllowed(source, req, policy) {
				return true
			}
		}
	}
	return false
}

func securityCredentialSourceAllowed(source string, req openAPISecurityRequirement, policy projectPolicy) bool {
	source = strings.TrimSpace(source)
	if securityBindingMatches(source, req) {
		return true
	}
	credentialSource := strings.TrimPrefix(source, "credentials.")
	for _, binding := range credentialBindingNames(policy) {
		binding = strings.TrimSpace(binding)
		if strings.EqualFold(source, binding) || strings.EqualFold(credentialSource, binding) {
			return true
		}
	}
	return false
}

func openAPIOperationIndex(candidates []openapidisco.Candidate) map[string]*rollout.OperationInfo {
	out := map[string]*rollout.OperationInfo{}
	for _, candidate := range candidates {
		sourceType := sourceDescriptionTypeForPath(candidate.RelativePath)
		if sourceType != "openapi" {
			for alias, op := range nativeOperationInfoIndex(candidate.Path, string(sourceType)) {
				out[operationKey(candidate.RelativePath, alias)] = op
			}
			continue
		}
		spec, err := rollout.LoadOpenAPISpec(candidate.Path)
		if err != nil {
			continue
		}
		for _, op := range spec.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			out[operationKey(candidate.RelativePath, op.OperationID)] = op
		}
	}
	return out
}

func localNativeOperationIndex(exampleDir string) map[string]*rollout.OperationInfo {
	out := map[string]*rollout.OperationInfo{}
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
		for alias, op := range nativeOperationInfoIndex(path, string(sourceType)) {
			out[operationKey(rel, alias)] = op
		}
	}
	return out
}

func nativeOperationInfoIndex(path string, sourceType string) map[string]*rollout.OperationInfo {
	out := map[string]*rollout.OperationInfo{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	switch sourceType {
	case "google-discovery":
		model, err := googlediscovery.Parse(data)
		if err != nil {
			return out
		}
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			info := &rollout.OperationInfo{
				OperationID: op.OperationID,
				Method:      op.HTTPMethod,
				Path:        op.Path,
				Summary:     op.Summary,
				Description: op.Description,
				Tags:        append([]string(nil), op.Tags...),
			}
			for _, param := range op.Parameters {
				if param == nil {
					continue
				}
				info.Parameters = append(info.Parameters, &rollout.ParameterInfo{
					Name:        param.Name,
					In:          param.Location,
					Required:    param.Required,
					Description: param.Description,
				})
			}
			if body := googleDiscoveryRequestBodySummary(model, op); body != nil && body.Required {
				for _, field := range requiredRequestBodyParameterNames(body) {
					info.Parameters = append(info.Parameters, &rollout.ParameterInfo{Name: field, In: "body", Required: true})
				}
			}
			for _, alias := range operationIDAliases(op.OperationID, op.ID, op.Name) {
				out[alias] = info
			}
		}
		return out
	case "aws-smithy":
		model, err := awssmithy.Parse(data)
		if err != nil {
			return out
		}
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.Name) == "" {
				continue
			}
			info := &rollout.OperationInfo{
				OperationID: op.Name,
				Method:      op.Method,
				Path:        op.URI,
			}
			for _, param := range smithyOperationParameters(op) {
				info.Parameters = append(info.Parameters, param)
			}
			for _, alias := range operationIDAliases(op.Name, op.ID) {
				out[alias] = info
			}
		}
		return out
	default:
		return out
	}
}

func requiredRequestBodyParameterNames(body *apitools.RequestBodySummary) []string {
	if body == nil || !body.Required {
		return nil
	}
	var out []string
	for _, field := range body.Fields {
		if strings.TrimSpace(field.Path) != "" && field.Required {
			out = append(out, strings.TrimSpace(field.Path))
		}
	}
	if len(out) == 0 && len(body.Fields) > 0 {
		for _, preferred := range []string{"raw", "message", "text", "body", "content", "html", "payload"} {
			for _, field := range body.Fields {
				if strings.EqualFold(strings.TrimSpace(field.Path), preferred) {
					return []string{strings.TrimSpace(field.Path)}
				}
			}
		}
	}
	if len(out) == 0 {
		out = append(out, "body")
	}
	return sortedUnique(out)
}

func smithyOperationParameters(op *awssmithy.Operation) []*rollout.ParameterInfo {
	if op == nil {
		return nil
	}
	var out []*rollout.ParameterInfo
	for _, binding := range op.InputBindings {
		if binding == nil {
			continue
		}
		name := firstNonEmpty(binding.WireName, binding.MemberName)
		if name == "" {
			continue
		}
		location := strings.TrimSpace(binding.Location)
		switch location {
		case "label":
			location = "path"
		case "":
			location = "body"
		}
		out = append(out, &rollout.ParameterInfo{Name: name, In: location, Required: binding.Required})
	}
	return out
}

func operationIDAliases(ids ...string) []string {
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out = append(out, id)
		alias := operationIDAlias(id)
		if alias != "" && alias != id {
			out = append(out, alias)
		}
	}
	return sortedUnique(out)
}

func operationIDAlias(id string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

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

func localAdvisorySecurityIndex(exampleDir string) map[string][]openAPISecurityRequirement {
	out := map[string][]openAPISecurityRequirement{}
	paths, err := packageartifacts.CollectAPISourcePaths(exampleDir)
	if err != nil {
		return out
	}
	for _, rel := range paths {
		path := filepath.Join(exampleDir, filepath.FromSlash(rel))
		overlay, ok := readSecuritySidecar(path)
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
	return out
}

func readSecuritySidecar(sourcePath string) (catalog.SecurityMetadata, bool) {
	for _, path := range securitySidecarPaths(sourcePath) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var metadata catalog.SecurityMetadata
		if err := json.Unmarshal(data, &metadata); err == nil && securityMetadataPresent(metadata) {
			return metadata, true
		}
		if err := yaml.Unmarshal(data, &metadata); err == nil && securityMetadataPresent(metadata) {
			return metadata, true
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
				return metadata, true
			}
		}
		if err := yaml.Unmarshal(data, &overlay); err == nil {
			metadata = catalog.SecurityMetadata{
				SecuritySchemes:   overlay.SecuritySchemes,
				RootSecurity:      overlay.RootSecurity,
				RootSecuritySets:  overlay.RootSecuritySets,
				OperationSecurity: overlay.OperationSecurity,
			}
			if securityMetadataPresent(metadata) {
				return metadata, true
			}
		}
	}
	return catalog.SecurityMetadata{}, false
}

func securitySidecarPaths(sourcePath string) []string {
	ext := filepath.Ext(sourcePath)
	base := strings.TrimSuffix(sourcePath, ext)
	return []string{
		sourcePath + ".security.json",
		sourcePath + ".security.yaml",
		base + ".security.json",
		base + ".security.yaml",
		base + ".security-overlay.json",
		base + ".security-overlay.yaml",
	}
}

func securityMetadataPresent(metadata catalog.SecurityMetadata) bool {
	return len(metadata.SecuritySchemes) > 0 || len(metadata.RootSecurity) > 0 || len(metadata.RootSecuritySets) > 0 || len(metadata.OperationSecurity) > 0
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

type securityOperationRef struct {
	Aliases []string
	Method  string
	Path    string
	Tags    []string
}

func operationRefsForSecuritySource(rel, path string) []securityOperationRef {
	sourceType := sourceDescriptionTypeForPath(rel)
	if sourceType != "openapi" {
		return nativeOperationRefsForSecurity(path, string(sourceType))
	}
	spec, err := rollout.LoadOpenAPISpec(path)
	if err != nil {
		return nil
	}
	var out []securityOperationRef
	for _, op := range spec.Operations {
		if op == nil || strings.TrimSpace(op.OperationID) == "" {
			continue
		}
		out = append(out, securityOperationRef{
			Aliases: []string{strings.TrimSpace(op.OperationID)},
			Method:  op.Method,
			Path:    op.Path,
			Tags:    append([]string(nil), op.Tags...),
		})
	}
	return out
}

func nativeOperationRefsForSecurity(path string, sourceType string) []securityOperationRef {
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
		var out []securityOperationRef
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.OperationID) == "" {
				continue
			}
			out = append(out, securityOperationRef{
				Aliases: operationIDAliases(op.OperationID, op.ID, op.Name),
				Method:  op.HTTPMethod,
				Path:    op.Path,
				Tags:    append([]string(nil), op.Tags...),
			})
		}
		return out
	case "aws-smithy":
		model, err := awssmithy.Parse(data)
		if err != nil {
			return nil
		}
		var out []securityOperationRef
		for _, op := range model.Operations {
			if op == nil || strings.TrimSpace(op.Name) == "" {
				continue
			}
			out = append(out, securityOperationRef{
				Aliases: operationIDAliases(op.Name, op.ID),
				Method:  op.Method,
				Path:    firstNonEmpty(op.Path, op.URI),
			})
		}
		return out
	default:
		return nil
	}
}

func matchingOperationAliases(match catalog.OperationMatch, ops []securityOperationRef) [][]string {
	var out [][]string
	for _, op := range ops {
		if operationSecurityMatches(match, op) {
			out = append(out, op.Aliases)
		}
	}
	return out
}

func operationSecurityMatches(match catalog.OperationMatch, op securityOperationRef) bool {
	if strings.TrimSpace(match.OperationID) != "" {
		for _, alias := range op.Aliases {
			if strings.EqualFold(strings.TrimSpace(match.OperationID), alias) {
				return true
			}
		}
		return false
	}
	if strings.TrimSpace(match.Method) != "" && !strings.EqualFold(strings.TrimSpace(match.Method), strings.TrimSpace(op.Method)) {
		return false
	}
	if strings.TrimSpace(match.Path) != "" && strings.TrimSpace(match.Path) != strings.TrimSpace(op.Path) {
		return false
	}
	if len(match.Tags) > 0 {
		for _, want := range match.Tags {
			if !containsFold(op.Tags, want) {
				return false
			}
		}
	}
	return strings.TrimSpace(match.Method) != "" || strings.TrimSpace(match.Path) != "" || len(match.Tags) > 0
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func advisorySecuritySchemes(schemes []catalog.SecurityScheme) map[string]openAPISecurityRequirement {
	out := map[string]openAPISecurityRequirement{}
	for _, scheme := range schemes {
		name := strings.TrimSpace(scheme.Name)
		if name == "" {
			continue
		}
		out[name] = openAPISecurityRequirement{
			Scheme: name,
			Name:   strings.TrimSpace(scheme.ParameterName),
			In:     string(scheme.In),
			Type:   string(scheme.Type),
		}
	}
	return out
}

func advisorySecurityRequirements(requirements []catalog.SecurityRequirement, sets []catalog.SecurityRequirementSet, schemes map[string]openAPISecurityRequirement) []openAPISecurityRequirement {
	var out []openAPISecurityRequirement
	for _, req := range requirements {
		out = append(out, advisorySecurityRequirement(req, schemes))
	}
	for _, set := range sets {
		for _, req := range set.Requirements {
			out = append(out, advisorySecurityRequirement(req, schemes))
		}
	}
	return sortedSecurityRequirements(out)
}

func advisorySecurityRequirement(req catalog.SecurityRequirement, schemes map[string]openAPISecurityRequirement) openAPISecurityRequirement {
	name := strings.TrimSpace(req.Scheme)
	if scheme, ok := schemes[name]; ok {
		return scheme
	}
	return openAPISecurityRequirement{Scheme: name}
}

func readOpenAPISecurityDocument(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return asMap(raw), nil
}

func openAPISecuritySchemes(doc map[string]any) map[string]openAPISecurityRequirement {
	out := map[string]openAPISecurityRequirement{}
	components := asMap(doc["components"])
	schemes := asMap(components["securitySchemes"])
	if len(schemes) == 0 {
		schemes = asMap(doc["securityDefinitions"])
	}
	for name, raw := range schemes {
		scheme := asMap(raw)
		out[name] = openAPISecurityRequirement{
			Scheme: name,
			Name:   asString(scheme["name"]),
			In:     asString(scheme["in"]),
			Type:   asString(scheme["type"]),
		}
	}
	return out
}

func openAPISecurityRequirements(raw map[string]any, schemes map[string]openAPISecurityRequirement) []openAPISecurityRequirement {
	var out []openAPISecurityRequirement
	for _, item := range asSlice(raw) {
		req := asMap(item)
		for name := range req {
			if scheme, ok := schemes[name]; ok {
				out = append(out, scheme)
				continue
			}
			out = append(out, openAPISecurityRequirement{Scheme: name})
		}
	}
	return sortedSecurityRequirements(out)
}

func sortedSecurityRequirements(values []openAPISecurityRequirement) []openAPISecurityRequirement {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Scheme != values[j].Scheme {
			return values[i].Scheme < values[j].Scheme
		}
		return values[i].Name < values[j].Name
	})
	return values
}

func asMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		out := map[string]any{}
		for key, val := range typed {
			out[fmt.Sprint(key)] = val
		}
		return out
	case []any:
		out := map[string]any{}
		for i, item := range typed {
			out[fmt.Sprint(i)] = item
		}
		return out
	default:
		return nil
	}
}

func asSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case map[string]any:
		if len(typed) == 0 {
			return nil
		}
		out := make([]any, 0, len(typed))
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			out = append(out, typed[key])
		}
		return out
	default:
		return nil
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func sortedMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func operationKey(specPath, operation string) string {
	return strings.TrimSpace(specPath) + "\x00" + strings.TrimSpace(operation)
}

func intentStepOpenAPIPath(intent *rollout.Intent, step *rollout.Step, primary string) string {
	if step != nil {
		if source := strings.TrimSpace(step.Source); source != "" {
			return source
		}
		if openapi := strings.TrimSpace(step.OpenAPI); openapi != "" {
			return openapi
		}
	}
	if intent != nil {
		if source := strings.TrimSpace(intent.Source); source != "" {
			return source
		}
		if openapi := strings.TrimSpace(intent.OpenAPI); openapi != "" {
			return openapi
		}
	}
	return strings.TrimSpace(primary)
}

func intentInputNames(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	for _, input := range intent.Inputs {
		if input != nil && strings.TrimSpace(input.Name) != "" {
			out[strings.TrimSpace(input.Name)] = true
		}
	}
	return out
}

func credentialLikeParam(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, token := range []string{"key", "token", "secret", "password", "appid", "api_key", "apikey", "authorization"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func stepSatisfiesParam(step *rollout.Step, param *rollout.ParameterInfo, inputs map[string]bool) bool {
	if step == nil || param == nil {
		return false
	}
	names := paramTargetNames(param)
	for _, name := range names {
		if step.With[name] != "" {
			return true
		}
		for _, bind := range step.Binds {
			if bind != nil && bind.Fields[name] != "" {
				return true
			}
		}
	}
	if inputs[param.Name] {
		return true
	}
	for _, value := range step.With {
		if referencesInputName(value, param.Name) {
			return true
		}
	}
	return false
}

func paramTargetNames(param *rollout.ParameterInfo) []string {
	name := strings.TrimSpace(param.Name)
	if name == "" {
		return nil
	}
	var out []string
	out = append(out, name)
	if param.In != "" {
		out = append(out, strings.TrimSpace(param.In)+"."+name)
	}
	if param.In == "query" {
		out = append(out, "query_pars."+name)
	}
	if param.In == "path" {
		out = append(out, "path_pars."+name)
	}
	return out
}

func referencesInputName(value, name string) bool {
	value = strings.TrimSpace(value)
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return value == name || strings.Contains(value, "inputs."+name) || strings.Contains(value, "input."+name)
}

func validateIntentDataFlowSources(intent *rollout.Intent) error {
	if intent == nil {
		return nil
	}
	stepNames := intentStepNameSet(intent)
	inputs := intentInputNames(intent)
	var unresolved []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for _, dep := range step.DependsOn {
			dep = strings.TrimSpace(dep)
			if dep != "" && !stepNames[dep] {
				unresolved = append(unresolved, fmt.Sprintf("%s depends_on %q", name, dep))
			}
		}
		for target, source := range step.With {
			for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
				unresolved = append(unresolved, fmt.Sprintf("%s.%s references %q", name, target, ref))
			}
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			from := strings.TrimSpace(bind.From)
			if from != "" && !stepNames[from] {
				unresolved = append(unresolved, fmt.Sprintf("%s bind.from %q", name, from))
			}
			for target, source := range bind.Fields {
				for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
					unresolved = append(unresolved, fmt.Sprintf("%s.%s references %q", name, target, ref))
				}
			}
		}
		for label, source := range map[string]string{
			"when":       step.When,
			"for_each":   step.ForEach,
			"items":      step.Items,
			"batch_size": step.BatchSize,
		} {
			for _, ref := range unresolvedDataFlowReferences(source, stepNames, inputs) {
				unresolved = append(unresolved, fmt.Sprintf("%s %s references %q", name, label, ref))
			}
		}
	})
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		for _, ref := range unresolvedDataFlowReferences(output.From, stepNames, inputs) {
			name := strings.TrimSpace(output.Name)
			if name == "" {
				name = "<unnamed>"
			}
			unresolved = append(unresolved, fmt.Sprintf("output %s references %q", name, ref))
		}
	}
	if len(unresolved) > 0 {
		return fmt.Errorf("%s. Use declared step names, inputs, or credential binding names only.", strings.Join(sortedCopy(unresolved), "; "))
	}
	return nil
}

func unresolvedDataFlowReferences(source string, stepNames, inputs map[string]bool) []string {
	var out []string
	for _, ref := range dataFlowReferencePrefixes(source) {
		lower := strings.ToLower(ref)
		if stepNames[ref] || inputs[ref] ||
			lower == "input" || lower == "inputs" || lower == "var" || lower == "vars" ||
			lower == "each" ||
			lower == "workflow" || lower == "trigger" || lower == "security" || lower == "credentials" ||
			lower == "body" || lower == "received_body" || lower == "request" || lower == "response" {
			continue
		}
		out = append(out, ref)
	}
	return sortedUnique(out)
}

func dataFlowReferencePrefixes(source string) []string {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	re := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_-]*)\s*\.`)
	matches := re.FindAllStringSubmatchIndex(source, -1)
	var out []string
	for _, match := range matches {
		if len(match) < 4 || dataFlowReferenceIsLiteralDomain(source, match[0]) {
			continue
		}
		out = append(out, source[match[2]:match[3]])
	}
	return sortedUnique(out)
}

func dataFlowReferenceIsLiteralDomain(source string, start int) bool {
	if start <= 0 || start > len(source) {
		return false
	}
	switch source[start-1] {
	case '@', '/', ':', '.':
		return true
	default:
		return false
	}
}

func intentStepNameSet(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	if intent == nil {
		return out
	}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			out[strings.TrimSpace(step.Name)] = true
		}
	})
	return out
}

type responsePathValidation struct {
	Failures []string
	Warnings []string
}

func validateIntentResponsePaths(intent *rollout.Intent, candidates []openapidisco.Candidate, primary string) responsePathValidation {
	var result responsePathValidation
	if intent == nil {
		return result
	}
	ops := openAPIOperationIndex(candidates)
	stepOps := map[string]*rollout.OperationInfo{}
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil || strings.TrimSpace(step.Name) == "" || strings.TrimSpace(step.Operation) == "" {
			return
		}
		op := ops[operationKey(intentStepOpenAPIPath(intent, step, primary), step.Operation)]
		if op != nil {
			stepOps[strings.TrimSpace(step.Name)] = op
		}
	})
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		for target, source := range step.With {
			result.addResponsePathChecks(fmt.Sprintf("%s.%s", name, target), source, stepOps)
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			for target, source := range bind.Fields {
				checkSource := strings.TrimSpace(source)
				from := strings.TrimSpace(bind.From)
				if from != "" && (strings.HasPrefix(checkSource, "body") || strings.HasPrefix(checkSource, "received_body")) {
					checkSource = from + "." + checkSource
				}
				result.addResponsePathChecks(fmt.Sprintf("%s.%s", name, target), checkSource, stepOps)
			}
		}
	})
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		name := strings.TrimSpace(output.Name)
		if name == "" {
			name = "<unnamed>"
		}
		result.addResponsePathChecks("output "+name, output.From, stepOps)
	}
	return result
}

func (r *responsePathValidation) addResponsePathChecks(label, source string, stepOps map[string]*rollout.OperationInfo) {
	for _, ref := range responsePathReferences(source) {
		op := stepOps[ref.Step]
		if op == nil {
			continue
		}
		switch responsePathStatus(op, ref.Path) {
		case "missing":
			r.Failures = append(r.Failures, fmt.Sprintf("%s references missing response path %s.%s", label, ref.Step, ref.Path))
		case "opaque":
			r.Warnings = append(r.Warnings, fmt.Sprintf("%s references unverified response path %s.%s", label, ref.Step, ref.Path))
		}
	}
}

type responsePathReference struct {
	Step string
	Path string
}

func responsePathReferences(source string) []responsePathReference {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	matches := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_-]*)\.(?:received_body|body)([A-Za-z0-9_\.\[\]-]*)`).FindAllStringSubmatch(source, -1)
	var out []responsePathReference
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		path := strings.TrimPrefix(match[2], ".")
		if path != "" {
			out = append(out, responsePathReference{Step: match[1], Path: path})
		}
	}
	return out
}

func responsePathStatus(op *rollout.OperationInfo, path string) string {
	schema := preferredResponseSchema(op)
	if len(schema) == 0 {
		return "opaque"
	}
	return schemaPathStatus(schema, responsePathTokens(path))
}

func preferredResponseSchema(op *rollout.OperationInfo) map[string]any {
	if op == nil {
		return nil
	}
	for _, code := range []string{"200", "201", "202", "204", "default"} {
		if response := op.Responses[code]; response != nil && len(response.Schema) > 0 {
			return response.Schema
		}
	}
	codes := make([]string, 0, len(op.Responses))
	for code := range op.Responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if response := op.Responses[code]; response != nil && len(response.Schema) > 0 {
			return response.Schema
		}
	}
	return nil
}

func responsePathTokens(path string) []string {
	path = strings.Trim(strings.TrimSpace(path), ".")
	if path == "" {
		return nil
	}
	path = regexp.MustCompile(`\[[^\]]+\]`).ReplaceAllString(path, "")
	path = strings.Trim(path, ".")
	if path == "" {
		return nil
	}
	return strings.Split(path, ".")
}

func schemaHasPath(schema map[string]any, tokens []string) bool {
	return schemaPathStatus(schema, tokens) == "present"
}

func schemaPathStatus(schema map[string]any, tokens []string) string {
	if len(tokens) == 0 {
		return "present"
	}
	if len(schema) == 0 {
		return "missing"
	}
	if strings.TrimSpace(asString(schema["$ref"])) != "" {
		return "opaque"
	}
	if strings.EqualFold(asString(schema["type"]), "array") {
		return schemaPathStatus(asMap(schema["items"]), tokens)
	}
	props := asMap(schema["properties"])
	if len(props) == 0 {
		return "missing"
	}
	next, ok := props[tokens[0]]
	if !ok {
		return "missing"
	}
	return schemaPathStatus(asMap(next), tokens[1:])
}
