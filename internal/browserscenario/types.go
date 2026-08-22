// Package browserscenario owns deterministic loopback and optional public
// browser-scenario qualification without widening normal iCoT authoring.
package browserscenario

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/evidence/redact"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	ManifestVersion        = "openudon.browser-scenario.v1"
	JourneyManifestVersion = "openudon.browser-journey.v1"
	LockVersion            = "openudon.browser-scenario-lock.v2"
	SuiteLoopback          = "loopback"
	SuiteJourney           = "journey"
	SuitePublic            = "public"
)

var (
	idPattern  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	keyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
)

//go:embed manifests/*.json compatibility-lock.json
var contracts embed.FS

type Manifest struct {
	Version        string          `json:"version"`
	ID             string          `json:"id"`
	Suite          string          `json:"suite"`
	Authentication *Authentication `json:"authentication,omitempty"`
	Goal           *Goal           `json:"goal,omitempty"`
	Outputs        []Output        `json:"outputs"`
	Fault          string          `json:"fault,omitempty"`
	ReplayVariants []string        `json:"replayVariants"`
	Expected       Expected        `json:"expected"`
	Target         *PublicTarget   `json:"target,omitempty"`
	Probes         []Probe         `json:"probes"`
	Quarantine     *Quarantine     `json:"quarantine,omitempty"`
	Journey        *Journey        `json:"journey,omitempty"`
}

type Journey struct {
	Kind string `json:"kind"`
}

type Authentication struct {
	Credentials   []string `json:"credentials"`
	ChallengeKind string   `json:"challengeKind,omitempty"`
	ContextMode   string   `json:"contextMode"`
}

type Goal struct {
	Context string `json:"context"`
	Role    string `json:"role"`
	Name    string `json:"name"`
}

type Output struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Role        string `json:"role"`
	Name        string `json:"name,omitempty"`
	LocatorMode string `json:"locatorMode"`
}

type Expected struct {
	Authoring      string `json:"authoring"`
	Replay         string `json:"replay"`
	FailureCode    string `json:"failureCode,omitempty"`
	BrowserProfile string `json:"browserProfile,omitempty"`
	UWSVersion     string `json:"uwsVersion,omitempty"`
}

type PublicTarget struct {
	URL     string   `json:"url"`
	Origins []string `json:"origins"`
}

type Probe struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Name       string `json:"name,omitempty"`
	MinMatches int    `json:"minMatches"`
	MaxMatches int    `json:"maxMatches"`
}

type Quarantine struct {
	Since  string `json:"since"`
	Until  string `json:"until"`
	Reason string `json:"reason"`
}

type CompatibilityLock struct {
	Version     string           `json:"version"`
	Components  []LockedRevision `json:"components"`
	GoVersion   string           `json:"goVersion"`
	NodeVersion string           `json:"nodeVersion"`
	Playwright  string           `json:"playwright"`
	Chromium    string           `json:"chromium"`
}

type LockedRevision struct {
	Name    string `json:"name"`
	Commit  string `json:"commit,omitempty"`
	Module  string `json:"module,omitempty"`
	Version string `json:"version,omitempty"`
}

func LoadManifests(now time.Time) ([]Manifest, error) {
	entries, err := fs.Glob(contracts, "manifests/*.json")
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	manifests := make([]Manifest, 0, len(entries))
	seen := map[string]bool{}
	for _, name := range entries {
		data, err := contracts.ReadFile(name)
		if err != nil {
			return nil, err
		}
		var manifest Manifest
		if err := decodeStrict(data, &manifest); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if err := ValidateManifest(manifest, now); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if seen[manifest.ID] {
			return nil, fmt.Errorf("duplicate browser scenario %q", manifest.ID)
		}
		seen[manifest.ID] = true
		manifests = append(manifests, manifest)
	}
	if len(manifests) == 0 {
		return nil, fmt.Errorf("browser scenario corpus is empty")
	}
	return manifests, nil
}

func LoadCompatibilityLock() (CompatibilityLock, error) {
	data, err := contracts.ReadFile("compatibility-lock.json")
	if err != nil {
		return CompatibilityLock{}, err
	}
	var lock CompatibilityLock
	if err := decodeStrict(data, &lock); err != nil {
		return CompatibilityLock{}, err
	}
	if err := ValidateCompatibilityLock(lock); err != nil {
		return CompatibilityLock{}, err
	}
	return lock, nil
}

func SelectManifests(all []Manifest, suite string, ids []string) ([]Manifest, error) {
	if suite != SuiteLoopback && suite != SuiteJourney && suite != SuitePublic {
		return nil, fmt.Errorf("browser scenario suite must be loopback, journey, or public")
	}
	requested := make(map[string]bool, len(ids))
	filtered := len(ids) > 0
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if !idPattern.MatchString(id) || requested[id] {
			return nil, fmt.Errorf("browser scenario filter %q is invalid or duplicated", id)
		}
		requested[id] = true
	}
	selected := []Manifest{}
	for _, manifest := range all {
		if manifest.Suite == suite && (!filtered || requested[manifest.ID]) {
			selected = append(selected, manifest)
			delete(requested, manifest.ID)
		}
	}
	if len(requested) > 0 {
		unknown := make([]string, 0, len(requested))
		for id := range requested {
			unknown = append(unknown, id)
		}
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown %s browser scenarios: %s", suite, strings.Join(unknown, ", "))
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no %s browser scenarios selected", suite)
	}
	return selected, nil
}

func ValidateManifest(manifest Manifest, now time.Time) error {
	wantVersion := ManifestVersion
	if manifest.Suite == SuiteJourney {
		wantVersion = JourneyManifestVersion
	}
	if manifest.Version != wantVersion || !idPattern.MatchString(manifest.ID) {
		return fmt.Errorf("browser scenario identity is invalid")
	}
	if manifest.Suite != SuiteLoopback && manifest.Suite != SuiteJourney && manifest.Suite != SuitePublic {
		return fmt.Errorf("browser scenario suite is invalid")
	}
	if manifest.Suite == SuiteLoopback {
		return validateLoopbackManifest(manifest)
	}
	if manifest.Suite == SuiteJourney {
		return validateJourneyManifest(manifest)
	}
	return validatePublicManifest(manifest, now)
}

func validateLoopbackManifest(manifest Manifest) error {
	if manifest.Authentication == nil || manifest.Goal == nil || manifest.Target != nil || len(manifest.Probes) != 0 || manifest.Quarantine != nil || manifest.Journey != nil {
		return fmt.Errorf("loopback scenario boundary is invalid")
	}
	credentials := strings.Join(manifest.Authentication.Credentials, ",")
	if credentials != "identifier,password" || !allowedContextModes[manifest.Authentication.ContextMode] || !allowedChallengeKinds[manifest.Authentication.ChallengeKind] {
		return fmt.Errorf("loopback authentication shape is invalid")
	}
	if manifest.Goal.Context == "" || manifest.Goal.Role == "" || manifest.Goal.Name == "" {
		return fmt.Errorf("loopback goal is incomplete")
	}
	if len(manifest.Outputs) > 17 {
		return fmt.Errorf("loopback output request bound exceeded")
	}
	seenKeys := map[string]bool{}
	for _, output := range manifest.Outputs {
		if !keyPattern.MatchString(output.Key) || output.Key == "goal_present" || redact.SensitiveKey(strings.ToLower(output.Key)) || seenKeys[output.Key] ||
			!allowedOutputTypes[output.Type] || !allowedLocatorModes[output.LocatorMode] || output.Role == "" ||
			(output.LocatorMode == "exact_name" && output.Name == "") || (output.LocatorMode == "unique_role" && output.Name != "") {
			return fmt.Errorf("loopback output declaration is invalid")
		}
		seenKeys[output.Key] = true
	}
	if !allowedFaults[manifest.Fault] || !allowedOutcome[manifest.Expected.Authoring] || !allowedOutcome[manifest.Expected.Replay] || !allowedFailureCodes[manifest.Expected.FailureCode] {
		return fmt.Errorf("loopback expected result is invalid")
	}
	if manifest.Expected.Authoring == "pass" && (manifest.Expected.BrowserProfile == "" || manifest.Expected.UWSVersion == "") {
		return fmt.Errorf("successful loopback scenario requires profile expectations")
	}
	for _, variant := range manifest.ReplayVariants {
		if !allowedReplayVariants[variant] {
			return fmt.Errorf("loopback replay variant %q is invalid", variant)
		}
	}
	return nil
}

func validateJourneyManifest(manifest Manifest) error {
	if manifest.Authentication != nil || manifest.Goal != nil || len(manifest.Outputs) != 0 || manifest.Fault != "" || manifest.Target != nil || len(manifest.Probes) != 0 || manifest.Quarantine != nil || manifest.Journey == nil ||
		!allowedJourneyKinds[manifest.Journey.Kind] || manifest.Expected.Authoring != "pass" || !allowedOutcome[manifest.Expected.Replay] || !allowedJourneyFailureCodes[manifest.Expected.FailureCode] ||
		manifest.Expected.BrowserProfile != "uws.browser.1.5" || manifest.Expected.UWSVersion != "1.8.0" {
		return fmt.Errorf("journey scenario boundary is invalid")
	}
	if manifest.ID != strings.ReplaceAll(manifest.Journey.Kind, "_", "-") {
		return fmt.Errorf("journey scenario identity does not match its kind")
	}
	for _, variant := range manifest.ReplayVariants {
		if !allowedJourneyReplayVariants[variant] {
			return fmt.Errorf("journey replay variant %q is invalid", variant)
		}
	}
	return validateJourneyExpectedContract(manifest)
}

func validateJourneyExpectedContract(manifest Manifest) error {
	wantReplay, wantFailure := "pass", ""
	var wantVariants []string
	switch manifest.Journey.Kind {
	case "record_update_unapproved":
		wantReplay, wantFailure = "rejected", "approval_required"
	case "record_update_ambiguous":
		wantReplay, wantFailure = "rejected", "ambiguous_locator"
	case "parameter_contract_rejected":
		wantReplay, wantFailure = "rejected", "invalid_parameters"
		wantVariants = []string{"missing_required", "additional_parameter", "wrong_type", "origin_escape"}
	}
	if manifest.Expected.Replay != wantReplay || manifest.Expected.FailureCode != wantFailure || !equalStrings(manifest.ReplayVariants, wantVariants) {
		return fmt.Errorf("journey expected outcome does not match its kind")
	}
	return nil
}

func validatePublicManifest(manifest Manifest, now time.Time) error {
	if manifest.Authentication != nil || manifest.Goal != nil || len(manifest.Outputs) != 0 || manifest.Fault != "" || len(manifest.ReplayVariants) != 0 ||
		manifest.Expected.Authoring != "pass" || manifest.Expected.Replay != "pass" || manifest.Expected.FailureCode != "" || manifest.Expected.BrowserProfile != "uws.browser.1.5" || manifest.Expected.UWSVersion != "1.7.0" || manifest.Target == nil || manifest.Journey != nil {
		return fmt.Errorf("public scenario boundary is invalid")
	}
	parsed, err := url.Parse(manifest.Target.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || net.ParseIP(parsed.Hostname()) != nil || !cleanPath(parsed.EscapedPath()) {
		return fmt.Errorf("public scenario URL is invalid")
	}
	if len(manifest.Target.Origins) == 0 || len(manifest.Target.Origins) > 8 || len(manifest.Probes) == 0 || len(manifest.Probes) > 8 {
		return fmt.Errorf("public scenario bounds are invalid")
	}
	origins := map[string]bool{}
	previousOrigin := ""
	for _, origin := range manifest.Target.Origins {
		canonical, err := exactHTTPSOrigin(origin)
		if err != nil || origins[canonical] || previousOrigin >= canonical {
			return fmt.Errorf("public scenario origin inventory is invalid")
		}
		origins[canonical] = true
		previousOrigin = canonical
	}
	targetOrigin, _ := exactHTTPSOrigin(parsed.Scheme + "://" + parsed.Host)
	if !origins[targetOrigin] {
		return fmt.Errorf("public target origin is not allowlisted")
	}
	seen := map[string]bool{}
	for _, probe := range manifest.Probes {
		if !idPattern.MatchString(probe.ID) || seen[probe.ID] || !allowedPublicProbeRoles[probe.Role] || probe.MinMatches != 1 || probe.MaxMatches != 1 {
			return fmt.Errorf("public scenario probe is invalid")
		}
		if probe.Name != "" {
			reduced := authorsession.ReduceAccessibilityLabel(probe.Name)
			if len(probe.Name) > 256 || reduced.Reason != authorsession.LabelReasonUnchanged || reduced.Value != probe.Name || redact.String(probe.Name) != probe.Name {
				return fmt.Errorf("public scenario probe label is invalid")
			}
		}
		seen[probe.ID] = true
	}
	if manifest.Quarantine != nil {
		since, err1 := time.Parse("2006-01-02", manifest.Quarantine.Since)
		until, err2 := time.Parse("2006-01-02", manifest.Quarantine.Until)
		if err1 != nil || err2 != nil || until.Before(since) || until.Sub(since) > 14*24*time.Hour || !allowedQuarantineReasons[manifest.Quarantine.Reason] {
			return fmt.Errorf("public scenario quarantine is invalid")
		}
		if now.UTC().After(until.Add(24*time.Hour - time.Nanosecond)) {
			return fmt.Errorf("public scenario quarantine expired")
		}
	}
	return nil
}

func ValidateCompatibilityLock(lock CompatibilityLock) error {
	if lock.Version != LockVersion || lock.GoVersion == "" || lock.NodeVersion == "" || lock.Playwright == "" || lock.Chromium == "" || len(lock.Components) != 4 {
		return fmt.Errorf("browser scenario compatibility lock is incomplete")
	}
	want := []string{"browserdriver", "browsertools", "udon", "uws"}
	for index, component := range lock.Components {
		if component.Name != want[index] {
			return fmt.Errorf("browser scenario compatibility lock order is invalid")
		}
		if component.Name == "browsertools" || component.Name == "uws" {
			if component.Module == "" || component.Version == "" || component.Commit == "" {
				return fmt.Errorf("browser scenario module lock is incomplete")
			}
		} else if component.Commit == "" || component.Module != "" || component.Version != "" {
			return fmt.Errorf("browser scenario repository lock is incomplete")
		}
		if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(component.Commit) {
			return fmt.Errorf("browser scenario component commit is invalid")
		}
	}
	return nil
}

// RepositoryState is the clean immutable state required from one locked
// sibling repository before browser release evidence can run.
type RepositoryState struct {
	Commit string
	Dirty  bool
}

// ValidateRepositoryStates applies the shared compatibility lock used by both
// browser-scenario and browser-integration evidence.
func ValidateRepositoryStates(lock CompatibilityLock, states map[string]RepositoryState) error {
	for _, component := range lock.Components {
		state, ok := states[component.Name]
		if !ok || state.Commit != component.Commit {
			return fmt.Errorf("%s revision does not match the browser-scenario compatibility lock", component.Name)
		}
		if state.Dirty {
			return fmt.Errorf("%s worktree is dirty; browser release evidence requires locked clean siblings", component.Name)
		}
	}
	return nil
}

// ValidateGoModulePins verifies the dependency edges that compose the locked
// browser stack. OpenUdon must use the locked Browsertools and UWS modules, and
// Browsertools must itself use that same locked UWS revision.
func ValidateGoModulePins(openUdonRoot, browsertoolsRoot string, lock CompatibilityLock) error {
	locked := make(map[string]LockedRevision, len(lock.Components))
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	checks := []struct {
		root, owner, dependency string
	}{
		{openUdonRoot, "openudon", "browsertools"},
		{openUdonRoot, "openudon", "uws"},
		{browsertoolsRoot, "browsertools", "uws"},
	}
	for _, check := range checks {
		dependency, ok := locked[check.dependency]
		if !ok || !goModRequires(check.root, dependency.Module, dependency.Version) {
			return fmt.Errorf("%s %s dependency does not match the browser-scenario compatibility lock", check.owner, dependency.Module)
		}
	}
	return nil
}

func decodeStrict(data []byte, target any) error {
	return evidencefile.DecodeStrict(data, target)
}

func exactHTTPSOrigin(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("origin must be exact HTTPS")
	}
	host := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port == "443" {
		port = ""
	}
	if port != "" {
		host = net.JoinHostPort(host, port)
	}
	return "https://" + host, nil
}

func cleanPath(value string) bool {
	if value == "" {
		value = "/"
	}
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "?#\\") || strings.Contains(strings.TrimPrefix(value, "/"), "//") {
		return false
	}
	for _, part := range strings.Split(strings.TrimPrefix(value, "/"), "/") {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "." || decoded == ".." || strings.Contains(decoded, "/") {
			return false
		}
	}
	return true
}

var allowedContextModes = map[string]bool{"main": true, "popup": true, "frame": true}
var allowedChallengeKinds = map[string]bool{"": true, "totp": true, "sms_otp": true, "email_otp": true, "voice_otp": true, "push": true, "push_number_match": true, "passkey": true, "security_key": true}
var allowedOutputTypes = map[string]bool{"string": true, "integer": true, "number": true, "boolean": true, "presence": true}
var allowedLocatorModes = map[string]bool{"exact_name": true, "unique_role": true}

var allowedPublicProbeRoles = map[string]bool{
	"button": true, "link": true, "textbox": true, "checkbox": true, "radio": true,
	"dialog": true, "status": true, "alert": true, "heading": true, "img": true,
	"list": true, "listitem": true, "combobox": true, "option": true, "menu": true,
	"menuitem": true, "tab": true, "tabpanel": true, "table": true, "row": true,
	"cell": true, "region": true, "navigation": true, "article": true, "form": true,
	"search": true, "switch": true, "group": true,
}
var allowedOutcome = map[string]bool{"pass": true, "rejected": true}
var allowedFaults = map[string]bool{"": true, "outputs_17": true, "stale_candidate": true, "ambiguous_unique_role": true, "context_substitution": true, "invalid_scalars": true, "secret_output": true, "origin_escape": true, "path_injection": true, "fabricated_trace": true}
var allowedFailureCodes = map[string]bool{"": true, "output_bound": true, "stale_candidate": true, "ambiguous_output": true, "invalid_context": true, "invalid_response": true, "secret_output": true, "origin_rejected": true, "path_disclosure": true, "fabricated_trace": true}
var allowedReplayVariants = map[string]bool{"integer_leading_zero": true, "integer_plus": true, "integer_comma": true, "integer_unsafe": true, "number_nan": true, "number_infinity": true, "number_comma": true, "boolean_uppercase": true, "boolean_numeric": true, "empty": true}
var allowedQuarantineReasons = map[string]bool{"target_unavailable": true, "upstream_markup_drift": true, "origin_inventory_drift": true}
var allowedJourneyKinds = map[string]bool{
	"catalog_search_filter": true, "catalog_pagination": true, "order_structured_read": true,
	"record_update_approved": true, "record_update_unapproved": true, "record_update_ambiguous": true,
	"parameter_contract_rejected": true, "session_lifecycle": true,
}
var allowedJourneyFailureCodes = map[string]bool{"": true, "approval_required": true, "ambiguous_locator": true, "invalid_parameters": true}
var allowedJourneyReplayVariants = map[string]bool{"missing_required": true, "additional_parameter": true, "wrong_type": true, "origin_escape": true}
