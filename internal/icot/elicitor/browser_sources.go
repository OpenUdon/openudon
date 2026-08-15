package elicitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/browsertools/bundle"
	"github.com/OpenUdon/browsertools/profile"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const browserSourceFamily = "browser-profile"

type browserAuthoringDiscovery struct {
	Report browsertools.LocalSourceDiscoveryReport
	Docs   []APIDocument
	Plans  []SourceMaterialization
}

func discoverBrowserAuthoringSources(ctx context.Context, exampleDir string, explicit []BrowserSourceInput, roots []string, at time.Time) (browserAuthoringDiscovery, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	browserRoots := append([]string(nil), roots...)
	for _, dir := range []string{"browser-profiles", "capability-bundles"} {
		path := filepath.Join(exampleDir, dir)
		if _, err := os.Lstat(path); err == nil {
			browserRoots = append(browserRoots, path)
		} else if !os.IsNotExist(err) {
			return browserAuthoringDiscovery{}, err
		}
	}
	explicitIDs := map[string]string{}
	for _, source := range explicit {
		id := strings.TrimSpace(source.ID)
		path := strings.TrimSpace(source.Path)
		if id == "" || path == "" {
			return browserAuthoringDiscovery{}, fmt.Errorf("browser source ID and path are required")
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return browserAuthoringDiscovery{}, err
		}
		absolute = filepath.Clean(absolute)
		if prior, exists := explicitIDs[absolute]; exists && prior != id {
			return browserAuthoringDiscovery{}, fmt.Errorf("browser profile %s has conflicting IDs %q and %q", absolute, prior, id)
		}
		explicitIDs[absolute] = id
		browserRoots = append(browserRoots, absolute)
	}
	if len(browserRoots) == 0 {
		return browserAuthoringDiscovery{Report: emptyBrowserDiscoveryReport()}, nil
	}
	report, err := browsertools.DiscoverLocalSources(ctx, browsertools.LocalSourceDiscoveryOptions{
		Roots: browserRoots,
		At:    at,
	})
	result := browserAuthoringDiscovery{Report: report}
	if err != nil {
		return result, err
	}
	for _, candidate := range report.Candidates {
		if candidate.Status != "active" {
			continue
		}
		plan, doc, materializeErr := browserMaterializationForCandidate(exampleDir, candidate, explicitIDs, at)
		if materializeErr != nil {
			return result, materializeErr
		}
		result.Plans = append(result.Plans, plan)
		result.Docs = append(result.Docs, doc)
	}
	sort.SliceStable(result.Docs, func(i, j int) bool {
		return result.Docs[i].RelativePath < result.Docs[j].RelativePath
	})
	result.Plans = normalizeSourcePlan(result.Plans)
	return result, nil
}

func emptyBrowserDiscoveryReport() browsertools.LocalSourceDiscoveryReport {
	return browsertools.LocalSourceDiscoveryReport{
		Candidates: []browsertools.LocalSourceCandidate{},
		Rejected:   []browsertools.LocalSourceDiagnostic{},
		Ambiguous:  []browsertools.LocalSourceDiagnostic{},
		Truncated:  []browsertools.LocalSourceDiagnostic{},
	}
}

func browserMaterializationForCandidate(exampleDir string, candidate browsertools.LocalSourceCandidate, explicitIDs map[string]string, at time.Time) (SourceMaterialization, APIDocument, error) {
	absSource, err := filepath.Abs(candidate.Path)
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, err
	}
	data, err := readStableBrowserCandidate(absSource, candidate.Digest)
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, err
	}
	var value *profile.Profile
	materialized := data
	sourceKind := string(candidate.Kind)
	release := candidate.Release
	id := strings.TrimSpace(explicitIDs[filepath.Clean(absSource)])
	if candidate.Kind == browsertools.LocalSourceBundle {
		capability, parseErr := bundle.Parse(data)
		if parseErr != nil {
			return SourceMaterialization{}, APIDocument{}, parseErr
		}
		if verifyErr := bundle.Verify(capability, at); verifyErr != nil {
			return SourceMaterialization{}, APIDocument{}, verifyErr
		}
		value = &capability.Payload.Profile
		if id == "" {
			id = capability.Payload.Identity.ID
		}
		if release == "" {
			release = capability.Payload.Identity.Release
		}
		materialized, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			return SourceMaterialization{}, APIDocument{}, err
		}
		materialized = append(materialized, '\n')
	} else {
		value, err = parseBrowserProfile(absSource, data)
		if err != nil {
			return SourceMaterialization{}, APIDocument{}, err
		}
	}
	if err := validateBrowserAuthoringProfile(value); err != nil {
		return SourceMaterialization{}, APIDocument{}, fmt.Errorf("browser source %s: %w", absSource, err)
	}
	if id == "" {
		id = strings.TrimSpace(candidate.ID)
	}
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(absSource), filepath.Ext(absSource))
	}
	id = strings.Trim(sourceIDSanitizer.ReplaceAllString(id, "-"), ".-")
	if id == "" {
		return SourceMaterialization{}, APIDocument{}, fmt.Errorf("cannot derive a stable browser source ID for %s", absSource)
	}
	ext := strings.ToLower(filepath.Ext(absSource))
	if candidate.Kind == browsertools.LocalSourceBundle || ext == "" {
		ext = ".json"
	}
	target := filepath.ToSlash(filepath.Join("browser-profiles", id+ext))
	if candidate.Kind == browsertools.LocalSourceProfile {
		if rel, relErr := filepath.Rel(exampleDir, absSource); relErr == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && strings.HasPrefix(filepath.ToSlash(rel), "browser-profiles/") {
			target = filepath.ToSlash(rel)
		}
	}
	digest := sha256.Sum256(materialized)
	expiresAt, err := browserProfileExpiry(value)
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, err
	}
	plan := SourceMaterialization{
		Kind: browserSourceFamily, SourceKind: sourceKind, ID: id, Release: release,
		SourcePath: absSource, TargetPath: target, SHA256: hex.EncodeToString(digest[:]),
		SourceSHA256: strings.TrimPrefix(strings.ToLower(candidate.Digest), "sha256:"),
		Title:        value.Info.Title, OperationCount: len(value.Actions), Actions: value.SortedActionNames(),
		Origins: append([]string(nil), value.Info.Origin...), Lifecycle: "active",
		ExpiresAt: expiresAt.Format(time.RFC3339), LoginStateRequired: value.Info.LoginStateRequired,
		Provenance:          firstNonEmpty(candidate.Provenance, "local:"+filepath.ToSlash(absSource)),
		MaterializedContent: append([]byte(nil), materialized...),
	}
	doc := browserProfileDocument(plan, value)
	return plan, doc, nil
}

func parseBrowserProfile(path string, data []byte) (*profile.Profile, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return profile.ParseJSON(data)
	case ".yaml", ".yml":
		return profile.ParseYAML(data)
	default:
		return nil, fmt.Errorf("browser profile must use .json, .yaml, or .yml")
	}
}

func readStableBrowserCandidate(path, expected string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("browser source must remain a non-symlink regular file: %s", path)
	}
	if before.Size() > browsertools.DefaultLocalMaxBytes {
		return nil, fmt.Errorf("browser source exceeds %d bytes: %s", browsertools.DefaultLocalMaxBytes, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	actual := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	if strings.TrimSpace(expected) != "" && actual != strings.ToLower(strings.TrimSpace(expected)) {
		return nil, fmt.Errorf("browser source %s changed after discovery: digest %s, want %s", path, actual, expected)
	}
	return data, nil
}

// SourceMaterializationContent returns the exact approved package bytes and
// revalidates a local capability bundle when an in-memory profile copy is not
// available (for example after resuming a draft).
func SourceMaterializationContent(source SourceMaterialization, at time.Time) ([]byte, error) {
	if err := validateBrowserMaterializationFreshness(source, at); err != nil {
		return nil, err
	}
	if len(source.MaterializedContent) > 0 {
		data := append([]byte(nil), source.MaterializedContent...)
		if fmt.Sprintf("%x", sha256.Sum256(data)) != strings.ToLower(source.SHA256) {
			return nil, fmt.Errorf("selected browser source materialization content does not match SHA-256 %s", source.SHA256)
		}
		return data, nil
	}
	if source.Kind != browserSourceFamily {
		data, err := os.ReadFile(source.SourcePath)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	expectedSource := strings.TrimSpace(source.SourceSHA256)
	if expectedSource != "" {
		expectedSource = "sha256:" + strings.TrimPrefix(expectedSource, "sha256:")
	}
	data, err := readStableBrowserCandidate(source.SourcePath, expectedSource)
	if err != nil {
		return nil, err
	}
	materialized := data
	if source.SourceKind == string(browsertools.LocalSourceBundle) {
		value, parseErr := bundle.Parse(data)
		if parseErr != nil {
			return nil, parseErr
		}
		if at.IsZero() {
			at = time.Now().UTC()
		}
		if verifyErr := bundle.Verify(value, at); verifyErr != nil {
			return nil, verifyErr
		}
		materialized, err = json.MarshalIndent(value.Payload.Profile, "", "  ")
		if err != nil {
			return nil, err
		}
		materialized = append(materialized, '\n')
	}
	if fmt.Sprintf("%x", sha256.Sum256(materialized)) != strings.ToLower(source.SHA256) {
		return nil, fmt.Errorf("selected browser source %s changed after discovery", source.SourcePath)
	}
	return materialized, nil
}

func validateBrowserMaterializationFreshness(source SourceMaterialization, at time.Time) error {
	if source.Kind != browserSourceFamily {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if !strings.EqualFold(strings.TrimSpace(source.Lifecycle), "active") {
		return fmt.Errorf("selected browser source %s is not active", source.ID)
	}
	expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(source.ExpiresAt))
	if err != nil {
		return fmt.Errorf("selected browser source %s has invalid expiry: %w", source.ID, err)
	}
	if !at.Before(expiresAt) {
		return fmt.Errorf("selected browser source %s expired at %s", source.ID, expiresAt.UTC().Format(time.RFC3339))
	}
	return nil
}

func browserProfileExpiry(value *profile.Profile) (time.Time, error) {
	if value == nil {
		return time.Time{}, fmt.Errorf("browser profile is required")
	}
	verified, err := time.Parse(time.RFC3339, value.Verification.LastVerifiedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("browser profile verification.lastVerifiedAt: %w", err)
	}
	expires, err := value.ExpiresAfter.AddTo(verified)
	if err != nil {
		return time.Time{}, fmt.Errorf("browser profile expiresAfter: %w", err)
	}
	return expires.UTC().Round(0), nil
}

func validateBrowserAuthoringProfile(value *profile.Profile) error {
	if value == nil {
		return fmt.Errorf("browser profile is required")
	}
	for actionName, action := range value.Actions {
		if sensitiveBrowserName(actionName) {
			return fmt.Errorf("action %q is credential, session, or raw-capture shaped", actionName)
		}
		mutating := false
		for _, effect := range action.SideEffects {
			if effect != profile.SideEffectReadOnly {
				mutating = true
			}
		}
		if mutating && (!action.ConfirmationPolicy.Required || strings.TrimSpace(action.ConfirmationPolicy.Prompt) == "") {
			return fmt.Errorf("mutating action %q requires an explicit confirmation prompt", actionName)
		}
		for parameter := range browserSchemaProperties(action.Parameters) {
			if sensitiveBrowserName(parameter) {
				return fmt.Errorf("action %q parameter %q is credential, session, or raw-capture shaped", actionName, parameter)
			}
		}
		for output := range action.Outputs {
			if sensitiveBrowserName(output) {
				return fmt.Errorf("action %q output %q is credential, session, or raw-capture shaped", actionName, output)
			}
		}
	}
	return nil
}

func sensitiveBrowserName(value string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(value))
	for _, marker := range []string{"cookie", "session", "storage", "dom", "html", "screenshot", "raw_capture", "raw_browser", "password", "secret", "credential", "access_token", "refresh_token"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func browserProfileDocument(plan SourceMaterialization, value *profile.Profile) APIDocument {
	doc := APIDocument{
		ID: plan.ID, Path: plan.SourcePath, RelativePath: plan.TargetPath, Title: plan.Title,
		Description: "Verified browser capability profile. API sources remain preferred when they cover the active capability.",
	}
	for _, actionName := range value.SortedActionNames() {
		action := value.Actions[actionName]
		op := apitools.OperationSummary{
			ID: actionName, OperationID: actionName, Method: "BROWSER", Path: "#/actions/" + actionName,
			Summary: action.Description, DocumentName: plan.ID, DocumentPath: plan.SourcePath,
			DocumentRelativePath: plan.TargetPath, Provenance: plan.Provenance,
			Extensions: map[string]string{
				"openudon.source_family":                 browserSourceFamily,
				"openudon.browser.side_effects":          browserSideEffects(action),
				"openudon.browser.confirmation_required": fmt.Sprintf("%t", action.ConfirmationPolicy.Required),
				"openudon.browser.login_state_required":  fmt.Sprintf("%t", value.Info.LoginStateRequired),
				"openudon.browser.expires_at":            plan.ExpiresAt,
			},
		}
		properties := browserSchemaProperties(action.Parameters)
		required := browserSchemaRequired(action.Parameters)
		if len(properties) > 0 {
			op.RequestBody = &apitools.RequestBodySummary{ContentTypes: []string{"application/json"}, Schema: &apitools.SchemaSummary{Type: "object"}}
			for _, name := range sortedBrowserSchemaKeys(properties) {
				property, _ := properties[name].(map[string]any)
				field := apitools.RequestFieldSummary{Path: name, Required: required[name], Type: strings.TrimSpace(fmt.Sprint(property["type"])), Description: strings.TrimSpace(fmt.Sprint(property["description"]))}
				op.RequestBody.Fields = append(op.RequestBody.Fields, field)
				if field.Required {
					op.RequestBody.Required = true
					op.RequestBody.RequiredFieldPaths = append(op.RequestBody.RequiredFieldPaths, name)
				}
			}
		}
		if len(action.Outputs) > 0 {
			op.ResponseBody = &apitools.ResponseBodySummary{StatusCode: "success", ContentTypes: []string{"application/json"}, Schema: &apitools.SchemaSummary{Type: "object"}}
			for _, name := range sortedBrowserOutputKeys(action.Outputs) {
				output := action.Outputs[name]
				op.ResponseBody.Fields = append(op.ResponseBody.Fields, apitools.RequestFieldSummary{Path: name, Type: string(output.Type)})
			}
		}
		doc.Operations = append(doc.Operations, op)
	}
	return doc
}

func browserSchemaProperties(schema profile.JSONSchema) map[string]any {
	if schema == nil {
		return nil
	}
	properties, _ := schema["properties"].(map[string]any)
	return properties
}

func browserSchemaRequired(schema profile.JSONSchema) map[string]bool {
	result := map[string]bool{}
	if schema == nil {
		return result
	}
	switch values := schema["required"].(type) {
	case []any:
		for _, value := range values {
			result[strings.TrimSpace(fmt.Sprint(value))] = true
		}
	case []string:
		for _, value := range values {
			result[strings.TrimSpace(value)] = true
		}
	}
	return result
}

func browserSideEffects(action profile.Action) string {
	values := make([]string, 0, len(action.SideEffects))
	for _, effect := range action.SideEffects {
		values = append(values, string(effect))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func sortedBrowserSchemaKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedBrowserOutputKeys(values map[string]profile.Output) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isBrowserDocument(doc APIDocument) bool {
	if strings.HasPrefix(filepath.ToSlash(strings.TrimSpace(doc.RelativePath)), "browser-profiles/") {
		return true
	}
	for _, operation := range doc.Operations {
		if operation.Extensions["openudon.source_family"] == browserSourceFamily {
			return true
		}
	}
	return false
}

func selectedBrowserOperation(session Session, docs []APIDocument) (*rollout.Step, APIDocument, *apitools.OperationSummary) {
	var selectedStep *rollout.Step
	var selectedDoc APIDocument
	var selectedOperation *apitools.OperationSummary
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if selectedStep != nil || step == nil {
			return
		}
		ref := stepAPISourceRef(session, step)
		for _, doc := range docs {
			if !isBrowserDocument(doc) || doc.RelativePath != ref {
				continue
			}
			selectedStep = step
			selectedDoc = doc
			if operation, ok := operationByID([]APIDocument{doc}, doc.RelativePath, step.Operation); ok {
				copy := *operation
				selectedOperation = &copy
			}
			return
		}
	})
	return selectedStep, selectedDoc, selectedOperation
}

func browserOperationMutates(operation *apitools.OperationSummary) bool {
	if operation == nil {
		return false
	}
	for _, effect := range strings.Split(operation.Extensions["openudon.browser.side_effects"], ",") {
		if effect = strings.TrimSpace(effect); effect != "" && effect != string(profile.SideEffectReadOnly) {
			return true
		}
	}
	return false
}

func isBrowserOperationSummary(operation *apitools.OperationSummary) bool {
	return operation != nil && operation.Extensions["openudon.source_family"] == browserSourceFamily
}

func stringSliceContains(values []string, wanted string) bool {
	wanted = strings.TrimSpace(wanted)
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}

func filterCrossFamilyDiagnostics(apiReport *apitools.LocalSourceDiscoveryReport, browserReport *browsertools.LocalSourceDiscoveryReport) {
	if apiReport == nil || browserReport == nil {
		return
	}
	apiPaths := map[string]bool{}
	for _, candidate := range apiReport.Candidates {
		apiPaths[cleanDiscoveryPath(candidate.Path)] = true
	}
	browserPaths := map[string]bool{}
	for _, candidate := range browserReport.Candidates {
		browserPaths[cleanDiscoveryPath(candidate.Path)] = true
	}
	apiAmbiguous := apiReport.Ambiguous[:0]
	for _, diagnostic := range apiReport.Ambiguous {
		if !browserPaths[cleanDiscoveryPath(diagnostic.Path)] {
			apiAmbiguous = append(apiAmbiguous, diagnostic)
		}
	}
	apiReport.Ambiguous = apiAmbiguous
	browserAmbiguous := browserReport.Ambiguous[:0]
	for _, diagnostic := range browserReport.Ambiguous {
		if !apiPaths[cleanDiscoveryPath(diagnostic.Path)] {
			browserAmbiguous = append(browserAmbiguous, diagnostic)
		}
	}
	browserReport.Ambiguous = browserAmbiguous
}

func cleanDiscoveryPath(path string) string {
	absolute, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(absolute)
}
