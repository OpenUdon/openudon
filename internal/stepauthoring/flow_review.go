package stepauthoring

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/credentialpolicy"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	FlowReviewCommand        = "flow-review"
	MaxModelReviewInputBytes = 256 << 10
)

var (
	flowReviewProviderPattern = regexp.MustCompile(`^(openai|anthropic|gemini|copilot-api)$`)
	flowReviewModelPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/+-]{0,127}$`)
	flowReviewCodePattern     = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)
	flowReviewEmailPattern    = regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)
	flowReviewPhonePattern    = regexp.MustCompile(`(?:\+?\d[\d .()/-]{8,}\d)`)
	flowReviewInjectionText   = regexp.MustCompile(`(?i)(ignore (all )?(previous|prior) instructions|system prompt|developer prompt|reveal (the )?(secret|credential))`)
)

type FlowReviewModelRequest struct {
	Enabled  bool    `json:"enabled"`
	Provider *string `json:"provider,omitempty"`
	Model    *string `json:"model,omitempty"`
}

type FlowReviewRequest struct {
	Version      string                  `json:"version"`
	Kind         string                  `json:"kind"`
	Command      string                  `json:"command"`
	IntentSHA256 string                  `json:"intent_sha256"`
	ModelReview  *FlowReviewModelRequest `json:"model_review"`
}

type ReviewFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	StepID   string `json:"step_id,omitempty"`
	Pointer  string `json:"pointer,omitempty"`
}

type FlowReviewSection struct {
	Status   string          `json:"status"`
	Findings []ReviewFinding `json:"findings"`
}

type FlowReviewData struct {
	IntentSHA256 string            `json:"intent_sha256"`
	LocalReview  FlowReviewSection `json:"local_review"`
	ModelReview  FlowReviewSection `json:"model_review"`
}

type FlowReviewWireResult struct {
	Version     string          `json:"version"`
	Kind        string          `json:"kind"`
	Command     string          `json:"command"`
	Status      string          `json:"status"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
	Result      *FlowReviewData `json:"result,omitempty"`
}

type FlowReviewOutcome struct {
	Result   FlowReviewWireResult
	ExitCode int
}

// DraftReviewer is the narrow part of the iCoT extractor used by flow-review.
// Keeping the interface small lets tests exercise failures without a provider.
type DraftReviewer interface {
	ReviewDraft(context.Context, elicitor.DraftReviewRequest) (elicitor.DraftReviewResponse, error)
}

type DraftReviewerFactory func(provider, model string) (DraftReviewer, error)

// ReviewFlow validates a single intent revision, performs deterministic local
// review, and optionally invokes the existing iCoT advisory model review. It
// never writes files or returns source paths, prompt payloads, or provider
// errors.
func ReviewFlow(ctx context.Context, exampleDir string, request FlowReviewRequest, factory DraftReviewerFactory) FlowReviewOutcome {
	if request.Version != "" && request.Version != WireVersion {
		return flowReviewFailure("request.unsupported_version", "The flow-review request version is not supported.", 2)
	}
	if err := validateFlowReviewRequest(request); err != nil {
		return flowReviewFailure("request.invalid", "The flow-review request is invalid.", 2)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return flowReviewBlocked("review.cancelled", "The flow review was cancelled before it started.")
	}
	root, err := resolveExampleRoot(exampleDir)
	if err != nil {
		return flowReviewBlocked("example.invalid", "The selected example is unavailable or unsafe.")
	}
	intentBytes, err := readWithin(root, defaultIntentPath, evidencefile.DefaultMaxBytes)
	if err != nil || !utf8.Valid(intentBytes) {
		return flowReviewBlocked("intent.unavailable", "The workflow intent is unavailable or unsafe to read.")
	}
	intentDigest := sourceDigest(intentBytes)
	if request.IntentSHA256 != intentDigest {
		return flowReviewConflict("intent.stale", "The workflow intent changed after the review request was created.")
	}
	intent, err := workflowintent.ParseIntent(intentBytes, defaultIntentPath)
	if err != nil {
		return flowReviewBlocked("intent.invalid", "The workflow intent could not be validated safely.")
	}
	if err := ctx.Err(); err != nil {
		return flowReviewBlocked("review.cancelled", "The flow review was cancelled before local review completed.")
	}

	query := ""
	if intent.Workflow != nil {
		query = strings.TrimSpace(intent.Workflow.Description)
	}
	discovery, err := elicitor.DiscoverAuthoringSources(ctx, root, query, nil, nil)
	if err != nil {
		if ctx.Err() != nil {
			return flowReviewBlocked("review.cancelled", "The flow review was cancelled during source inspection.")
		}
		return flowReviewBlocked("source.discovery_failed", "Local source evidence could not be inspected safely.")
	}
	if err := ctx.Err(); err != nil {
		return flowReviewBlocked("review.cancelled", "The flow review was cancelled during source inspection.")
	}
	session, err := elicitor.SessionFromIntent(intent, projectwizard.Answers{})
	if err != nil {
		return flowReviewBlocked("intent.invalid", "The workflow intent could not be prepared for review.")
	}
	// Model review receives only symbolic credential requirements. Do not load
	// project.md or any runtime credential data into this command.
	session.Credentials = nil
	session.Project.Credentials = nil
	localIssues := elicitor.ReviewDraftLocally(session, discovery.Docs)
	data := &FlowReviewData{
		IntentSHA256: intentDigest,
		LocalReview:  FlowReviewSection{Status: "completed", Findings: make([]ReviewFinding, 0)},
		ModelReview:  FlowReviewSection{Status: "skipped", Findings: make([]ReviewFinding, 0)},
	}
	for _, issue := range localIssues {
		finding, ok := localReviewFinding(issue)
		if ok {
			data.LocalReview.Findings = append(data.LocalReview.Findings, finding)
		}
		if len(data.LocalReview.Findings) >= 64 {
			break
		}
	}
	result := FlowReviewWireResult{
		Version: WireVersion, Kind: "result", Command: FlowReviewCommand,
		Status: "completed", Diagnostics: make([]Diagnostic, 0), Result: data,
	}
	if discovery.Report.Truncated || len(discovery.Report.Ambiguous) > 0 || len(discovery.BrowserReport.Truncated) > 0 || len(discovery.BrowserReport.Ambiguous) > 0 || hasInactiveBrowserCandidate(discovery.BrowserReport.Candidates) {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "source.discovery_incomplete", Severity: "warning",
			Message: "Local source discovery was incomplete; source-backed review coverage may be partial.",
		})
	}
	if !request.ModelReview.Enabled {
		return FlowReviewOutcome{Result: result, ExitCode: 0}
	}
	if factory == nil {
		factory = defaultDraftReviewerFactory
	}
	reviewRequest := elicitor.BuildDraftReviewRequest(session, discovery.Docs, nil)
	reviewRequest.Credentials = nil
	for i := range reviewRequest.Steps {
		// The model reviews operation semantics, not local file identity. Avoid
		// exporting example-relative paths or source provenance to the provider.
		reviewRequest.Steps[i].Source = ""
		reviewRequest.Steps[i].Operation.Provenance = ""
	}
	if len(reviewRequest.Steps) == 0 || strings.TrimSpace(reviewRequest.Goal) == "" {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "model_review.no_reviewable_steps", Severity: "warning",
			Message: "Model review was skipped because the intent has no reviewable goal and steps.",
		})
		return FlowReviewOutcome{Result: result, ExitCode: 0}
	}
	requestBytes, err := json.Marshal(reviewRequest)
	if err != nil {
		return flowReviewBlocked("review.request_failed", "The model-review request could not be prepared safely.")
	}
	if len(requestBytes) > MaxModelReviewInputBytes {
		data.ModelReview.Status = "unavailable"
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "model_review.context_too_large", Severity: "warning",
			Message: "Model review was not sent because the bounded draft context exceeds its size limit.",
		})
		return FlowReviewOutcome{Result: result, ExitCode: 0}
	}
	if authoring.ContainsLikelyCredentialValue(requestBytes) {
		data.ModelReview.Status = "unavailable"
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "model_review.input_rejected", Severity: "warning",
			Message: "Model review was not sent because its bounded draft context appears to contain a credential value.",
		})
		return FlowReviewOutcome{Result: result, ExitCode: 0}
	}
	if err := ctx.Err(); err != nil {
		data.ModelReview.Status = "failed"
		result.Status = "blocked"
		result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: "review.cancelled", Severity: "error", Message: "The flow review was cancelled before model review."})
		return FlowReviewOutcome{Result: result, ExitCode: 4}
	}
	reviewer, err := factory(*request.ModelReview.Provider, *request.ModelReview.Model)
	if err != nil || reviewer == nil {
		data.ModelReview.Status = "unavailable"
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "model_review.unavailable", Severity: "warning",
			Message: "The requested model reviewer is unavailable; only local review was completed.",
		})
		return FlowReviewOutcome{Result: result, ExitCode: 0}
	}
	response, err := reviewer.ReviewDraft(ctx, reviewRequest)
	if err != nil {
		data.ModelReview.Status = "failed"
		code, status, exitCode := "model_review.failed", "failed", 1
		message := "The configured model review failed; its findings are not available."
		if ctx.Err() != nil {
			code, status, exitCode = "review.cancelled", "blocked", 4
			message = "The flow review was cancelled during model review."
		}
		result.Status = status
		result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: code, Severity: "error", Message: message})
		return FlowReviewOutcome{Result: result, ExitCode: exitCode}
	}
	modelFindings, safe := modelReviewFindings(response)
	if !safe {
		data.ModelReview.Status = "failed"
		result.Status = "failed"
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Code: "model_review.output_rejected", Severity: "error",
			Message: "The model response included a finding that could not be safely disclosed.",
		})
		return FlowReviewOutcome{Result: result, ExitCode: 1}
	}
	data.ModelReview.Status = "completed"
	data.ModelReview.Findings = modelFindings
	return FlowReviewOutcome{Result: result, ExitCode: 0}
}

func validateFlowReviewRequest(request FlowReviewRequest) error {
	if request.Version == "" || request.Version != WireVersion || request.Kind != "request" || request.Command != FlowReviewCommand || !sha256Pattern.MatchString(request.IntentSHA256) || request.ModelReview == nil {
		return fmt.Errorf("invalid wire envelope")
	}
	if request.ModelReview.Enabled {
		if request.ModelReview.Provider == nil || request.ModelReview.Model == nil ||
			!flowReviewProviderPattern.MatchString(*request.ModelReview.Provider) || !flowReviewModelPattern.MatchString(*request.ModelReview.Model) {
			return fmt.Errorf("model review requires an explicit supported provider and model")
		}
		return nil
	}
	if request.ModelReview.Provider != nil || request.ModelReview.Model != nil {
		return fmt.Errorf("disabled model review cannot select a provider or model")
	}
	return nil
}

// ValidFlowReviewRequestJSON preserves the presence-sensitive distinctions in
// the public schema (notably a forbidden empty/null provider property when
// model review is disabled) before ordinary typed decoding loses them.
func ValidFlowReviewRequestJSON(data []byte) bool {
	var envelope map[string]json.RawMessage
	if json.Unmarshal(data, &envelope) != nil {
		return false
	}
	reviewRaw, ok := envelope["model_review"]
	if !ok || string(reviewRaw) == "null" {
		return false
	}
	var review map[string]json.RawMessage
	if json.Unmarshal(reviewRaw, &review) != nil || review == nil {
		return false
	}
	enabledRaw, ok := review["enabled"]
	if !ok || (string(enabledRaw) != "true" && string(enabledRaw) != "false") {
		return false
	}
	if string(enabledRaw) == "true" {
		for _, name := range []string{"provider", "model"} {
			raw, exists := review[name]
			if !exists {
				return false
			}
			var value string
			if json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" {
				return false
			}
		}
		return true
	}
	_, providerPresent := review["provider"]
	_, modelPresent := review["model"]
	return !providerPresent && !modelPresent
}

func defaultDraftReviewerFactory(provider, model string) (DraftReviewer, error) {
	client, _, _, err := workflowintent.NewLLMClientFromEnvWithOptions(provider, model, workflowintent.LLMOptions{})
	if err != nil {
		return nil, err
	}
	chat, ok := client.(workflowintent.ChatClient)
	if !ok {
		return nil, fmt.Errorf("configured provider does not support chat")
	}
	return elicitor.NewChatExtractor(chat, nil), nil
}

func flowReviewFailure(code, message string, exitCode int) FlowReviewOutcome {
	return FlowReviewOutcome{Result: FlowReviewWireResult{
		Version: WireVersion, Kind: "result", Command: FlowReviewCommand, Status: "failed",
		Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}},
	}, ExitCode: exitCode}
}

func flowReviewBlocked(code, message string) FlowReviewOutcome {
	return FlowReviewOutcome{Result: FlowReviewWireResult{
		Version: WireVersion, Kind: "result", Command: FlowReviewCommand, Status: "blocked",
		Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}},
	}, ExitCode: 4}
}

func flowReviewConflict(code, message string) FlowReviewOutcome {
	return FlowReviewOutcome{Result: FlowReviewWireResult{Version: WireVersion, Kind: "result", Command: FlowReviewCommand,
		Status: "conflict", Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}},
	}, ExitCode: 3}
}

func localReviewFinding(issue elicitor.DraftReviewIssue) (ReviewFinding, bool) {
	code := strings.TrimSpace(issue.Code)
	if !flowReviewCodePattern.MatchString(code) {
		return ReviewFinding{}, false
	}
	message := localReviewMessage(code)
	stepID := reviewStepID(issue.Slot)
	return ReviewFinding{Code: code, Severity: "warning", Message: message, StepID: stepID}, message != ""
}

func localReviewMessage(code string) string {
	switch code {
	case "browser_review_operation_unavailable":
		return "The browser step does not resolve to one reviewed capability action."
	case "browser_review_external_session_missing":
		return "A login-dependent browser step lacks a symbolic execution-local session binding."
	case "browser_review_mutation_unapproved":
		return "A mutating browser step lacks its exact authoring approval."
	case "llm_flow_review_missing_rendered_request_body":
		return "A downstream delivery step lacks a rendered request-body mapping required by its selected operation."
	default:
		return "The local flow review identified an advisory concern."
	}
}

func modelReviewFindings(response elicitor.DraftReviewResponse) ([]ReviewFinding, bool) {
	sanitized := elicitor.SanitizeDraftReview(response)
	findings := make([]ReviewFinding, 0, len(sanitized.Issues))
	for _, issue := range sanitized.Issues {
		code := strings.TrimSpace(issue.Code)
		message := safeFindingMessage(issue.Message)
		if !flowReviewCodePattern.MatchString(code) || message == "" {
			return nil, false
		}
		findings = append(findings, ReviewFinding{Code: code, Severity: "warning", Message: message})
		if len(findings) >= 32 {
			break
		}
	}
	return findings, true
}

func safeFindingMessage(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !utf8.ValidString(value) || unsafeReviewText(value) {
		return ""
	}
	var builder strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			if unicode.IsSpace(r) {
				space = builder.Len() > 0
				continue
			}
			return ""
		}
		if unicode.IsSpace(r) {
			space = builder.Len() > 0
			continue
		}
		if space {
			builder.WriteByte(' ')
			space = false
		}
		builder.WriteRune(r)
	}
	message := strings.TrimSpace(builder.String())
	if len(message) > 512 {
		message = truncateUTF8(message, 512)
	}
	if message == "" || unsafeReviewText(message) {
		return ""
	}
	return message
}

func unsafeReviewText(value string) bool {
	if authoring.ContainsLikelyCredentialValue([]byte(value)) || flowReviewEmailPattern.MatchString(value) ||
		flowReviewPhonePattern.MatchString(value) || flowReviewInjectionText.MatchString(value) {
		return true
	}
	for _, token := range strings.FieldsFunc(value, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._-", r))
	}) {
		if credentialpolicy.IsLikelyLiteral(token) {
			return true
		}
	}
	return false
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.RuneStart(value[maxBytes]) {
		maxBytes--
	}
	return strings.TrimSpace(value[:maxBytes])
}

func reviewStepID(slot string) string {
	parts := strings.Split(slot, ".")
	if len(parts) >= 2 && parts[0] == "steps" && symbol(parts[1]) {
		return parts[1]
	}
	return ""
}

func hasInactiveBrowserCandidate(candidates []browsertools.LocalSourceCandidate) bool {
	for _, candidate := range candidates {
		if candidate.Status != "active" {
			return true
		}
	}
	return false
}
