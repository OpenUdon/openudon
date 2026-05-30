package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	publicprompt "github.com/OpenUdon/authoring/prompt"
	publicsession "github.com/OpenUdon/authoring/session"
	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
)

// PromptTurn records one local prompt and answer.
type PromptTurn struct {
	Label  string `json:"label"`
	Answer string `json:"answer"`
}

// PromptEvent records a structured event from an interactive authoring loop.
type PromptEvent struct {
	Kind string `json:"kind"`
	Data any    `json:"data,omitempty"`
}

// PromptTranscript is a persisted local transcript for replay and review.
type PromptTranscript struct {
	Version string        `json:"version"`
	TimeUTC string        `json:"time_utc"`
	Turns   []PromptTurn  `json:"turns"`
	Events  []PromptEvent `json:"events,omitempty"`
	Session any           `json:"session,omitempty"`
}

// ReplayScript is a deterministic prompt replay fixture.
type ReplayScript struct {
	Turns []PromptTurn `json:"turns"`
	Input string       `json:"input"`
}

// PromptSession prompts on a reader/writer pair and records prompt turns.
type PromptSession struct {
	session *publicprompt.Session
}

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
	return &PromptSession{session: publicprompt.NewSession(in, out)}
}

// SetDefaultMode controls whether defaulted prompts ask, auto-accept visibly,
// or auto-accept silently. Required free-form prompts still ask.
func (session *PromptSession) SetDefaultMode(mode PromptDefaultMode) {
	if session == nil {
		return
	}
	session.session.SetDefaultMode(mode)
}

// Ask prompts for a required free-form value.
func (session *PromptSession) Ask(label string) (string, error) {
	return session.session.Ask(label)
}

// AskDefault prompts for a value, returning current when the answer is blank.
func (session *PromptSession) AskDefault(label, current string) (string, error) {
	return session.session.AskDefault(label, current)
}

// AskDefaultForced prints a defaulted prompt and waits for user input even when
// the session default mode would normally auto-accept the default.
func (session *PromptSession) AskDefaultForced(label, current string) (string, error) {
	return session.session.AskDefaultForced(label, current)
}

// AskOptionalDefault prompts for an optional value, allowing automatic default
// acceptance even when the current value is blank.
func (session *PromptSession) AskOptionalDefault(label, current string) (string, error) {
	return session.session.AskOptionalDefault(label, current)
}

// AskDefaultRequired prompts until a non-empty value is available.
func (session *PromptSession) AskDefaultRequired(label, current string) (string, error) {
	return session.session.AskDefaultRequired(label, current)
}

// AskYesNo prompts for a yes/no answer with a default.
func (session *PromptSession) AskYesNo(label string, defaultYes bool) (bool, error) {
	return session.session.AskYesNo(label, defaultYes)
}

// Turns returns a copy of recorded prompt turns.
func (session *PromptSession) Turns() []PromptTurn {
	if session == nil {
		return nil
	}
	return fromPublicPromptTurns(session.session.Turns())
}

// OneLine normalizes a prompt default for display.
func OneLine(value string) string {
	return publicprompt.OneLine(value)
}

func fromPublicPromptTurns(turns []publicsession.PromptTurn) []PromptTurn {
	out := make([]PromptTurn, 0, len(turns))
	for _, turn := range turns {
		out = append(out, PromptTurn{Label: turn.Label, Answer: turn.Answer})
	}
	return out
}

// AssertPromptLabelsInOrder verifies that prompt labels were emitted in replay
// order.
func AssertPromptLabelsInOrder(output string, turns []PromptTurn) error {
	offset := 0
	for _, turn := range turns {
		index := strings.Index(output[offset:], turn.Label)
		if index < 0 {
			return fmt.Errorf("prompt label %q not found after offset %d", turn.Label, offset)
		}
		offset += index + len(turn.Label)
	}
	return nil
}

// SavePromptTranscript writes a prompt transcript with private-file
// permissions. Empty paths are ignored.
func SavePromptTranscript(path, version string, turns []PromptTurn, events []PromptEvent, session any) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if strings.TrimSpace(version) == "" {
		version = "openudon.prompt-transcript.v1"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	transcript := PromptTranscript{
		Version: version,
		TimeUTC: time.Now().UTC().Format(time.RFC3339),
		Turns:   append([]PromptTurn(nil), turns...),
		Events:  append([]PromptEvent(nil), events...),
		Session: session,
	}
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.Write(path, data, 0o600)
}

// InteractiveDraftRequest is the model-facing input for an interactive draft.
type InteractiveDraftRequest[S, D any] struct {
	Opening           string           `json:"opening"`
	Brief             string           `json:"brief,omitempty"`
	Session           S                `json:"session"`
	Docs              []D              `json:"docs"`
	TranscriptTurns   []PromptTurn     `json:"transcript_turns,omitempty"`
	ReadinessFeedback []ReadinessIssue `json:"readiness_feedback,omitempty"`
}

// InteractiveExtractor provides optional AI assistance for an interactive
// authoring loop.
type InteractiveExtractor[S, D any] interface {
	Kickoff(context.Context, string) (S, error)
	Draft(context.Context, InteractiveDraftRequest[S, D]) (S, error)
	Refine(context.Context, S) (S, error)
	Disambiguate(context.Context, string, []D) ([]string, error)
}

// NoopInteractiveExtractor disables AI assistance.
type NoopInteractiveExtractor[S, D any] struct{}

func (NoopInteractiveExtractor[S, D]) noopInteractiveExtractor() {}

func (NoopInteractiveExtractor[S, D]) Kickoff(context.Context, string) (S, error) {
	var zero S
	return zero, nil
}

func (NoopInteractiveExtractor[S, D]) Draft(context.Context, InteractiveDraftRequest[S, D]) (S, error) {
	var zero S
	return zero, nil
}

func (NoopInteractiveExtractor[S, D]) Refine(_ context.Context, session S) (S, error) {
	return session, nil
}

func (NoopInteractiveExtractor[S, D]) Disambiguate(context.Context, string, []D) ([]string, error) {
	return nil, nil
}

// ProgressiveLoopHooks supplies product-specific behavior for the generic iCoT
// loop.
type ProgressiveLoopHooks[S, D, A any] struct {
	Session       S
	Documents     []D
	Opening       string
	Brief         string
	NoLLM         bool
	DefaultMode   PromptDefaultMode
	MaxAttempts   int
	OpeningPrompt string

	Extractor InteractiveExtractor[S, D]

	Normalize            func(*S)
	ApplyOpeningAnswer   func(*S, string, []D) error
	OpeningEvents        func(S) []PromptEvent
	Autosave             func(S) error
	RankDocuments        func([]D, []string) []D
	DeterministicPrefill func(*S, []D) bool
	LooksLikeSession     func(S) bool
	MergeDraft           func(S, S, []D) S
	AfterDraft           func(S) error
	DraftResultSummary   func(S) any
	DraftEvents          func(S) []PromptEvent
	OnDraftError         func(error)
	RefreshDocuments     func(S, []D) ([]D, error)
	ShouldDraft          func(S, []D, []ReadinessIssue) bool
	ShouldDraftQuestion  func(S, []D, []ReadinessIssue, InteractiveQuestion) bool
	DraftQuestion        func(context.Context, *S, []D, []ReadinessIssue, InteractiveQuestion) (bool, error)
	CheckReadiness       func(S, []D) []ReadinessIssue
	Ready                func(S, []ReadinessIssue) bool
	PlanQuestion         func(S, []D, []ReadinessIssue) InteractiveQuestion
	ApplyAnswer          func(*S, InteractiveQuestion, string, []D) error
	FinalConfirm         func(*PromptSession, *S, []D, *[]PromptEvent) (A, error)
	FinalResultSummary   func(A) any
	SaveTranscript       func([]PromptTurn, []PromptEvent, A) error
}

// RunProgressiveICOT runs the domain-neutral progressive iCoT control loop.
func RunProgressiveICOT[S, D, A any](ctx context.Context, in io.Reader, out io.Writer, hooks ProgressiveLoopHooks[S, D, A]) (A, error) {
	var zero A
	prompts := NewPromptSession(in, out)
	prompts.SetDefaultMode(hooks.DefaultMode)
	extractor := hooks.Extractor
	if extractor == nil {
		extractor = NoopInteractiveExtractor[S, D]{}
	}
	_, noopExtractor := extractor.(interface{ noopInteractiveExtractor() })
	attempts := hooks.MaxAttempts
	if attempts <= 0 {
		attempts = 20
	}
	session := hooks.Session
	docs := append([]D(nil), hooks.Documents...)
	if hooks.Normalize != nil {
		hooks.Normalize(&session)
	}
	var events []PromptEvent
	record := func(kind string, data any) {
		events = append(events, PromptEvent{Kind: kind, Data: data})
	}
	opening := strings.TrimSpace(hooks.Opening)
	runDraft := func(session *S, docs []D, issues []ReadinessIssue, kind string) ([]ReadinessIssue, bool, error) {
		request := InteractiveDraftRequest[S, D]{
			Opening:           opening,
			Brief:             hooks.Brief,
			Session:           *session,
			Docs:              docs,
			TranscriptTurns:   prompts.Turns(),
			ReadinessFeedback: append([]ReadinessIssue(nil), issues...),
		}
		record(kind, map[string]any{
			"opening":          request.Opening,
			"turn_count":       len(request.TranscriptTurns),
			"readiness_issues": request.ReadinessFeedback,
		})
		draft, draftErr := extractor.Draft(ctx, request)
		if draftErr == nil && (hooks.LooksLikeSession == nil || hooks.LooksLikeSession(draft)) {
			if hooks.MergeDraft != nil {
				*session = hooks.MergeDraft(*session, draft, docs)
			} else {
				*session = draft
			}
			if hooks.Normalize != nil {
				hooks.Normalize(session)
			}
			if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(session, docs) && hooks.Normalize != nil {
				hooks.Normalize(session)
			}
			if hooks.CheckReadiness != nil {
				issues = hooks.CheckReadiness(*session, docs)
			}
			if hooks.DraftResultSummary != nil {
				record("model_draft_result", hooks.DraftResultSummary(*session))
			}
			if hooks.DraftEvents != nil {
				for _, event := range hooks.DraftEvents(*session) {
					if strings.TrimSpace(event.Kind) != "" {
						record(event.Kind, event.Data)
					}
				}
			}
			if hooks.Autosave != nil {
				if err := hooks.Autosave(*session); err != nil {
					return issues, true, err
				}
			}
			if hooks.AfterDraft != nil {
				if err := hooks.AfterDraft(*session); err != nil {
					return issues, true, err
				}
			}
			return issues, true, nil
		}
		if draftErr != nil {
			record("model_draft_error", draftErr.Error())
			if hooks.OnDraftError != nil {
				hooks.OnDraftError(draftErr)
			}
		}
		return issues, false, nil
	}

	if opening == "" {
		if hooks.OpeningPrompt != "" {
			fmt.Fprintln(out, hooks.OpeningPrompt)
		}
		answer, err := prompts.Ask("Workflow goal")
		if err != nil {
			return zero, err
		}
		opening = strings.TrimSpace(answer)
		if hooks.ApplyOpeningAnswer != nil {
			if err := hooks.ApplyOpeningAnswer(&session, opening, docs); err != nil {
				return zero, err
			}
		}
		if hooks.OpeningEvents != nil {
			for _, event := range hooks.OpeningEvents(session) {
				if strings.TrimSpace(event.Kind) != "" {
					record(event.Kind, event.Data)
				}
			}
		}
		if hooks.RefreshDocuments != nil {
			refreshed, err := hooks.RefreshDocuments(session, docs)
			if err != nil {
				return zero, err
			}
			docs = refreshed
		}
		if hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		if hooks.Autosave != nil {
			if err := hooks.Autosave(session); err != nil {
				return zero, err
			}
		}
		record("progressive_question", InteractiveQuestion{Prompt: "Workflow goal", Slots: []string{"workflow.goal"}})
		record("progressive_answer", PromptTurn{Label: "Workflow goal", Answer: answer})
	}
	if !hooks.NoLLM && len(docs) > 1 && opening != "" {
		ranked, err := extractor.Disambiguate(ctx, opening, docs)
		if err == nil && hooks.RankDocuments != nil {
			docs = hooks.RankDocuments(docs, ranked)
		} else if err != nil && hooks.OnDraftError != nil {
			hooks.OnDraftError(fmt.Errorf("OpenAPI ranking skipped: %w", err))
		}
	}

	var issues []ReadinessIssue
	for attempt := 0; attempt < attempts; attempt++ {
		if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(&session, docs) && hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		if hooks.CheckReadiness != nil {
			issues = hooks.CheckReadiness(session, docs)
		}
		shouldDraft := !noopExtractor
		if shouldDraft && hooks.ShouldDraft != nil {
			shouldDraft = hooks.ShouldDraft(session, docs, issues)
		}
		if shouldDraft {
			var err error
			issues, _, err = runDraft(&session, docs, issues, "model_draft_call")
			if err != nil {
				return zero, err
			}
		}
		record("readiness_decision", issues)
		if hooks.Ready != nil && hooks.Ready(session, issues) {
			record("next_question_decision", InteractiveQuestion{
				Prompt:          "Confirm first valid intent",
				SuggestedAnswer: "save",
				Slots:           []string{"confirmation"},
			})
			if hooks.FinalConfirm == nil {
				return zero, fmt.Errorf("final confirmation hook is required")
			}
			artifacts, err := hooks.FinalConfirm(prompts, &session, docs, &events)
			if err == nil {
				if hooks.FinalResultSummary != nil {
					record("final_generated_artifacts", hooks.FinalResultSummary(artifacts))
				}
				if hooks.SaveTranscript != nil {
					if saveErr := hooks.SaveTranscript(prompts.Turns(), events, artifacts); saveErr != nil {
						return artifacts, saveErr
					}
				}
			}
			return artifacts, err
		}
		if hooks.PlanQuestion == nil || hooks.ApplyAnswer == nil {
			return zero, fmt.Errorf("question planning and answer hooks are required")
		}
		question := hooks.PlanQuestion(session, docs, issues)
		if !noopExtractor && hooks.DraftQuestion != nil {
			drafted, err := hooks.DraftQuestion(ctx, &session, docs, issues, question)
			if err != nil {
				return zero, err
			}
			if drafted {
				if hooks.Normalize != nil {
					hooks.Normalize(&session)
				}
				if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(&session, docs) && hooks.Normalize != nil {
					hooks.Normalize(&session)
				}
				if hooks.CheckReadiness != nil {
					issues = hooks.CheckReadiness(session, docs)
				}
				record("question_draft_result", map[string]any{
					"question":         question,
					"readiness_issues": issues,
				})
				if hooks.DraftEvents != nil {
					for _, event := range hooks.DraftEvents(session) {
						if strings.TrimSpace(event.Kind) != "" {
							record(event.Kind, event.Data)
						}
					}
				}
				if hooks.Autosave != nil {
					if err := hooks.Autosave(session); err != nil {
						return zero, err
					}
				}
				record("readiness_decision", issues)
				if hooks.Ready != nil && hooks.Ready(session, issues) {
					record("next_question_decision", InteractiveQuestion{
						Prompt:          "Confirm first valid intent",
						SuggestedAnswer: "save",
						Slots:           []string{"confirmation"},
					})
					if hooks.FinalConfirm == nil {
						return zero, fmt.Errorf("final confirmation hook is required")
					}
					artifacts, err := hooks.FinalConfirm(prompts, &session, docs, &events)
					if err == nil {
						if hooks.FinalResultSummary != nil {
							record("final_generated_artifacts", hooks.FinalResultSummary(artifacts))
						}
						if hooks.SaveTranscript != nil {
							if saveErr := hooks.SaveTranscript(prompts.Turns(), events, artifacts); saveErr != nil {
								return artifacts, saveErr
							}
						}
					}
					return artifacts, err
				}
				question = hooks.PlanQuestion(session, docs, issues)
			}
		}
		if !noopExtractor && hooks.ShouldDraftQuestion != nil && hooks.ShouldDraftQuestion(session, docs, issues, question) {
			draftedIssues, drafted, err := runDraft(&session, docs, issues, "model_question_draft_call")
			if err != nil {
				return zero, err
			}
			if drafted {
				issues = draftedIssues
				record("readiness_decision", issues)
				if hooks.Ready != nil && hooks.Ready(session, issues) {
					record("next_question_decision", InteractiveQuestion{
						Prompt:          "Confirm first valid intent",
						SuggestedAnswer: "save",
						Slots:           []string{"confirmation"},
					})
					if hooks.FinalConfirm == nil {
						return zero, fmt.Errorf("final confirmation hook is required")
					}
					artifacts, err := hooks.FinalConfirm(prompts, &session, docs, &events)
					if err == nil {
						if hooks.FinalResultSummary != nil {
							record("final_generated_artifacts", hooks.FinalResultSummary(artifacts))
						}
						if hooks.SaveTranscript != nil {
							if saveErr := hooks.SaveTranscript(prompts.Turns(), events, artifacts); saveErr != nil {
								return artifacts, saveErr
							}
						}
					}
					return artifacts, err
				}
				question = hooks.PlanQuestion(session, docs, issues)
			}
		}
		record("next_question_decision", question)
		var answer string
		var err error
		if question.ForceAsk {
			answer, err = prompts.AskDefaultForced(question.Prompt, question.SuggestedAnswer)
		} else {
			answer, err = prompts.AskDefault(question.Prompt, question.SuggestedAnswer)
		}
		if err != nil {
			return zero, err
		}
		if strings.EqualFold(strings.TrimSpace(answer), "cancel") {
			return zero, ErrCanceled
		}
		if strings.TrimSpace(answer) == "" && strings.TrimSpace(question.SuggestedAnswer) == "" {
			record("progressive_question", question)
			record("progressive_answer", PromptTurn{Label: question.Prompt, Answer: answer})
			return zero, fmt.Errorf("progressive iCoT requires operator input for: %s", question.Prompt)
		}
		if err := hooks.ApplyAnswer(&session, question, answer, docs); err != nil {
			return zero, err
		}
		if hooks.RefreshDocuments != nil {
			refreshed, err := hooks.RefreshDocuments(session, docs)
			if err != nil {
				return zero, err
			}
			docs = refreshed
		}
		if hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		if hooks.DeterministicPrefill != nil && hooks.DeterministicPrefill(&session, docs) && hooks.Normalize != nil {
			hooks.Normalize(&session)
		}
		record("progressive_question", question)
		record("progressive_answer", PromptTurn{Label: question.Prompt, Answer: answer})
		if hooks.Autosave != nil {
			if err := hooks.Autosave(session); err != nil {
				return zero, err
			}
		}
	}
	return zero, fmt.Errorf("progressive iCoT could not reach a valid intent after %d draft attempts", attempts)
}

// ErrCanceled reports user cancellation from a generic interactive loop.
var ErrCanceled = fmt.Errorf("interactive authoring canceled")
