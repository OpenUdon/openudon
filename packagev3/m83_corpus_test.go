package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

func TestCorrectedM83CorpusPreservesExactSourceShapesAndReview(t *testing.T) {
	names := []string{"openapi.yaml", "google-discovery.json", "aws-smithy.json", "asyncapi.json", "graphql.graphql", "openrpc.json", "grpc.proto", "odata.xml"}
	kinds := []string{"openapi", "google-discovery", "aws-smithy", "asyncapi", "graphql", "openrpc", "grpc-protobuf", "odata"}
	o := packagev3.BuildOptions{Scope: "workflows/W01-m83", DataJSON: []byte(`{}`)}
	descriptions := []any{}
	for i, kind := range kinds {
		data, err := os.ReadFile(filepath.Join("testdata", "m83", "sources", names[i]))
		if err != nil {
			t.Fatal(err)
		}
		path := "sources/" + kind + "/" + names[i]
		o.Sources = append(o.Sources, packagev3.SourceInput{ID: kind, Kind: kind, Path: path, Bytes: data})
		descriptions = append(descriptions, map[string]any{"name": kind, "type": kind, "url": path})
	}
	raw := map[string]any{"uws": "1.13.0", "info": map[string]any{"title": "M83", "version": "1"}, "sourceDescriptions": descriptions, "operations": []any{}, "workflows": []any{map[string]any{"workflowId": "main", "type": "sequence", "steps": []any{map[string]any{"stepId": "waiting", "pending": map[string]any{"purpose": "Review corrected sources", "inputs": map[string]any{"type": "object"}, "outputs": map[string]any{"type": "object"}, "effect": "read"}}}}}}
	var err error
	o.WorkflowYAML, err = json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := packagev3.Build(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join("testdata", "m83", "eight-families.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p.Files[packagev3.ShapesPath], bytes.TrimSuffix(expected, []byte("\n"))) {
		t.Fatal("native corrected metadata/source/security changed")
	}
	table, err := binding.ParseTable(p.Files[packagev3.ShapesPath])
	if err != nil || len(table.Sources) != 8 || len(table.Operations) != 13 {
		t.Fatal("corrected corpus coverage", err)
	}
	if p.Assessment.Outcome != "incompatible" {
		t.Fatal("pending corpus became executable")
	}
	if _, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files}); err != nil {
		t.Fatal(err)
	}
}
