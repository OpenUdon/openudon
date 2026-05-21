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
	"regexp"
	"strings"
	"time"
)

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

type requestParameter string

const (
	requestParameterMaxOutputTokens requestParameter = "max_output_tokens"
	requestParameterMaxTokens       requestParameter = "max_tokens"
	requestParameterResponseFormat  requestParameter = "response_format"
	requestParameterTemperature     requestParameter = "temperature"
	requestParameterTextFormat      requestParameter = "text.format"
	requestParameterToolChoice      requestParameter = "tool_choice"
	requestParameterTools           requestParameter = "tools"
)

var unsupportedParameterPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)unsupported parameter:\s*['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?i)unknown parameter:\s*['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?i)unsupported field:\s*['"]([^'"]+)['"]`),
	regexp.MustCompile(`(?i)unknown field:\s*['"]([^'"]+)['"]`),
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
	c.addTemperature(payload, c.temp)
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
	c.addTemperature(payload, structuredTemperature(c.temp, opts))
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
	c.addTemperature(payload, c.temp)
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
	c.addTemperature(payload, structuredTemperature(c.temp, opts))
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
	if c.supportsRequestParameter(requestParameterTemperature) && c.temp != nil {
		payload["generationConfig"] = map[string]any{string(requestParameterTemperature): *c.temp}
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
	if temp := structuredTemperature(c.temp, opts); temp != nil && c.supportsRequestParameter(requestParameterTemperature) {
		payload["generationConfig"].(map[string]any)[string(requestParameterTemperature)] = *temp
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
	c.addTemperature(payload, c.temp)
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
	c.addTemperature(payload, structuredTemperature(c.temp, opts))
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

func (c *providerLLMClient) addTemperature(payload map[string]any, value *float64) {
	if value == nil || !c.supportsRequestParameter(requestParameterTemperature) {
		return
	}
	payload[string(requestParameterTemperature)] = *value
}

func (c *providerLLMClient) supportsRequestParameter(parameter requestParameter) bool {
	switch parameter {
	case requestParameterTemperature:
		return !c.usesFixedSamplingResponsesModel()
	default:
		return true
	}
}

func (c *providerLLMClient) usesFixedSamplingResponsesModel() bool {
	return c.provider == "copilot-api" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.model)), "gpt-5")
}

func (c *providerLLMClient) openAIEndpoint() string {
	if c.provider == "copilot-api" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.model)), "gpt-5") {
		return "responses"
	}
	return "chat_completions"
}

func (c *providerLLMClient) postJSON(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	currentPayload := payload
	removedParameters := map[string]bool{}
	for {
		statusCode, status, body, err := c.postJSONOnce(ctx, url, headers, currentPayload, out)
		if err != nil {
			return err
		}
		if statusCode >= 200 && statusCode < 300 {
			return nil
		}
		nextPayload, removed := retryPayloadWithoutUnsupportedParameter(currentPayload, body, removedParameters)
		if removed == "" {
			return fmt.Errorf("%s returned %s: %s", c.provider, status, strings.TrimSpace(string(body)))
		}
		removedParameters[removed] = true
		currentPayload = nextPayload
	}
}

func (c *providerLLMClient) postJSONOnce(ctx context.Context, url string, headers map[string]string, payload any, out any) (int, string, []byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, "", nil, err
	}
	ctx, cancel := contextWithLLMDeadline(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return 0, "", nil, err
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
		return 0, "", nil, redactedLLMTransportError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, resp.Status, body, nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, resp.Status, body, err
	}
	return resp.StatusCode, resp.Status, body, nil
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

func retryPayloadWithoutUnsupportedParameter(payload any, body []byte, alreadyRemoved map[string]bool) (any, string) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, ""
	}
	for _, parameter := range unsupportedRequestParameters(string(body)) {
		if alreadyRemoved[parameter] || !optionalRequestParameter(parameter) {
			continue
		}
		next := clonePayloadMap(root)
		if removeRequestParameter(next, parameter) {
			return next, parameter
		}
	}
	return nil, ""
}

func unsupportedRequestParameters(message string) []string {
	seen := map[string]bool{}
	var out []string
	for _, pattern := range unsupportedParameterPatterns {
		for _, match := range pattern.FindAllStringSubmatch(message, -1) {
			if len(match) < 2 {
				continue
			}
			parameter := strings.TrimSpace(match[1])
			if parameter == "" || seen[parameter] {
				continue
			}
			seen[parameter] = true
			out = append(out, parameter)
		}
	}
	return out
}

func optionalRequestParameter(parameter string) bool {
	switch requestParameter(strings.TrimSpace(parameter)) {
	case requestParameterMaxOutputTokens,
		requestParameterMaxTokens,
		requestParameterResponseFormat,
		requestParameterTemperature,
		requestParameterTextFormat,
		requestParameterToolChoice,
		requestParameterTools,
		"generationConfig.maxOutputTokens",
		"generationConfig.responseJsonSchema",
		"generationConfig.responseMimeType",
		"generationConfig.temperature":
		return true
	default:
		return false
	}
}

func removeRequestParameter(root map[string]any, parameter string) bool {
	parameter = strings.TrimSpace(parameter)
	if parameter == "" {
		return false
	}
	if _, ok := root[parameter]; ok {
		delete(root, parameter)
		return true
	}
	parts := strings.Split(parameter, ".")
	if len(parts) < 2 {
		return false
	}
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			return false
		}
		current = next
	}
	leaf := parts[len(parts)-1]
	if _, ok := current[leaf]; !ok {
		return false
	}
	delete(current, leaf)
	return true
}

func clonePayloadMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = clonePayloadValue(value)
	}
	return out
}

func clonePayloadValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return clonePayloadMap(typed)
	case []map[string]string:
		out := make([]map[string]string, len(typed))
		for i, item := range typed {
			copied := make(map[string]string, len(item))
			for key, value := range item {
				copied[key] = value
			}
			out[i] = copied
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for i, item := range typed {
			out[i] = clonePayloadMap(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = clonePayloadValue(item)
		}
		return out
	default:
		return value
	}
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
		provider = strings.ToLower(strings.TrimSpace(os.Getenv("OPENUDON_LLM_PROVIDER")))
	}
	if provider == "" {
		provider = "copilot-api"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(os.Getenv("OPENUDON_LLM_MODEL"))
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
