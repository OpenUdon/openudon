package elicitor

import "github.com/genelet/openapisearch"

type Transcript struct {
	Version string            `json:"version"`
	TimeUTC string            `json:"time_utc"`
	Turns   []ReplayTurn      `json:"turns"`
	Events  []TranscriptEvent `json:"events,omitempty"`
	Session Session           `json:"session,omitempty"`
}

type TranscriptEvent = openapisearch.PromptEvent

func SaveTranscript(path string, turns []ReplayTurn, session Session) error {
	return SaveTranscriptWithEvents(path, turns, nil, session)
}

func SaveTranscriptWithEvents(path string, turns []ReplayTurn, events []TranscriptEvent, session Session) error {
	return openapisearch.SavePromptTranscript(path, "ramen.icot-transcript.v1", turns, events, session)
}
