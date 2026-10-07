package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/wire"
)

func TestPackageV3ContractGoldenFixture(t *testing.T) {
	root := filepath.Join("..", "docs", "fixtures", "package-v3-v1")
	var identity struct {
		Scope  string `json:"scope"`
		SHA256 string `json:"package_sha256"`
	}
	data, err := os.ReadFile(filepath.Join(root, "identity.json"))
	if err != nil || wire.DecodeStrictNumbers(data, &identity) != nil {
		t.Fatal("identity fixture", err)
	}
	files := map[string][]byte{}
	err = filepath.WalkDir(filepath.Join(root, "package"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Join(root, "package"), path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: identity.Scope, ExpectedSHA256: identity.SHA256, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	source := files["sources/openapi/api.yaml"]
	p, err := packagev3.Build(context.Background(), packagev3.BuildOptions{Scope: identity.Scope, WorkflowYAML: files[packagev3.WorkflowPath], DataJSON: files[packagev3.DataPath], Sources: []packagev3.SourceInput{{ID: "api", Kind: "openapi", Path: "sources/openapi/api.yaml", Bytes: source}}})
	if err != nil || p.SHA256 != identity.SHA256 {
		t.Fatal("package identity drift", err)
	}
	for path, expected := range files {
		if !bytes.Equal(p.Files[path], expected) {
			t.Fatalf("published synthetic package artifact %s changed", path)
		}
	}
	plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, executionOptions())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "plan.json"))
	if err != nil || !bytes.Equal(raw, expected) {
		t.Fatal("plan wire changed", err)
	}
	a, err := packagev3.DeriveBrokerAuthority(context.Background(), v, brokerOptions())
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	expected, err = os.ReadFile(filepath.Join(root, "authority.json"))
	if err != nil || !bytes.Equal(raw, expected) {
		t.Fatal("authority projection changed", err)
	}
	var read authority.Authority
	if wire.DecodeStrictNumbers(expected, &read) != nil || packagev3.CheckBrokerAuthority(context.Background(), v, executionOptions(), read, nil, brokerOptions().Now) != nil {
		t.Fatal("unchanged authority wire reader")
	}
}
