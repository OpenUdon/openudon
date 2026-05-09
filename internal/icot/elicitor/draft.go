package elicitor

import "github.com/OpenUdon/openudon/internal/authoring"

// DraftPath returns the canonical `.icot/session.yaml` path under exampleDir.
func DraftPath(exampleDir string) string {
	return authoring.DraftPath(exampleDir)
}

// LoadDraft reads a previously-saved Session from path. The boolean is
// true when a session was found AND it looks like a real session (per
// LooksLikeSession).
func LoadDraft(path string) (Session, bool, error) {
	session, ok, err := authoring.LoadDraft[Session](path)
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
	return authoring.SaveDraft(path, session)
}

// DeleteDraft removes the on-disk draft and prunes the enclosing `.icot/`
// directory.
func DeleteDraft(path string) error {
	return authoring.DeleteDraft(path)
}
