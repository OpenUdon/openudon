package eval

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ArchiveGeneratedDirs(results []EvalResult, archiveRoot string, runID string) ([]EvalResult, error) {
	archiveRoot = strings.TrimSpace(archiveRoot)
	if archiveRoot == "" {
		return results, nil
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = "run"
	}
	archived := append([]EvalResult(nil), results...)
	for i, result := range archived {
		src := strings.TrimSpace(result.GeneratedDir)
		if src == "" {
			continue
		}
		name := safeArchiveName(result.Name)
		if name == "" {
			name = safeArchiveName(filepath.Base(filepath.Clean(src)))
		}
		target := filepath.Join(archiveRoot, safeArchiveName(runID), name)
		if err := os.RemoveAll(target); err != nil {
			return nil, err
		}
		if err := copyArchiveTree(src, target); err != nil {
			return nil, err
		}
		archived[i].GeneratedDir = target
	}
	return archived, nil
}

func copyArchiveTree(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeType != 0 {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			_ = in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			return err
		}
		if err := in.Close(); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}

func safeArchiveName(value string) string {
	value = strings.TrimSpace(value)
	return strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':':
			return '-'
		default:
			return r
		}
	}, value)
}
