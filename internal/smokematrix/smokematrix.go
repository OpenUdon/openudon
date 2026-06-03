package smokematrix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

const (
	ReportVersion = "openudon.product-smoke-matrix.v1"

	ModeDryRun = "dry-run"
	ModeLive   = "live"

	StatusPass              = "pass"
	StatusFail              = "fail"
	StatusDryRunOnly        = "dry_run_only"
	StatusSkippedMissingEnv = "skipped_missing_env"
	StatusSkippedManual     = "skipped_manual_provider"

	maxScenarioDetailLength = 1600
	truncatedDetailSuffix   = "... [truncated]"
)

type Options struct {
	RepoRoot    string
	WorkDir     string
	OutPath     string
	Mode        string
	ScenarioIDs []string
	Stdout      io.Writer
	Stderr      io.Writer
	Now         func() time.Time
	RunCommand  func(context.Context, string, ...string) error
	Scenarios   []Scenario
}

type Scenario struct {
	ID            string            `json:"id"`
	Fixture       string            `json:"fixture"`
	Sentence      string            `json:"sentence"`
	LiveKind      string            `json:"live_kind"`
	RequiredLive  bool              `json:"required_live,omitempty"`
	RequiredEnv   []string          `json:"required_env,omitempty"`
	EnvAliases    map[string]string `json:"env_aliases,omitempty"`
	Inputs        map[string]any    `json:"inputs,omitempty"`
	Overlay       string            `json:"overlay,omitempty"`
	ManualReason  string            `json:"manual_reason,omitempty"`
	DummyCreds    bool              `json:"dummy_credentials,omitempty"`
	SensitiveKeys []string          `json:"-"`
}

type Report struct {
	Version   string           `json:"version"`
	CreatedAt string           `json:"created_at"`
	Mode      string           `json:"mode"`
	Status    string           `json:"status"`
	WorkDir   string           `json:"workdir"`
	Scenarios []ScenarioResult `json:"scenarios"`
}

type ScenarioResult struct {
	ID              string   `json:"id"`
	Fixture         string   `json:"fixture"`
	Sentence        string   `json:"sentence"`
	Status          string   `json:"status"`
	LiveKind        string   `json:"live_kind"`
	RequiredLive    bool     `json:"required_live,omitempty"`
	MissingEnv      []string `json:"missing_env,omitempty"`
	ExampleDir      string   `json:"example_dir,omitempty"`
	ApprovalPath    string   `json:"approval_path,omitempty"`
	RunEvidencePath string   `json:"run_evidence_path,omitempty"`
	PackageSHA256   string   `json:"package_sha256,omitempty"`
	Detail          string   `json:"detail,omitempty"`
}

func DefaultScenarios() []Scenario {
	return []Scenario{
		{
			ID:           "slack-post",
			Fixture:      "slack-message-audit-log",
			Sentence:     "Post `OpenUdon v0.1.2-a.1 Slack smoke test` to my Slack sandbox channel and return the Slack response metadata.",
			LiveKind:     "external-provider",
			RequiredLive: true,
			RequiredEnv:  []string{"OPENUDON_SLACK_CHANNEL_ID", "UDON_CREDENTIAL_SLACK_BOT_TOKEN"},
			EnvAliases: map[string]string{
				"UDON_CREDENTIAL_SLACK":       "UDON_CREDENTIAL_SLACK_BOT_TOKEN",
				"UDON_CREDENTIAL_SLACKBEARER": "UDON_CREDENTIAL_SLACK_BOT_TOKEN",
			},
			Inputs: map[string]any{
				"channel": "ENV:OPENUDON_SLACK_CHANNEL_ID",
				"text":    "OpenUdon v0.1.2-a.1 Slack smoke test",
			},
			Overlay:       "slack-live",
			SensitiveKeys: []string{"UDON_CREDENTIAL_SLACK_BOT_TOKEN", "UDON_CREDENTIAL_SLACKBEARER"},
		},
		{
			ID:          "weather-read",
			Fixture:     "weather-toronto",
			Sentence:    "Fetch the current weather for Toronto and prepare a short audit summary.",
			LiveKind:    "external-provider",
			RequiredEnv: []string{"UDON_CREDENTIAL_OPENWEATHERAPIKEY"},
			EnvAliases:  map[string]string{"UDON_CREDENTIAL_INPUTS_APPID": "UDON_CREDENTIAL_OPENWEATHERAPIKEY"},
			Overlay:     "weather-live",
			Inputs: map[string]any{
				"appid": map[string]string{"ENVIRONMENT": "UDON_CREDENTIAL_OPENWEATHERAPIKEY"},
			},
			SensitiveKeys: []string{
				"UDON_CREDENTIAL_OPENWEATHERAPIKEY",
			},
		},
		{
			ID:           "gmail-audit-receipt",
			Fixture:      "gmail-send-audit-receipt",
			Sentence:     "Send an audit receipt email through Gmail with the approved package digest.",
			LiveKind:     "manual-provider",
			ManualReason: "Gmail live execution needs operator-owned OAuth consent, recipient review, and provider-specific runtime support.",
			Inputs: map[string]any{
				"to":      "smoke-recipient@example.com",
				"subject": "OpenUdon v0.1.2-a.1 Gmail smoke dry-run",
				"message": "OpenUdon product smoke matrix dry-run package.",
			},
		},
		{
			ID:           "slack-jira-intake",
			Fixture:      "itops-slack-jira-issue-intake",
			Sentence:     "Read a Slack incident report, create a Jira issue, and post a Slack confirmation.",
			LiveKind:     "manual-provider",
			ManualReason: "Slack plus Jira live execution needs reviewed sandbox app/channel/project registrations and operator-owned credentials.",
			Inputs: map[string]any{
				"channel":    "C0000000000",
				"messageTs":  "1710000000.000000",
				"projectKey": "OPS",
			},
		},
		{
			ID:         "order-fulfillment",
			Fixture:    "order-fulfillment-chain",
			Sentence:   "Look up a customer, check inventory, and create an order if stock is available.",
			LiveKind:   "local-stub",
			Overlay:    "local-stub",
			DummyCreds: true,
			Inputs: map[string]any{
				"customerId": "cust-smoke-1",
				"sku":        "SKU-SMOKE-1",
				"quantity":   1,
			},
		},
		{
			ID:         "header-api-key-report",
			Fixture:    "api-header-key-report",
			Sentence:   "Fetch a compliance report using a header API key.",
			LiveKind:   "local-stub",
			Overlay:    "local-stub",
			DummyCreds: true,
			Inputs: map[string]any{
				"reportId": "report-smoke-1",
			},
		},
		{
			ID:         "bearer-profile-fetch",
			Fixture:    "api-oauth-profile-fetch",
			Sentence:   "Fetch a directory profile using bearer authorization.",
			LiveKind:   "local-stub",
			Overlay:    "local-stub",
			DummyCreds: true,
			Inputs: map[string]any{
				"employeeId": "employee-smoke-1",
			},
		},
		{
			ID:         "inventory-api-key",
			Fixture:    "inventory-api-key-binding",
			Sentence:   "Read inventory details using an API key credential binding.",
			LiveKind:   "local-stub",
			Overlay:    "local-stub",
			DummyCreds: true,
			Inputs: map[string]any{
				"sku": "SKU-SMOKE-1",
			},
		},
		{
			ID:           "runtime-only-render",
			Fixture:      "runtime-only-render",
			Sentence:     "Render a local audit note without calling an external API.",
			LiveKind:     "dry-run-only",
			ManualReason: "Runtime-only function execution is executor-profile behavior; M37 gates package and trusted-runner dry-run evidence here.",
			Inputs: map[string]any{
				"summary": "OpenUdon v0.1.2-a.1 product smoke matrix runtime-only dry-run.",
			},
		},
	}
}

func Run(ctx context.Context, opts Options) (*Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = ModeDryRun
	}
	if mode != ModeDryRun && mode != ModeLive {
		return nil, fmt.Errorf("smoke matrix mode must be %q or %q", ModeDryRun, ModeLive)
	}
	repoRoot, err := filepath.Abs(defaultString(opts.RepoRoot, "."))
	if err != nil {
		return nil, err
	}
	workdir, err := filepath.Abs(defaultString(opts.WorkDir, filepath.Join(repoRoot, ".openudon-run", "product-smoke")))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return nil, err
	}
	scenarios := opts.Scenarios
	if len(scenarios) == 0 {
		scenarios = DefaultScenarios()
	}
	scenarios = filterScenarios(scenarios, opts.ScenarioIDs)
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("no smoke scenarios selected")
	}
	report := &Report{
		Version:   ReportVersion,
		CreatedAt: resolveNow(opts.Now).UTC().Format(time.RFC3339),
		Mode:      mode,
		Status:    StatusPass,
		WorkDir:   filepath.ToSlash(workdir),
		Scenarios: make([]ScenarioResult, 0, len(scenarios)),
	}
	for _, scenario := range scenarios {
		result := runScenario(ctx, repoRoot, workdir, mode, scenario, opts)
		report.Scenarios = append(report.Scenarios, result)
		if result.Status == StatusFail {
			report.Status = StatusFail
		}
	}
	if err := writeReport(opts.OutPath, report); err != nil {
		return report, err
	}
	if report.Status == StatusFail {
		return report, fmt.Errorf("product smoke matrix failed")
	}
	return report, nil
}

func runScenario(ctx context.Context, repoRoot, workdir, mode string, scenario Scenario, opts Options) ScenarioResult {
	result := ScenarioResult{
		ID:           scenario.ID,
		Fixture:      scenario.Fixture,
		Sentence:     scenario.Sentence,
		LiveKind:     scenario.LiveKind,
		RequiredLive: scenario.RequiredLive,
	}
	if mode == ModeLive {
		missing := missingScenarioEnv(scenario)
		if len(missing) > 0 && scenario.RequiredLive {
			result.Status = StatusFail
			result.MissingEnv = missing
			result.Detail = "required live smoke environment is incomplete"
			return result
		}
		if len(missing) > 0 && scenario.LiveKind == "external-provider" {
			result.Status = StatusSkippedMissingEnv
			result.MissingEnv = missing
			result.Detail = "external provider live smoke skipped because required env vars are missing"
			return result
		}
	}

	server, err := maybeStartStubServer(mode, scenario)
	if err != nil {
		result.Status = StatusFail
		result.Detail = err.Error()
		return result
	}
	if server != nil {
		defer server.Close()
	}

	exampleDir, err := prepareScenario(ctx, repoRoot, workdir, mode, scenario, serverURL(server))
	if err != nil {
		result.Status = StatusFail
		result.Detail = err.Error()
		return result
	}
	result.ExampleDir = filepath.ToSlash(exampleDir)
	approvalPath, runResult, runOutput, err := runTrusted(ctx, repoRoot, workdir, exampleDir, scenario, mode == ModeDryRun || scenario.LiveKind == "dry-run-only" || scenario.LiveKind == "manual-provider", opts)
	result.ApprovalPath = filepath.ToSlash(approvalPath)
	if runResult != nil {
		result.RunEvidencePath = filepath.ToSlash(runResult.RunEvidencePath)
		result.PackageSHA256 = runResult.PackageSHA256
	}
	if err != nil {
		result.Status = StatusFail
		result.Detail = sanitizeDetail(joinDetail(err.Error(), runOutput), scenario.SensitiveKeys)
		return result
	}
	if mode == ModeLive {
		switch scenario.LiveKind {
		case "dry-run-only":
			result.Status = StatusDryRunOnly
			result.Detail = scenario.ManualReason
		case "manual-provider":
			result.Status = StatusSkippedManual
			result.Detail = scenario.ManualReason
		default:
			result.Status = StatusPass
		}
	} else {
		result.Status = StatusPass
	}
	return result
}

func prepareScenario(ctx context.Context, repoRoot, workdir, mode string, scenario Scenario, stubURL string) (string, error) {
	src := filepath.Join(repoRoot, "examples", "eval", scenario.Fixture)
	dst := filepath.Join(workdir, "workspaces", scenario.ID)
	if err := os.RemoveAll(dst); err != nil {
		return "", err
	}
	if err := copyDir(src, dst); err != nil {
		return "", err
	}
	intentSrc := filepath.Join(dst, "reference", "intent.hcl")
	intentDst := filepath.Join(dst, "workflows", "intent.hcl")
	if err := os.MkdirAll(filepath.Dir(intentDst), 0o755); err != nil {
		return "", err
	}
	if err := copyFile(intentSrc, intentDst, 0o644); err != nil {
		return "", err
	}
	if err := seedDataFile(dst, scenario.Inputs); err != nil {
		return "", err
	}
	if err := applyOverlay(dst, mode, scenario, stubURL); err != nil {
		return "", err
	}
	if _, err := synthesize.Build(ctx, synthesize.Options{ExampleDir: dst}); err != nil {
		return "", err
	}
	return dst, nil
}

func runTrusted(ctx context.Context, repoRoot, workdir, exampleDir string, scenario Scenario, dryRun bool, opts Options) (string, *trustedrunner.RunResult, string, error) {
	approval, err := trustedrunner.ApprovalTemplate(ctx, trustedrunner.TemplateOptions{
		RepoRoot:   repoRoot,
		ExampleDir: exampleDir,
		State:      trustedrunner.StateApprovedForSandbox,
		Reviewer:   "OpenUdon M37 Product Smoke Matrix",
		Notes:      "M37 product smoke matrix evidence for v0.1.2-a.1.",
		Now:        opts.Now,
	})
	if err != nil {
		return "", nil, "", err
	}
	approvalPath := filepath.Join(workdir, "approvals", scenario.ID+".json")
	if err := os.MkdirAll(filepath.Dir(approvalPath), 0o755); err != nil {
		return "", nil, "", err
	}
	var approvalBuf bytes.Buffer
	if err := trustedrunner.WriteApproval(&approvalBuf, approval); err != nil {
		return "", nil, "", err
	}
	if err := os.WriteFile(approvalPath, approvalBuf.Bytes(), 0o600); err != nil {
		return "", nil, "", err
	}
	restore, err := applyScenarioEnv(exampleDir, scenario)
	if err != nil {
		return approvalPath, nil, "", err
	}
	defer restore()
	var stdout, stderr bytes.Buffer
	result, err := trustedrunner.Run(ctx, trustedrunner.Options{
		RepoRoot:     repoRoot,
		ExampleDir:   exampleDir,
		Tier:         trustedrunner.TierSandbox,
		ApprovalPath: approvalPath,
		WorkDir:      filepath.Join(workdir, "runs", scenario.ID),
		DryRun:       dryRun,
		RunnerPath:   os.Getenv("OPENUDON_UDON_RUNNER"),
		Stdout:       &stdout,
		Stderr:       &stderr,
		Now:          opts.Now,
		RunCommand:   opts.RunCommand,
	})
	return approvalPath, result, strings.TrimSpace(stdout.String() + "\n" + stderr.String()), err
}

func applyScenarioEnv(exampleDir string, scenario Scenario) (func(), error) {
	values := map[string]string{}
	for alias, source := range scenario.EnvAliases {
		if strings.TrimSpace(os.Getenv(alias)) == "" {
			if value := strings.TrimSpace(os.Getenv(source)); value != "" {
				values[alias] = value
			}
		}
	}
	if scenario.DummyCreds {
		for _, name := range credentialEnvNamesFromHandoff(filepath.Join(exampleDir, "expected", "review-handoff.json")) {
			if strings.TrimSpace(os.Getenv(name)) == "" {
				values[name] = "openudon-smoke-dummy"
			}
		}
	}
	return setTemporaryEnv(values), nil
}

func credentialEnvNamesFromHandoff(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var manifest struct {
		CredentialBindings struct {
			Declared         []string `json:"declared"`
			ExpectedFromPlan []string `json:"expected_from_plan"`
		} `json:"credential_bindings"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, binding := range append(manifest.CredentialBindings.Declared, manifest.CredentialBindings.ExpectedFromPlan...) {
		if name := credentialEnvName(binding); name != "" {
			seen[name] = true
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func credentialEnvName(binding string) string {
	var b strings.Builder
	b.WriteString("UDON_CREDENTIAL_")
	lastUnderscore := false
	for _, ch := range strings.TrimSpace(binding) {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.TrimRight(strings.ToUpper(b.String()), "_")
}

func filterScenarios(scenarios []Scenario, ids []string) []Scenario {
	wanted := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			wanted[id] = true
		}
	}
	if len(wanted) == 0 {
		return scenarios
	}
	out := make([]Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if wanted[scenario.ID] {
			out = append(out, scenario)
		}
	}
	return out
}

func joinDetail(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, compactWhitespace(part))
		}
	}
	return strings.Join(out, "; ")
}

func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func setTemporaryEnv(values map[string]string) func() {
	type oldValue struct {
		value string
		set   bool
	}
	previous := map[string]oldValue{}
	for name, value := range values {
		old, ok := os.LookupEnv(name)
		previous[name] = oldValue{value: old, set: ok}
		_ = os.Setenv(name, value)
	}
	return func() {
		for name, old := range previous {
			if old.set {
				_ = os.Setenv(name, old.value)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	}
}

func maybeStartStubServer(mode string, scenario Scenario) (*httptest.Server, error) {
	if mode != ModeLive || scenario.LiveKind != "local-stub" {
		return nil, nil
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/customers/"):
			fmt.Fprint(w, `{"id":"cust-smoke-1","email":"smoke@example.test","defaultShippingAddress":{"id":"addr-smoke-1","country":"US"}}`)
		case strings.HasPrefix(r.URL.Path, "/inventory/"):
			fmt.Fprint(w, `{"sku":"SKU-SMOKE-1","available":true,"quantity":9,"preferredWarehouseId":"wh-smoke-1"}`)
		case r.URL.Path == "/fulfillment/orders":
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"orderId":"order-smoke-1","confirmationNumber":"CONF-SMOKE-1","status":"created"}`)
		case strings.HasPrefix(r.URL.Path, "/reports/"):
			fmt.Fprint(w, `{"id":"report-smoke-1","status":"ready"}`)
		case strings.HasPrefix(r.URL.Path, "/employees/"):
			fmt.Fprint(w, `{"id":"employee-smoke-1","email":"employee-smoke@example.test"}`)
		default:
			http.NotFound(w, r)
		}
	})), nil
}

func serverURL(server *httptest.Server) string {
	if server == nil {
		return ""
	}
	return server.URL
}

func applyOverlay(exampleDir, mode string, scenario Scenario, stubURL string) error {
	switch scenario.Overlay {
	case "":
		return nil
	case "local-stub":
		if mode != ModeLive || stubURL == "" {
			return ensureTrustedExecutionBoundary(exampleDir)
		}
		if err := ensureTrustedExecutionBoundary(exampleDir); err != nil {
			return err
		}
		if scenario.ID == "order-fulfillment" {
			if err := replaceInFile(filepath.Join(exampleDir, "workflows", "intent.hcl"), `from = "create_fulfillment_order.received_body.confirmationNumber"`, `from = "create_fulfillment_order.received_body"`); err != nil {
				return err
			}
		}
		return replaceInOpenAPIs(exampleDir, map[string]string{
			"https://customers.example.test":      stubURL,
			"https://inventory.example.test":      stubURL,
			"https://orders.example.test/sandbox": stubURL,
			"https://compliance.example.test":     stubURL,
			"https://directory.example.test":      stubURL,
		})
	case "local-udon":
		if err := ensureTrustedExecutionBoundary(exampleDir); err != nil {
			return err
		}
		return replaceInFile(filepath.Join(exampleDir, "workflows", "intent.hcl"), `step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
  with = {`, `step "render_report" {
  type     = "fnct"
  provider = "identity"
  do       = "Render the summary report."
  with     = {`)
	case "slack-live":
		if mode != ModeLive {
			return nil
		}
		path := filepath.Join(exampleDir, "openapi", "slack.yaml")
		data := `openapi: 3.0.0
info:
  title: Slack Chat API
  version: 1.0.0
servers:
  - url: https://slack.com
paths:
  /api/chat.postMessage:
    post:
      operationId: postMessage
      security:
        - slackBearer: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [channel, text]
              properties:
                channel:
                  type: string
                text:
                  type: string
      responses:
        "200":
          description: posted message
          content:
            application/json:
              schema:
                type: object
                properties:
                  ok:
                    type: boolean
                  channel:
                    type: string
                  ts:
                    type: string
components:
  securitySchemes:
    slackBearer:
      type: http
      scheme: bearer
`
		projectPath := filepath.Join(exampleDir, "project.md")
		project, err := os.ReadFile(projectPath)
		if err != nil {
			return err
		}
		projectText := strings.Replace(string(project), "- No credentials are required for this sandbox fixture.", "- Use credential binding `slackBearer` for the Slack bot bearer token.", 1)
		if err := os.WriteFile(projectPath, []byte(projectText), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			return err
		}
		intent := `openapi = "openapi/slack.yaml"
workflow {
  name        = "slack_message_audit_log"
  description = "Post a sandbox chat message and return Slack response metadata."
}
input "channel" {
  type     = "string"
  required = true
}
input "text" {
  type     = "string"
  required = true
}
step "post_message" {
  type = "http"
  do   = "Post one sandbox chat message."
  with = {
    Authorization = "credentials.slackBearer"
    channel       = "inputs.channel"
    text          = "inputs.text"
  }
  operation = "postMessage"
}
output "slack_response" {
  from = "post_message.received_body"
}
`
		return os.WriteFile(filepath.Join(exampleDir, "workflows", "intent.hcl"), []byte(intent), 0o644)
	case "weather-live":
		if mode != ModeLive {
			return nil
		}
		intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
		if err := replaceInFile(intentPath, `step "get_coordinates" {`, `input "appid" {
  type     = "string"
  required = true
}

step "get_coordinates" {`); err != nil {
			return err
		}
		if err := replaceInFile(intentPath, `    q = "Toronto,CA"`, `    appid = "inputs.appid"
    q     = "Toronto,CA"`); err != nil {
			return err
		}
		if err := replaceInFile(intentPath, `appid = "weather_appid"`, `appid = "inputs.appid"`); err != nil {
			return err
		}
		if err := replaceInFile(filepath.Join(exampleDir, "project.md"), "weather_appid", "OpenWeatherAPIKey"); err != nil {
			return err
		}
		path := filepath.Join(exampleDir, "openapi", "weather.yaml")
		data := `openapi: 3.0.0
info:
  title: Weather API
  version: 1.0.0
servers:
  - url: https://api.openweathermap.org
paths:
  /geo/1.0/direct:
    get:
      operationId: direct_get
      parameters:
        - name: appid
          in: query
          required: true
          schema:
            type: string
        - name: q
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: Coordinates
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    lat:
                      type: number
                    lon:
                      type: number
  /data/2.5/weather:
    get:
      operationId: getWeatherData
      parameters:
        - name: lat
          in: query
          required: true
          schema:
            type: number
        - name: lon
          in: query
          required: true
          schema:
            type: number
        - name: appid
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: Weather
          content:
            application/json:
              schema:
                type: object
                properties:
                  weather:
                    type: array
`
		return os.WriteFile(path, []byte(data), 0o644)
	default:
		return fmt.Errorf("unknown smoke overlay %q", scenario.Overlay)
	}
}

func ensureTrustedExecutionBoundary(exampleDir string) error {
	projectPath := filepath.Join(exampleDir, "project.md")
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return err
	}
	text := string(data)
	if strings.Contains(strings.ToLower(text), "trusted-runner execution") || strings.Contains(strings.ToLower(text), "trusted runner") {
		return nil
	}
	note := "- M37 product smoke execution requires sandbox approval plus trusted-runner execution.\n"
	if strings.Contains(text, "## Safety and Approval Boundary\n\n") {
		text = strings.Replace(text, "## Safety and Approval Boundary\n\n", "## Safety and Approval Boundary\n\n"+note, 1)
	} else {
		text += "\n## Safety and Approval Boundary\n\n" + note
	}
	return os.WriteFile(projectPath, []byte(text), 0o644)
}

func replaceInFile(path, old, newValue string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := strings.Replace(string(data), old, newValue, 1)
	return os.WriteFile(path, []byte(text), 0o644)
}

func replaceInOpenAPIs(exampleDir string, replacements map[string]string) error {
	root := filepath.Join(exampleDir, "openapi")
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for old, newValue := range replacements {
			text = strings.ReplaceAll(text, old, newValue)
		}
		return os.WriteFile(path, []byte(text), 0o644)
	})
}

func seedDataFile(exampleDir string, inputs map[string]any) error {
	if len(inputs) == 0 {
		return nil
	}
	path := filepath.Join(exampleDir, "expected", "data.hcl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("inputs {\n")
	for _, name := range names {
		fmt.Fprintf(&b, "  %s = %s\n", name, hclLiteral(resolveInputValue(inputs[name])))
	}
	b.WriteString("}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func resolveInputValue(value any) any {
	text, ok := value.(string)
	if !ok || !strings.HasPrefix(text, "ENV:") {
		return value
	}
	envName := strings.TrimPrefix(text, "ENV:")
	if envValue := os.Getenv(envName); strings.TrimSpace(envValue) != "" {
		return envValue
	}
	return ""
}

func hclLiteral(value any) string {
	switch v := value.(type) {
	case map[string]string:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("{ ")
		for i, key := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s = %s", key, strconv.Quote(v[key]))
		}
		b.WriteString(" }")
		return b.String()
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strconv.Quote(fmt.Sprint(v))
	}
}

func missingScenarioEnv(scenario Scenario) []string {
	var missing []string
	for _, name := range scenario.RequiredEnv {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			continue
		}
		sourceFound := false
		for alias, source := range scenario.EnvAliases {
			if alias == name && strings.TrimSpace(os.Getenv(source)) != "" {
				sourceFound = true
				break
			}
		}
		if !sourceFound {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func sanitizeDetail(detail string, keys []string) string {
	for _, key := range keys {
		if value := os.Getenv(key); strings.TrimSpace(value) != "" {
			detail = strings.ReplaceAll(detail, value, "[redacted]")
		}
	}
	return limitDetail(detail)
}

func limitDetail(detail string) string {
	if len(detail) <= maxScenarioDetailLength {
		return detail
	}
	cutoff := maxScenarioDetailLength - len(truncatedDetailSuffix)
	if cutoff < 0 {
		cutoff = 0
	}
	return strings.TrimSpace(detail[:cutoff]) + truncatedDetailSuffix
}

func writeReport(path string, report *Report) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	return os.WriteFile(dst, data, mode)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func resolveNow(now func() time.Time) time.Time {
	if now != nil {
		return now()
	}
	return time.Now()
}
