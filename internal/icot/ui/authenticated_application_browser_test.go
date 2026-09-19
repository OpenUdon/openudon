//go:build browser_system_qualification

package ui

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/processgroup"
)

type authPolicyCountingListener struct {
	net.Listener
	arrivals atomic.Int64
}

func (l *authPolicyCountingListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		l.arrivals.Add(1)
	}
	return c, err
}

func TestBrowserSystemSupervisedAuthenticatedPackage(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal("source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	blocked := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("blocked endpoint reached") }))
	denied := &authPolicyCountingListener{Listener: blocked.Listener}
	blocked.Listener = denied
	blocked.StartTLS()
	defer blocked.Close()
	var posts, visits, styles atomic.Int64
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/shared.css":
			styles.Add(1)
			w.Header().Set("Content-Type", "text/css")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			_, _ = io.WriteString(w, "body { color: black; }")
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
			fmt.Fprintf(w, `<script src="%s/optional.js"></script>`, blocked.URL)
			_, _ = io.WriteString(w, `<link rel="stylesheet" href="/shared.css">`)
			_, _ = io.WriteString(w, `<!doctype html><html><body><h1>Sign in</h1><form method="post" action="/login"><label>Email address<input name="identifier" autocomplete="username" onfocus="this.value='fixture-user'"></label><label>Password<input name="password" type="password" autocomplete="current-password" onfocus="this.value='fixture-password'"></label><button type="submit">Sign in</button></form></body></html>`)
		case "/campaigns":
			c, err := r.Cookie("fixture_session")
			if err != nil || c.Value != "authenticated" {
				w.WriteHeader(401)
				return
			}
			visits.Add(1)
			_, _ = io.WriteString(w, `<link rel="stylesheet" href="/shared.css">`)
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
	result, err := RunAuthenticatedApplicationQualification(ctx, RegistrationQualificationOptions{RepoRoot: root, ApplicationExecutable: binary, ExampleDir: example, PrivateRoot: filepath.Join(temp, "private"), ScratchParent: filepath.Join(temp, "scratch"), StoreDir: filepath.Join(temp, "store"), Scope: "qualification/authenticated", ProfileID: "qualification_auth", Origin: fixture.URL, InitialURL: fixture.URL + "/login"}, blocked.URL)
	if err != nil {
		t.Fatal(err)
	}
	if denied.arrivals.Load() != 0 {
		t.Fatal("blocked origin received contact")
	}
	if styles.Load() != 2 {
		t.Fatal("authoring cache isolation was not exercised before and after login")
	}
	if result.Promotion == nil || result.Transaction == nil || result.Transaction.Session == "" || posts.Load() != 1 || visits.Load() < 1 {
		t.Fatal("authenticated_package_evidence")
	}
}
