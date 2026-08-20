package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/browsertools/bundle"
	bevidence "github.com/OpenUdon/browsertools/evidence"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registry"
	"github.com/OpenUdon/browsertools/review"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestOpenEmptySeededAndResumed(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "empty")
		_, snapshot, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Frontier) == 0 || snapshot.Boundary.Outcome != "" {
			t.Fatalf("empty snapshot = %#v", snapshot)
		}
		if _, err := json.Marshal(snapshot); err != nil {
			t.Fatalf("snapshot is not JSON-marshalable: %v", err)
		}
		assertNoDeliverables(t, example)
	})

	t.Run("seeded", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "seeded")
		_, snapshot, err := Open(context.Background(), Config{
			ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never",
		})
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Preview == nil || snapshot.Preview.Incomplete || !strings.Contains(snapshot.Preview.IntentHCL, `step "render_report"`) {
			t.Fatalf("seeded preview = %#v", snapshot.Preview)
		}
		if !snapshot.ApprovalRequired || !snapshot.Ready {
			t.Fatalf("seeded readiness = ready %t approval %t issues %#v", snapshot.Ready, snapshot.ApprovalRequired, snapshot.Readiness)
		}
		assertNoDeliverables(t, example)
	})

	t.Run("resumed", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "resumed")
		session := elicitor.Session{
			Boundary: elicitor.WorkflowBoundary{Outcome: "resume this exact outcome"},
			Intent:   rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "resume", Description: "resume this exact outcome"}},
		}
		if err := elicitor.SaveDraft(elicitor.DraftPath(example), session); err != nil {
			t.Fatal(err)
		}
		_, snapshot, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Boundary.Outcome != session.Boundary.Outcome {
			t.Fatalf("resumed boundary = %#v", snapshot.Boundary)
		}
	})
}

func TestSnapshotDoesNotWriteDeliverables(t *testing.T) {
	example := filepath.Join(t.TempDir(), "preview-only")
	engine, opened, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Preview == nil || len(opened.ProposedActions) == 0 {
		t.Fatalf("open snapshot lacks preview/actions: %#v", opened)
	}
	for _, relative := range []string{".icot/browser-authentication.json", ".icot/browser-sources.json", ".icot/readiness.json", ".icot/session.yaml", "workflows/intent.draft.hcl"} {
		assertFileAction(t, opened.ProposedActions, "remove_if_present", filepath.Join(example, filepath.FromSlash(relative)))
	}
	if _, err := engine.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Preview(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertNoDeliverables(t, example)
}

func TestSnapshotReportsAcceptedBaselineWriteConflicts(t *testing.T) {
	example := filepath.Join(t.TempDir(), "conflicted-preview")
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(example, "project.md")
	if err := os.WriteFile(project, []byte("existing project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, snapshot, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.WriteConflicts) != 1 {
		t.Fatalf("write conflicts = %#v", snapshot.WriteConflicts)
	}
	conflict := snapshot.WriteConflicts[0]
	if conflict.Code != "overwrite_required" || conflict.Action != "write" || conflict.Path != project {
		t.Fatalf("write conflict = %#v", conflict)
	}
	data, err := os.ReadFile(project)
	if err != nil || string(data) != "existing project\n" {
		t.Fatalf("conflict inspection changed project = %q, %v", data, err)
	}
}

func TestApplyRoundRejectsInvalidAnswerSets(t *testing.T) {
	newEmpty := func(t *testing.T) (*Engine, Snapshot) {
		t.Helper()
		engine, snapshot, err := Open(context.Background(), Config{ExampleDir: filepath.Join(t.TempDir(), "round"), NetworkPolicy: "never"})
		if err != nil {
			t.Fatal(err)
		}
		return engine, snapshot
	}

	t.Run("duplicate", func(t *testing.T) {
		engine, snapshot := newEmpty(t)
		id := snapshot.Frontier[0].ID
		_, err := engine.ApplyRound(context.Background(), []authoring.RoundAnswer{{QuestionID: id, Value: "one"}, {QuestionID: id, Value: "two"}})
		if err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("duplicate error = %v", err)
		}
	})

	t.Run("non-frontier", func(t *testing.T) {
		engine, _ := newEmpty(t)
		_, err := engine.ApplyRound(context.Background(), []authoring.RoundAnswer{{QuestionID: "not.current", Value: "one"}})
		if err == nil || !strings.Contains(err.Error(), "non-frontier") {
			t.Fatalf("non-frontier error = %v", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		engine, snapshot := newEmpty(t)
		answers := make([]authoring.RoundAnswer, 0, len(snapshot.Frontier))
		for index, question := range snapshot.Frontier {
			value := "reviewed answer"
			if index == 0 {
				value = ""
			}
			answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Value: value})
		}
		_, err := engine.ApplyRound(context.Background(), answers)
		if err == nil || !strings.Contains(err.Error(), "requires an answer") {
			t.Fatalf("invalid error = %v", err)
		}
	})

	t.Run("malformed-deferral", func(t *testing.T) {
		seed := missingSourceSession()
		engine, snapshot, err := Open(context.Background(), Config{ExampleDir: filepath.Join(t.TempDir(), "deferral"), Seed: &seed, NetworkPolicy: "never"})
		if err != nil {
			t.Fatal(err)
		}
		answers := make([]authoring.RoundAnswer, 0, len(snapshot.Frontier))
		found := false
		for _, question := range snapshot.Frontier {
			value := firstTestAnswer(question)
			if question.ID == "source.selection" {
				value = "defer:operator | cannot author | missing next action"
				found = true
			}
			answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Slots: question.Slots, Value: value, Source: "user"})
		}
		if !found {
			t.Fatalf("source-selection frontier not found: %#v", snapshot.Frontier)
		}
		_, err = engine.ApplyRound(context.Background(), answers)
		if err == nil || !strings.Contains(err.Error(), "owner | impact | unblock condition | suggested next action") {
			t.Fatalf("malformed deferral error = %v", err)
		}
	})

	t.Run("frontier-slots-are-derived", func(t *testing.T) {
		engine, snapshot := newEmpty(t)
		answers := make([]authoring.RoundAnswer, 0, len(snapshot.Frontier))
		for _, question := range snapshot.Frontier {
			answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Value: firstTestAnswer(question), Source: "user"})
		}
		applied, err := engine.ApplyRound(context.Background(), answers)
		if err != nil {
			t.Fatal(err)
		}
		if applied.Boundary.Outcome == "" {
			t.Fatalf("slot-free answers did not apply to the frontier: %#v", applied.Boundary)
		}
	})

	t.Run("forged-slots", func(t *testing.T) {
		engine, snapshot := newEmpty(t)
		answers := make([]authoring.RoundAnswer, 0, len(snapshot.Frontier))
		for _, question := range snapshot.Frontier {
			slots := append([]string(nil), question.Slots...)
			if question.ID == snapshot.Frontier[0].ID {
				slots = []string{"intent.steps"}
			}
			answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Slots: slots, Value: firstTestAnswer(question), Source: "user"})
		}
		_, err := engine.ApplyRound(context.Background(), answers)
		if err == nil || !strings.Contains(err.Error(), "do not match the current frontier") {
			t.Fatalf("forged-slot error = %v", err)
		}
		unchanged, snapshotErr := engine.Snapshot(context.Background())
		if snapshotErr != nil {
			t.Fatal(snapshotErr)
		}
		if unchanged.Boundary.Outcome != snapshot.Boundary.Outcome {
			t.Fatalf("forged slots mutated engine boundary: before %#v after %#v", snapshot.Boundary, unchanged.Boundary)
		}
	})
}

func TestApplyRoundReplacesInadequateExistingGoal(t *testing.T) {
	seed, err := loadSession(Config{ExampleDir: t.TempDir(), FromExample: runtimeFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	seed.Project.Goal = "Use the local API document."
	seed.Intent.Workflow.Description = "Use the local API document."
	example := filepath.Join(t.TempDir(), "replace-goal")
	eng, snapshot, err := Open(context.Background(), Config{ExampleDir: example, Seed: &seed, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	const revisedGoal = "Render the reviewed runtime capability report."
	answers := answersForSnapshot(snapshot)
	found := false
	for index, question := range snapshot.Frontier {
		if len(question.Slots) == 1 && question.Slots[0] == "workflow.description" {
			answers[index].Value = revisedGoal
			found = true
		}
	}
	if !found {
		t.Fatalf("missing goal question not found: %#v", snapshot.Frontier)
	}
	updated, err := eng.ApplyRound(context.Background(), answers)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Boundary.Outcome != seed.Boundary.Outcome {
		t.Fatalf("goal correction changed boundary outcome to %q", updated.Boundary.Outcome)
	}
	if updated.Preview == nil || !strings.Contains(updated.Preview.ProjectMD, revisedGoal) || !strings.Contains(updated.Preview.IntentHCL, revisedGoal) {
		t.Fatalf("revised goal was not rendered: %#v", updated.Preview)
	}
	for _, question := range updated.Frontier {
		if len(question.Slots) == 1 && question.Slots[0] == "workflow.description" {
			t.Fatalf("goal question remained open after accepted answer: %#v", question)
		}
	}
}

func TestReopenDecisionPersistsExactReplacementFrontier(t *testing.T) {
	example := filepath.Join(t.TempDir(), "reopen")
	eng, opened, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	answers := answersForSnapshot(opened)
	const original = "Render the original reviewed report"
	for index, question := range opened.Frontier {
		if question.ID == "boundary.outcome" {
			answers[index].Value = original
		}
	}
	settled, err := eng.ApplyRound(context.Background(), answers)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRevisableDecision(settled.RevisableDecisions, "boundary.outcome", original) {
		t.Fatalf("settled outcome is not revisable: %#v", settled.RevisableDecisions)
	}

	reopened, err := eng.ReopenDecision(context.Background(), "boundary.outcome")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Boundary.Outcome != "" || reopened.ApprovalRequired || reopened.Ready || !hasFrontierQuestion(reopened.Frontier, "boundary.outcome") {
		t.Fatalf("reopened snapshot = %#v", reopened)
	}
	draft, ok, err := elicitor.LoadDraft(elicitor.DraftPath(example))
	if err != nil || !ok {
		t.Fatalf("load reopened draft = found %t, error %v", ok, err)
	}
	if draft.Boundary.Outcome != "" || !elicitor.HasPendingRevision(draft) {
		t.Fatalf("reopened draft = %#v", draft)
	}
	if _, err := eng.ApproveAndWrite(context.Background(), Approval{HumanApproved: true}); err == nil || !strings.Contains(err.Error(), "replacement round") {
		t.Fatalf("approval while revision pending = %v", err)
	}
	if _, err := eng.ReopenDecision(context.Background(), "boundary.outcome"); err == nil || !strings.Contains(err.Error(), "not currently settled") {
		t.Fatalf("duplicate reopen error = %v", err)
	}

	replacementAnswers := answersForSnapshot(reopened)
	const replacement = "Render the replacement reviewed report"
	for index, question := range reopened.Frontier {
		if question.ID == "boundary.outcome" {
			replacementAnswers[index].Value = replacement
		}
	}
	replaced, err := eng.ApplyRound(context.Background(), replacementAnswers)
	if err != nil {
		t.Fatal(err)
	}
	if replaced.Boundary.Outcome != replacement || !hasRevisableDecision(replaced.RevisableDecisions, "boundary.outcome", replacement) {
		t.Fatalf("replacement snapshot = %#v", replaced)
	}
}

func TestApplyRoundTransactionAndCancellationFinalization(t *testing.T) {
	example := filepath.Join(t.TempDir(), "transactional-round")
	eng, opened, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	answers := answersForSnapshot(opened)
	originalSave := saveEngineDraft
	defer func() { saveEngineDraft = originalSave }()
	saveEngineDraft = func(string, elicitor.Session) error { return errors.New("injected draft persistence failure") }
	if _, err := eng.ApplyRound(context.Background(), answers); err == nil {
		t.Fatal("expected draft persistence failure")
	} else if class, _ := FailureDetails(err); class != FailureOperational {
		t.Fatalf("persistence failure class = %s, error %v", class, err)
	}
	unchanged, err := eng.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(unchanged, opened) {
		t.Fatalf("failed persistence changed memory\nbefore: %#v\nafter: %#v", opened, unchanged)
	}
	if _, err := os.Stat(elicitor.DraftPath(example)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed persistence changed durable draft: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	saveEngineDraft = func(path string, session elicitor.Session) error {
		cancel()
		return originalSave(path, session)
	}
	applied, err := eng.ApplyRound(ctx, answers)
	if err != nil {
		t.Fatalf("cancellation after persistence started interrupted finalization: %v", err)
	}
	if ctx.Err() == nil || applied.Boundary.Outcome == "" {
		t.Fatalf("round did not finalize after cancellation: ctx=%v snapshot=%#v", ctx.Err(), applied.Boundary)
	}
	loaded, ok, err := elicitor.LoadDraft(elicitor.DraftPath(example))
	if err != nil || !ok || loaded.Boundary.Outcome != applied.Boundary.Outcome {
		t.Fatalf("durable/cache state diverged: loaded=%#v ok=%t err=%v applied=%#v", loaded.Boundary, ok, err, applied.Boundary)
	}
}

func TestApproveAndWriteInstallsSuccessfulCommitWithCleanupWarning(t *testing.T) {
	example := filepath.Join(t.TempDir(), "approval-cleanup-warning")
	eng, _, err := Open(context.Background(), Config{
		ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never",
	})
	if err != nil {
		t.Fatal(err)
	}
	originalCommit := commitPrepared
	defer func() { commitPrepared = originalCommit }()
	commitPrepared = func(prepared artifactwriter.Prepared, force bool, beforeReplace func() error) (artifactwriter.Result, error) {
		result, err := originalCommit(prepared, force, beforeReplace)
		result.CleanupWarnings = append(result.CleanupWarnings, "temporary backup cleanup incomplete")
		return result, err
	}
	approved, err := eng.ApproveAndWrite(context.Background(), Approval{HumanApproved: true})
	if err != nil {
		t.Fatalf("successful commit with cleanup warning failed: %v", err)
	}
	if len(approved.WriteResult.CleanupWarnings) != 1 {
		t.Fatalf("cleanup warnings = %#v", approved.WriteResult.CleanupWarnings)
	}
	cached, err := eng.Snapshot(context.Background())
	if err != nil || !reflect.DeepEqual(cached, approved.Snapshot) {
		t.Fatalf("approved snapshot was not installed: equal=%t err=%v", reflect.DeepEqual(cached, approved.Snapshot), err)
	}
	status, err := eng.WorkspaceStatus(context.Background())
	if err != nil || status.ExternallyModified {
		t.Fatalf("successful commit latched workspace drift: %#v, %v", status, err)
	}
}

func TestWorkspaceFingerprintLatchesExternalChanges(t *testing.T) {
	paths := []string{
		"project.md",
		"workflows/intent.hcl",
		".icot/session.yaml",
		".icot/browser-sources.json",
	}
	for _, relative := range paths {
		t.Run(strings.ReplaceAll(relative, "/", "-"), func(t *testing.T) {
			example := filepath.Join(t.TempDir(), "workspace")
			eng, opened, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(example, filepath.FromSlash(relative))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("external edit\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			status, err := eng.WorkspaceStatus(context.Background())
			if err != nil || !status.ExternallyModified {
				t.Fatalf("workspace status = %#v, %v", status, err)
			}
			cached, err := eng.Snapshot(context.Background())
			if err != nil || !reflect.DeepEqual(cached, opened) {
				t.Fatalf("cached inspection unavailable after drift: equal=%t err=%v", reflect.DeepEqual(cached, opened), err)
			}
			_, err = eng.ApproveAndWrite(context.Background(), Approval{HumanApproved: true, AllowOverwrite: true})
			if class, code := FailureDetails(err); class != FailureConflict || code != "workspace_changed" {
				t.Fatalf("drift approval = %s %s %v", class, code, err)
			}
		})
	}
}

func TestWorkspaceFingerprintCoversMaterializedSourcesAndCompetingEngines(t *testing.T) {
	profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	seed := browserSeedSession()
	example := filepath.Join(t.TempDir(), "browser-source-drift")
	eng, snapshot, err := Open(context.Background(), Config{
		ExampleDir: example, Seed: &seed,
		BrowserSources: []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}}, NetworkPolicy: "never",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.SelectedSources) != 1 {
		t.Fatalf("selected sources = %#v", snapshot.SelectedSources)
	}
	target := filepath.Join(example, filepath.FromSlash(snapshot.SelectedSources[0].TargetPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("external source replacement\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if status, err := eng.WorkspaceStatus(context.Background()); err != nil || !status.ExternallyModified {
		t.Fatalf("materialized source status = %#v, %v", status, err)
	}

	shared := filepath.Join(t.TempDir(), "competing")
	first, _, err := Open(context.Background(), Config{ExampleDir: shared, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := Open(context.Background(), Config{ExampleDir: shared, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.ApproveAndWrite(context.Background(), Approval{HumanApproved: true}); err != nil {
		t.Fatal(err)
	}
	status, err := second.WorkspaceStatus(context.Background())
	if err != nil || !status.ExternallyModified {
		t.Fatalf("second engine status = %#v, %v", status, err)
	}
	projectBefore, err := os.ReadFile(filepath.Join(shared, "project.md"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.ApproveAndWrite(context.Background(), Approval{HumanApproved: true, AllowOverwrite: true}); err == nil {
		t.Fatal("competing engine overwrote accepted workspace")
	}
	projectAfter, err := os.ReadFile(filepath.Join(shared, "project.md"))
	if err != nil || !bytes.Equal(projectBefore, projectAfter) {
		t.Fatalf("competing engine changed project: equal=%t err=%v", bytes.Equal(projectBefore, projectAfter), err)
	}
}

func TestMutationsRecheckWorkspaceAfterPreparation(t *testing.T) {
	t.Run("round", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "round-drift-during-preparation")
		draftPath := elicitor.DraftPath(example)
		armed := false
		mutated := false
		var mutationErr error
		now := func() time.Time {
			if armed && !mutated {
				mutated = true
				mutationErr = os.MkdirAll(filepath.Dir(draftPath), 0o755)
				if mutationErr == nil {
					mutationErr = os.WriteFile(draftPath, []byte("external draft edit\n"), 0o600)
				}
			}
			return time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
		}
		eng, opened, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never", Now: now})
		if err != nil {
			t.Fatal(err)
		}
		armed = true
		_, err = eng.ApplyRound(context.Background(), answersForSnapshot(opened))
		if mutationErr != nil {
			t.Fatal(mutationErr)
		}
		if class, code := FailureDetails(err); class != FailureConflict || code != "workspace_changed" {
			t.Fatalf("round drift = %s %s %v", class, code, err)
		}
		data, readErr := os.ReadFile(draftPath)
		if readErr != nil || string(data) != "external draft edit\n" {
			t.Fatalf("external draft was overwritten: %q, %v", data, readErr)
		}
	})

	t.Run("newly-selected-source-target", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "new-source-target-drift")
		profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
		target := filepath.Join(example, "browser-profiles", "status.json")
		armed := false
		mutated := false
		var eng *Engine
		var mutationErr error
		now := func() time.Time {
			if armed && !mutated {
				mutated = true
				eng.config.BrowserSources = []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}}
				mutationErr = os.MkdirAll(filepath.Dir(target), 0o755)
				if mutationErr == nil {
					mutationErr = os.WriteFile(target, []byte("external source target\n"), 0o600)
				}
			}
			return time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
		}
		var opened Snapshot
		var err error
		eng, opened, err = Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never", Now: now})
		if err != nil {
			t.Fatal(err)
		}
		if len(opened.SelectedSources) != 0 {
			t.Fatalf("source should not be selected initially: %#v", opened.SelectedSources)
		}
		for _, path := range eng.watchedPaths {
			if path == target {
				t.Fatalf("new source target was already watched: %s", target)
			}
		}
		armed = true
		applied, err := eng.ApplyRound(context.Background(), answersForSnapshot(opened))
		if mutationErr != nil {
			t.Fatal(mutationErr)
		}
		if class, code := FailureDetails(err); class != FailureConflict || code != "workspace_changed" {
			t.Fatalf("new target drift = %s %s %v; mutated=%t selected=%#v", class, code, err, mutated, applied.SelectedSources)
		}
		data, readErr := os.ReadFile(target)
		if readErr != nil || string(data) != "external source target\n" {
			t.Fatalf("external source target was overwritten: %q, %v", data, readErr)
		}
		status, statusErr := eng.WorkspaceStatus(context.Background())
		if statusErr != nil || !status.ExternallyModified {
			t.Fatalf("new target drift was not latched: %#v, %v", status, statusErr)
		}
	})

	t.Run("newly-selected-existing-source-target", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "existing-source-target-drift")
		profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
		target := filepath.Join(example, "browser-profiles", "status.json")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("pre-round source target\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		armed := false
		mutated := false
		var eng *Engine
		var mutationErr error
		now := func() time.Time {
			if armed && !mutated {
				mutated = true
				eng.config.BrowserSources = []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}}
				mutationErr = os.WriteFile(target, []byte("external source target\n"), 0o600)
			}
			return time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
		}
		var opened Snapshot
		var err error
		eng, opened, err = Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never", Now: now})
		if err != nil {
			t.Fatal(err)
		}
		if len(opened.SelectedSources) != 0 {
			t.Fatalf("source should not be selected initially: %#v", opened.SelectedSources)
		}
		for _, path := range eng.watchedPaths {
			if path == target {
				t.Fatalf("new source target was already watched: %s", target)
			}
		}
		armed = true
		applied, err := eng.ApplyRound(context.Background(), answersForSnapshot(opened))
		if mutationErr != nil {
			t.Fatal(mutationErr)
		}
		if class, code := FailureDetails(err); class != FailureConflict || code != "workspace_changed" {
			t.Fatalf("existing new-target drift = %s %s %v; selected=%#v", class, code, err, applied.SelectedSources)
		}
		data, readErr := os.ReadFile(target)
		if readErr != nil || string(data) != "external source target\n" {
			t.Fatalf("external source target changed unexpectedly: %q, %v", data, readErr)
		}
	})

	t.Run("approval", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "approval-drift-during-preparation")
		projectPath := filepath.Join(example, "project.md")
		armed := false
		mutated := false
		var mutationErr error
		now := func() time.Time {
			if armed && !mutated {
				mutated = true
				mutationErr = os.MkdirAll(example, 0o755)
				if mutationErr == nil {
					mutationErr = os.WriteFile(projectPath, []byte("external project edit\n"), 0o600)
				}
			}
			return time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
		}
		eng, _, err := Open(context.Background(), Config{
			ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never", Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		armed = true
		_, err = eng.ApproveAndWrite(context.Background(), Approval{HumanApproved: true, AllowOverwrite: true})
		if mutationErr != nil {
			t.Fatal(mutationErr)
		}
		if class, code := FailureDetails(err); class != FailureConflict || code != "workspace_changed" {
			t.Fatalf("approval drift = %s %s %v", class, code, err)
		}
		data, readErr := os.ReadFile(projectPath)
		if readErr != nil || string(data) != "external project edit\n" {
			t.Fatalf("external project was overwritten: %q, %v", data, readErr)
		}
	})
}

func TestWorkspaceInspectionRejectsUnsafeWatchedPath(t *testing.T) {
	example := filepath.Join(t.TempDir(), "unsafe-workspace")
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(example, "workflows")); err != nil {
		t.Fatal(err)
	}
	if _, err := eng.WorkspaceStatus(context.Background()); err == nil {
		t.Fatal("unsafe watched path was accepted")
	} else if class, _ := FailureDetails(err); class != FailureOperational {
		t.Fatalf("unsafe path class = %s, error %v", class, err)
	}
}

func TestMutationObservationIsBoundedToWatchedAndCandidateTargets(t *testing.T) {
	example := filepath.Join(t.TempDir(), "bounded-observation")
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(example, "unrelated"), 0o755); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(example, "unrelated", "large-or-private.bin")
	if err := os.WriteFile(unrelated, []byte("must not be observed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(example, "openapi", "candidate.json")
	if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("candidate baseline\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	eng.discovery.Plans = append(eng.discovery.Plans, elicitor.SourceMaterialization{TargetPath: "openapi/candidate.json"})

	observation, err := eng.observeMutationWorkspaceLocked(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := observation.entries[unrelated]; ok {
		t.Fatalf("unrelated workspace file was observed: %s", unrelated)
	}
	if entry, ok := observation.entries[candidate]; !ok || entry.Type != "regular" {
		t.Fatalf("candidate target observation = %#v, present %t", entry, ok)
	}

	accepted := acceptedFingerprint(append(eng.watchedPaths, candidate), eng.workspaceBaseline, observation)
	if err := os.WriteFile(candidate, []byte("candidate changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if class, code := FailureDetails(compareWorkspace(example, append(eng.watchedPaths, candidate), accepted)); class != FailureConflict || code != "workspace_changed" {
		t.Fatalf("candidate drift = %s %s", class, code)
	}
}

func TestUnobservedNewTargetIsConservativelyMissing(t *testing.T) {
	example := filepath.Join(t.TempDir(), "unobserved-target")
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	observation, err := eng.observeMutationWorkspaceLocked(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(example, "browser-profiles", "newly-produced.json")
	if _, ok := observation.entries[target]; ok {
		t.Fatalf("new target unexpectedly observed: %s", target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("concurrent existing target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := append(append([]string(nil), eng.watchedPaths...), target)
	accepted := acceptedFingerprint(paths, eng.workspaceBaseline, observation)
	if accepted.entries[target].Type != "missing" {
		t.Fatalf("unobserved target baseline = %#v", accepted.entries[target])
	}
	if class, code := FailureDetails(compareWorkspace(example, paths, accepted)); class != FailureConflict || code != "workspace_changed" {
		t.Fatalf("unobserved existing target = %s %s", class, code)
	}
}

type cancelingReader struct {
	cancel context.CancelFunc
	reads  int
}

func (r *cancelingReader) Read(buffer []byte) (int, error) {
	r.reads++
	if r.reads > 1 {
		return 0, io.EOF
	}
	for index := range buffer {
		buffer[index] = byte(index)
	}
	r.cancel()
	return len(buffer), nil
}

func TestStreamingFingerprintHashHonorsCancellationBetweenChunks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelingReader{cancel: cancel}
	digest, count, err := streamSHA256(ctx, reader)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("streaming hash error = %v", err)
	}
	if digest != "" || count == 0 || reader.reads != 1 {
		t.Fatalf("streaming cancellation = digest %q count %d reads %d", digest, count, reader.reads)
	}
}

func TestInvalidSourceSessionPlanIsDomainRejected(t *testing.T) {
	seed, err := loadExampleSession(runtimeFixture(t), true)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("reviewed source\n")
	digest := fmt.Sprintf("%x", sha256.Sum256(content))
	source := func(id, target string) elicitor.SourceMaterialization {
		return elicitor.SourceMaterialization{
			Kind: "openapi", ID: id, SourcePath: id + ".json", TargetPath: target,
			SHA256: digest, Provenance: "reviewed test", MaterializedContent: append([]byte(nil), content...),
		}
	}
	tests := []struct {
		name    string
		sources []elicitor.SourceMaterialization
	}{
		{name: "reserved", sources: []elicitor.SourceMaterialization{source("reserved", ".icot/source.json")}},
		{name: "case-folded", sources: []elicitor.SourceMaterialization{source("one", "openapi/Source.json"), source("two", "OPENAPI/source.JSON")}},
		{name: "ancestor", sources: []elicitor.SourceMaterialization{source("one", "openapi/source"), source("two", "openapi/source/schema.json")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, cloneErr := cloneSession(seed)
			if cloneErr != nil {
				t.Fatal(cloneErr)
			}
			value.SourcePlan = test.sources
			example := filepath.Join(t.TempDir(), "invalid-source-session")
			_, _, openErr := Open(context.Background(), Config{ExampleDir: example, Seed: &value, NetworkPolicy: "never"})
			if class, code := FailureDetails(openErr); class != FailureRejected || code != "engine_rejected" {
				t.Fatalf("source plan rejection = %s %s %v", class, code, openErr)
			}
			assertNoDeliverables(t, example)
		})
	}
}

func TestSnapshotsAreDeepCopies(t *testing.T) {
	profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	seed := browserSeedSession()
	seed.Interview.Evidence = []publicinterview.Evidence{{
		ID: "reviewed-scope", Kind: publicinterview.EvidenceObservedFact, Summary: "reviewed scope", Attributes: map[string]string{"scope": "read-only"},
	}}
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	engine, snapshot, err := Open(context.Background(), Config{
		ExampleDir: filepath.Join(t.TempDir(), "deep-copy"), Seed: &seed,
		BrowserSources: []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}}, NetworkPolicy: "never", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Boundary.SuccessEvidence) == 0 || len(snapshot.SelectedSources) != 1 || len(snapshot.SelectedSources[0].Actions) == 0 {
		t.Fatalf("snapshot lacks nested copy fixtures: %#v", snapshot)
	}
	evidenceIndex := evidenceIndexByID(snapshot.Evidence, "reviewed-scope")
	if evidenceIndex < 0 {
		t.Fatalf("snapshot lacks reviewed evidence: %#v", snapshot.Evidence)
	}
	snapshot.Boundary.SuccessEvidence[0] = "tampered boundary"
	snapshot.Evidence[evidenceIndex].Attributes["scope"] = "tampered evidence"
	snapshot.SelectedSources[0].Actions[0] = "tampered action"

	current, err := engine.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	currentEvidence := evidenceIndexByID(current.Evidence, "reviewed-scope")
	if current.Boundary.SuccessEvidence[0] == "tampered boundary" || currentEvidence < 0 || current.Evidence[currentEvidence].Attributes["scope"] != "read-only" || current.SelectedSources[0].Actions[0] == "tampered action" {
		t.Fatalf("caller mutation escaped snapshot copy: %#v", current)
	}
}

func TestApplyRoundDraftResumePreservesProvenance(t *testing.T) {
	example := filepath.Join(t.TempDir(), "provenance")
	seed := elicitor.Session{
		Annotations:      []elicitor.SourceAnnotation{{Slot: "draft", Source: "llm", PromptVersion: "test", Evidence: "reviewed draft evidence"}},
		Assumptions:      []elicitor.Assumption{{ID: "assumption-1", Slot: "intent", Value: "draft", Reason: "test", Evidence: "operator-visible", Risk: "review", RequiresConfirmation: true}},
		Classifications:  []elicitor.MappingClassification{{Slot: "steps.render.with.summary", Value: "inputs.summary", Source: "user", Confidence: "high", Evidence: "explicit answer"}},
		DecisionEvidence: []elicitor.DecisionEvidence{{Stage: "request_mapping", Slot: "steps.render.with.summary", Value: "inputs.summary", Source: "user", Confidence: "high", Evidence: "explicit answer"}},
		DraftOperations:  []elicitor.OperationDetailRef{{DocumentPath: "openapi/example.yaml", OperationID: "getExample"}},
		DraftEvents:      []elicitor.TranscriptEvent{{Kind: "operation_detail_fulfilled", Data: map[string]any{"operation_ids": []string{"getExample"}}}},
	}
	engine, snapshot, err := Open(context.Background(), Config{ExampleDir: example, Seed: &seed, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	answers := make([]authoring.RoundAnswer, 0, len(snapshot.Frontier))
	for _, question := range snapshot.Frontier {
		value := "preserve provenance"
		if question.ID != "boundary.outcome" {
			value = "Preserve public provenance through the authoring draft."
		}
		answers = append(answers, authoring.RoundAnswer{QuestionID: question.ID, Value: value, Source: "user"})
	}
	if _, err := engine.ApplyRound(context.Background(), answers); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := elicitor.LoadDraft(elicitor.DraftPath(example))
	if err != nil || !ok {
		t.Fatalf("LoadDraft = ok %t, err %v", ok, err)
	}
	if len(loaded.Annotations) != 1 || len(loaded.Assumptions) != 1 || len(loaded.Classifications) != 1 || len(loaded.DecisionEvidence) == 0 {
		t.Fatalf("restored provenance = annotations %#v assumptions %#v classifications %#v decisions %#v", loaded.Annotations, loaded.Assumptions, loaded.Classifications, loaded.DecisionEvidence)
	}
	loadedEvents, _ := json.Marshal(loaded.DraftEvents)
	seedEvents, _ := json.Marshal(seed.DraftEvents)
	if !reflect.DeepEqual(loaded.DraftOperations, seed.DraftOperations) || !bytes.Equal(loadedEvents, seedEvents) {
		t.Fatalf("restored draft provenance = operations %#v events %#v", loaded.DraftOperations, loaded.DraftEvents)
	}
	resumed, resumedSnapshot, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || len(resumedSnapshot.Evidence) < 6 || resumedSnapshot.Boundary.Outcome != "preserve provenance" {
		t.Fatalf("resumed snapshot = %#v", resumedSnapshot)
	}
}

func TestApproveAndWriteRequiresExplicitApproval(t *testing.T) {
	example := filepath.Join(t.TempDir(), "approval")
	engine, _, err := Open(context.Background(), Config{ExampleDir: example, FromExample: runtimeFixture(t), NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.ApproveAndWrite(context.Background(), Approval{}); err == nil || !strings.Contains(err.Error(), "explicit human approval") {
		t.Fatalf("approval error = %v", err)
	}
	assertNoDeliverables(t, example)
}

func TestCLIAndEngineArtifactParity(t *testing.T) {
	fixture := runtimeFixture(t)
	cliDir := filepath.Join(t.TempDir(), "cli")
	engineDir := filepath.Join(t.TempDir(), "engine")
	command := exec.Command("go", "run", "./cmd/icot", "--example", cliDir, "--from-example", fixture, "--no-llm", "--yes")
	command.Dir = repoRoot(t)
	command.Stdin = strings.NewReader("")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run terminal iCoT: %v\n%s", err, output)
	}
	engine, _, err := Open(context.Background(), Config{ExampleDir: engineDir, FromExample: fixture, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.ApproveAndWrite(context.Background(), Approval{HumanApproved: true}); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"project.md", "workflows/intent.hcl"} {
		cliData, err := os.ReadFile(filepath.Join(cliDir, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		engineData, err := os.ReadFile(filepath.Join(engineDir, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(cliData, engineData) {
			t.Fatalf("%s differs\nCLI:\n%s\nengine:\n%s", relative, cliData, engineData)
		}
	}
}

func TestEngineBrowserVerificationMetadataAndRevalidation(t *testing.T) {
	profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	value, err := profile.ParseJSON(profileData)
	if err != nil {
		t.Fatal(err)
	}
	profileDigest, err := browserverify.ProfileDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	report := map[string]any{
		"version": browserverify.LiveCheckVersion, "profileDigest": profileDigest, "checkedAt": now.Add(-time.Minute).Format(time.RFC3339Nano),
		"origin": "https://example.test", "actions": []string{"read_status"}, "ok": true,
		"checks": []map[string]any{{
			"kind": "output", "path": "actions.read_status.outputs.status", "ok": true, "matches": 1,
			"expectedType": "string", "observedType": "string", "message": "declared output source and JSON type matched",
		}, {
			"kind": "locator", "path": "actions.read_status.sequence[1].wait_for", "ok": true, "matches": 1,
			"message": "declared accessibility locator resolved exactly once",
		}},
	}
	reportPath := filepath.Join(t.TempDir(), "live-check.json")
	writeJSON(t, reportPath, report)

	seed := browserSeedSession()
	example := filepath.Join(t.TempDir(), "browser")
	engine, snapshot, err := Open(context.Background(), Config{
		ExampleDir: example, Seed: &seed, BrowserSources: []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}},
		BrowserVerifications: []string{reportPath}, NetworkPolicy: "never", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Preview == nil || len(snapshot.SelectedSources) != 1 || len(snapshot.SelectedSources[0].BrowserVerifications) != 1 {
		t.Fatalf("browser snapshot = %#v", snapshot)
	}
	if _, err := engine.ApproveAndWrite(context.Background(), Approval{HumanApproved: true}); err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile(filepath.Join(example, ".icot", "browser-sources.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"version": "openudon.browser-source-review.v1"`, `"report_version": "browsertools.live-check.v1"`, `"read_status"`} {
		if !strings.Contains(string(metadata), expected) {
			t.Fatalf("browser metadata missing %q:\n%s", expected, metadata)
		}
	}
	if strings.Contains(string(metadata), reportPath) {
		t.Fatalf("browser metadata leaked report path:\n%s", metadata)
	}

	tamperDir := filepath.Join(t.TempDir(), "browser-tamper")
	tamperEngine, _, err := Open(context.Background(), Config{
		ExampleDir: tamperDir, Seed: &seed, BrowserSources: []elicitor.BrowserSourceInput{{ID: "status", Path: profilePath}},
		BrowserVerifications: []string{reportPath}, NetworkPolicy: "never", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	report["checkedAt"] = now.Format(time.RFC3339Nano)
	writeJSON(t, reportPath, report)
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := tamperEngine.ApproveAndWrite(context.Background(), Approval{HumanApproved: true}); err == nil || !strings.Contains(err.Error(), "changed after review") {
			t.Fatalf("tampered verification attempt %d error = %v", attempt, err)
		}
		if len(tamperEngine.session.SourcePlan) != 1 || len(tamperEngine.session.SourcePlan[0].BrowserVerifications) != 1 {
			t.Fatalf("tampered verification attempt %d mutated retained review state: %#v", attempt, tamperEngine.session.SourcePlan)
		}
	}
	assertNoDeliverables(t, tamperDir)
}

func TestApprovalResultRetainsExactSnapshotAfterSourceInputChanges(t *testing.T) {
	original := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	data, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}
	seed := browserSeedSession()
	example := filepath.Join(t.TempDir(), "approved-snapshot")
	eng, _, err := Open(context.Background(), Config{
		ExampleDir: example, Seed: &seed,
		BrowserSources: []elicitor.BrowserSourceInput{{ID: "status", Path: input}}, NetworkPolicy: "never",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := eng.ApproveAndWrite(context.Background(), Approval{HumanApproved: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Snapshot.Preview == nil || !result.Snapshot.Ready || len(result.Snapshot.SelectedSources) != 1 || len(result.WriteResult.Written) == 0 {
		t.Fatalf("approval result = %#v", result)
	}
	if err := os.WriteFile(input, []byte("changed immediately after commit\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	inspected, err := eng.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inspected, result.Snapshot) {
		t.Fatalf("approved cached snapshot changed with source input\nresult=%#v\ninspection=%#v", result.Snapshot, inspected)
	}
}

func TestApproveAndWriteRequiresFreshRegistrySource(t *testing.T) {
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	registryRoot := filepath.Join(t.TempDir(), "registry")
	if _, err := registry.PublishLocal(context.Background(), registry.PublishOptions{Root: registryRoot, Bundle: browserRegistryBundle(t, now), At: now}); err != nil {
		t.Fatal(err)
	}
	example := filepath.Join(t.TempDir(), "registry-source")
	seed := browserSeedSession()
	seed.Boundary.Outcome = "status"
	engine, snapshot, err := Open(context.Background(), Config{
		ExampleDir: example, Seed: &seed, BrowserRegistries: []string{registryRoot}, NetworkPolicy: "never", Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.SelectedSources) != 1 || snapshot.SelectedSources[0].RegistryCoordinate != "status@1.0.0" {
		t.Fatalf("registry source was not selected: selected=%#v candidates=%#v", snapshot.SelectedSources, snapshot.SourceCandidates)
	}
	if err := os.Rename(registryRoot, registryRoot+".unavailable"); err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := engine.ApproveAndWrite(context.Background(), Approval{HumanApproved: true}); err == nil || !strings.Contains(err.Error(), "could not be freshly revalidated") {
			t.Fatalf("missing registry attempt %d error = %v", attempt, err)
		}
		if len(engine.session.SourcePlan) != 1 || engine.session.SourcePlan[0].RegistryCoordinate != "status@1.0.0" {
			t.Fatalf("missing registry attempt %d mutated retained source: %#v", attempt, engine.session.SourcePlan)
		}
	}
	assertNoDeliverables(t, example)
}

func missingSourceSession() elicitor.Session {
	return elicitor.Session{
		Boundary: elicitor.WorkflowBoundary{Outcome: "Fetch one item", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"item output is returned"}, Confirmed: true},
		Intent: rollout.Intent{
			Source: "openapi/missing.yaml", Workflow: &rollout.WorkflowMeta{Name: "fetch_item", Description: "Fetch one item"},
			Steps:   []*rollout.Step{{Name: "fetch", Type: "http", Source: "openapi/missing.yaml", Operation: "getItem"}},
			Outputs: []*rollout.Output{{Name: "item", From: "fetch.received_body"}},
		},
		Fallback: "stop cleanly", FallbackSet: true, SideEffectScope: "read-only", Safety: "read-only", SafetySet: true, CredentialsSet: true,
	}
}

func browserSeedSession() elicitor.Session {
	return elicitor.Session{
		Boundary: elicitor.WorkflowBoundary{Outcome: "Read browser status", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"status output is returned"}, Confirmed: true},
		Intent: rollout.Intent{
			Source: "browser-profiles/status.json", Workflow: &rollout.WorkflowMeta{Name: "browser_status", Description: "Read browser status"},
			Inputs:  []*rollout.Input{{Name: "item", Type: "string", Required: true}},
			Steps:   []*rollout.Step{{Name: "read", Type: "browser", Source: "browser-profiles/status.json", Operation: "read_status", With: map[string]string{"item": "inputs.item"}}},
			Outputs: []*rollout.Output{{Name: "status", From: "read.received_body.status"}},
		},
		BrowserRoute: "browser", BrowserSession: "none", Fallback: "stop cleanly", FallbackSet: true,
		SideEffectScope: "read-only", Safety: "read-only", SafetySet: true, CredentialsSet: true,
	}
}

func browserRegistryBundle(t *testing.T, now time.Time) *bundle.Bundle {
	t.Helper()
	profilePath := filepath.Join(repoRoot(t), "examples", "eval", "browser-status-read", "browser-profiles", "status.json")
	profileData, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	value, err := profile.ParseJSON(profileData)
	if err != nil {
		t.Fatal(err)
	}
	record, err := (&bevidence.RawRecord{Record: bevidence.Record{
		Origin: "https://example.test", ObservationKind: bevidence.ObservationA11ySnapshot,
		ObservedAt: now.Add(-time.Hour).Format(time.RFC3339), ActionHint: "read_status",
		CandidateLocators: []bevidence.CandidateLocator{{Role: "status", Name: "Ready"}},
		RedactionStatus:   bevidence.RedactionNotRequired, Provenance: bevidence.Provenance{Tool: "synthetic-engine-test", Version: "1"},
	}}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	reviewed, err := review.Build(value, []bevidence.Record{record}, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	result, err := bundle.Build(bundle.BuildOptions{
		ID: "status", Release: "1.0.0", Source: "reviewed_synthetic_fixture", License: "CC0-1.0",
		Authors: []string{"OpenUdon"}, Profile: value, Review: reviewed, Evidence: []bevidence.Record{record}, PublishedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func firstTestAnswer(question elicitor.QuestionPlan) string {
	if strings.TrimSpace(question.Recommendation) != "" {
		return question.Recommendation
	}
	if strings.TrimSpace(question.SuggestedAnswer) != "" {
		return question.SuggestedAnswer
	}
	return "reviewed answer"
}

func answersForSnapshot(snapshot Snapshot) []authoring.RoundAnswer {
	answers := make([]authoring.RoundAnswer, 0, len(snapshot.Frontier))
	for _, question := range snapshot.Frontier {
		answers = append(answers, authoring.RoundAnswer{
			QuestionID: question.ID, Value: firstTestAnswer(question), Source: "user",
		})
	}
	return answers
}

func hasFrontierQuestion(frontier []elicitor.QuestionPlan, questionID string) bool {
	for _, question := range frontier {
		if question.ID == questionID {
			return true
		}
	}
	return false
}

func hasRevisableDecision(decisions []elicitor.RevisableDecision, questionID, value string) bool {
	for _, decision := range decisions {
		if decision.QuestionID == questionID && decision.Value == value {
			return true
		}
	}
	return false
}

func runtimeFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func assertNoDeliverables(t *testing.T, example string) {
	t.Helper()
	for _, relative := range []string{"project.md", "workflows/intent.hcl", "workflows/intent.draft.hcl"} {
		if _, err := os.Stat(filepath.Join(example, filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Fatalf("deliverable %s exists before approval: %v", relative, err)
		}
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func evidenceIndexByID(values []publicinterview.Evidence, id string) int {
	for index := range values {
		if values[index].ID == id {
			return index
		}
	}
	return -1
}

func assertFileAction(t *testing.T, actions []elicitor.FileAction, action, path string) {
	t.Helper()
	for _, candidate := range actions {
		if candidate.Action == action && candidate.Path == path {
			return
		}
	}
	t.Fatalf("file action %s %s not found in %#v", action, path, actions)
}
