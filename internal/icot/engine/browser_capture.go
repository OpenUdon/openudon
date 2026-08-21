package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/openudon/internal/credentialpolicy"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/uws/schemas"
)

const (
	browserCaptureReviewVersion       = "openudon.authenticated-browser-authoring-review.v3"
	legacyBrowserCaptureReviewVersion = "openudon.authenticated-browser-authoring-review.v2"
	maxBrowserCaptureReviewBytes      = 2 << 20
)

var browserCaptureProfileID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// BrowserCaptureStage is the independently validated, value-free output of a
// completed Browsertools author session. The engine revalidates the canonical
// paths and public schemas before atomically adopting it.
type BrowserCaptureStage struct {
	ProfileID      string `json:"profile_id"`
	Authentication []byte `json:"-"`
	Capability     []byte `json:"-"`
	SafeReview     []byte `json:"-"`
}

type browserCaptureSafeReview struct {
	Version              string                          `json:"version"`
	ProfileID            string                          `json:"profile_id,omitempty"`
	AuthenticationTarget string                          `json:"authentication_target,omitempty"`
	CapabilityTarget     string                          `json:"capability_target,omitempty"`
	EnvelopeSHA256       string                          `json:"envelope_sha256"`
	ObservedAt           string                          `json:"observed_at"`
	Goal                 string                          `json:"goal"`
	GoalPredicate        authorresult.GoalPredicate      `json:"goal_predicate"`
	Origins              []string                        `json:"origins"`
	Contexts             map[string]authorresult.Context `json:"contexts"`
	Bounds               authorresult.Bounds             `json:"bounds"`
	TraceSteps           int                             `json:"trace_steps"`
	OutputSelections     []authorresult.OutputSelection  `json:"output_selections"`
	AuthenticationReview authorresult.Review             `json:"authentication_review"`
	CapabilityReview     authorresult.Review             `json:"capability_review"`
	Diagnostics          []string                        `json:"diagnostics"`
	PrivateEnvelopeKept  bool                            `json:"private_envelope_kept_outside_package"`
}

type browserCaptureReviewCollection struct {
	Version  string                     `json:"version"`
	Captures []browserCaptureSafeReview `json:"captures"`
}

// StageBrowserCapture atomically writes a new profile pair and the v3 safe
// review collection, then refreshes discovery in the same engine mutation.
func (e *Engine) StageBrowserCapture(ctx context.Context, stage BrowserCaptureStage) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	if strings.TrimSpace(e.config.PrivateRoot) == "" {
		return Snapshot{}, rejected(errors.New("browser capture staging requires an engine-configured private root"))
	}
	workspaceAtStart, err := e.observeMutationWorkspaceLocked(ctx)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	stage.ProfileID = strings.ToLower(strings.TrimSpace(stage.ProfileID))
	if !browserCaptureProfileID.MatchString(stage.ProfileID) {
		return Snapshot{}, rejected(errors.New("browser capture profile ID is invalid"))
	}
	if len(stage.Authentication) == 0 || len(stage.Capability) == 0 || len(stage.SafeReview) == 0 {
		return Snapshot{}, rejected(errors.New("browser capture stage is incomplete"))
	}
	if len(stage.Authentication) > maxBrowserCaptureReviewBytes || len(stage.Capability) > maxBrowserCaptureReviewBytes || len(stage.SafeReview) > maxBrowserCaptureReviewBytes {
		return Snapshot{}, rejected(errors.New("browser capture stage exceeds the supported size bound"))
	}
	if credentialpolicy.ContainsLikelyValue(stage.Authentication) || credentialpolicy.ContainsLikelyValue(stage.Capability) || credentialpolicy.ContainsLikelyValue(stage.SafeReview) {
		return Snapshot{}, rejected(errors.New("browser capture stage contains secret-like literal content"))
	}
	if err := schemas.ValidateBrowserAuthenticationProfile(stage.Authentication); err != nil {
		return Snapshot{}, rejected(fmt.Errorf("browser authentication profile: %w", err))
	}
	if err := schemas.ValidateBrowserSourceProfile(stage.Capability); err != nil {
		return Snapshot{}, rejected(fmt.Errorf("browser capability profile: %w", err))
	}
	review, err := decodeBrowserCaptureReviewCollection(stage.SafeReview)
	if err != nil {
		return Snapshot{}, rejected(fmt.Errorf("browser capture safe review collection is invalid: %w", err))
	}
	if err := validateBrowserCaptureReviewCollection(review); err != nil {
		return Snapshot{}, rejected(fmt.Errorf("browser capture safe review collection is invalid: %w", err))
	}
	authenticationRelative := filepath.ToSlash(filepath.Join("browser-authentication", stage.ProfileID+"-auth.json"))
	capabilityRelative := filepath.ToSlash(filepath.Join("browser-profiles", stage.ProfileID+".json"))
	found := false
	for _, capture := range review.Captures {
		if capture.ProfileID == stage.ProfileID && capture.AuthenticationTarget == authenticationRelative && capture.CapabilityTarget == capabilityRelative {
			found = true
		}
	}
	if !found {
		return Snapshot{}, rejected(errors.New("browser capture safe review does not bind the staged profile"))
	}
	authenticationPath, capabilityPath, reviewPath, err := browserCaptureTargets(e.workspaceRoot, stage.ProfileID)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	if err := requireBrowserCaptureTargetsAbsent(authenticationPath, capabilityPath); err != nil {
		return Snapshot{}, err
	}
	currentReview, err := readCurrentBrowserCaptureReview(reviewPath)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	if err := validateBrowserCaptureReviewAppend(currentReview, review, stage.ProfileID); err != nil {
		return Snapshot{}, rejected(err)
	}
	paths := uniqueSortedPaths(append(append([]string(nil), e.watchedPaths...), authenticationPath, capabilityPath, reviewPath))
	accepted := acceptedFingerprint(paths, e.workspaceBaseline, workspaceAtStart)
	reviewFingerprint := accepted.entries[reviewPath]
	reviewExists := reviewFingerprint.Type == "regular"
	expectedReviewSHA256 := ""
	if reviewExists {
		expectedReviewSHA256 = "sha256:" + reviewFingerprint.SHA256
	}
	prepared := artifactwriter.Prepared{ExampleRoot: e.workspaceRoot, Files: []artifactwriter.GeneratedFile{
		{Path: authenticationPath, Content: ensureTrailingNewline(stage.Authentication), Action: "stage_browser_authentication", Reason: "reviewed Browsertools author session"},
		{Path: capabilityPath, Content: ensureTrailingNewline(stage.Capability), Action: "stage_browser_capability", Reason: "reviewed Browsertools author session"},
		{Path: reviewPath, Content: ensureTrailingNewline(stage.SafeReview), AllowOverwrite: reviewExists, ExpectedCurrentSHA256: expectedReviewSHA256, Action: "update_browser_capture_review", Reason: "safe v3 capture collection"},
	}}
	if _, err := commitPrepared(prepared, false, func() error {
		if err := e.compareAndLatchWorkspaceLocked(paths, accepted); err != nil {
			return err
		}
		recheckedAuthentication, recheckedCapability, recheckedReview, err := browserCaptureTargets(e.workspaceRoot, stage.ProfileID)
		if err != nil || recheckedAuthentication != authenticationPath || recheckedCapability != capabilityPath || recheckedReview != reviewPath {
			return conflict("workspace_changed", errors.New("browser capture targets changed before replacement"))
		}
		if err := requireBrowserCaptureTargetsAbsent(authenticationPath, capabilityPath); err != nil {
			return err
		}
		currentReview, err := readCurrentBrowserCaptureReview(reviewPath)
		if err != nil {
			return rejected(err)
		}
		if err := validateBrowserCaptureReviewAppend(currentReview, review, stage.ProfileID); err != nil {
			return rejected(err)
		}
		return nil
	}); err != nil {
		return Snapshot{}, classifyCommit(err)
	}
	return e.refreshAfterAcquisitionCommitLocked(ctx)
}

func browserCaptureTargets(root, profileID string) (string, string, string, error) {
	authentication, err := artifactwriter.SafeExampleTarget(root, filepath.ToSlash(filepath.Join("browser-authentication", profileID+"-auth.json")))
	if err != nil {
		return "", "", "", err
	}
	capability, err := artifactwriter.SafeExampleTarget(root, filepath.ToSlash(filepath.Join("browser-profiles", profileID+".json")))
	if err != nil {
		return "", "", "", err
	}
	review, err := artifactwriter.SafeExampleTarget(root, ".icot/authenticated-browser-authoring.json")
	if err != nil {
		return "", "", "", err
	}
	return authentication, capability, review, nil
}

func requireBrowserCaptureTargetsAbsent(paths ...string) error {
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			return rejected(fmt.Errorf("browser capture target %s already exists", filepath.Base(path)))
		} else if !errors.Is(err, os.ErrNotExist) {
			return operational(err)
		}
	}
	return nil
}

func decodeBrowserCaptureReviewCollection(data []byte) (browserCaptureReviewCollection, error) {
	var collection browserCaptureReviewCollection
	if len(data) == 0 || len(data) > maxBrowserCaptureReviewBytes {
		return collection, errors.New("review bytes are empty or oversized")
	}
	if err := decodeStrictBrowserCaptureReview(data, &collection); err != nil {
		return collection, err
	}
	if collection.Version != browserCaptureReviewVersion || len(collection.Captures) == 0 || len(collection.Captures) > 128 {
		return collection, errors.New("review version or capture count is invalid")
	}
	return collection, nil
}

// ValidateBrowserCaptureReviewCollection applies the same semantic collection
// invariants used by engine staging. Terminal live authoring calls this before
// it appends to an existing v3 review file.
func ValidateBrowserCaptureReviewCollection(data []byte) error {
	collection, err := decodeBrowserCaptureReviewCollection(data)
	if err != nil {
		return err
	}
	return validateBrowserCaptureReviewCollection(collection)
}

func validateBrowserCaptureReviewCollection(review browserCaptureReviewCollection) error {
	seenIDs, seenAuthentication, seenCapabilities, seenEnvelopes := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for index, capture := range review.Captures {
		if capture.Version != browserCaptureReviewVersion || !browserCaptureProfileID.MatchString(capture.ProfileID) || seenIDs[capture.ProfileID] {
			return errors.New("browser capture safe review contains an invalid or duplicate profile")
		}
		if index > 0 && review.Captures[index-1].ProfileID >= capture.ProfileID {
			return errors.New("browser capture safe review profiles are not deterministically ordered")
		}
		seenIDs[capture.ProfileID] = true
		legacy := strings.HasPrefix(capture.ProfileID, "legacy-")
		if err := validateBrowserCaptureSafeEvidence(capture); err != nil {
			return err
		}
		if legacy {
			if capture.AuthenticationTarget != "" || capture.CapabilityTarget != "" {
				return errors.New("legacy browser capture targets must be empty")
			}
			digest := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(capture.EnvelopeSHA256)), "sha256:")
			if capture.ProfileID != "legacy-"+digest[:12] {
				return errors.New("legacy browser capture profile ID is not derived from its envelope digest")
			}
		} else {
			expectedAuthentication := filepath.ToSlash(filepath.Join("browser-authentication", capture.ProfileID+"-auth.json"))
			expectedCapability := filepath.ToSlash(filepath.Join("browser-profiles", capture.ProfileID+".json"))
			if capture.AuthenticationTarget != expectedAuthentication || capture.CapabilityTarget != expectedCapability {
				return errors.New("browser capture safe review profile binding is invalid")
			}
		}
		if capture.AuthenticationTarget != "" {
			if seenAuthentication[capture.AuthenticationTarget] {
				return errors.New("browser capture safe review contains a duplicate authentication target")
			}
			seenAuthentication[capture.AuthenticationTarget] = true
		}
		if capture.CapabilityTarget != "" {
			if seenCapabilities[capture.CapabilityTarget] {
				return errors.New("browser capture safe review contains a duplicate capability target")
			}
			seenCapabilities[capture.CapabilityTarget] = true
		}
		if !validBrowserCaptureDigest(capture.EnvelopeSHA256) || seenEnvelopes[capture.EnvelopeSHA256] {
			return errors.New("browser capture safe review contains an invalid or duplicate envelope digest")
		}
		seenEnvelopes[capture.EnvelopeSHA256] = true
	}
	return nil
}

func validateBrowserCaptureSafeEvidence(capture browserCaptureSafeReview) error {
	if strings.TrimSpace(capture.Goal) == "" || !capture.PrivateEnvelopeKept ||
		strings.TrimSpace(capture.GoalPredicate.Origin) == "" || strings.TrimSpace(capture.GoalPredicate.Path) == "" || strings.TrimSpace(capture.GoalPredicate.Role) == "" ||
		len(capture.Origins) == 0 || len(capture.Origins) > 64 || capture.Contexts == nil || len(capture.Contexts) > 64 ||
		capture.TraceSteps <= 0 || capture.TraceSteps > 512 || len(capture.OutputSelections) > capture.Bounds.MaxOutputs ||
		capture.Bounds.NavigationTimeoutMS <= 0 || capture.Bounds.TotalTimeoutMS <= 0 || capture.Bounds.MaxRequests <= 0 ||
		capture.Bounds.MaxResponseBytes <= 0 || capture.Bounds.MaxObservations <= 0 || capture.Bounds.MaxCandidates <= 0 || capture.Bounds.MaxOutputs <= 0 {
		return errors.New("browser capture safe review evidence is incomplete")
	}
	if _, err := time.Parse(time.RFC3339, capture.ObservedAt); err != nil || !validBrowserCaptureDigest(capture.EnvelopeSHA256) {
		return errors.New("browser capture safe review evidence is invalid")
	}
	for _, origin := range capture.Origins {
		if strings.TrimSpace(origin) == "" {
			return errors.New("browser capture safe review origin evidence is invalid")
		}
	}
	for _, review := range []struct {
		value authorresult.Review
		kind  string
	}{{capture.AuthenticationReview, "authentication"}, {capture.CapabilityReview, "capability"}} {
		if review.value.Schema != "browsertools.authenticated-profile-review.v1" || review.value.Kind != review.kind ||
			review.value.AssessedAt != capture.ObservedAt || !validBrowserCaptureDigest(review.value.ProfileDigest) ||
			len(review.value.Decisions) == 0 || len(review.value.Decisions) > 32 {
			return errors.New("browser capture safe profile review evidence is invalid")
		}
		for _, decision := range review.value.Decisions {
			if strings.TrimSpace(decision) == "" {
				return errors.New("browser capture safe profile review decision is invalid")
			}
		}
	}
	return nil
}

func readCurrentBrowserCaptureReview(path string) (browserCaptureReviewCollection, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return browserCaptureReviewCollection{Version: browserCaptureReviewVersion}, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > maxBrowserCaptureReviewBytes {
		return browserCaptureReviewCollection{}, errors.New("existing browser capture review is unavailable or unsafe")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return browserCaptureReviewCollection{}, errors.New("existing browser capture review is unreadable")
	}
	var header struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return browserCaptureReviewCollection{}, errors.New("existing browser capture review is malformed")
	}
	if header.Version == browserCaptureReviewVersion {
		return decodeBrowserCaptureReviewCollection(data)
	}
	if header.Version != legacyBrowserCaptureReviewVersion {
		return browserCaptureReviewCollection{}, errors.New("existing browser capture review version is unsupported")
	}
	var legacy browserCaptureSafeReview
	if err := decodeStrictBrowserCaptureReview(data, &legacy); err != nil {
		return browserCaptureReviewCollection{}, errors.New("existing legacy browser capture review is invalid")
	}
	digest := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(legacy.EnvelopeSHA256)), "sha256:")
	if !validBrowserCaptureDigest(digest) {
		return browserCaptureReviewCollection{}, errors.New("existing legacy browser capture review digest is invalid")
	}
	legacy.Version = browserCaptureReviewVersion
	legacy.ProfileID = "legacy-" + digest[:12]
	legacy.EnvelopeSHA256 = "sha256:" + digest
	collection := browserCaptureReviewCollection{Version: browserCaptureReviewVersion, Captures: []browserCaptureSafeReview{legacy}}
	if err := validateBrowserCaptureReviewCollection(collection); err != nil {
		return browserCaptureReviewCollection{}, fmt.Errorf("existing legacy browser capture review is invalid: %w", err)
	}
	return collection, nil
}

func validateBrowserCaptureReviewAppend(current, next browserCaptureReviewCollection, profileID string) error {
	if len(next.Captures) != len(current.Captures)+1 {
		return errors.New("browser capture review must append exactly one profile")
	}
	currentByID := make(map[string]browserCaptureSafeReview, len(current.Captures))
	for _, capture := range current.Captures {
		currentByID[capture.ProfileID] = capture
	}
	added := 0
	for _, capture := range next.Captures {
		if prior, ok := currentByID[capture.ProfileID]; ok {
			if !reflect.DeepEqual(prior, capture) {
				return errors.New("browser capture review would alter existing evidence")
			}
			continue
		}
		if capture.ProfileID != profileID {
			return errors.New("browser capture review appended an unexpected profile")
		}
		added++
	}
	if added != 1 {
		return errors.New("browser capture review did not append the staged profile")
	}
	return nil
}

func decodeStrictBrowserCaptureReview(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("review contains trailing JSON data")
	}
	return nil
}

func validBrowserCaptureDigest(value string) bool {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "sha256:")
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func ensureTrailingNewline(data []byte) string {
	return strings.TrimSuffix(string(data), "\n") + "\n"
}
