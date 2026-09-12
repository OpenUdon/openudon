package browsercheck

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCacheCopiesOutputsAndInvalidatesInputs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache")
	c, err := OpenCache(root, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	builds := 0
	build := func(path string) error { builds++; return os.WriteFile(path, []byte("compiled output"), 0700) }
	target := func() string { return filepath.Join(t.TempDir(), "binary") }
	first := target()
	if err := Build(WithCache(context.Background(), c), "tool", first, build); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("runtime copy changed"), 0700); err != nil {
		t.Fatal(err)
	}
	second := target()
	if err := Build(WithCache(context.Background(), c), "tool", second, build); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(second)
	if builds != 1 || string(data) != "compiled output" {
		t.Fatal("cache is not immutable or did not reuse")
	}
	changed, err := OpenCache(root, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err := Build(WithCache(context.Background(), changed), "tool", target(), build); err != nil {
		t.Fatal(err)
	}
	if builds != 2 {
		t.Fatal("changed inputs reused")
	}
	if err := Build(context.Background(), "tool", target(), build); err != nil {
		t.Fatal(err)
	}
	if builds != 3 {
		t.Fatal("qualification inherited development reuse")
	}
}
func TestBuildCacheRejectsCorruptionAndSymlinks(t *testing.T) {
	for _, variant := range []string{"bytes", "mode", "manifest", "entry_link", "artifact_link"} {
		t.Run(variant, func(t *testing.T) {
			c, err := OpenCache(filepath.Join(t.TempDir(), "cache"), strings.Repeat("a", 64))
			if err != nil {
				t.Fatal(err)
			}
			ctx := WithCache(context.Background(), c)
			build := func(path string) error { return os.WriteFile(path, []byte("output"), 0700) }
			if err := Build(ctx, "tool", filepath.Join(t.TempDir(), "output"), build); err != nil {
				t.Fatal(err)
			}
			entries, _ := filepath.Glob(filepath.Join(c.Root, "build-*"))
			entry := entries[0]
			artifact := filepath.Join(entry, "artifact")
			switch variant {
			case "bytes":
				err = os.WriteFile(artifact, []byte("corrupt"), 0700)
			case "mode":
				err = os.Chmod(artifact, 0600)
			case "manifest":
				err = os.WriteFile(filepath.Join(entry, "manifest.json"), []byte(`{"input":"bad","input":"bad"}`), 0600)
			case "entry_link":
				moved := entry + "-moved"
				err = os.Rename(entry, moved)
				if err == nil {
					err = os.Symlink(moved, entry)
				}
			case "artifact_link":
				moved := artifact + "-moved"
				err = os.Rename(artifact, moved)
				if err == nil {
					err = os.Symlink(moved, artifact)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if Build(ctx, "tool", filepath.Join(t.TempDir(), "output"), func(string) error { t.Fatal("corrupt cache silently rebuilt"); return nil }) == nil {
				t.Fatal("corrupt cache accepted")
			}
		})
	}
}
func TestFailedBuildLeavesNoReusableResult(t *testing.T) {
	c, err := OpenCache(filepath.Join(t.TempDir(), "cache"), strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	err = Build(WithCache(context.Background(), c), "tool", filepath.Join(t.TempDir(), "output"), func(path string) error { os.WriteFile(path, []byte("partial"), 0700); return errors.New("failed") })
	entries, _ := os.ReadDir(c.Root)
	if err == nil || len(entries) != 0 {
		t.Fatal("failed build retained")
	}
}
func TestTreeDigestTracksBytesModesAndRejectsEscapingLinks(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "source")
	os.WriteFile(p, []byte("a"), 0600)
	a, err := TreeDigest(root, true)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, []byte("b"), 0600)
	b, _ := TreeDigest(root, true)
	os.Chmod(p, 0700)
	c, _ := TreeDigest(root, true)
	if a == b || b == c {
		t.Fatal("bytes/modes not bound")
	}
	os.Symlink(t.TempDir(), filepath.Join(root, "escape"))
	if _, err := TreeDigest(root, true); err == nil {
		t.Fatal("external symlink accepted")
	}
}
func TestTimingRetainsOnlyClosedMetadata(t *testing.T) {
	p := filepath.Join(t.TempDir(), "trace.jsonl")
	ctx, trace, err := TraceFile(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	Span(Stage(ctx, "registration_ui_handoff"), "stage")(errors.New("private-test-value"))
	if err := trace.Close(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	var row map[string]any
	if json.Unmarshal(data, &row) != nil || row["status"] != "fail" || strings.Contains(string(data), "private-test-value") {
		t.Fatal("timing protocol")
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatal("permissions")
	}
	if _, _, err := TraceFile(context.Background(), p); err == nil {
		t.Fatal("existing timing overwritten")
	}
}

func TestCacheRejectsParentAliasesBeforeCreatingDirectories(t *testing.T) {
	root := t.TempDir()
	destination := t.TempDir()
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(destination, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCache(filepath.Join(alias, "new-cache"), strings.Repeat("a", 64)); err == nil {
		t.Fatal("alias accepted")
	}
	if _, err := os.Lstat(filepath.Join(destination, "new-cache")); !os.IsNotExist(err) {
		t.Fatal("created cache through source alias")
	}
}
func TestResultCacheRefusesChangedRootAndPublicFiles(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cache")
	c, err := OpenCache(root, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.WriteResult("stage", []byte("result")); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(root, "result-*"))
	os.Chmod(files[0], 0644)
	if _, err := c.ReadResult("stage"); err == nil {
		t.Fatal("public result accepted")
	}
	moved := root + "-moved"
	os.Rename(root, moved)
	os.Symlink(moved, root)
	if c.WriteResult("stage", []byte("replacement")) == nil {
		t.Fatal("replaced root accepted")
	}
}
