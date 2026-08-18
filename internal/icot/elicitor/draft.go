package elicitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenUdon/openudon/internal/authoring"
)

// DraftPath returns the canonical `.icot/session.yaml` path under exampleDir.
func DraftPath(exampleDir string) string {
	return authoring.DraftPath(exampleDir)
}

// LoadDraft reads a previously-saved Session from path. The boolean is
// true when a session was found AND it looks like a real session (per
// LooksLikeSession).
func LoadDraft(path string) (Session, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Session{}, false, nil
		}
		return Session{}, false, err
	}
	session, err := DecodeSession(data, filepath.Ext(path))
	if err != nil {
		return Session{}, false, err
	}
	return session, LooksLikeSession(session), nil
}

// SaveDraft writes session to path atomically. A no-op when path is empty
// or the session does not look real yet (avoids persisting empty drafts on
// the first prompt).
func SaveDraft(path string, session Session) error {
	if path == "" || !LooksLikeSession(session) {
		return nil
	}
	session.Normalize()
	_, _, err := DraftBytes(session)
	if err != nil {
		return err
	}
	return authoring.SaveDraft(path, session)
}

// DraftBytes returns the exact bytes SaveDraft will persist. The boolean is
// false when SaveDraft would intentionally be a no-op for an empty session.
func DraftBytes(session Session) ([]byte, bool, error) {
	if !LooksLikeSession(session) {
		return nil, false, nil
	}
	for index, event := range session.DraftEvents {
		if _, err := json.Marshal(event); err != nil {
			return nil, false, fmt.Errorf("draft event %d is not JSON-marshalable: %w", index, err)
		}
	}
	session.Normalize()
	data, err := authoring.MarshalDraft(session)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// DeleteDraft removes the on-disk draft and prunes the enclosing `.icot/`
// directory.
func DeleteDraft(path string) error {
	return authoring.DeleteDraft(path)
}
