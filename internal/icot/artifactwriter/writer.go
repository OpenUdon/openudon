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
	"time"

	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// GeneratedFile is one prepared mutation in an atomic authoring transaction.
type GeneratedFile struct {
	Path           string
	Content        string
	AllowOverwrite bool
	Remove         bool
	Action         string
	Reason         string
}

// Prepared is a fully revalidated authoring transaction ready to commit.
type Prepared struct {
	Artifacts elicitor.Artifacts
	Files     []GeneratedFile
}

// Result describes the paths affected by a committed transaction.
type Result struct {
	Written []string
	Removed []string
}

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
	for _, source := range session.SourcePlan {
		actions = append(actions, elicitor.FileAction{Action: "copy", Path: filepath.Join(exampleDir, filepath.FromSlash(source.TargetPath)), Reason: source.Kind + " source " + source.ID + " with SHA-256 " + source.SHA256})
		if source.Kind == "browser-profile" {
			hasBrowserSources = true
		}
		if source.Kind == "browser-authentication" {
			hasAuthenticationSources = true
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
	artifacts.Session.Normalize()
	revalidatedSources, err := elicitor.RevalidateBrowserVerifications(artifacts.Session.SourcePlan, at)
	if err != nil {
		return Prepared{}, fmt.Errorf("revalidate browser verification evidence: %w", err)
	}
	artifacts.Session.SourcePlan = revalidatedSources
	if err := elicitor.ValidateBrowserVerificationCoverage(artifacts.Session); err != nil {
		return Prepared{}, fmt.Errorf("validate browser verification evidence: %w", err)
	}
	selectedTargets := map[string]string{}
	for _, source := range artifacts.Session.SourcePlan {
		target := filepath.ToSlash(strings.TrimSpace(source.TargetPath))
		if prior, ok := selectedTargets[target]; ok && prior != source.SHA256 {
			return Prepared{}, fmt.Errorf("source target %s is selected with different content digests %s and %s; choose one explicit source", target, prior, source.SHA256)
		}
		selectedTargets[target] = source.SHA256
	}

	projectPath := filepath.Join(exampleDir, "project.md")
	intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
	if artifacts.Incomplete {
		intentPath = filepath.Join(exampleDir, "workflows", "intent.draft.hcl")
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
		files = append(files, GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "browser-sources.json"), Content: browserMetadata, AllowOverwrite: true, Action: "write", Reason: "record safe browser origin, action, digest, lifecycle, optional value-free verification, session-posture, and approval evidence"})
	} else if !artifacts.Incomplete {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "browser-sources.json"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove stale browser source review metadata"})
	}
	authenticationMetadata, hasAuthenticationSources, err := BrowserAuthenticationMetadataJSON(artifacts.Session)
	if err != nil {
		return Prepared{}, err
	}
	if hasAuthenticationSources {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "browser-authentication.json"), Content: authenticationMetadata, AllowOverwrite: true, Action: "write", Reason: "record safe browser authentication source, flow, credential-slot, session-binding, and approval evidence"})
	} else if !artifacts.Incomplete {
		files = append(files, GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "browser-authentication.json"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove stale browser authentication review metadata"})
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
			GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "session.yaml"), Content: string(sessionData) + "\n", AllowOverwrite: true, Action: "write", Reason: "persist resumable incomplete authoring state"},
			GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "readiness.json"), Content: string(readinessData) + "\n", AllowOverwrite: true, Action: "write", Reason: "persist incomplete authoring readiness and deferrals"},
		)
	} else {
		files = append(files,
			GeneratedFile{Path: filepath.Join(exampleDir, "workflows", "intent.draft.hcl"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "promote the completed draft"},
			GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "session.yaml"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove obsolete resumable draft state"},
			GeneratedFile{Path: filepath.Join(exampleDir, ".icot", "readiness.json"), Remove: true, AllowOverwrite: true, Action: "remove_if_present", Reason: "remove obsolete generated draft readiness"},
		)
	}
	for _, source := range artifacts.Session.SourcePlan {
		target, err := SafeExampleTarget(exampleDir, source.TargetPath)
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
		if existing, err := os.ReadFile(target); err == nil {
			if fmt.Sprintf("%x", sha256.Sum256(existing)) == digest {
				continue
			}
			if !force {
				return Prepared{}, fmt.Errorf("source target %s contains different content; pass --force to replace it", target)
			}
		} else if !os.IsNotExist(err) {
			return Prepared{}, err
		}
		files = append(files, GeneratedFile{Path: target, Content: string(data), Action: "copy", Reason: source.Kind + " source " + source.ID + " with SHA-256 " + source.SHA256})
	}
	return Prepared{Artifacts: artifacts, Files: files}, nil
}

// Commit atomically applies a prepared authoring transaction.
func Commit(prepared Prepared, force bool) (Result, error) {
	if err := WriteFilesAtomic(prepared.Files, force); err != nil {
		return Result{}, err
	}
	result := Result{}
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
	for _, file := range files {
		if err := validateGeneratedFile(file); err != nil {
			return err
		}
		if !file.Remove {
			if err := scaffoldDirs(exampleDirForGenerated(file.Path)); err != nil {
				return err
			}
		}
		if _, err := os.Stat(file.Path); err == nil && !force && !file.AllowOverwrite {
			return fmt.Errorf("%s already exists; pass --force to overwrite it", file.Path)
		} else if err != nil && !os.IsNotExist(err) {
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
		closeErr := tmp.Close()
		if writeErr != nil {
			cleanupTemps(tmpPaths)
			return writeErr
		}
		if closeErr != nil {
			cleanupTemps(tmpPaths)
			return closeErr
		}
	}
	backups := map[string]fileBackup{}
	for _, file := range files {
		if _, err := os.Stat(file.Path); err == nil {
			backupPath, err := backupFilePath(file.Path)
			if err != nil {
				cleanupTemps(tmpPaths)
				return err
			}
			backups[file.Path] = fileBackup{backupPath: backupPath, existed: true}
		} else if err != nil && !os.IsNotExist(err) {
			cleanupTemps(tmpPaths)
			return err
		}
	}
	var renamed []string
	for _, file := range files {
		var err error
		if file.Remove {
			err = os.Remove(file.Path)
			if os.IsNotExist(err) {
				err = nil
			}
		} else {
			err = os.Rename(tmpPaths[file.Path], file.Path)
		}
		if err != nil {
			restoreBackups(backups, renamed)
			cleanupTemps(tmpPaths)
			return err
		}
		renamed = append(renamed, file.Path)
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
	base, err := filepath.Abs(exampleDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(base, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source target %q escapes example directory", relative)
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

func restoreBackups(backups map[string]fileBackup, renamed []string) {
	for i := len(renamed) - 1; i >= 0; i-- {
		path := renamed[i]
		backup := backups[path]
		if backup.existed {
			_ = copyFile(backup.backupPath, path)
		} else {
			_ = os.Remove(path)
		}
	}
}

func backupFilePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	base := fmt.Sprintf("%s.bak.%d", path, time.Now().UnixNano())
	for i := 0; ; i++ {
		backupPath := base
		if i > 0 {
			backupPath = fmt.Sprintf("%s.%d", base, i)
		}
		file, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return "", writeErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return backupPath, nil
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func exampleDirForGenerated(path string) string {
	dir := filepath.Dir(path)
	if filepath.Base(dir) == "workflows" || filepath.Base(dir) == "openapi" || filepath.Base(dir) == "expected" {
		return filepath.Dir(dir)
	}
	return dir
}

func scaffoldDirs(exampleDir string) error {
	for _, dir := range []string{
		exampleDir,
		filepath.Join(exampleDir, "openapi"),
		filepath.Join(exampleDir, "workflows"),
		filepath.Join(exampleDir, "expected"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
