package engine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCaptureWorkspaceDetectsSameMetadataByteChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "openapi.yaml")
	if err := os.WriteFile(path, []byte("original-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := captureWorkspace(context.Background(), root, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("modified-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, time.Now().Add(-time.Hour), beforeInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(beforeInfo, afterInfo) || beforeInfo.Size() != afterInfo.Size() || !beforeInfo.ModTime().Equal(afterInfo.ModTime()) {
		t.Fatalf("test did not preserve metadata: before %#v after %#v", beforeInfo, afterInfo)
	}
	after, err := captureWorkspace(context.Background(), root, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	if after.digest == before.digest {
		t.Fatalf("same-metadata byte change was not detected: %s", after.digest)
	}
}

// BenchmarkCaptureWorkspaceOneMiB measures the exact-byte polling posture
// against one materialized-source-sized file plus the normal small missing
// path set. It intentionally does not introduce a metadata-only cache: same
// inode, size, and mtime are not proof that file bytes remain unchanged.
func BenchmarkCaptureWorkspaceOneMiB(b *testing.B) {
	root := b.TempDir()
	materialized := filepath.Join(root, "openapi", "materialized.yaml")
	if err := os.MkdirAll(filepath.Dir(materialized), 0o755); err != nil {
		b.Fatal(err)
	}
	content := bytes.Repeat([]byte("x"), 1<<20)
	if err := os.WriteFile(materialized, content, 0o600); err != nil {
		b.Fatal(err)
	}
	paths := []string{
		materialized,
		filepath.Join(root, "project.md"),
		filepath.Join(root, "workflows", "intent.hcl"),
		filepath.Join(root, "workflows", "intent.draft.hcl"),
		filepath.Join(root, ".icot", "session.yaml"),
		filepath.Join(root, ".icot", "readiness.json"),
		filepath.Join(root, ".icot", "browser-sources.json"),
		filepath.Join(root, ".icot", "browser-authentication.json"),
	}
	want, err := captureWorkspace(context.Background(), root, paths)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		got, err := captureWorkspace(context.Background(), root, paths)
		if err != nil {
			b.Fatal(err)
		}
		if got.digest != want.digest {
			b.Fatalf("workspace digest changed: got %s want %s", got.digest, want.digest)
		}
	}
}
