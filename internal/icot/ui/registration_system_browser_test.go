//go:build browser_system_qualification

package ui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/processgroup"
)

func TestBrowserSystemRealRegistrationUI(t *testing.T) {
	testBrowserSystemRegistrationPackage(t, false)
}
func TestBrowserSystemSupervisedRegistrationPackage(t *testing.T) {
	testBrowserSystemRegistrationPackage(t, true)
}
func TestBrowserSystemTypedRegistrationUI(t *testing.T) {
	testBrowserSystemRegistrationPackage(t, false, true)
}
func testBrowserSystemRegistrationPackage(t *testing.T, control bool, typed ...bool) {
	isTyped := len(typed) != 0 && typed[0]
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal("source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var mutations atomic.Int64
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			mutations.Add(1)
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		if r.Method == "HEAD" {
			return
		}
		if isTyped {
			_, _ = io.WriteString(w, SyntheticRegistrationForm)
			return
		}
		_, _ = io.WriteString(w, `<!doctype html><html><body><main><h1>Create account</h1><form method="post" action="/registration-complete"><label>Email<input name="identifier" autocomplete="email"></label><label>Password<input name="password" type="password"></label><button type="submit">Register</button><p role="status" aria-label="Registration complete">Registration proof marker</p></form></main></body></html>`)
	}))
	defer fixture.Close()
	temp := t.TempDir()
	binary := filepath.Join(temp, "browsertools")
	err = processgroup.Run(ctx, time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", binary, "./cmd/browsertools"}, Dir: filepath.Join(filepath.Dir(root), "browsertools"), Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		t.Fatal("worker_build")
	}
	for _, name := range []string{"private", "scratch", "store"} {
		if os.Mkdir(filepath.Join(temp, name), 0700) != nil {
			t.Fatal("state")
		}
	}
	parent := filepath.Join(root, "eval", "runs")
	if os.MkdirAll(parent, 0700) != nil {
		t.Fatal("workspace")
	}
	example, err := os.MkdirTemp(parent, ".browser-system-ui-")
	if err != nil {
		t.Fatal("workspace")
	}
	defer os.RemoveAll(example)
	application := ""
	if control {
		application = filepath.Join(temp, "icot")
		if err := processgroup.Run(ctx, time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", application, "./cmd/icot"}, Dir: root, Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard}); err != nil {
			t.Fatal("application_build")
		}
	}
	result, err := RunRegistrationQualification(ctx, RegistrationQualificationOptions{
		Typed:                 isTyped,
		ApplicationExecutable: application, RepoRoot: root, BrowsertoolsExecutable: binary, ExampleDir: example, PrivateRoot: filepath.Join(temp, "private"), ScratchParent: filepath.Join(temp, "scratch"), StoreDir: filepath.Join(temp, "store"), Scope: "qualification/brp", ProfileID: "qualification_brp", InitialURL: fixture.URL + "/register?action=startnew", Origin: fixture.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mutations.Load() != 0 || result.Snapshot.Promotion == nil || !result.RetainedQuery {
		t.Fatal("ui_handoff")
	}
}
