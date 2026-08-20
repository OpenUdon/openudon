package eval

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCompatibilityContractAndInstallationAreDocumented(t *testing.T) {
	root := filepath.Join("..", "..")
	checks := map[string][]string{
		"README.md": {
			"v0.2 Security Migration",
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
			"Stable During v0.2.x",
			"Experimental Before v1",
			"openudon.executor-run.v2",
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

func TestBrowserScenarioWorkflowsProvisionSandboxUserNamespaces(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, path := range []string{
		filepath.Join(".github", "workflows", "release.yml"),
		filepath.Join(".github", "workflows", "browser-scenario-public.yml"),
	} {
		workflow := readRepoFile(t, root, path)
		for _, want := range []string{
			"Enable Chromium sandbox user namespaces",
			"kernel.apparmor_restrict_unprivileged_userns=0",
			"kernel.unprivileged_userns_clone=1",
		} {
			if !strings.Contains(workflow, want) {
				t.Fatalf("%s missing %q", path, want)
			}
		}
		if strings.Contains(workflow, "--no-sandbox") {
			t.Fatalf("%s disables the Chromium sandbox", path)
		}
	}
}
