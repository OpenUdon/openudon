package icot

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/evidence/redact"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/processgroup"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	liveAuthorProtocol              = "browsertools.author-session.v2"
	liveAuthorMaxLineBytes          = 64 << 10
	liveAuthorDefaultTimeout        = 10 * time.Minute
	liveAuthorResultMaxBytes        = 20 << 20
	liveAuthorAbsoluteMaxCandidates = 512
	liveAuthorAbsoluteMaxOutputs    = 32
	liveAuthorSelectedMaxOutputs    = 16
	liveAuthorMaxDiagnostics        = 256
	liveAuthorMaxCapabilities       = 32
)

var (
	liveCandidatePattern      = regexp.MustCompile(`^candidate-[a-f0-9]{16}$`)
	liveContextPattern        = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	liveDiagnosticPattern     = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	liveReviewDecisionPattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
	liveOutputKeyPattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	liveApprovalPattern       = regexp.MustCompile(`^approval-[0-9]{4}$`)
	livePortableRoles         = map[string]bool{
		"button": true, "link": true, "textbox": true, "checkbox": true, "radio": true,
		"dialog": true, "status": true, "alert": true, "heading": true, "img": true,
		"list": true, "listitem": true, "combobox": true, "option": true, "menu": true,
		"menuitem": true, "tab": true, "tabpanel": true, "table": true, "row": true,
		"cell": true, "region": true, "navigation": true, "article": true, "form": true,
		"search": true, "switch": true, "group": true,
	}
	liveOutputTypes        = map[string]bool{"string": true, "integer": true, "number": true, "boolean": true, "presence": true}
	liveOutputLocatorModes = map[string]bool{"exact_name": true, "unique_role": true}
	liveOutputControlRoles = map[string]bool{"button": true, "textbox": true, "checkbox": true, "radio": true, "combobox": true, "option": true, "menuitem": true, "tab": true, "switch": true}
	liveOTPChallengeKinds  = []string{"totp", "sms_otp", "email_otp", "voice_otp"}
	liveMFAChallengeKinds  = []string{"push", "push_number_match", "passkey", "security_key"}
)

type liveAuthorConfig struct {
	ExampleDir          string
	Browsertools        string
	DriverDir           string
	URL                 string
	DashboardURL        string
	GoalURL             string
	Goal                string
	Origins             []string
	PrivateRoot         string
	ProfileID           string
	AfterAuthentication string
	GoalRole            string
	GoalLabel           string
	GoalContext         string
	Provider            string
	Model               string
	NoLLM               bool
	Temperature         float64
	Yes                 bool
}

type liveGoalPredicate struct {
	Origin      string `json:"origin"`
	Path        string `json:"path"`
	Context     string `json:"context,omitempty"`
	Role        string `json:"role"`
	Label       string `json:"label,omitempty"`
	CandidateID string `json:"candidateId,omitempty"`
}

type liveBounds struct {
	NavigationTimeoutMS int64 `json:"navigationTimeoutMs"`
	TotalTimeoutMS      int64 `json:"totalTimeoutMs"`
	MaxRequests         int   `json:"maxRequests"`
	MaxResponseBytes    int64 `json:"maxResponseBytes"`
	MaxObservations     int   `json:"maxObservations"`
	MaxCandidates       int   `json:"maxCandidates"`
	MaxOutputs          int   `json:"maxOutputs"`
}

type liveOutputRequest struct {
	CandidateID string `json:"candidateId"`
	Key         string `json:"key"`
	Type        string `json:"type"`
	LocatorMode string `json:"locatorMode"`
}

type liveClientMessage struct {
	Protocol      string               `json:"protocol"`
	Type          string               `json:"type"`
	Title         string               `json:"title,omitempty"`
	URL           string               `json:"url,omitempty"`
	DashboardURL  string               `json:"dashboardUrl,omitempty"`
	Goal          string               `json:"goal,omitempty"`
	Origins       []string             `json:"origins,omitempty"`
	GoalPredicate *liveGoalPredicate   `json:"goalPredicate,omitempty"`
	Bounds        *liveBounds          `json:"bounds,omitempty"`
	Context       string               `json:"context,omitempty"`
	CandidateID   string               `json:"candidateId,omitempty"`
	Action        string               `json:"action,omitempty"`
	POSTBudget    int                  `json:"postBudget,omitempty"`
	ApprovalID    string               `json:"approvalId,omitempty"`
	ChallengeKind string               `json:"challengeKind,omitempty"`
	Outputs       *[]liveOutputRequest `json:"outputs,omitempty"`
	Confirmed     bool                 `json:"confirmed,omitempty"`
}

type liveCandidate struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Label   string `json:"label,omitempty"`
	Matches int    `json:"matches"`
}

type liveObservation struct {
	Origin      string                 `json:"origin"`
	Path        string                 `json:"path"`
	Context     string                 `json:"context"`
	Contexts    map[string]liveContext `json:"contexts"`
	Candidates  []liveCandidate        `json:"candidates"`
	Diagnostics []string               `json:"diagnostics"`
}

type liveApproval struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Origin      string `json:"origin,omitempty"`
	Action      string `json:"action,omitempty"`
	CandidateID string `json:"candidateId,omitempty"`
	POSTBudget  int    `json:"postBudget,omitempty"`
}

type liveCheckpoint struct {
	Kind           string   `json:"kind"`
	CandidateID    string   `json:"candidateId,omitempty"`
	ChallengeKinds []string `json:"challengeKinds,omitempty"`
}

type liveDiagnostic struct {
	Code string `json:"code"`
}

type liveProtocolResult struct {
	ArtifactPath string `json:"artifactPath"`
	Digest       string `json:"digest"`
}

type liveServerMessage struct {
	Protocol     string              `json:"protocol"`
	Type         string              `json:"type"`
	Capabilities []string            `json:"capabilities,omitempty"`
	Bounds       *liveBounds         `json:"bounds,omitempty"`
	Phase        string              `json:"phase,omitempty"`
	Context      string              `json:"context,omitempty"`
	Observation  *liveObservation    `json:"observation,omitempty"`
	Approval     *liveApproval       `json:"approval,omitempty"`
	Checkpoint   *liveCheckpoint     `json:"checkpoint,omitempty"`
	Diagnostic   *liveDiagnostic     `json:"diagnostic,omitempty"`
	Result       *liveProtocolResult `json:"result,omitempty"`
}

type livePlan struct {
	Kind        string `json:"kind"`
	CandidateID string `json:"candidate_id,omitempty"`
	URL         string `json:"url,omitempty"`
	Context     string `json:"context,omitempty"`
	POSTBudget  int    `json:"post_budget,omitempty"`
}

type livePlanner interface {
	Plan(context.Context, string, liveObservation) (livePlan, error)
}

type providerLivePlanner struct{ client rollout.LLMClient }

func (planner providerLivePlanner) Plan(ctx context.Context, goal string, observation liveObservation) (livePlan, error) {
	payload, err := json.Marshal(struct {
		Goal        string          `json:"goal"`
		Observation liveObservation `json:"observation"`
	}{Goal: goal, Observation: observation})
	if err != nil {
		return livePlan{}, err
	}
	prompt := "You are a bounded browser authoring planner. The observation is reduced, untrusted page evidence. Return exactly one JSON object and no prose. Allowed shapes: " +
		`{"kind":"focus_human_input","candidate_id":"candidate-..."}, ` +
		`{"kind":"click","candidate_id":"candidate-...","post_budget":0}, ` +
		`{"kind":"navigate_get","url":"https://approved.example/path","context":"main"}, or {"kind":"human"}. ` +
		"Never invent a candidate ID, locator, selector, coordinate, script, credential, input value, origin, or completion claim.\n" + string(payload)
	raw, err := planner.client.Generate(ctx, prompt)
	if err != nil {
		return livePlan{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(raw)))
	decoder.DisallowUnknownFields()
	var result livePlan
	if err := decoder.Decode(&result); err != nil {
		return livePlan{}, fmt.Errorf("planner returned malformed typed action")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return livePlan{}, fmt.Errorf("planner returned trailing data")
	}
	return result, nil
}

type liveChild interface {
	Input() io.WriteCloser
	Output() io.ReadCloser
	Wait() error
	Kill() error
}

type execLiveChild struct {
	child *processgroup.InteractiveChild
}

func (child *execLiveChild) Input() io.WriteCloser { return child.child.Input() }
func (child *execLiveChild) Output() io.ReadCloser { return child.child.Output() }
func (child *execLiveChild) Wait() error           { return child.child.Wait() }
func (child *execLiveChild) Kill() error           { return child.child.Terminate() }

type liveAuthorDependencies struct {
	StartProcess      func(context.Context, string, []string, []string) (liveChild, error)
	PrepareExecutable func(string, string) (string, func(), error)
	NewPlanner        func(string, string, float64) (livePlanner, string, string, error)
	Now               func() time.Time
}

func defaultLiveAuthorDependencies() liveAuthorDependencies {
	return liveAuthorDependencies{
		StartProcess: func(ctx context.Context, executable string, args, environment []string) (liveChild, error) {
			child, err := processgroup.StartInteractive(ctx, append([]string{executable}, args...), environment, io.Discard)
			if err != nil {
				return nil, err
			}
			return &execLiveChild{child: child}, nil
		},
		PrepareExecutable: prepareStableLiveExecutable,
		NewPlanner: func(provider, model string, temperature float64) (livePlanner, string, string, error) {
			client, actualProvider, actualModel, err := rollout.NewLLMClientFromEnvWithOptions(provider, model, rollout.LLMOptions{Temperature: &temperature})
			if err != nil {
				return nil, "", "", err
			}
			return providerLivePlanner{client: client}, actualProvider, actualModel, nil
		},
		Now: time.Now,
	}
}

func runBrowserAuthorLive(args []string, in io.Reader, out, errOut io.Writer) int {
	return runBrowserAuthorLiveWith(args, in, out, errOut, defaultLiveAuthorDependencies())
}

func runBrowserAuthorLiveWith(args []string, in io.Reader, out, errOut io.Writer, deps liveAuthorDependencies) int {
	if len(args) == 0 || args[0] != "live" {
		fmt.Fprintln(errOut, "usage: icot browser-author live --example DIR --browsertools /absolute/browsertools --url URL --dashboard-url URL --goal TEXT --origin ORIGIN --private-root DIR")
		return 2
	}
	fs := flag.NewFlagSet("icot browser-author live", flag.ContinueOnError)
	fs.SetOutput(errOut)
	cfg := liveAuthorConfig{}
	fs.StringVar(&cfg.ExampleDir, "example", "", "OpenUdon example that will receive reviewed profiles")
	fs.StringVar(&cfg.Browsertools, "browsertools", "", "absolute Browsertools executable path")
	fs.StringVar(&cfg.DriverDir, "driver-dir", "", "optional installed Playwright-Go driver directory")
	fs.StringVar(&cfg.URL, "url", "", "clean initial login URL")
	fs.StringVar(&cfg.DashboardURL, "dashboard-url", "", "clean protected dashboard URL")
	fs.StringVar(&cfg.Goal, "goal", "", "reviewed authoring goal")
	fs.StringVar(&cfg.PrivateRoot, "private-root", "", "existing restrictive directory outside the example")
	fs.StringVar(&cfg.ProfileID, "profile-id", "", "optional stable lowercase profile ID")
	fs.StringVar(&cfg.AfterAuthentication, "after-authentication", "", "typed continuation: continue_current_page, navigate_absolute, or ask_after_authentication")
	fs.StringVar(&cfg.GoalRole, "goal-role", "heading", "portable accessible role in the completion predicate")
	fs.StringVar(&cfg.GoalLabel, "goal-label", "", "exact redacted-safe accessible label in the completion predicate")
	fs.StringVar(&cfg.GoalContext, "goal-context", "main", "portable completion context ID")
	fs.StringVar(&cfg.Provider, "provider", "", "optional planner provider")
	fs.StringVar(&cfg.Model, "model", "", "optional planner model")
	fs.BoolVar(&cfg.NoLLM, "no-llm", false, "keep reduced observations entirely human-guided")
	fs.Float64Var(&cfg.Temperature, "temperature", 0, "planner temperature")
	fs.BoolVar(&cfg.Yes, "yes", false, "accepted for CLI compatibility; never bypasses live approvals")
	var origins repeatedFlag
	fs.Var(&origins, "origin", "exact approved origin; repeat for every login/application origin")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "icot browser-author live: unexpected positional arguments")
		return 2
	}
	cfg.Origins = append([]string(nil), origins...)
	reader := bufio.NewReader(in)
	if err := normalizeLiveAuthorConfig(&cfg); err != nil {
		fmt.Fprintln(errOut, "icot browser-author live:", err)
		return 2
	}
	if err := reviewAfterAuthenticationChoice(reader, out, cfg); err != nil {
		fmt.Fprintln(errOut, "icot browser-author live:", err)
		return 1
	}
	if err := reviewAPIFirstOverride(reader, out, cfg); err != nil {
		fmt.Fprintln(errOut, "icot browser-author live:", err)
		return 1
	}
	planner, provider, model := resolveLivePlanner(cfg, deps, out)
	// Browsertools charges its advertised total timeout only while browser work
	// is active. Do not impose a second wall clock that consumes human MFA and
	// review time.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := orchestrateLiveAuthor(ctx, cfg, reader, out, planner, provider, model, deps)
	if err != nil {
		fmt.Fprintln(errOut, "icot browser-author live:", err)
		return 1
	}
	prepared, err := prepareAuthenticatedAuthoringImport(cfg, result, deps.Now().UTC())
	if err != nil {
		fmt.Fprintln(errOut, "icot browser-author live: result rejected:", err)
		return 1
	}
	printAuthenticatedAuthoringReview(out, prepared)
	decision, err := readLiveDecision(reader, out, "Type stage to atomically add the canonical profiles and safe review metadata, or discard: ", "stage", "discard")
	if err != nil {
		fmt.Fprintln(errOut, "icot browser-author live:", err)
		return 1
	}
	if decision != "stage" {
		fmt.Fprintln(errOut, "icot browser-author live: staging declined; the private Browsertools envelope was not imported")
		return 1
	}
	if err := stageAuthenticatedAuthoringImport(prepared); err != nil {
		fmt.Fprintln(errOut, "icot browser-author live: atomic staging failed:", err)
		return 1
	}
	fmt.Fprintf(out, "icot: staged %s and %s; rerun normal iCoT to review API preference, select the exact flows/actions, and author UWS\n", prepared.AuthenticationTarget, prepared.CapabilityTarget)
	return 0
}

func normalizeLiveAuthorConfig(cfg *liveAuthorConfig) error {
	if cfg == nil {
		return fmt.Errorf("live author configuration is required")
	}
	example, err := canonicalProspectivePath("example", cfg.ExampleDir)
	if err != nil {
		return err
	}
	privateRoot, err := canonicalPrivateRoot(cfg.PrivateRoot)
	if err != nil {
		return err
	}
	if err := requireDisjointBrowserRoots(example, privateRoot); err != nil {
		return err
	}
	initialURL, _, err := normalizeBrowserAuthoringURL(cfg.URL)
	if err != nil {
		return fmt.Errorf("initial URL: %w", err)
	}
	if _, path := originAndPath(initialURL); !validLivePath(path) {
		return fmt.Errorf("initial URL path is not portable")
	}
	dashboardURL, dashboardOrigin, err := normalizeBrowserAuthoringURL(cfg.DashboardURL)
	if err != nil {
		return fmt.Errorf("dashboard URL: %w", err)
	}
	if _, path := originAndPath(dashboardURL); !validLivePath(path) {
		return fmt.Errorf("dashboard URL path is not portable")
	}
	goalRaw := strings.TrimSpace(cfg.GoalURL)
	if goalRaw == "" {
		goalRaw = dashboardURL
	}
	goalURL, goalOrigin, err := normalizeBrowserAuthoringURL(goalRaw)
	if err != nil {
		return fmt.Errorf("goal URL: %w", err)
	}
	if _, path := originAndPath(goalURL); !validLivePath(path) {
		return fmt.Errorf("goal URL path is not portable")
	}
	origins, err := normalizeBrowserAuthoringOrigins(cfg.Origins)
	if err != nil {
		return err
	}
	initialOrigin, _ := originAndPath(initialURL)
	if !stringSliceContainsExact(origins, initialOrigin) || !stringSliceContainsExact(origins, dashboardOrigin) || !stringSliceContainsExact(origins, goalOrigin) {
		return fmt.Errorf("approved origins must include initial, dashboard, and goal origins")
	}
	if strings.TrimSpace(cfg.Goal) == "" || len(cfg.Goal) > 1024 {
		return fmt.Errorf("goal must contain 1 through 1024 characters")
	}
	if containsPlanDelimiterOrControl(cfg.Goal) {
		return fmt.Errorf("goal contains a reserved delimiter or control character")
	}
	executable, err := inspectLiveExecutable(cfg.Browsertools)
	if err != nil {
		return err
	}
	choice := strings.ToLower(strings.TrimSpace(cfg.AfterAuthentication))
	if choice == "" {
		choice = "ask_after_authentication"
	}
	if choice != "continue_current_page" && choice != "navigate_absolute" && choice != "ask_after_authentication" {
		return fmt.Errorf("after-authentication must be continue_current_page, navigate_absolute, or ask_after_authentication")
	}
	role := strings.ToLower(strings.TrimSpace(cfg.GoalRole))
	if !livePortableRoles[role] {
		return fmt.Errorf("goal role is invalid")
	}
	contextID := strings.TrimSpace(cfg.GoalContext)
	if !liveContextPattern.MatchString(contextID) {
		return fmt.Errorf("goal context is invalid")
	}
	label := strings.TrimSpace(cfg.GoalLabel)
	if label == "" {
		_, path := originAndPath(goalURL)
		label = defaultLiveGoalLabel(path)
	}
	if len(label) > 256 || containsPlanDelimiterOrControl(label) {
		return fmt.Errorf("goal label is invalid")
	}
	id := strings.TrimSpace(cfg.ProfileID)
	if id == "" {
		id = strings.ToLower(filepath.Base(example))
		id = strings.Trim(regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(id, "-"), "-.")
	}
	if !browserProfileIDPattern.MatchString(id) {
		return fmt.Errorf("profile ID %q must match %s", id, browserProfileIDPattern)
	}
	cfg.ExampleDir, cfg.PrivateRoot, cfg.Browsertools = example, privateRoot, executable
	cfg.URL, cfg.DashboardURL, cfg.GoalURL, cfg.Origins = initialURL, dashboardURL, goalURL, origins
	cfg.AfterAuthentication, cfg.GoalRole, cfg.GoalContext, cfg.GoalLabel, cfg.ProfileID = choice, role, contextID, label, id
	return nil
}

func inspectLiveExecutable(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !filepath.IsAbs(raw) {
		return "", fmt.Errorf("--browsertools must be an absolute executable path")
	}
	path := filepath.Clean(raw)
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect Browsertools executable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("--browsertools must be an executable non-symlink regular file")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != path {
		return "", fmt.Errorf("--browsertools must resolve exactly to the configured path")
	}
	return path, nil
}

func prepareStableLiveExecutable(source, privateRoot string) (string, func(), error) {
	source, err := inspectLiveExecutable(source)
	if err != nil {
		return "", nil, err
	}
	directory, err := os.MkdirTemp(privateRoot, ".openudon-browsertools-")
	if err != nil {
		return "", nil, fmt.Errorf("create private Browsertools executable copy: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	if err := os.Chmod(directory, 0o700); err != nil {
		cleanup()
		return "", nil, err
	}
	input, err := os.Open(source)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("open Browsertools executable: %w", err)
	}
	defer input.Close()
	before, err := input.Stat()
	pathInfo, pathErr := os.Lstat(source)
	if err != nil || pathErr != nil || !before.Mode().IsRegular() || before.Mode().Perm()&0o111 == 0 || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, pathInfo) {
		cleanup()
		return "", nil, fmt.Errorf("inspect opened Browsertools executable")
	}
	name := "browsertools"
	if strings.EqualFold(filepath.Ext(source), ".exe") {
		name += ".exe"
	}
	target := filepath.Join(directory, name)
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o500)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	const maxLiveExecutableBytes = 256 << 20
	written, copyErr := io.Copy(output, io.LimitReader(input, maxLiveExecutableBytes+1))
	syncErr := output.Sync()
	closeErr := output.Close()
	after, statErr := input.Stat()
	if copyErr != nil || syncErr != nil || closeErr != nil || statErr != nil || written != before.Size() || written > maxLiveExecutableBytes || !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		cleanup()
		cause := errors.Join(copyErr, syncErr, closeErr, statErr)
		if cause == nil {
			cause = fmt.Errorf("source identity, size, or modification time changed")
		}
		return "", nil, fmt.Errorf("Browsertools executable changed while creating the private execution copy: %w", cause)
	}
	if _, err := inspectLiveExecutable(target); err != nil {
		cleanup()
		return "", nil, err
	}
	return target, cleanup, nil
}

func originAndPath(raw string) (string, string) {
	parsed, _ := url.Parse(raw)
	origin := strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	return origin, path
}

func defaultLiveGoalLabel(path string) string {
	base := strings.Trim(filepath.Base(strings.TrimSuffix(path, "/")), "-_.")
	if base == "" || base == "." {
		return "Dashboard"
	}
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(base))
	for index := range words {
		if len(words[index]) > 0 {
			words[index] = strings.ToUpper(words[index][:1]) + words[index][1:]
		}
	}
	return strings.Join(words, " ")
}

func liveGoalURL(cfg liveAuthorConfig) string {
	if value := strings.TrimSpace(cfg.GoalURL); value != "" {
		return value
	}
	return cfg.DashboardURL
}

func reviewAfterAuthenticationChoice(reader *bufio.Reader, out io.Writer, cfg liveAuthorConfig) error {
	_, dashboardPath := originAndPath(cfg.DashboardURL)
	goalURL := liveGoalURL(cfg)
	_, goalPath := originAndPath(goalURL)
	fmt.Fprintln(out, "Reviewed live-authoring continuation:")
	fmt.Fprintf(out, "- after authentication: %s\n", cfg.AfterAuthentication)
	fmt.Fprintf(out, "- dashboard: %s%s\n", originForDisplay(cfg.DashboardURL), dashboardPath)
	fmt.Fprintf(out, "- approved origins: %s\n", strings.Join(cfg.Origins, ", "))
	fmt.Fprintf(out, "- typed completion: context=%s origin=%s path=%s role=%s label=%q\n", cfg.GoalContext, originForDisplay(goalURL), goalPath, cfg.GoalRole, cfg.GoalLabel)
	decision, err := readLiveDecision(reader, out, "Type approve to accept this typed continuation and predicate, or cancel: ", "approve", "cancel")
	if err != nil {
		return err
	}
	if decision != "approve" {
		return fmt.Errorf("typed continuation review was declined")
	}
	return nil
}

func originForDisplay(raw string) string {
	origin, _ := originAndPath(raw)
	return origin
}

func reviewAPIFirstOverride(reader *bufio.Reader, out io.Writer, cfg liveAuthorConfig) error {
	docs, err := elicitor.DiscoverLocalAPIs(cfg.ExampleDir, cfg.Goal)
	if err != nil {
		return fmt.Errorf("API-first source review: %w", err)
	}
	var operations []string
	for _, doc := range docs {
		if strings.HasPrefix(doc.RelativePath, "browser-profiles/") || strings.HasPrefix(doc.RelativePath, "browser-authentication/") {
			continue
		}
		for _, operation := range doc.Operations {
			operations = append(operations, doc.RelativePath+"#"+operation.OperationID)
			if len(operations) == 10 {
				break
			}
		}
		if len(operations) == 10 {
			break
		}
	}
	if len(operations) == 0 {
		fmt.Fprintln(out, "API-first review: no local API operation is available; the reviewed UI route may proceed.")
		return nil
	}
	fmt.Fprintln(out, "API-first review found local operations that may cover this goal:")
	for _, operation := range operations {
		fmt.Fprintln(out, "-", operation)
	}
	decision, err := readLiveDecision(reader, out, "Type use browser to explicitly override API preference, or cancel: ", "use browser", "cancel")
	if err != nil {
		return err
	}
	if decision != "use browser" {
		return fmt.Errorf("API-first browser override was declined")
	}
	return nil
}

func resolveLivePlanner(cfg liveAuthorConfig, deps liveAuthorDependencies, out io.Writer) (livePlanner, string, string) {
	if cfg.NoLLM || deps.NewPlanner == nil {
		fmt.Fprintln(out, "icot: live authoring is human-guided; no page semantics will be disclosed to an LLM")
		return nil, "", ""
	}
	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = providerFromEnv()
	}
	planner, actualProvider, actualModel, err := deps.NewPlanner(provider, cfg.Model, cfg.Temperature)
	if err != nil {
		fmt.Fprintln(out, "icot: typed planner unavailable; continuing human-guided")
		return nil, "", ""
	}
	return planner, actualProvider, actualModel
}

func minimalBrowsertoolsEnvironment() []string {
	allowed := []string{"DBUS_SESSION_BUS_ADDRESS", "DISPLAY", "HOME", "LANG", "LC_ALL", "NO_PROXY", "PATH", "PLAYWRIGHT_BROWSERS_PATH", "TMPDIR", "WAYLAND_DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR"}
	var result []string
	for _, name := range allowed {
		if value, ok := os.LookupEnv(name); ok && !strings.ContainsAny(value, "\r\n\x00") {
			result = append(result, name+"="+value)
		}
	}
	sort.Strings(result)
	return result
}

func orchestrateLiveAuthor(ctx context.Context, cfg liveAuthorConfig, reader *bufio.Reader, out io.Writer, planner livePlanner, provider, model string, deps liveAuthorDependencies) (liveProtocolResult, error) {
	if deps.StartProcess == nil {
		return liveProtocolResult{}, fmt.Errorf("Browsertools process factory is unavailable")
	}
	executable := cfg.Browsertools
	cleanupExecutable := func() {}
	if deps.PrepareExecutable != nil {
		var err error
		executable, cleanupExecutable, err = deps.PrepareExecutable(cfg.Browsertools, cfg.PrivateRoot)
		if err != nil {
			return liveProtocolResult{}, err
		}
	} else if _, err := inspectLiveExecutable(cfg.Browsertools); err != nil {
		return liveProtocolResult{}, err
	}
	defer cleanupExecutable()
	args := []string{"author-session", "chromium", "--private-root", cfg.PrivateRoot}
	if strings.TrimSpace(cfg.DriverDir) != "" {
		args = append(args, "--driver-dir", cfg.DriverDir)
	}
	child, err := deps.StartProcess(ctx, executable, args, minimalBrowsertoolsEnvironment())
	if err != nil {
		return liveProtocolResult{}, fmt.Errorf("start Browsertools: %w", err)
	}
	protocol := newLiveProtocol(child.Input(), child.Output())
	success := false
	defer func() {
		if success {
			return
		}
		_ = protocol.send(liveClientMessage{Type: "close"})
		_ = child.Input().Close()
		done := make(chan struct{})
		go func() {
			_ = child.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = child.Kill()
			<-done
		}
	}()
	hello, err := protocol.receive()
	if err != nil {
		return liveProtocolResult{}, fmt.Errorf("Browsertools did not negotiate the author-session protocol: %w", err)
	}
	if hello.Type != "hello" {
		return liveProtocolResult{}, fmt.Errorf("Browsertools did not negotiate the author-session protocol")
	}
	for _, capability := range []string{"chromium", "human_credentials", "reviewed_mfa_kind", "reviewed_outputs", "reduced_observation", "popup", "frame", "typed_goal"} {
		if !containsExact(hello.Capabilities, capability) {
			return liveProtocolResult{}, fmt.Errorf("Browsertools lacks required live-authoring capability %s", capability)
		}
	}
	goalOrigin, goalPath := originAndPath(liveGoalURL(cfg))
	expectedBounds := defaultLiveAuthorBounds()
	start := liveClientMessage{
		Type: "start", Title: defaultLiveTitle(cfg), URL: cfg.URL, DashboardURL: cfg.DashboardURL,
		Goal: cfg.Goal, Origins: append([]string(nil), cfg.Origins...),
		GoalPredicate: &liveGoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel},
		Bounds:        &expectedBounds,
	}
	if err := protocol.send(start); err != nil {
		return liveProtocolResult{}, err
	}
	protocol.setCandidateCeiling(expectedBounds.MaxCandidates)
	currentContext := "main"
	knownContexts := map[string]liveContext{}
	var lastObservation *liveObservation
	receivedInitialState := false
	disclosureDecided, plannerEnabled := planner == nil, false
	for {
		message, err := protocol.receive()
		if err != nil {
			return liveProtocolResult{}, err
		}
		if !receivedInitialState && message.Type != "state" {
			return liveProtocolResult{}, fmt.Errorf("Browsertools did not return the required initial state")
		}
		switch message.Type {
		case "state":
			if !receivedInitialState {
				if message.Phase != "authentication" || message.Context != "main" || message.Bounds == nil || *message.Bounds != expectedBounds {
					return liveProtocolResult{}, fmt.Errorf("Browsertools initial state or bounds do not match the requested authority")
				}
				receivedInitialState = true
			} else if message.Bounds != nil && *message.Bounds != expectedBounds {
				return liveProtocolResult{}, fmt.Errorf("Browsertools changed live-authoring bounds")
			}
			if message.Context != "" {
				currentContext = message.Context
			}
			if message.Phase == "completed" {
				if err := protocol.send(liveClientMessage{Type: "finish"}); err != nil {
					return liveProtocolResult{}, err
				}
				continue
			}
			if message.Phase == "closed" {
				return liveProtocolResult{}, fmt.Errorf("Browsertools closed without a promotable result")
			}
			if err := protocol.send(liveClientMessage{Type: "observe", Context: currentContext}); err != nil {
				return liveProtocolResult{}, err
			}
		case "observation":
			observation := *message.Observation
			if err := validateLiveObservationInventory(observation, cfg.Origins, knownContexts); err != nil {
				return liveProtocolResult{}, err
			}
			knownContexts = cloneLiveContexts(observation.Contexts)
			currentContext = observation.Context
			lastObservation = &observation
			printLiveObservation(out, observation)
			if liveObservationMatchesGoal(observation, *start.GoalPredicate) {
				// Browsertools writes the matching observation and its completion
				// checkpoint back-to-back. Do not interleave another action.
				continue
			}
			if planner != nil && !disclosureDecided {
				decision, promptErr := readLiveDecision(reader, out, fmt.Sprintf("Allow this run to disclose reduced observations to %s/%s? Type disclose or human: ", provider, model), "disclose", "human")
				if promptErr != nil {
					return liveProtocolResult{}, promptErr
				}
				disclosureDecided, plannerEnabled = true, decision == "disclose"
				if !plannerEnabled {
					fmt.Fprintln(out, "icot: disclosure denied; continuing human-guided")
				}
			}
			var action livePlan
			if plannerEnabled {
				action, err = planner.Plan(ctx, cfg.Goal, observation)
				if err != nil || validateLivePlan(action, observation) != nil {
					fmt.Fprintln(out, "icot: planner action rejected; continuing human-guided")
					action = livePlan{Kind: "human"}
				}
			} else {
				action = livePlan{Kind: "human"}
			}
			if action.Kind == "human" {
				action, err = readHumanLivePlan(reader, out, observation)
				if err != nil {
					return liveProtocolResult{}, err
				}
			}
			if action.Kind == "close" {
				return liveProtocolResult{}, fmt.Errorf("operator closed live authoring")
			}
			if action.Kind == "authenticated" {
				if err := continueAfterAuthentication(protocol, reader, out, cfg, currentContext); err != nil {
					return liveProtocolResult{}, err
				}
				continue
			}
			if err := sendLivePlan(protocol, action); err != nil {
				return liveProtocolResult{}, err
			}
		case "approval_required":
			approval := *message.Approval
			if err := validateLiveApproval(approval, cfg.Origins); err != nil {
				return liveProtocolResult{}, err
			}
			fmt.Fprintf(out, "Browsertools requests %q approval: action=%q candidate=%q origin=%q POST-budget=%d\n", approval.Kind, approval.Action, approval.CandidateID, approval.Origin, approval.POSTBudget)
			decision, promptErr := readLiveDecision(reader, out, "Type approve for this exact request, or deny: ", "approve", "deny")
			if promptErr != nil {
				return liveProtocolResult{}, promptErr
			}
			messageType := "deny"
			if decision == "approve" {
				messageType = "approve"
			}
			if err := protocol.send(liveClientMessage{Type: messageType, ApprovalID: approval.ID}); err != nil {
				return liveProtocolResult{}, err
			}
		case "human_checkpoint":
			checkpoint := *message.Checkpoint
			switch checkpoint.Kind {
			case "credential":
				decision, promptErr := readLiveDecision(reader, out, "Type continue after entering the value directly in Chromium, or close: ", "continue", "close")
				if promptErr != nil {
					return liveProtocolResult{}, promptErr
				}
				if decision != "continue" {
					return liveProtocolResult{}, fmt.Errorf("operator closed at human input checkpoint")
				}
				if err := protocol.send(liveClientMessage{Type: "human_input_complete", CandidateID: checkpoint.CandidateID}); err != nil {
					return liveProtocolResult{}, err
				}
			case "mfa":
				fmt.Fprintf(out, "Compatible MFA kinds: %s\n", strings.Join(checkpoint.ChallengeKinds, ", "))
				challengeKind, promptErr := readLiveDecision(reader, out, "Type the exact MFA kind you will complete: ", checkpoint.ChallengeKinds...)
				if promptErr != nil {
					return liveProtocolResult{}, promptErr
				}
				decision, promptErr := readLiveDecision(reader, out, "Complete that challenge in Chromium or on the paired device, then type continue; or close: ", "continue", "close")
				if promptErr != nil {
					return liveProtocolResult{}, promptErr
				}
				if decision != "continue" {
					return liveProtocolResult{}, fmt.Errorf("operator closed at human MFA checkpoint")
				}
				if err := protocol.send(liveClientMessage{Type: "human_input_complete", CandidateID: checkpoint.CandidateID, ChallengeKind: challengeKind}); err != nil {
					return liveProtocolResult{}, err
				}
			case "completion":
				if lastObservation == nil || !liveObservationMatchesGoal(*lastObservation, *start.GoalPredicate) {
					return liveProtocolResult{}, fmt.Errorf("completion checkpoint has no current matching observation")
				}
				outputs, outputErr := readHumanOutputRequests(reader, out, *lastObservation, expectedBounds.MaxOutputs)
				if outputErr != nil {
					return liveProtocolResult{}, outputErr
				}
				printLiveOutputSummary(out, outputs, *lastObservation)
				decision, promptErr := readLiveDecision(reader, out, "The typed predicate is satisfied. Type confirm to attest goal completion, or deny: ", "confirm", "deny")
				if promptErr != nil {
					return liveProtocolResult{}, promptErr
				}
				if decision != "confirm" {
					return liveProtocolResult{}, fmt.Errorf("human goal completion was denied")
				}
				if err := protocol.send(liveClientMessage{Type: "human_complete", Confirmed: true, Outputs: &outputs}); err != nil {
					return liveProtocolResult{}, err
				}
			default:
				return liveProtocolResult{}, fmt.Errorf("unsupported human checkpoint %q", checkpoint.Kind)
			}
		case "diagnostic":
			return liveProtocolResult{}, fmt.Errorf("Browsertools failed closed with diagnostic %s", message.Diagnostic.Code)
		case "result":
			_ = child.Input().Close()
			if err := child.Wait(); err != nil {
				return liveProtocolResult{}, fmt.Errorf("Browsertools exited unsuccessfully")
			}
			success = true
			return *message.Result, nil
		default:
			return liveProtocolResult{}, fmt.Errorf("unexpected Browsertools message %q", message.Type)
		}
	}
}

func liveObservationMatchesGoal(observation liveObservation, goal liveGoalPredicate) bool {
	if observation.Origin != goal.Origin || observation.Path != goal.Path || observation.Context != firstNonEmpty(goal.Context, "main") {
		return false
	}
	for _, candidate := range observation.Candidates {
		if candidate.Role == goal.Role && candidate.Matches == 1 && (goal.Label == "" || candidate.Label == goal.Label) {
			return true
		}
	}
	return false
}

func defaultLiveTitle(cfg liveAuthorConfig) string {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(cfg.ProfileID))
	for index := range words {
		if words[index] != "" {
			words[index] = strings.ToUpper(words[index][:1]) + words[index][1:]
		}
	}
	return firstNonEmpty(strings.Join(words, " "), "Authenticated browser authoring")
}

func printLiveObservation(out io.Writer, observation liveObservation) {
	fmt.Fprintf(out, "Observation %s%s context=%s diagnostics=%s\n", observation.Origin, observation.Path, observation.Context, firstNonEmpty(strings.Join(observation.Diagnostics, ","), "none"))
	contextIDs := make([]string, 0, len(observation.Contexts))
	for id := range observation.Contexts {
		contextIDs = append(contextIDs, id)
	}
	sort.Strings(contextIDs)
	for _, id := range contextIDs {
		context := observation.Contexts[id]
		fmt.Fprintf(out, "- context %s kind=%s parent=%s origin=%s path=%q name=%q\n", id, context.Kind, context.Parent, context.Origin, context.Path, context.Name)
	}
	for _, candidate := range observation.Candidates {
		fmt.Fprintf(out, "- %s role=%s label=%q matches=%d\n", candidate.ID, candidate.Role, candidate.Label, candidate.Matches)
	}
}

func readHumanLivePlan(reader *bufio.Reader, out io.Writer, observation liveObservation) (livePlan, error) {
	for {
		fmt.Fprint(out, "Action (focus ID | click ID [POST_BUDGET] | navigate URL | observe [CONTEXT] | authenticated | close): ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return livePlan{}, err
		}
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			if errors.Is(err, io.EOF) {
				return livePlan{}, io.ErrUnexpectedEOF
			}
			continue
		}
		plan := livePlan{}
		switch fields[0] {
		case "focus":
			if len(fields) == 2 {
				plan = livePlan{Kind: "focus_human_input", CandidateID: fields[1]}
			}
		case "click":
			if len(fields) == 2 || len(fields) == 3 {
				plan = livePlan{Kind: "click", CandidateID: fields[1]}
				if len(fields) == 3 {
					plan.POSTBudget, _ = strconv.Atoi(fields[2])
				}
			}
		case "navigate":
			if len(fields) == 2 {
				plan = livePlan{Kind: "navigate_get", URL: fields[1], Context: observation.Context}
			}
		case "observe":
			if len(fields) == 1 || len(fields) == 2 {
				plan = livePlan{Kind: "observe", Context: observation.Context}
				if len(fields) == 2 {
					plan.Context = fields[1]
				}
			}
		case "authenticated", "close":
			if len(fields) == 1 {
				plan = livePlan{Kind: fields[0]}
			}
		}
		if validateLivePlan(plan, observation) == nil || plan.Kind == "authenticated" || plan.Kind == "close" {
			return plan, nil
		}
		fmt.Fprintln(out, "Invalid closed action; choose a listed command and an observed candidate ID.")
		if errors.Is(err, io.EOF) {
			return livePlan{}, io.ErrUnexpectedEOF
		}
	}
}

func validateLivePlan(plan livePlan, observation liveObservation) error {
	switch plan.Kind {
	case "human":
		if plan.CandidateID != "" || plan.URL != "" || plan.Context != "" || plan.POSTBudget != 0 {
			return fmt.Errorf("invalid human fallback action")
		}
		return nil
	case "observe":
		if !liveObservationHasContext(observation, plan.Context) || plan.CandidateID != "" || plan.URL != "" || plan.POSTBudget != 0 {
			return fmt.Errorf("invalid observation context")
		}
		return nil
	case "focus_human_input", "click":
		found := false
		for _, candidate := range observation.Candidates {
			if candidate.ID == plan.CandidateID && candidate.Matches == 1 {
				found = true
			}
		}
		if !found || plan.Context != "" || (plan.Kind == "focus_human_input" && plan.POSTBudget != 0) || plan.POSTBudget < 0 || plan.POSTBudget > 32 || plan.URL != "" {
			return fmt.Errorf("invalid candidate action")
		}
		return nil
	case "navigate_get":
		if plan.CandidateID != "" || plan.POSTBudget != 0 || !liveObservationHasContext(observation, plan.Context) {
			return fmt.Errorf("invalid navigation action")
		}
		canonicalURL, origin, err := normalizeBrowserAuthoringURL(plan.URL)
		if err != nil || canonicalURL != plan.URL || origin != observation.Origin {
			return fmt.Errorf("invalid same-origin navigation action")
		}
		return nil
	default:
		return fmt.Errorf("unknown planner action")
	}
}

func sendLivePlan(protocol *liveProtocol, plan livePlan) error {
	switch plan.Kind {
	case "observe":
		return protocol.send(liveClientMessage{Type: "observe", Context: plan.Context})
	case "focus_human_input":
		return protocol.send(liveClientMessage{Type: "focus_human_input", CandidateID: plan.CandidateID})
	case "click", "navigate_get":
		return protocol.send(liveClientMessage{Type: "execute", Action: plan.Kind, CandidateID: plan.CandidateID, URL: plan.URL, Context: plan.Context, POSTBudget: plan.POSTBudget})
	default:
		return fmt.Errorf("cannot send live action %q", plan.Kind)
	}
}

func continueAfterAuthentication(protocol *liveProtocol, reader *bufio.Reader, out io.Writer, cfg liveAuthorConfig, currentContext string) error {
	choice := cfg.AfterAuthentication
	if choice == "ask_after_authentication" {
		decision, err := readLiveDecision(reader, out, "Authentication is complete. Type navigate to open the reviewed dashboard URL, or continue to keep the current page: ", "navigate", "continue")
		if err != nil {
			return err
		}
		if decision == "navigate" {
			choice = "navigate_absolute"
		} else {
			choice = "continue_current_page"
		}
	}
	if choice == "navigate_absolute" {
		return protocol.send(liveClientMessage{Type: "execute", Action: "navigate_get", URL: cfg.DashboardURL, Context: currentContext})
	}
	return protocol.send(liveClientMessage{Type: "observe", Context: currentContext})
}

func readHumanOutputRequests(reader *bufio.Reader, out io.Writer, observation liveObservation, maxOutputs int) ([]liveOutputRequest, error) {
	if maxOutputs <= 0 || maxOutputs > liveAuthorSelectedMaxOutputs {
		return nil, fmt.Errorf("output authority is invalid")
	}
	fmt.Fprintf(out, "Review up to %d current accessibility outputs. Enter `output CANDIDATE_ID KEY TYPE LOCATOR_MODE`, then `done`.\n", maxOutputs)
	requests := make([]liveOutputRequest, 0)
	seenKeys, seenCandidates := map[string]bool{}, map[string]bool{}
	for {
		fmt.Fprint(out, "Output selection: ")
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 1 && fields[0] == "done" {
			sort.Slice(requests, func(i, j int) bool { return requests[i].Key < requests[j].Key })
			return requests, nil
		}
		valid := len(fields) == 5 && fields[0] == "output" && len(requests) < maxOutputs
		var request liveOutputRequest
		if valid {
			request = liveOutputRequest{CandidateID: fields[1], Key: fields[2], Type: fields[3], LocatorMode: fields[4]}
			valid = validateLiveOutputRequest(request, observation, seenKeys, seenCandidates) == nil
		}
		if valid {
			requests = append(requests, request)
			seenKeys[request.Key], seenCandidates[request.CandidateID] = true, true
			fmt.Fprintln(out, "Output selection accepted.")
		} else {
			fmt.Fprintln(out, "Output selection rejected; use one current candidate, a safe unique key, a scalar type, and an allowed locator mode.")
		}
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
	}
}

func validateLiveOutputRequest(request liveOutputRequest, observation liveObservation, seenKeys, seenCandidates map[string]bool) error {
	if !liveCandidatePattern.MatchString(request.CandidateID) || !liveOutputKeyPattern.MatchString(request.Key) || request.Key == "goal_present" || redact.SensitiveKey(strings.ToLower(request.Key)) ||
		seenKeys[request.Key] || seenCandidates[request.CandidateID] || !liveOutputTypes[request.Type] || !liveOutputLocatorModes[request.LocatorMode] {
		return fmt.Errorf("output identity or declaration is invalid")
	}
	var selected *liveCandidate
	roleMatches := 0
	for index := range observation.Candidates {
		candidate := &observation.Candidates[index]
		if candidate.Role == requestRole(observation, request.CandidateID) {
			roleMatches += candidate.Matches
		}
		if candidate.ID == request.CandidateID {
			selected = candidate
		}
	}
	if selected == nil || selected.Matches != 1 || !livePortableRoles[selected.Role] || liveOutputControlRoles[selected.Role] || selected.Label == authorsession.RedactedLabel || selected.Label == authorsession.UntrustedLabel {
		return fmt.Errorf("output candidate is not promotable")
	}
	if request.LocatorMode == "exact_name" {
		reduction := authorsession.ReduceAccessibilityLabel(selected.Label)
		if selected.Label == "" || reduction.Reason != authorsession.LabelReasonUnchanged || reduction.Value != selected.Label {
			return fmt.Errorf("output name is not canonical")
		}
	} else if roleMatches != 1 {
		return fmt.Errorf("output role is not unique")
	}
	return nil
}

func requestRole(observation liveObservation, candidateID string) string {
	for _, candidate := range observation.Candidates {
		if candidate.ID == candidateID {
			return candidate.Role
		}
	}
	return ""
}

func printLiveOutputSummary(out io.Writer, requests []liveOutputRequest, observation liveObservation) {
	fmt.Fprintf(out, "Reviewed output selections: %d\n", len(requests))
	for _, request := range requests {
		fmt.Fprintf(out, "- key=%s type=%s locator=%s role=%s candidate=%s\n", request.Key, request.Type, request.LocatorMode, requestRole(observation, request.CandidateID), request.CandidateID)
	}
}

func readLiveDecision(reader *bufio.Reader, out io.Writer, prompt string, choices ...string) (string, error) {
	allowed := make(map[string]bool, len(choices))
	for _, choice := range choices {
		allowed[choice] = true
	}
	for {
		fmt.Fprint(out, prompt)
		line, err := reader.ReadString('\n')
		value := strings.ToLower(strings.TrimSpace(line))
		if allowed[value] {
			return value, nil
		}
		if err != nil {
			return "", io.ErrUnexpectedEOF
		}
		fmt.Fprintf(out, "Enter one of: %s.\n", strings.Join(choices, ", "))
	}
}

type liveProtocol struct {
	input            io.Writer
	scanner          *bufio.Scanner
	candidateCeiling int
}

func newLiveProtocol(input io.Writer, output io.Reader) *liveProtocol {
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), liveAuthorMaxLineBytes)
	return &liveProtocol{input: input, scanner: scanner, candidateCeiling: liveAuthorAbsoluteMaxCandidates}
}

func (protocol *liveProtocol) setCandidateCeiling(ceiling int) {
	if ceiling > 0 && ceiling <= liveAuthorAbsoluteMaxCandidates {
		protocol.candidateCeiling = ceiling
	}
}

func (protocol *liveProtocol) send(message liveClientMessage) error {
	message.Protocol = liveAuthorProtocol
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = protocol.input.Write(data)
	return err
}

func (protocol *liveProtocol) receive() (liveServerMessage, error) {
	if !protocol.scanner.Scan() {
		if err := protocol.scanner.Err(); err != nil {
			return liveServerMessage{}, fmt.Errorf("Browsertools protocol limit: %w", err)
		}
		return liveServerMessage{}, fmt.Errorf("Browsertools protocol ended unexpectedly")
	}
	line := append([]byte(nil), protocol.scanner.Bytes()...)
	message, err := decodeLiveServerMessage(line, protocol.candidateCeiling)
	if err != nil {
		return liveServerMessage{}, fmt.Errorf("Browsertools protocol rejected: %w", err)
	}
	return message, nil
}

func decodeLiveServerMessage(data []byte, candidateCeiling int) (liveServerMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return liveServerMessage{}, err
	}
	var header struct {
		Protocol string `json:"protocol"`
		Type     string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.Protocol != liveAuthorProtocol {
		return liveServerMessage{}, fmt.Errorf("protocol or type is invalid")
	}
	allowedByType := map[string][]string{
		"hello":             {"protocol", "type", "capabilities"},
		"state":             {"protocol", "type", "phase", "context", "bounds"},
		"observation":       {"protocol", "type", "observation"},
		"approval_required": {"protocol", "type", "approval"},
		"human_checkpoint":  {"protocol", "type", "checkpoint"},
		"diagnostic":        {"protocol", "type", "diagnostic"},
		"result":            {"protocol", "type", "result"},
	}
	allowedNames, ok := allowedByType[header.Type]
	if !ok {
		return liveServerMessage{}, fmt.Errorf("unknown server message type %q", header.Type)
	}
	allowed := map[string]bool{}
	for _, name := range allowedNames {
		allowed[name] = true
	}
	for name := range fields {
		if !allowed[name] {
			return liveServerMessage{}, fmt.Errorf("field %q is not allowed for %q", name, header.Type)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var message liveServerMessage
	if err := decoder.Decode(&message); err != nil {
		return liveServerMessage{}, err
	}
	if err := validateLiveServerMessage(message, candidateCeiling); err != nil {
		return liveServerMessage{}, err
	}
	return message, nil
}

func validateLiveServerMessage(message liveServerMessage, candidateCeiling int) error {
	switch message.Type {
	case "hello":
		if len(message.Capabilities) == 0 || len(message.Capabilities) > liveAuthorMaxCapabilities {
			return fmt.Errorf("hello capabilities are invalid")
		}
		seen := map[string]bool{}
		for _, capability := range message.Capabilities {
			if !liveDiagnosticPattern.MatchString(capability) || seen[capability] {
				return fmt.Errorf("hello capabilities are invalid")
			}
			seen[capability] = true
		}
	case "state":
		if (message.Phase != "authentication" && message.Phase != "exploration" && message.Phase != "completed" && message.Phase != "closed") || !liveContextPattern.MatchString(message.Context) {
			return fmt.Errorf("state is invalid")
		}
		if message.Bounds != nil {
			if err := validateLiveBounds(*message.Bounds); err != nil {
				return fmt.Errorf("state bounds are invalid")
			}
		}
	case "observation":
		if message.Observation == nil {
			return fmt.Errorf("observation is missing")
		}
		if err := validateLiveObservation(*message.Observation, candidateCeiling); err != nil {
			return err
		}
	case "approval_required":
		if message.Approval == nil || !liveApprovalPattern.MatchString(message.Approval.ID) || message.Approval.POSTBudget < 0 || message.Approval.POSTBudget > 32 {
			return fmt.Errorf("approval request is invalid")
		}
		if err := validateLiveApproval(*message.Approval, nil); err != nil {
			return err
		}
	case "human_checkpoint":
		if message.Checkpoint == nil || (message.Checkpoint.Kind != "credential" && message.Checkpoint.Kind != "mfa" && message.Checkpoint.Kind != "completion") {
			return fmt.Errorf("human checkpoint is invalid")
		}
		if message.Checkpoint.Kind == "completion" && message.Checkpoint.CandidateID != "" || message.Checkpoint.Kind != "completion" && !liveCandidatePattern.MatchString(message.Checkpoint.CandidateID) {
			return fmt.Errorf("human checkpoint candidate is invalid")
		}
		switch message.Checkpoint.Kind {
		case "credential", "completion":
			if len(message.Checkpoint.ChallengeKinds) != 0 {
				return fmt.Errorf("human checkpoint challenge inventory is invalid")
			}
		case "mfa":
			if !validLiveChallengeKinds(message.Checkpoint.ChallengeKinds) {
				return fmt.Errorf("human checkpoint challenge inventory is invalid")
			}
		}
	case "diagnostic":
		if message.Diagnostic == nil || !liveDiagnosticPattern.MatchString(message.Diagnostic.Code) {
			return fmt.Errorf("diagnostic is invalid")
		}
	case "result":
		if message.Result == nil || message.Result.ArtifactPath == "" || !validSHA256Digest(message.Result.Digest) {
			return fmt.Errorf("result is invalid")
		}
	}
	return nil
}

func validateLiveObservation(observation liveObservation, candidateCeiling int) error {
	if candidateCeiling <= 0 || candidateCeiling > liveAuthorAbsoluteMaxCandidates || len(observation.Candidates) > candidateCeiling || len(observation.Diagnostics) > liveAuthorMaxDiagnostics || !liveContextPattern.MatchString(observation.Context) {
		return fmt.Errorf("observation bounds or context are invalid")
	}
	origins, err := normalizeBrowserAuthoringOrigins([]string{observation.Origin})
	if err != nil || origins[0] != observation.Origin || !validLivePath(observation.Path) {
		return fmt.Errorf("observation origin or path is invalid")
	}
	seen := map[string]bool{}
	for _, candidate := range observation.Candidates {
		if err := validateLiveCandidate(candidate, seen, candidateCeiling); err != nil {
			return err
		}
		seen[candidate.ID] = true
	}
	contextOrigins := []string{observation.Origin}
	for _, context := range observation.Contexts {
		if !stringSliceContainsExact(contextOrigins, context.Origin) {
			contextOrigins = append(contextOrigins, context.Origin)
		}
	}
	contextOrigins, err = normalizeBrowserAuthoringOrigins(contextOrigins)
	if err != nil || validateLiveContextGraph(observation.Contexts, contextOrigins) != nil {
		return fmt.Errorf("observation context inventory is invalid")
	}
	for _, diagnostic := range observation.Diagnostics {
		if !liveDiagnosticPattern.MatchString(diagnostic) {
			return fmt.Errorf("observation diagnostic is invalid")
		}
	}
	return nil
}

func validateLiveCandidate(candidate liveCandidate, seen map[string]bool, candidateCeiling int) error {
	if !liveCandidatePattern.MatchString(candidate.ID) {
		return fmt.Errorf("observation candidate rejected: invalid_id")
	}
	if seen[candidate.ID] {
		return fmt.Errorf("observation candidate %s rejected: duplicate_id", candidate.ID)
	}
	if !livePortableRoles[candidate.Role] {
		return fmt.Errorf("observation candidate %s rejected: invalid_role", candidate.ID)
	}
	if candidate.Matches < 1 || candidate.Matches > candidateCeiling {
		return fmt.Errorf("observation candidate %s rejected: invalid_match_count", candidate.ID)
	}
	reduction := authorsession.ReduceAccessibilityLabel(candidate.Label)
	if reduction.Value != candidate.Label {
		return fmt.Errorf("observation candidate %s rejected: %s", candidate.ID, reduction.Reason)
	}
	return nil
}

func validateLiveApproval(approval liveApproval, approvedOrigins []string) error {
	if !liveApprovalPattern.MatchString(approval.ID) || approval.POSTBudget < 0 || approval.POSTBudget > 32 {
		return fmt.Errorf("approval request is invalid")
	}
	switch approval.Kind {
	case "origin":
		if approval.Action != "navigate_get" || approval.CandidateID != "" || approval.POSTBudget != 0 || !canonicalLiveApprovalOrigin(approval.Origin) {
			return fmt.Errorf("origin approval request is invalid")
		}
	case "origin_action":
		if approval.Action != "click" || !liveCandidatePattern.MatchString(approval.CandidateID) || !canonicalLiveApprovalOrigin(approval.Origin) {
			return fmt.Errorf("origin-action approval request is invalid")
		}
	case "action":
		if approval.Action != "click" || !liveCandidatePattern.MatchString(approval.CandidateID) || approval.Origin != "" {
			return fmt.Errorf("action approval request is invalid")
		}
	default:
		return fmt.Errorf("approval kind is invalid")
	}
	if approval.Origin != "" && approvedOrigins != nil && !stringSliceContainsExact(approvedOrigins, approval.Origin) {
		return fmt.Errorf("Browsertools approval escaped reviewed origins")
	}
	return nil
}

func canonicalLiveApprovalOrigin(origin string) bool {
	values, err := normalizeBrowserAuthoringOrigins([]string{origin})
	return err == nil && len(values) == 1 && values[0] == origin
}

func validLiveChallengeKinds(values []string) bool {
	if len(values) == 0 || len(values) > len(liveOTPChallengeKinds)+len(liveMFAChallengeKinds) {
		return false
	}
	family := 0
	seen := map[string]bool{}
	for _, value := range values {
		current := 0
		if containsExact(liveOTPChallengeKinds, value) {
			current = 1
		} else if containsExact(liveMFAChallengeKinds, value) {
			current = 2
		}
		if current == 0 || seen[value] || (family != 0 && current != family) {
			return false
		}
		family = current
		seen[value] = true
	}
	return true
}

func defaultLiveAuthorBounds() liveBounds {
	return liveBounds{
		NavigationTimeoutMS: 20_000, TotalTimeoutMS: liveAuthorDefaultTimeout.Milliseconds(),
		MaxRequests: 512, MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128, MaxOutputs: liveAuthorSelectedMaxOutputs,
	}
}

func liveObservationHasContext(observation liveObservation, id string) bool {
	if id == "main" {
		return true
	}
	if !liveContextPattern.MatchString(id) {
		return false
	}
	_, ok := observation.Contexts[id]
	return ok
}

func validateLiveObservationInventory(observation liveObservation, approvedOrigins []string, previous map[string]liveContext) error {
	if observation.Contexts == nil {
		return fmt.Errorf("Browsertools observation omitted its context inventory")
	}
	if !stringSliceContainsExact(approvedOrigins, observation.Origin) {
		return fmt.Errorf("Browsertools observation escaped reviewed origins")
	}
	if !liveObservationHasContext(observation, observation.Context) {
		return fmt.Errorf("Browsertools observation names an unknown active context")
	}
	for id, context := range observation.Contexts {
		if !stringSliceContainsExact(approvedOrigins, context.Origin) {
			return fmt.Errorf("Browsertools observation context %q escaped reviewed origins", id)
		}
	}
	for id, context := range previous {
		if current, ok := observation.Contexts[id]; !ok || current != context {
			return fmt.Errorf("Browsertools observation context inventory changed or dropped %q", id)
		}
	}
	return nil
}

func validLivePath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#\\") {
		return false
	}
	for _, part := range strings.Split(path, "/")[1:] {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, "/\\") {
			return false
		}
	}
	return true
}

func validSHA256Digest(value string) bool {
	if value != strings.ToLower(strings.TrimSpace(value)) || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	decoded, err := hex.DecodeString(raw)
	return err == nil && len(decoded) == sha256.Size
}

func containsExact(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
