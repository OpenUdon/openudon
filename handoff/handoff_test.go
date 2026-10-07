package handoff_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/handoff"
)

func TestPublishedHandoffCanonicalBytes(t *testing.T) {
	data, err := os.ReadFile("../docs/examples/simulation/v1/example/expected/review-handoff.json")
	if err != nil {
		t.Fatal(err)
	}
	var value handoff.ReviewHandoff
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(value)
	if digest.SHA256(before) != "fdd488872b60545bafbc6811427ff6b66366011d01b48155e4cf8fbaeba5971a" {
		t.Fatal("published handoff bytes changed")
	}
	got, err := handoff.ReviewHandoffSelfDigest(value, "expected/review-handoff.json")
	if err != nil {
		t.Fatal(err)
	}
	if got != "ddbb12cfdb439be26ac2f2757b77799b4040f8b937696ea08b2827fd851ea14f" {
		t.Fatal("published canonical digest changed")
	}
	after, _ := json.Marshal(value)
	if !bytes.Equal(before, after) {
		t.Fatal("digest modified caller input")
	}
	if _, err := handoff.ReviewHandoffSelfDigest(value, "absent.json"); err == nil {
		t.Fatal("missing self entry accepted")
	}
	value.HandoffInputs[0].SHA256 = "changed"
	changed, _ := handoff.ReviewHandoffSelfDigest(value, "expected/review-handoff.json")
	if changed == got {
		t.Fatal("artifact digest drift was ignored")
	}
}

func TestPackageDigestV1Golden(t *testing.T) {
	files := []handoff.DigestFile{{Path: "project.md", SHA256: digest.SHA256([]byte("brief\n"))}, {Path: "expected/quality.json", SHA256: digest.SHA256([]byte("{}\n"))}}
	got, err := handoff.DigestFiles("examples/demo", "openudon.handoff-package-digest.v1", files)
	if err != nil {
		t.Fatal(err)
	}
	if got != "96922a875b13b2fca26ab644d3ed879b508a6a4a0afc6a55b06c41c22934339c" {
		t.Fatal("package digest-v1 envelope changed")
	}
	files = append(files, files[0])
	if _, err := handoff.DigestFiles("examples/demo", "", files); err == nil {
		t.Fatal("ambiguous file inventory accepted")
	}
}
