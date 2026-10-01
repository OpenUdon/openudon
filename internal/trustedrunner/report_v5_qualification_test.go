package trustedrunner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

const acceptedM44 = "1a5e9aa2045e3d875da2e18aab2d6db869ac5223"
const acceptedM44Closure = "603a1c4dae5606dd24683ca40050c918e0b7e2b6f31c10e6d4399ce3c5d739c6"

const acceptedM44Executor = "d2d593ac6d5a6a19406180eb70c5993ee4811fecb20c1a31cdb5a1f59278575b"

type reportQualification struct {
	name, source, binary, closure string
	verificationCount             int
}

var m44ReportQualification = reportQualification{"M44", acceptedM44, acceptedM44Executor, acceptedM44Closure, 3}
var m45ReportQualification = reportQualification{"M45", "238f2e487d50ffec057b7a109a35c9db03f59c55", "cb4b94c968aa3f3de4106a440fdcd02e6c210941eb25e6666b84cfbc7f63868b", "10d4c613c4882365f2799789d456e8a3b15484b1995a052e334616cad9d0fd59", 4}

func qualifiedReportExecutor(t *testing.T, identity reportQualification) string {
	t.Helper()
	if os.Getenv("OPENUDON_"+identity.name+"_QUALIFY") != "1" {
		t.Skip("explicit loopback-only real executor qualification")
	}
	path := os.Getenv("OPENUDON_" + identity.name + "_EXECUTOR")
	if !filepath.IsAbs(path) {
		t.Fatal("set the absolute qualified executor path")
	}
	binary, _, err := evidencefile.ReadRegular(path, 512<<20)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(binary)
	if hex.EncodeToString(sum[:]) != identity.binary {
		t.Fatal("executor does not match accepted frozen build")
	}
	data, _, err := evidencefile.ReadRegular(os.Getenv("OPENUDON_"+identity.name+"_CLOSURE"), 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	if evidencefile.SHA256(data) != identity.closure {
		t.Fatal("closure bytes differ from accepted qualification")
	}
	var closure struct {
		SourceRevision   string            `json:"source_revision"`
		WorktreeIncluded bool              `json:"worktree_included"`
		Overlays         []json.RawMessage `json:"overlays"`
		ExecutorSHA256   string            `json:"executor_sha256"`
		Siblings         []json.RawMessage `json:"siblings"`
		Verification     []struct {
			ExitCode int `json:"exit_code"`
		} `json:"verification"`
	}
	if err := json.Unmarshal(data, &closure); err != nil {
		t.Fatal(err)
	}
	if closure.SourceRevision != identity.source || closure.WorktreeIncluded || len(closure.Overlays) != 0 || closure.ExecutorSHA256 != identity.binary || len(closure.Siblings) != 14 || len(closure.Verification) != identity.verificationCount {
		t.Fatal("not accepted clean build closure")
	}
	for _, v := range closure.Verification {
		if v.ExitCode != 0 {
			t.Fatal("unqualified closure")
		}
	}
	// Execute a private copy of the exact verified binary, retaining source provenance.
	snapshot := filepath.Join(t.TempDir(), "udon")
	if err := os.WriteFile(snapshot, binary, 0700); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestPublishedM44Qualification(t *testing.T) {
	qualifyReportV5(t, qualifiedReportExecutor(t, m44ReportQualification), m44ReportQualification, "1.11.0")
}

func TestPublishedM45Qualification(t *testing.T) {
	executor := qualifiedReportExecutor(t, m45ReportQualification)
	for _, version := range []string{"1.11.0", "1.12.0"} {
		t.Run(version, func(t *testing.T) { qualifyReportV5(t, executor, m45ReportQualification, version) })
	}
}

func qualifyReportV5(t *testing.T, executor string, identity reportQualification, version string) {
	for _, mode := range []string{"success", "failed-read", "failed-write", "kill-write", "checkpoint", "missing", "stale", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			var reads, writes atomic.Int32
			var current atomic.Pointer[udonrunner.Invocation]
			var cancel context.CancelFunc
			var interrupted atomic.Bool
			release := make(chan struct{})
			service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/read":
					reads.Add(1)
					if mode == "checkpoint" {
						dir := filepath.Dir(argValue(t, current.Load().Argv, "--execution-report"))
						if err := os.Rename(dir, dir+"-retained"); err != nil {
							t.Error(err)
						}
						if err := os.WriteFile(dir, []byte("blocked checkpoint"), 0600); err != nil {
							t.Error(err)
						}
						interrupted.Store(true)
					}
					if mode == "failed-read" {
						w.WriteHeader(500)
					}
				case "/write":
					writes.Add(1)
					if mode == "failed-write" {
						w.WriteHeader(500)
					}
					if mode == "kill-write" {
						cancel()
						<-release
					}
				default:
					t.Error("unexpected fixture request")
					w.WriteHeader(404)
				}
				w.Header().Set("X-Canary", "HEADER_CANARY")
				_, _ = io.WriteString(w, `{"value":"BODY_CANARY"}`)
			}))
			defer service.Close()
			root := t.TempDir()
			example := filepath.Join(root, "examples", "report")
			mustWriteFile(t, filepath.Join(example, "project.md"), []byte(`# Report fixture

## Goal
Read /read then write /write on the reviewed disposable loopback service.

## Runtime Policy
- openapi and http are allowed.
- fnct, cmd and ssh are not allowed.

## Safety and Approval Boundary
Read is read-only. Write is side-effectful and requires exact sandbox approval.
No credentials or production endpoints are used.
`))
			mustWriteFile(t, filepath.Join(example, "openapi/service.yaml"), []byte(`openapi: 3.0.0
info: {title: Report fixture, version: 1.0.0}
servers:
  - url: `+service.URL+`
paths:
  /read:
    get:
      operationId: get
      responses:
        '200': {description: ok}
  /write:
    post:
      operationId: post
      responses:
        '200': {description: ok}
`))
			mustWriteFile(t, filepath.Join(example, "workflows/intent.hcl"), []byte(`openapi = "openapi/service.yaml"
workflow {
 name = "fixture"
 description = "Read then write on the disposable loopback fixture."
}
step "read" {
 type = "http"
 do = "Read the fixture."
 operation = "get"
}
step "write" {
 type = "http"
 do = "Write the fixture."
 operation = "post"
}
`))
			// The legacy branch reauthors an already declared package; the
			// fresh branch must adopt 1.12 without selecting a version override.
			if version != "1.12.0" {
				mustWriteFile(t, filepath.Join(example, "workflows/workflow.hcl"), []byte("uws = \""+version+"\"\n"))
			}
			if _, err := synthesize.Build(context.Background(), synthesize.Options{ExampleDir: example}); err != nil {
				t.Fatal(err)
			}
			approved, err := ApprovalTemplate(context.Background(), TemplateOptions{RepoRoot: root, ExampleDir: example, State: StateApprovedForSandbox, Reviewer: identity.name + " loopback qualification"})
			if err != nil {
				t.Fatal(err)
			}
			approvalPath := filepath.Join(root, "approval.json")
			f, err := os.Create(approvalPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := WriteApproval(f, approved); err != nil {
				t.Fatal(err)
			}
			f.Close()
			runCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
			defer stop()
			runCtx, cancel = context.WithCancel(runCtx)
			defer cancel()
			result, err := Run(runCtx, Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath, ExecutorReportVersion: "v5", Env: []string{"OPENUDON_EXECUTOR=" + executor},
				Invoke: func(ctx context.Context, call udonrunner.Invocation) error {
					current.Store(&call)
					cmd := exec.CommandContext(ctx, call.Argv[0], call.Argv[1:]...)
					cmd.Dir = call.Dir
					cmd.Env = call.Env
					cmd.Stdout = io.Discard
					cmd.Stderr = io.Discard
					err := cmd.Run()
					if mode == "kill-write" {
						close(release)
					}
					reportPath := argValue(t, call.Argv, "--execution-report")
					if mode == "checkpoint" && interrupted.Load() {
						dir := filepath.Dir(reportPath)
						if err := os.Remove(dir); err != nil {
							t.Fatal(err)
						}
						if err := os.Rename(dir+"-retained", dir); err != nil {
							t.Fatal(err)
						}
					}
					if mode == "missing" {
						if err := os.Remove(reportPath); err != nil {
							t.Fatal(err)
						}
						return errors.New("report lost")
					}
					if mode == "stale" {
						data, err := os.ReadFile(reportPath)
						if err != nil {
							t.Fatal(err)
						}
						var r udonreport.ReportV5
						json.Unmarshal(data, &r)
						r.RunID = "stale-attempt"
						data, _ = json.Marshal(r)
						mustWriteFile(t, reportPath, data)
						return errors.New("stale report")
					}
					if mode == "duplicate" {
						second := exec.CommandContext(ctx, call.Argv[0], call.Argv[1:]...)
						second.Dir = call.Dir
						second.Env = call.Env
						second.Stdout = io.Discard
						second.Stderr = io.Discard
						if second.Run() == nil {
							t.Fatal("duplicate existing report allowed execution")
						}
					}
					return err
				}})
			if result == nil || result.RunEvidencePath == "" {
				t.Fatalf("missing evidence: %v", err)
			}
			expectSuccess := mode == "success" || mode == "duplicate"
			if (err == nil) != expectSuccess {
				t.Fatalf("wrong execution outcome: %v", err)
			}
			e := readRunEvidenceFile(t, result.RunEvidencePath)
			if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
				t.Fatal(err)
			}
			if reads.Load() != 1 {
				t.Fatalf("reads=%d", reads.Load())
			}
			wantWrites := int32(1)
			if mode == "failed-read" || mode == "checkpoint" {
				wantWrites = 0
			}
			if writes.Load() != wantWrites {
				t.Fatalf("writes=%d, want %d", writes.Load(), wantWrites)
			}
			o := e.StepExecution
			if mode == "missing" || mode == "stale" {
				for _, s := range o.Steps {
					if s.Outcome != "unknown" {
						t.Fatal("missing/stale proves outcome")
					}
				}
			} else {
				if o.State != "validated" {
					t.Fatalf("observation %+v", o)
				}
				want := map[string][]string{"success": {"succeeded", "succeeded"}, "duplicate": {"succeeded", "succeeded"}, "failed-read": {"failed", "not_started"}, "failed-write": {"succeeded", "failed"}, "kill-write": {"succeeded", "unknown"}, "checkpoint": {"unknown", "not_started"}}[mode]
				for n, s := range o.Steps {
					if s.Outcome != want[n] {
						t.Fatalf("step %d=%s want=%s", n, s.Outcome, want[n])
					}
				}
			}
			data, err := os.ReadFile(result.RunEvidencePath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "CANARY") {
				t.Fatal("payload leaked")
			}
			t.Logf("%s source=%s executor=%s UWS=%s mode=%s observation=%s reads=%d writes=%d", identity.name, identity.source, identity.binary, version, mode, o.State, reads.Load(), writes.Load())
		})
	}
}
