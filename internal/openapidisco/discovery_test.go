package openapidisco

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFilesFindsAndScoresOpenAPI(t *testing.T) {
	base := t.TempDir()
	openAPIDir := filepath.Join(base, "openapi")
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(openAPIDir, "support.yaml"), []byte(`openapi: 3.0.0
info:
  title: Support Ticket API
  version: 1.0.0
  description: Fetch and update support tickets.
paths: {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(openAPIDir, "notes.yaml"), []byte(`not: openapi`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LocalFiles(openAPIDir, base, "When a support ticket is created, fetch ticket details.")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].RelativePath != "openapi/support.yaml" {
		t.Fatalf("relative path = %q", got[0].RelativePath)
	}
	if got[0].Title != "Support Ticket API" {
		t.Fatalf("title = %q", got[0].Title)
	}
	if got[0].Score == 0 {
		t.Fatalf("score should be positive")
	}
}

func TestSelectPrimarySortsByScore(t *testing.T) {
	got, err := SelectPrimary([]Candidate{
		{RelativePath: "openapi/low.yaml", Score: 1},
		{RelativePath: "openapi/high.yaml", Score: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RelativePath != "openapi/high.yaml" {
		t.Fatalf("primary = %q", got.RelativePath)
	}
}

func TestRedirectSafeClientRejectsPrivateRedirectTarget(t *testing.T) {
	client := (&Discoverer{}).redirectSafeClient()
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/openapi.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	via, err := http.NewRequest(http.MethodGet, "https://public.example.test/openapi.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.CheckRedirect(req, []*http.Request{via}); err == nil {
		t.Fatalf("expected private redirect target to be rejected")
	}
}

func TestImportBestAPIsGuruMatchRejectsPrivateListURLBeforeRequest(t *testing.T) {
	var called bool
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})}
	_, err := (&Discoverer{
		HTTPClient:      client,
		APIsGuruListURL: "http://127.0.0.1/list.json",
	}).ImportBestAPIsGuruMatch(context.Background(), t.TempDir(), t.TempDir(), "weather")
	if err == nil {
		t.Fatalf("expected private APIs.guru list URL to be rejected")
	}
	if called {
		t.Fatalf("HTTP client was called before private host rejection")
	}
}

func TestSafeDialContextRejectsPrivateAddress(t *testing.T) {
	if _, err := safeDialContext(context.Background(), "tcp", "127.0.0.1:80"); err == nil {
		t.Fatalf("expected private dial address to be rejected")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
