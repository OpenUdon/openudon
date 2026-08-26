package artifactwriter

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserregistration"
)

// GeneratedFile is one prepared mutation in an atomic authoring transaction.
type GeneratedFile struct {
	Path                  string
	Content               string
	AllowOverwrite        bool
	ExpectedCurrentSHA256 string
	Remove                bool
	Action                string
	Reason                string
}

// Prepared is a fully revalidated authoring transaction ready to commit.
type Prepared struct {
	ExampleRoot string
	Artifacts   elicitor.Artifacts
	Files       []GeneratedFile
}

// WriteConflict is one prepared mutation that requires explicit overwrite
// authority before CommitChecked will apply it.
type WriteConflict struct {
	Code   string `json:"code"`
	Action string `json:"action"`
	Path   string `json:"path"`
}

// Result describes the paths affected by a committed transaction.
type Result struct {
	Written         []string
	Removed         []string
	CleanupWarnings []string
}

// TransactionError reports a write whose rollback did not complete. Callers
// must treat the durable result as indeterminate and must not install a new
// in-memory state.
type TransactionError struct {
	Cause error
}

func (e *TransactionError) Error() string {
	return "authoring artifact transaction is indeterminate: " + e.Cause.Error()
}

func (e *TransactionError) Unwrap() error { return e.Cause }

// ProposedFileActions returns the exact package mutations in a prepared
// transaction. The list is derived from the same file plan Commit consumes.
func ProposedFileActions(prepared Prepared) []elicitor.FileAction {
	actions := make([]elicitor.FileAction, 0, len(prepared.Files))
	for _, file := range prepared.Files {
		action := strings.TrimSpace(file.Action)
		if action == "" {
			action = "write"
			if file.Remove {
				action = "remove_if_present"
			}
		}
		actions = append(actions, elicitor.FileAction{Action: action, Path: file.Path, Reason: file.Reason})
	}
	sortFileActions(actions)
	return actions
}

// WriteConflicts reports the exact regular-file collisions that the prepared
// transaction would reject without force. It performs no writes and applies
// the same path-safety checks as commit preflight.
func WriteConflicts(prepared Prepared) ([]WriteConflict, error) {
	root, err := validateTransactionPlan(prepared.ExampleRoot, prepared.Files)
	if err != nil {
		return nil, err
	}
	conflicts := make([]WriteConflict, 0)
	for _, file := range prepared.Files {
		if err := validateOutputPath(root, file.Path, false); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		info, err := os.Lstat(file.Path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("output path %s is not a regular file", file.Path)
		}
		if file.Remove || file.AllowOverwrite {
			continue
		}
		action := strings.TrimSpace(file.Action)
		if action == "" {
			action = "write"
		}
		conflicts = append(conflicts, WriteConflict{
			Code: "overwrite_required", Action: action, Path: file.Path,
		})
	}
	sort.SliceStable(conflicts, func(i, j int) bool {
		if conflicts[i].Path != conflicts[j].Path {
			return conflicts[i].Path < conflicts[j].Path
		}
		return conflicts[i].Action < conflicts[j].Action
	})
	return conflicts, nil
}

// PotentialFileActions returns the mutation shape available before artifacts
// can be rendered and prepared. Approval-capable snapshots use
// ProposedFileActions instead.
func PotentialFileActions(exampleDir string, session elicitor.Session, complete bool) []elicitor.FileAction {
	intentName := "workflows/intent.draft.hcl"
	if complete {
		intentName = "workflows/intent.hcl"
	}
	actions := []elicitor.FileAction{
		{Action: "write", Path: filepath.Join(exampleDir, "project.md"), Reason: "render the reviewed active boundary and candidate workflows"},
		{Action: "write", Path: filepath.Join(exampleDir, filepath.FromSlash(intentName)), Reason: "render the active workflow intent"},
	}
	hasBrowserSources := false
	hasAuthenticationSources := false
	hasRegistrationSources := false
	for _, source := range session.SourcePlan {
		actions = append(actions, elicitor.FileAction{Action: "copy", Path: filepath.Join(exampleDir, filepath.FromSlash(source.TargetPath)), Reason: source.Kind + " source " + source.ID + " with SHA-256 " + source.SHA256})
		if source.Kind == "browser-registration" && source.ReviewPath != "" {
			actions = append(actions, elicitor.FileAction{Action: "copy", Path: filepath.Join(exampleDir, filepath.FromSlash(source.ReviewPath)), Reason: "independent browser registration review for " + source.ID + " with SHA-256 " + source.ReviewSHA256})
		}
		if source.Kind == "browser-profile" {
			hasBrowserSources = true
		}
		if source.Kind == "browser-authentication" {
			hasAuthenticationSources = true
		}
		if source.Kind == "browser-registration" {
			hasRegistrationSources = true
		}
	}
	if hasBrowserSources {
		actions = append(actions, elicitor.FileAction{Action: "write", Path: filepath.Join(exampleDir, ".icot", "browser-sources.json"), Reason: "record safe browser origin, action, digest, lifecycle, optional value-free verification, session-posture, and approval evidence"})
	} else if complete {
		actions = append(actions, elicitor.FileAction{Action: "remove_if_present", Path: filepath.Join(exampleDir, ".icot", "browser-sources.json"), Reason: "remove stale browser source review metadata"})
	}
	if hasAuthenticationSources {
		actions = append(actions, elicitor.FileAction{Action: "write", Path: filepath.Join(exampleDir, ".icot", "browser-authentication.json"), Reason: "record safe browser authentication source, flow, credential-slot, session-binding, and approval evidence"})
	} else if complete {
		actions = append(actions, elicitor.FileAction{Action: "remove_if_present", Path: filepath.Join(exampleDir, ".icot", "browser-authentication.json"), Reason: "remove stale browser authentication review metadata"})
	}
	if hasRegistrationSources {
		actions = append(actions, elicitor.FileAction{Action: "write", Path: filepath.Join(exampleDir, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)), Reason: "record reviewed registration source, symbolic bindings, fixed duplicate/ambiguity/cleanup policy, timeout, and approval evidence"})
	} else if complete {
		actions = append(actions, elicitor.FileAction{Action: "remove_if_present", Path: filepath.Join(exampleDir, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)), Reason: "remove stale browser registration review metadata"})
	}
	if complete {
		actions = append(actions,
			elicitor.FileAction{Action: "remove_if_present", Path: filepath.Join(exampleDir, "workflows", "intent.draft.hcl"), Reason: "promote the completed draft"},
			elicitor.FileAction{Action: "remove_if_present", Path: filepath.Join(exampleDir, ".icot", "session.yaml"), Reason: "remove obsolete resumable draft state"},
			elicitor.FileAction{Action: "remove_if_present", Path: filepath.Join(exampleDir, ".icot", "readiness.json"), Reason: "remove obsolete generated draft readiness"},
		)
	} else {
		actions = append(actions,
			elicitor.FileAction{Action: "write", Path: filepath.Join(exampleDir, ".icot", "session.yaml"), Reason: "persist resumable incomplete authoring state"},
			elicitor.FileAction{Action: "write", Path: filepath.Join(exampleDir, ".icot", "readiness.json"), Reason: "persist incomplete authoring readiness and deferrals"},
		)
	}
	sortFileActions(actions)
	return actions
}

func sortFileActions(actions []elicitor.FileAction) {
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].Path != actions[j].Path {
			return actions[i].Path < actions[j].Path
		}
		return actions[i].Action < actions[j].Action
	})
}

// Prepare revalidates source evidence and builds the exact authoring
// transaction without mutating the example directory.
func Prepare(exampleDir string, artifacts elicitor.Artifacts, force bool, at time.Time) (Prepared, error) {
	if strings.TrimSpace(exampleDir) == "" {
		return Prepared{}, errors.New("example directory is required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	exampleRoot, err := canonicalExampleRoot(exampleDir)
	if err != nil {
		return Prepared{}, err
	}
	// Reject ambiguous supplied plans before normalization can coalesce
	// duplicate entries, then validate the exact revalidated plan below.
	if err := validateSourceMaterializationTargets(exampleRoot, artifacts.Session.SourcePlan); err != nil {
		return Prepared{}, err
	}
	artifacts.Session.Normalize()
	revalidatedSources, err := elicitor.RevalidateBrowserVerifications(artifacts.Session.SourcePlan, at)
	if err != nil {
		return Prepared{}, fmt.Errorf("revalidate browser verification evidence: %w", err)
	}
	artifacts.Session.SourcePlan = revalidatedSources
	if err := validateSourceMaterializationTargets(exampleRoot, artifacts.Session.SourcePlan); err != nil {
		return Prepared{}, err
	}
	if err := elicitor.ValidateBrowserVerificationCoverage(artifacts.Session); err != nil {
		return Prepared{}, fmt.Errorf("validate browser verification evidence: %w", err)
	}
	projectPath := filepath.Join(exampleRoot, "project.md")
	intentPath := filepath.Join(exampleRoot, "workflows", "intent.hcl")
	if artifacts.Incomplete {
		intentPath = filepath.Join(exampleRoot, "workflows", "intent.draft.hcl")
	}
	files := []GeneratedFile{
		{Path: projectPath, Content: artifacts.ProjectMD, Action: "write", Reason: "render the reviewed active boundary and candidate workflows"},
		{Path: intentPath, Content: artifacts.IntentHCL, Action: "write", Reason: "render the active workflow intent"},
	}
	browserMetadata, hasBrowserSources, err := BrowserSourceMetadataJSON(artifacts.Session)
	if err != nil {
		return Prepared{}, err
	}
	if hasBrowserSources {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "browser-sources.json"), Content: browserMetadata, AllowOverwrite: true, Action: "write", Reason: "record safe browser origin, action, digest, lifecycle, optional value-free verification, session-posture, and approval evidence"})
	} else if !artifacts.Incomplete {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "browser-sources.json"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove stale browser source review metadata"})
	}
	authenticationMetadata, hasAuthenticationSources, err := BrowserAuthenticationMetadataJSON(artifacts.Session)
	if err != nil {
		return Prepared{}, err
	}
	if hasAuthenticationSources {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "browser-authentication.json"), Content: authenticationMetadata, AllowOverwrite: true, Action: "write", Reason: "record safe browser authentication source, flow, credential-slot, session-binding, and approval evidence"})
	} else if !artifacts.Incomplete {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "browser-authentication.json"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove stale browser authentication review metadata"})
	}
	registrationMetadata, hasRegistrationSources, err := BrowserRegistrationMetadataJSON(artifacts.Session, at)
	if err != nil {
		return Prepared{}, err
	}
	if hasRegistrationSources {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleRoot, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)), Content: registrationMetadata, AllowOverwrite: true, Action: "write", Reason: "record reviewed registration source, symbolic bindings, fixed duplicate/ambiguity/cleanup policy, timeout, and approval evidence"})
	} else if !artifacts.Incomplete {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleRoot, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove stale browser registration review metadata"})
	}
	if artifacts.Incomplete {
		sessionData, err := json.MarshalIndent(artifacts.Session, "", "  ")
		if err != nil {
			return Prepared{}, err
		}
		readinessData, err := json.MarshalIndent(struct {
			Version   string `json:"version"`
			Interview any    `json:"interview"`
			Deferrals any    `json:"deferrals"`
		}{Version: "openudon.icot-readiness.v2", Interview: artifacts.Session.Interview, Deferrals: artifacts.Session.Interview.Deferrals}, "", "  ")
		if err != nil {
			return Prepared{}, err
		}
		files = append(files,
			GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "session.yaml"), Content: string(sessionData) + "\n", AllowOverwrite: true, Action: "write", Reason: "persist resumable incomplete authoring state"},
			GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "readiness.json"), Content: string(readinessData) + "\n", AllowOverwrite: true, Action: "write", Reason: "persist incomplete authoring readiness and deferrals"},
		)
	} else {
		files = append(files,
			GeneratedFile{Path: filepath.Join(exampleRoot, "workflows", "intent.draft.hcl"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "promote the completed draft"},
			GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "session.yaml"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove obsolete resumable draft state"},
			GeneratedFile{Path: filepath.Join(exampleRoot, ".icot", "readiness.json"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove obsolete generated draft readiness"},
		)
	}
	for _, source := range artifacts.Session.SourcePlan {
		target, err := SafeExampleTarget(exampleRoot, source.TargetPath)
		if err != nil {
			return Prepared{}, err
		}
		data, err := elicitor.SourceMaterializationContent(source, at)
		if err != nil {
			return Prepared{}, fmt.Errorf("read selected source %s: %w", source.SourcePath, err)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		if digest != strings.ToLower(source.SHA256) {
			return Prepared{}, fmt.Errorf("selected source %s changed after discovery: digest %s, want %s", source.SourcePath, digest, source.SHA256)
		}
		sourceAlreadyCurrent := false
		if existing, err := os.ReadFile(target); err == nil {
			if fmt.Sprintf("%x", sha256.Sum256(existing)) == digest {
				if source.Kind != "browser-registration" {
					continue
				}
				sourceAlreadyCurrent = true
			}
			if !sourceAlreadyCurrent && !force {
				return Prepared{}, fmt.Errorf("source target %s contains different content; pass --force to replace it", target)
			}
		} else if !os.IsNotExist(err) {
			return Prepared{}, err
		}
		if !sourceAlreadyCurrent {
			files = append(files, GeneratedFile{Path: target, Content: string(data), Action: "copy", Reason: source.Kind + " source " + source.ID + " with SHA-256 " + source.SHA256})
		}
		if source.Kind == "browser-registration" {
			reviewTarget, err := SafeExampleTarget(exampleRoot, source.ReviewPath)
			if err != nil {
				return Prepared{}, err
			}
			reviewData, err := elicitor.SourceMaterializationReviewContent(source, at)
			if err != nil {
				return Prepared{}, fmt.Errorf("read selected registration review %s: %w", source.SourcePath, err)
			}
			if existing, err := os.ReadFile(reviewTarget); err == nil {
				if fmt.Sprintf("%x", sha256.Sum256(existing)) == strings.ToLower(source.ReviewSHA256) {
					continue
				}
				if !force {
					return Prepared{}, fmt.Errorf("source target %s contains different content; pass --force to replace it", reviewTarget)
				}
			} else if !os.IsNotExist(err) {
				return Prepared{}, err
			}
			files = append(files, GeneratedFile{Path: reviewTarget, Content: string(reviewData), Action: "copy", Reason: "independent browser registration review for " + source.ID + " with SHA-256 " + source.ReviewSHA256})
		}
	}
	prepared := Prepared{ExampleRoot: exampleRoot, Artifacts: artifacts, Files: files}
	if _, err := validateTransactionPlan(prepared.ExampleRoot, prepared.Files); err != nil {
		return Prepared{}, err
	}
	return prepared, nil
}

// Commit atomically applies a prepared authoring transaction.
func Commit(prepared Prepared, force bool) (Result, error) {
	return CommitChecked(prepared, force, nil)
}

// CommitChecked atomically applies a prepared transaction. The optional
// beforeReplace check runs after staging and immediately before the first
// artifact replacement, allowing an engine to bind the commit to its accepted
// workspace fingerprint.
func CommitChecked(prepared Prepared, force bool, beforeReplace func() error) (Result, error) {
	var cleanupWarnings []string
	if err := writeFilesAtomicReporting(prepared.ExampleRoot, prepared.Files, force, beforeReplace, func(err error) {
		cleanupWarnings = append(cleanupWarnings, err.Error())
	}); err != nil {
		return Result{}, err
	}
	result := Result{CleanupWarnings: cleanupWarnings}
	for _, file := range prepared.Files {
		if file.Remove {
			result.Removed = append(result.Removed, file.Path)
		} else {
			result.Written = append(result.Written, file.Path)
		}
	}
	sort.Strings(result.Written)
	sort.Strings(result.Removed)
	return result, nil
}

// WriteFilesAtomic validates and applies a set of file mutations as one
// rollback-capable transaction.
func WriteFilesAtomic(files []GeneratedFile, force bool) error {
	return writeFilesAtomic("", files, force, nil)
}

func writeFilesAtomic(exampleRoot string, files []GeneratedFile, force bool, beforeReplace func() error) error {
	return writeFilesAtomicReporting(exampleRoot, files, force, beforeReplace, nil)
}

func writeFilesAtomicReporting(exampleRoot string, files []GeneratedFile, force bool, beforeReplace func() error, reportCleanup func(error)) error {
	// Validate the complete transaction before creating a directory, temporary
	// file, or backup. Ambiguous plans must be byte-for-byte read-only failures.
	root, err := validateTransactionPlan(exampleRoot, files)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.ExpectedCurrentSHA256 != "" && !file.AllowOverwrite {
			return fmt.Errorf("output path %s has an expected digest without overwrite authority", file.Path)
		}
		if err := validateOutputPath(root, file.Path, false); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		info, err := os.Lstat(file.Path)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path %s is a symlink", file.Path)
		}
		if err == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("output path %s is not a regular file", file.Path)
		}
		if err == nil && !force && !file.AllowOverwrite {
			return fmt.Errorf("%s already exists; pass --force to overwrite it", file.Path)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := validateExpectedCurrentSHA256(file); err != nil {
			return err
		}
	}
	// Preserve the established authoring workspace layout, but only after the
	// complete plan and every overwrite precondition have been accepted.
	for _, relative := range []string{"openapi", "workflows", "expected"} {
		if err := createSafeDirectories(root, filepath.Join(root, relative)); err != nil {
			return err
		}
	}
	for _, file := range files {
		if file.Remove {
			continue
		}
		if err := validateOutputPath(root, file.Path, true); err != nil {
			return err
		}
	}
	tmpPaths := map[string]string{}
	for _, file := range files {
		if file.Remove {
			continue
		}
		tmp, err := os.CreateTemp(filepath.Dir(file.Path), "."+filepath.Base(file.Path)+".tmp.")
		if err != nil {
			cleanupTemps(tmpPaths)
			return err
		}
		tmpPaths[file.Path] = tmp.Name()
		_, writeErr := tmp.WriteString(file.Content)
		syncErr := tmp.Sync()
		closeErr := tmp.Close()
		if writeErr != nil {
			cleanupTemps(tmpPaths)
			return writeErr
		}
		if syncErr != nil || closeErr != nil {
			cleanupTemps(tmpPaths)
			return errors.Join(syncErr, closeErr)
		}
	}
	backups := map[string]fileBackup{}
	for _, file := range files {
		info, err := os.Lstat(file.Path)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				cleanupTemps(tmpPaths)
				cleanupBackups(backups)
				return fmt.Errorf("output path %s changed to an unsafe file type", file.Path)
			}
			backupPath, err := backupFilePath(file.Path)
			if err != nil {
				cleanupTemps(tmpPaths)
				cleanupBackups(backups)
				return err
			}
			backups[file.Path] = fileBackup{backupPath: backupPath, existed: true}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupTemps(tmpPaths)
			cleanupBackups(backups)
			return err
		}
	}
	for _, file := range files {
		if err := validateOutputPath(root, file.Path, false); err != nil && !(file.Remove && errors.Is(err, os.ErrNotExist)) {
			cleanupTemps(tmpPaths)
			cleanupBackups(backups)
			return err
		}
	}
	var renamed []string
	for index, file := range files {
		if err := validateOutputPath(root, file.Path, false); err != nil && !(file.Remove && errors.Is(err, os.ErrNotExist)) {
			return rollbackFailure(err, backups, renamed, tmpPaths)
		}
		if index == 0 && beforeReplace != nil {
			if err := beforeReplace(); err != nil {
				cleanupTemps(tmpPaths)
				if cleanupErr := cleanupBackups(backups); cleanupErr != nil {
					return errors.Join(err, cleanupErr)
				}
				return err
			}
		}
		if err := validateExpectedCurrentSHA256(file); err != nil {
			return rollbackFailure(err, backups, renamed, tmpPaths)
		}
		var err error
		changed := true
		createOnly := !file.Remove && !force && !file.AllowOverwrite && !backups[file.Path].existed
		if file.Remove {
			err = removeFile(file.Path)
			if os.IsNotExist(err) {
				err = nil
				changed = false
			}
		} else if createOnly {
			err = linkFile(tmpPaths[file.Path], file.Path)
		} else {
			err = renameFile(tmpPaths[file.Path], file.Path)
		}
		if err != nil {
			return rollbackFailure(err, backups, renamed, tmpPaths)
		}
		if !changed {
			continue
		}
		renamed = append(renamed, file.Path)
		if createOnly {
			if err := os.Remove(tmpPaths[file.Path]); err != nil {
				return rollbackFailure(err, backups, renamed, tmpPaths)
			}
			delete(tmpPaths, file.Path)
		}
		if err := syncTransactionDirectory(filepath.Dir(file.Path)); err != nil {
			return rollbackFailure(err, backups, renamed, tmpPaths)
		}
	}
	cleanupTemps(tmpPaths)
	if err := cleanupBackups(backups); err != nil {
		if reportCleanup != nil {
			reportCleanup(fmt.Errorf("remove committed transaction backups: %w", err))
		}
	}
	for _, file := range files {
		if file.Remove {
			_ = os.Remove(filepath.Dir(file.Path))
		}
	}
	return nil
}

func validateExpectedCurrentSHA256(file GeneratedFile) error {
	expected := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(file.ExpectedCurrentSHA256)), "sha256:")
	if expected == "" {
		return nil
	}
	if len(expected) != sha256.Size*2 {
		return fmt.Errorf("output path %s has an invalid expected SHA-256", file.Path)
	}
	for _, character := range expected {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return fmt.Errorf("output path %s has an invalid expected SHA-256", file.Path)
		}
	}
	before, err := os.Lstat(file.Path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fmt.Errorf("output path %s no longer matches its prepared bytes", file.Path)
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return fmt.Errorf("read expected output path %s: %w", file.Path, err)
	}
	after, err := os.Lstat(file.Path)
	if err != nil || after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(before, after) || before.Size() != after.Size() || before.ModTime() != after.ModTime() || int64(len(data)) != after.Size() {
		return fmt.Errorf("output path %s changed while its prepared bytes were checked", file.Path)
	}
	digest := sha256.Sum256(data)
	if fmt.Sprintf("%x", digest[:]) != expected {
		return fmt.Errorf("output path %s no longer matches its prepared bytes", file.Path)
	}
	return nil
}

// BrowserSourceMetadataJSON renders the reviewed browser capability metadata
// included in approved authoring transactions.
func BrowserSourceMetadataJSON(session elicitor.Session) (string, bool, error) {
	type reviewedSource struct {
		ID                 string                  `json:"id"`
		Release            string                  `json:"release,omitempty"`
		TargetPath         string                  `json:"target_path"`
		SHA256             string                  `json:"sha256"`
		SourceSHA256       string                  `json:"source_sha256,omitempty"`
		Title              string                  `json:"title,omitempty"`
		Actions            []string                `json:"actions"`
		Origins            []string                `json:"origins"`
		Lifecycle          string                  `json:"lifecycle"`
		ExpiresAt          string                  `json:"expires_at,omitempty"`
		LoginStateRequired bool                    `json:"login_state_required,omitempty"`
		Provenance         string                  `json:"provenance"`
		Registry           string                  `json:"registry,omitempty"`
		Coordinate         string                  `json:"coordinate,omitempty"`
		Verifications      []browserverify.Summary `json:"verifications,omitempty"`
	}
	var sources []reviewedSource
	for _, source := range session.SourcePlan {
		if source.Kind != "browser-profile" {
			continue
		}
		verifications := make([]browserverify.Summary, 0, len(source.BrowserVerifications))
		for _, value := range source.BrowserVerifications {
			verifications = append(verifications, value.Summary)
		}
		sources = append(sources, reviewedSource{
			ID: source.ID, Release: source.Release, TargetPath: source.TargetPath, SHA256: source.SHA256,
			SourceSHA256: source.SourceSHA256, Title: source.Title, Actions: append([]string(nil), source.Actions...),
			Origins: append([]string(nil), source.Origins...), Lifecycle: source.Lifecycle, ExpiresAt: source.ExpiresAt,
			LoginStateRequired: source.LoginStateRequired, Provenance: source.Provenance,
			Registry: source.Registry, Coordinate: source.RegistryCoordinate, Verifications: verifications,
		})
	}
	if len(sources) == 0 {
		return "", false, nil
	}
	data, err := json.MarshalIndent(struct {
		Version           string           `json:"version"`
		Route             string           `json:"route"`
		SessionPosture    string           `json:"session_posture"`
		MutationApprovals []string         `json:"mutation_approvals,omitempty"`
		Sources           []reviewedSource `json:"sources"`
	}{
		Version: "openudon.browser-source-review.v1", Route: session.BrowserRoute,
		SessionPosture: session.BrowserSession, MutationApprovals: append([]string(nil), session.BrowserApprovals...), Sources: sources,
	}, "", "  ")
	if err != nil {
		return "", false, err
	}
	if len(data)+1 > browserverify.MaxReviewBytes {
		return "", false, fmt.Errorf("browser source review exceeds %d bytes", browserverify.MaxReviewBytes)
	}
	return string(append(data, '\n')), true, nil
}

// BrowserRegistrationMetadataJSON renders the exact value-free registration
// source and call inventory consumed by package quality and trusted dry-run.
func BrowserRegistrationMetadataJSON(session elicitor.Session, at time.Time) (string, bool, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	type reviewedCall struct {
		Step                string            `json:"step"`
		Source              string            `json:"source"`
		Flow                string            `json:"flow"`
		CredentialBindings  map[string]string `json:"credential_bindings"`
		Approval            string            `json:"approval"`
		DuplicatePrevention string            `json:"duplicate_prevention"`
		OnDuplicate         string            `json:"on_duplicate"`
		AmbiguousOutcome    string            `json:"ambiguous_outcome"`
		CleanupDisposition  string            `json:"cleanup_disposition"`
		Timeout             float64           `json:"timeout"`
	}
	type reviewedSource struct {
		ID                  string              `json:"id"`
		TargetPath          string              `json:"target_path"`
		SHA256              string              `json:"sha256"`
		ReviewPath          string              `json:"review_path"`
		ReviewSHA256        string              `json:"review_sha256"`
		ProfileDigest       string              `json:"profile_digest"`
		Title               string              `json:"title"`
		Flows               []string            `json:"flows"`
		FlowCredentialSlots map[string][]string `json:"flow_credential_slots"`
		Origins             []string            `json:"origins"`
		Lifecycle           string              `json:"lifecycle"`
		ExpiresAt           string              `json:"expires_at"`
		Provenance          string              `json:"provenance"`
	}
	byTarget := map[string]*registrationprofile.Profile{}
	var sources []reviewedSource
	for _, source := range session.SourcePlan {
		if source.Kind != "browser-registration" {
			continue
		}
		data, err := elicitor.SourceMaterializationContent(source, at)
		if err != nil {
			return "", false, err
		}
		if _, err := elicitor.SourceMaterializationReviewContent(source, at); err != nil {
			return "", false, err
		}
		value, err := registrationprofile.Parse(data)
		if err != nil {
			return "", false, err
		}
		if err := registrationprofile.ValidateAt(value, at); err != nil {
			return "", false, err
		}
		profileDigest, err := registrationprofile.Digest(value)
		if err != nil {
			return "", false, err
		}
		expiresAt, err := registrationprofile.ExpiresAt(value)
		if err != nil {
			return "", false, err
		}
		flowCredentialSlots := make(map[string][]string, len(value.Flows))
		for flowName, flow := range value.Flows {
			flowCredentialSlots[flowName] = registrationProfileFlowSlots(flow)
		}
		byTarget[filepath.ToSlash(source.TargetPath)] = value
		sources = append(sources, reviewedSource{
			ID: source.ID, TargetPath: filepath.ToSlash(source.TargetPath), SHA256: source.SHA256,
			ReviewPath: filepath.ToSlash(source.ReviewPath), ReviewSHA256: source.ReviewSHA256, ProfileDigest: profileDigest,
			Title: value.Info.Title, Flows: registrationprofile.SortedFlowNames(value), FlowCredentialSlots: flowCredentialSlots,
			Origins: registrationprofile.Origins(value), Lifecycle: source.Lifecycle, ExpiresAt: expiresAt.Format(time.RFC3339), Provenance: value.Evidence.Source,
		})
	}
	if len(sources) == 0 {
		return "", false, nil
	}
	var calls []reviewedCall
	var callErr error
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if callErr != nil || step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "browser_registration") {
			return
		}
		source := filepath.ToSlash(strings.TrimSpace(firstNonEmptyArtifactWriter(step.Source, session.Intent.Source)))
		value := byTarget[source]
		if value == nil {
			callErr = fmt.Errorf("registration step %s references unavailable source %s", step.Name, source)
			return
		}
		flow, ok := value.Flows[strings.TrimSpace(step.RegistrationFlow)]
		if !ok || step.Timeout == nil || strings.TrimSpace(step.BrowserSession) != "" {
			callErr = fmt.Errorf("registration step %s is incomplete or carries a session", step.Name)
			return
		}
		slots := registrationProfileFlowSlots(flow)
		if !exactRegistrationCredentialBindings(step.CredentialBindings, slots) {
			callErr = fmt.Errorf("registration step %s symbolic bindings do not exactly cover its flow", step.Name)
			return
		}
		calls = append(calls, reviewedCall{
			Step: step.Name, Source: source, Flow: step.RegistrationFlow, CredentialBindings: step.CredentialBindings,
			Approval: step.RegistrationApproval, DuplicatePrevention: step.DuplicatePrevention, OnDuplicate: step.OnDuplicate,
			AmbiguousOutcome: step.AmbiguousOutcome, CleanupDisposition: step.CleanupDisposition, Timeout: *step.Timeout,
		})
	})
	if callErr != nil {
		return "", false, callErr
	}
	sort.Slice(calls, func(i, j int) bool { return calls[i].Step < calls[j].Step })
	data, err := json.MarshalIndent(struct {
		Version string           `json:"version"`
		Calls   []reviewedCall   `json:"registration_calls"`
		Sources []reviewedSource `json:"sources"`
	}{Version: "openudon.browser-registration-review.v1", Calls: calls, Sources: sources}, "", "  ")
	if err != nil {
		return "", false, err
	}
	return string(append(data, '\n')), true, nil
}

func registrationProfileFlowSlots(flow browserregistration.Flow) []string {
	set := map[string]bool{}
	for _, step := range flow.Sequence {
		if step.TypeCredential != nil {
			set[step.TypeCredential.Slot] = true
		}
	}
	result := make([]string, 0, len(set))
	for slot := range set {
		result = append(result, slot)
	}
	sort.Strings(result)
	return result
}

func exactRegistrationCredentialBindings(bindings map[string]string, slots []string) bool {
	if len(bindings) != len(slots) {
		return false
	}
	for _, slot := range slots {
		if strings.TrimSpace(bindings[slot]) == "" {
			return false
		}
	}
	return true
}

func firstNonEmptyArtifactWriter(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// BrowserAuthenticationMetadataJSON renders the reviewed browser
// authentication metadata included in approved authoring transactions.
func BrowserAuthenticationMetadataJSON(session elicitor.Session) (string, bool, error) {
	type reviewedSource struct {
		ID                  string              `json:"id"`
		TargetPath          string              `json:"target_path"`
		SHA256              string              `json:"sha256"`
		SourceSHA256        string              `json:"source_sha256,omitempty"`
		Title               string              `json:"title,omitempty"`
		Flows               []string            `json:"flows"`
		FlowCredentialSlots map[string][]string `json:"flow_credential_slots"`
		Origins             []string            `json:"origins"`
		Lifecycle           string              `json:"lifecycle"`
		ExpiresAt           string              `json:"expires_at"`
		Provenance          string              `json:"provenance"`
	}
	type sessionBinding struct {
		Step    string `json:"step"`
		Session string `json:"session"`
	}
	var sources []reviewedSource
	for _, source := range session.SourcePlan {
		if source.Kind != "browser-authentication" {
			continue
		}
		sources = append(sources, reviewedSource{
			ID: source.ID, TargetPath: source.TargetPath, SHA256: source.SHA256, SourceSHA256: source.SourceSHA256,
			Title: source.Title, Flows: append([]string(nil), source.Flows...), FlowCredentialSlots: source.FlowCredentialSlots,
			Origins: append([]string(nil), source.Origins...), Lifecycle: source.Lifecycle, ExpiresAt: source.ExpiresAt, Provenance: source.Provenance,
		})
	}
	if len(sources) == 0 {
		return "", false, nil
	}
	var sessions []sessionBinding
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.BrowserSession) != "" {
			sessions = append(sessions, sessionBinding{Step: step.Name, Session: step.BrowserSession})
		}
	})
	data, err := json.MarshalIndent(struct {
		Version   string           `json:"version"`
		Approvals []string         `json:"authentication_approvals"`
		Sessions  []sessionBinding `json:"session_bindings"`
		Sources   []reviewedSource `json:"sources"`
	}{
		Version: "openudon.browser-authentication-review.v1", Approvals: append([]string(nil), session.BrowserAuthenticationApprovals...), Sessions: sessions, Sources: sources,
	}, "", "  ")
	if err != nil {
		return "", false, err
	}
	return string(append(data, '\n')), true, nil
}

// SafeExampleTarget resolves a package-relative source target beneath an
// example directory.
func SafeExampleTarget(exampleDir, relative string) (string, error) {
	if strings.TrimSpace(relative) == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("source target %q must be a relative package path", relative)
	}
	base, err := canonicalExampleRoot(exampleDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(base, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	if !pathWithin(base, target) {
		return "", fmt.Errorf("source target %q escapes example directory", relative)
	}
	if err := validateOutputPath(base, target, false); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return target, nil
}

func walkSteps(steps []*rollout.Step, visit func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		visit(step)
		walkSteps(step.Steps, visit)
		for _, branch := range step.Cases {
			if branch != nil {
				walkSteps(branch.Steps, visit)
			}
		}
		if step.Default != nil {
			walkSteps(step.Default.Steps, visit)
		}
	}
}

type fileBackup struct {
	backupPath string
	existed    bool
}

var (
	renameFile = os.Rename
	linkFile   = os.Link
	removeFile = os.Remove
)

func validateSourceMaterializationTargets(root string, sources []elicitor.SourceMaterialization) error {
	paths := make([]string, 0, len(sources))
	digests := make(map[string]string, len(sources))
	for _, source := range sources {
		target, err := SafeExampleTarget(root, source.TargetPath)
		if err != nil {
			return fmt.Errorf("invalid source materialization target %q: %w", source.TargetPath, err)
		}
		relative, err := filepath.Rel(root, target)
		if err != nil {
			return fmt.Errorf("resolve source materialization target %q: %w", source.TargetPath, err)
		}
		relative = filepath.ToSlash(relative)
		if reservedSourceTarget(relative) {
			return fmt.Errorf("source materialization target %q is reserved for iCoT authoring state or artifacts", relative)
		}
		if prior, ok := digests[target]; ok && !strings.EqualFold(prior, source.SHA256) {
			return fmt.Errorf("source target %s is selected with different content digests %s and %s; choose one explicit source", relative, prior, source.SHA256)
		}
		digests[target] = source.SHA256
		paths = append(paths, target)
		if source.Kind == "browser-registration" {
			expectedReviewPath := strings.TrimSuffix(filepath.ToSlash(source.TargetPath), filepath.Ext(source.TargetPath)) + ".review.json"
			if filepath.ToSlash(source.ReviewPath) != expectedReviewPath {
				return fmt.Errorf("registration review materialization target %q must be adjacent to source %q", source.ReviewPath, source.TargetPath)
			}
			reviewTarget, err := SafeExampleTarget(root, source.ReviewPath)
			if err != nil {
				return fmt.Errorf("invalid registration review materialization target %q: %w", source.ReviewPath, err)
			}
			relative, err := filepath.Rel(root, reviewTarget)
			if err != nil {
				return err
			}
			if reservedSourceTarget(filepath.ToSlash(relative)) || strings.TrimSpace(source.ReviewSHA256) == "" {
				return fmt.Errorf("registration review materialization target %q or digest is invalid", source.ReviewPath)
			}
			if prior, ok := digests[reviewTarget]; ok && !strings.EqualFold(prior, source.ReviewSHA256) {
				return fmt.Errorf("registration review target %s is selected with different content digests", relative)
			}
			digests[reviewTarget] = source.ReviewSHA256
			paths = append(paths, reviewTarget)
		} else if source.ReviewPath != "" || source.ReviewSHA256 != "" || len(source.MaterializedReview) != 0 {
			return fmt.Errorf("non-registration source %s carries registration review materialization", source.ID)
		}
	}
	if err := validateDistinctOutputPaths(root, paths, "source materialization"); err != nil {
		return err
	}
	return nil
}

func reservedSourceTarget(relative string) bool {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(relative))))
	parts := strings.Split(clean, "/")
	if len(parts) > 0 && strings.EqualFold(parts[0], ".icot") {
		return true
	}
	for _, reserved := range []string{"project.md", "workflows/intent.hcl", "workflows/intent.draft.hcl"} {
		if strings.EqualFold(clean, reserved) {
			return true
		}
	}
	return false
}

// validateTransactionPlan is the shared, read-only preflight for Prepare,
// WriteConflicts, and commit. It rejects any plan whose map of output paths is
// ambiguous on either case-sensitive or case-insensitive filesystems.
func validateTransactionPlan(exampleRoot string, files []GeneratedFile) (string, error) {
	root, err := transactionRoot(exampleRoot, files)
	if err != nil {
		return "", err
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if err := validateGeneratedFile(file); err != nil {
			return "", err
		}
		path, err := filepath.Abs(file.Path)
		if err != nil {
			return "", err
		}
		path = filepath.Clean(path)
		if err := validateOutputPath(root, path, false); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if info, statErr := os.Lstat(path); statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("output path %s is a symlink", path)
			}
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("output path %s is not a regular file", path)
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}
		paths = append(paths, path)
	}
	if err := validateDistinctOutputPaths(root, paths, "prepared transaction"); err != nil {
		return "", err
	}
	return root, nil
}

func validateDistinctOutputPaths(root string, paths []string, planName string) error {
	cleaned := make([]string, len(paths))
	components := make([][]string, len(paths))
	for index, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		abs = filepath.Clean(abs)
		if abs == root {
			return fmt.Errorf("%s output path %s must not be the canonical example root itself", planName, path)
		}
		if !pathWithin(root, abs) {
			return fmt.Errorf("%s output path %s is outside the canonical example root", planName, path)
		}
		relative, err := filepath.Rel(root, abs)
		if err != nil {
			return err
		}
		cleaned[index] = abs
		components[index] = splitPathComponents(relative)
	}
	for left := 0; left < len(cleaned); left++ {
		for right := left + 1; right < len(cleaned); right++ {
			if equalPathComponentsFold(components[left], components[right]) {
				return fmt.Errorf("%s has duplicate or case-insensitive-equivalent output paths %s and %s", planName, cleaned[left], cleaned[right])
			}
			if pathComponentsAncestorFold(components[left], components[right]) || pathComponentsAncestorFold(components[right], components[left]) {
				return fmt.Errorf("%s has overlapping ancestor and descendant output paths %s and %s", planName, cleaned[left], cleaned[right])
			}
		}
	}
	return nil
}

func splitPathComponents(path string) []string {
	clean := filepath.Clean(path)
	if clean == "." {
		return nil
	}
	return strings.Split(clean, string(filepath.Separator))
}

func equalPathComponentsFold(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !strings.EqualFold(left[index], right[index]) {
			return false
		}
	}
	return true
}

func pathComponentsAncestorFold(ancestor, descendant []string) bool {
	if len(ancestor) >= len(descendant) {
		return false
	}
	for index := range ancestor {
		if !strings.EqualFold(ancestor[index], descendant[index]) {
			return false
		}
	}
	return true
}

func validateGeneratedFile(file GeneratedFile) error {
	if strings.TrimSpace(file.Path) == "" {
		return errors.New("empty output path")
	}
	if filepath.Base(file.Path) == "intent.hcl" {
		_, err := rollout.ParseIntent([]byte(file.Content), file.Path)
		return err
	}
	return nil
}

func cleanupTemps(paths map[string]string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func restoreBackups(backups map[string]fileBackup, renamed []string) error {
	var failures []error
	for i := len(renamed) - 1; i >= 0; i-- {
		path := renamed[i]
		backup := backups[path]
		if backup.existed {
			if err := removeFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, fmt.Errorf("remove changed output %s: %w", path, err))
				continue
			}
			if err := renameFile(backup.backupPath, path); err != nil {
				failures = append(failures, fmt.Errorf("restore output %s: %w", path, err))
			}
		} else {
			if err := removeFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				failures = append(failures, fmt.Errorf("remove new output %s: %w", path, err))
			}
		}
		if err := syncTransactionDirectory(filepath.Dir(path)); err != nil {
			failures = append(failures, fmt.Errorf("sync restored output directory %s: %w", filepath.Dir(path), err))
		}
	}
	return errors.Join(failures...)
}

func backupFilePath(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("output path %s is not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".backup.")
	if err != nil {
		return "", err
	}
	backupPath := file.Name()
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		_ = file.Close()
		_ = os.Remove(backupPath)
		return "", err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(backupPath)
		return "", errors.Join(writeErr, syncErr, closeErr)
	}
	return backupPath, nil
}

func syncTransactionDirectory(directory string) error {
	value, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer value.Close()
	if err := value.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) && !errors.Is(err, syscall.EBADF) {
		return err
	}
	return nil
}

func rollbackFailure(cause error, backups map[string]fileBackup, changed []string, temps map[string]string) error {
	rollbackErr := restoreBackups(backups, changed)
	cleanupTemps(temps)
	if rollbackErr != nil {
		return &TransactionError{Cause: errors.Join(cause, rollbackErr)}
	}
	if cleanupErr := cleanupBackups(backups); cleanupErr != nil {
		return &TransactionError{Cause: errors.Join(cause, cleanupErr)}
	}
	return cause
}

func cleanupBackups(backups map[string]fileBackup) error {
	var failures []error
	for _, backup := range backups {
		if backup.backupPath == "" {
			continue
		}
		if err := removeFile(backup.backupPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, err)
			continue
		}
		if err := syncTransactionDirectory(filepath.Dir(backup.backupPath)); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func transactionRoot(exampleRoot string, files []GeneratedFile) (string, error) {
	if strings.TrimSpace(exampleRoot) != "" {
		return canonicalExampleRoot(exampleRoot)
	}
	if len(files) == 0 {
		return "", errors.New("authoring transaction has no files")
	}
	paths := make([]string, 0, len(files))
	for _, file := range files {
		path, err := filepath.Abs(file.Path)
		if err != nil {
			return "", err
		}
		paths = append(paths, filepath.Clean(path))
	}
	root := filepath.Dir(paths[0])
	for _, path := range paths[1:] {
		for !pathWithin(root, path) {
			parent := filepath.Dir(root)
			if parent == root {
				return "", errors.New("generated file paths do not share a safe transaction root")
			}
			root = parent
		}
	}
	return canonicalExampleRoot(root)
}

func canonicalExampleRoot(exampleDir string) (string, error) {
	if strings.TrimSpace(exampleDir) == "" {
		return "", errors.New("example directory is required")
	}
	abs, err := filepath.Abs(exampleDir)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	probe := abs
	var suffix []string
	for {
		info, statErr := os.Lstat(probe)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("example root component %s is a symlink", probe)
			}
			if !info.IsDir() {
				return "", fmt.Errorf("example root component %s is not a directory", probe)
			}
			resolved, resolveErr := filepath.EvalSymlinks(probe)
			if resolveErr != nil {
				return "", resolveErr
			}
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", statErr
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", statErr
		}
		suffix = append(suffix, filepath.Base(probe))
		probe = parent
	}
}

func validateOutputPath(root, path string, createParents bool) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if abs == root {
		return fmt.Errorf("output path %s must not be the canonical example root itself", path)
	}
	if !pathWithin(root, abs) {
		return fmt.Errorf("output path %s is outside the canonical example root", path)
	}
	parent := filepath.Dir(abs)
	if createParents {
		if err := createSafeDirectories(root, parent); err != nil {
			return err
		}
	}
	if err := validateDirectoryChain(root, parent); err != nil {
		return err
	}
	if info, statErr := os.Lstat(abs); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("output path %s is a symlink", abs)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	return nil
}

func createSafeDirectories(root, target string) error {
	if !pathWithin(root, target) {
		return fmt.Errorf("directory %s is outside the canonical example root", target)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := validateDirectoryChain(root, root); err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, target)
	current := root
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("output path ancestor %s is not a safe directory", current)
		}
	}
	return nil
}

func validateDirectoryChain(root, target string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("canonical example root %s is not a safe directory", root)
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("directory %s is outside the canonical example root", target)
	}
	current := root
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("output path ancestor %s is not a safe directory", current)
		}
	}
	return nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
