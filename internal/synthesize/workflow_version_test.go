package synthesize

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/convert"
)

func TestGeneratedWorkflowAdopts112OnlyForNewPackages(t *testing.T) {
	intent, err := rollout.ParseIntentFile(filepath.Join("..", "..", "examples", "eval", "runtime-only-render", "reference", "intent.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	result := resultPaths(root)
	fresh, err := generateWorkflowDocument(result, intent)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.UWS != "1.12.0" {
		t.Fatalf("new version %q", fresh.UWS)
	}
	if err := fresh.Validate(); err != nil {
		t.Fatal(err)
	}
	fresh.UWS = "1.11.0"
	hcl, err := convert.MarshalHCL(fresh)
	if err != nil {
		t.Fatal(err)
	}
	yaml, err := convert.MarshalYAML(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(result.WorkflowPath), 0700); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string][]byte{result.WorkflowPath: hcl, result.UWSPath: yaml} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	rebuilt, err := generateWorkflowDocument(result, intent)
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.UWS != "1.11.0" {
		t.Fatalf("legacy version changed to %q", rebuilt.UWS)
	}
	for path, before := range map[string][]byte{result.WorkflowPath: hcl, result.UWSPath: yaml} {
		after, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
			t.Fatalf("generation changed existing artifact %s: %v", path, err)
		}
	}
	if err := os.Remove(result.WorkflowPath); err != nil {
		t.Fatal(err)
	}
	if version, err := workflowUWSVersion(Result{ExampleDir: root}); err != nil || version != "1.11.0" {
		t.Fatalf("export-only legacy = %q, %v", version, err)
	}
}

func TestExistingWorkflowVersionRefusesDriftAndUnsafeArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name, hcl, yaml string
		symlink         bool
	}{
		{name: "conflicting versions", hcl: `uws = "1.11.0"`, yaml: "uws: 1.10.0\n"},
		{name: "missing version", hcl: `info { title = "version absent" }`},
		{name: "unpublished version", hcl: `uws = "1.99.0"`},
		{name: "invalid private version", hcl: `uws = "credential-value-canary"`},
		{name: "malformed document", hcl: `this is not a UWS document`},
		{name: "symlink document", hcl: `uws = "1.11.0"`, symlink: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := resultPaths(t.TempDir())
			if err := os.MkdirAll(filepath.Dir(result.WorkflowPath), 0700); err != nil {
				t.Fatal(err)
			}
			if tc.symlink {
				outside := filepath.Join(t.TempDir(), "outside.hcl")
				if err := os.WriteFile(outside, []byte(tc.hcl), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, result.WorkflowPath); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(result.WorkflowPath, []byte(tc.hcl), 0600); err != nil {
				t.Fatal(err)
			}
			if tc.yaml != "" {
				if err := os.WriteFile(result.UWSPath, []byte(tc.yaml), 0600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := workflowUWSVersion(result)
			if err == nil {
				t.Fatal("accepted invalid existing declaration")
			}
			if strings.Contains(err.Error(), "credential-value-canary") {
				t.Fatal("version error disclosed private field")
			}
			data, err := os.ReadFile(result.WorkflowPath)
			if err != nil || string(data) != tc.hcl {
				t.Fatalf("refusal changed source: %v", err)
			}
		})
	}
}

func TestVersionRefusalPrecedesRefinementWrites(t *testing.T) {
	root := t.TempDir()
	result := resultPaths(root)
	if err := os.MkdirAll(filepath.Dir(result.WorkflowPath), 0700); err != nil {
		t.Fatal(err)
	}
	hcl := []byte(`uws = "1.11.0"`)
	yaml := []byte("uws: 1.12.0\n")
	for path, data := range map[string][]byte{result.WorkflowPath: hcl, result.UWSPath: yaml} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := PackageFromIntent(context.Background(), Options{ExampleDir: root}); err == nil || !strings.Contains(err.Error(), "versions disagree") {
		t.Fatalf("refusal = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("refusal created artifacts: %v, %v", entries, err)
	}
	entries, err = os.ReadDir(filepath.Dir(result.WorkflowPath))
	if err != nil || len(entries) != 2 {
		t.Fatalf("refusal wrote artifacts: %v, %v", entries, err)
	}
	for path, before := range map[string][]byte{result.WorkflowPath: hcl, result.UWSPath: yaml} {
		after, err := os.ReadFile(path)
		if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
			t.Fatalf("refusal changed bytes: %v", err)
		}
	}
}
