// Package engine provides the driver-agnostic iCoT authoring lifecycle used by
// interactive frontends. It performs no prompting and exposes no reader or
// writer handles.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/apitools"
	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// Config identifies one local authoring workspace and its reviewed discovery
// inputs. Seed, SessionPath, and FromExample are mutually exclusive.
type Config struct {
	ExampleDir string

	Seed         *elicitor.Session
	SessionPath  string
	FromExample  string
	LoadExisting bool

	LocalSources         []apitools.LocalSource
	BrowserSources       []elicitor.BrowserSourceInput
	BrowserVerifications []string
	BrowserRegistries    []string
	SourceRoots          []string
	NetworkPolicy        string

	Now func() time.Time
}

// SourceCandidates contains the bounded source-discovery evidence attached to
// a snapshot.
type SourceCandidates struct {
	Local           apitools.LocalSourceDiscoveryReport     `json:"local"`
	Browser         browsertools.LocalSourceDiscoveryReport `json:"browser"`
	BrowserRegistry []elicitor.BrowserRegistryCandidate     `json:"browser_registry,omitempty"`
	RegistryBlocks  []elicitor.BrowserRegistryBlocker       `json:"browser_registry_blockers,omitempty"`
	Remote          []elicitor.RemoteSourceCandidate        `json:"remote,omitempty"`
	RemoteBlocker   *elicitor.RemoteSourceBlocker           `json:"remote_blocker,omitempty"`
}

// Preview is an in-memory rendering of the deliverables that would be written
// after explicit approval.
type Preview struct {
	ProjectMD   string `json:"project_md"`
	IntentHCL   string `json:"intent_hcl"`
	Incomplete  bool   `json:"incomplete"`
	ProjectPath string `json:"project_path"`
	IntentPath  string `json:"intent_path"`
}

// Snapshot is the JSON-marshalable current authoring state intended for
// frontend drivers.
type Snapshot struct {
	Boundary           elicitor.WorkflowBoundary        `json:"boundary"`
	CandidateWorkflows []elicitor.CandidateWorkflow     `json:"candidate_workflows,omitempty"`
	Evidence           []publicinterview.Evidence       `json:"evidence,omitempty"`
	Frontier           []elicitor.QuestionPlan          `json:"frontier"`
	Readiness          []elicitor.ReadinessIssue        `json:"readiness"`
	TopIssue           *elicitor.ReadinessIssue         `json:"top_issue,omitempty"`
	Ready              bool                             `json:"ready"`
	ApprovalRequired   bool                             `json:"approval_required"`
	SelectedSources    []elicitor.SourceMaterialization `json:"selected_sources,omitempty"`
	SourceCandidates   SourceCandidates                 `json:"source_candidates"`
	ProposedActions    []elicitor.FileAction            `json:"proposed_file_actions"`
	Preview            *Preview                         `json:"preview,omitempty"`
}

// Approval is explicit human authority to write the currently rendered
// proposal. Zero values never approve a write.
type Approval struct {
	HumanApproved     bool `json:"human_approved"`
	AllowOverwrite    bool `json:"allow_overwrite,omitempty"`
	ApproveIncomplete bool `json:"approve_incomplete,omitempty"`
}

// WriteResult reports the completed atomic transaction.
type WriteResult struct {
	Written    []string `json:"written"`
	Removed    []string `json:"removed,omitempty"`
	Incomplete bool     `json:"incomplete"`
	Preview    Preview  `json:"preview"`
}

// Engine owns one mutable local authoring session.
type Engine struct {
	mu sync.Mutex

	config          Config
	session         elicitor.Session
	discovery       elicitor.LocalSourceDiscovery
	registry        elicitor.BrowserRegistryDiscovery
	remote          elicitor.RemoteSourceLookupReport
	discoveryIssues []elicitor.ReadinessIssue
}

type refreshedEngineState struct {
	session         elicitor.Session
	discovery       elicitor.LocalSourceDiscovery
	registry        elicitor.BrowserRegistryDiscovery
	remote          elicitor.RemoteSourceLookupReport
	discoveryIssues []elicitor.ReadinessIssue
}

// Open loads the configured seed, resumable draft, or explicit session,
// performs bounded source discovery, and returns the current round state.
func Open(ctx context.Context, config Config) (*Engine, Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, Snapshot{}, err
	}
	config.ExampleDir = strings.TrimSpace(config.ExampleDir)
	if config.ExampleDir == "" {
		return nil, Snapshot{}, errors.New("example directory is required")
	}
	if err := validateSeedConfig(config); err != nil {
		return nil, Snapshot{}, err
	}
	policy, err := normalizeNetworkPolicy(config.NetworkPolicy)
	if err != nil {
		return nil, Snapshot{}, err
	}
	config.NetworkPolicy = policy
	session, err := loadSession(config)
	if err != nil {
		return nil, Snapshot{}, err
	}
	session.Normalize()
	engine := &Engine{config: config, session: session}
	if err := engine.refreshLocked(ctx); err != nil {
		return nil, Snapshot{}, err
	}
	snapshot, err := engine.snapshotLocked(ctx)
	if err != nil {
		return nil, Snapshot{}, err
	}
	return engine, snapshot, nil
}

// Snapshot returns the current dependency frontier, readiness, sources, file
// actions, and render preview without writing deliverables.
func (e *Engine) Snapshot(ctx context.Context) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, errors.New("engine is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked(ctx)
}

// ApplyRound applies exactly one complete dependency-ready frontier round,
// refreshes source selection, and autosaves resumable state.
func (e *Engine) ApplyRound(ctx context.Context, answers []authoring.RoundAnswer) (Snapshot, error) {
	if e == nil {
		return Snapshot{}, errors.New("engine is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	issues := e.currentReadinessLocked()
	frontier, err := elicitor.PlanFrontier(&e.session, e.discovery.Docs, issues)
	if err != nil {
		return Snapshot{}, err
	}
	if len(frontier) == 0 {
		return Snapshot{}, errors.New("current authoring state has no dependency-ready frontier")
	}
	canonicalAnswers, err := canonicalizeCompleteRound(frontier, answers)
	if err != nil {
		return Snapshot{}, err
	}
	nextSession, err := cloneSession(e.session)
	if err != nil {
		return Snapshot{}, err
	}
	if err := elicitor.ApplyFrontierRound(&nextSession, canonicalAnswers, e.discovery.Docs); err != nil {
		return Snapshot{}, err
	}
	nextSession.Normalize()
	refreshed, err := e.buildRefreshedStateLocked(ctx, nextSession)
	if err != nil {
		return Snapshot{}, err
	}
	if err := elicitor.SaveDraft(elicitor.DraftPath(e.config.ExampleDir), refreshed.session); err != nil {
		return Snapshot{}, err
	}
	e.applyRefreshedStateLocked(refreshed)
	return e.snapshotLocked(ctx)
}

// Preview renders final or explicitly incomplete authoring artifacts without
// writing final deliverables.
func (e *Engine) Preview(ctx context.Context) (Preview, error) {
	if e == nil {
		return Preview{}, errors.New("engine is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Preview{}, err
	}
	preview, _, err := e.renderLocked()
	return preview, err
}

// ApproveAndWrite revalidates and atomically writes the current proposal only
// after explicit human approval.
func (e *Engine) ApproveAndWrite(ctx context.Context, approval Approval) (WriteResult, error) {
	if e == nil {
		return WriteResult{}, errors.New("engine is nil")
	}
	if !approval.HumanApproved {
		return WriteResult{}, errors.New("explicit human approval is required before writing authoring artifacts")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return WriteResult{}, err
	}
	if err := e.refreshLocked(ctx); err != nil {
		return WriteResult{}, err
	}
	preview, artifacts, err := e.renderLocked()
	if err != nil {
		return WriteResult{}, err
	}
	if artifacts.Incomplete && !approval.ApproveIncomplete {
		return WriteResult{}, errors.New("the proposal is incomplete; explicit incomplete-draft approval is required")
	}
	prepared, err := artifactwriter.Prepare(e.config.ExampleDir, artifacts, approval.AllowOverwrite, e.now())
	if err != nil {
		return WriteResult{}, err
	}
	committed, err := artifactwriter.Commit(prepared, approval.AllowOverwrite)
	if err != nil {
		return WriteResult{}, err
	}
	e.session = prepared.Artifacts.Session
	if !artifacts.Incomplete {
		if err := elicitor.DeleteDraft(elicitor.DraftPath(e.config.ExampleDir)); err != nil {
			return WriteResult{}, err
		}
	}
	return WriteResult{Written: committed.Written, Removed: committed.Removed, Incomplete: artifacts.Incomplete, Preview: preview}, nil
}

func (e *Engine) snapshotLocked(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	issues := e.currentReadinessLocked()
	frontier, err := elicitor.PlanFrontier(&e.session, e.discovery.Docs, issues)
	if err != nil {
		return Snapshot{}, err
	}
	issues = e.currentReadinessLocked()
	var preview *Preview
	actions := artifactwriter.PotentialFileActions(e.config.ExampleDir, e.session, false)
	if rendered, artifacts, renderErr := e.renderLocked(); renderErr == nil {
		preview = &rendered
		prepared, prepareErr := artifactwriter.Prepare(e.config.ExampleDir, artifacts, true, e.now())
		if prepareErr != nil {
			return Snapshot{}, prepareErr
		}
		actions = artifactwriter.ProposedFileActions(prepared)
	}
	top := topIssue(issues)
	snapshot := Snapshot{
		Boundary:           e.session.Boundary,
		CandidateWorkflows: append([]elicitor.CandidateWorkflow(nil), e.session.CandidateWorkflows...),
		Evidence:           append([]publicinterview.Evidence(nil), e.session.Interview.Evidence...),
		Frontier:           append([]elicitor.QuestionPlan(nil), frontier...),
		Readiness:          append([]elicitor.ReadinessIssue(nil), issues...),
		TopIssue:           top,
		Ready:              preview != nil && !preview.Incomplete && !hasBlockingIssue(issues),
		ApprovalRequired:   preview != nil,
		SelectedSources:    append([]elicitor.SourceMaterialization(nil), e.session.SourcePlan...),
		SourceCandidates: SourceCandidates{
			Local: e.discovery.Report, Browser: e.discovery.BrowserReport,
			BrowserRegistry: append([]elicitor.BrowserRegistryCandidate(nil), e.registry.Candidates...),
			RegistryBlocks:  append([]elicitor.BrowserRegistryBlocker(nil), e.registry.Blockers...),
			Remote:          append([]elicitor.RemoteSourceCandidate(nil), e.remote.Candidates...), RemoteBlocker: e.remote.Blocker,
		},
		ProposedActions: actions,
		Preview:         preview,
	}
	return cloneSnapshot(snapshot)
}

func (e *Engine) renderLocked() (Preview, elicitor.Artifacts, error) {
	if hasBlockingIssue(e.discoveryIssues) {
		return Preview{}, elicitor.Artifacts{}, errors.New(e.discoveryIssues[0].Message)
	}
	renderSession, err := cloneSession(e.session)
	if err != nil {
		return Preview{}, elicitor.Artifacts{}, err
	}
	artifacts, err := elicitor.RenderArtifacts(renderSession)
	if err != nil && len(renderSession.Interview.Deferrals) > 0 {
		artifacts, err = elicitor.RenderDraftArtifacts(renderSession)
	}
	if err != nil {
		return Preview{}, elicitor.Artifacts{}, err
	}
	intentName := "intent.hcl"
	if artifacts.Incomplete {
		intentName = "intent.draft.hcl"
	}
	return Preview{
		ProjectMD: artifacts.ProjectMD, IntentHCL: artifacts.IntentHCL, Incomplete: artifacts.Incomplete,
		ProjectPath: filepath.Join(e.config.ExampleDir, "project.md"), IntentPath: filepath.Join(e.config.ExampleDir, "workflows", intentName),
	}, artifacts, nil
}

func (e *Engine) refreshLocked(ctx context.Context) error {
	refreshed, err := e.buildRefreshedStateLocked(ctx, e.session)
	if err != nil {
		return err
	}
	e.applyRefreshedStateLocked(refreshed)
	return nil
}

func (e *Engine) buildRefreshedStateLocked(ctx context.Context, baseSession elicitor.Session) (refreshedEngineState, error) {
	if err := ctx.Err(); err != nil {
		return refreshedEngineState{}, err
	}
	session, err := cloneSession(baseSession)
	if err != nil {
		return refreshedEngineState{}, err
	}
	at := e.now()
	roots := append([]string(nil), e.config.SourceRoots...)
	if seedDir := strings.TrimSpace(e.config.FromExample); seedDir != "" && filepath.Clean(seedDir) != filepath.Clean(e.config.ExampleDir) {
		roots = appendSeedSourceRoots(roots, seedDir)
	}
	discovery, err := elicitor.DiscoverAuthoringSourcesWithBrowser(ctx, e.config.ExampleDir, projectwizard.Render(session.Project), e.config.LocalSources, roots, e.config.BrowserSources, at)
	if err != nil {
		return refreshedEngineState{}, err
	}
	registry := elicitor.BrowserRegistryDiscovery{Candidates: []elicitor.BrowserRegistryCandidate{}, Blockers: []elicitor.BrowserRegistryBlocker{}}
	if len(e.config.BrowserRegistries) > 0 && (len(discovery.Docs) == 0 || session.BrowserRoute == "browser" || sessionUsesBrowserRegistry(session)) {
		approved := e.config.NetworkPolicy == "allow" || strings.EqualFold(session.Interview.Metadata["browser_registry_lookup_decision"], "allow")
		registry, err = elicitor.DiscoverBrowserRegistrySources(ctx, e.config.BrowserRegistries, firstNonEmpty(session.Boundary.Outcome, session.Project.Goal), e.config.NetworkPolicy, approved, at)
		if err != nil {
			return refreshedEngineState{}, err
		}
		discovery = elicitor.MergeBrowserRegistrySources(discovery, registry)
	}
	if err := requireFreshRegistrySources(session.SourcePlan, registry.Plans); err != nil {
		return refreshedEngineState{}, err
	}
	session.SourcePlan = elicitor.SyncSelectedSourcePlansWithBrowser(session, discovery.Plans, e.config.LocalSources, e.config.BrowserSources)
	session.SourcePlan, err = elicitor.AttachBrowserVerifications(session.SourcePlan, e.config.BrowserVerifications, at)
	if err != nil {
		return refreshedEngineState{}, err
	}
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata["network_policy"] = e.config.NetworkPolicy
	if len(e.config.BrowserRegistries) > 0 {
		session.Interview.Metadata["browser_registry_configured"] = "true"
	}
	session.Normalize()

	remote := elicitor.RemoteSourceLookupReport{}
	if len(discovery.Docs) == 0 && session.Intent.RequiresOpenAPI() {
		approved := e.config.NetworkPolicy == "allow" || strings.EqualFold(session.Interview.Metadata["remote_lookup_decision"], "allow")
		remote, err = elicitor.DiscoverRemoteSourceHints(ctx, firstNonEmpty(session.Boundary.Outcome, session.Project.Goal), elicitor.RemoteSourceLookupOptions{Policy: e.config.NetworkPolicy, Approved: approved})
		if err != nil {
			return refreshedEngineState{}, err
		}
	}
	return refreshedEngineState{session: session, discovery: discovery, registry: registry, remote: remote, discoveryIssues: discoveryReadinessIssues(discovery)}, nil
}

func (e *Engine) applyRefreshedStateLocked(refreshed refreshedEngineState) {
	e.session = refreshed.session
	e.discovery = refreshed.discovery
	e.registry = refreshed.registry
	e.remote = refreshed.remote
	e.discoveryIssues = refreshed.discoveryIssues
}

func (e *Engine) currentReadinessLocked() []elicitor.ReadinessIssue {
	issues := append([]elicitor.ReadinessIssue(nil), e.discoveryIssues...)
	issues = append(issues, elicitor.CheckReadiness(e.session, e.discovery.Docs)...)
	return issues
}

func (e *Engine) now() time.Time {
	if e.config.Now != nil {
		return e.config.Now().UTC()
	}
	return time.Now().UTC()
}

func canonicalizeCompleteRound(frontier []elicitor.QuestionPlan, answers []authoring.RoundAnswer) ([]authoring.RoundAnswer, error) {
	frontierQuestions := make(map[string]elicitor.QuestionPlan, len(frontier))
	for _, question := range frontier {
		frontierQuestions[question.ID] = question
	}
	seen := make(map[string]bool, len(answers))
	canonical := make([]authoring.RoundAnswer, 0, len(answers))
	for _, answer := range answers {
		id := strings.TrimSpace(answer.QuestionID)
		question, ok := frontierQuestions[id]
		if !ok {
			return nil, fmt.Errorf("answer references non-frontier decision %q", answer.QuestionID)
		}
		if seen[id] {
			return nil, fmt.Errorf("answer references duplicate decision %q", answer.QuestionID)
		}
		if len(answer.Slots) > 0 && !equalStrings(answer.Slots, question.Slots) {
			return nil, fmt.Errorf("answer slots for decision %q do not match the current frontier", answer.QuestionID)
		}
		seen[id] = true
		answer.QuestionID = question.ID
		answer.Slots = append([]string(nil), question.Slots...)
		canonical = append(canonical, answer)
	}
	if len(seen) != len(frontierQuestions) {
		var missing []string
		for _, question := range frontier {
			if !seen[question.ID] {
				missing = append(missing, question.ID)
			}
		}
		return nil, fmt.Errorf("round must answer the complete frontier; missing %s", strings.Join(missing, ", "))
	}
	return canonical, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func requireFreshRegistrySources(selected, discovered []elicitor.SourceMaterialization) error {
	for _, source := range selected {
		if source.Kind != "browser-profile" || strings.TrimSpace(source.Registry) == "" {
			continue
		}
		matched := false
		for _, candidate := range discovered {
			if candidate.Kind == source.Kind && candidate.Registry == source.Registry &&
				candidate.RegistryCoordinate == source.RegistryCoordinate && candidate.TargetPath == source.TargetPath &&
				strings.EqualFold(candidate.SHA256, source.SHA256) && strings.EqualFold(candidate.SourceSHA256, source.SourceSHA256) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("selected browser registry profile %s could not be freshly revalidated; use an available configured registry or provide the profile explicitly", firstNonEmpty(source.RegistryCoordinate, source.ID))
		}
	}
	return nil
}

func discoveryReadinessIssues(discovery elicitor.LocalSourceDiscovery) []elicitor.ReadinessIssue {
	var issues []elicitor.ReadinessIssue
	if discovery.Report.Truncated || len(discovery.Report.Ambiguous) > 0 {
		issues = append(issues, elicitor.ReadinessIssue{Code: "source_discovery_blocked", Severity: "blocking", Slot: "source.selection", Message: "Local source discovery is incomplete; narrow roots or declare ambiguous documents with --api-source KIND:ID=PATH."})
	}
	var inactive int
	for _, candidate := range discovery.BrowserReport.Candidates {
		if candidate.Status != "active" {
			inactive++
		}
	}
	if len(discovery.BrowserReport.Truncated) > 0 || len(discovery.BrowserReport.Ambiguous) > 0 || inactive > 0 {
		issues = append(issues, elicitor.ReadinessIssue{Code: "browser_source_discovery_blocked", Severity: "blocking", Slot: "source.browser", Message: "Browser source discovery is incomplete; narrow roots or declare a verified profile with --browser-profile ID=PATH."})
	}
	return issues
}

func topIssue(issues []elicitor.ReadinessIssue) *elicitor.ReadinessIssue {
	for i := range issues {
		if strings.EqualFold(issues[i].Severity, "blocking") {
			issue := issues[i]
			return &issue
		}
	}
	if len(issues) == 0 {
		return nil
	}
	issue := issues[0]
	return &issue
}

func hasBlockingIssue(issues []elicitor.ReadinessIssue) bool {
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, "blocking") {
			return true
		}
	}
	return false
}

func loadSession(config Config) (elicitor.Session, error) {
	if config.Seed != nil {
		seed := *config.Seed
		seed.Normalize()
		return cloneSession(seed)
	}
	if path := strings.TrimSpace(config.SessionPath); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return elicitor.Session{}, err
		}
		session, err := elicitor.DecodeSession(data, filepath.Ext(path))
		if err != nil {
			return elicitor.Session{}, fmt.Errorf("parse v2 session: %w", err)
		}
		if !elicitor.LooksLikeSession(session) {
			return elicitor.Session{}, errors.New("v2 session has no workflow state")
		}
		return session, nil
	}
	if strings.TrimSpace(config.FromExample) == "" && !config.LoadExisting {
		if draft, ok, err := elicitor.LoadDraft(elicitor.DraftPath(config.ExampleDir)); err != nil {
			return elicitor.Session{}, err
		} else if ok {
			return draft, nil
		}
		return elicitor.Session{}, nil
	}
	seedDir := strings.TrimSpace(config.FromExample)
	if seedDir == "" {
		seedDir = config.ExampleDir
	}
	return loadExampleSession(seedDir, config.FromExample != "")
}

func loadExampleSession(seedDir string, required bool) (elicitor.Session, error) {
	var project projectwizard.Answers
	projectData, projectErr := os.ReadFile(filepath.Join(seedDir, "project.md"))
	if projectErr == nil {
		loaded, err := projectwizard.LoadAnswersFromMarkdown(string(projectData))
		if err != nil {
			return elicitor.Session{}, err
		}
		project = loaded
	} else if !os.IsNotExist(projectErr) || required {
		return elicitor.Session{}, projectErr
	}
	intent, intentErr := parseSeedIntent(seedDir)
	if intentErr == nil {
		return elicitor.SessionFromIntent(intent, project), nil
	}
	if projectErr == nil {
		return elicitor.NewSessionFromAnswers(project), nil
	}
	if required {
		return elicitor.Session{}, intentErr
	}
	return elicitor.Session{}, nil
}

func parseSeedIntent(seedDir string) (*rollout.Intent, error) {
	intent, err := rollout.ParseIntentFile(filepath.Join(seedDir, "workflows", "intent.hcl"))
	if err == nil {
		return intent, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	reference, referenceErr := rollout.ParseIntentFile(filepath.Join(seedDir, "reference", "intent.hcl"))
	if referenceErr == nil {
		return reference, nil
	}
	if errors.Is(referenceErr, os.ErrNotExist) {
		return nil, err
	}
	return nil, referenceErr
}

func cloneSession(session elicitor.Session) (elicitor.Session, error) {
	for index, event := range session.DraftEvents {
		if _, err := json.Marshal(event); err != nil {
			return elicitor.Session{}, fmt.Errorf("draft event %d is not JSON-marshalable: %w", index, err)
		}
	}
	materialized := map[string][]byte{}
	for _, source := range session.SourcePlan {
		if len(source.MaterializedContent) > 0 {
			materialized[source.TargetPath+"\x00"+source.SHA256] = append([]byte(nil), source.MaterializedContent...)
		}
	}
	data, err := json.Marshal(session)
	if err != nil {
		return elicitor.Session{}, err
	}
	cloned, err := elicitor.DecodeSession(data, ".json")
	if err != nil {
		return elicitor.Session{}, err
	}
	for index := range cloned.SourcePlan {
		key := cloned.SourcePlan[index].TargetPath + "\x00" + cloned.SourcePlan[index].SHA256
		cloned.SourcePlan[index].MaterializedContent = append([]byte(nil), materialized[key]...)
	}
	return cloned, nil
}

func cloneSnapshot(snapshot Snapshot) (Snapshot, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal engine snapshot: %w", err)
	}
	var cloned Snapshot
	if err := json.Unmarshal(data, &cloned); err != nil {
		return Snapshot{}, fmt.Errorf("clone engine snapshot: %w", err)
	}
	return cloned, nil
}

func validateSeedConfig(config Config) error {
	count := 0
	if config.Seed != nil {
		count++
	}
	if strings.TrimSpace(config.SessionPath) != "" {
		count++
	}
	if strings.TrimSpace(config.FromExample) != "" || config.LoadExisting {
		count++
	}
	if count > 1 {
		return errors.New("seed, session path, from-example, and load-existing modes are mutually exclusive")
	}
	return nil
}

func normalizeNetworkPolicy(policy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "", "ask":
		return "ask", nil
	case "never":
		return "never", nil
	case "allow":
		return "allow", nil
	default:
		return "", errors.New("network policy must be never, ask, or allow")
	}
}

func appendSeedSourceRoots(roots []string, seedDir string) []string {
	for _, dir := range []string{"openapi", "google-discovery", "discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata", "browser-profiles", "browser-authentication", "capability-bundles"} {
		path := filepath.Join(seedDir, dir)
		if info, err := os.Lstat(path); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			roots = append(roots, path)
		}
	}
	return roots
}

func sessionUsesBrowserRegistry(session elicitor.Session) bool {
	for _, source := range session.SourcePlan {
		if source.Kind == "browser-profile" && strings.TrimSpace(source.Registry) != "" {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
