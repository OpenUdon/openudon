package icot

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/browsertools/disclosurepath"
	"github.com/OpenUdon/browsertools/profile"
)

const browserAuthoringPlanVersion = "openudon.browser-authoring-handoff.v1"

var (
	browserProfileIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	browserActionIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

type browserAuthoringPlanInput struct {
	ExampleDir  string
	TargetURL   string
	Origins     []string
	ProfileID   string
	ActionHint  string
	LoginState  string
	PrivateRoot string
}

type browserAuthoringPlan struct {
	Version              string                       `json:"version"`
	Status               string                       `json:"status"`
	Example              string                       `json:"example"`
	Target               browserAuthoringTarget       `json:"target"`
	PrivateRoot          string                       `json:"private_root"`
	Authority            browserAuthoringAuthority    `json:"authority"`
	Stages               []browserAuthoringStage      `json:"stages"`
	AcceptedOutputs      []browserAuthoringOutput     `json:"accepted_outputs"`
	RejectedInputs       []string                     `json:"rejected_inputs"`
	Resume               *browserAuthoringResume      `json:"resume,omitempty"`
	Diagnostics          []browserAuthoringDiagnostic `json:"diagnostics"`
	RetentionClass       string                       `json:"retention_class"`
	SafeToArchive        bool                         `json:"safe_to_archive"`
	RedactionBeforeShare bool                         `json:"redaction_required_before_share"`
}

type browserAuthoringTarget struct {
	URL                string   `json:"url"`
	ApprovedOrigins    []string `json:"approved_origins"`
	ProfileID          string   `json:"profile_id"`
	ActionHint         string   `json:"action_hint"`
	LoginStateRequired bool     `json:"login_state_required"`
}

type browserAuthoringAuthority struct {
	ICoTLaunchesBrowser                  bool     `json:"icot_launches_browser"`
	ICoTRunsBrowsertools                 bool     `json:"icot_runs_browsertools"`
	ICoTReadsCredentialEnvironment       bool     `json:"icot_reads_credential_environment"`
	HandoffCarriesCredentialValues       bool     `json:"handoff_carries_credential_values"`
	HandoffCarriesSessionState           bool     `json:"handoff_carries_session_state"`
	ExternalRunMayLaunchBrowser          bool     `json:"external_run_may_launch_browser"`
	ExternalCaptureAllowedNetworkMethods []string `json:"external_capture_allowed_network_methods"`
	RequiresExternalOperatorRun          bool     `json:"requires_external_operator_run"`
}

type browserAuthoringStage struct {
	ID           string                        `json:"id"`
	Kind         string                        `json:"kind"`
	Argv         []string                      `json:"argv,omitempty"`
	Produces     string                        `json:"produces,omitempty"`
	Requires     []string                      `json:"requires,omitempty"`
	OperatorGate string                        `json:"operator_gate,omitempty"`
	Placeholders []browserAuthoringPlaceholder `json:"placeholders,omitempty"`
}

type browserAuthoringPlaceholder struct {
	Token      string `json:"token"`
	Expansion  string `json:"expansion"`
	Source     string `json:"source"`
	Constraint string `json:"constraint"`
}

type browserAuthoringOutput struct {
	Version string `json:"version"`
	Use     string `json:"use"`
}

type browserAuthoringResume struct {
	Argv []string `json:"argv"`
}

type browserAuthoringDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func runBrowserAuthoring(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] != "plan" {
		fmt.Fprintln(errOut, "usage: icot browser-authoring plan --example DIR --url URL --origin ORIGIN --profile-id ID --action-hint ACTION --login-state required|not-required --private-root DIR [--out FILE|-]")
		return 2
	}
	fs := flag.NewFlagSet("icot browser-authoring plan", flag.ContinueOnError)
	fs.SetOutput(errOut)
	example := fs.String("example", "", "OpenUdon example directory that will later consume the reviewed profile")
	targetURL := fs.String("url", "", "Clean HTTPS or loopback URL selected for authoring")
	profileID := fs.String("profile-id", "", "Stable lowercase profile ID")
	actionHint := fs.String("action-hint", "", "Stable action hint for the requested UI capability")
	loginState := fs.String("login-state", "", "Login posture: required or not-required")
	privateRoot := fs.String("private-root", "", "Disjoint private Browsertools work/cache root")
	outPath := fs.String("out", "-", "New 0600 JSON path or - for stdout")
	var origins repeatedFlag
	fs.Var(&origins, "origin", "Exact approved origin; repeat for every origin")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "icot browser-authoring plan: unexpected positional arguments")
		return 2
	}
	plan, err := buildBrowserAuthoringPlan(browserAuthoringPlanInput{
		ExampleDir: *example, TargetURL: *targetURL, Origins: append([]string(nil), origins...),
		ProfileID: *profileID, ActionHint: *actionHint, LoginState: *loginState,
		PrivateRoot: *privateRoot,
	})
	if err != nil {
		fmt.Fprintln(errOut, "icot browser-authoring plan:", err)
		return 2
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fmt.Fprintln(errOut, "icot browser-authoring plan:", err)
		return 1
	}
	data = append(data, '\n')
	if strings.TrimSpace(*outPath) == "-" {
		if _, err := out.Write(data); err != nil {
			fmt.Fprintln(errOut, "icot browser-authoring plan:", err)
			return 1
		}
		return 0
	}
	if err := writeNewPrivatePlan(*outPath, plan.PrivateRoot, data); err != nil {
		fmt.Fprintln(errOut, "icot browser-authoring plan:", err)
		return 1
	}
	return 0
}

func optionalBrowserAuthoringPlan(input browserAuthoringPlanInput) (*browserAuthoringPlan, error) {
	if !browserAuthoringInputProvided(input) {
		return nil, nil
	}
	plan, err := buildBrowserAuthoringPlan(input)
	if err != nil {
		return nil, fmt.Errorf("browser authoring handoff: %w", err)
	}
	return &plan, nil
}

func browserAuthoringInputProvided(input browserAuthoringPlanInput) bool {
	return strings.TrimSpace(input.TargetURL) != "" || strings.TrimSpace(input.ProfileID) != "" ||
		strings.TrimSpace(input.ActionHint) != "" || strings.TrimSpace(input.LoginState) != "" ||
		strings.TrimSpace(input.PrivateRoot) != "" || len(input.Origins) > 0
}

func buildBrowserAuthoringPlan(input browserAuthoringPlanInput) (browserAuthoringPlan, error) {
	example, err := canonicalProspectivePath("example", input.ExampleDir)
	if err != nil {
		return browserAuthoringPlan{}, err
	}
	privateRoot, err := canonicalPrivateRoot(input.PrivateRoot)
	if err != nil {
		return browserAuthoringPlan{}, err
	}
	if err := requireDisjointBrowserRoots(example, privateRoot); err != nil {
		return browserAuthoringPlan{}, err
	}
	targetURL, targetOrigin, err := normalizeBrowserAuthoringURL(input.TargetURL)
	if err != nil {
		return browserAuthoringPlan{}, err
	}
	origins, err := normalizeBrowserAuthoringOrigins(input.Origins)
	if err != nil {
		return browserAuthoringPlan{}, err
	}
	if !stringSliceContainsExact(origins, targetOrigin) {
		return browserAuthoringPlan{}, fmt.Errorf("approved origins must include target origin %s", targetOrigin)
	}
	profileID := strings.TrimSpace(input.ProfileID)
	if !browserProfileIDPattern.MatchString(profileID) {
		return browserAuthoringPlan{}, fmt.Errorf("profile ID %q must match %s", profileID, browserProfileIDPattern)
	}
	actionHint := strings.TrimSpace(input.ActionHint)
	if !browserActionIDPattern.MatchString(actionHint) {
		return browserAuthoringPlan{}, fmt.Errorf("action hint %q must match %s", actionHint, browserActionIDPattern)
	}
	loginState := strings.ToLower(strings.TrimSpace(input.LoginState))
	if loginState != "required" && loginState != "not-required" {
		return browserAuthoringPlan{}, fmt.Errorf("login state must be required or not-required")
	}

	plan := browserAuthoringPlan{
		Version: browserAuthoringPlanVersion, Status: "ready", Example: example,
		Target: browserAuthoringTarget{
			URL: targetURL, ApprovedOrigins: origins, ProfileID: profileID,
			ActionHint: actionHint, LoginStateRequired: loginState == "required",
		},
		PrivateRoot: privateRoot,
		Authority: browserAuthoringAuthority{
			ICoTLaunchesBrowser: false, ICoTRunsBrowsertools: false,
			ICoTReadsCredentialEnvironment: false, HandoffCarriesCredentialValues: false,
			HandoffCarriesSessionState: false, ExternalRunMayLaunchBrowser: true,
			ExternalCaptureAllowedNetworkMethods: []string{"GET", "HEAD"},
			RequiresExternalOperatorRun:          true,
		},
		Stages: []browserAuthoringStage{},
		AcceptedOutputs: []browserAuthoringOutput{
			{Version: "browsertools.guided-authoring.v1", Use: "pass the reviewed JSON explicitly as --browser-profile ID=PATH; OpenUdon stages only its verified embedded uws.browser.1.5 profile"},
			{Version: "uws.browser.1.5", Use: "pass the reviewed profile explicitly as --browser-profile ID=PATH"},
			{Version: "uws.browser-authentication.1.0", Use: "pass a separately reviewed local authentication profile as another --browser-profile ID=PATH"},
		},
		RejectedInputs: []string{
			"private_raw cache payload", "browsertools.private-rich-evidence.v1",
			"screenshot", "trace", "HAR", "cookie or storage-state export",
			"credential value", "MFA response", "browser session handle",
		},
		Diagnostics:    []browserAuthoringDiagnostic{},
		RetentionClass: "local_ephemeral", SafeToArchive: false, RedactionBeforeShare: true,
	}
	if loginState == "required" {
		plan.Status = "needs_reviewed_profiles"
		plan.Authority.ExternalCaptureAllowedNetworkMethods = []string{}
		plan.Diagnostics = append(plan.Diagnostics,
			browserAuthoringDiagnostic{Code: "authenticated_capability_observation_not_supported", Message: "Browsertools does not carry an authenticated authoring context into capability capture; provide separately reviewed authentication and capability profiles."},
			browserAuthoringDiagnostic{Code: "interactive_login_remains_external", Message: "Credentials, MFA, cookies, storage state, and the headed operator session remain Browsertools/runtime-owned and never enter iCoT."},
		)
		plan.Stages = append(plan.Stages,
			browserAuthoringStage{ID: "author_authentication_profile", Kind: "manual_review", Produces: "uws.browser-authentication.1.0", OperatorGate: "Author and inspect an exact secret-free authentication recipe in Browsertools; popup/iframe SSO, CAPTCHA, enrollment, recovery, and consent are outside the current contract."},
			browserAuthoringStage{ID: "author_capability_profile", Kind: "manual_review", Produces: "uws.browser.1.5", OperatorGate: "Author the protected-page capability from separately reviewed evidence; do not export or reuse the live authentication session."},
			browserAuthoringStage{ID: "resume_icot", Kind: "operator_handoff", Requires: []string{"uws.browser-authentication.1.0", "uws.browser.1.5"}, OperatorGate: "Rerun iCoT with both reviewed files supplied through separate --browser-profile ID=PATH flags."},
		)
		if err := validateBrowserAuthoringPlan(plan); err != nil {
			return browserAuthoringPlan{}, err
		}
		return plan, nil
	}

	capturePath := filepath.Join(privateRoot, profileID+".capture.json")
	evidencePath := filepath.Join(privateRoot, profileID+".evidence.json")
	guidedPath := filepath.Join(privateRoot, profileID+".guided-authoring.json")
	cacheRoot := filepath.Join(privateRoot, "cache")
	captureArgv := []string{"browsertools", "capture", "chromium", "--url", targetURL, "--action-hint", actionHint}
	for _, origin := range origins {
		captureArgv = append(captureArgv, "--allow-origin", origin)
	}
	captureArgv = append(captureArgv, "--cache-root", cacheRoot, "--retain-for", "1h")
	plan.Stages = append(plan.Stages,
		browserAuthoringStage{ID: "doctor", Kind: "external_command", Argv: []string{"browsertools", "playwright", "doctor", "--engine", "chromium"}, OperatorGate: "Browser installation is explicit; iCoT never installs or launches it."},
		browserAuthoringStage{ID: "capture", Kind: "external_command", Argv: captureArgv, Produces: "private_raw cache manifest", Requires: []string{"doctor"}, OperatorGate: "Run explicitly in Browsertools; the command is headless, ephemeral, exact-origin, and GET/HEAD-only."},
		browserAuthoringStage{ID: "export_private_raw", Kind: "external_command", Argv: []string{"browsertools", "cache", "get", "--root", cacheRoot, "--id", "<capture-id>", "--at", "<assessment-rfc3339>", "--out", capturePath}, Produces: capturePath, Requires: []string{"capture"}, OperatorGate: "Use the exact cache ID printed by Browsertools; keep this file private and within retention.", Placeholders: []browserAuthoringPlaceholder{{Token: "<capture-id>", Expansion: "scalar", Source: "capture stdout manifest", Constraint: "exact complete sha256 cache ID"}, {Token: "<assessment-rfc3339>", Expansion: "scalar", Source: "operator clock at export", Constraint: "RFC3339 time before cache expiry"}}},
		browserAuthoringStage{ID: "review_private_raw", Kind: "manual_review", Requires: []string{"export_private_raw"}, OperatorGate: "Inspect and redact credentials, personal data, URLs, headers, and session material before normalization."},
		browserAuthoringStage{ID: "normalize", Kind: "external_command", Argv: []string{"browsertools", "evidence", "import", "--adapter", "playwright", "--input", capturePath, "--origin", targetOrigin, "--action-hint", actionHint, "<reviewed-redaction-argv>", "--out", evidencePath}, Produces: evidencePath, Requires: []string{"review_private_raw"}, OperatorGate: "Choose one exact redaction argv expansion only after inspecting and, when necessary, redacting the raw capture.", Placeholders: []browserAuthoringPlaceholder{{Token: "<reviewed-redaction-argv>", Expansion: "argv_fragment", Source: "operator raw-evidence review", Constraint: "either --redaction-status not_required, or --redaction-status redacted followed by one or more --redacted-field FIELD pairs naming every replaced field"}}},
		browserAuthoringStage{ID: "guide", Kind: "external_command", Argv: []string{"browsertools", "guide", "author", "--evidence", evidencePath, "--at", "<assessment-rfc3339>", "--out", guidedPath}, Produces: guidedPath, Requires: []string{"normalize"}, OperatorGate: "The operator explicitly chooses action intent, parameters, outputs, macros, side effects, confirmation, expiry, and ambiguity resolutions.", Placeholders: []browserAuthoringPlaceholder{{Token: "<assessment-rfc3339>", Expansion: "scalar", Source: "operator clock at guided review", Constraint: "RFC3339 time at or after evidence observation and before expiry"}}},
		browserAuthoringStage{ID: "resume_icot", Kind: "operator_handoff", Argv: []string{"icot", "--example", example, "--browser-profile", profileID + "=" + guidedPath}, Requires: []string{"guide"}, OperatorGate: "Review the guided bundle, then rerun iCoT. OpenUdon revalidates it and stages only the embedded profile after proposal approval."},
	)
	plan.Resume = &browserAuthoringResume{Argv: append([]string(nil), plan.Stages[len(plan.Stages)-1].Argv...)}
	if err := validateBrowserAuthoringPlan(plan); err != nil {
		return browserAuthoringPlan{}, err
	}
	return plan, nil
}

func validateBrowserAuthoringPlan(plan browserAuthoringPlan) error {
	if plan.Version != browserAuthoringPlanVersion {
		return fmt.Errorf("browser authoring plan version is invalid")
	}
	if plan.Authority.ICoTLaunchesBrowser || plan.Authority.ICoTRunsBrowsertools || plan.Authority.ICoTReadsCredentialEnvironment || plan.Authority.HandoffCarriesCredentialValues || plan.Authority.HandoffCarriesSessionState || !plan.Authority.RequiresExternalOperatorRun {
		return fmt.Errorf("browser authoring plan exceeds the iCoT authority boundary")
	}
	if plan.RetentionClass != "local_ephemeral" || plan.SafeToArchive || !plan.RedactionBeforeShare {
		return fmt.Errorf("browser authoring plan retention metadata is invalid")
	}
	switch plan.Status {
	case "ready":
		if plan.Target.LoginStateRequired || plan.Resume == nil || !plan.Authority.ExternalRunMayLaunchBrowser || !exactStringSlicesEqual(plan.Authority.ExternalCaptureAllowedNetworkMethods, []string{"GET", "HEAD"}) {
			return fmt.Errorf("ready browser authoring plan is missing its external resume boundary")
		}
	case "needs_reviewed_profiles":
		if !plan.Target.LoginStateRequired || plan.Resume != nil || len(plan.Authority.ExternalCaptureAllowedNetworkMethods) != 0 {
			return fmt.Errorf("login-required browser authoring plan exposes capture authority")
		}
	default:
		return fmt.Errorf("browser authoring plan status %q is unsupported", plan.Status)
	}
	seenStages := map[string]bool{}
	for _, stage := range plan.Stages {
		if stage.ID == "" || stage.Kind == "" || seenStages[stage.ID] {
			return fmt.Errorf("browser authoring plan contains an invalid or duplicate stage")
		}
		if plan.Status == "needs_reviewed_profiles" && len(stage.Argv) != 0 {
			return fmt.Errorf("login-required browser authoring plan exposes executable argv")
		}
		for _, dependency := range stage.Requires {
			if strings.HasPrefix(dependency, "uws.") {
				continue
			}
			if !seenStages[dependency] {
				return fmt.Errorf("browser authoring stage %s requires unknown or later stage %s", stage.ID, dependency)
			}
		}
		declared := map[string]bool{}
		for _, placeholder := range stage.Placeholders {
			if placeholder.Token == "" || placeholder.Source == "" || placeholder.Constraint == "" || (placeholder.Expansion != "scalar" && placeholder.Expansion != "argv_fragment") || declared[placeholder.Token] {
				return fmt.Errorf("browser authoring stage %s has an invalid placeholder declaration", stage.ID)
			}
			declared[placeholder.Token] = true
		}
		used := map[string]bool{}
		for _, argument := range stage.Argv {
			if strings.HasPrefix(argument, "<") && strings.HasSuffix(argument, ">") {
				if !declared[argument] {
					return fmt.Errorf("browser authoring stage %s uses undeclared placeholder %s", stage.ID, argument)
				}
				used[argument] = true
			}
		}
		for token := range declared {
			if !used[token] {
				return fmt.Errorf("browser authoring stage %s declares unused placeholder %s", stage.ID, token)
			}
		}
		seenStages[stage.ID] = true
	}
	return nil
}

func normalizeBrowserAuthoringURL(raw string) (string, string, error) {
	raw = strings.TrimSpace(raw)
	if containsPlanDelimiterOrControl(raw) {
		return "", "", fmt.Errorf("target URL contains a reserved delimiter or control character")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Host == "" {
		return "", "", fmt.Errorf("target URL must be an absolute HTTPS or loopback HTTP URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", "", fmt.Errorf("target URL must not contain userinfo, query, fragment, or opaque data")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && browserAuthoringLoopbackHost(parsed.Hostname())) {
		return "", "", fmt.Errorf("target URL must use HTTPS or loopback HTTP")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	if err := disclosurepath.Validate(parsed.EscapedPath()); err != nil {
		return "", "", fmt.Errorf("target URL path is unsafe")
	}
	origin, err := profile.ParseOrigin(strings.ToLower(parsed.Scheme) + "://" + parsed.Host)
	if err != nil {
		return "", "", fmt.Errorf("target URL origin: %w", err)
	}
	canonicalOrigin, err := url.Parse(origin)
	if err != nil {
		return "", "", err
	}
	parsed.Scheme = canonicalOrigin.Scheme
	parsed.Host = canonicalOrigin.Host
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String(), origin, nil
}

func normalizeBrowserAuthoringOrigins(values []string) ([]string, error) {
	if len(values) == 0 || len(values) > 32 {
		return nil, fmt.Errorf("between 1 and 32 approved origins are required")
	}
	seen := map[string]bool{}
	origins := make([]string, 0, len(values))
	for _, raw := range values {
		origin, err := profile.ParseOrigin(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("approved origin is invalid: %w", err)
		}
		parsed, parseErr := url.Parse(origin)
		if parseErr != nil || parsed == nil || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && browserAuthoringLoopbackHost(parsed.Hostname()))) {
			return nil, fmt.Errorf("approved origin must use HTTPS or loopback HTTP")
		}
		if seen[origin] {
			return nil, fmt.Errorf("approved origin %s is duplicated", origin)
		}
		seen[origin] = true
		origins = append(origins, origin)
	}
	sort.Strings(origins)
	return origins, nil
}

func browserAuthoringLoopbackHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func cleanRequiredPath(label, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	if containsPlanDelimiterOrControl(value) {
		return "", fmt.Errorf("%s contains a reserved delimiter or control character", label)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	return filepath.Clean(absolute), nil
}

func containsPlanDelimiterOrControl(value string) bool {
	return strings.ContainsAny(value, "<>") || strings.IndexFunc(value, func(r rune) bool {
		return r < 0x20 || r == 0x7f
	}) >= 0
}

func canonicalProspectivePath(label, value string) (string, error) {
	absolute, err := cleanRequiredPath(label, value)
	if err != nil {
		return "", err
	}
	probe := absolute
	var suffix []string
	for {
		_, statErr := os.Lstat(probe)
		if statErr == nil {
			break
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("%s: %w", label, statErr)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", fmt.Errorf("%s has no existing path ancestor", label)
		}
		suffix = append(suffix, filepath.Base(probe))
		probe = parent
	}
	resolved, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	for index := len(suffix) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, suffix[index])
	}
	return filepath.Clean(resolved), nil
}

func canonicalPrivateRoot(value string) (string, error) {
	absolute, err := cleanRequiredPath("private root", value)
	if err != nil {
		return "", err
	}
	volumeRoot := filepath.Clean(filepath.VolumeName(absolute) + string(filepath.Separator))
	if absolute == volumeRoot {
		return "", fmt.Errorf("private root must not be a filesystem root")
	}
	if home, homeErr := os.UserHomeDir(); homeErr == nil && absolute == filepath.Clean(home) {
		return "", fmt.Errorf("private root must not be the home directory")
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("private root must be an existing directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("private root must be an existing non-symlink directory")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("private root permissions must exclude group and other access")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("private root: %w", err)
	}
	return filepath.Clean(resolved), nil
}

func requireDisjointBrowserRoots(example, privateRoot string) error {
	volumeRoot := filepath.Clean(filepath.VolumeName(privateRoot) + string(filepath.Separator))
	if privateRoot == volumeRoot {
		return fmt.Errorf("private root must not be a filesystem root")
	}
	if home, err := os.UserHomeDir(); err == nil && privateRoot == filepath.Clean(home) {
		return fmt.Errorf("private root must not be the home directory")
	}
	if pathWithin(example, privateRoot) || pathWithin(privateRoot, example) {
		return fmt.Errorf("private root and OpenUdon example must be disjoint")
	}
	return nil
}

func pathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func stringSliceContainsExact(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func writeNewPrivatePlan(path, privateRoot string, data []byte) error {
	var err error
	path, err = canonicalProspectivePath("output path", path)
	if err != nil {
		return err
	}
	if path == privateRoot || !pathWithin(privateRoot, path) {
		return fmt.Errorf("output path must be inside the private root")
	}
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil {
		return fmt.Errorf("inspect output directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("output directory must be an existing non-symlink directory")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func exactStringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneBrowserAuthoringPlan(plan *browserAuthoringPlan) *browserAuthoringPlan {
	if plan == nil {
		return nil
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return nil
	}
	var cloned browserAuthoringPlan
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil
	}
	return &cloned
}

func exampleDirForPlan(value string) string { return strings.TrimSpace(value) }
