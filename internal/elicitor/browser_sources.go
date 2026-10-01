package elicitor

import (
	"bytes"
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
	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/bundle"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/evidence/redact"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserauthentication"
)

const (
	browserSourceFamily               = "browser-profile"
	browserAuthenticationSourceFamily = "browser-authentication"
	browserRegistrationSourceFamily   = "browser-registration"
)

type browserAuthoringDiscovery struct {
	Report browsertools.LocalSourceDiscoveryReport
	Docs   []APIDocument
	Plans  []SourceMaterialization
}

func discoverBrowserAuthoringSources(ctx context.Context, exampleDir string, explicit []BrowserSourceInput, roots []string, at time.Time) (browserAuthoringDiscovery, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if len(explicit) > browsertools.DefaultLocalMaxCandidates {
		return browserAuthoringDiscovery{}, fmt.Errorf("explicit browser sources exceed limit %d; narrow --browser-profile inputs", browsertools.DefaultLocalMaxCandidates)
	}
	browserRoots := append([]string(nil), roots...)
	for _, dir := range sourcecatalog.Browser() {
		path := filepath.Join(exampleDir, dir)
		if _, err := os.Lstat(path); err == nil {
			browserRoots = append(browserRoots, path)
		} else if !os.IsNotExist(err) {
			return browserAuthoringDiscovery{}, err
		}
	}
	explicitIDs := map[string]string{}
	var guidedCandidates []browsertools.LocalSourceCandidate
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
		} else if exists {
			continue
		}
		explicitIDs[absolute] = id
		guided, recognized, inspectErr := inspectExplicitGuidedBundle(absolute, at)
		if inspectErr != nil {
			return browserAuthoringDiscovery{}, fmt.Errorf("browser source %s: %w", absolute, inspectErr)
		}
		if recognized {
			guidedCandidates = append(guidedCandidates, guided)
			continue
		}
		browserRoots = append(browserRoots, absolute)
	}
	if len(browserRoots) == 0 && len(guidedCandidates) == 0 {
		return browserAuthoringDiscovery{Report: emptyBrowserDiscoveryReport()}, nil
	}
	report := emptyBrowserDiscoveryReport()
	var err error
	if len(browserRoots) > 0 {
		report, err = browsertools.DiscoverLocalSources(ctx, browsertools.LocalSourceDiscoveryOptions{
			Roots: browserRoots,
			At:    at,
		})
	}
	result := browserAuthoringDiscovery{Report: report}
	if err != nil {
		return result, err
	}
	if len(guidedCandidates) > 0 {
		if result.Report.Visited+len(guidedCandidates) > browsertools.DefaultLocalMaxVisited {
			return result, fmt.Errorf("browser source visits exceed limit %d; narrow explicit --browser-profile inputs", browsertools.DefaultLocalMaxVisited)
		}
		guidedPaths := map[string]bool{}
		sort.SliceStable(guidedCandidates, func(i, j int) bool {
			return guidedCandidates[i].Path < guidedCandidates[j].Path
		})
		for _, candidate := range guidedCandidates {
			guidedPaths[filepath.Clean(candidate.Path)] = true
			result.Report.Roots = append(result.Report.Roots, candidate.Path)
		}
		result.Report.Visited += len(guidedCandidates)
		result.Report.Roots = sortedUniqueBrowserStrings(result.Report.Roots)
		result.Report.Rejected = removeGuidedPathDiagnostics(result.Report.Rejected, guidedPaths)
		result.Report.Ambiguous = removeGuidedPathDiagnostics(result.Report.Ambiguous, guidedPaths)
		byDigest := map[string]int{}
		for index, candidate := range result.Report.Candidates {
			byDigest[candidate.Digest] = index
		}
		for _, candidate := range guidedCandidates {
			if priorIndex, duplicate := byDigest[candidate.Digest]; duplicate {
				prior := result.Report.Candidates[priorIndex]
				if candidate.Path < prior.Path {
					result.Report.Candidates[priorIndex] = candidate
					result.Report.Rejected = append(result.Report.Rejected, browsertools.LocalSourceDiagnostic{Path: prior.Path, Code: "duplicate", Detail: "identical content was already discovered", DuplicateOf: candidate.Path})
				} else {
					result.Report.Rejected = append(result.Report.Rejected, browsertools.LocalSourceDiagnostic{Path: candidate.Path, Code: "duplicate", Detail: "identical content was already discovered", DuplicateOf: prior.Path})
				}
				continue
			}
			if len(result.Report.Candidates) >= browsertools.DefaultLocalMaxCandidates {
				return result, fmt.Errorf("browser source candidates exceed limit %d; narrow explicit --browser-profile inputs", browsertools.DefaultLocalMaxCandidates)
			}
			byDigest[candidate.Digest] = len(result.Report.Candidates)
			result.Report.Candidates = append(result.Report.Candidates, candidate)
		}
		sort.SliceStable(result.Report.Candidates, func(i, j int) bool {
			return result.Report.Candidates[i].Path < result.Report.Candidates[j].Path
		})
		sort.SliceStable(result.Report.Rejected, func(i, j int) bool {
			return result.Report.Rejected[i].Path < result.Report.Rejected[j].Path
		})
	}
	for _, candidate := range result.Report.Candidates {
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

func removeGuidedPathDiagnostics(values []browsertools.LocalSourceDiagnostic, guidedPaths map[string]bool) []browsertools.LocalSourceDiagnostic {
	kept := values[:0]
	for _, diagnostic := range values {
		if !guidedPaths[filepath.Clean(diagnostic.Path)] {
			kept = append(kept, diagnostic)
		}
	}
	return kept
}

func sortedUniqueBrowserStrings(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
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
	if candidate.Kind == browsertools.LocalSourceAuthenticationProfile {
		return browserAuthenticationMaterialization(exampleDir, absSource, data, candidate, explicitIDs, at)
	}
	var value *profile.Profile
	materialized := data
	sourceKind := string(candidate.Kind)
	release := candidate.Release
	id := strings.TrimSpace(explicitIDs[filepath.Clean(absSource)])
	if candidate.Kind == browserGuidedSourceKind {
		value, materialized, err = materializeBrowserGuidedProfile(data, at)
		if err != nil {
			return SourceMaterialization{}, APIDocument{}, err
		}
	} else if candidate.Kind == browsertools.LocalSourceBundle {
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
	if candidate.Kind == browsertools.LocalSourceBundle || candidate.Kind == browserGuidedSourceKind || ext == "" {
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

func browserAuthenticationMaterialization(exampleDir, absSource string, data []byte, candidate browsertools.LocalSourceCandidate, explicitIDs map[string]string, at time.Time) (SourceMaterialization, APIDocument, error) {
	value, err := authprofile.Parse(data)
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, err
	}
	if err := authprofile.ValidateAt(value, at); err != nil {
		return SourceMaterialization{}, APIDocument{}, err
	}
	id := strings.TrimSpace(explicitIDs[filepath.Clean(absSource)])
	if id == "" {
		id = strings.TrimSpace(candidate.ID)
	}
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(absSource), filepath.Ext(absSource))
	}
	id = strings.Trim(sourceIDSanitizer.ReplaceAllString(id, "-"), ".-")
	if id == "" {
		return SourceMaterialization{}, APIDocument{}, fmt.Errorf("cannot derive a stable browser authentication source ID for %s", absSource)
	}
	ext := strings.ToLower(filepath.Ext(absSource))
	if ext == "" {
		ext = ".yaml"
	}
	target := filepath.ToSlash(filepath.Join("browser-authentication", id+ext))
	if rel, relErr := filepath.Rel(exampleDir, absSource); relErr == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && strings.HasPrefix(filepath.ToSlash(rel), "browser-authentication/") {
		target = filepath.ToSlash(rel)
	}
	digest := sha256.Sum256(data)
	expiresAt, err := authprofile.ExpiresAt(value)
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, err
	}
	flowSlots := map[string][]string{}
	for _, flowName := range authprofile.SortedFlowNames(value) {
		flowSlots[flowName] = browserAuthenticationFlowSlots(value.Flows[flowName])
	}
	plan := SourceMaterialization{
		Kind: browserAuthenticationSourceFamily, SourceKind: string(candidate.Kind), ID: id,
		SourcePath: absSource, TargetPath: target, SHA256: hex.EncodeToString(digest[:]),
		SourceSHA256: strings.TrimPrefix(strings.ToLower(candidate.Digest), "sha256:"),
		Title:        value.Info.Title, OperationCount: len(value.Flows), Flows: authprofile.SortedFlowNames(value),
		FlowCredentialSlots: flowSlots, Origins: authprofile.Origins(value), Lifecycle: "active",
		ExpiresAt: expiresAt.Format(time.RFC3339), Provenance: firstNonEmpty(candidate.Provenance, "local:"+filepath.ToSlash(absSource)),
		MaterializedContent: append([]byte(nil), data...),
	}
	return plan, browserAuthenticationDocument(plan, value), nil
}

func browserAuthenticationFlowSlots(flow browserauthentication.Flow) []string {
	var slots []string
	for _, step := range flow.Sequence {
		if step.TypeCredential != nil {
			slots = append(slots, step.TypeCredential.Slot)
		}
		if step.Challenge != nil && strings.TrimSpace(step.Challenge.Slot) != "" {
			slots = append(slots, step.Challenge.Slot)
		}
	}
	return dedupeStrings(slots)
}

func browserAuthenticationDocument(plan SourceMaterialization, value *authprofile.Profile) APIDocument {
	doc := APIDocument{
		ID: plan.ID, Path: plan.SourcePath, RelativePath: plan.TargetPath, Title: plan.Title,
		Description: "Verified, secret-free browser sign-in profile. Authentication requires separate authoring and runtime approval.",
	}
	for _, flowName := range authprofile.SortedFlowNames(value) {
		flow := value.Flows[flowName]
		effects := append([]string(nil), flow.Effects...)
		sort.Strings(effects)
		doc.Operations = append(doc.Operations, apitools.OperationSummary{
			ID: flowName, OperationID: flowName, Method: "BROWSER_AUTHENTICATION", Path: "#/flows/" + flowName,
			Summary: firstNonEmpty(flow.Description, "Establish a reviewed browser session."), DocumentName: plan.ID,
			DocumentPath: plan.SourcePath, DocumentRelativePath: plan.TargetPath, Provenance: plan.Provenance,
			Extensions: map[string]string{
				"openudon.source_family":                           browserAuthenticationSourceFamily,
				"openudon.browser_authentication.credential_slots": strings.Join(plan.FlowCredentialSlots[flowName], ","),
				"openudon.browser_authentication.effects":          strings.Join(effects, ","),
				"openudon.browser.expires_at":                      plan.ExpiresAt,
			},
		})
	}
	return doc
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
	data, _, err := evidencefile.ReadRegular(path, browsertools.DefaultLocalMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("read stable browser source: %w", err)
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
	if strings.HasPrefix(source.SourcePath, virtualBrowserPrefix) {
		return nil, fmt.Errorf("selected virtual browser source %s is unavailable from the current in-memory catalog", source.ID)
	}
	if source.Kind != browserSourceFamily && source.Kind != browserAuthenticationSourceFamily {
		data, _, err := evidencefile.ReadRegular(source.SourcePath, apitools.DefaultMaxBytes)
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
	if source.Kind == browserAuthenticationSourceFamily {
		value, parseErr := authprofile.Parse(data)
		if parseErr != nil {
			return nil, parseErr
		}
		if at.IsZero() {
			at = time.Now().UTC()
		}
		if validateErr := authprofile.ValidateAt(value, at); validateErr != nil {
			return nil, validateErr
		}
	}
	if source.SourceKind == string(browserGuidedSourceKind) {
		if at.IsZero() {
			at = time.Now().UTC()
		}
		_, materialized, err = materializeBrowserGuidedProfile(data, at)
		if err != nil {
			return nil, err
		}
	} else if source.SourceKind == string(browsertools.LocalSourceBundle) {
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

// SourceMaterializationReviewContent returns the exact independently reviewed
// registration bundle retained in memory until the ordinary package approval
// boundary. Resumed virtual sources must be rediscovered before it is available.
func SourceMaterializationReviewContent(source SourceMaterialization, at time.Time) ([]byte, error) {
	if source.Kind != browserRegistrationSourceFamily || strings.TrimSpace(source.ReviewPath) == "" || strings.TrimSpace(source.ReviewSHA256) == "" {
		return nil, fmt.Errorf("selected browser source %s has no registration review materialization", source.ID)
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	if err := validateBrowserMaterializationFreshness(source, at); err != nil {
		return nil, err
	}
	if len(source.MaterializedReview) == 0 {
		return nil, fmt.Errorf("selected virtual browser registration review %s is unavailable from the current in-memory catalog", source.ID)
	}
	data := append([]byte(nil), source.MaterializedReview...)
	if fmt.Sprintf("%x", sha256.Sum256(data)) != strings.ToLower(source.ReviewSHA256) {
		return nil, fmt.Errorf("selected browser registration review does not match SHA-256 %s", source.ReviewSHA256)
	}
	profileData, err := SourceMaterializationContent(source, at)
	if err != nil {
		return nil, err
	}
	profileData = bytes.TrimSuffix(profileData, []byte{'\n'})
	reviewData := bytes.TrimSuffix(data, []byte{'\n'})
	if _, _, err := canonicalVirtualRegistration(profileData, reviewData, at); err != nil {
		return nil, fmt.Errorf("selected browser registration review is invalid: %w", err)
	}
	return data, nil
}

func validateBrowserMaterializationFreshness(source SourceMaterialization, at time.Time) error {
	if source.Kind != browserSourceFamily && source.Kind != browserAuthenticationSourceFamily && source.Kind != browserRegistrationSourceFamily {
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
	if redact.SensitiveKey(normalized) {
		return true
	}
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
	path := filepath.ToSlash(strings.TrimSpace(doc.RelativePath))
	if strings.HasPrefix(path, "browser-profiles/") || strings.HasPrefix(path, "browser-authentication/") || strings.HasPrefix(path, "browser-registration/") {
		return true
	}
	for _, operation := range doc.Operations {
		if operation.Extensions["openudon.source_family"] == browserSourceFamily || operation.Extensions["openudon.source_family"] == browserAuthenticationSourceFamily || operation.Extensions["openudon.source_family"] == browserRegistrationSourceFamily {
			return true
		}
	}
	return false
}

func selectedBrowserOperation(session Session, docs []APIDocument) (*rollout.Step, APIDocument, *apitools.OperationSummary) {
	selected := selectedBrowserOperations(session, docs)
	if len(selected) == 0 {
		return nil, APIDocument{}, nil
	}
	return selected[0].Step, selected[0].Document, selected[0].Operation
}

type selectedBrowserAction struct {
	Step      *rollout.Step
	Document  APIDocument
	Operation *apitools.OperationSummary
}

func selectedBrowserOperations(session Session, docs []APIDocument) []selectedBrowserAction {
	var selected []selectedBrowserAction
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "browser") {
			return
		}
		item := selectedBrowserAction{Step: step}
		ref := stepAPISourceRef(session, step)
		for _, doc := range docs {
			if !isBrowserActionDocument(doc) || doc.RelativePath != ref {
				continue
			}
			item.Document = doc
			if operation, ok := operationByID([]APIDocument{doc}, doc.RelativePath, step.Operation); ok {
				copy := *operation
				item.Operation = &copy
			}
			break
		}
		selected = append(selected, item)
	})
	return selected
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

func isBrowserAuthenticationOperationSummary(operation *apitools.OperationSummary) bool {
	return operation != nil && operation.Extensions["openudon.source_family"] == browserAuthenticationSourceFamily
}

func isBrowserActionDocument(doc APIDocument) bool {
	for i := range doc.Operations {
		if isBrowserOperationSummary(&doc.Operations[i]) {
			return true
		}
	}
	return false
}

func isBrowserAuthenticationDocument(doc APIDocument) bool {
	for i := range doc.Operations {
		if isBrowserAuthenticationOperationSummary(&doc.Operations[i]) {
			return true
		}
	}
	return false
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
