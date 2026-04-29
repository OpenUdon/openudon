package elicitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Transcript struct {
	Version string       `json:"version"`
	TimeUTC string       `json:"time_utc"`
	Turns   []ReplayTurn `json:"turns"`
	Session Session      `json:"session,omitempty"`
}

func SaveTranscript(path string, turns []ReplayTurn, session Session) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	transcript := Transcript{
		Version: "ramen.icot-transcript.v1",
		TimeUTC: time.Now().UTC().Format(time.RFC3339),
		Turns:   append([]ReplayTurn(nil), turns...),
		Session: session,
	}
	data, err := json.MarshalIndent(transcript, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o600)
}
