package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
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
		if parseErr != nil || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" || parsed.Path != "/" || parsed.RawQuery != "" {
			t.Fatalf("bootstrap URL = %q, %v", target, parseErr)
		}
		if !strings.HasPrefix(stdout.String(), "icot ui: "+target+"\nicot ui access code: ") {
			t.Fatalf("stdout = %q", stdout.String())
		}
		code := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(stdout.String(), "\n", 2)[1], "icot ui access code: "))
		if len(code) != 12 || strings.ContainsAny(code, "ILOU") {
			t.Fatalf("access code = %q", code)
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
	if strings.Contains(stdout.String(), "token=") || !strings.Contains(stdout.String(), "icot ui access code: ") {
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

func TestGenerateAccessCodeIsCrockfordAndUnique(t *testing.T) {
	first, err := GenerateAccessCode()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateAccessCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 12 || first == second || strings.ContainsAny(first, "ILOU") {
		t.Fatalf("access codes = %q %q", first, second)
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

func TestHTTPServerResourceLimits(t *testing.T) {
	server := newHTTPServer(http.NotFoundHandler())
	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 15*time.Second || server.IdleTimeout != 30*time.Second {
		t.Fatalf("server timeouts = header %v read %v idle %v", server.ReadHeaderTimeout, server.ReadTimeout, server.IdleTimeout)
	}
	if server.WriteTimeout != 0 || server.MaxHeaderBytes != 32<<10 {
		t.Fatalf("server resource limits = write %v headers %d", server.WriteTimeout, server.MaxHeaderBytes)
	}
}
