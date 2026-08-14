package elicitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDiscoverRemoteSourceHintsRequiresApproval(t *testing.T) {
	for _, tc := range []struct {
		policy string
		code   string
	}{
		{policy: "never", code: "network.denied"},
		{policy: "ask", code: "network.approval_required"},
	} {
		t.Run(tc.policy, func(t *testing.T) {
			report, err := DiscoverRemoteSourceHints(context.Background(), "zebracorp workflow", RemoteSourceLookupOptions{Policy: tc.policy})
			if err != nil {
				t.Fatal(err)
			}
			if report.Blocker == nil || report.Blocker.Code != tc.code || !report.Blocker.Deferrable || len(report.Candidates) != 0 {
				t.Fatalf("report = %#v", report)
			}
		})
	}
}

func TestDiscoverRemoteSourceHintsUsesOneBoundedAPIsGuruLookup(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"zebracorp.example": map[string]any{
				"preferred": "1.0.0",
				"versions": map[string]any{"1.0.0": map[string]any{
					"swaggerUrl": "https://api.zebracorp.example/openapi.json",
					"info":       map[string]any{"title": "ZebraCorp Workflow API"},
				}},
			},
		})
	}))
	defer server.Close()

	report, err := DiscoverRemoteSourceHints(context.Background(), "zebracorp workflow", RemoteSourceLookupOptions{
		Policy: "ask", Approved: true, APIsGuruListURL: server.URL, AllowUnsafeHosts: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || len(report.Candidates) != 1 || report.Candidates[0].Provenance != "apis.guru" || len(report.Candidates) > remoteLookupLimit || report.Blocker != nil {
		t.Fatalf("requests = %d, report = %#v", requests.Load(), report)
	}
}

func TestDiscoverRemoteSourceHintsReportsEmptyAndTimeout(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()
		report, err := DiscoverRemoteSourceHints(context.Background(), "zebracorp workflow", RemoteSourceLookupOptions{
			Policy: "allow", APIsGuruListURL: server.URL, AllowUnsafeHosts: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if report.Blocker == nil || report.Blocker.Code != "network.empty" {
			t.Fatalf("report = %#v", report)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer server.Close()
		report, err := DiscoverRemoteSourceHints(context.Background(), "zebracorp workflow", RemoteSourceLookupOptions{
			Policy: "allow", APIsGuruListURL: server.URL, AllowUnsafeHosts: true, Deadline: 20 * time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		if report.Blocker == nil || report.Blocker.Code != "network.timeout" {
			t.Fatalf("report = %#v", report)
		}
	})
}

func TestSafeRemoteSourceURLRejectsUnsafeHosts(t *testing.T) {
	for _, raw := range []string{
		"http://example.com/openapi.json",
		"https://localhost/openapi.json",
		"https://127.0.0.1/openapi.json",
		"https://10.0.0.1/openapi.json",
	} {
		if err := safeRemoteSourceURL(raw, false); err == nil {
			t.Fatalf("safeRemoteSourceURL(%q) succeeded", raw)
		}
	}
	if err := safeRemoteSourceURL("https://api.example.com/openapi.json", false); err != nil {
		t.Fatal(err)
	}
	if err := safeRemoteSourceURL("http://127.0.0.1/openapi.json", true); err != nil {
		t.Fatal(err)
	}
}
