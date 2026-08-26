package packagepipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/synthesize"
)

func TestPromotePublishesCompleteGenerationPreservesPriorAndRepeats(t *testing.T) {
	first, source, scratch := qualifiedPromotionFixture(t)
	store := promotionStore(t)

	promoted, err := Promote(context.Background(), first, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	firstSelection := promoted.Selection()
	if firstSelection.Version != SelectionVersion || !validTaggedSHA256(firstSelection.SelectedGenerationSHA256) || firstSelection.PriorGenerationSHA256 != "" || firstSelection.PackageSHA256 != first.Prepared().Manifest().PackageSHA256 {
		t.Fatalf("first selection = %#v", firstSelection)
	}
	if !reflect.DeepEqual(promoted.Files(), first.Prepared().Files()) {
		t.Fatal("promoted files differ from qualified preparation")
	}
	current, err := ReadCurrent(context.Background(), store)
	if err != nil || !reflect.DeepEqual(current.Selection(), firstSelection) || !reflect.DeepEqual(current.Files(), first.Prepared().Files()) {
		t.Fatalf("current first generation = %#v, %v", current.Selection(), err)
	}
	pointerBefore, err := os.ReadFile(filepath.Join(store, promotionCurrentFile))
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := Promote(context.Background(), first, PromotionOptions{StoreDir: store})
	if err != nil || !reflect.DeepEqual(repeated.Selection(), firstSelection) {
		t.Fatalf("repeated promotion = %#v, %v", repeated.Selection(), err)
	}
	pointerAfter, err := os.ReadFile(filepath.Join(store, promotionCurrentFile))
	if err != nil || !reflect.DeepEqual(pointerBefore, pointerAfter) {
		t.Fatal("repeated promotion rewrote the current selection")
	}

	second := changedQualifiedPromotionFixture(t, source, scratch)
	promoted, err = Promote(context.Background(), second, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	secondSelection := promoted.Selection()
	if secondSelection.SelectedGenerationSHA256 == firstSelection.SelectedGenerationSHA256 || secondSelection.PriorGenerationSHA256 != firstSelection.SelectedGenerationSHA256 {
		t.Fatalf("second selection did not preserve exact prior: %#v", secondSelection)
	}
	if entries := mustReadDir(t, filepath.Join(store, promotionGenerationsDir)); len(entries) != 2 {
		t.Fatalf("generation inventory = %#v", entries)
	}
	if _, err := loadGeneration(context.Background(), store, firstSelection.SelectedGenerationSHA256); err != nil {
		t.Fatalf("prior generation was not preserved: %v", err)
	}
	assertPromotionStorePosture(t, store)
	encoded, err := json.Marshal(secondSelection)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), filepath.ToSlash(store)) || strings.Contains(string(encoded), filepath.ToSlash(source)) {
		t.Fatalf("selection exposed local paths: %s", encoded)
	}
}

func TestReadCurrentObservesOldOrNewCompleteGeneration(t *testing.T) {
	first, source, scratch := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	old, err := Promote(context.Background(), first, PromotionOptions{StoreDir: store})
	if err != nil {
		t.Fatal(err)
	}
	second := changedQualifiedPromotionFixture(t, source, scratch)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	promotionBeforeSelectHook = func() {
		once.Do(func() { close(started) })
		<-release
	}
	defer func() { promotionBeforeSelectHook = nil }()
	type promotionResult struct {
		promoted Promoted
		err      error
	}
	done := make(chan promotionResult, 1)
	go func() {
		promoted, err := Promote(context.Background(), second, PromotionOptions{StoreDir: store})
		done <- promotionResult{promoted: promoted, err: err}
	}()
	<-started
	for range 32 {
		current, err := ReadCurrent(context.Background(), store)
		if err != nil || current.Selection().SelectedGenerationSHA256 != old.Selection().SelectedGenerationSHA256 || !reflect.DeepEqual(current.Files(), first.Prepared().Files()) {
			t.Fatalf("reader observed incomplete pre-selection state: %#v, %v", current.Selection(), err)
		}
	}
	close(release)
	outcome := <-done
	if outcome.err != nil {
		t.Fatal(outcome.err)
	}
	current, err := ReadCurrent(context.Background(), store)
	if err != nil || current.Selection().SelectedGenerationSHA256 != outcome.promoted.Selection().SelectedGenerationSHA256 || !reflect.DeepEqual(current.Files(), second.Prepared().Files()) {
		t.Fatalf("reader observed incomplete post-selection state: %#v, %v", current.Selection(), err)
	}
}

func TestConcurrentPromotionFailsClosedAndLeavesConsistentSelection(t *testing.T) {
	first, source, scratch := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	if _, err := Promote(context.Background(), first, PromotionOptions{StoreDir: store}); err != nil {
		t.Fatal(err)
	}
	second := changedQualifiedPromotionFixture(t, source, scratch)

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	promotionBeforeSelectHook = func() {
		once.Do(func() { close(started) })
		<-release
	}
	defer func() { promotionBeforeSelectHook = nil }()
	done := make(chan error, 1)
	go func() {
		_, err := Promote(context.Background(), second, PromotionOptions{StoreDir: store})
		done <- err
	}()
	<-started
	_, competingErr := Promote(context.Background(), second, PromotionOptions{StoreDir: store})
	if code, ok := PromotionFailureCode(competingErr); !ok || code != PromotionBusy {
		t.Fatalf("competing promotion error = %v, code=%q", competingErr, code)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	current, err := ReadCurrent(context.Background(), store)
	if err != nil || !reflect.DeepEqual(current.Files(), second.Prepared().Files()) {
		t.Fatalf("current generation after concurrency = %#v, %v", current.Selection(), err)
	}
	assertPromotionStorePosture(t, store)
}

func TestPromoteRejectsInvalidInputsAndGenerationCollision(t *testing.T) {
	qualified, _, _ := qualifiedPromotionFixture(t)
	store := promotionStore(t)
	assertCode := func(t *testing.T, err error, want PromotionCode) {
		t.Helper()
		code, ok := PromotionFailureCode(err)
		if !ok || code != want || strings.Contains(err.Error(), store) {
			t.Fatalf("promotion error = %v, code=%q present=%t, want=%q", err, code, ok, want)
		}
	}
	_, err := Promote(context.Background(), Qualified{}, PromotionOptions{StoreDir: store})
	assertCode(t, err, PromotionInvalidQualification)
	_, err = Promote(context.Background(), qualified, PromotionOptions{StoreDir: "relative"})
	assertCode(t, err, PromotionStoreFailed)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Promote(ctx, qualified, PromotionOptions{StoreDir: store})
	assertCode(t, err, PromotionCanceled)

	if err := ensureGenerationDirectory(store); err != nil {
		t.Fatal(err)
	}
	record, err := buildGenerationRecord(qualified)
	if err != nil {
		t.Fatal(err)
	}
	collision := generationPath(store, record.GenerationSHA256)
	if err := os.Mkdir(collision, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(collision, "foreign"), []byte("not this generation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Promote(context.Background(), qualified, PromotionOptions{StoreDir: store})
	assertCode(t, err, PromotionGenerationCollision)
	if _, statErr := os.Stat(filepath.Join(store, promotionCurrentFile)); !os.IsNotExist(statErr) {
		t.Fatalf("collision changed current selection: %v", statErr)
	}
	assertNoPromotionTransients(t, store)
}

func qualifiedPromotionFixture(t *testing.T) (Qualified, string, string) {
	t.Helper()
	prepared, source, scratch := pipelinePassingPreparation(t)
	qualified, err := Qualify(context.Background(), prepared, QualifyOptions{
		ScratchParent: scratch, Now: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return qualified, source, scratch
}

func changedQualifiedPromotionFixture(t *testing.T, source, scratch string) Qualified {
	t.Helper()
	projectPath := filepath.Join(source, "project.md")
	project, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, append(project, []byte("\n<!-- next package generation -->\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := synthesize.Build(context.Background(), synthesize.Options{ExampleDir: source}); err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: source, Scope: "examples/support-priority-routing"})
	if err != nil {
		t.Fatal(err)
	}
	qualified, err := Qualify(context.Background(), prepared, QualifyOptions{
		ScratchParent: scratch, Now: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return qualified
}

func promotionStore(t *testing.T) string {
	t.Helper()
	store := filepath.Join(t.TempDir(), "store")
	if err := os.Mkdir(store, 0o700); err != nil {
		t.Fatal(err)
	}
	return store
}

func assertPromotionStorePosture(t *testing.T, store string) {
	t.Helper()
	current, err := os.Lstat(filepath.Join(store, promotionCurrentFile))
	if err != nil || !current.Mode().IsRegular() || current.Mode().Perm() != 0o600 {
		t.Fatalf("current selection posture = %v, %v", current, err)
	}
	generations, err := os.Lstat(filepath.Join(store, promotionGenerationsDir))
	if err != nil || !generations.IsDir() || generations.Mode().Perm() != 0o700 {
		t.Fatalf("generation directory posture = %v, %v", generations, err)
	}
	assertNoPromotionTransients(t, store)
}

func assertNoPromotionTransients(t *testing.T, store string) {
	t.Helper()
	for _, entry := range mustReadDir(t, store) {
		if entry.Name() == promotionLockFile || strings.HasPrefix(entry.Name(), promotionStagePrefix) || strings.HasPrefix(entry.Name(), ".openudon-current-") {
			t.Fatalf("promotion transient was retained: %s", entry.Name())
		}
	}
}
