package workflowintent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/uws/uws1"
	"github.com/genelet/ramen/internal/authoring"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"gopkg.in/yaml.v3"
)

const IntentPath = "workflows/intent.hcl"
const DefaultCopilotAPIModel = "gpt-5.4-mini"

const (
	defaultLLMTimeout        = 60 * time.Second
	structuredToolName       = "emit_json"
	structuredSchemaName     = "structured_output"
	structuredToolDescriptor = "Emit JSON matching the supplied schema."
)

type ChatMessage struct {
	Role    string
	Content string
}

type ChatClient interface {
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
}

type StructuredChat interface {
	ChatClient
	StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error)
}

type StructuredOpts struct {
	Temperature *float64
	MaxTokens   int
}

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type LLMClientOption func(*llmClientConfig)

type llmClientConfig struct {
	httpClient  *http.Client
	temperature *float64
}

func WithLLMHTTPClient(client *http.Client) LLMClientOption {
	return func(config *llmClientConfig) {
		if client != nil {
			config.httpClient = client
		}
	}
}

func WithLLMTemperature(value float64) LLMClientOption {
	return func(config *llmClientConfig) {
		config.temperature = &value
	}
}

type providerLLMClient struct {
	provider string
	model    string
	apiKey   string
	client   *http.Client
	temp     *float64
	timeout  time.Duration
}

func (c *providerLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	return c.Chat(ctx, []ChatMessage{{Role: "user", Content: prompt}})
}

func (c *providerLLMClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	switch c.provider {
	case "openai", "copilot-api":
		if c.openAIEndpoint() == "responses" {
			return c.responsesChat(ctx, messages)
		}
		return c.chatOpenAI(ctx, c.openAIBaseURL()+"/v1/chat/completions", messages)
	case "anthropic":
		base := strings.TrimRight(os.Getenv("ANTHROPIC_BASE_URL"), "/")
		if base == "" {
			base = "https://api.anthropic.com"
		}
		return c.chatAnthropic(ctx, base+"/v1/messages", messages)
	case "gemini":
		base := strings.TrimRight(os.Getenv("GEMINI_BASE_URL"), "/")
		if base == "" {
			base = "https://generativelanguage.googleapis.com"
		}
		model := strings.TrimPrefix(strings.TrimSpace(c.model), "models/")
		apiURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", base, url.PathEscape(model), url.QueryEscape(c.apiKey))
		return c.chatGemini(ctx, apiURL, messages)
	default:
		return "", fmt.Errorf("unknown provider: %s", c.provider)
	}
}

func (c *providerLLMClient) StructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	if len(schema) == 0 {
		return "", fmt.Errorf("structured output schema is required")
	}
	switch c.provider {
	case "openai", "copilot-api":
		if c.openAIEndpoint() == "responses" {
			return c.responsesStructuredChat(ctx, messages, schema, opts)
		}
		return c.chatCompletionsStructuredChat(ctx, messages, schema, opts)
	case "anthropic":
		return c.anthropicStructuredChat(ctx, messages, schema, opts)
	case "gemini":
		return c.geminiStructuredChat(ctx, messages, schema, opts)
	default:
		return "", fmt.Errorf("unknown provider: %s", c.provider)
	}
}

func (c *providerLLMClient) chatOpenAI(ctx context.Context, url string, messages []ChatMessage) (string, error) {
	payload := map[string]any{"model": c.model, "messages": openAIChatMessages(messages)}
	if c.temp != nil {
		payload["temperature"] = *c.temp
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.postJSON(ctx, url, map[string]string{"Authorization": "Bearer " + c.apiKey}, payload, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai response contained no choices")
	}
	return out.Choices[0].Message.Content, nil
}

func (c *providerLLMClient) chatCompletionsStructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	payload := map[string]any{
		"model":    c.model,
		"messages": openAIChatMessages(messages),
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   structuredSchemaName,
				"strict": true,
				"schema": schema,
			},
		},
	}
	if temp := structuredTemperature(c.temp, opts); temp != nil {
		payload["temperature"] = *temp
	}
	if opts.MaxTokens > 0 {
		payload["max_tokens"] = opts.MaxTokens
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.postJSON(ctx, c.openAIBaseURL()+"/v1/chat/completions", map[string]string{"Authorization": "Bearer " + c.apiKey}, payload, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai response contained no choices")
	}
	return out.Choices[0].Message.Content, nil
}

func openAIChatMessages(messages []ChatMessage) []map[string]string {
	out := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role == "" {
			role = "user"
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		out = append(out, map[string]string{"role": role, "content": content})
	}
	return out
}

func (c *providerLLMClient) chatAnthropic(ctx context.Context, url string, messages []ChatMessage) (string, error) {
	var system string
	var apiMessages []map[string]string
	for _, message := range messages {
		if message.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += message.Content
			continue
		}
		apiMessages = append(apiMessages, map[string]string{"role": message.Role, "content": message.Content})
	}
	payload := map[string]any{"model": c.model, "max_tokens": 8192, "messages": apiMessages}
	if system != "" {
		payload["system"] = system
	}
	if c.temp != nil {
		payload["temperature"] = *c.temp
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := c.postJSON(ctx, url, map[string]string{"x-api-key": c.apiKey, "anthropic-version": "2023-06-01"}, payload, &out); err != nil {
		return "", err
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("anthropic response contained no content")
	}
	return out.Content[0].Text, nil
}

func (c *providerLLMClient) anthropicStructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	base := strings.TrimRight(os.Getenv("ANTHROPIC_BASE_URL"), "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	var system string
	var apiMessages []map[string]string
	for _, message := range messages {
		if message.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += message.Content
			continue
		}
		apiMessages = append(apiMessages, map[string]string{"role": message.Role, "content": message.Content})
	}
	payload := map[string]any{
		"model":      c.model,
		"max_tokens": structuredMaxTokens(opts, 8192),
		"messages":   apiMessages,
		"tools": []map[string]any{{
			"name":         structuredToolName,
			"description":  structuredToolDescriptor,
			"input_schema": schema,
		}},
		"tool_choice": map[string]string{"type": "tool", "name": structuredToolName},
	}
	if system != "" {
		payload["system"] = system
	}
	if temp := structuredTemperature(c.temp, opts); temp != nil {
		payload["temperature"] = *temp
	}
	var out struct {
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := c.postJSON(ctx, base+"/v1/messages", map[string]string{"x-api-key": c.apiKey, "anthropic-version": "2023-06-01"}, payload, &out); err != nil {
		return "", err
	}
	for _, content := range out.Content {
		if content.Type == "tool_use" && content.Name == structuredToolName && len(content.Input) > 0 {
			return string(content.Input), nil
		}
	}
	return "", fmt.Errorf("structured output unsupported: Anthropic response did not include %s tool input", structuredToolName)
}

func (c *providerLLMClient) chatGemini(ctx context.Context, url string, messages []ChatMessage) (string, error) {
	var parts []map[string]string
	for _, message := range messages {
		parts = append(parts, map[string]string{"text": strings.ToUpper(message.Role) + ": " + message.Content})
	}
	payload := map[string]any{"contents": []map[string]any{{"parts": parts}}}
	if c.temp != nil {
		payload["generationConfig"] = map[string]any{"temperature": *c.temp}
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := c.postJSON(ctx, url, nil, payload, &out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini response contained no content")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

func (c *providerLLMClient) geminiStructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	base := strings.TrimRight(os.Getenv("GEMINI_BASE_URL"), "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	model := strings.TrimPrefix(strings.TrimSpace(c.model), "models/")
	apiURL := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", base, url.PathEscape(model), url.QueryEscape(c.apiKey))
	var parts []map[string]string
	for _, message := range messages {
		if strings.TrimSpace(message.Content) != "" {
			parts = append(parts, map[string]string{"text": strings.ToUpper(message.Role) + ": " + message.Content})
		}
	}
	payload := map[string]any{
		"contents": []map[string]any{{"parts": parts}},
		"generationConfig": map[string]any{
			"responseMimeType":   "application/json",
			"responseJsonSchema": schema,
		},
	}
	if temp := structuredTemperature(c.temp, opts); temp != nil {
		payload["generationConfig"].(map[string]any)["temperature"] = *temp
	}
	if opts.MaxTokens > 0 {
		payload["generationConfig"].(map[string]any)["maxOutputTokens"] = opts.MaxTokens
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := c.postJSON(ctx, apiURL, nil, payload, &out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini response contained no content")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

func (c *providerLLMClient) responsesChat(ctx context.Context, messages []ChatMessage) (string, error) {
	payload := map[string]any{
		"model": c.model,
		"input": responsesInput(messages),
	}
	if c.temp != nil {
		payload["temperature"] = *c.temp
	}
	return c.doResponsesRequest(ctx, payload)
}

func (c *providerLLMClient) responsesStructuredChat(ctx context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	payload := map[string]any{
		"model": c.model,
		"input": responsesInput(messages),
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   structuredSchemaName,
				"strict": true,
				"schema": schema,
			},
		},
	}
	if temp := structuredTemperature(c.temp, opts); temp != nil {
		payload["temperature"] = *temp
	}
	if opts.MaxTokens > 0 {
		payload["max_output_tokens"] = opts.MaxTokens
	}
	return c.doResponsesRequest(ctx, payload)
}

func (c *providerLLMClient) doResponsesRequest(ctx context.Context, payload any) (string, error) {
	var out struct {
		OutputText *string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := c.postJSON(ctx, c.openAIBaseURL()+"/v1/responses", map[string]string{"Authorization": "Bearer " + c.apiKey}, payload, &out); err != nil {
		return "", err
	}
	if out.OutputText != nil && strings.TrimSpace(*out.OutputText) != "" {
		return *out.OutputText, nil
	}
	var parts []string
	for _, item := range out.Output {
		for _, content := range item.Content {
			if content.Text != "" {
				parts = append(parts, content.Text)
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("openai responses output was empty")
	}
	return strings.Join(parts, ""), nil
}

func responsesInput(messages []ChatMessage) []map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(message.Role))
		if role == "" {
			role = "user"
		}
		input = append(input, map[string]any{
			"role": role,
			"content": []map[string]string{{
				"type": "input_text",
				"text": content,
			}},
		})
	}
	return input
}

func (c *providerLLMClient) openAIBaseURL() string {
	if c.provider == "copilot-api" {
		base := strings.TrimRight(os.Getenv("COPILOT_API_BASE_URL"), "/")
		if base == "" {
			base = "http://localhost:4141"
		}
		return base
	}
	base := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	if base == "" {
		base = "https://api.openai.com"
	}
	return base
}

func (c *providerLLMClient) openAIEndpoint() string {
	if c.provider == "copilot-api" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.model)), "gpt-5") {
		return "responses"
	}
	return "chat_completions"
}

func (c *providerLLMClient) postJSON(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return redactedLLMTransportError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s: %s", c.provider, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, out)
}

func contextWithLLMDeadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	if timeout <= 0 {
		timeout = defaultLLMTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

func structuredTemperature(defaultValue *float64, opts StructuredOpts) *float64 {
	if opts.Temperature != nil {
		return opts.Temperature
	}
	return defaultValue
}

func structuredMaxTokens(opts StructuredOpts, fallback int) int {
	if opts.MaxTokens > 0 {
		return opts.MaxTokens
	}
	return fallback
}

func redactedLLMTransportError(err error) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr != nil {
		redacted := *urlErr
		redacted.URL = redactURLSecret(redacted.URL)
		return fmt.Errorf("%s", redacted.Error())
	}
	return err
}

func redactURLSecret(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return raw
	}
	query := parsed.Query()
	changed := false
	for _, key := range []string{"key", "api_key", "access_token"} {
		if _, ok := query[key]; ok {
			query.Set(key, "REDACTED")
			changed = true
		}
	}
	if !changed {
		return raw
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

type LLMOptions struct {
	Temperature *float64
}

func NewLLMClientFromEnvWithOptions(provider, model string, opts LLMOptions) (LLMClient, string, string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "copilot-api"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(os.Getenv("ROLLOUT_MODEL"))
	}
	switch provider {
	case "anthropic":
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			return nil, "", "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		if model == "" {
			model = "claude-3-opus-20240229"
		}
	case "openai":
		if os.Getenv("OPENAI_API_KEY") == "" {
			return nil, "", "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		if model == "" {
			model = "gpt-4-turbo"
		}
	case "gemini":
		if os.Getenv("GEMINI_API_KEY") == "" {
			return nil, "", "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
		}
		if model == "" {
			model = "gemini-1.5-pro"
		}
	case "copilot-api":
		if model == "" {
			model = DefaultCopilotAPIModel
		}
	default:
		return nil, "", "", fmt.Errorf("unknown provider: %s", provider)
	}
	return &providerLLMClient{
		provider: provider,
		model:    model,
		apiKey:   apiKeyForProvider(provider),
		client:   &http.Client{},
		temp:     opts.Temperature,
		timeout:  defaultLLMTimeout,
	}, provider, model, nil
}

func apiKeyForProvider(provider string) string {
	switch provider {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		return os.Getenv("OPENAI_API_KEY")
	case "gemini":
		return os.Getenv("GEMINI_API_KEY")
	case "copilot-api":
		if key := os.Getenv("COPILOT_API_KEY"); key != "" {
			return key
		}
		return "copilot-api"
	default:
		return ""
	}
}

type Intent struct {
	OpenAPI   string            `hcl:"openapi,optional" json:"openapi,omitempty"`
	ServerURL string            `hcl:"server_url,optional" json:"server_url,omitempty"`
	Workflow  *WorkflowMeta     `hcl:"workflow,block" json:"workflow,omitempty"`
	Inputs    []*Input          `hcl:"input,block" json:"inputs,omitempty"`
	Triggers  []*TriggerIntent  `hcl:"trigger,block" json:"triggers,omitempty"`
	Steps     []*Step           `hcl:"step,block" json:"steps,omitempty"`
	Security  []*SecurityIntent `hcl:"security,block" json:"security,omitempty"`
	Outputs   []*Output         `hcl:"output,block" json:"outputs,omitempty"`
	Locals    map[string]string `hcl:"locals,optional" json:"locals,omitempty"`
}

type WorkflowMeta struct {
	Name        string            `hcl:"name,optional" json:"name,omitempty"`
	Description string            `hcl:"description,optional" json:"description,omitempty"`
	Timeout     *float64          `hcl:"timeout,optional" json:"timeout,omitempty"`
	Idempotency *uws1.Idempotency `hcl:"idempotency,block" json:"idempotency,omitempty"`
}

type Input struct {
	Name        string `hcl:"name,label" json:"name,omitempty"`
	Type        string `hcl:"type,optional" json:"type,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
	Required    bool   `hcl:"required,optional" json:"required,omitempty"`
	Sensitive   bool   `hcl:"sensitive,optional" json:"sensitive,omitempty"`
	Default     string `hcl:"default,optional" json:"default,omitempty"`
}

type Step struct {
	Name            string                `hcl:"name,label" json:"name,omitempty"`
	Type            string                `hcl:"type,optional" json:"type,omitempty"`
	Do              string                `hcl:"do,optional" json:"do,omitempty"`
	Using           string                `hcl:"using,optional" json:"using,omitempty"`
	Set             string                `hcl:"set,optional" json:"set,omitempty"`
	When            string                `hcl:"when,optional" json:"when,omitempty"`
	ForEach         string                `hcl:"for_each,optional" json:"for_each,omitempty"`
	DependsOn       []string              `hcl:"depends_on,optional" json:"depends_on,omitempty"`
	With            map[string]string     `hcl:"with,optional" json:"with,omitempty"`
	Provider        string                `hcl:"provider,optional" json:"provider,omitempty"`
	OpenAPI         string                `hcl:"openapi,optional" json:"openapi,omitempty"`
	Operation       string                `hcl:"operation,optional" json:"operation,omitempty"`
	Timeout         *float64              `hcl:"timeout,optional" json:"timeout,omitempty"`
	Binds           []*StepBind           `hcl:"bind,block" json:"bind,omitempty"`
	Items           string                `hcl:"items,optional" json:"items,omitempty"`
	Mode            string                `hcl:"mode,optional" json:"mode,omitempty"`
	BatchSize       string                `hcl:"batch_size,optional" json:"batch_size,omitempty"`
	SuccessCriteria []*uws1.Criterion     `hcl:"successCriteria,block" json:"successCriteria,omitempty"`
	OnFailure       []*uws1.FailureAction `hcl:"onFailure,block" json:"onFailure,omitempty"`
	OnSuccess       []*uws1.SuccessAction `hcl:"onSuccess,block" json:"onSuccess,omitempty"`
	Steps           []*Step               `hcl:"step,block" json:"steps,omitempty"`
	Cases           []*StepCase           `hcl:"case,block" json:"cases,omitempty"`
	Default         *StepDefault          `hcl:"default,block" json:"default,omitempty"`
}

type StepBind struct {
	From   string            `hcl:"from" json:"from,omitempty"`
	Fields map[string]string `hcl:"fields,optional" json:"fields,omitempty"`
}

type StepCase struct {
	Name  string  `hcl:"name,label" json:"name,omitempty"`
	When  string  `hcl:"when,optional" json:"when,omitempty"`
	Steps []*Step `hcl:"step,block" json:"steps,omitempty"`
}

type StepDefault struct {
	Steps []*Step `hcl:"step,block" json:"steps,omitempty"`
}

type TriggerIntent struct {
	Name           string                `hcl:"name,label" json:"name,omitempty"`
	Path           string                `hcl:"path,optional" json:"path,omitempty"`
	Authentication string                `hcl:"authentication,optional" json:"authentication,omitempty"`
	Methods        []string              `hcl:"methods,optional" json:"methods,omitempty"`
	Options        map[string]string     `hcl:"options,optional" json:"options,omitempty"`
	Outputs        []string              `hcl:"outputs,optional" json:"outputs,omitempty"`
	Routes         []*TriggerRouteIntent `hcl:"route,block" json:"routes,omitempty"`
}

type TriggerRouteIntent struct {
	Output string   `hcl:"output,label" json:"output,omitempty"`
	To     []string `hcl:"to,optional" json:"to,omitempty"`
}

type SecurityIntent struct {
	Name        string `hcl:"name,label" json:"name,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
	TokenFrom   string `hcl:"token_from,optional" json:"token_from,omitempty"`
}

type Output struct {
	Name        string `hcl:"name,label" json:"name,omitempty"`
	From        string `hcl:"from" json:"from,omitempty"`
	Description string `hcl:"description,optional" json:"description,omitempty"`
}

func ParseIntentFile(path string) (*Intent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return ParseIntent(data, path)
}

func ParseIntent(data []byte, path string) (*Intent, error) {
	if strings.TrimSpace(path) == "" {
		path = IntentPath
	}
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(rewriteIntentHCLCompatibility(data), path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding HCL: %s", diags.Error())
	}
	var raw hclIntent
	diags = gohcl.DecodeBody(file.Body, nil, &raw)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding HCL: %s", diags.Error())
	}
	intent, err := raw.toIntent()
	if err != nil {
		return nil, err
	}
	if err := validateIntent(&intent); err != nil {
		return nil, err
	}
	return &intent, nil
}

type hclIntent struct {
	OpenAPI   string            `hcl:"openapi,optional" json:"openapi,omitempty"`
	ServerURL string            `hcl:"server_url,optional" json:"server_url,omitempty"`
	Workflow  *hclWorkflowMeta  `hcl:"workflow,block" json:"workflow,omitempty"`
	Inputs    []*Input          `hcl:"input,block" json:"inputs,omitempty"`
	Triggers  []*TriggerIntent  `hcl:"trigger,block" json:"triggers,omitempty"`
	Steps     []*hclStep        `hcl:"step,block" json:"steps,omitempty"`
	Security  []*SecurityIntent `hcl:"security,block" json:"security,omitempty"`
	Outputs   []*Output         `hcl:"output,block" json:"outputs,omitempty"`
	Locals    map[string]string `hcl:"locals,optional" json:"locals,omitempty"`
	Remain    hcl.Body          `hcl:",remain" json:"-"`
}

type hclWorkflowMeta struct {
	Name        string          `hcl:"name,optional" json:"name,omitempty"`
	Description string          `hcl:"description,optional" json:"description,omitempty"`
	Timeout     *float64        `hcl:"timeout,optional" json:"timeout,omitempty"`
	Idempotency *hclIdempotency `hcl:"idempotency,block" json:"idempotency,omitempty"`
}

type hclIdempotency struct {
	Key        string   `hcl:"key" json:"key,omitempty"`
	OnConflict string   `hcl:"onConflict,optional" json:"onConflict,omitempty"`
	TTL        *float64 `hcl:"ttl,optional" json:"ttl,omitempty"`
}

type hclStep struct {
	Name            string              `hcl:"name,label" json:"name,omitempty"`
	Type            string              `hcl:"type,optional" json:"type,omitempty"`
	Do              string              `hcl:"do,optional" json:"do,omitempty"`
	Using           string              `hcl:"using,optional" json:"using,omitempty"`
	Set             string              `hcl:"set,optional" json:"set,omitempty"`
	When            string              `hcl:"when,optional" json:"when,omitempty"`
	ForEach         string              `hcl:"for_each,optional" json:"for_each,omitempty"`
	DependsOn       []string            `hcl:"depends_on,optional" json:"depends_on,omitempty"`
	With            map[string]string   `hcl:"with,optional" json:"with,omitempty"`
	Provider        string              `hcl:"provider,optional" json:"provider,omitempty"`
	OpenAPI         string              `hcl:"openapi,optional" json:"openapi,omitempty"`
	Operation       string              `hcl:"operation,optional" json:"operation,omitempty"`
	Timeout         *float64            `hcl:"timeout,optional" json:"timeout,omitempty"`
	Binds           []*StepBind         `hcl:"bind,block" json:"bind,omitempty"`
	Items           string              `hcl:"items,optional" json:"items,omitempty"`
	Mode            string              `hcl:"mode,optional" json:"mode,omitempty"`
	BatchSize       string              `hcl:"batch_size,optional" json:"batch_size,omitempty"`
	SuccessCriteria []*hclCriterion     `hcl:"successCriteria,block" json:"successCriteria,omitempty"`
	OnFailure       []*hclFailureAction `hcl:"onFailure,block" json:"onFailure,omitempty"`
	OnSuccess       []*hclSuccessAction `hcl:"onSuccess,block" json:"onSuccess,omitempty"`
	Steps           []*hclStep          `hcl:"step,block" json:"steps,omitempty"`
	Cases           []*hclStepCase      `hcl:"case,block" json:"cases,omitempty"`
	Default         *hclStepDefault     `hcl:"default,block" json:"default,omitempty"`
	Remain          hcl.Body            `hcl:",remain" json:"-"`
}

type hclStepCase struct {
	Name  string     `hcl:"name,label" json:"name,omitempty"`
	When  string     `hcl:"when,optional" json:"when,omitempty"`
	Steps []*hclStep `hcl:"step,block" json:"steps,omitempty"`
}

type hclStepDefault struct {
	Steps []*hclStep `hcl:"step,block" json:"steps,omitempty"`
}

type hclCriterion struct {
	Condition string `hcl:"condition" json:"condition,omitempty"`
	Type      string `hcl:"type,optional" json:"type,omitempty"`
	Context   string `hcl:"context,optional" json:"context,omitempty"`
}

type hclFailureAction struct {
	Name       string          `hcl:"name,label" json:"name,omitempty"`
	Type       string          `hcl:"type" json:"type,omitempty"`
	WorkflowID string          `hcl:"workflowId,optional" json:"workflowId,omitempty"`
	StepID     string          `hcl:"stepId,optional" json:"stepId,omitempty"`
	RetryAfter float64         `hcl:"retryAfter,optional" json:"retryAfter,omitempty"`
	RetryLimit int             `hcl:"retryLimit,optional" json:"retryLimit,omitempty"`
	Criteria   []*hclCriterion `hcl:"criterion,block" json:"criteria,omitempty"`
}

type hclSuccessAction struct {
	Name       string          `hcl:"name,label" json:"name,omitempty"`
	Type       string          `hcl:"type" json:"type,omitempty"`
	WorkflowID string          `hcl:"workflowId,optional" json:"workflowId,omitempty"`
	StepID     string          `hcl:"stepId,optional" json:"stepId,omitempty"`
	Criteria   []*hclCriterion `hcl:"criterion,block" json:"criteria,omitempty"`
}

func (raw hclIntent) toIntent() (Intent, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return Intent{}, err
	}
	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return Intent{}, err
	}
	return intent, nil
}

func validateIntent(intent *Intent) error {
	if intent == nil {
		return fmt.Errorf("intent is required")
	}
	if intent.Workflow != nil {
		if err := validateTimeout(intent.Workflow.Timeout, "workflow.timeout"); err != nil {
			return err
		}
		if idem := intent.Workflow.Idempotency; idem != nil {
			if strings.TrimSpace(idem.Key) == "" {
				return fmt.Errorf("workflow.idempotency.key is required")
			}
			switch idem.OnConflict {
			case "", "reject", "returnPrevious":
			default:
				return fmt.Errorf("workflow.idempotency.onConflict must be reject or returnPrevious")
			}
			if idem.TTL != nil && *idem.TTL <= 0 {
				return fmt.Errorf("workflow.idempotency.ttl must be positive")
			}
		}
	}
	if len(intent.Steps) == 0 && len(intent.Triggers) == 0 {
		return fmt.Errorf("at least one step or trigger is required")
	}
	for i, step := range intent.Steps {
		if err := validateStep(step, fmt.Sprintf("step %d", i)); err != nil {
			return err
		}
	}
	for i, trigger := range intent.Triggers {
		if trigger == nil {
			continue
		}
		if strings.TrimSpace(trigger.Name) == "" {
			return fmt.Errorf("trigger %d: name label is required", i)
		}
		for routeIndex, route := range trigger.Routes {
			if route != nil && strings.TrimSpace(route.Output) == "" {
				return fmt.Errorf("trigger %d (%s) route %d: output label is required", i, trigger.Name, routeIndex)
			}
		}
	}
	return nil
}

func validateStep(step *Step, label string) error {
	if step == nil {
		return nil
	}
	if strings.TrimSpace(step.Name) == "" {
		return fmt.Errorf("%s: name label is required", label)
	}
	if err := validateTimeout(step.Timeout, label+".timeout"); err != nil {
		return err
	}
	for i, nested := range step.Steps {
		if err := validateStep(nested, fmt.Sprintf("%s.step %d", label, i)); err != nil {
			return err
		}
	}
	for i, branch := range step.Cases {
		if branch == nil {
			continue
		}
		if strings.TrimSpace(branch.Name) == "" {
			return fmt.Errorf("%s.case %d: name label is required", label, i)
		}
		for j, nested := range branch.Steps {
			if err := validateStep(nested, fmt.Sprintf("%s.case %s.step %d", label, branch.Name, j)); err != nil {
				return err
			}
		}
	}
	if step.Default != nil {
		for i, nested := range step.Default.Steps {
			if err := validateStep(nested, fmt.Sprintf("%s.default.step %d", label, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTimeout(value *float64, path string) error {
	if value != nil && *value <= 0 {
		return fmt.Errorf("%s must be positive", path)
	}
	return nil
}

var labelBindPattern = regexp.MustCompile(`(?m)^([ \t]*)bind\s+(?:"([^"]+)"|([A-Za-z_][A-Za-z0-9_-]*))\s*\{\s*$`)
var idempotencyAttrPattern = regexp.MustCompile(`(?m)^([ \t]*)idempotency\s*=\s*\{\s*$`)

func rewriteIntentHCLCompatibility(data []byte) []byte {
	return rewriteIdempotencyAttributeSyntax(rewriteLabelBindSyntax(data))
}

func rewriteLabelBindSyntax(data []byte) []byte {
	input := string(data)
	if !strings.Contains(input, "bind ") {
		return data
	}
	rewritten := labelBindPattern.ReplaceAllStringFunc(input, func(line string) string {
		match := labelBindPattern.FindStringSubmatch(line)
		if len(match) < 4 {
			return line
		}
		indent := match[1]
		label := strings.TrimSpace(match[2])
		if label == "" {
			label = strings.TrimSpace(match[3])
		}
		if label == "" {
			return line
		}
		return fmt.Sprintf("%sbind {\n%s  from = %q", indent, indent, label)
	})
	return []byte(rewritten)
}

func rewriteIdempotencyAttributeSyntax(data []byte) []byte {
	input := string(data)
	if !strings.Contains(input, "idempotency") {
		return data
	}
	return []byte(idempotencyAttrPattern.ReplaceAllString(input, `${1}idempotency {`))
}

func RenderIntentHCL(intent *Intent) (string, error) {
	if intent == nil {
		return "", fmt.Errorf("intent is required")
	}
	file := hclwrite.NewEmptyFile()
	body := file.Body()
	setAttrString(body, "openapi", intent.OpenAPI)
	setAttrString(body, "server_url", intent.ServerURL)
	if len(intent.Locals) > 0 {
		setAttrMap(body, "locals", intent.Locals, true)
	}
	if intent.Workflow != nil {
		block := body.AppendNewBlock("workflow", nil)
		wb := block.Body()
		setAttrString(wb, "name", intent.Workflow.Name)
		setAttrString(wb, "description", intent.Workflow.Description)
		setAttrFloatPtr(wb, "timeout", intent.Workflow.Timeout)
		if intent.Workflow.Idempotency != nil {
			addIdempotencyBlock(wb, intent.Workflow.Idempotency)
		}
	}
	for _, input := range intent.Inputs {
		if input == nil {
			continue
		}
		block := body.AppendNewBlock("input", []string{input.Name})
		ib := block.Body()
		setAttrString(ib, "type", input.Type)
		setAttrString(ib, "description", input.Description)
		setAttrBool(ib, "required", input.Required)
		setAttrBool(ib, "sensitive", input.Sensitive)
		setAttrString(ib, "default", input.Default)
	}
	for _, trigger := range intent.Triggers {
		addTriggerBlock(body, trigger)
	}
	for _, step := range intent.Steps {
		addStepBlock(body, step)
	}
	for _, sec := range intent.Security {
		if sec == nil {
			continue
		}
		block := body.AppendNewBlock("security", []string{sec.Name})
		sb := block.Body()
		setAttrString(sb, "description", sec.Description)
		setAttrString(sb, "token_from", sec.TokenFrom)
	}
	for _, output := range intent.Outputs {
		if output == nil {
			continue
		}
		block := body.AppendNewBlock("output", []string{output.Name})
		ob := block.Body()
		setAttrString(ob, "from", output.From)
		setAttrString(ob, "description", output.Description)
	}
	data := hclwrite.Format(file.Bytes())
	if _, err := ParseIntent(data, IntentPath); err != nil {
		return "", err
	}
	return string(data), nil
}

func addTriggerBlock(body *hclwrite.Body, trigger *TriggerIntent) {
	if trigger == nil {
		return
	}
	block := body.AppendNewBlock("trigger", []string{trigger.Name})
	tb := block.Body()
	setAttrString(tb, "path", trigger.Path)
	setAttrString(tb, "authentication", trigger.Authentication)
	setAttrList(tb, "methods", trigger.Methods)
	setAttrMap(tb, "options", trigger.Options, true)
	setAttrList(tb, "outputs", trigger.Outputs)
	for _, route := range trigger.Routes {
		if route == nil {
			continue
		}
		rb := tb.AppendNewBlock("route", []string{route.Output})
		setAttrList(rb.Body(), "to", route.To)
	}
}

func addStepBlock(body *hclwrite.Body, step *Step) {
	if step == nil {
		return
	}
	block := body.AppendNewBlock("step", []string{step.Name})
	sb := block.Body()
	setAttrString(sb, "type", step.Type)
	setAttrString(sb, "do", step.Do)
	setAttrString(sb, "using", step.Using)
	setAttrString(sb, "set", step.Set)
	setAttrString(sb, "when", step.When)
	setAttrString(sb, "for_each", step.ForEach)
	setAttrList(sb, "depends_on", step.DependsOn)
	setAttrMap(sb, "with", step.With, false)
	setAttrString(sb, "provider", step.Provider)
	setAttrString(sb, "openapi", step.OpenAPI)
	setAttrString(sb, "operation", step.Operation)
	setAttrFloatPtr(sb, "timeout", step.Timeout)
	setAttrString(sb, "items", step.Items)
	setAttrString(sb, "mode", step.Mode)
	setAttrString(sb, "batch_size", step.BatchSize)
	for _, bind := range step.Binds {
		addBindBlock(sb, bind)
	}
	for _, criterion := range step.SuccessCriteria {
		if criterion != nil {
			gohcl.EncodeIntoBody(criterion, sb.AppendNewBlock("successCriteria", nil).Body())
		}
	}
	for _, action := range step.OnFailure {
		if action != nil {
			gohcl.EncodeIntoBody(action, sb.AppendNewBlock("onFailure", nil).Body())
		}
	}
	for _, action := range step.OnSuccess {
		if action != nil {
			gohcl.EncodeIntoBody(action, sb.AppendNewBlock("onSuccess", nil).Body())
		}
	}
	for _, nested := range step.Steps {
		addStepBlock(sb, nested)
	}
	for _, branch := range step.Cases {
		if branch == nil {
			continue
		}
		cb := sb.AppendNewBlock("case", []string{branch.Name})
		setAttrString(cb.Body(), "when", branch.When)
		for _, nested := range branch.Steps {
			addStepBlock(cb.Body(), nested)
		}
	}
	if step.Default != nil {
		db := sb.AppendNewBlock("default", nil)
		for _, nested := range step.Default.Steps {
			addStepBlock(db.Body(), nested)
		}
	}
}

func addBindBlock(body *hclwrite.Body, bind *StepBind) {
	if bind == nil {
		return
	}
	block := body.AppendNewBlock("bind", nil)
	bb := block.Body()
	setAttrString(bb, "from", bind.From)
	setAttrMap(bb, "fields", bind.Fields, false)
}

func addIdempotencyBlock(body *hclwrite.Body, idem *uws1.Idempotency) {
	block := body.AppendNewBlock("idempotency", nil)
	ib := block.Body()
	setAttrString(ib, "key", idem.Key)
	setAttrString(ib, "onConflict", idem.OnConflict)
	setAttrFloatPtr(ib, "ttl", idem.TTL)
}

func setAttrString(body *hclwrite.Body, name, value string) {
	if strings.TrimSpace(value) != "" {
		body.SetAttributeValue(name, cty.StringVal(value))
	}
}

func setAttrBool(body *hclwrite.Body, name string, value bool) {
	if value {
		body.SetAttributeValue(name, cty.BoolVal(value))
	}
}

func setAttrFloatPtr(body *hclwrite.Body, name string, value *float64) {
	if value != nil {
		body.SetAttributeValue(name, cty.NumberFloatVal(*value))
	}
}

func setAttrList(body *hclwrite.Body, name string, values []string) {
	var out []cty.Value
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, cty.StringVal(value))
		}
	}
	if len(out) > 0 {
		body.SetAttributeValue(name, cty.ListVal(out))
	}
}

func setAttrMap(body *hclwrite.Body, name string, values map[string]string, sortKeys bool) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return
	}
	if sortKeys {
		sort.Strings(keys)
	}
	out := make(map[string]cty.Value, len(keys))
	for _, key := range keys {
		out[key] = cty.StringVal(values[key])
	}
	body.SetAttributeValue(name, cty.ObjectVal(out))
}

func ValidateHCL(content string) error {
	parser := hclparse.NewParser()
	_, diags := parser.ParseHCL([]byte(content), "workflow.hcl")
	if diags.HasErrors() {
		return fmt.Errorf("HCL validation error: %s", diags.Error())
	}
	return nil
}

func FormatHCL(content string) (string, error) {
	if err := ValidateHCL(content); err != nil {
		return "", err
	}
	return string(hclwrite.Format([]byte(content))), nil
}

func (intent *Intent) MissingSlots() []string {
	var missing []string
	if intent.missingDefaultOpenAPIContext() {
		missing = append(missing, "OpenAPI specification URL or content")
	}
	if len(intent.Steps) == 0 && len(intent.Triggers) == 0 {
		missing = append(missing, "At least one workflow step")
	}
	for i, step := range intent.Steps {
		if step != nil && stepRequiresDo(step) && step.Do == "" {
			missing = append(missing, fmt.Sprintf("Description for step %d", i+1))
		}
	}
	return missing
}

func (intent *Intent) RequiresOpenAPI() bool {
	if intent == nil {
		return false
	}
	if strings.TrimSpace(intent.OpenAPI) != "" {
		return true
	}
	required := false
	walkSteps(intent.Steps, func(step *Step) {
		if step != nil && !required && (strings.TrimSpace(step.OpenAPI) != "" || strings.TrimSpace(step.Operation) != "") {
			required = true
		}
	})
	return required
}

func (intent *Intent) missingDefaultOpenAPIContext() bool {
	if intent == nil || strings.TrimSpace(intent.OpenAPI) != "" {
		return false
	}
	missing := false
	walkSteps(intent.Steps, func(step *Step) {
		if step != nil && !missing && strings.TrimSpace(step.Operation) != "" && strings.TrimSpace(step.OpenAPI) == "" {
			missing = true
		}
	})
	return missing
}

func stepRequiresDo(step *Step) bool {
	if step == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(step.Type)) {
	case "sequence", "parallel", "switch", "merge", "loop", "await":
		return false
	default:
		return strings.TrimSpace(step.Operation) == ""
	}
}

func walkSteps(steps []*Step, fn func(*Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		fn(step)
		walkSteps(step.Steps, fn)
		for _, branch := range step.Cases {
			if branch != nil {
				walkSteps(branch.Steps, fn)
			}
		}
		if step.Default != nil {
			walkSteps(step.Default.Steps, fn)
		}
	}
}

func (intent *Intent) NormalizedForGeneration() *Intent {
	clone := intent.Clone()
	if clone == nil {
		return nil
	}
	for _, step := range clone.Steps {
		normalizeStepForGeneration(step)
	}
	clone.EnsureActionDescriptions()
	return clone
}

func (intent *Intent) EnsureActionDescriptions() {
	if intent == nil {
		return
	}
	ensureStepActionDescriptions(intent.Steps)
}

func ensureStepActionDescriptions(steps []*Step) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.Do) == "" {
			if op := strings.TrimSpace(step.Operation); op != "" {
				step.Do = "Run operation " + op + "."
			} else if typ := strings.TrimSpace(step.Type); typ != "" {
				step.Do = "Run " + typ + " step."
			}
		}
		ensureStepActionDescriptions(step.Steps)
		for _, branch := range step.Cases {
			if branch != nil {
				ensureStepActionDescriptions(branch.Steps)
			}
		}
		if step.Default != nil {
			ensureStepActionDescriptions(step.Default.Steps)
		}
	}
}

func (intent *Intent) ToPromptContext() string {
	var result string
	if intent == nil {
		return result
	}
	if intent.Workflow != nil {
		result += fmt.Sprintf("Workflow: %s\n", intent.Workflow.Name)
		if intent.Workflow.Description != "" {
			result += fmt.Sprintf("Description: %s\n", intent.Workflow.Description)
		}
		result += "\n"
	}
	if len(intent.Inputs) > 0 {
		result += "Inputs:\n"
		for _, input := range intent.Inputs {
			if input == nil {
				continue
			}
			req := ""
			if input.Required {
				req = " (required)"
			}
			result += fmt.Sprintf("  - %s: %s%s\n", input.Name, input.Type, req)
		}
		result += "\n"
	}
	result += "Steps:\n"
	for _, step := range intent.Steps {
		appendStepPrompt(&result, step, "  ")
	}
	if len(intent.Outputs) > 0 {
		result += "\nOutputs:\n"
		for _, out := range intent.Outputs {
			if out != nil {
				result += fmt.Sprintf("  - %s: from %s\n", out.Name, out.From)
			}
		}
	}
	return result
}

func appendStepPrompt(result *string, step *Step, indent string) {
	if step == nil {
		return
	}
	*result += fmt.Sprintf("%s- %s", indent, step.Name)
	if step.Type != "" {
		*result += fmt.Sprintf(" (%s)", step.Type)
	}
	if step.Do != "" {
		*result += fmt.Sprintf(": %s", step.Do)
	}
	*result += "\n"
	for _, nested := range step.Steps {
		appendStepPrompt(result, nested, indent+"  ")
	}
	for _, branch := range step.Cases {
		if branch != nil {
			for _, nested := range branch.Steps {
				appendStepPrompt(result, nested, indent+"  ")
			}
		}
	}
	if step.Default != nil {
		for _, nested := range step.Default.Steps {
			appendStepPrompt(result, nested, indent+"  ")
		}
	}
}

func (intent *Intent) Clone() *Intent {
	if intent == nil {
		return nil
	}
	data, _ := json.Marshal(intent)
	var clone Intent
	_ = json.Unmarshal(data, &clone)
	return &clone
}

func normalizeStepForGeneration(step *Step) {
	if step == nil {
		return
	}
	step.Type = normalizeIntentStepType(step.Type)
	applyStepBindHints(step)
	for _, nested := range step.Steps {
		normalizeStepForGeneration(nested)
	}
	for _, branch := range step.Cases {
		if branch != nil {
			for _, nested := range branch.Steps {
				normalizeStepForGeneration(nested)
			}
		}
	}
	if step.Default != nil {
		for _, nested := range step.Default.Steps {
			normalizeStepForGeneration(nested)
		}
	}
}

func normalizeIntentStepType(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "format", "formatter", "formatting", "process", "processing", "transform", "transformer", "mapping", "compose", "composition":
		return "fnct"
	default:
		return kind
	}
}

func applyStepBindHints(step *Step) {
	if step == nil || len(step.Binds) == 0 {
		return
	}
	if step.With == nil {
		step.With = map[string]string{}
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		from := strings.TrimSpace(bind.From)
		if from == "" {
			continue
		}
		step.DependsOn = appendUnique(step.DependsOn, from)
		for target, source := range bind.Fields {
			target = normalizeRequestFieldTarget(target)
			if target == "" {
				continue
			}
			if _, exists := step.With[target]; !exists {
				step.With[target] = bindFieldReference(from, target, source)
			}
		}
	}
}

func normalizeRequestFieldTarget(target string) string {
	target = strings.TrimSpace(target)
	if strings.HasPrefix(target, "payload.") {
		return strings.TrimPrefix(target, "payload.")
	}
	return target
}

func bindFieldReference(from, target, source string) string {
	source = strings.TrimSpace(source)
	if source == "" || source == "received_body" {
		return from + ".received_body." + leafName(target)
	}
	if strings.HasPrefix(source, from+".") {
		return source
	}
	return from + "." + source
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.TrimSpace(existing) == value {
			return values
		}
	}
	return append(values, value)
}

func leafName(path string) string {
	path = strings.Trim(path, ".")
	if idx := strings.LastIndex(path, "."); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

type OpenAPISpec struct {
	Title       string
	Version     string
	Description string
	ServerURL   string
	Operations  []*OperationInfo
	Security    []string
	RawSpec     map[string]any
}

type OpenAPISpecContext struct {
	Path         string
	ResolvedPath string
	Provider     string
	StepNames    []string
	Spec         *OpenAPISpec
}

type OperationInfo struct {
	OperationID string
	Method      string
	Path        string
	Summary     string
	Description string
	Parameters  []*ParameterInfo
	RequestBody *RequestBodyInfo
	Responses   map[string]*ResponseInfo
	Security    []string
	Tags        []string
}

type ParameterInfo struct {
	Name        string
	In          string
	Required    bool
	Type        string
	Description string
}

type RequestBodyInfo struct {
	Required    bool
	ContentType string
	Schema      map[string]any
}

type ResponseInfo struct {
	Description string
	ContentType string
	Schema      map[string]any
}

func LoadOpenAPISpec(path string) (*OpenAPISpec, error) {
	index, err := apitools.LoadOperationIndex(path)
	if err != nil {
		return nil, err
	}
	spec := &OpenAPISpec{Operations: []*OperationInfo{}}
	if len(index.Inventory.Documents) > 0 {
		doc := index.Inventory.Documents[0]
		spec.Title = doc.Title
		spec.Description = doc.Description
		spec.Version = firstNonEmpty(doc.OpenAPI, doc.Swagger)
	}
	for _, op := range index.Inventory.Operations {
		info := &OperationInfo{
			OperationID: op.OperationID,
			Method:      strings.ToUpper(op.Method),
			Path:        op.Path,
			Summary:     op.Summary,
			Description: op.Description,
			Responses:   map[string]*ResponseInfo{},
			Tags:        append([]string(nil), op.Tags...),
		}
		for _, param := range op.Parameters {
			info.Parameters = append(info.Parameters, &ParameterInfo{
				Name:        param.Name,
				In:          param.In,
				Required:    param.Required,
				Type:        param.Type,
				Description: param.Description,
			})
		}
		if op.RequestBody != nil {
			info.RequestBody = &RequestBodyInfo{
				Required:    op.RequestBody.Required,
				ContentType: firstNonEmpty(op.RequestBody.ContentTypes...),
				Schema:      schemaSummaryToMap(op.RequestBody.Schema),
			}
		}
		for _, security := range op.Security {
			if security.Name != "" {
				info.Security = append(info.Security, security.Name)
			}
		}
		spec.Operations = append(spec.Operations, info)
	}
	augmentOpenAPIResponses(path, spec)
	return spec, nil
}

func augmentOpenAPIResponses(path string, spec *OpenAPISpec) {
	if spec == nil {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return
	}
	root := normalizeYAMLMap(raw)
	rootMap := asMap(root)
	if spec.ServerURL == "" {
		if servers, ok := rootMap["servers"].([]any); ok && len(servers) > 0 {
			spec.ServerURL = asString(asMap(servers[0])["url"])
		}
		if spec.ServerURL == "" {
			spec.ServerURL = asString(rootMap["host"])
		}
	}
	paths := asMap(rootMap["paths"])
	if len(paths) == 0 {
		return
	}
	byID := map[string]*OperationInfo{}
	for _, op := range spec.Operations {
		if op != nil && strings.TrimSpace(op.OperationID) != "" {
			byID[strings.TrimSpace(op.OperationID)] = op
		}
	}
	for _, rawPathItem := range paths {
		pathItem := asMap(rawPathItem)
		for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options"} {
			rawOp := asMap(pathItem[method])
			if len(rawOp) == 0 {
				continue
			}
			op := byID[asString(rawOp["operationId"])]
			if op == nil {
				continue
			}
			if op.Responses == nil {
				op.Responses = map[string]*ResponseInfo{}
			}
			for code, rawResp := range asMap(rawOp["responses"]) {
				resp := asMap(rawResp)
				info := &ResponseInfo{Description: asString(resp["description"])}
				if schema := firstResponseSchema(resp); len(schema) > 0 {
					info.Schema = schema
					info.ContentType = "application/json"
				}
				op.Responses[fmt.Sprint(code)] = info
			}
		}
	}
}

func firstResponseSchema(resp map[string]any) map[string]any {
	if schema := asMap(resp["schema"]); len(schema) > 0 {
		return schema
	}
	content := asMap(resp["content"])
	if len(content) == 0 {
		return nil
	}
	if jsonMedia := asMap(content["application/json"]); len(jsonMedia) > 0 {
		return asMap(jsonMedia["schema"])
	}
	for _, media := range content {
		if schema := asMap(asMap(media)["schema"]); len(schema) > 0 {
			return schema
		}
	}
	return nil
}

func normalizeYAMLMap(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = normalizeYAMLMap(child)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[fmt.Sprint(key)] = normalizeYAMLMap(child)
		}
		return out
	case []any:
		for i, child := range typed {
			typed[i] = normalizeYAMLMap(child)
		}
		return typed
	default:
		return value
	}
}

func asMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func schemaSummaryToMap(schema *apitools.SchemaSummary) map[string]any {
	if schema == nil {
		return nil
	}
	out := map[string]any{}
	if schema.Type != "" {
		out["type"] = schema.Type
	}
	if schema.Format != "" {
		out["format"] = schema.Format
	}
	if schema.Ref != "" {
		out["$ref"] = schema.Ref
	}
	if schema.Description != "" {
		out["description"] = schema.Description
	}
	if len(schema.Properties) > 0 {
		props := map[string]any{}
		for _, prop := range schema.Properties {
			props[prop.Name] = map[string]any{"type": prop.Type, "description": prop.Description}
		}
		out["properties"] = props
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func documentSourceName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = regexp.MustCompile(`[^A-Za-z0-9_]+`).ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "openapi"
	}
	return name
}

func WorkflowFlow() authoring.Flow[*Intent] {
	adapter := WorkflowAdapter{}
	return authoring.Flow[*Intent]{
		Parser:       adapter,
		Renderer:     adapter,
		Validator:    adapter,
		SlotProvider: adapter,
	}
}

func ParseFile(ctx context.Context, path string) (*Intent, error) {
	draft, diagnostics, err := WorkflowAdapter{}.ParseIntent(ctx, authoring.Artifact{Path: path})
	if err != nil {
		return nil, err
	}
	if authoring.HasErrors(diagnostics) {
		return nil, authoring.DiagnosticError{Diagnostics: diagnostics}
	}
	return draft, nil
}

func RenderHCL(ctx context.Context, draft *Intent) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	flow := WorkflowFlow()
	diagnostics := flow.Validator.ValidateIntent(ctx, draft)
	if authoring.HasErrors(diagnostics) {
		return "", authoring.DiagnosticError{Diagnostics: diagnostics}
	}
	set, renderDiagnostics, err := flow.Renderer.RenderIntent(ctx, draft)
	diagnostics = append(diagnostics, renderDiagnostics...)
	if err != nil {
		return "", err
	}
	if authoring.HasErrors(diagnostics) {
		return "", authoring.DiagnosticError{Diagnostics: diagnostics}
	}
	for _, artifact := range set.Artifacts {
		if artifact.Path == IntentPath || artifact.MediaType == "text/hcl" {
			return string(artifact.Content), nil
		}
	}
	return "", fmt.Errorf("rendered intent artifact %q not found", IntentPath)
}

func ValidateComplete(ctx context.Context, draft *Intent) []authoring.Diagnostic {
	adapter := WorkflowAdapter{}
	diagnostics := adapter.ValidateIntent(ctx, draft)
	for _, slot := range adapter.MissingSlots(ctx, draft) {
		if !slot.Required {
			continue
		}
		diagnostics = append(diagnostics, authoring.Diagnostic{
			Severity: "error",
			Code:     "missing_slot",
			Message:  fmt.Sprintf("required slot %q is missing", slot.Name),
			Path:     slot.Name,
		})
	}
	return diagnostics
}

type WorkflowAdapter struct{}

func (WorkflowAdapter) ParseIntent(ctx context.Context, artifact authoring.Artifact) (*Intent, []authoring.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	path := strings.TrimSpace(artifact.Path)
	data := artifact.Content
	if len(data) == 0 && path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, []authoring.Diagnostic{diagnostic("error", "intent_read_failed", err.Error(), path)}, err
		}
	}
	if path == "" {
		path = IntentPath
	}
	draft, err := ParseIntent(data, path)
	if err != nil {
		return nil, []authoring.Diagnostic{diagnostic("error", "intent_parse_failed", err.Error(), path)}, err
	}
	return draft, nil, nil
}

func (WorkflowAdapter) RenderIntent(ctx context.Context, draft *Intent) (authoring.ArtifactSet, []authoring.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return authoring.ArtifactSet{}, nil, err
	}
	hcl, err := RenderIntentHCL(draft)
	if err != nil {
		return authoring.ArtifactSet{}, []authoring.Diagnostic{diagnostic("error", "intent_render_failed", err.Error(), IntentPath)}, err
	}
	return authoring.ArtifactSet{Artifacts: []authoring.Artifact{{
		Path:      IntentPath,
		MediaType: "text/hcl",
		Content:   []byte(hcl),
	}}}, nil, nil
}

func (WorkflowAdapter) ValidateIntent(ctx context.Context, draft *Intent) []authoring.Diagnostic {
	if err := ctx.Err(); err != nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_validation_cancelled", err.Error(), "")}
	}
	if draft == nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_required", "intent is required", "")}
	}
	hcl, err := RenderIntentHCL(draft)
	if err != nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_render_failed", err.Error(), IntentPath)}
	}
	if _, err := ParseIntent([]byte(hcl), IntentPath); err != nil {
		return []authoring.Diagnostic{diagnostic("error", "intent_parse_failed", err.Error(), IntentPath)}
	}
	return nil
}

func (WorkflowAdapter) MissingSlots(ctx context.Context, draft *Intent) []authoring.Slot {
	_ = ctx
	var missing []string
	if draft == nil {
		return []authoring.Slot{{Name: "intent", Required: true}}
	}
	if draft.Workflow == nil || strings.TrimSpace(draft.Workflow.Name) == "" {
		missing = append(missing, "workflow name")
	}
	if draft.Workflow == nil || strings.TrimSpace(draft.Workflow.Description) == "" {
		missing = append(missing, "workflow goal")
	}
	missing = append(missing, draft.MissingSlots()...)
	collectOperationSlots(&missing, draft.OpenAPI, draft.Steps)
	if len(draft.Outputs) == 0 {
		missing = append(missing, "at least one output")
	}
	missing = dedupe(missing)
	slots := make([]authoring.Slot, 0, len(missing))
	for _, name := range missing {
		slots = append(slots, authoring.Slot{Name: name, Required: true})
	}
	return slots
}

type ChatAdapter struct {
	Client      ChatClient
	Temperature *float64
	MaxTokens   int
}

func (adapter ChatAdapter) Complete(ctx context.Context, transcript []authoring.TranscriptTurn) (authoring.TranscriptTurn, error) {
	if adapter.Client == nil {
		return authoring.TranscriptTurn{}, fmt.Errorf("chat client is required")
	}
	content, err := adapter.Client.Chat(ctx, TranscriptToMessages(transcript))
	if err != nil {
		return authoring.TranscriptTurn{}, err
	}
	return authoring.TranscriptTurn{Role: "assistant", Content: content}, nil
}

func (adapter ChatAdapter) CompleteStructured(ctx context.Context, transcript []authoring.TranscriptTurn, schema any, out any) error {
	if adapter.Client == nil {
		return fmt.Errorf("chat client is required")
	}
	structured, ok := adapter.Client.(StructuredChat)
	if !ok {
		return fmt.Errorf("structured chat unavailable")
	}
	rawSchema, err := authoring.RawSchema(schema)
	if err != nil {
		return err
	}
	raw, err := structured.StructuredChat(ctx, TranscriptToMessages(transcript), rawSchema, StructuredOpts{
		Temperature: adapter.Temperature,
		MaxTokens:   adapter.MaxTokens,
	})
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}

func TranscriptToMessages(transcript []authoring.TranscriptTurn) []ChatMessage {
	messages := make([]ChatMessage, 0, len(transcript))
	for _, turn := range transcript {
		messages = append(messages, ChatMessage{Role: turn.Role, Content: turn.Content})
	}
	return messages
}

func MessagesToTranscript(messages []ChatMessage) []authoring.TranscriptTurn {
	transcript := make([]authoring.TranscriptTurn, 0, len(messages))
	for _, message := range messages {
		transcript = append(transcript, authoring.TranscriptTurn{Role: message.Role, Content: message.Content})
	}
	return transcript
}

func collectOperationSlots(missing *[]string, defaultOpenAPI string, steps []*Step) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		stepOpenAPI := firstNonEmpty(step.OpenAPI, defaultOpenAPI)
		stepType := strings.ToLower(strings.TrimSpace(step.Type))
		if (stepType == "http" || stepType == "openapi") && strings.TrimSpace(stepOpenAPI) != "" && strings.TrimSpace(step.Operation) == "" {
			*missing = append(*missing, "operation for step "+firstNonEmpty(step.Name, "unnamed"))
		}
		collectOperationSlots(missing, stepOpenAPI, step.Steps)
		for _, branch := range step.Cases {
			if branch != nil {
				collectOperationSlots(missing, stepOpenAPI, branch.Steps)
			}
		}
		if step.Default != nil {
			collectOperationSlots(missing, stepOpenAPI, step.Default.Steps)
		}
	}
}

func diagnostic(severity, code, message, path string) authoring.Diagnostic {
	return authoring.Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  strings.TrimSpace(message),
		Path:     path,
	}
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

var _ = hcl.DiagError
