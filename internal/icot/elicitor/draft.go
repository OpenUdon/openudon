package elicitor

import (
	"github.com/OpenUdon/apitools/icot"
)

// DraftPath returns the canonical `.icot/session.yaml` path under exampleDir.
func DraftPath(exampleDir string) string {
	return icot.DraftPath(exampleDir)
}

// LoadDraft reads a previously-saved Session from path. The boolean is
// true when a session was found AND it looks like a real session (per
// LooksLikeSession).
func LoadDraft(path string) (Session, bool, error) {
	session, ok, err := icot.LoadDraft[Session](path)
	if err != nil || !ok {
		return Session{}, false, err
	}
	session.Normalize()
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
	return icot.SaveDraft(path, session)
}

// DeleteDraft removes the on-disk draft and prunes the enclosing `.icot/`
// directory.
func DeleteDraft(path string) error {
	return icot.DeleteDraft(path)
}
