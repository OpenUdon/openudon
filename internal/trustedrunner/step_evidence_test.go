package trustedrunner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

const v5Workflow = `uws: 1.11.0
info: {title: Report fixture, version: 1.0.0}
sourceDescriptions:
  - {name: api, type: openapi, url: openapi/service.yaml}
operations:
  - {operationId: get, sourceDescription: api, sourceOperationId: get}
  - {operationId: post, sourceDescription: api, sourceOperationId: post}
workflows:
  - workflowId: fixture
    type: sequence
    steps:
      - {stepId: read, operationRef: get}
      - {stepId: write, operationRef: post, dependsOn: [read]}
`

func fixtureV5(t *testing.T) (string, string, string) {
	t.Helper()
	root, example := writeFixture(t, fixtureOptions{})
	mustWriteFile(t, filepath.Join(example, "workflows/workflow.uws.yaml"), []byte(v5Workflow))
	refreshFixtureHandoffFile(t, example)
	approval := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, fixedNow())
	return root, example, approval
}

func TestV5TrustedRunConservativeEvidenceAndArchive(t *testing.T) {
	for _, name := range []string{"dry-run", "success", "failed-read", "failed-write", "interrupted", "missing", "stale", "mismatched", "incomplete-inventory", "malformed"} {
		t.Run(name, func(t *testing.T) {
			root, example, approval := fixtureV5(t)
			binary := filepath.Join(root, "executor")
			mustWriteFile(t, binary, []byte("#!/bin/sh\nexit 0\n"))
			if err := os.Chmod(binary, 0700); err != nil {
				t.Fatal(err)
			}
			invoked := false
			result, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approval, WorkDir: filepath.Join(root, "runs"), ExecutorReportVersion: "v5", DryRun: name == "dry-run", Env: []string{"OPENUDON_EXECUTOR=" + binary}, Now: fixedNow(), Assess: passAssess,
				Invoke: func(_ context.Context, call udonrunner.Invocation) error {
					invoked = true
					runID := argValue(t, call.Argv, "--execution-run-id")
					if argValue(t, call.Argv, "--execution-report-version") != "v5" {
						t.Fatal("not explicit v5")
					}
					if name == "missing" {
						return errors.New("executor stopped")
					}
					if name == "malformed" {
						mustWriteFile(t, argValue(t, call.Argv, "--execution-report"), []byte(`{"secret":"BODY_CANARY"}`))
						return errors.New("executor stopped")
					}
					fixtureName := name
					if name == "mismatched" || name == "incomplete-inventory" {
						fixtureName = "success"
					}
					data, err := os.ReadFile("../../docs/fixtures/per-step-run-evidence-v3/" + fixtureName + ".report.json")
					if err != nil {
						t.Fatal(err)
					}
					var report udonreport.ReportV5
					if err := json.Unmarshal(data, &report); err != nil {
						t.Fatal(err)
					}
					workflow, err := os.ReadFile(argValue(t, call.Argv, "--workflow"))
					if err != nil {
						t.Fatal(err)
					}
					expected, err := udonreport.InventoryFromWorkflowV5(workflow, "uws-yaml", runID)
					if err != nil {
						t.Fatal(err)
					}
					report.RunID = runID
					report.WorkflowDigest = expected.WorkflowDigest
					if name == "stale" {
						report.RunID = "stale-attempt"
					}
					if name == "mismatched" {
						report.WorkflowDigest = "sha256:" + strings.Repeat("f", 64)
					}
					if name == "incomplete-inventory" {
						report.InventoryComplete = false
					}
					data, _ = json.Marshal(report)
					mustWriteFile(t, argValue(t, call.Argv, "--execution-report"), data)
					if name != "success" {
						return errors.New("executor stopped")
					}
					return nil
				}})
			if name == "success" || name == "dry-run" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("executor failure lost")
			}
			if result == nil || result.RunEvidencePath == "" {
				t.Fatalf("missing conservative evidence: %v", err)
			}
			if invoked == (name == "dry-run") {
				t.Fatal("wrong invocation posture")
			}
			evidence := readRunEvidenceFile(t, result.RunEvidencePath)
			if evidence.Version != RunEvidenceVersionV3 || evidence.StepExecution == nil {
				t.Fatal("missing v3 observation")
			}
			if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
				t.Fatal(err)
			}
			expectedState := "validated"
			switch name {
			case "dry-run":
				expectedState = "dry_run"
			case "missing":
				expectedState = "missing"
			case "stale", "mismatched":
				expectedState = "mismatched"
			case "malformed", "incomplete-inventory":
				expectedState = "invalid"
			}
			if evidence.StepExecution.State != expectedState {
				t.Fatalf("state %s", evidence.StepExecution.State)
			}
			privateKey, publicKey := filepath.Join(root, "signing.pem"), filepath.Join(root, "public.pem")
			if err := GenerateSigningKey(privateKey, publicKey); err != nil {
				t.Fatal(err)
			}
			if _, err := SignRunEvidenceFile(result.RunEvidencePath, privateKey); err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyRunEvidenceFileWithOptions(result.RunEvidencePath, VerifyRunEvidenceOptions{RequireSignature: true, TrustedPublicKey: publicKey}); err != nil {
				t.Fatal(err)
			}
			archive, err := ArchiveRunEvidence(ArchiveOptions{RunEvidencePath: result.RunEvidencePath, ArchiveDir: filepath.Join(root, "archive")})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := VerifyRunEvidenceFile(archive.RunEvidencePath); err != nil {
				t.Fatal(err)
			}
			bytes, err := os.ReadFile(archive.RunEvidencePath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(bytes), "BODY_CANARY") {
				t.Fatal("payload retained")
			}
			if expectedState != "validated" && archive.ExecutorReport != "" {
				t.Fatal("untrusted report archived")
			}
			if expectedState == "validated" {
				forged := *evidence.StepExecution
				forged.RunID = "another-attempt"
				evidence.StepExecution = &forged
				if err := validateRunEvidenceForVerify(evidence); err == nil {
					t.Fatal("stale observation accepted")
				}
				evidence.StepExecution.RunID = evidence.RunID
				evidence.StepExecution.Steps = append([]udonreport.StepV5(nil), evidence.StepExecution.Steps...)
				evidence.StepExecution.Steps[1].OperationID = "different-operation"
				if err := verifyStepExecutionV3(filepath.Dir(result.RunEvidencePath), evidence, false); err == nil {
					t.Fatal("changed inventory accepted")
				}
			}
			// Even after archiving, an observation cannot be changed into unstarted proof.
			evidence.StepExecution.Steps[1].Outcome = "not_started"
			if expectedState != "validated" {
				if err := validateRunEvidenceForVerify(evidence); err == nil {
					t.Fatal("uncertainty rewritten")
				}
			}
		})
	}
}

func TestV5RefusesUnsupportedShapeBeforeInvocation(t *testing.T) {
	root, example, approval := fixtureV5(t)
	data := strings.Replace(v5Workflow, "type: sequence", "type: parallel", 1)
	mustWriteFile(t, filepath.Join(example, "workflows/workflow.uws.yaml"), []byte(data))
	refreshFixtureHandoffFile(t, example)
	approval = writeApprovalTemplate(t, root, example, StateApprovedForSandbox, fixedNow())
	_, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approval, ExecutorReportVersion: "v5", Assess: passAssess, Now: fixedNow(),
		Invoke: func(context.Context, udonrunner.Invocation) error { t.Fatal("unsupported shape invoked"); return nil }})
	if err == nil {
		t.Fatal("unsupported shape accepted")
	}
}

func TestV5ExternalCanonicalHandoffPreservesInterruptedReport(t *testing.T) {
	for _, fixture := range []string{"success", "interrupted"} {
		t.Run(fixture, func(t *testing.T) {
			root, example, approval := fixtureV5(t)
			binary := filepath.Join(root, "executor")
			mustWriteFile(t, binary, []byte("#!/bin/sh\nexit 0\n"))
			if err := os.Chmod(binary, 0700); err != nil {
				t.Fatal(err)
			}
			result, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approval, ExecutorReportVersion: "v5", Env: []string{"OPENUDON_EXECUTOR=" + binary}, RunnerPath: binary, Assess: passAssess, Now: fixedNow(),
				Invoke: func(ctx context.Context, outer udonrunner.Invocation) error {
					_, err := RunExternal(ctx, ExternalOptions{ConfigPath: argValue(t, outer.Argv, "--config"), ConfigSHA256: argValue(t, outer.Argv, "--config-sha256"), ApprovalPath: approval, Env: []string{"OPENUDON_EXECUTOR=" + binary}, Assess: passAssess, Now: fixedNow(),
						Invoke: func(_ context.Context, inner udonrunner.Invocation) error {
							data, err := os.ReadFile("../../docs/fixtures/per-step-run-evidence-v3/" + fixture + ".report.json")
							if err != nil {
								t.Fatal(err)
							}
							var report udonreport.ReportV5
							json.Unmarshal(data, &report)
							workflow, err := os.ReadFile(argValue(t, inner.Argv, "--workflow"))
							if err != nil {
								t.Fatal(err)
							}
							i, err := udonreport.InventoryFromWorkflowV5(workflow, "uws-yaml", argValue(t, inner.Argv, "--execution-run-id"))
							if err != nil {
								t.Fatal(err)
							}
							report.RunID = i.RunID
							report.WorkflowDigest = i.WorkflowDigest
							data, _ = json.Marshal(report)
							mustWriteFile(t, argValue(t, inner.Argv, "--execution-report"), data)
							if fixture == "interrupted" {
								return errors.New("killed")
							}
							return nil
						}})
					return err
				}})
			if result == nil || result.RunEvidencePath == "" {
				t.Fatalf("no evidence: %v", err)
			}
			if (err != nil) != (fixture == "interrupted") {
				t.Fatalf("wrong result: %v", err)
			}
			e := readRunEvidenceFile(t, result.RunEvidencePath)
			if e.StepExecution.State != "validated" {
				t.Fatalf("lost external report: %+v", e.StepExecution)
			}
			if fixture == "interrupted" && (e.StepExecution.ReportStatus != "incomplete" || e.StepExecution.Steps[1].Outcome != "unknown") {
				t.Fatal("lost uncertain write")
			}
			if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
				t.Fatal(err)
			}
		})
	}
}
