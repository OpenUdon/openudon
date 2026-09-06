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

func TestBrowserSystemSupervisedAuthenticatedPackage(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal("source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var posts, visits atomic.Int64
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/login":
			if r.Method == "POST" {
				posts.Add(1)
				if r.ParseForm() != nil || r.Form.Get("identifier") != "fixture-user" || r.Form.Get("password") != "fixture-password" {
					w.WriteHeader(403)
					return
				}
				http.SetCookie(w, &http.Cookie{Name: "fixture_session", Value: "authenticated", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
				http.Redirect(w, r, "/campaigns", 303)
				return
			}
			_, _ = io.WriteString(w, `<!doctype html><html><body><h1>Sign in</h1><form method="post" action="/login"><label>Email address<input name="identifier" autocomplete="username" onfocus="this.value='fixture-user'"></label><label>Password<input name="password" type="password" autocomplete="current-password" onfocus="this.value='fixture-password'"></label><button type="submit">Sign in</button></form></body></html>`)
		case "/campaigns":
			c, err := r.Cookie("fixture_session")
			if err != nil || c.Value != "authenticated" {
				w.WriteHeader(401)
				return
			}
			visits.Add(1)
			_, _ = io.WriteString(w, `<!doctype html><html><body><h1>Campaigns</h1><p role="status" aria-label="Campaign list">Available</p></body></html>`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer fixture.Close()
	temp := t.TempDir()
	binary := filepath.Join(temp, "icot")
	if err := processgroup.Run(ctx, time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", binary, "./cmd/icot"}, Dir: root, Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard}); err != nil {
		t.Fatal("application_build")
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
	example, err := os.MkdirTemp(parent, ".application-auth-")
	if err != nil {
		t.Fatal("workspace")
	}
	defer os.RemoveAll(example)
	result, err := RunAuthenticatedApplicationQualification(ctx, RegistrationQualificationOptions{RepoRoot: root, ApplicationExecutable: binary, ExampleDir: example, PrivateRoot: filepath.Join(temp, "private"), ScratchParent: filepath.Join(temp, "scratch"), StoreDir: filepath.Join(temp, "store"), Scope: "qualification/authenticated", ProfileID: "qualification_auth", Origin: fixture.URL, InitialURL: fixture.URL + "/login"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Promotion == nil || result.Transaction == nil || result.Transaction.Session == "" || posts.Load() != 1 || visits.Load() < 1 {
		t.Fatal("authenticated_package_evidence")
	}
}
