package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OpenUdon/authoring/structured"
	publictranscript "github.com/OpenUdon/authoring/transcript"
)

const (
	JSONCompletionModeStructured = "structured"
	JSONCompletionModeLegacy     = "legacy"
)

// JSONCompletionOptions configures structured-chat fallback behavior.
type JSONCompletionOptions struct {
	LegacyInstruction           string
	FallbackOnStructuredError   bool
	DisableStructuredCompletion bool
}

// JSONCompletionResult reports which completion path was used.
type JSONCompletionResult struct {
	Mode string `json:"mode,omitempty"`
	Raw  string `json:"raw,omitempty"`
}

// CompleteJSONWithFallback tries structured completion first, then optionally
// falls back to legacy chat plus JSON extraction.
func CompleteJSONWithFallback(ctx context.Context, client ChatClient, transcript []TranscriptTurn, schema any, out any, opts JSONCompletionOptions) (JSONCompletionResult, error) {
	if client == nil {
		return JSONCompletionResult{}, fmt.Errorf("chat client is required")
	}
	result, err := structured.CompleteJSON(ctx, chatAdapter{client: client}, toPublicTurns(transcript), schema, out, structured.Options{
		LegacyInstruction:           opts.LegacyInstruction,
		FallbackOnStructuredError:   opts.FallbackOnStructuredError,
		DisableStructuredCompletion: opts.DisableStructuredCompletion,
	})
	if err != nil {
		return JSONCompletionResult{Mode: result.Mode, Raw: result.Raw}, err
	}
	return JSONCompletionResult{Mode: result.Mode, Raw: result.Raw}, nil
}

// ExtractJSONBlock extracts a JSON object from a raw model response.
func ExtractJSONBlock(response string) (string, error) {
	raw, err := structured.ExtractJSONBlock(response)
	return string(raw), err
}

// DecodeJSONBlock extracts and decodes a JSON object from a model response.
func DecodeJSONBlock(raw string, target any) error {
	return structured.DecodeJSONBlock(raw, target)
}

// AppendLegacyJSONInstruction appends a JSON-only instruction to the last user
// turn if one is not already present.
func AppendLegacyJSONInstruction(transcript []TranscriptTurn, instruction string) []TranscriptTurn {
	return fromPublicTurns(structured.AppendLegacyJSONInstruction(toPublicTurns(transcript), instruction))
}

// RenderTranscriptSnapshot renders a markdown-ish transcript snapshot.
func RenderTranscriptSnapshot(transcript []TranscriptTurn) string {
	var b strings.Builder
	for _, turn := range transcript {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", strings.ToUpper(strings.TrimSpace(turn.Role)), strings.TrimSpace(turn.Content))
	}
	return strings.TrimSpace(b.String())
}

// RawSchema normalizes supported schema inputs to JSON bytes.
func RawSchema(schema any) (json.RawMessage, error) {
	return structured.NormalizeSchema(schema)
}

type chatAdapter struct {
	client ChatClient
}

func (adapter chatAdapter) Complete(ctx context.Context, turns []publictranscript.Turn) (publictranscript.Turn, error) {
	reply, err := adapter.client.Complete(ctx, fromPublicTurns(turns))
	if err != nil {
		return publictranscript.Turn{}, err
	}
	return publictranscript.Turn{Role: reply.Role, Content: reply.Content}, nil
}

func (adapter chatAdapter) CompleteStructured(ctx context.Context, turns []publictranscript.Turn, schema json.RawMessage, out any) error {
	structuredClient, ok := adapter.client.(StructuredChatClient)
	if !ok {
		return fmt.Errorf("structured chat client is not available")
	}
	return structuredClient.CompleteStructured(ctx, fromPublicTurns(turns), schema, out)
}

func toPublicTurns(turns []TranscriptTurn) []publictranscript.Turn {
	out := make([]publictranscript.Turn, 0, len(turns))
	for _, turn := range turns {
		out = append(out, publictranscript.Turn{
			Role:    turn.Role,
			Content: turn.Content,
		})
	}
	return out
}

func fromPublicTurns(turns []publictranscript.Turn) []TranscriptTurn {
	out := make([]TranscriptTurn, 0, len(turns))
	for _, turn := range turns {
		out = append(out, TranscriptTurn{
			Role:    turn.Role,
			Content: turn.Content,
		})
	}
	return out
}
