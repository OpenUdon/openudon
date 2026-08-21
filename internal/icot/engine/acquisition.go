package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/credentialpolicy"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
)

const (
	MaxUploadBytes       = int64(20 << 20)
	maxUploadedSources   = 128
	uploadRegistrySchema = "openudon.icot-upload-inbox.v1"
	stagedRegistrySchema = "openudon.icot-staged-sources.v1"
)

var uploadIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// UploadedSource is a validated private-inbox candidate. It deliberately does
// not expose its private filesystem path to HTTP clients.
type UploadedSource struct {
	ID              string `json:"id"`
	OriginalName    string `json:"original_name"`
	Kind            string `json:"kind"`
	SourceFamily    string `json:"source_family"`
	Title           string `json:"title,omitempty"`
	OperationCount  int    `json:"operation_count"`
	SHA256          string `json:"sha256"`
	Bytes           int64  `json:"bytes"`
	CanonicalTarget string `json:"canonical_target"`
	InboxFile       string `json:"-"`
}

// StagedSource records UI authority over one workspace source. Removal is
// allowed only while the exact path and digest still match this registry.
type StagedSource struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	TargetPath string `json:"target_path"`
	SHA256     string `json:"sha256"`
	Title      string `json:"title,omitempty"`
}

type uploadRecord struct {
	Version string         `json:"version"`
	Source  UploadedSource `json:"source"`
	File    string         `json:"file"`
}

type stagedRegistry struct {
	Version string         `json:"version"`
	Sources []StagedSource `json:"sources,omitempty"`
}

// SelectJourney persists the selected starter and goal through the same
// optimistic workspace transaction as interview rounds.
func (e *Engine) SelectJourney(ctx context.Context, starter, goal string) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	workspaceAtStart, err := e.observeMutationWorkspaceLocked(ctx)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	next, err := cloneSession(e.session)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	if err := elicitor.SelectJourney(&next, starter, goal); err != nil {
		return Snapshot{}, rejected(err)
	}
	return e.persistSessionMutationLocked(ctx, next, workspaceAtStart, nil)
}

// ResumeAuthoring revalidates the current engine-owned state after a package
// quality failure. The transport owns the lifecycle transition; subsequent
// authoring mutations and final approval still use the ordinary engine paths.
func (e *Engine) ResumeAuthoring(ctx context.Context) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	if err := e.refreshLocked(ctx); err != nil {
		return Snapshot{}, classifyRefreshFailure(err)
	}
	return e.refreshCachedSnapshotLocked(ctx)
}

// UploadSource validates one bounded API-family document into the private
// inbox. It does not write any workspace file.
func (e *Engine) UploadSource(ctx context.Context, name string, input io.Reader) (UploadedSource, Snapshot, error) {
	if e == nil {
		return UploadedSource{}, Snapshot{}, operational(errors.New("engine is nil"))
	}
	if input == nil {
		return UploadedSource{}, Snapshot{}, rejected(errors.New("uploaded source body is required"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return UploadedSource{}, Snapshot{}, err
	}
	if e.config.PrivateRoot == "" {
		return UploadedSource{}, Snapshot{}, rejected(errors.New("API upload requires icot ui --private-root"))
	}
	if len(e.uploadedSources) >= maxUploadedSources {
		return UploadedSource{}, Snapshot{}, rejected(fmt.Errorf("private inbox is limited to %d sources", maxUploadedSources))
	}
	data, err := io.ReadAll(io.LimitReader(input, MaxUploadBytes+1))
	if err != nil {
		return UploadedSource{}, Snapshot{}, operational(err)
	}
	if int64(len(data)) > MaxUploadBytes {
		return UploadedSource{}, Snapshot{}, rejected(fmt.Errorf("uploaded source exceeds the %d-byte limit", MaxUploadBytes))
	}
	if len(data) == 0 || !utf8.Valid(data) {
		return UploadedSource{}, Snapshot{}, rejected(errors.New("uploaded source must be non-empty UTF-8 text"))
	}
	if credentialpolicy.ContainsLikelyValue(data) {
		return UploadedSource{}, Snapshot{}, rejected(errors.New("uploaded source contains secret-like literal content"))
	}
	digest := sha256.Sum256(data)
	id := hex.EncodeToString(digest[:])
	if existing, ok := e.uploadedSources[id]; ok {
		snapshot, snapshotErr := e.snapshotResultLocked()
		return existing, snapshot, snapshotErr
	}
	inboxDir, metadataDir, err := e.ensureInboxDirectories()
	if err != nil {
		return UploadedSource{}, Snapshot{}, operational(err)
	}
	extension := safeUploadExtension(name)
	fileName := id + extension
	inboxPath := filepath.Join(inboxDir, fileName)
	if err := writeNewPrivateFile(inboxPath, data); err != nil {
		return UploadedSource{}, Snapshot{}, operational(err)
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.Remove(inboxPath)
		}
	}()
	report, err := apitools.DiscoverLocalSources(ctx, apitools.LocalSourceDiscoveryOptions{
		Roots: []string{inboxPath}, MaxVisitedEntries: 1, MaxCandidates: 2, MaxBytes: MaxUploadBytes,
	})
	if err != nil {
		return UploadedSource{}, Snapshot{}, rejected(fmt.Errorf("validate uploaded API source: %w", err))
	}
	if report.Truncated || len(report.Ambiguous) != 0 || len(report.Rejected) != 0 || len(report.Candidates) != 1 {
		return UploadedSource{}, Snapshot{}, rejected(errors.New("uploaded document is unsupported, ambiguous, or invalid"))
	}
	candidate := report.Candidates[0]
	candidate.ID = sourceIDFromName(name, id)
	plan, err := elicitor.SourceMaterializationForCandidate(e.workspaceRoot, candidate)
	if err != nil {
		return UploadedSource{}, Snapshot{}, rejected(err)
	}
	source := UploadedSource{
		ID: id, OriginalName: safeOriginalName(name), Kind: candidate.Kind, SourceFamily: candidate.SourceFamily,
		Title: candidate.Title, OperationCount: candidate.OperationCount, SHA256: candidate.SHA256,
		Bytes: candidate.Bytes, CanonicalTarget: plan.TargetPath, InboxFile: fileName,
	}
	metadata, err := json.MarshalIndent(uploadRecord{Version: uploadRegistrySchema, Source: source, File: fileName}, "", "  ")
	if err != nil {
		return UploadedSource{}, Snapshot{}, operational(err)
	}
	metadata = append(metadata, '\n')
	if err := writeNewPrivateFile(filepath.Join(metadataDir, id+".json"), metadata); err != nil {
		return UploadedSource{}, Snapshot{}, operational(err)
	}
	keep = true
	e.uploadedSources[id] = source
	snapshot, err := e.refreshCachedSnapshotLocked(ctx)
	if err != nil {
		delete(e.uploadedSources, id)
		_ = os.Remove(filepath.Join(metadataDir, id+".json"))
		_ = os.Remove(inboxPath)
		return UploadedSource{}, Snapshot{}, err
	}
	return source, snapshot, nil
}

// StageUploadedSource copies one reviewed inbox candidate to its canonical
// workspace target and records removal authority in the same atomic commit.
func (e *Engine) StageUploadedSource(ctx context.Context, id string) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	id = strings.ToLower(strings.TrimSpace(id))
	source, ok := e.uploadedSources[id]
	if !ok || !uploadIDPattern.MatchString(id) {
		return Snapshot{}, rejected(errors.New("uploaded source is not available for staging"))
	}
	if _, collision := e.stagedSources[id]; collision {
		return Snapshot{}, rejected(errors.New("uploaded source is already staged"))
	}
	content, err := e.readUploadedSource(source)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	target, err := artifactwriter.SafeExampleTarget(e.workspaceRoot, source.CanonicalTarget)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	if _, err := os.Lstat(target); err == nil {
		return Snapshot{}, rejected(fmt.Errorf("canonical source target %s already exists", source.CanonicalTarget))
	} else if !errors.Is(err, os.ErrNotExist) {
		return Snapshot{}, operational(err)
	}
	workspaceAtStart, err := e.observeMutationWorkspaceLocked(ctx)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	staged := StagedSource{ID: id, Kind: source.Kind, TargetPath: source.CanonicalTarget, SHA256: source.SHA256, Title: source.Title}
	nextRegistry := cloneStagedSources(e.stagedSources)
	nextRegistry[id] = staged
	registryBytes, err := marshalStagedRegistry(nextRegistry)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	registryPath := filepath.Join(e.workspaceRoot, ".icot", "ui-sources.json")
	paths := uniqueSortedPaths(append(append([]string(nil), e.watchedPaths...), target, registryPath))
	accepted := acceptedFingerprint(paths, e.workspaceBaseline, workspaceAtStart)
	prepared := artifactwriter.Prepared{ExampleRoot: e.workspaceRoot, Files: []artifactwriter.GeneratedFile{
		{Path: target, Content: string(content), Action: "stage_uploaded_source", Reason: "explicitly reviewed API source upload"},
		{Path: registryPath, Content: string(registryBytes), AllowOverwrite: true, Action: "update_source_ownership", Reason: "record UI-owned source removal authority"},
	}}
	if _, err := commitPrepared(prepared, false, func() error { return e.compareAndLatchWorkspaceLocked(paths, accepted) }); err != nil {
		return Snapshot{}, classifyCommit(err)
	}
	e.stagedSources = nextRegistry
	delete(e.uploadedSources, id)
	e.removeUploadedSourceFiles(source)
	return e.refreshAfterAcquisitionCommitLocked(ctx)
}

// RemoveStagedSource removes only an unchanged source previously staged by
// this UI and updates the ownership registry atomically.
func (e *Engine) RemoveStagedSource(ctx context.Context, id string) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, operational(errors.New("engine is nil"))
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.requireMutableWorkspaceLocked(ctx); err != nil {
		return Snapshot{}, err
	}
	workspaceAtStart, err := e.observeMutationWorkspaceLocked(ctx)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	id = strings.ToLower(strings.TrimSpace(id))
	staged, ok := e.stagedSources[id]
	if !ok {
		return Snapshot{}, rejected(errors.New("source was not staged by this UI"))
	}
	target, err := verifyStagedSourceTarget(e.workspaceRoot, staged)
	if err != nil {
		return Snapshot{}, err
	}
	nextRegistry := cloneStagedSources(e.stagedSources)
	delete(nextRegistry, id)
	registryBytes, err := marshalStagedRegistry(nextRegistry)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	registryPath := filepath.Join(e.workspaceRoot, ".icot", "ui-sources.json")
	paths := uniqueSortedPaths(append(append([]string(nil), e.watchedPaths...), target, registryPath))
	accepted := acceptedFingerprint(paths, e.workspaceBaseline, workspaceAtStart)
	files := []artifactwriter.GeneratedFile{
		{Path: target, Remove: true, AllowOverwrite: true, ExpectedCurrentSHA256: "sha256:" + staged.SHA256, Action: "remove_staged_source", Reason: "explicitly removed UI-owned source"},
		{Path: registryPath, Content: string(registryBytes), AllowOverwrite: true, Action: "update_source_ownership", Reason: "remove UI source authority"},
	}
	nextSession, err := cloneSession(e.session)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	filtered := nextSession.SourcePlan[:0]
	for _, plan := range nextSession.SourcePlan {
		if filepath.ToSlash(plan.TargetPath) != filepath.ToSlash(staged.TargetPath) {
			filtered = append(filtered, plan)
		}
	}
	nextSession.SourcePlan = filtered
	if draft, persists, draftErr := elicitor.DraftBytes(nextSession); draftErr != nil {
		return Snapshot{}, rejected(draftErr)
	} else if persists {
		files = append(files, artifactwriter.GeneratedFile{Path: elicitor.DraftPath(e.workspaceRoot), Content: string(draft), AllowOverwrite: true, Action: "update_authoring_session"})
	}
	prepared := artifactwriter.Prepared{ExampleRoot: e.workspaceRoot, Files: files}
	if _, err := commitPrepared(prepared, true, func() error {
		if err := e.compareAndLatchWorkspaceLocked(paths, accepted); err != nil {
			return err
		}
		rechecked, err := verifyStagedSourceTarget(e.workspaceRoot, staged)
		if err != nil {
			return err
		}
		if rechecked != target {
			return conflict("workspace_changed", errors.New("staged source target changed before replacement"))
		}
		return nil
	}); err != nil {
		return Snapshot{}, classifyCommit(err)
	}
	e.session = nextSession
	e.stagedSources = nextRegistry
	return e.refreshAfterAcquisitionCommitLocked(ctx)
}

func verifyStagedSourceTarget(root string, staged StagedSource) (string, error) {
	target, err := artifactwriter.SafeExampleTarget(root, staged.TargetPath)
	if err != nil {
		return "", rejected(err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return "", conflict("workspace_changed", errors.New("staged source is missing or unreadable"))
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != staged.SHA256 {
		return "", conflict("workspace_changed", errors.New("staged source changed after UI staging"))
	}
	return target, nil
}

func validateAcquisitionPrivateRoot(raw, workspace string) (string, error) {
	if !filepath.IsAbs(strings.TrimSpace(raw)) {
		return "", errors.New("private root must be an absolute path")
	}
	path := filepath.Clean(raw)
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("private root must be an existing non-symlink directory")
	}
	if info.Mode().Perm() != 0o700 {
		return "", errors.New("private root permissions must be exactly 0700")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || filepath.Clean(resolved) != path {
		return "", errors.New("private root must not traverse symlinks")
	}
	if acquisitionPathWithin(workspace, path) || acquisitionPathWithin(path, workspace) {
		return "", errors.New("private root must be outside and disjoint from the example")
	}
	return path, nil
}

func acquisitionPathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func (e *Engine) ensureInboxDirectories() (string, string, error) {
	inbox := filepath.Join(e.config.PrivateRoot, "openudon-upload-inbox")
	metadata := filepath.Join(e.config.PrivateRoot, "openudon-upload-metadata")
	for _, path := range []string{inbox, metadata} {
		if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return "", "", err
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
			return "", "", fmt.Errorf("private inbox path %s is not a mode-0700 directory", path)
		}
	}
	return inbox, metadata, nil
}

func writeNewPrivateFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	if _, err := file.Write(content); err != nil {
		_ = os.Remove(path)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	closed = true
	return nil
}

func safeUploadExtension(name string) string {
	extension := strings.ToLower(filepath.Ext(filepath.Base(strings.TrimSpace(name))))
	if len(extension) < 2 || len(extension) > 16 {
		return ".json"
	}
	for _, char := range extension[1:] {
		if char < 'a' || char > 'z' {
			return ".json"
		}
	}
	return extension
}

func safeOriginalName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "" || len(name) > 255 || strings.ContainsAny(name, "\r\n\x00") {
		return "uploaded-source" + safeUploadExtension(name)
	}
	return name
}

func sourceIDFromName(name, fallback string) string {
	base := strings.TrimSuffix(safeOriginalName(name), filepath.Ext(safeOriginalName(name)))
	base = strings.ToLower(base)
	var out strings.Builder
	for _, char := range base {
		if char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.' {
			out.WriteRune(char)
		} else if out.Len() > 0 && !strings.HasSuffix(out.String(), "-") {
			out.WriteByte('-')
		}
	}
	id := strings.Trim(out.String(), ".-_")
	if id == "" {
		id = "uploaded-" + fallback[:12]
	}
	return id
}

func (e *Engine) readUploadedSource(source UploadedSource) ([]byte, error) {
	path := filepath.Join(e.config.PrivateRoot, "openudon-upload-inbox", source.InboxFile)
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() > MaxUploadBytes {
		return nil, errors.New("private inbox source is unavailable or unsafe")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != source.SHA256 || credentialpolicy.ContainsLikelyValue(data) {
		return nil, errors.New("private inbox source changed or contains secret-like content")
	}
	return data, nil
}

func (e *Engine) loadAcquisitionState() error {
	registryPath := filepath.Join(e.workspaceRoot, ".icot", "ui-sources.json")
	data, err := os.ReadFile(registryPath)
	if err == nil {
		var registry stagedRegistry
		if err := strictAcquisitionJSON(data, &registry); err != nil || registry.Version != stagedRegistrySchema {
			return fmt.Errorf("load staged source registry: invalid registry")
		}
		for _, source := range registry.Sources {
			if !uploadIDPattern.MatchString(source.ID) || source.TargetPath == "" || source.SHA256 == "" {
				return fmt.Errorf("load staged source registry: invalid source record")
			}
			e.stagedSources[source.ID] = source
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if e.config.PrivateRoot == "" {
		return nil
	}
	_, metadataDir, err := e.ensureInboxDirectories()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		return err
	}
	if len(entries) > maxUploadedSources {
		return fmt.Errorf("private inbox metadata exceeds %d entries", maxUploadedSources)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".json" {
			return fmt.Errorf("private inbox metadata contains an unsafe entry")
		}
		data, err := os.ReadFile(filepath.Join(metadataDir, entry.Name()))
		if err != nil {
			return err
		}
		var record uploadRecord
		if err := strictAcquisitionJSON(data, &record); err != nil || record.Version != uploadRegistrySchema || !uploadIDPattern.MatchString(record.Source.ID) || record.File != filepath.Base(record.File) {
			return fmt.Errorf("private inbox metadata is invalid")
		}
		record.Source.InboxFile = record.File
		if _, err := e.readUploadedSource(record.Source); err != nil {
			return err
		}
		e.uploadedSources[record.Source.ID] = record.Source
	}
	return nil
}

func strictAcquisitionJSON(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func (e *Engine) uploadedSourceListLocked() []UploadedSource {
	result := make([]UploadedSource, 0, len(e.uploadedSources))
	for _, source := range e.uploadedSources {
		copy := source
		copy.InboxFile = ""
		result = append(result, copy)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (e *Engine) stagedSourceListLocked() []StagedSource {
	result := make([]StagedSource, 0, len(e.stagedSources))
	for _, source := range e.stagedSources {
		result = append(result, source)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (e *Engine) snapshotResultLocked() (Snapshot, error) {
	return cloneSnapshot(e.cachedSnapshot)
}

func (e *Engine) refreshCachedSnapshotLocked(ctx context.Context) (Snapshot, error) {
	snapshot, err := e.snapshotLocked(ctx)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	cached, err := cloneSnapshot(snapshot)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	e.cachedSnapshot = cached
	return cloneSnapshot(snapshot)
}

func (e *Engine) refreshAfterAcquisitionCommitLocked(ctx context.Context) (Snapshot, error) {
	if err := e.refreshLocked(ctx); err != nil {
		return Snapshot{}, classifyRefreshFailure(err)
	}
	snapshot, err := e.snapshotLocked(ctx)
	if err != nil {
		return Snapshot{}, rejected(err)
	}
	paths := watchedPaths(e.workspaceRoot, snapshot)
	baseline, err := captureWorkspace(ctx, e.workspaceRoot, paths)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	cached, err := cloneSnapshot(snapshot)
	if err != nil {
		return Snapshot{}, operational(err)
	}
	e.cachedSnapshot, e.watchedPaths, e.workspaceBaseline = cached, paths, baseline
	return cloneSnapshot(snapshot)
}

func cloneStagedSources(input map[string]StagedSource) map[string]StagedSource {
	result := make(map[string]StagedSource, len(input))
	for id, source := range input {
		result[id] = source
	}
	return result
}

func marshalStagedRegistry(sources map[string]StagedSource) ([]byte, error) {
	values := make([]StagedSource, 0, len(sources))
	for _, source := range sources {
		values = append(values, source)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	data, err := json.MarshalIndent(stagedRegistry{Version: stagedRegistrySchema, Sources: values}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func (e *Engine) removeUploadedSourceFiles(source UploadedSource) {
	if e.config.PrivateRoot == "" {
		return
	}
	_ = os.Remove(filepath.Join(e.config.PrivateRoot, "openudon-upload-metadata", source.ID+".json"))
	_ = os.Remove(filepath.Join(e.config.PrivateRoot, "openudon-upload-inbox", source.InboxFile))
}
