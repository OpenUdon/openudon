package eval

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func archiveWorkspace(src, archiveRoot, runID, name string) (string, error) {
	runName := safeArchiveName(strings.TrimSpace(runID))
	if runName == "" {
		runName = "run"
	}
	name = safeArchiveName(name)
	if name == "" {
		name = safeArchiveName(filepath.Base(filepath.Clean(src)))
	}
	if name == "" {
		name = "eval"
	}
	relative := filepath.Join(runName, name)
	cleanRoot, err := filepath.Abs(archiveRoot)
	if err != nil {
		return "", fmt.Errorf("resolve eval archive root: %w", err)
	}
	target := filepath.Join(cleanRoot, relative)
	contained, err := filepath.Rel(cleanRoot, target)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("eval archive target escapes archive root")
	}
	if err := os.MkdirAll(cleanRoot, 0o755); err != nil {
		return "", err
	}
	runRoot := filepath.Join(cleanRoot, runName)
	if err := ensureArchiveDirectory(runRoot); err != nil {
		return "", err
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("eval archive target already exists: %s", target)
		}
		return "", err
	}
	if err := copyArchiveTree(src, target); err != nil {
		if cleanupErr := os.RemoveAll(target); cleanupErr != nil {
			return "", fmt.Errorf("copy eval archive: %w; clean partial archive: %v", err, cleanupErr)
		}
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

func ensureArchiveDirectory(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(path, 0o755); err != nil && !os.IsExist(err) {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("eval archive run directory must be a real directory: %s", path)
	}
	return nil
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
		if rel == "." {
			return nil
		}
		if entry.IsDir() {
			return os.Mkdir(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("eval workspace contains non-regular file: %s", path)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		opened, err := in.Stat()
		if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			_ = in.Close()
			return fmt.Errorf("eval workspace file changed before archival: %s", path)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
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
		if err := out.Sync(); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}

func safeArchiveName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
			return r
		}
		return '-'
	}, value)
	value = strings.Trim(value, "-")
	if value == "." || value == ".." {
		return ""
	}
	if len(value) > 128 {
		value = value[:128]
	}
	return value
}
