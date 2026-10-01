package stepauthoring

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/workflowintent"
)

func pendingRequestFromBind(request BindRequest) PendingRequest {
	return PendingRequest{Version: PendingWireVersion, Kind: "request", Command: PendingCommand, StepID: request.StepID, Contract: request.Contract, DependsOn: request.DependsOn, IntentRevision: request.IntentRevision, Scaffold: request.Scaffold}
}

func TestPendingAuthorResolveAndConfirmedEffect(t *testing.T) {
	fixture := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	bind := readBindRequest(t, fixture)
	example := copyBindExample(t, fixture, false)
	pending := pendingRequestFromBind(bind)
	out := Pending(context.Background(), example, pending)
	if out.ExitCode != 0 || out.Result.Version != PendingWireVersion || out.Result.Command != PendingCommand || out.Result.Result == nil {
		t.Fatalf("pending result=%+v", out)
	}
	intentBytes, err := os.ReadFile(filepath.Join(example, defaultIntentPath))
	if err != nil {
		t.Fatal(err)
	}
	intent, err := workflowintent.ParseIntent(intentBytes, defaultIntentPath)
	if err != nil {
		t.Fatal(err)
	}
	steps := findSteps(intent.Steps, bind.StepID)
	if len(steps) != 1 || steps[0].Pending == nil || steps[0].Operation != "" || steps[0].Pending.Effect != "read" {
		t.Fatalf("pending not preserved: %+v", steps)
	}
	got, _ := json.Marshal(steps[0].Pending)
	want, _ := json.Marshal(pendingContract(bind.Contract))
	if string(got) != string(want) {
		t.Fatal("pending field-set contract changed during HCL round trip")
	}
	bind.IntentRevision = IntentRevision{State: "present", SHA256: sourceDigest(intentBytes)}
	bind.Scaffold = nil
	changed := bind
	changed.Contract.Effect = "write"
	refused := Bind(context.Background(), example, changed)
	if refused.ExitCode != 3 || refused.Result.Diagnostics[0].Code != "contract.changed" {
		t.Fatalf("changed contract accepted: %+v", refused)
	}
	after, _ := os.ReadFile(filepath.Join(example, defaultIntentPath))
	if !bytesEqual(after, intentBytes) {
		t.Fatal("conflict changed intent")
	}
	resolved := Bind(context.Background(), example, bind)
	if resolved.ExitCode != 0 {
		t.Fatalf("resolve=%+v", resolved)
	}
	after, err = os.ReadFile(filepath.Join(example, defaultIntentPath))
	if err != nil {
		t.Fatal(err)
	}
	intent, err = workflowintent.ParseIntent(after, defaultIntentPath)
	if err != nil {
		t.Fatal(err)
	}
	steps = findSteps(intent.Steps, bind.StepID)
	if len(steps) != 1 || steps[0].Pending != nil || steps[0].Effect != "read" || steps[0].Operation != "listProjects" {
		t.Fatalf("resolution lost confirmed contract or retained pending: %+v", steps)
	}
	stale := Pending(context.Background(), example, pending)
	if stale.ExitCode != 3 {
		t.Fatalf("stale revision accepted: %+v", stale)
	}
}

func TestPendingRefusalsAreReadOnly(t *testing.T) {
	fixture := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	for _, kind := range []string{"invalid-schema", "mixed-fields", "wrong-version", "cancelled", "unsafe-intent", "legacy-package"} {
		t.Run(kind, func(t *testing.T) {
			example := copyBindExample(t, fixture, false)
			request := pendingRequestFromBind(readBindRequest(t, fixture))
			ctx := context.Background()
			switch kind {
			case "invalid-schema":
				request.Contract.Outputs.Type = "string"
			case "mixed-fields":
				request.Contract.Outputs.Required = append(request.Contract.Outputs.Required, "missing-property")
			case "wrong-version":
				request.Version = "openudon.step-pending.v99"
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "unsafe-intent":
				os.MkdirAll(filepath.Join(example, "workflows"), 0700)
				outside := filepath.Join(t.TempDir(), "outside")
				os.WriteFile(outside, []byte("private canary"), 0600)
				if err := os.Symlink(outside, filepath.Join(example, defaultIntentPath)); err != nil {
					t.Fatal(err)
				}
			case "legacy-package":
				os.MkdirAll(filepath.Join(example, "workflows"), 0700)
				os.WriteFile(filepath.Join(example, "workflows/workflow.hcl"), []byte(`uws = "1.11.0"`), 0600)
			}
			out := Pending(ctx, example, request)
			if out.ExitCode == 0 || strings.Contains(out.Result.Diagnostics[0].Message, "private canary") {
				t.Fatalf("unsafe request accepted or leaked: %+v", out)
			}
			if kind != "unsafe-intent" {
				if _, err := os.Lstat(filepath.Join(example, defaultIntentPath)); !os.IsNotExist(err) {
					t.Fatalf("refusal created intent: %v", err)
				}
			}
		})
	}
}
