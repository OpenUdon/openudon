//go:build browser_system_qualification

package ui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
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
func TestBrowserSystemVerificationRegistrationUI(t *testing.T) {
	testBrowserSystemRegistrationPackage(t, false, true, true)
}

func TestBrowserSystemRegistrationFailureDiagnosticUI(t *testing.T) {
	fake := &fakeEngine{}
	session := newFakeRegistrationAuthoringSession()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	handler, err := NewHandler(HandlerConfig{
		Context: ctx, Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example",
		Token: testToken, AccessCode: registrationQualificationCode, Authority: testAuthority, PrivateRoot: "/tmp/private",
		BrowserTransactions: newFakeBrowserTransactions(),
		StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
			return session, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	initial := currentResponse(t, handler)
	q, err := newRegistrationBrowserQualification(ctx, handler)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := q.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := q.act("/api/v4/registration-authoring/start", []byte(registrationStartBody(initial, "https://app.example.test/register"))); err != nil {
		t.Fatal(err)
	}
	session.events <- browserauthor.RegistrationEvent{State: "failed", ErrorCode: "worker_failed", Diagnostic: "browser_failure"}
	session.close()
	for ctx.Err() == nil {
		status, err := q.page.Locator("#registration-authoring-status").InnerText()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(status, "Worker diagnostic: browser_failure.") && strings.Contains(status, "consumed") {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("validated worker failure did not reach the rendered UI")
}
func testBrowserSystemRegistrationPackage(t *testing.T, control bool, typed ...bool) {
	isTyped := len(typed) != 0 && typed[0]
	isVerification := len(typed) > 1 && typed[1]
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
			form := SyntheticRegistrationForm
			if isVerification {
				form = SyntheticVerificationRegistrationForm
			}
			_, _ = io.WriteString(w, form)
			return
		}
		_, _ = io.WriteString(w, `<!doctype html><html><body><main><h1>Create account</h1><form method="post" action="/registration-complete"><label>Email<input name="identifier" autocomplete="email"></label><label>Password<input name="password" type="password"></label><button type="submit">Register</button><p role="status" aria-label="Registration complete">Registration proof marker</p></form></main></body></html>`)
	}))
	defer fixture.Close()
	temp := t.TempDir()
	binary := filepath.Join(temp, "browsertools")
	if supplied := os.Getenv("OPENUDON_TEST_BROWSERTOOLS_EXECUTABLE"); supplied != "" {
		// Development-only test seam for an explicitly prepared dependency.
		// Acceptance qualification continues to build its frozen source closure.
		binary = supplied
	} else {
		err = processgroup.Run(ctx, time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", binary, "./cmd/browsertools"}, Dir: filepath.Join(filepath.Dir(root), "browsertools"), Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard})
		if err != nil {
			t.Fatal("worker_build")
		}
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
		Verification:          isVerification,
		ApplicationExecutable: application, RepoRoot: root, BrowsertoolsExecutable: binary, ExampleDir: example, PrivateRoot: filepath.Join(temp, "private"), ScratchParent: filepath.Join(temp, "scratch"), StoreDir: filepath.Join(temp, "store"), Scope: "qualification/brp", ProfileID: "qualification_brp", InitialURL: fixture.URL + "/register?action=startnew", Origin: fixture.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mutations.Load() != 0 || result.Snapshot.Promotion == nil || !result.RetainedQuery {
		t.Fatal("ui_handoff")
	}
}
