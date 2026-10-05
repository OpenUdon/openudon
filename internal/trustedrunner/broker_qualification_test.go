package trustedrunner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/OpenUdon/openudon/internal/brokerhandoff"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/udonreport"
)

var m46BrokerQualification = reportQualification{"M46", "95c5850fd446e06ac6f79d943774db67e417c989", "53bb9e8976f67c6a5880f07248a99cd195e6285ce7a5a68239793dd5dc6eb429", "a286f21a5dcd4180b7ec19bf0630188ddacec26e0532198db14decd0c86b2069", 5}

// TestPublishedM46BrokerQualification is explicitly invoked contract evidence,
// never a default external-runtime test. It uses the accepted executor bytes,
// frozen fourteen-source closure and synthetic Unix broker only.
func TestPublishedM46BrokerQualification(t *testing.T) {
	executor := qualifiedReportExecutor(t, m46BrokerQualification)
	for _, mode := range []string{"success", "lost-write", "wrong-identity", "unknown-write", "refused-read", "failed-write", "cancel-write"} {
		t.Run(mode, func(t *testing.T) {
			opts, approval, privatePath, listener := fixtureBroker(t)
			approval.Broker.ExecutorSHA256 = m46BrokerQualification.binary
			approval.Broker.PolicySHA256 = approval.Broker.Digest()
			writeApprovalFile(t, opts.ApprovalPath, approval)
			data, err := os.ReadFile(privatePath)
			if err != nil {
				t.Fatal(err)
			}
			var private brokerhandoff.PrivateConfig
			if err := json.Unmarshal(data, &private); err != nil {
				t.Fatal(err)
			}
			private.PolicyDigest = approval.Broker.PolicySHA256
			data, _ = json.Marshal(private)
			if err := os.WriteFile(privatePath, data, 0600); err != nil {
				t.Fatal(err)
			}
			opts.Env = []string{"OPENUDON_EXECUTOR=" + executor, "UDON_CREDENTIAL_TOKEN=SECRET_CANARY", "HTTP_PROXY=http://proxy.invalid"}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var reads, writes atomic.Int32
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/v1/dispatch" || r.Header.Get("Authorization") != "Bearer "+private.Capability {
					t.Error("private RPC posture changed")
					w.WriteHeader(403)
					return
				}
				var request struct {
					Version  string `json:"version"`
					Identity struct {
						RunID        string `json:"run_id"`
						StepID       string `json:"step_id"`
						OperationID  string `json:"operation_id"`
						InvocationID string `json:"invocation_id"`
						PolicyDigest string `json:"policy_digest"`
						RequestID    string `json:"request_id"`
					} `json:"identity"`
					Method   string                  `json:"method"`
					URL      string                  `json:"url"`
					Headers  http.Header             `json:"headers"`
					Body     []byte                  `json:"body,omitempty"`
					Bindings []brokerhandoff.Binding `json:"bindings,omitempty"`
				}
				if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&request); err != nil {
					t.Error("invalid RPC")
					w.WriteHeader(400)
					return
				}
				if request.Version != brokerhandoff.TransportVersion || request.Identity.RunID != approval.Broker.RunID || request.Identity.PolicyDigest != private.PolicyDigest || request.Identity.RequestID == "" || request.URL != "https://service.test/value" || request.Headers.Get("Authorization") != "" || len(request.Bindings) != 1 || request.Bindings[0].Name != "token" {
					t.Error("public transport binding drift")
					w.WriteHeader(400)
					return
				}
				outcome, code, status := "succeeded", "", 200
				if request.Method == "GET" {
					reads.Add(1)
				} else if request.Method == "POST" {
					// Read the exact worker checkpoint before treating this as a
					// synthetic write. It must already show durable uncertainty.
					pattern := filepath.Join(opts.WorkDir, "run-"+approval.Broker.RunID, "stage.*", "executor-report-"+approval.Broker.RunID+".json")
					paths, _ := filepath.Glob(pattern)
					if len(paths) != 1 {
						t.Errorf("missing pre-dispatch report: %d", len(paths))
						w.WriteHeader(500)
						return
					}
					checkpoint, _, err := evidencefile.ReadRegular(paths[0], 256<<10)
					if err != nil {
						t.Error(err)
						w.WriteHeader(500)
						return
					}
					var report udonreport.ReportV5
					if json.Unmarshal(checkpoint, &report) != nil || report.Steps[1].StartedAt == "" || report.Steps[1].Outcome != "unknown" {
						t.Error("write preceded durable start")
						w.WriteHeader(500)
						return
					}
					writes.Add(1)
					switch mode {
					case "lost-write":
						conn, _, err := w.(http.Hijacker).Hijack()
						if err == nil {
							_ = conn.Close()
						}
						return
					case "wrong-identity":
						request.Identity.RunID = "wrong-run"
					case "unknown-write":
						outcome, code, status = "unknown", "completion_lost", 0
					case "failed-write":
						outcome, code, status = "failed", "upstream_failed", 500
					case "cancel-write":
						cancel()
						<-r.Context().Done()
						return
					}
				} else {
					t.Error("unexpected HTTP method")
					w.WriteHeader(400)
					return
				}
				if mode == "refused-read" {
					outcome, code, status = "refused", "host_refused", 0
				}
				response := map[string]any{"version": brokerhandoff.TransportVersion, "identity": request.Identity, "outcome": outcome}
				if code != "" {
					response["code"] = code
				}
				if status != 0 {
					response["status_code"] = status
					response["headers"] = http.Header{"Content-Type": {"application/json"}}
					response["body"] = []byte(`{"ok":true,"value":"BODY_CANARY"}`)
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error("response delivery failed")
				}
			})}
			go func() { _ = server.Serve(listener) }()
			defer server.Close()
			opts.Stdout, opts.Stderr = io.Discard, io.Discard
			result, runErr := Run(ctx, opts)
			if mode == "success" && runErr != nil {
				t.Fatalf("real broker run failed: %v", runErr)
			}
			if mode != "success" && runErr == nil {
				t.Fatal("uncertain/refused/failed run claimed success")
			}
			if result == nil || result.RunEvidencePath == "" {
				t.Fatalf("missing durable real evidence: %v", runErr)
			}
			e := readRunEvidenceFile(t, result.RunEvidencePath)
			if e.Version != BrokerRunEvidenceVersion || e.Broker == nil || e.StepExecution == nil || e.StepExecution.State != "validated" {
				t.Fatal("actual report not bound to broker evidence")
			}
			if reads.Load() != 1 {
				t.Fatal("read repeated")
			}
			if mode == "refused-read" {
				if writes.Load() != 0 {
					t.Fatal("refused read allowed a write")
				}
			} else if writes.Load() != 1 {
				t.Fatal("write missing or repeated")
			}
			if mode == "lost-write" || mode == "wrong-identity" || mode == "unknown-write" || mode == "cancel-write" {
				if e.StepExecution.Steps[1].Outcome != "unknown" || e.StepExecution.Steps[1].StartedAt == "" {
					t.Fatal("possible write lost uncertainty")
				}
			}
			if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
				t.Fatal(err)
			}
			data, _ = os.ReadFile(result.RunEvidencePath)
			if strings.Contains(string(data), "SECRET_CANARY") || strings.Contains(string(data), "BODY_CANARY") || strings.Contains(string(data), private.Capability) {
				t.Fatal("private values entered evidence")
			}
			if _, err := Run(context.Background(), opts); err == nil {
				t.Fatal("completed/uncertain run was replayed")
			}
			if mode == "refused-read" {
				if writes.Load() != 0 {
					t.Fatal("replay sent write")
				}
			} else if writes.Load() != 1 {
				t.Fatal("replay sent another write")
			}
		})
	}
}
