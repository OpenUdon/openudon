package trust_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/trust"
)

func snapshot(t *testing.T) trust.Request {
	t.Helper()
	const self = "expected/review-handoff.json"
	content := []byte("opaque bytes; verification does not execute or assess them\n")
	manifest := handoff.NewReviewHandoff(handoff.ReviewHandoffOptions{
		HandoffInputs: []handoff.ReviewHandoffInput{{Path: "artifact.bin", Required: true, SHA256: digest.SHA256(content)}, {Path: self, Required: true, SHA256: strings.Repeat("0", 64)}},
		OwnerSplit:    handoff.ReviewOwnerSplit{"host": {"assessment and authority"}},
	})
	selfDigest, err := handoff.ReviewHandoffSelfDigest(manifest, self)
	if err != nil {
		t.Fatal(err)
	}
	for i := range manifest.HandoffInputs {
		if manifest.HandoffInputs[i].Path == self {
			manifest.HandoffInputs[i].SHA256 = selfDigest
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return trust.Request{Scope: "packages/demo", ManifestPath: self, Files: map[string][]byte{self: data, "artifact.bin": content}, RequiredPaths: []string{"artifact.bin"}}
}

func TestInspectOpaqueSnapshotAndRefuseDrift(t *testing.T) {
	request := snapshot(t)
	result, err := trust.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageSHA256 == "" || result.HandoffSHA256 != digest.SHA256(request.Files[request.ManifestPath]) {
		t.Fatal("missing exact identity")
	}
	request.Files["artifact.bin"][0] = 'X'
	if _, err := trust.Inspect(context.Background(), request); !errors.Is(err, trust.ErrInvalidPackage) {
		t.Fatal("accepted byte drift")
	}
	if result.Manifest.HandoffInputs[0].SHA256 == digest.SHA256(request.Files["artifact.bin"]) {
		t.Fatal("returned manifest aliases bytes")
	}
}

func TestInspectRefusalsAreBoundedAndValueFree(t *testing.T) {
	for name, mutate := range map[string]func(*trust.Request){
		"unlisted":  func(r *trust.Request) { r.Files["private-secret.txt"] = []byte("secret") },
		"missing":   func(r *trust.Request) { delete(r.Files, "artifact.bin") },
		"inventory": func(r *trust.Request) { r.RequiredPaths = append(r.RequiredPaths, "missing.bin") },
		"path":      func(r *trust.Request) { r.ManifestPath = "../private-secret" },
		"scope":     func(r *trust.Request) { r.Scope = "../private-secret" },
		"oversize":  func(r *trust.Request) { r.Files["artifact.bin"] = make([]byte, trust.MaxFileBytes+1) },
		"unknown":   func(r *trust.Request) { r.Files[r.ManifestPath] = []byte(`{"secret":"private-value"}`) },
		"duplicate": func(r *trust.Request) { r.Files[r.ManifestPath] = []byte(`{"version":"one","Version":"two"}`) },
	} {
		t.Run(name, func(t *testing.T) {
			r := snapshot(t)
			mutate(&r)
			_, err := trust.Inspect(context.Background(), r)
			if !errors.Is(err, trust.ErrInvalidPackage) || strings.Contains(err.Error(), "private") {
				t.Fatalf("refusal: %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := trust.Inspect(ctx, snapshot(t)); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
	if _, err := trust.Inspect(nil, snapshot(t)); err == nil {
		t.Fatal("nil context accepted")
	}
}
