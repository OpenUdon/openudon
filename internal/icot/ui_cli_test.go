package icot

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browsertransaction"
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
	if code := Main([]string{"ui", "--example", t.TempDir(), "--browser-transaction", "transaction.json"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "must be supplied together") {
		t.Fatalf("partial browser transaction = code %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"ui", "--example", t.TempDir(), "--port", "65536"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "--port") {
		t.Fatalf("invalid port = code %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestUICommandStartsTheSharedBrowserTransactionEngine(t *testing.T) {
	original := runUIServer
	defer func() { runUIServer = original }()
	var captured uiserver.RunConfig
	runUIServer = func(_ context.Context, config uiserver.RunConfig) error {
		captured = config
		return nil
	}
	root := t.TempDir()
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "ui-shared-engine", Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: "uws.browser-registration.1.0", SourceSHA256: terminalDigest("a"), ReviewSHA256: terminalDigest("b")}},
		Provenance: browsertransaction.Provenance{
			Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1, ResultSHA256: terminalDigest("c"),
			ObservedAt: time.Now().UTC().Format(time.RFC3339Nano), ExpiresAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://example.test"},
		},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "email", Binding: "account_email"}},
	}
	transactionPath := filepath.Join(root, "transaction.json")
	writeBrowserTransactionTerminalFixture(t, transactionPath, transaction)
	scratch, store := filepath.Join(root, "scratch"), filepath.Join(root, "store")
	for _, path := range []string{scratch, store} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"ui", "--example", filepath.Join(root, "example"), "--browser-transaction", transactionPath,
		"--package-scope", "examples/ui", "--package-scratch", scratch, "--package-store", store, "--no-open",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || captured.BrowserTransactions == nil {
		t.Fatalf("UI transaction config = code %d config %#v stdout %q stderr %q", code, captured, stdout.String(), stderr.String())
	}
	snapshot, err := captured.BrowserTransactions.Observe(context.Background())
	if err != nil || snapshot.Transaction == nil || snapshot.Transaction.ID != "ui-shared-engine" || snapshot.RuntimeExecutionSupported {
		t.Fatalf("UI shared transaction = %#v, %v", snapshot, err)
	}
	for _, forbidden := range []string{transactionPath, scratch, store} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("UI launch output exposed %q: stdout %q stderr %q", forbidden, stdout.String(), stderr.String())
		}
	}
}

func TestUICommandConfiguresInactiveBrowserTransactionEngineForRegistrationAuthoring(t *testing.T) {
	original := runUIServer
	defer func() { runUIServer = original }()
	var captured uiserver.RunConfig
	runUIServer = func(_ context.Context, config uiserver.RunConfig) error {
		captured = config
		return nil
	}
	root := t.TempDir()
	scratch, store := filepath.Join(root, "scratch"), filepath.Join(root, "store")
	for _, path := range []string{scratch, store} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"ui", "--example", filepath.Join(root, "example"),
		"--package-scope", "examples/ui", "--package-scratch", scratch, "--package-store", store, "--no-open",
	}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || captured.BrowserTransactions == nil {
		t.Fatalf("UI inactive transaction config = code %d config %#v stdout %q stderr %q", code, captured, stdout.String(), stderr.String())
	}
	snapshot, err := captured.BrowserTransactions.Observe(context.Background())
	if err != nil || snapshot.Transaction != nil || snapshot.Version == "" || snapshot.RuntimeExecutionSupported {
		t.Fatalf("UI inactive transaction engine = %#v, %v", snapshot, err)
	}
	for _, forbidden := range []string{scratch, store} {
		if strings.Contains(stdout.String()+stderr.String(), forbidden) {
			t.Fatalf("UI launch output exposed %q: stdout %q stderr %q", forbidden, stdout.String(), stderr.String())
		}
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
