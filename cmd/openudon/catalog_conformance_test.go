package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
)

func catalogConformanceCase(t *testing.T, name string) (stepauthoring.CatalogInstallation, apitools.CatalogDiscoveryRequest) {
	t.Helper()
	installation := preparedDiscoveryCatalog(t, name == "ambiguous")
	request := discoveryRequest()
	switch name {
	case "no_qualifying_api":
		request.Contract.Effect = apitools.OperationEffectWrite
	case "insufficient_evidence":
		installation.Root.Directory = ""
	case "blocked":
		request.ProviderKeys = []string{"unknown provider"}
	}
	return installation, request
}

func TestPublishedCatalogConformanceMatchesRealDispatcher(t *testing.T) {
	base := "../../docs/fixtures/catalog-discovery-v1"
	for _, name := range []string{"match", "ambiguous", "no_qualifying_api", "insufficient_evidence", "blocked"} {
		t.Run(name, func(t *testing.T) {
			installation, request := catalogConformanceCase(t, name)
			data, _ := json.Marshal(request)
			fixtureRequest, err := os.ReadFile(filepath.Join(base, name+".request.json"))
			if err != nil || !jsonEqual(data, fixtureRequest) {
				t.Fatal("request fixture drift", err)
			}
			cmd := helperCommand(append([]string{"step", "discover"}, discoveryCommandArgs(installation)...)...)
			cmd.Stdin = bytes.NewReader(fixtureRequest)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err = cmd.Run()
			if (name == "blocked") != (err != nil) || stderr.Len() != 0 {
				t.Fatal("dispatcher outcome drift", err, stderr.String())
			}
			want, err := os.ReadFile(filepath.Join(base, name+".report.json"))
			if err != nil || !jsonEqual(stdout.Bytes(), want) {
				t.Fatal("published native report drift", err)
			}
		})
	}
	installation := preparedDiscoveryCatalog(t, false)
	root := t.TempDir()
	data, err := os.ReadFile(filepath.Join(base, "source.request.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = stepauthoring.DecodeCatalogSourceRequest(data)
	if err != nil {
		t.Fatal(err)
	}
	expected := catalogSourceSelection(t, installation)
	expectedJSON, _ := json.Marshal(expected)
	if !jsonEqual(data, expectedJSON) {
		t.Fatal("published selected native references drift")
	}
	cmd := helperCommand("step", "source", "add", "--catalog", "--catalog-root", installation.Root.Directory, "--catalog-metadata", installation.MetadataPath, "--example", root, "--request", "-")
	cmd.Stdin = bytes.NewReader(data)
	actual, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(base, "source.result.json"))
	if err != nil || !jsonEqual(actual, want) {
		t.Fatal("published source result drift", err)
	}
	var result stepauthoring.SourceAddWireResult
	if json.Unmarshal(actual, &result) != nil || result.Result == nil {
		t.Fatal("invalid source result")
	}
	actual, err = os.ReadFile(filepath.Join(root, result.Result.ProvenancePath))
	if err != nil {
		t.Fatal(err)
	}
	want, err = os.ReadFile(filepath.Join(base, "source.provenance.json"))
	if err != nil || !jsonEqual(actual, want) {
		t.Fatal("published provenance drift", err)
	}
}

func jsonEqual(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	xj, _ := json.Marshal(x)
	yj, _ := json.Marshal(y)
	return bytes.Equal(xj, yj)
}
