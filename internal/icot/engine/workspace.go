package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
)

// WorkspaceStatus is the optimistic ownership state for the one example.
type WorkspaceStatus struct {
	ExternallyModified bool `json:"externally_modified"`
}

type pathFingerprint struct {
	Path   string
	Type   string
	SHA256 string
}

type workspaceFingerprint struct {
	entries map[string]pathFingerprint
	digest  string
}

type workspaceObservation struct {
	entries map[string]pathFingerprint
}

func (e *Engine) WorkspaceStatus(ctx context.Context) (WorkspaceStatus, error) {
	if e == nil {
		return WorkspaceStatus{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.workspaceStatusLocked(ctx)
}

func (e *Engine) workspaceStatusLocked(ctx context.Context) (WorkspaceStatus, error) {
	if err := ctx.Err(); err != nil {
		return WorkspaceStatus{}, operational(err)
	}
	if e.externallyModified {
		return WorkspaceStatus{ExternallyModified: true}, nil
	}
	current, err := captureWorkspace(ctx, e.workspaceRoot, e.watchedPaths)
	if err != nil {
		return WorkspaceStatus{}, operational(err)
	}
	if current.digest != e.workspaceBaseline.digest {
		e.externallyModified = true
		return WorkspaceStatus{ExternallyModified: true}, nil
	}
	return WorkspaceStatus{}, nil
}

func (e *Engine) requireMutableWorkspaceLocked(ctx context.Context) error {
	status, err := e.workspaceStatusLocked(ctx)
	if err != nil {
		return err
	}
	if status.ExternallyModified {
		return conflict("workspace_changed", errors.New("the authoring workspace changed outside this process; restart is required"))
	}
	return nil
}

func watchedPaths(root string, snapshot Snapshot) []string {
	paths := []string{
		filepath.Join(root, "project.md"),
		filepath.Join(root, "workflows", "intent.hcl"),
		filepath.Join(root, "workflows", "intent.draft.hcl"),
		filepath.Join(root, ".icot", "session.yaml"),
		filepath.Join(root, ".icot", "readiness.json"),
		filepath.Join(root, ".icot", "browser-sources.json"),
		filepath.Join(root, ".icot", "browser-authentication.json"),
		filepath.Join(root, ".icot", "authenticated-browser-authoring.json"),
		filepath.Join(root, ".icot", "ui-sources.json"),
	}
	for _, action := range snapshot.ProposedActions {
		paths = append(paths, action.Path)
	}
	for _, source := range snapshot.SelectedSources {
		paths = append(paths, filepath.Join(root, filepath.FromSlash(source.TargetPath)))
	}
	return uniqueSortedPaths(paths)
}

func uniqueSortedPaths(paths []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if !seen[abs] {
			seen[abs] = true
			result = append(result, abs)
		}
	}
	sort.Strings(result)
	return result
}

func captureWorkspace(ctx context.Context, root string, paths []string) (workspaceFingerprint, error) {
	entries := make(map[string]pathFingerprint, len(paths))
	for _, path := range uniqueSortedPaths(paths) {
		if err := ctx.Err(); err != nil {
			return workspaceFingerprint{}, err
		}
		entry, err := fingerprintPath(ctx, root, path)
		if err != nil {
			return workspaceFingerprint{}, err
		}
		entries[path] = entry
	}
	return workspaceFingerprint{entries: entries, digest: fingerprintDigest(entries)}, nil
}

// observeMutationWorkspaceLocked bounds pre-refresh observation to paths the
// engine already watches plus targets advertised by the current local and
// registry discovery plans. A target produced only by the refresh is absent
// from this observation and acceptedFingerprint therefore treats it as
// missing, so a concurrently-created existing path fails closed as drift.
func (e *Engine) observeMutationWorkspaceLocked(ctx context.Context) (workspaceObservation, error) {
	paths, err := e.mutationObservationPathsLocked()
	if err != nil {
		return workspaceObservation{}, err
	}
	fingerprint, err := captureWorkspace(ctx, e.workspaceRoot, paths)
	if err != nil {
		return workspaceObservation{}, err
	}
	return workspaceObservation{entries: fingerprint.entries}, nil
}

func (e *Engine) mutationObservationPathsLocked() ([]string, error) {
	paths := append([]string(nil), e.watchedPaths...)
	plans := make([]elicitor.SourceMaterialization, 0, len(e.discovery.Plans)+len(e.registry.Plans))
	plans = append(plans, e.discovery.Plans...)
	plans = append(plans, e.registry.Plans...)
	for _, plan := range plans {
		target, err := artifactwriter.SafeExampleTarget(e.workspaceRoot, plan.TargetPath)
		if err != nil {
			return nil, fmt.Errorf("inspect candidate materialization target %q: %w", plan.TargetPath, err)
		}
		paths = append(paths, target)
	}
	return uniqueSortedPaths(paths), nil
}

func acceptedFingerprint(paths []string, prior workspaceFingerprint, observed workspaceObservation) workspaceFingerprint {
	entries := make(map[string]pathFingerprint, len(paths))
	for _, path := range uniqueSortedPaths(paths) {
		if entry, ok := prior.entries[path]; ok {
			entries[path] = entry
			continue
		}
		if entry, ok := observed.entries[path]; ok {
			entries[path] = entry
			continue
		}
		entries[path] = pathFingerprint{Path: path, Type: "missing"}
	}
	return workspaceFingerprint{entries: entries, digest: fingerprintDigest(entries)}
}

func fingerprintPath(ctx context.Context, root, path string) (pathFingerprint, error) {
	if _, err := artifactwriter.SafeExampleTarget(root, relativePath(root, path)); err != nil {
		return pathFingerprint{}, fmt.Errorf("inspect watched path %s: %w", path, err)
	}
	before, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return pathFingerprint{Path: path, Type: "missing"}, nil
	}
	if err != nil {
		return pathFingerprint{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return pathFingerprint{}, fmt.Errorf("watched path %s is not a safe regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return pathFingerprint{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !sameFingerprintFileState(before, opened) {
		return pathFingerprint{}, fmt.Errorf("watched path %s changed while it was inspected", path)
	}
	digest, count, err := streamSHA256(ctx, file)
	if err != nil {
		return pathFingerprint{}, err
	}
	afterOpen, openErr := file.Stat()
	afterPath, pathErr := os.Lstat(path)
	if openErr != nil || pathErr != nil || count != before.Size() ||
		!sameFingerprintFileState(before, afterOpen) || !sameFingerprintFileState(before, afterPath) {
		return pathFingerprint{}, fmt.Errorf("watched path %s changed while it was inspected", path)
	}
	return pathFingerprint{Path: path, Type: "regular", SHA256: digest}, nil
}

func streamSHA256(ctx context.Context, reader io.Reader) (string, int64, error) {
	hash := sha256.New()
	buffer := make([]byte, 32*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return "", total, err
		}
		read, err := reader.Read(buffer)
		if read > 0 {
			written, writeErr := hash.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return "", total, writeErr
			}
			if written != read {
				return "", total, io.ErrShortWrite
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if contextErr := ctx.Err(); contextErr != nil {
					return "", total, contextErr
				}
				return hex.EncodeToString(hash.Sum(nil)), total, nil
			}
			return "", total, err
		}
		if read == 0 {
			return "", total, io.ErrNoProgress
		}
	}
}

func sameFingerprintFileState(left, right os.FileInfo) bool {
	return left != nil && right != nil &&
		left.Mode().IsRegular() && right.Mode().IsRegular() &&
		left.Mode() == right.Mode() && left.Size() == right.Size() &&
		left.ModTime().Equal(right.ModTime()) && os.SameFile(left, right)
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "../invalid"
	}
	return filepath.ToSlash(rel)
}

func fingerprintDigest(entries map[string]pathFingerprint) string {
	paths := make([]string, 0, len(entries))
	for path := range entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		entry := entries[path]
		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry.Type))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry.SHA256))
		_, _ = hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func fingerprintWithDraft(baseline workspaceFingerprint, path string, data []byte, exists bool) workspaceFingerprint {
	return fingerprintWithFiles(baseline, []artifactwriter.GeneratedFile{{Path: path, Content: string(data), Remove: !exists}})
}

func fingerprintWithFiles(baseline workspaceFingerprint, files []artifactwriter.GeneratedFile) workspaceFingerprint {
	entries := make(map[string]pathFingerprint, len(baseline.entries))
	for path, entry := range baseline.entries {
		entries[path] = entry
	}
	for _, file := range files {
		path, err := filepath.Abs(file.Path)
		if err != nil {
			continue
		}
		path = filepath.Clean(path)
		if file.Remove {
			entries[path] = pathFingerprint{Path: path, Type: "missing"}
			continue
		}
		digest := sha256.Sum256([]byte(file.Content))
		entries[path] = pathFingerprint{Path: path, Type: "regular", SHA256: hex.EncodeToString(digest[:])}
	}
	return workspaceFingerprint{entries: entries, digest: fingerprintDigest(entries)}
}

func compareWorkspace(root string, paths []string, accepted workspaceFingerprint) error {
	current, err := captureWorkspace(context.Background(), root, paths)
	if err != nil {
		return operational(err)
	}
	if current.digest != accepted.digest {
		return conflict("workspace_changed", errors.New("the authoring workspace changed before artifact replacement; restart is required"))
	}
	return nil
}

func (e *Engine) compareAndLatchWorkspaceLocked(paths []string, accepted workspaceFingerprint) error {
	err := compareWorkspace(e.workspaceRoot, paths, accepted)
	if err == nil {
		return nil
	}
	class, _ := FailureDetails(err)
	if class == FailureConflict {
		e.externallyModified = true
	}
	return err
}

func canonicalWorkspaceRoot(exampleDir string) (string, error) {
	projectPath, err := artifactwriter.SafeExampleTarget(exampleDir, "project.md")
	if err != nil {
		return "", err
	}
	return filepath.Dir(projectPath), nil
}
