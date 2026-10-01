package browsercapture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type captureTestOutput struct{ bytes.Buffer }

func (*captureTestOutput) Close() error { return nil }

func TestCaptureCommandRejectsBeforeWorkerWithoutEchoingInput(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "start.json")
	invalid := []byte(`{"password":"PRIVATE_START_CANARY"}`)
	if err := os.WriteFile(path, invalid, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(invalid)
	for _, args := range [][]string{
		{}, {"--unknown", "PRIVATE_ARG_CANARY"},
		{"--start", path, "--approve-start-sha256", strings.Repeat("0", 64)},
		{"--start", path, "--approve-start-sha256", hex.EncodeToString(sum[:])},
	} {
		out := &captureTestOutput{}
		var diagnostics bytes.Buffer
		code := RunCommand(context.Background(), args, io.NopCloser(strings.NewReader("")), out, &diagnostics)
		if code != 2 || out.Len() != 0 || strings.Contains(diagnostics.String(), "PRIVATE_") || strings.Contains(diagnostics.String(), root) {
			t.Fatalf("unsafe invalid-start result: %d %s %s", code, out.String(), diagnostics.String())
		}
	}
}
