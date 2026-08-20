package evidencefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRegularRejectsSymlinkAndOversize(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err == nil {
		if _, _, err := ReadRegular(link, 10); err == nil {
			t.Fatal("ReadRegular accepted a symlink")
		}
	}
	if _, _, err := ReadRegular(target, 4); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("ReadRegular oversize error = %v", err)
	}
}

func TestDecodeStrictRejectsUnknownAndTrailingJSON(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}
	if err := DecodeStrict([]byte(`{"name":"ok","extra":true}`), &dst); err == nil {
		t.Fatal("DecodeStrict accepted an unknown field")
	}
	if err := DecodeStrict([]byte(`{"name":"ok"} {}`), &dst); err == nil {
		t.Fatal("DecodeStrict accepted trailing JSON")
	}
	if err := DecodeStrict([]byte(`{"outer":{"name":"one","name":"two"}}`), &dst); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("DecodeStrict duplicate error = %v", err)
	}
}

func TestGitObjectRequiresFullObjectID(t *testing.T) {
	if ValidGitObject("deadbeef") {
		t.Fatal("short object ID was accepted")
	}
	if !ValidGitObject(strings.Repeat("a", 40)) || !ValidGitObject(strings.Repeat("b", 64)) {
		t.Fatal("full object ID was rejected")
	}
}

func TestWriteDigestSidecar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	if err := WriteDigestSidecar(path, []byte("evidence\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	want := "sha256:" + SHA256([]byte("evidence\n")) + "  report.json\n"
	if string(data) != want {
		t.Fatalf("sidecar = %q, want %q", data, want)
	}
	info, err := os.Stat(path + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("sidecar mode = %o, want 640", got)
	}
}

func TestWriteDigestSidecarRejectsEmptyEvidencePath(t *testing.T) {
	if err := WriteDigestSidecar(" ", []byte("evidence\n"), 0o640); err == nil {
		t.Fatal("WriteDigestSidecar accepted an empty evidence path")
	}
}
