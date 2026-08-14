package elicitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestDiscoverAuthoringSourcesStagesSelectedSecuritySidecar(t *testing.T) {
	sourceRoot := t.TempDir()
	sourcePath := filepath.Join(sourceRoot, "support.json")
	if err := os.WriteFile(sourcePath, []byte(`{"openapi":"3.0.3","info":{"title":"Support","version":"1"},"paths":{"/tickets":{"get":{"operationId":"getTicket"}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "support.security-overlay.json"), []byte(`{"security_schemes":[{"name":"support_token","type":"oauth2"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	discovery, err := DiscoverAuthoringSources(context.Background(), t.TempDir(), "support ticket", nil, []string{sourceRoot})
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Docs) != 1 || len(discovery.Plans) != 2 {
		t.Fatalf("discovery = %#v", discovery)
	}
	session := Session{Intent: rollout.Intent{OpenAPI: discovery.Docs[0].RelativePath}}
	selected := SyncSelectedSourcePlans(session, discovery.Plans, nil)
	if len(selected) != 2 || selected[1].Kind != "security-overlay" {
		t.Fatalf("selected = %#v", selected)
	}
}
