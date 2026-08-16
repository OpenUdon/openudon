package icot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/evidence/redact"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/uws/browserauthentication"
	"github.com/OpenUdon/uws/schemas"
)

const authenticatedAuthoringReviewVersion = "openudon.authenticated-browser-authoring-review.v1"

type liveContext struct {
	Kind   string `json:"kind"`
	Parent string `json:"parent"`
	Origin string `json:"origin"`
	Path   string `json:"path,omitempty"`
	Name   string `json:"name,omitempty"`
}

type liveGoalProof struct {
	Origin  string `json:"origin"`
	Path    string `json:"path"`
	Context string `json:"context"`
	Role    string `json:"role"`
	Label   string `json:"label,omitempty"`
	Matches int    `json:"matches"`
}

type liveTraceStep struct {
	Kind         string `json:"kind"`
	Phase        string `json:"phase"`
	CandidateID  string `json:"candidateId,omitempty"`
	Context      string `json:"context,omitempty"`
	Role         string `json:"role,omitempty"`
	Label        string `json:"label,omitempty"`
	InputKind    string `json:"inputKind,omitempty"`
	URL          string `json:"url,omitempty"`
	POSTBudget   int    `json:"postBudget,omitempty"`
	POSTObserved int    `json:"postObserved,omitempty"`
	OpensContext string `json:"opensContext,omitempty"`
}

type liveProfileReview struct {
	Schema        string   `json:"schema"`
	Kind          string   `json:"kind"`
	ProfileDigest string   `json:"profileDigest"`
	AssessedAt    string   `json:"assessedAt"`
	Decisions     []string `json:"decisions"`
}

type authenticatedAuthoringEnvelope struct {
	Schema                string                 `json:"schema"`
	ObservedAt            string                 `json:"observedAt"`
	Goal                  string                 `json:"goal"`
	GoalPredicate         liveGoalPredicate      `json:"goalPredicate"`
	GoalProof             liveGoalProof          `json:"goalProof"`
	HumanConfirmed        bool                   `json:"humanConfirmed"`
	Origins               []string               `json:"origins"`
	Contexts              map[string]liveContext `json:"contexts"`
	Bounds                liveBounds             `json:"bounds"`
	Trace                 []liveTraceStep        `json:"trace"`
	AuthenticationProfile json.RawMessage        `json:"authenticationProfile"`
	AuthenticationReview  liveProfileReview      `json:"authenticationReview"`
	CapabilityProfile     json.RawMessage        `json:"capabilityProfile"`
	CapabilityReview      liveProfileReview      `json:"capabilityReview"`
	Diagnostics           []string               `json:"diagnostics"`
}

type authenticatedAuthoringSafeReview struct {
	Version              string                 `json:"version"`
	EnvelopeSHA256       string                 `json:"envelope_sha256"`
	ObservedAt           string                 `json:"observed_at"`
	Goal                 string                 `json:"goal"`
	GoalPredicate        liveGoalPredicate      `json:"goal_predicate"`
	Origins              []string               `json:"origins"`
	Contexts             map[string]liveContext `json:"contexts"`
	Bounds               liveBounds             `json:"bounds"`
	TraceSteps           int                    `json:"trace_steps"`
	AuthenticationReview liveProfileReview      `json:"authentication_review"`
	CapabilityReview     liveProfileReview      `json:"capability_review"`
	Diagnostics          []string               `json:"diagnostics"`
	PrivateEnvelopeKept  bool                   `json:"private_envelope_kept_outside_package"`
}

type preparedAuthenticatedImport struct {
	ExampleDir           string
	AuthenticationTarget string
	CapabilityTarget     string
	AuthenticationSchema string
	CapabilitySchema     string
	EnvelopeDigest       string
	Origins              []string
	Contexts             int
	Goal                 liveGoalPredicate
	Files                []generatedFile
}

func prepareAuthenticatedAuthoringImport(cfg liveAuthorConfig, result liveProtocolResult, at time.Time) (preparedAuthenticatedImport, error) {
	data, actualDigest, err := readStablePrivateAuthorResult(result.ArtifactPath, cfg.PrivateRoot)
	if err != nil {
		return preparedAuthenticatedImport{}, err
	}
	if actualDigest != strings.ToLower(result.Digest) {
		return preparedAuthenticatedImport{}, fmt.Errorf("Browsertools result digest mismatch")
	}
	envelope, err := decodeAuthenticatedAuthoringEnvelope(data)
	if err != nil {
		return preparedAuthenticatedImport{}, err
	}
	if err := validateAuthenticatedAuthoringEnvelope(cfg, envelope, at); err != nil {
		return preparedAuthenticatedImport{}, err
	}
	authenticationBytes, err := canonicalEmbeddedProfile(envelope.AuthenticationProfile)
	if err != nil {
		return preparedAuthenticatedImport{}, fmt.Errorf("authentication profile: %w", err)
	}
	capabilityBytes, err := canonicalEmbeddedProfile(envelope.CapabilityProfile)
	if err != nil {
		return preparedAuthenticatedImport{}, fmt.Errorf("capability profile: %w", err)
	}
	if err := schemas.ValidateBrowserAuthenticationProfile(authenticationBytes); err != nil {
		return preparedAuthenticatedImport{}, fmt.Errorf("authentication profile schema: %w", err)
	}
	if err := schemas.ValidateBrowserSourceProfile(capabilityBytes); err != nil {
		return preparedAuthenticatedImport{}, fmt.Errorf("capability profile schema: %w", err)
	}
	authentication, err := authprofile.Parse(authenticationBytes)
	if err != nil {
		return preparedAuthenticatedImport{}, err
	}
	if err := authprofile.ValidateAt(authentication, at); err != nil {
		return preparedAuthenticatedImport{}, err
	}
	capability, err := profile.ParseJSON(capabilityBytes)
	if err != nil {
		return preparedAuthenticatedImport{}, err
	}
	capabilityExpiry, err := liveCapabilityExpiry(capability)
	if err != nil || !at.Before(capabilityExpiry) {
		return preparedAuthenticatedImport{}, fmt.Errorf("capability profile is stale or has invalid expiry")
	}
	if !allowedAuthenticatedProfilePair(authentication.Profile, capability.Schema) {
		return preparedAuthenticatedImport{}, fmt.Errorf("candidate profile versions cannot be mixed")
	}
	authenticationTarget := filepath.ToSlash(filepath.Join("browser-authentication", cfg.ProfileID+"-auth.json"))
	capabilityTarget := filepath.ToSlash(filepath.Join("browser-profiles", cfg.ProfileID+".json"))
	authenticationStaged := append(append([]byte(nil), authenticationBytes...), '\n')
	capabilityStaged := append(append([]byte(nil), capabilityBytes...), '\n')
	authenticationSum := sha256.Sum256(authenticationStaged)
	capabilitySum := sha256.Sum256(capabilityStaged)
	flowSlots := make(map[string][]string)
	for _, flowName := range authprofile.SortedFlowNames(authentication) {
		flowSlots[flowName] = liveAuthenticationFlowSlots(authentication.Flows[flowName])
	}
	provenance := "browsertools-authenticated-authoring:" + actualDigest
	session := elicitor.Session{
		BrowserRoute: "browser", BrowserSession: "none",
		SourcePlan: []elicitor.SourceMaterialization{
			{
				Kind: "browser-authentication", SourceKind: "authenticated_authoring", ID: cfg.ProfileID + "-auth",
				SourcePath: "private-browsertools-envelope", TargetPath: authenticationTarget,
				SHA256: hex.EncodeToString(authenticationSum[:]), SourceSHA256: strings.TrimPrefix(envelope.AuthenticationReview.ProfileDigest, "sha256:"),
				Title: authentication.Info.Title, OperationCount: len(authentication.Flows), Flows: authprofile.SortedFlowNames(authentication),
				FlowCredentialSlots: flowSlots, Origins: authprofile.Origins(authentication), Lifecycle: "active",
				ExpiresAt: mustLiveAuthenticationExpiry(authentication).Format(time.RFC3339), Provenance: provenance,
				MaterializedContent: authenticationStaged,
			},
			{
				Kind: "browser-profile", SourceKind: "authenticated_authoring", ID: cfg.ProfileID,
				SourcePath: "private-browsertools-envelope", TargetPath: capabilityTarget,
				SHA256: hex.EncodeToString(capabilitySum[:]), SourceSHA256: strings.TrimPrefix(envelope.CapabilityReview.ProfileDigest, "sha256:"),
				Title: capability.Info.Title, OperationCount: len(capability.Actions), Actions: capability.SortedActionNames(),
				Origins: append([]string(nil), capability.Info.Origin...), Lifecycle: "active", ExpiresAt: capabilityExpiry.Format(time.RFC3339),
				LoginStateRequired: capability.Info.LoginStateRequired, Provenance: provenance, MaterializedContent: capabilityStaged,
			},
		},
	}
	session.Normalize()
	browserMetadata, ok, err := browserSourceMetadataJSON(session)
	if err != nil || !ok {
		return preparedAuthenticatedImport{}, firstNonNil(err, fmt.Errorf("capability review metadata was not produced"))
	}
	authenticationMetadata, ok, err := browserAuthenticationMetadataJSON(session)
	if err != nil || !ok {
		return preparedAuthenticatedImport{}, firstNonNil(err, fmt.Errorf("authentication review metadata was not produced"))
	}
	safeReview := authenticatedAuthoringSafeReview{
		Version: authenticatedAuthoringReviewVersion, EnvelopeSHA256: actualDigest,
		ObservedAt: envelope.ObservedAt, Goal: envelope.Goal, GoalPredicate: envelope.GoalPredicate,
		Origins: append([]string(nil), envelope.Origins...), Contexts: cloneLiveContexts(envelope.Contexts), Bounds: envelope.Bounds,
		TraceSteps: len(envelope.Trace), AuthenticationReview: envelope.AuthenticationReview,
		CapabilityReview: envelope.CapabilityReview, Diagnostics: append([]string(nil), envelope.Diagnostics...), PrivateEnvelopeKept: true,
	}
	reviewData, err := json.MarshalIndent(safeReview, "", "  ")
	if err != nil {
		return preparedAuthenticatedImport{}, err
	}
	reviewData = append(reviewData, '\n')
	prepared := preparedAuthenticatedImport{
		ExampleDir: cfg.ExampleDir, AuthenticationTarget: authenticationTarget, CapabilityTarget: capabilityTarget,
		AuthenticationSchema: authentication.Profile, CapabilitySchema: capability.Schema,
		EnvelopeDigest: actualDigest, Origins: append([]string(nil), envelope.Origins...), Contexts: len(envelope.Contexts), Goal: envelope.GoalPredicate,
		Files: []generatedFile{
			{Path: filepath.Join(cfg.ExampleDir, filepath.FromSlash(authenticationTarget)), Content: string(authenticationStaged)},
			{Path: filepath.Join(cfg.ExampleDir, filepath.FromSlash(capabilityTarget)), Content: string(capabilityStaged)},
			{Path: filepath.Join(cfg.ExampleDir, ".icot", "browser-authentication.json"), Content: authenticationMetadata},
			{Path: filepath.Join(cfg.ExampleDir, ".icot", "browser-sources.json"), Content: browserMetadata},
			{Path: filepath.Join(cfg.ExampleDir, ".icot", "authenticated-browser-authoring.json"), Content: string(reviewData)},
		},
	}
	if err := validatePreparedAuthenticatedImport(prepared); err != nil {
		return preparedAuthenticatedImport{}, err
	}
	return prepared, nil
}

func firstNonNil(err error, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

func readStablePrivateAuthorResult(path, privateRoot string) ([]byte, string, error) {
	canonical, err := canonicalProspectivePath("Browsertools result", path)
	if err != nil {
		return nil, "", err
	}
	if canonical == privateRoot || !pathWithin(privateRoot, canonical) {
		return nil, "", fmt.Errorf("Browsertools result must remain inside the private root")
	}
	pathBefore, err := os.Lstat(canonical)
	if err != nil {
		return nil, "", err
	}
	if pathBefore.Mode()&os.ModeSymlink != 0 {
		return nil, "", fmt.Errorf("Browsertools result must not be a symlink")
	}
	file, err := os.Open(canonical)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, "", err
	}
	if !os.SameFile(pathBefore, before) || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Mode().Perm()&0o077 != 0 || before.Size() <= 0 || before.Size() > liveAuthorResultMaxBytes {
		return nil, "", fmt.Errorf("Browsertools result must be a bounded private regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, liveAuthorResultMaxBytes+1))
	if err != nil {
		return nil, "", err
	}
	after, err := file.Stat()
	pathAfter, pathErr := os.Lstat(canonical)
	if err != nil || pathErr != nil || pathAfter.Mode()&os.ModeSymlink != 0 || len(data) > liveAuthorResultMaxBytes || !os.SameFile(before, after) || !os.SameFile(after, pathAfter) || before.Size() != after.Size() || before.Size() != int64(len(data)) || !before.ModTime().Equal(after.ModTime()) {
		return nil, "", fmt.Errorf("Browsertools result changed while being read")
	}
	sum := sha256.Sum256(data)
	return data, "sha256:" + hex.EncodeToString(sum[:]), nil
}

func decodeAuthenticatedAuthoringEnvelope(data []byte) (*authenticatedAuthoringEnvelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var envelope authenticatedAuthoringEnvelope
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("authenticated-authoring envelope: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("authenticated-authoring envelope contains trailing JSON")
	}
	return &envelope, nil
}

func validateAuthenticatedAuthoringEnvelope(cfg liveAuthorConfig, envelope *authenticatedAuthoringEnvelope, at time.Time) error {
	if envelope == nil || envelope.Schema != "browsertools.authenticated-authoring.v1" || !envelope.HumanConfirmed || envelope.Goal != cfg.Goal {
		return fmt.Errorf("authenticated-authoring identity, goal, or human confirmation is invalid")
	}
	observedAt, err := time.Parse(time.RFC3339, envelope.ObservedAt)
	if err != nil || at.Before(observedAt.Add(-time.Minute)) || at.Sub(observedAt) > 31*time.Minute {
		return fmt.Errorf("authenticated-authoring observation time is invalid or stale")
	}
	goalOrigin, goalPath := originAndPath(cfg.DashboardURL)
	wantGoal := liveGoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel}
	if !reflect.DeepEqual(envelope.GoalPredicate, wantGoal) || envelope.GoalProof.Origin != wantGoal.Origin || envelope.GoalProof.Path != wantGoal.Path || envelope.GoalProof.Context != wantGoal.Context || envelope.GoalProof.Role != wantGoal.Role || envelope.GoalProof.Label != wantGoal.Label || envelope.GoalProof.Matches != 1 {
		return fmt.Errorf("authenticated-authoring typed goal proof does not match the reviewed predicate")
	}
	origins, err := normalizeBrowserAuthoringOrigins(envelope.Origins)
	if err != nil || !reflect.DeepEqual(origins, envelope.Origins) {
		return fmt.Errorf("authenticated-authoring origin inventory is not canonical")
	}
	for _, required := range cfg.Origins {
		if !stringSliceContainsExact(origins, required) {
			return fmt.Errorf("authenticated-authoring origin inventory dropped a reviewed origin")
		}
	}
	if len(envelope.Contexts) > 64 || len(envelope.Trace) == 0 || len(envelope.Trace) > 512 || len(envelope.Diagnostics) > liveAuthorMaxDiagnostics {
		return fmt.Errorf("authenticated-authoring context, trace, or diagnostic bounds are invalid")
	}
	if err := validateLiveBounds(envelope.Bounds); err != nil {
		return err
	}
	if envelope.Bounds != defaultLiveAuthorBounds() {
		return fmt.Errorf("authenticated-authoring bounds do not match the authority sent to Browsertools")
	}
	if err := validateLiveContextGraph(envelope.Contexts, origins); err != nil {
		return err
	}
	for _, step := range envelope.Trace {
		if step.POSTBudget < 0 || step.POSTBudget > 32 || step.POSTObserved < 0 || step.POSTObserved > step.POSTBudget || !liveDiagnosticPattern.MatchString(step.Kind) || !liveDiagnosticPattern.MatchString(step.Phase) {
			return fmt.Errorf("authenticated-authoring trace is invalid")
		}
		contextID := firstNonEmpty(step.Context, "main")
		if contextID != "main" {
			if _, ok := envelope.Contexts[contextID]; !ok {
				return fmt.Errorf("authenticated-authoring trace references an unknown context")
			}
		}
		if step.CandidateID != "" && !liveCandidatePattern.MatchString(step.CandidateID) {
			return fmt.Errorf("authenticated-authoring trace candidate is invalid")
		}
		if step.URL != "" {
			clean, _, urlErr := normalizeBrowserAuthoringURL(step.URL)
			if urlErr != nil || clean != step.URL {
				return fmt.Errorf("authenticated-authoring trace URL is invalid")
			}
		}
		if step.OpensContext != "" {
			opened, ok := envelope.Contexts[step.OpensContext]
			if !ok || opened.Kind != "popup" || step.Kind != "click" || opened.Parent != contextID {
				return fmt.Errorf("authenticated-authoring popup binding is invalid")
			}
		}
	}
	for _, diagnostic := range envelope.Diagnostics {
		if !liveDiagnosticPattern.MatchString(diagnostic) {
			return fmt.Errorf("authenticated-authoring diagnostic is invalid")
		}
	}
	if err := validateLiveProfileReview(envelope.AuthenticationReview, "authentication", envelope.AuthenticationProfile, envelope.ObservedAt); err != nil {
		return err
	}
	if err := validateLiveProfileReview(envelope.CapabilityReview, "capability", envelope.CapabilityProfile, envelope.ObservedAt); err != nil {
		return err
	}
	var generic any
	if err := json.Unmarshal(mustJSONMarshal(envelope), &generic); err != nil {
		return err
	}
	if liveValueContainsSecret(generic) {
		return fmt.Errorf("authenticated-authoring envelope contains a secret-like literal")
	}
	return nil
}

func mustJSONMarshal(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func liveValueContainsSecret(value any) bool {
	switch typed := value.(type) {
	case string:
		return redact.String(typed) != typed
	case []any:
		for _, item := range typed {
			if liveValueContainsSecret(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range typed {
			if liveValueContainsSecret(item) {
				return true
			}
		}
	}
	return false
}

func validateLiveBounds(bounds liveBounds) error {
	if bounds.NavigationTimeoutMS <= 0 || bounds.NavigationTimeoutMS > time.Minute.Milliseconds() || bounds.TotalTimeoutMS < bounds.NavigationTimeoutMS || bounds.TotalTimeoutMS > (30*time.Minute).Milliseconds() || bounds.MaxRequests <= 0 || bounds.MaxRequests > 4096 || bounds.MaxResponseBytes <= 0 || bounds.MaxResponseBytes > 128<<20 || bounds.MaxObservations <= 0 || bounds.MaxObservations > 256 || bounds.MaxCandidates <= 0 || bounds.MaxCandidates > liveAuthorMaxCandidates {
		return fmt.Errorf("authenticated-authoring bounds are invalid")
	}
	return nil
}

func validateLiveContextGraph(contexts map[string]liveContext, origins []string) error {
	for id, context := range contexts {
		if id == "main" || !liveContextPattern.MatchString(id) || !liveContextPattern.MatchString(context.Parent) || (context.Kind != "popup" && context.Kind != "frame") || !stringSliceContainsExact(origins, context.Origin) {
			return fmt.Errorf("authenticated-authoring context is invalid")
		}
		if context.Kind == "popup" && (context.Path != "" || context.Name != "") || context.Kind == "frame" && context.Path == "" && context.Name == "" {
			return fmt.Errorf("authenticated-authoring context identity is invalid")
		}
		if context.Path != "" && (!strings.HasPrefix(context.Path, "/") || strings.ContainsAny(context.Path, "?#\\")) {
			return fmt.Errorf("authenticated-authoring frame path is invalid")
		}
	}
	for id := range contexts {
		seen := map[string]bool{}
		current := id
		for depth := 0; current != "main"; depth++ {
			if depth >= 4 || seen[current] {
				return fmt.Errorf("authenticated-authoring context graph is cyclic or too deep")
			}
			seen[current] = true
			context, ok := contexts[current]
			if !ok {
				return fmt.Errorf("authenticated-authoring context parent is missing")
			}
			current = context.Parent
		}
	}
	return nil
}

func validateLiveProfileReview(review liveProfileReview, kind string, raw json.RawMessage, observedAt string) error {
	if review.Schema != "browsertools.authenticated-profile-review.v1" || review.Kind != kind || review.AssessedAt != observedAt || !validSHA256Digest(review.ProfileDigest) || len(review.Decisions) == 0 || len(review.Decisions) > 32 {
		return fmt.Errorf("authenticated-authoring %s review is invalid", kind)
	}
	sum := sha256.Sum256(raw)
	if review.ProfileDigest != "sha256:"+hex.EncodeToString(sum[:]) {
		return fmt.Errorf("authenticated-authoring %s profile digest mismatch", kind)
	}
	seen := map[string]bool{}
	for _, decision := range review.Decisions {
		if !liveReviewDecisionPattern.MatchString(decision) || seen[decision] {
			return fmt.Errorf("authenticated-authoring %s review decision is invalid", kind)
		}
		seen[decision] = true
	}
	return nil
}

func canonicalEmbeddedProfile(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 || len(raw) > liveAuthorResultMaxBytes {
		return nil, fmt.Errorf("embedded profile is missing or oversized")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return nil, err
	}
	if !bytes.Equal(bytes.TrimSpace(raw), compact.Bytes()) {
		return nil, fmt.Errorf("embedded profile is not canonical compact JSON")
	}
	return append([]byte(nil), compact.Bytes()...), nil
}

func allowedAuthenticatedProfilePair(authentication, capability string) bool {
	return authentication == "uws.browser-authentication.1.0" && capability == "uws.browser.1.5" ||
		authentication == "uws.browser-authentication.1.1" && (capability == "uws.browser.1.5" || capability == "uws.browser.1.6")
}

func liveCapabilityExpiry(value *profile.Profile) (time.Time, error) {
	verified, err := time.Parse(time.RFC3339, value.Verification.LastVerifiedAt)
	if err != nil {
		return time.Time{}, err
	}
	return value.ExpiresAfter.AddTo(verified)
}

func mustLiveAuthenticationExpiry(value *authprofile.Profile) time.Time {
	expires, _ := authprofile.ExpiresAt(value)
	return expires
}

func liveAuthenticationFlowSlots(flow browserauthentication.Flow) []string {
	set := map[string]bool{}
	for _, step := range flow.Sequence {
		if step.TypeCredential != nil && step.TypeCredential.Slot != "" {
			set[step.TypeCredential.Slot] = true
		}
		if step.Challenge != nil && step.Challenge.Slot != "" {
			set[step.Challenge.Slot] = true
		}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func cloneLiveContexts(values map[string]liveContext) map[string]liveContext {
	result := make(map[string]liveContext, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func validatePreparedAuthenticatedImport(prepared preparedAuthenticatedImport) error {
	if prepared.ExampleDir == "" || len(prepared.Files) != 5 {
		return fmt.Errorf("prepared authenticated-authoring import is incomplete")
	}
	for _, directory := range []string{"browser-authentication", "browser-profiles"} {
		path := filepath.Join(prepared.ExampleDir, directory)
		entries, err := os.ReadDir(path)
		if err == nil && len(entries) > 0 {
			return fmt.Errorf("%s already contains browser sources; use normal iCoT to merge reviewed profiles", path)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, file := range prepared.Files {
		if _, err := os.Lstat(file.Path); err == nil {
			return fmt.Errorf("refusing to overwrite existing live-authoring target %s", file.Path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func stageAuthenticatedAuthoringImport(prepared preparedAuthenticatedImport) error {
	if err := validatePreparedAuthenticatedImport(prepared); err != nil {
		return err
	}
	return writeGeneratedFilesAtomic(prepared.Files, false)
}

func printAuthenticatedAuthoringReview(out io.Writer, prepared preparedAuthenticatedImport) {
	fmt.Fprintln(out, "Authenticated Browsertools result passed independent OpenUdon review:")
	fmt.Fprintf(out, "- private envelope digest: %s (the envelope is not packaged)\n", prepared.EnvelopeDigest)
	fmt.Fprintf(out, "- authentication: %s -> %s\n", prepared.AuthenticationSchema, prepared.AuthenticationTarget)
	fmt.Fprintf(out, "- capability: %s -> %s\n", prepared.CapabilitySchema, prepared.CapabilityTarget)
	fmt.Fprintf(out, "- approved origins: %s; child contexts: %d\n", strings.Join(prepared.Origins, ", "), prepared.Contexts)
	fmt.Fprintf(out, "- typed goal: %s%s context=%s role=%s label=%q\n", prepared.Goal.Origin, prepared.Goal.Path, prepared.Goal.Context, prepared.Goal.Role, prepared.Goal.Label)
}
