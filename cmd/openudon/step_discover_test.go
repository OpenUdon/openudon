package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/sqlitecache"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
)

// Prepare the producer's real index from source bytes and registrations. These
// fixtures are exact upstream bytes; no expected outcome is inserted into it.
func preparedDiscoveryCatalog(t *testing.T, ambiguous bool) stepauthoring.CatalogInstallation {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"catalog.json", "registrations.json", "openapi/notes.json"} {
		data, err := os.ReadFile(filepath.Join("testdata/catalog-root", name))
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(filepath.Join(root, "registrations.json"))
	var rows []sqlitecache.CatalogArtifact
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if ambiguous {
		path := filepath.Join(root, "openapi/notes.json")
		data, _ := os.ReadFile(path)
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		paths := doc["paths"].(map[string]any)
		original, _ := json.Marshal(paths["/notes"].(map[string]any)["get"])
		var operation map[string]any
		_ = json.Unmarshal(original, &operation)
		operation["operationId"] = "listArchivedNotes"
		paths["/archive"] = map[string]any{"get": operation}
		data, _ = json.Marshal(doc)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		for i := range rows {
			rows[i].SHA256 = hex.EncodeToString(hash[:])
			rows[i].Bytes = int64(len(data))
		}
	}
	cache, err := sqlitecache.Open(filepath.Join(root, "cache.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if err := cache.StoreCatalogArtifact(context.Background(), row); err != nil {
			t.Fatal(err)
		}
	}
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	installation := stepauthoring.CatalogInstallation{MetadataPath: filepath.Join(root, "catalog.json")}
	installation.Root.Directory = root
	options, err := installation.IndexOptions()
	if err != nil {
		t.Fatal(err)
	}
	index, err := apitools.BuildCatalogOperationIndex(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if err := apitools.WriteCatalogOperationIndex(context.Background(), options, index); err != nil {
		t.Fatal(err)
	}
	return installation
}

func discoveryCommandArgs(c stepauthoring.CatalogInstallation) []string {
	return []string{"--catalog-root", c.Root.Directory, "--catalog-metadata", c.MetadataPath, "--request", "-"}
}

func discoveryRequest() apitools.CatalogDiscoveryRequest {
	return apitools.CatalogDiscoveryRequest{SchemaVersion: apitools.CatalogDiscoverySchemaVersion, Contract: apitools.StepContract{Purpose: "list notes records", Effect: apitools.OperationEffectRead}}
}

func TestStepDiscoverPreservesAllFiveNativeOutcomes(t *testing.T) {
	for _, outcome := range []apitools.CatalogDiscoveryOutcome{apitools.CatalogDiscoveryMatch, apitools.CatalogDiscoveryAmbiguous, apitools.CatalogDiscoveryNoQualifyingAPI, apitools.CatalogDiscoveryInsufficientEvidence, apitools.CatalogDiscoveryBlocked} {
		t.Run(string(outcome), func(t *testing.T) {
			installation := preparedDiscoveryCatalog(t, outcome == apitools.CatalogDiscoveryAmbiguous)
			request := discoveryRequest()
			switch outcome {
			case apitools.CatalogDiscoveryNoQualifyingAPI:
				request.Contract.Effect = apitools.OperationEffectWrite
			case apitools.CatalogDiscoveryInsufficientEvidence:
				installation.Root.Directory = ""
			case apitools.CatalogDiscoveryBlocked:
				request.ProviderKeys = []string{"unknown provider"}
			}
			options, err := installation.IndexOptions()
			if err != nil {
				t.Fatal(err)
			}
			native, err := apitools.DiscoverCatalogOperations(context.Background(), apitools.CatalogDiscoveryOptions{Index: options, Request: request})
			if err != nil || native.Outcome != outcome {
				t.Fatalf("native fixture: outcome=%s err=%v", native.Outcome, err)
			}
			encoded, _ := json.Marshal(request)
			var stdout, stderr bytes.Buffer
			code := runStepDiscoverCommand(discoveryCommandArgs(installation), bytes.NewReader(encoded), &stdout, &stderr)
			wantCode := 0
			if outcome == apitools.CatalogDiscoveryBlocked {
				wantCode = 4
			}
			want, _ := json.Marshal(native)
			if code != wantCode || !bytes.Equal(bytes.TrimSpace(stdout.Bytes()), want) || stderr.Len() != 0 {
				t.Fatalf("native report changed: code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
			}
			if bytes.Contains(stdout.Bytes(), []byte(installation.MetadataPath)) {
				t.Fatal("installation path leaked")
			}
		})
	}
}

func TestStepDiscoverPreservesProviderKeysEmptyScopeAndRelocation(t *testing.T) {
	first, second := preparedDiscoveryCatalog(t, false), preparedDiscoveryCatalog(t, false)
	for _, keys := range [][]string{nil, {}, {"Example Notes Shared"}} {
		request := discoveryRequest()
		request.ProviderKeys = keys
		encoded, _ := json.Marshal(request)
		var a, b, stderr bytes.Buffer
		if runStepDiscoverCommand(discoveryCommandArgs(first), bytes.NewReader(encoded), &a, &stderr) != 0 || runStepDiscoverCommand(discoveryCommandArgs(second), bytes.NewReader(encoded), &b, &stderr) != 0 {
			t.Fatal("discovery failed")
		}
		if !bytes.Equal(a.Bytes(), b.Bytes()) {
			t.Fatal("root relocation changed evidence")
		}
		var report apitools.CatalogDiscoveryReport
		if err := json.Unmarshal(a.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if keys != nil && len(keys) == 0 && (report.Outcome != apitools.CatalogDiscoveryInsufficientEvidence || len(report.Scope.ProviderIDs) != 0) {
			t.Fatal("empty provider list broadened scope")
		}
		if len(keys) == 1 && (len(report.Scope.ProviderIDs) != 1 || report.Scope.ProviderIDs[0] != "example-notes-alias") {
			t.Fatal("multiword provider key split or broadened")
		}
		for _, candidate := range report.Candidates {
			for _, source := range candidate.Sources {
				if source.LicenseIdentifier != "unknown" || source.Redistribution != "unknown" {
					t.Fatal("license or redistribution permission inferred")
				}
			}
		}
	}
}

func TestStepDiscoverNeverBuildsMissingIndexOrBroadensRemoteAuthority(t *testing.T) {
	installation := preparedDiscoveryCatalog(t, false)
	indexPath := filepath.Join(installation.Root.Directory, "operations.v1.json")
	if err := os.Remove(indexPath); err != nil {
		t.Fatal(err)
	}
	request := discoveryRequest()
	data, _ := json.Marshal(request)
	var stdout, stderr bytes.Buffer
	if code := runStepDiscoverCommand(discoveryCommandArgs(installation), bytes.NewReader(data), &stdout, &stderr); code != 0 {
		t.Fatal(code)
	}
	var report apitools.CatalogDiscoveryReport
	_ = json.Unmarshal(stdout.Bytes(), &report)
	if report.Outcome != apitools.CatalogDiscoveryInsufficientEvidence {
		t.Fatal("missing index became absence")
	}
	if _, err := os.Stat(indexPath); !os.IsNotExist(err) {
		t.Fatal("read-only discovery created index")
	}
	request.RemoteLookup = true
	data, _ = json.Marshal(request)
	stdout.Reset()
	if code := runStepDiscoverCommand(discoveryCommandArgs(installation), bytes.NewReader(data), &stdout, &stderr); code != 4 {
		t.Fatal("request-only remote opt-in was accepted", code)
	}
	if !strings.Contains(stdout.String(), "discovery.remote_unavailable") {
		t.Fatal("remote refusal lost native diagnostic")
	}
}

func TestStepDiscoverRejectsUntrustedInstallationAndMalformedRequests(t *testing.T) {
	for _, input := range []string{`{"schema_version":"apitools.catalog-discovery/v1","contract":{"purpose":"list notes"},"root":"secret-canary"}`, `{"schema_version":"a","schema_version":"b"}`, `{} {}`, strings.Repeat("x", apitools.MaxCatalogDiscoveryRequestBytes+1)} {
		var stdout, stderr bytes.Buffer
		if code := runStepDiscoverCommand([]string{"--request", "-"}, strings.NewReader(input), &stdout, &stderr); code != 2 {
			t.Fatal("malformed request accepted", code)
		}
		if strings.Contains(stdout.String()+stderr.String(), "secret-canary") {
			t.Fatal("raw invalid request leaked")
		}
	}
	var stdout, stderr bytes.Buffer
	if code := runStepDiscoverCommand([]string{"--secret-canary"}, strings.NewReader("{}"), &stdout, &stderr); code != 2 || strings.Contains(stdout.String()+stderr.String(), "secret-canary") {
		t.Fatal("invalid flags accepted or echoed")
	}
}
