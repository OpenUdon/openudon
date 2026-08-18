package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
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
	icotcli "github.com/OpenUdon/openudon/internal/icot"
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
	if code := icotcli.Main([]string{"--example", cliDir, "--from-example", fixture, "--no-llm", "--yes"}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("CLI exit code = %d", code)
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
