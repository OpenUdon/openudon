package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/engine"
)

const shutdownTimeout = 5 * time.Second

// RunConfig starts one loopback-only UI process.
type RunConfig struct {
	EngineConfig engine.Config
	Port         int
	NoOpen       bool
	Out          io.Writer
	ErrOut       io.Writer
	OpenURL      func(string) error
	Listen       func(network, address string) (net.Listener, error)
}

// Run opens one engine, binds 127.0.0.1, optionally opens the browser, and
// serves until cancellation or an HTTP server failure.
func Run(ctx context.Context, config RunConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if config.Port < 0 || config.Port > 65535 {
		return errors.New("port must be between 0 and 65535")
	}
	if config.Out == nil {
		config.Out = io.Discard
	}
	if config.ErrOut == nil {
		config.ErrOut = io.Discard
	}
	listen := config.Listen
	if listen == nil {
		listen = net.Listen
	}
	authoringEngine, snapshot, err := engine.Open(ctx, config.EngineConfig)
	if err != nil {
		return err
	}
	token, err := GenerateToken()
	if err != nil {
		return err
	}
	accessCode, err := GenerateAccessCode()
	if err != nil {
		return err
	}
	listener, err := listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(config.Port)))
	if err != nil {
		return fmt.Errorf("listen on 127.0.0.1:%d: %w", config.Port, err)
	}
	defer listener.Close()
	authority := listener.Addr().String()
	handler, err := NewHandler(HandlerConfig{
		Engine: authoringEngine, Snapshot: snapshot, ExampleDir: config.EngineConfig.ExampleDir,
		Token: token, AccessCode: accessCode, Authority: authority, ErrOut: config.ErrOut,
	})
	if err != nil {
		return err
	}
	server := newHTTPServer(handler)
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	bootstrap := "http://" + authority + "/"
	fmt.Fprintf(config.Out, "icot ui: %s\n", bootstrap)
	fmt.Fprintf(config.Out, "icot ui access code: %s\n", accessCode)
	if !config.NoOpen {
		opener := config.OpenURL
		if opener == nil {
			opener = openBrowser
		}
		if err := opener(bootstrap); err != nil {
			fmt.Fprintf(config.ErrOut, "icot ui: warning: could not open browser: %v\n", err)
			fmt.Fprintf(config.ErrOut, "icot ui: open %s\n", bootstrap)
		}
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shut down iCoT UI server: %w", err)
		}
		err := <-serveErrors
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve iCoT UI: %w", err)
		}
		return nil
	case err := <-serveErrors:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve iCoT UI: %w", err)
	}
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
}

func openBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}
