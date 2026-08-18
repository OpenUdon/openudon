// Copyright (c) Greetingland LLC

// Local helpers for on-disk interactive authoring sessions.
package authoring

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"gopkg.in/yaml.v3"
)

// DraftPath returns the canonical `.icot/session.yaml` path under exampleDir.
// Engines that want a different on-disk layout can ignore this helper and
// pass their own path to LoadDraft / SaveDraft.
func DraftPath(exampleDir string) string {
	return filepath.Join(exampleDir, ".icot", "session.yaml")
}

// LoadDraft reads a YAML-encoded session of type T from path. It returns
// (zero, false, nil) when the file does not exist, (decoded, true, nil) on
// success, and propagates other errors.
func LoadDraft[T any](path string) (T, bool, error) {
	var zero T
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var session T
	if err := yaml.Unmarshal(data, &session); err != nil {
		return zero, false, err
	}
	return session, true, nil
}

// SaveDraft writes session as YAML to path atomically. The parent directory
// is created with 0o755 if it does not exist; the file is written 0o600.
// A no-op when path is empty.
func SaveDraft[T any](path string, session T) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := MarshalDraft(session)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 0o600)
}

// MarshalDraft returns the exact YAML bytes SaveDraft persists. Callers that
// install an in-memory state only after persistence can use these bytes to
// construct the post-commit state without a fallible read after the rename.
func MarshalDraft[T any](session T) ([]byte, error) {
	// Marshal through JSON first so nested shared Authoring contracts that
	// intentionally publish JSON tags (for example interview.State) retain
	// their exact snake_case wire names inside the YAML session file.
	jsonData, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}
	var wire any
	if err := json.Unmarshal(jsonData, &wire); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(wire)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DeleteDraft removes the draft file and prunes the enclosing `.icot/`
// directory if it becomes empty. Missing files are not an error.
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
	return nil
}
