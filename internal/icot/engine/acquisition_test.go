package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const uploadedOpenAPI = `{
  "openapi": "3.0.3",
  "info": {"title": "Member API", "version": "1.0.0"},
  "paths": {"/members": {"get": {"operationId": "listMembers", "responses": {"200": {"description": "ok"}}}}}
}`

func TestJourneyAndUploadedSourceLifecycle(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "example")
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: privateRoot, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := eng.SelectJourney(context.Background(), "api", "List the members visible to an operator")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Journey.Starter != "api" || snapshot.Journey.Goal == "" {
		t.Fatalf("journey = %#v", snapshot.Journey)
	}
	uploaded, snapshot, err := eng.UploadSource(context.Background(), "member-api.json", strings.NewReader(uploadedOpenAPI))
	if err != nil {
		t.Fatal(err)
	}
	if uploaded.Kind != "openapi" || uploaded.CanonicalTarget != "openapi/member-api.json" || len(snapshot.UploadedSources) != 1 {
		t.Fatalf("uploaded = %#v, snapshot = %#v", uploaded, snapshot.UploadedSources)
	}
	snapshot, err = eng.StageUploadedSource(context.Background(), uploaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.UploadedSources) != 0 || len(snapshot.StagedSources) != 1 || len(snapshot.SourceCandidates.Local.Candidates) != 1 {
		t.Fatalf("staged snapshot = uploads %#v staged %#v candidates %#v", snapshot.UploadedSources, snapshot.StagedSources, snapshot.SourceCandidates.Local.Candidates)
	}
	if data, err := os.ReadFile(filepath.Join(example, "openapi", "member-api.json")); err != nil || !strings.Contains(string(data), "listMembers") {
		t.Fatalf("staged bytes = %q, %v", data, err)
	}
	snapshot, err = eng.RemoveStagedSource(context.Background(), uploaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.StagedSources) != 0 {
		t.Fatalf("staged sources after removal = %#v", snapshot.StagedSources)
	}
	if _, err := os.Stat(filepath.Join(example, "openapi", "member-api.json")); !os.IsNotExist(err) {
		t.Fatalf("removed source stat = %v", err)
	}
}

func TestUploadRejectsSecretsAndRequiresPrivateRoot(t *testing.T) {
	example := t.TempDir()
	eng, _, err := Open(context.Background(), Config{ExampleDir: example, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := eng.UploadSource(context.Background(), "api.json", strings.NewReader(uploadedOpenAPI)); err == nil || !strings.Contains(err.Error(), "--private-root") {
		t.Fatalf("missing private root error = %v", err)
	}
	privateRoot := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	eng, _, err = Open(context.Background(), Config{ExampleDir: example, PrivateRoot: privateRoot, NetworkPolicy: "never"})
	if err != nil {
		t.Fatal(err)
	}
	secret := strings.Replace(uploadedOpenAPI, `"paths"`, `"api_key":"sk-proj-012345678901234567890123456789","paths"`, 1)
	if _, _, err := eng.UploadSource(context.Background(), "api.json", strings.NewReader(secret)); err == nil || !strings.Contains(err.Error(), "secret-like") {
		t.Fatalf("secret upload error = %v", err)
	}
}

func TestPrivateRootMustBeExactModeAndDisjoint(t *testing.T) {
	example := t.TempDir()
	inside := filepath.Join(example, "private")
	if err := os.Mkdir(inside, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: inside}); err == nil || !strings.Contains(err.Error(), "disjoint") {
		t.Fatalf("inside private root error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(outside, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(context.Background(), Config{ExampleDir: example, PrivateRoot: outside}); err == nil || !strings.Contains(err.Error(), "0700") {
		t.Fatalf("mode private root error = %v", err)
	}
}
