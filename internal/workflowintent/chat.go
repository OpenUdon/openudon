package workflowintent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/genelet/ramen/internal/authoring"
)

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
