// Package browserverify validates value-free Browsertools verification reports
// without importing Browsertools' live browser acquisition implementation.
package browserverify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/profile"
)

const (
	LiveCheckVersion     = "browsertools.live-check.v1"
	PortabilityVersion   = "browsertools.portability-check.v1"
	MaxReportBytes       = 1 << 20
	MaxReports           = 128
	MaxReportsPerProfile = 16
	MaxReviewBytes       = 64 << 20
	maxActions           = 64
	maxChecks            = 512
	maxMatchCount        = 1_000_000
)

const (
	probeLocator        = "locator"
	probeNavigationWait = "navigation_wait"
	probeOutput         = "output"
)

// Check is one declared-path, value-free observation fact.
type Check struct {
	Kind         string             `json:"kind" yaml:"kind"`
	Path         string             `json:"path" yaml:"path"`
	OK           bool               `json:"ok" yaml:"ok"`
	Matches      int                `json:"matches,omitempty" yaml:"matches,omitempty"`
	ExpectedType profile.OutputType `json:"expectedType,omitempty" yaml:"expectedType,omitempty"`
	ObservedType profile.OutputType `json:"observedType,omitempty" yaml:"observedType,omitempty"`
	Message      string             `json:"message" yaml:"message"`
}

// EngineResult is one value-free portability result.
type EngineResult struct {
	Engine     string  `json:"engine" yaml:"engine"`
	Status     string  `json:"status" yaml:"status"`
	Diagnostic string  `json:"diagnostic,omitempty" yaml:"diagnostic,omitempty"`
	Checks     []Check `json:"checks" yaml:"checks"`
}

type contextPressure struct {
	Capability  string `json:"capability"`
	Disposition string `json:"disposition"`
	Browser15   string `json:"browser15"`
	NextStep    string `json:"nextStep"`
}

type liveReport struct {
	Version       string   `json:"version"`
	ProfileDigest string   `json:"profileDigest"`
	CheckedAt     string   `json:"checkedAt"`
	Origin        string   `json:"origin"`
	Actions       []string `json:"actions"`
	OK            bool     `json:"ok"`
	Checks        []Check  `json:"checks"`
}

type portabilityReport struct {
	Version          string            `json:"version"`
	ProfileDigest    string            `json:"profileDigest"`
	CheckedAt        string            `json:"checkedAt"`
	Origin           string            `json:"origin"`
	Actions          []string          `json:"actions"`
	OK               bool              `json:"ok"`
	Engines          []EngineResult    `json:"engines"`
	ContractPressure []contextPressure `json:"contractPressure"`
}

// Summary is the normalized, value-free evidence retained by OpenUdon. It
// deliberately omits the input path and Browsertools' static pressure prose.
type Summary struct {
	ReportVersion string         `json:"report_version" yaml:"report_version"`
	SourceSHA256  string         `json:"source_sha256" yaml:"source_sha256"`
	ProfileDigest string         `json:"profile_digest" yaml:"profile_digest"`
	CheckedAt     string         `json:"checked_at" yaml:"checked_at"`
	Origin        string         `json:"origin" yaml:"origin"`
	Actions       []string       `json:"actions" yaml:"actions"`
	OK            bool           `json:"ok" yaml:"ok"`
	Engine        string         `json:"engine,omitempty" yaml:"engine,omitempty"`
	Checks        []Check        `json:"checks,omitempty" yaml:"checks,omitempty"`
	Engines       []EngineResult `json:"engines,omitempty" yaml:"engines,omitempty"`
}

// Attachment retains an operator-local report path only in resumable iCoT
// state. Package review metadata receives Summary, never SourcePath.
type Attachment struct {
	SourcePath string  `json:"source_path" yaml:"source_path"`
	Summary    Summary `json:"summary" yaml:"summary"`
}

type requirement struct {
	Kind   string
	Path   string
	Output *profile.Output
}

// ReadVersionAndProfileDigest strictly reads enough public identity to select
// the exact profile before full validation.
func ReadVersionAndProfileDigest(path string) (string, string, []byte, error) {
	data, err := readStableRegularFile(path)
	if err != nil {
		return "", "", nil, err
	}
	var discriminator struct {
		Version       string `json:"version"`
		ProfileDigest string `json:"profileDigest"`
	}
	if err := decodeStrict(data, &discriminator, false); err != nil {
		// The discriminator intentionally permits the report's remaining fields;
		// full decoding below is strict. Use a generic decode here only to choose
		// the closed wire type.
		var loose map[string]any
		if looseErr := decodeOne(data, &loose); looseErr != nil {
			return "", "", nil, fmt.Errorf("browser verification report: %w", looseErr)
		}
		version, _ := loose["version"].(string)
		digest, _ := loose["profileDigest"].(string)
		discriminator.Version, discriminator.ProfileDigest = version, digest
	}
	return strings.TrimSpace(discriminator.Version), strings.TrimSpace(discriminator.ProfileDigest), data, nil
}

// Inspect validates one exact Browsertools report against a parsed profile and
// derives the safe summary OpenUdon may retain.
func Inspect(path string, prof *profile.Profile, at time.Time) (Summary, error) {
	version, reportDigest, data, err := ReadVersionAndProfileDigest(path)
	if err != nil {
		return Summary{}, err
	}
	if prof == nil {
		return Summary{}, fmt.Errorf("browser verification profile is required")
	}
	wantDigest, err := ProfileDigest(prof)
	if err != nil {
		return Summary{}, err
	}
	if reportDigest != wantDigest {
		return Summary{}, fmt.Errorf("browser verification profile digest %q does not match %q", reportDigest, wantDigest)
	}
	sourceSum := sha256.Sum256(data)
	sourceDigest := "sha256:" + hex.EncodeToString(sourceSum[:])
	switch version {
	case LiveCheckVersion:
		if err := validateReportShape(data, false); err != nil {
			return Summary{}, fmt.Errorf("live-check report: %w", err)
		}
		var report liveReport
		if err := decodeStrict(data, &report, true); err != nil {
			return Summary{}, fmt.Errorf("live-check report: %w", err)
		}
		if err := validateLive(prof, report, at); err != nil {
			return Summary{}, err
		}
		return Summary{
			ReportVersion: report.Version, SourceSHA256: sourceDigest, ProfileDigest: report.ProfileDigest,
			CheckedAt: report.CheckedAt, Origin: report.Origin, Actions: append([]string(nil), report.Actions...),
			OK: report.OK, Engine: "chromium", Checks: cloneChecks(report.Checks),
		}, nil
	case PortabilityVersion:
		if err := validateReportShape(data, true); err != nil {
			return Summary{}, fmt.Errorf("portability report: %w", err)
		}
		var report portabilityReport
		if err := decodeStrict(data, &report, true); err != nil {
			return Summary{}, fmt.Errorf("portability report: %w", err)
		}
		if err := validatePortability(prof, report, at, true); err != nil {
			return Summary{}, err
		}
		return Summary{
			ReportVersion: report.Version, SourceSHA256: sourceDigest, ProfileDigest: report.ProfileDigest,
			CheckedAt: report.CheckedAt, Origin: report.Origin, Actions: append([]string(nil), report.Actions...),
			OK: report.OK, Engines: cloneEngines(report.Engines),
		}, nil
	default:
		if strings.HasPrefix(version, "browsertools.") {
			return Summary{}, fmt.Errorf("browser verification report version %q is not a value-free live or portability contract", version)
		}
		return Summary{}, fmt.Errorf("browser verification report version %q is unsupported", version)
	}
}

// ValidateSummary independently rechecks retained package facts against the
// exact packaged profile. It does not need the external report file.
func ValidateSummary(prof *profile.Profile, summary Summary, at time.Time) error {
	if !validSHA256(summary.SourceSHA256) {
		return fmt.Errorf("browser verification source_sha256 is invalid")
	}
	switch summary.ReportVersion {
	case LiveCheckVersion:
		if summary.Engine != "chromium" || len(summary.Engines) != 0 {
			return fmt.Errorf("live-check summary engine must be chromium")
		}
		return validateLive(prof, liveReport{
			Version: summary.ReportVersion, ProfileDigest: summary.ProfileDigest, CheckedAt: summary.CheckedAt,
			Origin: summary.Origin, Actions: summary.Actions, OK: summary.OK, Checks: summary.Checks,
		}, at)
	case PortabilityVersion:
		if summary.Engine != "" || len(summary.Checks) != 0 {
			return fmt.Errorf("portability summary must contain only per-engine checks")
		}
		return validatePortability(prof, portabilityReport{
			Version: summary.ReportVersion, ProfileDigest: summary.ProfileDigest, CheckedAt: summary.CheckedAt,
			Origin: summary.Origin, Actions: summary.Actions, OK: summary.OK, Engines: summary.Engines,
			ContractPressure: contractPressure(),
		}, at, false)
	default:
		return fmt.Errorf("browser verification report version %q is unsupported", summary.ReportVersion)
	}
}

// ProfileDigest matches Browsertools' canonical typed-profile digest.
func ProfileDigest(prof *profile.Profile) (string, error) {
	if prof == nil {
		return "", fmt.Errorf("browser verification profile is required")
	}
	data, err := json.Marshal(prof)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// LogicalKey identifies duplicate reports without trusting input formatting.
func LogicalKey(summary Summary) string {
	return strings.Join([]string{summary.ProfileDigest, summary.ReportVersion, summary.Origin, strings.Join(summary.Actions, "\x1f")}, "\x00")
}

// EquivalentFacts compares normalized facts while ignoring the raw source
// digest, which can differ only because of harmless JSON formatting.
func EquivalentFacts(left, right Summary) bool {
	left.SourceSHA256, right.SourceSHA256 = "", ""
	return reflect.DeepEqual(left, right)
}

// DecodeStrictJSON rejects unknown fields, duplicate object names, and trailing
// values for package metadata that embeds verification summaries.
func DecodeStrictJSON(data []byte, target any) error {
	return decodeStrict(data, target, true)
}

func validateLive(prof *profile.Profile, report liveReport, at time.Time) error {
	if report.Version != LiveCheckVersion {
		return fmt.Errorf("live-check version must be %q", LiveCheckVersion)
	}
	requirements, err := validateCommon(prof, report.ProfileDigest, report.CheckedAt, report.Origin, report.Actions, at)
	if err != nil {
		return fmt.Errorf("live-check report: %w", err)
	}
	if err := validateChecks(requirements, report.Checks); err != nil {
		return fmt.Errorf("live-check report: %w", err)
	}
	wantOK := allChecksPass(report.Checks)
	if report.OK != wantOK {
		return fmt.Errorf("live-check report ok does not match its validated checks")
	}
	return nil
}

func validatePortability(prof *profile.Profile, report portabilityReport, at time.Time, requirePressure bool) error {
	if report.Version != PortabilityVersion {
		return fmt.Errorf("portability version must be %q", PortabilityVersion)
	}
	requirements, err := validateCommon(prof, report.ProfileDigest, report.CheckedAt, report.Origin, report.Actions, at)
	if err != nil {
		return fmt.Errorf("portability report: %w", err)
	}
	if requirePressure && !reflect.DeepEqual(report.ContractPressure, contractPressure()) {
		return fmt.Errorf("portability report contractPressure does not match the fixed Browsertools inventory")
	}
	if len(report.Engines) < 2 || len(report.Engines) > 3 {
		return fmt.Errorf("portability report must contain Chromium and one or two alternate engines")
	}
	order := []string{"chromium", "firefox", "webkit"}
	position := -1
	seen := map[string]bool{}
	for _, engine := range report.Engines {
		index := indexOf(order, engine.Engine)
		if index < 0 || index <= position || seen[engine.Engine] {
			return fmt.Errorf("portability report engines must be unique and ordered chromium, firefox, webkit")
		}
		position = index
		seen[engine.Engine] = true
	}
	if !seen["chromium"] || (!seen["firefox"] && !seen["webkit"]) {
		return fmt.Errorf("portability report requires Chromium and Firefox or WebKit")
	}
	for index := range report.Engines {
		if err := validateEngine(requirements, report.Engines[index]); err != nil {
			return fmt.Errorf("portability report engine %s: %w", report.Engines[index].Engine, err)
		}
	}
	chromium := report.Engines[0]
	baselineReady := len(chromium.Checks) > 0
	for _, engine := range report.Engines[1:] {
		equalBaseline := baselineReady && reflect.DeepEqual(engine.Checks, chromium.Checks)
		switch engine.Diagnostic {
		case "chromium_baseline_unavailable":
			if baselineReady || !allChecksPass(engine.Checks) {
				return fmt.Errorf("engine %s invents chromium_baseline_unavailable", engine.Engine)
			}
		case "check_shape_mismatch":
			if !baselineReady || !allChecksPass(engine.Checks) || equalBaseline {
				return fmt.Errorf("engine %s invents check_shape_mismatch", engine.Engine)
			}
		case "":
			if engine.Status == "passed" && (!baselineReady || !equalBaseline) {
				return fmt.Errorf("engine %s invents portability success", engine.Engine)
			}
		}
	}
	wantOK := true
	for _, engine := range report.Engines {
		if engine.Status != "passed" {
			wantOK = false
		}
	}
	if report.OK != wantOK {
		return fmt.Errorf("portability report ok does not match its engine results")
	}
	return nil
}

func validateEngine(requirements []requirement, engine EngineResult) error {
	switch engine.Status {
	case "passed":
		if engine.Diagnostic != "" {
			return fmt.Errorf("passed result must not contain a diagnostic")
		}
		if err := validateChecks(requirements, engine.Checks); err != nil {
			return err
		}
		if !allChecksPass(engine.Checks) {
			return fmt.Errorf("passed result contains a failed check")
		}
	case "unavailable":
		if engine.Diagnostic != "engine_unavailable" || len(engine.Checks) != 0 {
			return fmt.Errorf("unavailable result must use engine_unavailable with no checks")
		}
	case "failed":
		switch engine.Diagnostic {
		case "browser_observation_failed":
			if len(engine.Checks) != 0 {
				return fmt.Errorf("browser_observation_failed must not contain checks")
			}
		case "profile_check_failed":
			if err := validateChecks(requirements, engine.Checks); err != nil {
				return err
			}
			if allChecksPass(engine.Checks) {
				return fmt.Errorf("profile_check_failed requires a failed check")
			}
		case "chromium_baseline_unavailable", "check_shape_mismatch":
			if engine.Engine == "chromium" {
				return fmt.Errorf("chromium cannot use alternate-engine diagnostic %q", engine.Diagnostic)
			}
			if err := validateChecks(requirements, engine.Checks); err != nil {
				return err
			}
			if !allChecksPass(engine.Checks) {
				return fmt.Errorf("%s requires locally passing checks", engine.Diagnostic)
			}
		default:
			return fmt.Errorf("failed result has unsupported diagnostic %q", engine.Diagnostic)
		}
	default:
		return fmt.Errorf("status %q is unsupported", engine.Status)
	}
	return nil
}

func validateCommon(prof *profile.Profile, digest, checkedAt, origin string, actions []string, at time.Time) ([]requirement, error) {
	if prof == nil {
		return nil, fmt.Errorf("profile is required")
	}
	wantDigest, err := ProfileDigest(prof)
	if err != nil {
		return nil, err
	}
	if digest != wantDigest {
		return nil, fmt.Errorf("profileDigest must be %q", wantDigest)
	}
	checked, err := time.Parse(time.RFC3339Nano, checkedAt)
	if err != nil || checkedAt != checked.UTC().Format(time.RFC3339Nano) {
		return nil, fmt.Errorf("checkedAt must be canonical UTC RFC3339")
	}
	if at.IsZero() {
		return nil, fmt.Errorf("verification time is required")
	}
	at = at.UTC().Round(0)
	verified, err := time.Parse(time.RFC3339, prof.Verification.LastVerifiedAt)
	if err != nil {
		return nil, fmt.Errorf("profile verification time is invalid")
	}
	expires, err := prof.ExpiresAfter.AddTo(verified)
	if err != nil {
		return nil, fmt.Errorf("profile expiry is invalid")
	}
	if checked.Before(verified) || !checked.Before(expires) || checked.After(at) || !at.Before(expires) {
		return nil, fmt.Errorf("checkedAt is stale, predates the profile, is in the future, or is outside the active profile lifecycle")
	}
	canonicalOrigin, err := profile.ParseOrigin(origin)
	if err != nil || canonicalOrigin != origin || !contains([]string(prof.Info.Origin), origin) {
		return nil, fmt.Errorf("origin must be one exact canonical profile origin")
	}
	if len(actions) == 0 || len(actions) > maxActions || !sortedUnique(actions) {
		return nil, fmt.Errorf("actions must be a non-empty sorted unique set of at most %d names", maxActions)
	}
	for _, action := range actions {
		if _, ok := prof.Actions[action]; !ok {
			return nil, fmt.Errorf("action %q is not declared by the profile", action)
		}
	}
	requirements := buildRequirements(prof, actions)
	if len(requirements) == 0 || len(requirements) > maxChecks {
		return nil, fmt.Errorf("selected actions must derive between 1 and %d read-only checks", maxChecks)
	}
	return requirements, nil
}

func buildRequirements(prof *profile.Profile, actions []string) []requirement {
	var out []requirement
	for _, actionName := range actions {
		action := prof.Actions[actionName]
		for index, step := range action.Sequence {
			base := fmt.Sprintf("actions.%s.sequence[%d]", actionName, index)
			if step.Kind == profile.StepWaitFor {
				if step.WaitFor != nil && step.WaitFor.Locator != nil {
					out = append(out, requirement{Kind: probeLocator, Path: base + ".wait_for"})
				} else if step.WaitFor != nil {
					out = append(out, requirement{Kind: probeNavigationWait, Path: base + ".wait_for.navigation"})
				}
				continue
			}
			if step.Locator() != nil {
				out = append(out, requirement{Kind: probeLocator, Path: base + ".locator"})
			}
			if wait := step.PostWait(); wait != nil {
				if wait.Locator != nil {
					out = append(out, requirement{Kind: probeLocator, Path: base + ".wait_for"})
				} else {
					out = append(out, requirement{Kind: probeNavigationWait, Path: base + ".wait_for.navigation"})
				}
			}
		}
		outputs := make([]string, 0, len(action.Outputs))
		for name := range action.Outputs {
			outputs = append(outputs, name)
		}
		sort.Strings(outputs)
		for _, name := range outputs {
			output := action.Outputs[name]
			out = append(out, requirement{Kind: probeOutput, Path: fmt.Sprintf("actions.%s.outputs.%s", actionName, name), Output: &output})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func validateChecks(requirements []requirement, checks []Check) error {
	if len(checks) != len(requirements) || len(checks) == 0 || len(checks) > maxChecks {
		return fmt.Errorf("checks must exactly match the %d profile-derived requirements", len(requirements))
	}
	for index := range requirements {
		requirement, check := requirements[index], checks[index]
		if check.Kind != requirement.Kind || check.Path != requirement.Path {
			return fmt.Errorf("check[%d] kind/path does not match the profile-derived requirement", index)
		}
		if check.Matches < 0 || check.Matches > maxMatchCount {
			return fmt.Errorf("check[%d] matches is outside bounds", index)
		}
		if err := validateCheck(requirement, check); err != nil {
			return fmt.Errorf("check[%d]: %w", index, err)
		}
	}
	return nil
}

func validateCheck(requirement requirement, check Check) error {
	const genericFailure = "read-only browser observation failed closed"
	if check.Message == genericFailure {
		if check.OK {
			return fmt.Errorf("failed-closed check cannot pass")
		}
		// Browsertools returns before its kind-specific assessment when the
		// backend supplies a fixed failure code, so even an output failure has
		// no expected or observed type in this branch.
		if check.ExpectedType != "" || check.ObservedType != "" {
			return fmt.Errorf("failed-closed check must not contain output types")
		}
		return nil
	}
	switch requirement.Kind {
	case probeLocator:
		if check.ExpectedType != "" || check.ObservedType != "" {
			return fmt.Errorf("locator check contains output types")
		}
		wantOK := check.Matches == 1
		wantMessage := "declared accessibility locator did not resolve exactly once"
		if wantOK {
			wantMessage = "declared accessibility locator resolved exactly once"
		}
		if check.OK != wantOK || check.Message != wantMessage {
			return fmt.Errorf("locator result is inconsistent with its match count")
		}
	case probeNavigationWait:
		if check.ExpectedType != "" || check.ObservedType != "" {
			return fmt.Errorf("navigation check contains output types")
		}
		wantMessage := "declared navigation wait was not reached within the bounded observation"
		if check.OK {
			wantMessage = "declared navigation wait was reached without executing an action macro"
		}
		if check.Message != wantMessage {
			return fmt.Errorf("navigation result message does not match its status")
		}
	case probeOutput:
		if check.ExpectedType != requirement.Output.Type {
			return fmt.Errorf("output expectedType does not match the profile")
		}
		if err := validateObservedType(check.ObservedType); err != nil {
			return err
		}
		presence := requirement.Output.Source == profile.OutputA11y && requirement.Output.Presence != nil && *requirement.Output.Presence
		countOK := check.Matches == 1
		if requirement.Output.Type == profile.OutputArray {
			countOK = check.Matches > 0
		}
		if presence {
			countOK = true
		}
		wantOK := countOK && check.ObservedType == requirement.Output.Type
		wantMessage := "declared output source or JSON type did not match"
		if wantOK {
			wantMessage = "declared output source and JSON type matched"
		}
		if check.OK != wantOK || check.Message != wantMessage {
			return fmt.Errorf("output result is inconsistent with its declared type and match count")
		}
	default:
		return fmt.Errorf("unsupported check kind %q", requirement.Kind)
	}
	return nil
}

func validateObservedType(value profile.OutputType) error {
	if value == "" {
		return nil
	}
	for _, allowed := range []profile.OutputType{profile.OutputString, profile.OutputInteger, profile.OutputNumber, profile.OutputBoolean, profile.OutputArray, profile.OutputObject, profile.OutputNull} {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("observedType %q is unsupported", value)
}

func readStableRegularFile(path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("browser verification report path is required")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("browser verification report %s: %w", path, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("browser verification report must be a non-symlink regular file: %s", path)
	}
	if before.Size() <= 0 || before.Size() > MaxReportBytes {
		return nil, fmt.Errorf("browser verification report size must be between 1 and %d bytes: %s", MaxReportBytes, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("browser verification report %s: %w", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("browser verification report changed while opening: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxReportBytes+1))
	if err != nil {
		return nil, fmt.Errorf("browser verification report %s: %w", path, err)
	}
	if len(data) == 0 || len(data) > MaxReportBytes {
		return nil, fmt.Errorf("browser verification report size must be between 1 and %d bytes: %s", MaxReportBytes, path)
	}
	after, err := file.Stat()
	if err != nil || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return nil, fmt.Errorf("browser verification report changed while reading: %s", path)
	}
	return data, nil
}

func decodeStrict(data []byte, target any, disallowUnknown bool) error {
	if err := rejectDuplicateJSONNames(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("contains trailing JSON")
		}
		return err
	}
	return nil
}

func validateReportShape(data []byte, portability bool) error {
	var object map[string]json.RawMessage
	if err := decodeStrict(data, &object, false); err != nil {
		return err
	}
	required := []string{"version", "profileDigest", "checkedAt", "origin", "actions", "ok"}
	if portability {
		required = append(required, "engines", "contractPressure")
	} else {
		required = append(required, "checks")
	}
	if err := requireJSONFields(object, required...); err != nil {
		return err
	}
	if err := requireJSONArray(object["actions"], "actions", false); err != nil {
		return err
	}
	if !portability {
		return validateCheckArrayShape(object["checks"], "checks", false)
	}
	if err := requireJSONArray(object["contractPressure"], "contractPressure", false); err != nil {
		return err
	}
	var pressure []map[string]json.RawMessage
	if err := json.Unmarshal(object["contractPressure"], &pressure); err != nil {
		return fmt.Errorf("contractPressure must be an array of objects")
	}
	for index, item := range pressure {
		if err := requireJSONFields(item, "capability", "disposition", "browser15", "nextStep"); err != nil {
			return fmt.Errorf("contractPressure[%d]: %w", index, err)
		}
	}
	if err := requireJSONArray(object["engines"], "engines", false); err != nil {
		return err
	}
	var engines []map[string]json.RawMessage
	if err := json.Unmarshal(object["engines"], &engines); err != nil {
		return fmt.Errorf("engines must be an array of objects")
	}
	for index, engine := range engines {
		if err := requireJSONFields(engine, "engine", "status", "checks"); err != nil {
			return fmt.Errorf("engines[%d]: %w", index, err)
		}
		if err := validateCheckArrayShape(engine["checks"], fmt.Sprintf("engines[%d].checks", index), true); err != nil {
			return err
		}
	}
	return nil
}

func validateCheckArrayShape(raw json.RawMessage, label string, allowEmpty bool) error {
	if err := requireJSONArray(raw, label, allowEmpty); err != nil {
		return err
	}
	var checks []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &checks); err != nil {
		return fmt.Errorf("%s must be an array of objects", label)
	}
	for index, check := range checks {
		if err := requireJSONFields(check, "kind", "path", "ok", "message"); err != nil {
			return fmt.Errorf("%s[%d]: %w", label, index, err)
		}
	}
	return nil
}

func requireJSONFields(object map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		raw, ok := object[field]
		if !ok || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("required field %q is missing or null", field)
		}
	}
	return nil
}

func requireJSONArray(raw json.RawMessage, label string, allowEmpty bool) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' || bytes.Equal(trimmed, []byte("null")) {
		return fmt.Errorf("%s must be a JSON array", label)
	}
	var values []json.RawMessage
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return fmt.Errorf("%s must be a JSON array", label)
	}
	if !allowEmpty && len(values) == 0 {
		return fmt.Errorf("%s must not be empty", label)
	}
	return nil
}

func rejectDuplicateJSONNames(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("contains trailing JSON token %v", token)
		}
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := nameToken.(string)
			if !ok {
				return fmt.Errorf("JSON object member name is not a string")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON field %q", name)
			}
			seen[name] = true
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return fmt.Errorf("JSON array is not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func decodeOne(data []byte, target any) error { return decodeStrict(data, target, false) }

func contractPressure() []contextPressure {
	return []contextPressure{
		{Capability: "screenshot", Disposition: "supported_private", Browser15: "no portable field", NextStep: "retain only as reviewed private evidence"},
		{Capability: "trace", Disposition: "supported_private", Browser15: "no portable field", NextStep: "retain only as reviewed private evidence"},
		{Capability: "har", Disposition: "supported_private", Browser15: "no portable field", NextStep: "retain only as reviewed private evidence"},
		{Capability: "popup_context", Disposition: "proposal_candidate", Browser15: "no page-context selector", NextStep: "define an explicit page-context reference for browser.1.6 review"},
		{Capability: "iframe_context", Disposition: "proposal_candidate", Browser15: "no frame-context selector", NextStep: "define an exact-origin frame reference for browser.1.6 review"},
		{Capability: "download", Disposition: "deferred", Browser15: "no download result contract", NextStep: "keep blocked until lifecycle and artifact semantics are specified"},
		{Capability: "upload", Disposition: "deferred", Browser15: "no private-input binding", NextStep: "keep blocked until runtime-owned private input semantics are specified"},
		{Capability: "permission", Disposition: "deferred", Browser15: "no permission grant contract", NextStep: "keep ungranted until origin-scoped runtime policy is specified"},
		{Capability: "visual_interaction", Disposition: "proposal_candidate", Browser15: "accessibility locators only", NextStep: "define reviewed bounded visual locator evidence for browser.1.6 review"},
	}
}

func cloneChecks(values []Check) []Check { return append([]Check(nil), values...) }

func cloneEngines(values []EngineResult) []EngineResult {
	out := append([]EngineResult(nil), values...)
	for index := range out {
		out[index].Checks = cloneChecks(out[index].Checks)
	}
	return out
}

func allChecksPass(values []Check) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !value.OK {
			return false
		}
	}
	return true
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if strings.TrimSpace(value) == "" || value != strings.TrimSpace(value) || (index > 0 && values[index-1] >= value) {
			return false
		}
	}
	return true
}

func validSHA256(value string) bool {
	raw := strings.TrimPrefix(value, "sha256:")
	if value != "sha256:"+raw || len(raw) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func indexOf(values []string, want string) int {
	for index, value := range values {
		if value == want {
			return index
		}
	}
	return -1
}
