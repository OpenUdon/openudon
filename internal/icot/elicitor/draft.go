package elicitor

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func DraftPath(exampleDir string) string {
	return filepath.Join(exampleDir, ".icot", "session.yaml")
}

func LoadDraft(path string) (Session, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	session, err := DecodeSession(data, filepath.Ext(path))
	if err != nil {
		return Session{}, false, err
	}
	session.Normalize()
	return session, LooksLikeSession(session), nil
}

func SaveDraft(path string, session Session) error {
	if path == "" || !LooksLikeSession(session) {
		return nil
	}
	session.Normalize()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(session)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o600)
}

func DeleteDraft(path string) error {
	if path == "" {
		return nil
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return err
	}
	draftDir := filepath.Dir(path)
	_ = os.Remove(draftDir)
	if filepath.Base(draftDir) == ".icot" {
		_ = os.Remove(filepath.Dir(draftDir))
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp.")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
