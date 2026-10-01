package elicitor

import "github.com/OpenUdon/openudon/internal/authoring"

type Transcript struct {
	Version string            `json:"version"`
	TimeUTC string            `json:"time_utc"`
	Turns   []ReplayTurn      `json:"turns"`
	Events  []TranscriptEvent `json:"events,omitempty"`
	Session Session           `json:"session,omitempty"`
}

type TranscriptEvent = authoring.PromptEvent
