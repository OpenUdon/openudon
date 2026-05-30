package authoring

import (
	"context"
	"io"

	sharedicot "github.com/OpenUdon/authoring/icot"
	publicprompt "github.com/OpenUdon/authoring/prompt"
)

// PromptTurn records one local prompt and answer.
type PromptTurn = sharedicot.PromptTurn

// PromptEvent records a structured event from an interactive authoring loop.
type PromptEvent = sharedicot.Event

// PromptTranscript is a persisted local transcript for replay and review.
type PromptTranscript = sharedicot.PromptTranscript

// ReplayScript is a deterministic prompt replay fixture.
type ReplayScript struct {
	Turns []PromptTurn `json:"turns"`
	Input string       `json:"input"`
}

// PromptSession prompts on a reader/writer pair and records prompt turns.
type PromptSession = sharedicot.PromptSession

// PromptDefaultMode controls how prompt defaults are handled.
type PromptDefaultMode = publicprompt.DefaultMode

const (
	// PromptDefaultsAsk prints defaulted prompts and waits for user input.
	PromptDefaultsAsk = publicprompt.DefaultsAsk
	// PromptDefaultsShow prints defaulted prompts and accepts their defaults.
	PromptDefaultsShow = publicprompt.DefaultsShow
	// PromptDefaultsSilent accepts defaulted prompts without printing them.
	PromptDefaultsSilent = publicprompt.DefaultsSilent
)

// NewPromptSession creates a local prompt session.
func NewPromptSession(in io.Reader, out io.Writer) *PromptSession {
	return sharedicot.NewPromptSession(in, out)
}

// OneLine normalizes a prompt default for display.
func OneLine(value string) string {
	return sharedicot.OneLine(value)
}

// AssertPromptLabelsInOrder verifies that prompt labels were emitted in replay
// order.
func AssertPromptLabelsInOrder(output string, turns []PromptTurn) error {
	return sharedicot.AssertPromptLabelsInOrder(output, turns)
}

// SavePromptTranscript writes a prompt transcript with private-file
// permissions. Empty paths are ignored.
func SavePromptTranscript(path, version string, turns []PromptTurn, events []PromptEvent, session any) error {
	return sharedicot.SavePromptTranscript(path, version, turns, events, session)
}

// InteractiveDraftRequest is the model-facing input for an interactive draft.
type InteractiveDraftRequest[S, D any] = sharedicot.DraftRequest[S, D]

// InteractiveExtractor provides optional AI assistance for an interactive
// authoring loop.
type InteractiveExtractor[S, D any] = sharedicot.Extractor[S, D]

// NoopInteractiveExtractor disables AI assistance.
type NoopInteractiveExtractor[S, D any] = sharedicot.NoopExtractor[S, D]

// ProgressiveLoopHooks supplies product-specific behavior for the generic iCoT
// loop.
type ProgressiveLoopHooks[S, D, A any] = sharedicot.InteractiveHooks[S, D, A]

// RunProgressiveICOT runs the domain-neutral progressive iCoT control loop.
func RunProgressiveICOT[S, D, A any](ctx context.Context, in io.Reader, out io.Writer, hooks ProgressiveLoopHooks[S, D, A]) (A, error) {
	return sharedicot.RunInteractive(ctx, in, out, hooks)
}

// ErrCanceled reports user cancellation from a generic interactive loop.
var ErrCanceled = sharedicot.ErrCanceled
