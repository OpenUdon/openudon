package elicitor

import (
	"strings"
	"testing"

	"github.com/OpenUdon/browsertools"
)

func TestAssessSourceDiscoveryBlocksInactiveBrowserCandidates(t *testing.T) {
	discovery := LocalSourceDiscovery{
		BrowserReport: browsertools.LocalSourceDiscoveryReport{
			Candidates: []browsertools.LocalSourceCandidate{{Path: "/operator/profile.json", Status: "expired"}},
			Rejected:   []browsertools.LocalSourceDiagnostic{},
			Ambiguous:  []browsertools.LocalSourceDiagnostic{},
			Truncated:  []browsertools.LocalSourceDiagnostic{},
		},
	}
	issues := AssessSourceDiscovery(discovery)
	if len(issues) != 1 || issues[0].Code != "browser_source_discovery_blocked" || issues[0].Severity != readinessBlocking {
		t.Fatalf("inactive browser assessment = %#v", issues)
	}
}

func TestRequireFreshRegistrySourcesBindsExactCoordinateTargetAndDigests(t *testing.T) {
	selected := SourceMaterialization{
		Kind: browserSourceFamily, ID: "status", Registry: "https://registry.example.test",
		RegistryCoordinate: "example/status@1.0.0", TargetPath: "browser-profiles/status.json",
		SHA256: strings.Repeat("a", 64), SourceSHA256: strings.Repeat("b", 64),
	}
	if err := RequireFreshRegistrySources([]SourceMaterialization{selected}, []SourceMaterialization{selected}); err != nil {
		t.Fatalf("exact registry source was rejected: %v", err)
	}
	for _, mutate := range []func(*SourceMaterialization){
		func(value *SourceMaterialization) { value.RegistryCoordinate = "example/status@2.0.0" },
		func(value *SourceMaterialization) { value.TargetPath = "browser-profiles/other.json" },
		func(value *SourceMaterialization) { value.SHA256 = strings.Repeat("c", 64) },
		func(value *SourceMaterialization) { value.SourceSHA256 = strings.Repeat("d", 64) },
	} {
		candidate := selected
		mutate(&candidate)
		if err := RequireFreshRegistrySources([]SourceMaterialization{selected}, []SourceMaterialization{candidate}); err == nil {
			t.Fatalf("mismatched registry source was accepted: %#v", candidate)
		}
	}
}
