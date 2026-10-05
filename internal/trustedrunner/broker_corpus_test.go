package trustedrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/OpenUdon/openudon/internal/brokerhandoff"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type brokerCorpusEntry struct {
	Kind   string `json:"kind"`
	Valid  bool   `json:"valid"`
	SHA256 string `json:"sha256"`
}
type brokerCorpusManifest struct {
	Version            string                       `json:"version"`
	UdonSource         string                       `json:"udon_source"`
	UdonPublication    string                       `json:"udon_publication"`
	UdonExecutorSHA256 string                       `json:"udon_executor_sha256"`
	UdonClosureSHA256  string                       `json:"udon_closure_sha256"`
	UdonFixturesSHA256 string                       `json:"udon_fixtures_sha256"`
	Files              map[string]brokerCorpusEntry `json:"files"`
}

// This explicit generator publishes only synthetic, normalized contract
// examples. Actual qualification reports remain private, at their original
// source/build provenance. It never captures a real capability or credential.
func TestGenerateBrokerConsumerCorpus(t *testing.T) {
	if os.Getenv("OPENUDON_BROKER_GENERATE_CORPUS") != "1" {
		t.Skip("explicit synthetic corpus generation")
	}
	opts, a, _, _ := fixtureBroker(t)
	opts.DryRun = true
	r, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	c, err := udonrunner.LoadConfig(r.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	e := readRunEvidenceFile(t, r.RunEvidencePath)
	inspection, err := InspectBrokerPackage(context.Background(), TemplateOptions{RepoRoot: opts.RepoRoot, ExampleDir: opts.ExampleDir, Assess: passAssess})
	if err != nil {
		t.Fatal(err)
	}
	// Only transport-independent public fields survive normalization.
	c.PackageRoot, c.WorkDir = "/fixture/examples/support-email", "/fixture/run-"+c.RunID
	cbytes, _ := json.MarshalIndent(c, "", "  ")
	cbytes = append(cbytes, '\n')
	e.PackageRoot, e.WorkDir, e.StagePath, e.WorkflowPath, e.RunConfigPath = c.PackageRoot, c.WorkDir, c.WorkDir+"/stage", c.WorkDir+"/stage/workflows/workflow.uws.yaml", c.WorkDir+"/run-config.json"
	e.RunConfigSHA256 = evidencefile.SHA256(cbytes)
	unknown := e
	unknown.DryRun = false
	unknown.StageKind = "executor"
	unknown.Executor.Invoked, unknown.Executor.Mode = true, "internal-runner"
	unknown.StepExecution = nil
	o := udonreport.UnknownV5(e.StepExecution.Inventory(), "missing")
	unknown.StepExecution = &o
	unknown.Gates = append(append([]RunEvidenceGate(nil), e.Gates...), RunEvidenceGate{Name: "executor_invocation", Status: "fail"})
	root := "../../docs/fixtures/broker-execution-v1"
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	m := brokerCorpusManifest{Version: "openudon.broker-execution-fixtures.v1", UdonSource: m46BrokerQualification.source, UdonPublication: "71071537890599e98541abe8ca564490660dfd16", UdonExecutorSHA256: m46BrokerQualification.binary, UdonClosureSHA256: m46BrokerQualification.closure, UdonFixturesSHA256: "7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f", Files: map[string]brokerCorpusEntry{}}
	add := func(name, kind string, valid bool, value any) {
		t.Helper()
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		if strings.Contains(string(data), "SECRET_CANARY") || strings.Contains(string(data), strings.Repeat("c", 64)) {
			t.Fatal("private corpus value")
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0644); err != nil {
			t.Fatal(err)
		}
		m.Files[name] = brokerCorpusEntry{Kind: kind, Valid: valid, SHA256: evidencefile.SHA256(data)}
	}
	add("authority-valid.json", "authority", true, a.Broker)
	add("approval-valid.json", "approval", true, a)
	add("config-valid.json", "config", true, c)
	add("evidence-dry-run.json", "evidence", true, e)
	add("evidence-unknown.json", "evidence", true, unknown)
	add("plan-valid.json", "plan", true, inspection.Plan)
	changed := c
	changed.Version = RunConfigVersion
	add("config-downgrade.json", "config", false, changed)
	changed = c
	changed.Version = "openudon.executor-run.v99"
	add("config-unsupported.json", "config", false, changed)
	changed = c
	changed.PackageSHA256 = strings.Repeat("f", 64)
	add("config-stale-package.json", "config", false, changed)
	bad := unknown
	bad.StepExecution = new(udonreport.ObservationV5)
	*bad.StepExecution = *unknown.StepExecution
	bad.StepExecution.Steps = append([]udonreport.StepV5(nil), unknown.StepExecution.Steps...)
	bad.StepExecution.Steps[0].OperationID = "wrong-operation"
	add("evidence-wrong-operation.json", "evidence", false, bad)
	bad = unknown
	bad.Version = RunEvidenceVersionV3
	add("evidence-downgrade.json", "evidence", false, bad)
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerConsumerCorpus(t *testing.T) {
	root := "../../docs/fixtures/broker-execution-v1"
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m brokerCorpusManifest
	if err := evidencefile.DecodeStrict(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Version != "openudon.broker-execution-fixtures.v1" || m.UdonSource != m46BrokerQualification.source || m.UdonPublication != "71071537890599e98541abe8ca564490660dfd16" || m.UdonExecutorSHA256 != m46BrokerQualification.binary || m.UdonClosureSHA256 != m46BrokerQualification.closure || m.UdonFixturesSHA256 != "7289085b14f744504019ae6d607d351f74522ed3f930ecbbfd159200fd2ddb5f" {
		t.Fatal("corpus provenance mismatch")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != len(m.Files)+1 {
		t.Fatal("unlisted corpus fixture")
	}
	compiler := jsonschema.NewCompiler()
	for _, name := range []string{brokerhandoff.Version, BrokerApprovalVersion, udonrunner.BrokerRunConfigVersion, BrokerRunEvidenceVersion} {
		data, err := schemas.BrokerHandoffResources.ReadFile(name + ".schema.json")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource("https://openudon.org/schemas/"+name+".schema.json", doc); err != nil {
			t.Fatal(err)
		}
	}
	for name, entry := range m.Files {
		t.Run(name, func(t *testing.T) {
			if filepath.Base(name) != name {
				t.Fatal("unsafe fixture path")
			}
			data, err := os.ReadFile(filepath.Join(root, name))
			if err != nil || evidencefile.SHA256(data) != entry.SHA256 {
				t.Fatal("fixture hash mismatch")
			}
			version := ""
			switch entry.Kind {
			case "authority":
				var a brokerhandoff.Authority
				err = evidencefile.DecodeStrict(data, &a)
				if err == nil {
					err = a.Validate()
				}
				version = brokerhandoff.Version
			case "approval":
				var a Approval
				err = evidencefile.DecodeStrict(data, &a)
				if err == nil {
					err = validateApproval(a, a.Scope, a.PackageSHA256, TierSandbox, fixedNow()())
				}
				version = BrokerApprovalVersion
			case "config":
				var c RunConfig
				err = evidencefile.DecodeStrict(data, &c)
				if err == nil && (!udonrunner.ValidConfigVersion(c) || c.Broker == nil || c.Broker.Validate() != nil || c.PackageSHA256 != c.Broker.PackageSHA256 || c.HandoffSHA256 != c.Broker.HandoffSHA256 || c.RunID != c.Broker.RunID) {
					err = errors.New("fixture configuration mismatch")
				}
				version = udonrunner.BrokerRunConfigVersion
			case "evidence":
				var e RunEvidence
				err = evidencefile.DecodeStrict(data, &e)
				if err == nil {
					err = validateRunEvidenceForVerify(e)
				}
				if err == nil {
					err = verifyStepExecutionV3("", e, !e.DryRun && evidenceGateStatus(e, "executor_invocation") == "pass")
				}
				version = BrokerRunEvidenceVersion
			case "plan":
				var p udonrunner.BrokerPlan
				err = evidencefile.DecodeStrict(data, &p)
				if err == nil {
					if len(p.Requests) != len(p.Operations) {
						t.Fatal("request inventory mismatch")
					}
					for i, r := range p.Requests {
						var compact bytes.Buffer
						if json.Compact(&compact, r.Constraints) != nil || evidencefile.SHA256(compact.Bytes()) != p.Operations[i].ConstraintsSHA256 {
							t.Fatal("constraint bytes changed")
						}
					}
				}
			default:
				t.Fatal("unsupported corpus kind")
			}
			if (err == nil) != entry.Valid {
				t.Fatalf("fixture semantic result drift: %v", err)
			}
			if entry.Valid && version != "" {
				schema, err := compiler.Compile("https://openudon.org/schemas/" + version + ".schema.json")
				if err != nil {
					t.Fatal(err)
				}
				instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				if err := schema.Validate(instance); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	upstream := "../../docs/fixtures/udon-http-broker-v1"
	data, err = os.ReadFile(filepath.Join(upstream, "manifest.json"))
	if err != nil || evidencefile.SHA256(data) != m.UdonFixturesSHA256 {
		t.Fatal("original upstream manifest changed")
	}
	var wire struct {
		Version string                       `json:"version"`
		Files   map[string]brokerCorpusEntry `json:"files"`
	}
	if evidencefile.DecodeStrict(data, &wire) != nil || wire.Version != "udon.http-broker.fixtures.v1" {
		t.Fatal("upstream manifest format")
	}
	entries, err = os.ReadDir(upstream)
	if err != nil || len(entries) != len(wire.Files)+1 {
		t.Fatal("unlisted upstream fixture")
	}
	for name, entry := range wire.Files {
		if filepath.Base(name) != name {
			t.Fatal("unsafe upstream path")
		}
		data, err := os.ReadFile(filepath.Join(upstream, name))
		if err != nil || evidencefile.SHA256(data) != entry.SHA256 {
			t.Fatal("original upstream fixture changed")
		}
	}
}
