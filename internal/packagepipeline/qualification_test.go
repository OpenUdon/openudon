package packagepipeline

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

func TestQualifyUsesRestrictiveScratchRunsAllGatesAndCleans(t *testing.T) {
	prepared, source, parent := pipelinePassingPreparation(t)
	before := pipelineTreeState(t, source)
	qualified, err := Qualify(context.Background(), prepared, QualifyOptions{
		ScratchParent: parent, Now: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	report := qualified.Report()
	if report.Version != QualificationVersion || !strings.HasPrefix(report.QualificationSHA256, "sha256:") || report.PackageSHA256 != prepared.Manifest().PackageSHA256 || len(report.Gates) != 5 {
		t.Fatalf("qualification report = %#v", report)
	}
	for _, gate := range report.Gates {
		if gate.Name == "" || gate.Status != "pass" {
			t.Fatalf("qualification gate = %#v", gate)
		}
	}
	if !reflect.DeepEqual(qualified.Prepared().Manifest(), prepared.Manifest()) {
		t.Fatal("qualification changed the prepared generation")
	}
	if after := pipelineTreeState(t, source); !reflect.DeepEqual(before, after) {
		t.Fatalf("qualification changed the source package:\nbefore=%#v\nafter=%#v", before, after)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("qualification scratch was retained: %#v", entries)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), filepath.ToSlash(parent)) || strings.Contains(string(encoded), filepath.ToSlash(source)) {
		t.Fatalf("qualification report exposed local paths: %s", encoded)
	}
	copy := qualified.Report()
	copy.Gates[0].Name = "changed"
	if qualified.Report().Gates[0].Name == "changed" {
		t.Fatal("qualification report exposed mutable internal state")
	}
	repeated, err := Qualify(context.Background(), prepared, QualifyOptions{
		ScratchParent: parent, Now: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repeated.Report(), report) {
		t.Fatalf("qualification evidence is not deterministic:\nfirst=%#v\nsecond=%#v", report, repeated.Report())
	}
	if entries := mustReadDir(t, parent); len(entries) != 0 {
		t.Fatalf("repeated qualification retained scratch: %#v", entries)
	}
}

func TestQualifyFailureCodesAndCleanup(t *testing.T) {
	prepared, source, parent := pipelinePassingPreparation(t)
	assertCode := func(t *testing.T, err error, want QualificationCode) {
		t.Helper()
		got, ok := QualificationFailureCode(err)
		if !ok || got != want || strings.Contains(err.Error(), parent) {
			t.Fatalf("qualification error = %v, code=%q present=%t, want %q", err, got, ok, want)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Qualify(ctx, prepared, QualifyOptions{ScratchParent: parent})
	assertCode(t, err, QualificationCanceled)

	_, err = Qualify(context.Background(), Prepared{}, QualifyOptions{ScratchParent: parent})
	assertCode(t, err, QualificationInvalidPreparation)

	_, err = Qualify(context.Background(), prepared, QualifyOptions{ScratchParent: "relative"})
	assertCode(t, err, QualificationScratchFailed)

	originalMaterializedHook := qualificationMaterializedHook
	for name, mutate := range map[string]func(string){
		"unsafe mode": func(root string) { _ = os.Chmod(filepath.Join(root, "project.md"), 0o644) },
		"symlink": func(root string) {
			_ = os.Remove(filepath.Join(root, "project.md"))
			_ = os.Symlink(filepath.Join(source, "project.md"), filepath.Join(root, "project.md"))
		},
		"hard link": func(root string) {
			_ = os.Link(filepath.Join(root, "project.md"), filepath.Join(root, "project.alias"))
		},
		"path alias": func(root string) {
			_ = os.WriteFile(filepath.Join(root, "Project.md"), []byte("alias\n"), 0o600)
		},
		"unsupported member": func(root string) {
			_ = os.Mkdir(filepath.Join(root, "unsupported"), 0o700)
		},
	} {
		t.Run(name, func(t *testing.T) {
			qualificationMaterializedHook = mutate
			_, err := Qualify(context.Background(), prepared, QualifyOptions{ScratchParent: parent})
			qualificationMaterializedHook = originalMaterializedHook
			assertCode(t, err, QualificationStructureFailed)
		})
	}

	originalDryRunHook := qualificationBeforeDryRunHook
	qualificationBeforeDryRunHook = func() error { return errors.New("injected dry-run failure") }
	_, err = Qualify(context.Background(), prepared, QualifyOptions{ScratchParent: parent})
	qualificationBeforeDryRunHook = originalDryRunHook
	assertCode(t, err, QualificationDryRunFailed)

	originalCleanupHook := qualificationCleanupHook
	qualificationCleanupHook = func(string) error { return errors.New("injected cleanup failure") }
	_, err = Qualify(context.Background(), prepared, QualifyOptions{ScratchParent: parent})
	qualificationCleanupHook = originalCleanupHook
	assertCode(t, err, QualificationCleanupFailed)
	for _, entry := range mustReadDir(t, parent) {
		if strings.HasPrefix(entry.Name(), ".openudon-package-qualification-") {
			if err := os.RemoveAll(filepath.Join(parent, entry.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestQualifyRejectsStaleStoredQualityInScratch(t *testing.T) {
	example := filepath.Join(t.TempDir(), "stale-package")
	pipelineCopyTree(t, filepath.Join(pipelineRepoRoot(t), "examples", "slack-message-audit-log"), example)
	reviewPath := filepath.Join(example, "expected", "review.md")
	review, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	review = []byte(strings.Replace(string(review), "--dry-run", "--review-only", 1))
	if err := os.WriteFile(reviewPath, review, 0o600); err != nil {
		t.Fatal(err)
	}
	refreshQualificationHandoff(t, example)
	prepared, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: example, Scope: "examples/support-priority-routing"})
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	_, err = Qualify(context.Background(), prepared, QualifyOptions{ScratchParent: parent})
	if code, ok := QualificationFailureCode(err); !ok || code != QualificationQualityFailed {
		t.Fatalf("stale quality error = %v, code=%q", err, code)
	}
	if entries := mustReadDir(t, parent); len(entries) != 0 {
		t.Fatalf("failed qualification retained scratch: %#v", entries)
	}
}

func refreshQualificationHandoff(t *testing.T, example string) {
	t.Helper()
	const self = "expected/review-handoff.json"
	path := filepath.Join(example, filepath.FromSlash(self))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest authoring.ReviewHandoff
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for i := range manifest.HandoffInputs {
		input := &manifest.HandoffInputs[i]
		if input.Path == self {
			input.SHA256 = strings.Repeat("0", 64)
			continue
		}
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(input.Path)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		input.SHA256 = fmt.Sprintf("%x", digest[:])
	}
	digest, err := authoring.ReviewHandoffSelfDigest(manifest, self)
	if err != nil {
		t.Fatal(err)
	}
	for i := range manifest.HandoffInputs {
		if manifest.HandoffInputs[i].Path == self {
			manifest.HandoffInputs[i].SHA256 = digest
		}
	}
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func pipelinePassingPreparation(t *testing.T) (Prepared, string, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "examples", "support-priority-routing")
	pipelineCopyTree(t, filepath.Join(pipelineRepoRoot(t), "examples", "slack-message-audit-log"), source)
	if _, err := synthesize.Build(context.Background(), synthesize.Options{ExampleDir: source}); err != nil {
		t.Fatalf("build passing package fixture: %v", err)
	}
	prepared, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: source, Scope: "examples/support-priority-routing"})
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "scratch")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	return prepared, source, parent
}

func mustReadDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}
