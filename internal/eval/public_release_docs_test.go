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
		`for COMMAND in udon-runner`,
		`SHA256SUMS`,
		`gh release create`,
		`runtime-only-render`,
		`--dry-run`,
		`GOWORK=off go build ./cmd/openudon ./cmd/udon-runner`,
	} {
		if !strings.Contains(workflow, want) {
			t.Fatalf("release workflow missing %q", want)
		}
	}
}

func TestRequiredCaptureGateRetainsBothNativeModesAndSandbox(t *testing.T) {
	root := filepath.Join("..", "..")
	makefile := readRepoFile(t, root, "Makefile")
	for _, want := range []string{"browser-capture-check:", "sandbox-disable override is forbidden", "-tags=browser_system_qualification", "./internal/capturequalification", "TestNeutral(Registration|Authenticated)PublicCapturePackage"} {
		if !strings.Contains(makefile, want) {
			t.Fatalf("capture gate missing %q", want)
		}
	}
	for _, forbidden := range []string{"./cmd/icot", "./internal/authoringui", "icot-ui-browser-check:", "--no-sandbox"} {
		if strings.Contains(makefile, forbidden) {
			t.Fatalf("retired or unsafe release gate %q", forbidden)
		}
	}
}

func TestNormalCIStartsWithStandalonePublicBuild(t *testing.T) {
	workflow := readRepoFile(t, filepath.Join("..", ".."), ".github", "workflows", "test.yml")
	build := strings.Index(workflow, "GOWORK=off go build ./cmd/openudon ./cmd/udon-runner")
	tests := strings.Index(workflow, "GOWORK=off go test ./...")
	if build < 0 || tests < 0 || build > tests {
		t.Fatal("normal CI must build standalone public commands before the full test suite")
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

func TestBrowserScenarioWorkflowsUseLockedPrivateUdonCheckout(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, path := range []string{
		filepath.Join(".github", "workflows", "release.yml"),
		filepath.Join(".github", "workflows", "browser-scenario-public.yml"),
	} {
		workflow := readRepoFile(t, root, path)
		if filepath.Base(path) == "release.yml" {
			for _, want := range []string{
				`current-compatibility-lock-v5.json`, `current-qualification-build-inputs-v5.json`,
				`conflicting browser/executor source locks`, `len(closure["components"]) != 14 or len(selected) != 16`,
				`GENELET_READ_TOKEN: ${{ secrets.GENELET_READ_TOKEN }}`,
				`GIT_CONFIG_GLOBAL="/dev/null"`, `GIT_CONFIG_COUNT="1"`,
				`GIT_CONFIG_VALUE_0="AUTHORIZATION: basic " + authorization`,
				`actual != revision or dirty`, `"--detach", revision`,
			} {
				if !strings.Contains(workflow, want) {
					t.Fatalf("%s missing current locked closure control %q", path, want)
				}
			}
			for _, unsafe := range []string{`git config --global`, `git config --local`, `https://x-access-token:`} {
				if strings.Contains(workflow, unsafe) {
					t.Fatalf("%s persists source credentials: %q", path, unsafe)
				}
			}
			continue
		}
		for _, want := range []string{
			`for COMPONENT in browsertools browserdriver`,
			`select(.name == "udon")`,
			`repository: genelet/udon`,
			`ref: ${{ steps.browser-lock.outputs.udon_commit }}`,
			`token: ${{ secrets.GENELET_READ_TOKEN }}`,
			`persist-credentials: false`,
		} {
			if !strings.Contains(workflow, want) {
				t.Fatalf("%s missing private Udon checkout control %q", path, want)
			}
		}
		if strings.Contains(workflow, `for COMPONENT in browsertools udon browserdriver`) {
			t.Fatalf("%s attempts an anonymous Udon checkout", path)
		}
	}
}
