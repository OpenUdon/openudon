package artifactwriter

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
)

func TestProposedFileActionsMatchPreparedTransaction(t *testing.T) {
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)

	t.Run("complete-removals", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "complete")
		prepared, err := Prepare(example, elicitor.Artifacts{ProjectMD: "project\n", IntentHCL: "intent\n"}, true, now)
		if err != nil {
			t.Fatal(err)
		}
		actions := ProposedFileActions(prepared)
		assertActionsCoverPreparedFiles(t, prepared, actions)
		for _, relative := range []string{".icot/browser-authentication.json", ".icot/browser-sources.json", ".icot/readiness.json", ".icot/session.yaml", "workflows/intent.draft.hcl"} {
			assertAction(t, actions, "remove_if_present", filepath.Join(example, filepath.FromSlash(relative)))
		}
	})

	t.Run("incomplete-authentication-writes", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "incomplete")
		materialized := []byte("reviewed authentication profile\n")
		digest := fmt.Sprintf("%x", sha256.Sum256(materialized))
		session := elicitor.Session{SourcePlan: []elicitor.SourceMaterialization{{
			Kind: "browser-authentication", ID: "member", SourcePath: "reviewed-auth.json", TargetPath: "browser-authentication/member.json", SHA256: digest,
			Title: "Member authentication", Flows: []string{"sign_in"}, FlowCredentialSlots: map[string][]string{"sign_in": {"username", "password"}},
			Origins: []string{"https://example.test"}, Lifecycle: "active", ExpiresAt: "2126-08-18T12:00:00Z", Provenance: "reviewed test fixture",
			MaterializedContent: materialized,
		}}}
		prepared, err := Prepare(example, elicitor.Artifacts{ProjectMD: "project\n", IntentHCL: "draft intent\n", Incomplete: true, Session: session}, true, now)
		if err != nil {
			t.Fatal(err)
		}
		actions := ProposedFileActions(prepared)
		assertActionsCoverPreparedFiles(t, prepared, actions)
		assertAction(t, actions, "write", filepath.Join(example, ".icot", "browser-authentication.json"))
		assertAction(t, actions, "write", filepath.Join(example, ".icot", "session.yaml"))
		assertAction(t, actions, "write", filepath.Join(example, ".icot", "readiness.json"))
		assertAction(t, actions, "copy", filepath.Join(example, "browser-authentication", "member.json"))
	})
}

func assertActionsCoverPreparedFiles(t *testing.T, prepared Prepared, actions []elicitor.FileAction) {
	t.Helper()
	if len(actions) != len(prepared.Files) {
		t.Fatalf("actions cover %d files, prepared transaction has %d: %#v", len(actions), len(prepared.Files), actions)
	}
	for _, file := range prepared.Files {
		action := file.Action
		if action == "" {
			action = "write"
			if file.Remove {
				action = "remove_if_present"
			}
		}
		assertAction(t, actions, action, file.Path)
	}
}

func assertAction(t *testing.T, actions []elicitor.FileAction, action, path string) {
	t.Helper()
	for _, candidate := range actions {
		if candidate.Action == action && candidate.Path == path {
			return
		}
	}
	t.Fatalf("action %s %s not found in %#v", action, path, actions)
}
