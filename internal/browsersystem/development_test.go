package browsersystem

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browsercheck"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

func TestDevelopmentCachePreservesIdentityAndRefusesInvalidEvidence(t *testing.T) {
	now := time.Now().UTC()
	input := strings.Repeat("a", 64)
	original := DevelopmentReport{Version: DevelopmentVersion, Mode: "smoke", InputSHA256: input, ExecutionID: strings.Repeat("b", 32), ExecutedAt: now.Add(-time.Minute), DurationMS: 400, Stage: proof("udon_browser_cli", Tests{Passed: 2, InventorySHA256: strings.Repeat("c", 64)})}
	if !validDevelopment(original, input, original.Stage.ID, now) {
		t.Fatal("valid fixture")
	}
	for _, variant := range []string{"valid", "source", "stage", "future", "expired", "failed", "proof", "authority", "version", "identity"} {
		t.Run(variant, func(t *testing.T) {
			r := original
			switch variant {
			case "source":
				r.InputSHA256 = strings.Repeat("c", 64)
			case "stage":
				r.Stage.ID = "registration_ui"
			case "future":
				r.ExecutedAt = now.Add(time.Second)
			case "expired":
				r.ExecutedAt = now.Add(-25 * time.Hour)
			case "failed":
				r.Stage.Status = "fail"
			case "proof":
				r.Stage.Evidence = json.RawMessage(`{"passed":0}`)
				r.Stage.SHA256 = evidenceHash(r.Stage.Evidence)
			case "authority":
				r.QualifiesRuntime = true
			case "version":
				r.Version = Version
			case "identity":
				r.ExecutionID = strings.Repeat("z", 32)
			}
			if validDevelopment(r, input, original.Stage.ID, now) != (variant == "valid") {
				t.Fatal("invalid cache accepted")
			}
		})
	}
	cache, err := browsercheck.OpenCache(filepath.Join(t.TempDir(), "cache"), input)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(original)
	if err := cache.WriteResult(original.Stage.ID, data); err != nil {
		t.Fatal(err)
	}
	data, err = cache.ReadResult(original.Stage.ID)
	var reused DevelopmentReport
	if err != nil || evidencefile.DecodeStrict(data, &reused) != nil {
		t.Fatal("cache roundtrip")
	}
	reused.Reused = true
	if reused.ExecutionID != original.ExecutionID || !reused.ExecutedAt.Equal(original.ExecutedAt) || !validDevelopment(reused, input, original.Stage.ID, now) {
		t.Fatal("reuse rewrote execution")
	}
	path := filepath.Join(t.TempDir(), "development.json")
	data, _ = json.Marshal(reused)
	os.WriteFile(path, data, 0600)
	if _, err := Verify(path); err == nil {
		t.Fatal("development evidence authorized qualification")
	}
}
func TestDevelopmentStageSelectionIsClosed(t *testing.T) {
	if id, err := developmentStage("smoke", ""); err != nil || id != "registration_ui_handoff" {
		t.Fatal("default")
	}
	for _, pair := range [][2]string{{"live", ""}, {"smoke", "https://www.w8m.com"}, {"fast", "registration_ui"}, {"qualify", ""}} {
		if _, err := developmentStage(pair[0], pair[1]); err == nil {
			t.Fatal("unexpected stage")
		}
	}
	if outsideDevelopmentSources("/work/openudon/result.json", "/work/openudon", "/inputs/udon") || outsideDevelopmentSources("/inputs/cache", "/work/openudon", "/inputs/udon") {
		t.Fatal("source mutation allowed")
	}
}

func TestOnlyInProcessTransactionsUseCustomCache(t *testing.T) {
	for _, id := range inventory("loopback") {
		expected := id == "registration_ui_handoff" || id == "bap_bcp_transaction"
		if cacheableDevelopmentStage(id) != expected {
			t.Fatal("unsupported build environment cached", id)
		}
	}
}

func TestFastDoesNotRequirePrivateExecutorCheckout(t *testing.T) {
	root := t.TempDir()
	outDir, err := os.MkdirTemp("", "openudon-fast-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := RunDevelopment(ctx, DevelopmentOptions{Root: root, UdonRepo: filepath.Join(root, "missing-private-executor"), Out: filepath.Join(outDir, "report.json"), Mode: "fast"})
	if err == nil || report == nil || report.Stage.ID != "openudon_unit" || report.Stage.Status != "fail" {
		t.Fatal("fast did not reach unit checks without private executor", err)
	}
}
