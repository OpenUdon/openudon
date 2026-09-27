package stepauthoring

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBindMatchesRunnablePublishedFixture(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	example := copyBindExample(t, fixtureRoot, false)
	outcome := Bind(context.Background(), example, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil || outcome.Result.Result.WriteOutcome != "written" {
		t.Fatalf("bind outcome = %#v", outcome)
	}
	got, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(fixtureRoot, "results", "step-bind.json"))
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("bind result differs from published fixture\ngot:  %s\nwant: %s", got, want)
	}
	intentBytes, err := os.ReadFile(filepath.Join(example, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	intent, err := workflowintent.ParseIntent(intentBytes, workflowintent.IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	steps := findSteps(intent.Steps, "list_projects")
	if len(steps) != 1 || steps[0].Source != "openapi/project-api.yaml" ||
		steps[0].With["page_size"] != "inputs.page_size" ||
		steps[0].With["X-API-Key"] != "credentials.project_api_key" {
		t.Fatalf("bound step = %#v", steps)
	}
}

func TestBindDistinguishesMissingAndUnsupportedVersions(t *testing.T) {
	request := BindRequest{Kind: "request", Command: BindCommand, StepID: "step"}
	missing := Bind(nil, t.TempDir(), request)
	if missing.Result.Diagnostics[0].Code != "request.invalid" {
		t.Fatalf("missing version diagnostic = %#v", missing.Result.Diagnostics)
	}
	request.Version = "openudon.step-authoring.v99"
	unsupported := Bind(nil, t.TempDir(), request)
	if unsupported.Result.Diagnostics[0].Code != "request.unsupported_version" {
		t.Fatalf("unsupported version diagnostic = %#v", unsupported.Result.Diagnostics)
	}
}

func TestBindPreservesUnrelatedIntentAndRepeatIsUnchanged(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	example := copyBindExample(t, fixtureRoot, true)
	intentPath := filepath.Join(example, "workflows", "intent.hcl")
	original, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	decorated := "# unrelated operator note stays byte-identical\n" + string(original) + `
step "keep_me" {
  type = "fnct"
  do = "Keep this step."
  operation = "local.keep"
}
`
	if err := os.WriteFile(intentPath, []byte(decorated), 0o600); err != nil {
		t.Fatal(err)
	}
	request.IntentRevision = IntentRevision{State: "present", SHA256: sourceDigest([]byte(decorated))}
	request.Scaffold = nil
	first := Bind(nil, example, request)
	if first.ExitCode != 0 || first.Result.Status != "completed" || first.Result.Result == nil || first.Result.Result.WriteOutcome != "written" {
		t.Fatalf("first bind = %#v", first)
	}
	updated, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(updated), "# unrelated operator note stays byte-identical\n") {
		t.Fatalf("unrelated content was not preserved:\n%s", updated)
	}
	parsed, err := workflowintent.ParseIntent(updated, workflowintent.IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	kept := findSteps(parsed.Steps, "keep_me")
	if len(kept) != 1 || kept[0].Type != "fnct" || kept[0].Do != "Keep this step." || kept[0].Operation != "local.keep" {
		t.Fatalf("unrelated step changed: %#v", kept)
	}
	request.IntentRevision.SHA256 = sourceDigest(updated)
	second := Bind(nil, example, request)
	if second.ExitCode != 0 || second.Result.Status != "completed" || second.Result.Result == nil || second.Result.Result.WriteOutcome != "unchanged" {
		t.Fatalf("repeat bind = %#v", second)
	}
	afterRepeat, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(updated, afterRepeat) {
		t.Fatal("repeat bind changed the intent bytes")
	}
}

func TestBindRejectsSelfReferencesAndDependencyCyclesWithoutMutation(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	for _, test := range []struct {
		name    string
		prepare func(*testing.T, *BindRequest, string) string
	}{
		{
			name: "self output reference",
			prepare: func(t *testing.T, request *BindRequest, intent string) string {
				request.RequestMappings["page_size"] = "steps.list_projects.received_body.projects"
				return intent
			},
		},
		{
			name: "transitive back edge",
			prepare: func(t *testing.T, request *BindRequest, intent string) string {
				request.DependsOn = []string{"prepare"}
				return intent + `
step "prepare" {
  type = "fnct"
  do = "Prepare the downstream step."
  depends_on = ["list_projects"]
}
`
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := readBindRequest(t, fixtureRoot)
			example := copyBindExample(t, fixtureRoot, true)
			intentPath := filepath.Join(example, "workflows", "intent.hcl")
			original, err := os.ReadFile(intentPath)
			if err != nil {
				t.Fatal(err)
			}
			changed := test.prepare(t, &request, string(original))
			if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
				t.Fatal(err)
			}
			request.IntentRevision = IntentRevision{State: "present", SHA256: sourceDigest([]byte(changed))}
			request.Scaffold = nil
			outcome := Bind(nil, example, request)
			if outcome.ExitCode != 4 || outcome.Result.Status != "needs_input" || outcome.Result.Diagnostics[0].Code != "dependency.cycle" {
				t.Fatalf("dependency-cycle bind = %#v", outcome)
			}
			after, err := os.ReadFile(intentPath)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, []byte(changed)) {
				t.Fatal("rejected dependency cycle changed the intent bytes")
			}
		})
	}
}

func TestBindStaleIntentAndSourceConflictWithoutMutation(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	example := copyBindExample(t, fixtureRoot, false)
	sourcePath := filepath.Join(example, "openapi", "project-api.yaml")
	if err := os.WriteFile(sourcePath, []byte("changed source bytes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Bind(nil, example, request)
	if got.ExitCode != 3 || got.Result.Status != "conflict" || got.Result.Diagnostics[0].Code != "source.stale" {
		t.Fatalf("stale source result = %#v", got)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("stale source created intent: %v", err)
	}

	example = copyBindExample(t, fixtureRoot, true)
	intentPath := filepath.Join(example, "workflows", "intent.hcl")
	before, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	request.IntentRevision = IntentRevision{State: "present", SHA256: "sha256:" + strings.Repeat("0", 64)}
	request.Scaffold = nil
	got = Bind(nil, example, request)
	if got.ExitCode != 3 || got.Result.Status != "conflict" || got.Result.Diagnostics[0].Code != "intent.stale" {
		t.Fatalf("stale intent result = %#v", got)
	}
	after, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("stale intent request changed bytes")
	}
}

func TestBindRejectsSecretAndReservedCredentialBindingsBeforeMutation(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	for _, binding := range []string{"clear", "none", "Bearer abcdefghijklmnopqrstuvwxyz012345"} {
		t.Run(binding, func(t *testing.T) {
			request := readBindRequest(t, fixtureRoot)
			request.CredentialBindings["api_key_auth"] = binding
			example := copyBindExample(t, fixtureRoot, false)
			got := Bind(nil, example, request)
			if got.ExitCode != 2 || got.Result.Status != "failed" {
				t.Fatalf("invalid credential binding result = %#v", got)
			}
			if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
				t.Fatalf("invalid binding created intent: %v", err)
			}
		})
	}
}

func TestBindRejectsMissingMappingsAndUnsafeSourceSymlink(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	request.RequestMappings = map[string]string{}
	example := copyBindExample(t, fixtureRoot, false)
	got := Bind(nil, example, request)
	if got.ExitCode != 4 || got.Result.Status != "needs_input" {
		t.Fatalf("incomplete mapping result = %#v", got)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("incomplete mapping created intent: %v", err)
	}

	example = t.TempDir()
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "source.yaml")
	if err := os.WriteFile(outside, []byte("private source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(example, "openapi", "project-api.yaml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	request = readBindRequest(t, fixtureRoot)
	got = Bind(nil, example, request)
	if got.ExitCode != 4 || got.Result.Status != "blocked" {
		t.Fatalf("symlink source result = %#v", got)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("symlink source created intent: %v", err)
	}
}

func TestBindRequiresConfirmedEffectAndAuthenticationAlternatives(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	tests := []struct {
		name       string
		replace    func(*BindRequest, string) string
		wantCode   string
		wantStatus string
	}{
		{
			name: "compound read and mutation",
			replace: func(request *BindRequest, source string) string {
				request.OperationRef.OperationKey = "getProjects"
				request.OperationRef.OperationID = "getProjects"
				return strings.Replace(source, "operationId: listProjects\n      summary: List projects.", "operationId: getProjects\n      summary: Retrieves the next project and deletes it from the queue.", 1)
			},
			wantCode: "effect.unknown", wantStatus: "needs_input",
		},
		{
			name: "known conflicting effect",
			replace: func(request *BindRequest, source string) string {
				request.OperationRef.OperationKey = "deleteProjects"
				request.OperationRef.OperationID = "deleteProjects"
				return strings.Replace(source, "operationId: listProjects\n      summary: List projects.", "operationId: deleteProjects\n      summary: Delete projects.", 1)
			},
			wantCode: "effect.conflict", wantStatus: "needs_input",
		},
		{
			name: "unknown effect",
			replace: func(request *BindRequest, source string) string {
				request.OperationRef.OperationKey = "processProjects"
				request.OperationRef.OperationID = "processProjects"
				return strings.Replace(source, "operationId: listProjects\n      summary: List projects.", "operationId: processProjects\n      summary: Process project information.", 1)
			},
			wantCode: "effect.unknown", wantStatus: "needs_input",
		},
		{
			name: "missing authentication alternative",
			replace: func(_ *BindRequest, source string) string {
				return strings.Replace(source, "      security:\n        - apiKeyAuth: []\n", "", 1)
			},
			wantCode: "authentication.unknown", wantStatus: "needs_input",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := readBindRequest(t, fixtureRoot)
			example := copyBindExample(t, fixtureRoot, false)
			sourcePath := filepath.Join(example, "openapi", "project-api.yaml")
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			updated := test.replace(&request, string(data))
			if updated == string(data) {
				t.Fatal("test did not change the source document")
			}
			if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}
			request.OperationRef.SourceSHA256 = "sha256:" + evidencefile.SHA256([]byte(updated))
			outcome := Bind(nil, example, request)
			if outcome.ExitCode != 4 || outcome.Result.Status != test.wantStatus || len(outcome.Result.Diagnostics) != 1 || outcome.Result.Diagnostics[0].Code != test.wantCode {
				t.Fatalf("unsafe bind outcome = %#v", outcome)
			}
			if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
				t.Fatalf("rejected bind changed the intent: %v", err)
			}
		})
	}

	t.Run("explicit anonymous alternative remains bindable", func(t *testing.T) {
		request := readBindRequest(t, fixtureRoot)
		request.CredentialBindings = map[string]string{}
		example := copyBindExample(t, fixtureRoot, false)
		sourcePath := filepath.Join(example, "openapi", "project-api.yaml")
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.Replace(string(data), "      security:\n        - apiKeyAuth: []\n", "      security: []\n", 1)
		if updated == string(data) {
			t.Fatal("anonymous security alternative was not added")
		}
		if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
			t.Fatal(err)
		}
		request.OperationRef.SourceSHA256 = "sha256:" + evidencefile.SHA256([]byte(updated))
		outcome := Bind(nil, example, request)
		if outcome.ExitCode != 0 || outcome.Result.Status != "completed" {
			t.Fatalf("explicit anonymous bind outcome = %#v", outcome)
		}
	})
}

func TestBindRejectsContractTypeMismatchWithoutMutation(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	request.Contract.Inputs.Properties["page_size"].Type = "string"
	request.Contract.Outputs.Properties["projects"].Type = "string"
	example := copyBindExample(t, fixtureRoot, false)
	outcome := Bind(nil, example, request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "needs_input" || len(outcome.Result.Diagnostics) != 1 || outcome.Result.Diagnostics[0].Code != "mapping.contract_mismatch" {
		t.Fatalf("type-mismatch bind outcome = %#v", outcome)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("type-mismatched bind created intent: %v", err)
	}
}

func TestBindRejectsConflictingCredentialAliasesWithoutMutation(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	request.CredentialBindings = map[string]string{"api-key": "first_key", "api_key": "second_key"}
	example := copyBindExample(t, fixtureRoot, false)
	sourcePath := filepath.Join(example, "openapi", "project-api.yaml")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.ReplaceAll(string(source), "apiKeyAuth", "api-key")
	if updated == string(source) {
		t.Fatal("source authentication scheme was not renamed")
	}
	if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	request.OperationRef.SourceSHA256 = "sha256:" + evidencefile.SHA256([]byte(updated))
	outcome := Bind(nil, example, request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "needs_input" || len(outcome.Result.Diagnostics) != 1 || outcome.Result.Diagnostics[0].Code != "binding.incomplete" {
		t.Fatalf("conflicting credential aliases outcome = %#v", outcome)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("conflicting aliases created intent: %v", err)
	}
}

func TestBindRejectsUnsupportedContractSchemaWithoutMutation(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	request.Contract.Inputs.Ref = "#/components/schemas/Inputs"
	example := copyBindExample(t, fixtureRoot, false)
	outcome := Bind(nil, example, request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "needs_input" || len(outcome.Result.Diagnostics) != 1 || outcome.Result.Diagnostics[0].Code != "mapping.incomplete" {
		t.Fatalf("unsupported-schema bind outcome = %#v", outcome)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("unsupported-schema bind created intent: %v", err)
	}
}

func TestBindCancellationDoesNotCreateIntent(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readBindRequest(t, fixtureRoot)
	example := copyBindExample(t, fixtureRoot, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome := Bind(ctx, example, request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "blocked" || len(outcome.Result.Diagnostics) != 1 || outcome.Result.Diagnostics[0].Code != "request.cancelled" {
		t.Fatalf("cancelled bind outcome = %#v", outcome)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("cancelled bind created intent: %v", err)
	}
}

func TestVerifySourceRevisionDetectsConcurrentSourceChange(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "openapi", "service.yaml")
	original := []byte("source revision one")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	expected := sourceDigest(original)
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	beforeReplace := bindBeforeReplace(root, "openapi/service.yaml", expected, intentPath, false)
	if err := os.WriteFile(path, []byte("source revision two"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := beforeReplace(); !errors.Is(err, errSourceChanged) {
		t.Fatalf("changed source revision error = %v, want errSourceChanged", err)
	}
}

func TestBindOptimisticWriterRejectsConcurrentIntentEdit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "workflows", "intent.hcl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("workflow {\n  name = \"before\"\n  description = \"Before.\"\n}\nstep \"one\" {\n  type = \"fnct\"\n  do = \"One.\"\n}\n")
	concurrent := []byte("workflow {\n  name = \"operator_edit\"\n  description = \"Keep this concurrent edit.\"\n}\nstep \"one\" {\n  type = \"fnct\"\n  do = \"One.\"\n}\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	prepared := artifactwriter.Prepared{ExampleRoot: root, Files: []artifactwriter.GeneratedFile{{
		Path: path, Content: "workflow {\n  name = \"after\"\n  description = \"After.\"\n}\nstep \"one\" {\n  type = \"fnct\"\n  do = \"One.\"\n}\n",
		AllowOverwrite: true, ExpectedCurrentSHA256: sourceDigest(original),
	}}}
	_, err := artifactwriter.CommitChecked(prepared, false, func() error {
		return os.WriteFile(path, concurrent, 0o600)
	})
	if err == nil {
		t.Fatal("concurrent edit was overwritten")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !reflect.DeepEqual(after, concurrent) {
		t.Fatalf("concurrent bytes changed: %q (writer error: %v)", after, err)
	}
}

func readBindRequest(t *testing.T, fixtureRoot string) BindRequest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "requests", "step-bind.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request BindRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		t.Fatal(err)
	}
	return request
}

func copyBindExample(t *testing.T, fixtureRoot string, withIntent bool) string {
	t.Helper()
	root := t.TempDir()
	source, err := os.ReadFile(filepath.Join(fixtureRoot, "bind-example", "openapi", "project-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "openapi", "project-api.yaml"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	if withIntent {
		intent, err := os.ReadFile(filepath.Join(fixtureRoot, "bind-example", "workflows", "intent.hcl"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "workflows"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "workflows", "intent.hcl"), intent, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
