package packagepipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestPrepareCurrentIsDeterministicWriteFreeAndDefensive(t *testing.T) {
	root := pipelineRepoRoot(t)
	example := filepath.Join(root, "examples", "slack-message-audit-log")
	before := pipelineTreeState(t, example)
	first, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: example, Scope: "examples/support-priority-routing"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PrepareCurrent(context.Background(), PrepareOptions{
		ExampleDir: example, Scope: "examples/support-priority-routing", ExpectedInputSHA256: first.Manifest().InputSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Manifest(), second.Manifest()) || first.Manifest().Version != PreparationVersion || first.Manifest().QualityStatus != "pass" || len(first.Manifest().Files) == 0 {
		t.Fatalf("preparation is not deterministic and complete:\nfirst=%#v\nsecond=%#v", first.Manifest(), second.Manifest())
	}
	manifestJSON, err := json.Marshal(first.Manifest())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifestJSON), filepath.ToSlash(root)) {
		t.Fatalf("value-free manifest exposed local root: %s", manifestJSON)
	}
	if after := pipelineTreeState(t, example); !reflect.DeepEqual(before, after) {
		t.Fatalf("preparation mutated source tree:\nbefore=%#v\nafter=%#v", before, after)
	}
	files := first.Files()
	for path, data := range files {
		if len(data) == 0 {
			continue
		}
		data[0] ^= 0xff
		delete(files, path)
		break
	}
	manifest := first.Manifest()
	manifest.Files[0].Path = "changed"
	if reflect.DeepEqual(files, first.Files()) || first.Manifest().Files[0].Path == "changed" {
		t.Fatal("prepared generation exposed mutable internal state")
	}
	if _, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: example, Scope: "examples/support-priority-routing", ExpectedInputSHA256: strings.Repeat("0", 64)}); err == nil || !strings.Contains(err.Error(), "not expected") {
		t.Fatalf("unexpected input generation error = %v", err)
	}
	if _, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: example}); err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("missing portable scope error = %v", err)
	}
}

func TestPrepareCurrentHonorsCancellationAndRejectsDrift(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PrepareCurrent(ctx, PrepareOptions{ExampleDir: filepath.Join(pipelineRepoRoot(t), "examples", "slack-message-audit-log")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled preparation error = %v", err)
	}

	source := filepath.Join(pipelineRepoRoot(t), "examples", "slack-message-audit-log")
	example := filepath.Join(t.TempDir(), "package")
	pipelineCopyTree(t, source, example)
	originalHook := prepareReadHook
	defer func() { prepareReadHook = originalHook }()
	prepareReadHook = func() {
		_ = os.WriteFile(filepath.Join(example, "project.md"), []byte("changed during preparation\n"), 0o600)
	}
	if _, err := PrepareCurrent(context.Background(), PrepareOptions{ExampleDir: example, Scope: "examples/support-priority-routing"}); err == nil || !strings.Contains(err.Error(), "changed during preparation") {
		t.Fatalf("generation drift error = %v", err)
	}
}

func pipelineRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func pipelineTreeState(t *testing.T, root string) []string {
	t.Helper()
	var result []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result = append(result, filepath.ToSlash(relative)+"\x00"+info.Mode().String()+"\x00"+hex.EncodeToString(digest[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(result)
	return result
}

func pipelineCopyTree(t *testing.T, source, target string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("test fixture contains a symlink")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}
