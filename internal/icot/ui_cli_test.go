package icot

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	uiserver "github.com/OpenUdon/openudon/internal/icot/ui"
)

func TestUICommandRequiresExampleAndValidFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"ui", "--no-open"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "--example is required") {
		t.Fatalf("missing example = code %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"ui", "--example", t.TempDir(), "--answers", "a.yaml", "--from-example", "seed"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("conflicting seed = code %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"ui", "--example", t.TempDir(), "--port", "65536"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "--port") {
		t.Fatalf("invalid port = code %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestUICommandSeedPrecedence(t *testing.T) {
	original := runUIServer
	defer func() { runUIServer = original }()
	var captured uiserver.RunConfig
	runUIServer = func(_ context.Context, config uiserver.RunConfig) error {
		captured = config
		return nil
	}

	call := func(t *testing.T, args ...string) uiserver.RunConfig {
		t.Helper()
		captured = uiserver.RunConfig{}
		var stdout, stderr bytes.Buffer
		if code := Main(append([]string{"ui"}, args...), strings.NewReader(""), &stdout, &stderr); code != 0 {
			t.Fatalf("code = %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
		}
		return captured
	}

	t.Run("answers override workspace state", func(t *testing.T) {
		example := makeUIWorkspace(t, true, true)
		config := call(t, "--example", example, "--answers", "explicit.yaml")
		if config.EngineConfig.SessionPath != "explicit.yaml" || config.EngineConfig.FromExample != "" || config.EngineConfig.LoadExisting {
			t.Fatalf("engine config = %#v", config.EngineConfig)
		}
	})
	t.Run("from-example override workspace state", func(t *testing.T) {
		example := makeUIWorkspace(t, true, true)
		config := call(t, "--example", example, "--from-example", "seed-example")
		if config.EngineConfig.FromExample != "seed-example" || config.EngineConfig.SessionPath != "" || config.EngineConfig.LoadExisting {
			t.Fatalf("engine config = %#v", config.EngineConfig)
		}
	})
	t.Run("draft precedes final", func(t *testing.T) {
		example := makeUIWorkspace(t, true, true)
		config := call(t, "--example", example)
		if config.EngineConfig.SessionPath != "" || config.EngineConfig.FromExample != "" || config.EngineConfig.LoadExisting {
			t.Fatalf("engine config = %#v", config.EngineConfig)
		}
	})
	t.Run("final precedes empty", func(t *testing.T) {
		example := makeUIWorkspace(t, false, true)
		config := call(t, "--example", example)
		if !config.EngineConfig.LoadExisting {
			t.Fatalf("engine config = %#v", config.EngineConfig)
		}
	})
	t.Run("empty", func(t *testing.T) {
		example := makeUIWorkspace(t, false, false)
		config := call(t, "--example", example)
		if config.EngineConfig.LoadExisting || config.EngineConfig.SessionPath != "" || config.EngineConfig.FromExample != "" {
			t.Fatalf("engine config = %#v", config.EngineConfig)
		}
	})
}

func TestUICommandMapsSourcesPortNetworkAndOpenPolicy(t *testing.T) {
	original := runUIServer
	defer func() { runUIServer = original }()
	var captured uiserver.RunConfig
	runUIServer = func(_ context.Context, config uiserver.RunConfig) error {
		captured = config
		return nil
	}
	example := t.TempDir()
	args := []string{
		"ui", "--example", example, "--port", "8123", "--no-open", "--network", "allow",
		"--api-source", "openapi:primary=/tmp/primary.yaml", "--openapi", "legacy=/tmp/legacy.yaml",
		"--browser-profile", "status=/tmp/status.json", "--browser-verification", "/tmp/check.json",
		"--browser-registry", "/tmp/registry", "--source-root", "/tmp/sources",
	}
	var stdout, stderr bytes.Buffer
	if code := Main(args, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
	if captured.Port != 8123 || !captured.NoOpen || captured.EngineConfig.NetworkPolicy != "allow" {
		t.Fatalf("run config = %#v", captured)
	}
	if len(captured.EngineConfig.LocalSources) != 2 || len(captured.EngineConfig.BrowserSources) != 1 || len(captured.EngineConfig.BrowserVerifications) != 1 || len(captured.EngineConfig.BrowserRegistries) != 1 || len(captured.EngineConfig.SourceRoots) != 1 {
		t.Fatalf("engine source config = %#v", captured.EngineConfig)
	}
}

func TestUICommandDefaultsAndRunnerFailure(t *testing.T) {
	original := runUIServer
	defer func() { runUIServer = original }()
	var captured uiserver.RunConfig
	runUIServer = func(_ context.Context, config uiserver.RunConfig) error {
		captured = config
		return errors.New("listen failed")
	}
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"ui", "--example", t.TempDir()}, strings.NewReader(""), &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "icot ui: listen failed") {
		t.Fatalf("code = %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
	}
	if captured.Port != 0 || captured.NoOpen || captured.EngineConfig.NetworkPolicy != "ask" {
		t.Fatalf("default run config = %#v", captured)
	}
}

func TestRootHelpStillDocumentsTerminalAuthoring(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"--help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d stderr = %q", code, stderr.String())
	}
	for _, expected := range []string{"Usage: icot --example", "icot reconcile", "icot ui --example"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("help missing %q:\n%s", expected, stdout.String())
		}
	}
}

func makeUIWorkspace(t *testing.T, draft, final bool) string {
	t.Helper()
	example := t.TempDir()
	if draft {
		if err := os.MkdirAll(filepath.Join(example, ".icot"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(example, ".icot", "session.yaml"), []byte("draft"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if final {
		if err := os.WriteFile(filepath.Join(example, "project.md"), []byte("# Existing\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return example
}
