package packagepipeline

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

func TestCurrentLifecycleAndSelectedCompatibilityAdapters(t *testing.T) {
	_, source, scratch := pipelinePassingPreparation(t)
	store := promotionStore(t)
	qualified, err := PrepareAndQualifyCurrent(context.Background(), CurrentOptions{
		ExampleDir: source, Scope: "examples/support-priority-routing", ScratchParent: scratch,
	})
	if err != nil {
		t.Fatal(err)
	}
	promoted, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	selection := promoted.Selection()

	inspection, err := InspectSelected(context.Background(), store, selection.SelectionSHA256)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Scope != selection.Scope || inspection.PackageSHA256 != selection.PackageSHA256 || inspection.HandoffSHA256 != qualified.Prepared().Manifest().HandoffSHA256 {
		t.Fatalf("selected inspection = %#v", inspection)
	}
	approval, err := ApprovalTemplateSelected(context.Background(), store, selection.SelectionSHA256, trustedrunner.TemplateOptions{
		State: trustedrunner.StateApprovedForSandbox, Reviewer: "Package reviewer",
		Now: func() time.Time { return time.Date(2026, 8, 26, 12, 30, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if approval.Scope != selection.Scope || approval.PackageSHA256 != selection.PackageSHA256 {
		t.Fatalf("selected approval = %#v", approval)
	}
	approvalPath := filepath.Join(t.TempDir(), "approval.json")
	file, err := os.OpenFile(approvalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := trustedrunner.WriteApproval(file, approval); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	run, err := RunSelected(context.Background(), store, selection.SelectionSHA256, trustedrunner.Options{
		Tier: trustedrunner.TierSandbox, ApprovalPath: approvalPath,
		WorkDir: filepath.Join(t.TempDir(), "dry-run"), DryRun: true, Env: []string{},
		Now: func() time.Time { return time.Date(2026, 8, 26, 12, 31, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if run == nil || !run.DryRun || run.Scope != selection.Scope || run.PackageSHA256 != selection.PackageSHA256 {
		t.Fatalf("selected dry-run = %#v", run)
	}
	current, err := ReadCurrent(context.Background(), store)
	if err != nil || !reflect.DeepEqual(current.Selection(), selection) || !reflect.DeepEqual(current.Files(), qualified.Prepared().Files()) {
		t.Fatalf("compatibility adapters changed selection: %#v, %v", current.Selection(), err)
	}
}

func TestSelectedCompatibilityRequiresExactCurrentObservation(t *testing.T) {
	qualified, _, _ := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	promoted, err := Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	for name, digest := range map[string]string{
		"missing": "",
		"stale":   "sha256:" + strings.Repeat("0", 64),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := InspectSelected(context.Background(), store, digest); err == nil {
				t.Fatal("selected inspection accepted a missing or stale observation")
			}
		})
	}
	if _, err := InspectSelected(context.Background(), store, promoted.Selection().SelectionSHA256); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "store-link")
	if err := os.Symlink(store, link); err != nil {
		t.Fatal(err)
	}
	if _, err := RunSelected(context.Background(), store, promoted.Selection().SelectionSHA256, trustedrunner.Options{
		ApprovalPath: filepath.Join(t.TempDir(), "approval.json"), WorkDir: filepath.Join(link, "run"), DryRun: true,
	}); err == nil {
		t.Fatal("selected run accepted a symlink-resolved work directory inside the generation store")
	}
}

func TestPromoteCurrentUsesTheExactCompatibilityLifecycle(t *testing.T) {
	prepared, source, scratch := pipelinePassingPreparation(t)
	store := promotionStore(t)
	promoted, err := PromoteCurrent(context.Background(), CurrentOptions{
		ExampleDir: source, Scope: prepared.Manifest().Scope, ExpectedInputSHA256: prepared.Manifest().InputSHA256,
		ScratchParent: scratch, StoreDir: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if promoted.Selection().PackageSHA256 != prepared.Manifest().PackageSHA256 || !reflect.DeepEqual(promoted.Files(), prepared.Files()) {
		t.Fatalf("promote-current changed supported package bytes: %#v", promoted.Selection())
	}
}
