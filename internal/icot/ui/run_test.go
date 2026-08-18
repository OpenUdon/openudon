package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/engine"
)

func TestRunEphemeralPortAutoOpenAndGracefulCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	opened := make(chan string, 1)
	err := Run(ctx, RunConfig{
		EngineConfig: engine.Config{
			ExampleDir:    filepath.Join(t.TempDir(), "target"),
			FromExample:   filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render"),
			NetworkPolicy: "never",
		},
		Port: 0, Out: &stdout, ErrOut: &stderr,
		OpenURL: func(target string) error {
			opened <- target
			cancel()
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case target := <-opened:
		parsed, parseErr := url.Parse(target)
		token := parsed.Query().Get("token")
		if parseErr != nil || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || token == "" || parsed.Path != instanceBasePath(token) || strings.Contains(parsed.Path, token) {
			t.Fatalf("bootstrap URL = %q, %v", target, parseErr)
		}
		if stdout.String() != "icot ui: "+target+"\n" {
			t.Fatalf("stdout = %q", stdout.String())
		}
	default:
		t.Fatal("default opener was not called")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunFixedPortNoOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requested := ""
	opens := 0
	listen := func(network, address string) (net.Listener, error) {
		requested = network + " " + address
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		cancel()
		return listener, err
	}
	err := Run(ctx, RunConfig{
		EngineConfig: engine.Config{
			ExampleDir:    filepath.Join(t.TempDir(), "target"),
			FromExample:   filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render"),
			NetworkPolicy: "never",
		},
		Port: 41234, NoOpen: true, Listen: listen,
		OpenURL: func(string) error { opens++; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if requested != "tcp4 127.0.0.1:41234" {
		t.Fatalf("listen request = %q", requested)
	}
	if opens != 0 {
		t.Fatalf("no-open invoked opener %d times", opens)
	}
}

func TestRunOpenerFailureWarnsAndKeepsServingUntilCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	err := Run(ctx, RunConfig{
		EngineConfig: engine.Config{
			ExampleDir:    filepath.Join(t.TempDir(), "target"),
			FromExample:   filepath.Join(repoRoot(t), "examples", "eval", "runtime-only-render"),
			NetworkPolicy: "never",
		},
		Out: &stdout, ErrOut: &stderr,
		OpenURL: func(string) error {
			cancel()
			return errors.New("opener unavailable")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "warning: could not open browser: opener unavailable") || !strings.Contains(stderr.String(), "icot ui: open http://127.0.0.1:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "?token=") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestGenerateTokenIs256BitAndUnique(t *testing.T) {
	first, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil || len(decoded) != 32 || first == second {
		t.Fatalf("tokens = %q %q, decoded bytes %d, err %v", first, second, len(decoded), err)
	}
}

func TestRunRejectsInvalidPortBeforeListening(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := Run(ctx, RunConfig{EngineConfig: engine.Config{ExampleDir: t.TempDir()}, Port: 65536})
	if err == nil || !strings.Contains(err.Error(), "port") {
		t.Fatalf("error = %v", err)
	}
}
