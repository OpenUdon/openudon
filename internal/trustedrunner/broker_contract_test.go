package trustedrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/internal/brokerhandoff"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestBrokerFieldsCannotDowngradeLegacyVersions(t *testing.T) {
	root, example, path := fixtureV5(t)
	a, _, err := readApprovalDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateApproval(a, a.Scope, a.PackageSHA256, TierSandbox, fixedNow()()); err != nil {
		t.Fatal(err)
	}
	a.Broker = &brokerhandoff.Authority{Version: brokerhandoff.Version}
	if validateApproval(a, a.Scope, a.PackageSHA256, TierSandbox, fixedNow()()) == nil {
		t.Fatal("legacy approval admitted broker authority")
	}
	result, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: path, DryRun: true, ExecutorReportVersion: "v5", Now: fixedNow(), Assess: passAssess})
	if err != nil {
		t.Fatal(err)
	}
	e := readRunEvidenceFile(t, result.RunEvidencePath)
	if err := validateRunEvidenceForVerify(e); err != nil {
		t.Fatal(err)
	}
	e.Broker = a.Broker
	if validateRunEvidenceForVerify(e) == nil {
		t.Fatal("legacy evidence admitted broker authority")
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var c udonrunner.Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	c.Broker = a.Broker
	data, _ = json.Marshal(c)
	path = filepath.Join(root, "downgrade.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := udonrunner.LoadConfig(path); err == nil {
		t.Fatal("legacy config admitted broker authority")
	}
}

func TestBrokerVersionedSchemasMatchGoEnvelopes(t *testing.T) {
	root, example, path := fixtureV5(t)
	a, _, err := readApprovalDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: path, DryRun: true, ExecutorReportVersion: "v5", Now: fixedNow(), Assess: passAssess})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var c RunConfig
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatal(err)
	}
	e := readRunEvidenceFile(t, result.RunEvidencePath)
	data, err = os.ReadFile("../../docs/fixtures/broker-handoff-v1/authority-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	var authority brokerhandoff.Authority
	if err := json.Unmarshal(data, &authority); err != nil {
		t.Fatal(err)
	}
	a.Version, a.Broker, a.ExpiresAt = BrokerApprovalVersion, &authority, authority.ExpiresAt
	c.Version, c.Broker = udonrunner.BrokerRunConfigVersion, &authority
	e.Version, e.Broker = BrokerRunEvidenceVersion, &authority
	compiler := jsonschema.NewCompiler()
	for _, version := range []string{brokerhandoff.Version, BrokerApprovalVersion, udonrunner.BrokerRunConfigVersion, BrokerRunEvidenceVersion} {
		data, err := os.ReadFile("../../docs/schemas/" + version + ".schema.json")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource("https://openudon.org/schemas/"+version+".schema.json", doc); err != nil {
			t.Fatal(err)
		}
	}
	for version, value := range map[string]any{BrokerApprovalVersion: a, udonrunner.BrokerRunConfigVersion: c, BrokerRunEvidenceVersion: e} {
		t.Run(version, func(t *testing.T) {
			schema, err := compiler.Compile("https://openudon.org/schemas/" + version + ".schema.json")
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(value)
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
			fields := instance.(map[string]any)
			fields["version"] = "legacy"
			if schema.Validate(fields) == nil {
				t.Fatal("schema accepted downgrade")
			}
		})
	}
}
