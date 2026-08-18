package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
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

func (e *Engine) installWorkspaceBaseline(snapshot Snapshot, baseline workspaceFingerprint) {
	e.watchedPaths = watchedPaths(e.workspaceRoot, snapshot)
	e.workspaceBaseline = baseline
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
		entry, err := fingerprintPath(root, path)
		if err != nil {
			return workspaceFingerprint{}, err
		}
		entries[path] = entry
	}
	return workspaceFingerprint{entries: entries, digest: fingerprintDigest(entries)}, nil
}

// observeWorkspaceTree records regular files before a mutation starts so a
// path that becomes engine-owned during refresh is still bound to its
// pre-refresh state. Unreadable or unsafe entries outside the eventual watched
// set are conservatively omitted; if one later becomes watched, comparison
// either reports drift from missing or fails closed while inspecting it.
func observeWorkspaceTree(ctx context.Context, root string) (workspaceObservation, error) {
	observation := workspaceObservation{entries: map[string]pathFingerprint{}}
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return observation, nil
	}
	if err != nil {
		return workspaceObservation{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return workspaceObservation{}, fmt.Errorf("workspace root %s is not a safe directory", root)
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		entryInfo, err := entry.Info()
		if err != nil || !entryInfo.Mode().IsRegular() {
			return nil
		}
		fingerprint, err := fingerprintPath(root, path)
		if err == nil {
			observation.entries[filepath.Clean(path)] = fingerprint
		}
		return nil
	})
	if err != nil {
		return workspaceObservation{}, err
	}
	return observation, nil
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

func fingerprintPath(root, path string) (pathFingerprint, error) {
	if _, err := artifactwriter.SafeExampleTarget(root, relativePath(root, path)); err != nil {
		return pathFingerprint{}, fmt.Errorf("inspect watched path %s: %w", path, err)
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return pathFingerprint{Path: path, Type: "missing"}, nil
	}
	if err != nil {
		return pathFingerprint{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return pathFingerprint{}, fmt.Errorf("watched path %s is not a safe regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return pathFingerprint{}, err
	}
	after, err := os.Lstat(path)
	if err != nil || after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() {
		return pathFingerprint{}, fmt.Errorf("watched path %s changed while it was inspected", path)
	}
	digest := sha256.Sum256(data)
	return pathFingerprint{Path: path, Type: "regular", SHA256: hex.EncodeToString(digest[:])}, nil
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

func workspacePathInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
