package eval

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicBetaContractAndInstallationAreDocumented(t *testing.T) {
	root := filepath.Join("..", "..")
	checks := map[string][]string{
		"README.md": {
			"v0.1 Public Beta",
			"openudon/cmd/openudon@v0.1.0",
			"openudon version --json",
			"SHA256SUMS",
			"docs/compatibility.md",
			"SUPPORT.md",
			"SECURITY.md",
		},
		"SUPPORT.md": {
			"OpenUdon v0.1 Support Policy",
			"openudon build",
			"openudon run",
			"supported Go-library API",
		},
		"SECURITY.md": {
			"private security-advisory",
			"latest v0.1.x release",
			"Credential values remain",
		},
		filepath.Join("docs", "compatibility.md"): {
			"Stable During v0.1.x",
			"Experimental Before v1",
			"OPENUDON_EXECUTOR",
		},
	}
	for path, expected := range checks {
		text := readRepoFile(t, root, path)
		for _, want := range expected {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q", path, want)
			}
		}
	}
}

func TestReleaseWorkflowPackagesAllPublicCommands(t *testing.T) {
	root := filepath.Join("..", "..")
	workflow := readRepoFile(t, root, ".github", "workflows", "release.yml")
	for _, want := range []string{
		`tags:`,
		`- "v*"`,
		`./cmd/openudon`,
		`./cmd/${COMMAND}`,
		`for COMMAND in icot udon-runner`,
		`SHA256SUMS`,
		`gh release create`,
		`runtime-only-render`,
		`--dry-run`,
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow missing %q", want)
		}
	}
}
