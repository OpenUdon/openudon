package qualityremediation

import (
	"strings"
	"testing"
)

func TestNextActionCoversSecurityAndArtifactFamilies(t *testing.T) {
	for _, code := range []string{"project.present", "openapi.local", "intent.data_flow.required_params", "credentials.bindings", "side_effects.environment", "review_handoff.input_sha256", "artifacts.no_secrets", "unknown"} {
		if action := NextAction(code); strings.TrimSpace(action) == "" {
			t.Fatalf("NextAction(%q) is empty", code)
		}
	}
}
