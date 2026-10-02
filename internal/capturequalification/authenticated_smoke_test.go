//go:build m95_capture_smoke || browser_system_qualification

package capturequalification

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

	"github.com/OpenUdon/openudon/internal/browsercheck"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

func qualificationEnvironment() []string {
	var env []string
	for _, name := range []string{"HOME", "PATH", "DISPLAY", "XAUTHORITY", "CHROME_DEVEL_SANDBOX", "PLAYWRIGHT_BROWSERS_PATH", "LANG", "LC_ALL"} {
		if value := os.Getenv(name); value != "" {
			env = append(env, name+"="+value)
		}
	}
	return append(env, "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOENV=off", "GOTOOLCHAIN=local")
}
func qualificationApplication(t *testing.T) string {
	t.Helper()
	if exe := os.Getenv("OPENUDON_M95_CAPTURE_EXE"); exe != "" {
		return exe
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal("source root unavailable")
	}
	exe := filepath.Join(t.TempDir(), "openudon")
	if err := browsercheck.Build(context.Background(), "openudon", exe, func(output string) error {
		return processgroup.Run(context.Background(), time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", output, "./cmd/openudon"}, Dir: root, Env: qualificationEnvironment(), Stdout: io.Discard, Stderr: io.Discard})
	}); err != nil {
		t.Fatal("public application build failed")
	}
	return exe
}
func TestNeutralAuthenticatedPublicCapturePackage(t *testing.T) {
	exe := qualificationApplication(t)
	var login, totp, unexpected atomic.Int64
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			switch r.URL.Path {
			case "/login":
				login.Add(1)
				w.Header().Set("Location", "/totp")
			case "/totp":
				totp.Add(1)
				w.Header().Set("Location", "/dashboard")
			default:
				unexpected.Add(1)
				w.WriteHeader(405)
				return
			}
			// No credential value is required, read, saved or reported by this synthetic service.
			w.WriteHeader(303)
			return
		}
		if r.Method != "GET" {
			unexpected.Add(1)
			w.WriteHeader(405)
			return
		}
		pages := map[string]string{
			"/login":     `<h1>Disposable sign in</h1><form method="post" action="/login"><label>Email address<input name="identifier" type="email" autocomplete="username" aria-label="Email address"></label><label>Password<input name="password" type="password" autocomplete="current-password" aria-label="Password"></label><button type="submit">Sign in</button></form>`,
			"/totp":      `<h1>Disposable verification</h1><form method="post" action="/totp"><label>Verification code<input name="challenge" type="text" inputmode="numeric" autocomplete="one-time-code" aria-label="Verification code"></label><button type="submit">Verify</button></form>`,
			"/dashboard": `<h1>Dashboard</h1><a href="/status">Review status</a>`,
			"/status":    `<h1>Status</h1><p role="status" aria-label="Account status">Synthetic active account</p>`,
		}
		page, ok := pages[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<!doctype html><html><body>"+page+"</body></html>")
	}))
	defer fixture.Close()
	base := t.TempDir()
	mkdir := func(name string) string {
		p := filepath.Join(base, name)
		if os.Mkdir(p, 0700) != nil {
			t.Fatal("private directory")
		}
		return p
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal("source root")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	r, err := RunAuthenticated(ctx, RegistrationOptions{Executable: exe, RepoRoot: root, ExampleDir: mkdir("example"), PrivateRoot: mkdir("private"), ScratchParent: mkdir("scratch"), StoreDir: mkdir("store"), Scope: "qualification/authenticated", ProfileID: "qualification_auth", InitialURL: fixture.URL + "/login", Origin: fixture.URL, Environment: qualificationEnvironment()})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.HumanKinds) != 3 || r.Selection.SelectionSHA256 == "" || r.Prepared.ManifestSHA256 == "" || r.Qualified.QualificationSHA256 == "" || login.Load() != 1 || totp.Load() != 1 || unexpected.Load() != 0 {
		t.Fatal("native login/TOTP/package evidence or exact POST authority missing")
	}
	t.Log("actual public login/TOTP/goal capture, independent profile import, separately confirmed authoring/build and package selection passed; no real account/model/runtime")
}
