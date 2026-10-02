//go:build m95_capture_smoke || browser_system_qualification

package capturequalification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestNeutralRegistrationPublicCapturePackage(t *testing.T) {
	exe := qualificationApplication(t)
	var gets, heads, posts atomic.Int64
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			gets.Add(1)
		case "HEAD":
			heads.Add(1)
		default:
			posts.Add(1)
			w.WriteHeader(405)
			return
		}
		if r.URL.Path != "/register" || r.URL.Query().Get("action") != "startnew" {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == "GET" {
			_, _ = w.Write([]byte(SyntheticVerificationRegistrationForm))
		}
	}))
	defer fixture.Close()
	base, err := os.MkdirTemp("", "openudon-m95-development-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if t.Failed() && os.Getenv("OPENUDON_M95_SMOKE_KEEP_FAILED") == "1" {
			t.Logf("failed private fixture retained: %s", base)
			return
		}
		_ = os.RemoveAll(base)
	}()
	mkdir := func(name string) string {
		p := filepath.Join(base, name)
		if os.Mkdir(p, 0700) != nil {
			t.Fatal("private directory")
		}
		return p
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	env := []string{}
	for _, name := range []string{"HOME", "PATH", "DISPLAY", "XAUTHORITY", "CHROME_DEVEL_SANDBOX", "PLAYWRIGHT_BROWSERS_PATH", "LANG", "LC_ALL"} {
		if value := os.Getenv(name); value != "" {
			env = append(env, name+"="+value)
		}
	}
	r, err := RunRegistration(ctx, RegistrationOptions{Executable: exe, RepoRoot: os.Getenv("OPENUDON_M95_REPO_ROOT"), ExampleDir: mkdir("example"), PrivateRoot: mkdir("private"), ScratchParent: mkdir("scratch"), StoreDir: mkdir("store"), Scope: "qualification/brp", ProfileID: "qualification_brp", InitialURL: fixture.URL + "/register?action=startnew", Origin: fixture.URL, Environment: env, Progress: os.Stdout})
	if err != nil {
		t.Fatal(err)
	}
	if !r.RetainedQuery || !r.VerificationRefused || !r.VerificationGranted || r.Selection.SelectionSHA256 == "" || r.Prepared.ManifestSHA256 == "" || r.Qualified.QualificationSHA256 == "" || gets.Load() == 0 || heads.Load() == 0 || posts.Load() != 0 {
		t.Fatal("retained native qualification missing or authoring exceeded GET/HEAD authority")
	}
	t.Log("public native registration, typed preview/history, verification refusal/grant, import, separate plan/refusal/author, package preparation/qualification/promotion and no submit passed")
}
