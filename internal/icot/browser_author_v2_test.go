package icot

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
)

func TestLiveAuthorRecordsOnlyHumanSelectedCompatibleMFAKind(t *testing.T) {
	tests := []struct {
		kind  string
		kinds []string
	}{
		{kind: "totp", kinds: []string{"totp"}},
		{kind: "push", kinds: []string{"push"}},
		{kind: "push_number_match", kinds: []string{"push_number_match"}},
	}
	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			root := t.TempDir()
			example, privateRoot := liveAuthorTestRoots(t, root)
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cfg := liveAuthorConfig{
				ExampleDir: example, Browsertools: executable, URL: "https://members.example.test/login",
				DashboardURL: "https://members.example.test/dashboard", Goal: "reach dashboard",
				Origins: []string{"https://members.example.test"}, PrivateRoot: privateRoot, ProfileID: "member",
				GoalRole: "heading", GoalLabel: "Dashboard", GoalContext: "main",
			}
			if err := normalizeLiveAuthorConfig(&cfg); err != nil {
				t.Fatal(err)
			}
			const candidateID = "candidate-0123456789abcdef"
			deps := liveAuthorDependencies{StartProcess: func(context.Context, string, []string, []string) (liveChild, error) {
				return newScriptedLiveChild(func(reader *bufio.Reader, writer io.Writer) error {
					encoder := json.NewEncoder(writer)
					if err := encoder.Encode(map[string]any{"protocol": liveAuthorProtocol, "type": "hello", "capabilities": liveAuthorTestCapabilities()}); err != nil {
						return err
					}
					if _, err := readTestClientMessage(reader); err != nil {
						return err
					}
					if err := encoder.Encode(map[string]any{"protocol": liveAuthorProtocol, "type": "state", "phase": "authentication", "context": "main", "bounds": defaultLiveAuthorBounds()}); err != nil {
						return err
					}
					if _, err := readTestClientMessage(reader); err != nil {
						return err
					}
					observation := map[string]any{
						"origin": "https://members.example.test", "path": "/login", "context": "main", "contexts": map[string]any{},
						"candidates": []any{map[string]any{"id": candidateID, "role": "textbox", "label": "Verification code", "matches": 1}}, "diagnostics": []any{},
					}
					if err := encoder.Encode(map[string]any{"protocol": liveAuthorProtocol, "type": "observation", "observation": observation}); err != nil {
						return err
					}
					focus, err := readTestClientMessage(reader)
					if err != nil || focus["type"] != "focus_human_input" || focus["candidateId"] != candidateID {
						return fmt.Errorf("focus message is invalid")
					}
					if err := encoder.Encode(map[string]any{"protocol": liveAuthorProtocol, "type": "human_checkpoint", "checkpoint": map[string]any{"kind": "mfa", "candidateId": candidateID, "challengeKinds": test.kinds}}); err != nil {
						return err
					}
					complete, err := readTestClientMessage(reader)
					if err != nil || complete["type"] != "human_input_complete" || complete["candidateId"] != candidateID || complete["challengeKind"] != test.kind {
						return fmt.Errorf("human input completion is invalid: %#v", complete)
					}
					return encoder.Encode(map[string]any{"protocol": liveAuthorProtocol, "type": "diagnostic", "diagnostic": map[string]any{"code": "test_complete"}})
				}), nil
			}}
			input := fmt.Sprintf("focus %s\n%s\ncontinue\n", candidateID, test.kind)
			_, err = orchestrateLiveAuthor(context.Background(), cfg, bufio.NewReader(strings.NewReader(input)), io.Discard, nil, "", "", deps)
			if err == nil || !strings.Contains(err.Error(), "test_complete") {
				t.Fatalf("orchestration did not reach the verified human choice: %v", err)
			}
		})
	}
}

func TestAuthenticatedAuthoringRejectsExtraOrigins(t *testing.T) {
	at := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	root := t.TempDir()
	example, privateRoot := liveAuthorTestRoots(t, root)
	path, _ := writeCustomV2Envelope(t, privateRoot, at, []authorresult.TraceStep{{Kind: "navigate", Phase: "authentication", Context: "main", URL: "https://members.example.test/login"}}, nil)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope authenticatedAuthoringEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Origins = append(envelope.Origins, "https://other.example.test")
	if err := validateAuthenticatedAuthoringEnvelope(testV2ImportConfig(example, privateRoot), &envelope, at); err == nil {
		t.Fatal("an extra result origin was accepted")
	}
}

func TestLiveOutputReviewIsBoundedTypedAndSafe(t *testing.T) {
	observation := liveObservation{
		Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Contexts: map[string]liveContext{},
		Candidates: []liveCandidate{
			{ID: "candidate-0000000000000001", Role: "status", Label: "Plan", Matches: 1},
			{ID: "candidate-0000000000000002", Role: "heading", Label: "Balance", Matches: 1},
			{ID: "candidate-0000000000000003", Role: "region", Label: "Active", Matches: 1},
		}, Diagnostics: []string{},
	}
	input := strings.Join([]string{
		"output raw-attacker-id leaked string exact_name",
		"output candidate-0000000000000001 access_token string exact_name",
		"output candidate-0000000000000001 plan presence exact_name",
		"output candidate-0000000000000002 balance number exact_name",
		"output candidate-0000000000000003 active boolean unique_role",
		"done",
	}, "\n") + "\n"
	var output strings.Builder
	requests, err := readHumanOutputRequests(bufio.NewReader(strings.NewReader(input)), &output, observation, 16)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 || requests[0].Key != "active" || requests[1].Key != "balance" || requests[2].Key != "plan" {
		t.Fatalf("reviewed requests = %#v", requests)
	}
	if strings.Contains(output.String(), "raw-attacker-id") || strings.Contains(output.String(), "access_token") {
		t.Fatalf("rejected identity entered diagnostics: %q", output.String())
	}
	if validateLivePlan(livePlan{Kind: "select_output", CandidateID: "candidate-0000000000000001"}, observation) == nil {
		t.Fatal("planner was allowed to select a completion output")
	}
}

func TestLiveOutputReviewIndependentlyRejectsUnsafeProofShapes(t *testing.T) {
	base := liveObservation{
		Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Contexts: map[string]liveContext{},
		Candidates: []liveCandidate{{ID: "candidate-0000000000000001", Role: "status", Label: "Plan", Matches: 1}}, Diagnostics: []string{},
	}
	valid := liveOutputRequest{CandidateID: "candidate-0000000000000001", Key: "plan", Type: "string", LocatorMode: "exact_name"}
	if err := validateLiveOutputRequest(valid, base, map[string]bool{}, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		request liveOutputRequest
		mutate  func(*liveObservation)
	}{
		{name: "reserved key", request: liveOutputRequest{CandidateID: valid.CandidateID, Key: "goal_present", Type: "string", LocatorMode: "exact_name"}},
		{name: "secret key", request: liveOutputRequest{CandidateID: valid.CandidateID, Key: "password", Type: "string", LocatorMode: "exact_name"}},
		{name: "composite type", request: liveOutputRequest{CandidateID: valid.CandidateID, Key: "plan", Type: "object", LocatorMode: "exact_name"}},
		{name: "unknown locator", request: liveOutputRequest{CandidateID: valid.CandidateID, Key: "plan", Type: "string", LocatorMode: "selector"}},
		{name: "control role", request: valid, mutate: func(value *liveObservation) { value.Candidates[0].Role = "textbox" }},
		{name: "marker label", request: valid, mutate: func(value *liveObservation) { value.Candidates[0].Label = "[redacted]" }},
		{name: "ambiguous target", request: valid, mutate: func(value *liveObservation) { value.Candidates[0].Matches = 2 }},
		{name: "noncanonical name", request: valid, mutate: func(value *liveObservation) { value.Candidates[0].Label = " Plan " }},
		{name: "nonunique role", request: liveOutputRequest{CandidateID: valid.CandidateID, Key: "plan", Type: "presence", LocatorMode: "unique_role"}, mutate: func(value *liveObservation) {
			value.Candidates = append(value.Candidates, liveCandidate{ID: "candidate-0000000000000002", Role: "status", Label: "Other", Matches: 1})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := base
			observation.Candidates = append([]liveCandidate(nil), base.Candidates...)
			if test.mutate != nil {
				test.mutate(&observation)
			}
			if err := validateLiveOutputRequest(test.request, observation, map[string]bool{}, map[string]bool{}); err == nil {
				t.Fatal("unsafe output proof was accepted")
			}
		})
	}
	if _, err := readHumanOutputRequests(bufio.NewReader(strings.NewReader("done\n")), io.Discard, base, 17); err == nil {
		t.Fatal("OpenUdon accepted output authority above its 16-selection contract")
	}
}

func TestAuthenticatedAuthoringV2ReconstructsTOTPAndScalarOutputs(t *testing.T) {
	at := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	root := t.TempDir()
	example, privateRoot := liveAuthorTestRoots(t, root)
	selections := []authorresult.OutputSelection{
		{CandidateID: "candidate-0000000000000001", Key: "account_name", Type: "string", LocatorMode: "exact_name", Observation: 3, Context: "main", Role: "status", Name: "Account name", Matches: 1, RoleMatches: 1},
		{CandidateID: "candidate-0000000000000002", Key: "active", Type: "boolean", LocatorMode: "exact_name", Observation: 3, Context: "main", Role: "region", Name: "Active", Matches: 1, RoleMatches: 1},
		{CandidateID: "candidate-0000000000000003", Key: "balance", Type: "number", LocatorMode: "exact_name", Observation: 3, Context: "main", Role: "heading", Name: "Balance", Matches: 1, RoleMatches: 2},
		{CandidateID: "candidate-0000000000000004", Key: "items", Type: "integer", LocatorMode: "exact_name", Observation: 3, Context: "main", Role: "alert", Name: "Items", Matches: 1, RoleMatches: 1},
		{CandidateID: "candidate-0000000000000005", Key: "plan_present", Type: "presence", LocatorMode: "unique_role", Observation: 3, Context: "main", Role: "group", Matches: 1, RoleMatches: 1},
	}
	trace := []authorresult.TraceStep{{
		Kind: "focus_human_input", Phase: "authentication", CandidateID: "candidate-abcdefabcdefabcd", Context: "main",
		Role: "textbox", Label: "Verification code", InputKind: "otp", ChallengeKind: "totp",
	}}
	path, digest := writeCustomV2Envelope(t, privateRoot, at, trace, selections)
	cfg := testV2ImportConfig(example, privateRoot)
	prepared, err := prepareAuthenticatedAuthoringImport(cfg, liveProtocolResult{ArtifactPath: path, Digest: digest}, at)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.AuthenticationSchema != "uws.browser-authentication.1.1" || prepared.CapabilitySchema != "uws.browser.1.7" {
		t.Fatalf("profile pair = %s / %s", prepared.AuthenticationSchema, prepared.CapabilitySchema)
	}
	var authentication map[string]any
	if err := json.Unmarshal([]byte(prepared.Files[0].Content), &authentication); err != nil {
		t.Fatal(err)
	}
	slots := authentication["credentialSlots"].(map[string]any)
	if slots["totp_seed"].(map[string]any)["kind"] != "totp_seed" {
		t.Fatalf("TOTP slot = %#v", slots)
	}
	var review authenticatedAuthoringSafeReview
	if err := json.Unmarshal([]byte(prepared.Files[4].Content), &review); err != nil || len(review.OutputSelections) != 5 {
		t.Fatalf("safe value-free review = %#v, %v", review, err)
	}
}

func TestAuthenticatedAuthoringRejectsProfileSubstitutionEvenWithMatchingDigest(t *testing.T) {
	at := time.Date(2026, 8, 17, 3, 0, 0, 0, time.UTC)
	root := t.TempDir()
	example, privateRoot := liveAuthorTestRoots(t, root)
	selection := authorresult.OutputSelection{CandidateID: "candidate-0000000000000001", Key: "balance", Type: "number", LocatorMode: "exact_name", Observation: 2, Context: "main", Role: "status", Name: "Balance", Matches: 1, RoleMatches: 1}
	path, _ := writeCustomV2Envelope(t, privateRoot, at, []authorresult.TraceStep{{Kind: "navigate", Phase: "authentication", Context: "main", URL: "https://members.example.test/login"}}, []authorresult.OutputSelection{selection})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var envelope authenticatedAuthoringEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	var capability map[string]any
	if err := json.Unmarshal(envelope.CapabilityProfile, &capability); err != nil {
		t.Fatal(err)
	}
	output := capability["actions"].(map[string]any)["reach_authenticated_goal"].(map[string]any)["outputs"].(map[string]any)["balance"].(map[string]any)
	output["locator"].(map[string]any)["role"] = "alert"
	envelope.CapabilityProfile, _ = json.Marshal(capability)
	profileSum := sha256.Sum256(envelope.CapabilityProfile)
	envelope.CapabilityReview.ProfileDigest = "sha256:" + hex.EncodeToString(profileSum[:])
	data, _ = json.Marshal(envelope)
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	cfg := testV2ImportConfig(example, privateRoot)
	_, err = prepareAuthenticatedAuthoringImport(cfg, liveProtocolResult{ArtifactPath: path, Digest: "sha256:" + hex.EncodeToString(sum[:])}, at)
	if err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("substituted profile was accepted: %v", err)
	}
}

func TestAuthorSessionV1AndMalformedV2CheckpointsFailClosed(t *testing.T) {
	v1 := []byte(`{"protocol":"browsertools.author-session.v1","type":"hello","capabilities":["chromium"]}`)
	if _, err := decodeLiveServerMessage(v1, 128); err == nil {
		t.Fatal("author-session v1 was accepted")
	}
	for _, checkpoint := range []string{
		`{"kind":"mfa","candidateId":"candidate-0123456789abcdef"}`,
		`{"kind":"mfa","candidateId":"candidate-0123456789abcdef","challengeKinds":["push","totp"]}`,
		`{"kind":"credential","candidateId":"candidate-0123456789abcdef","challengeKinds":["push"]}`,
	} {
		message := []byte(`{"protocol":"browsertools.author-session.v2","type":"human_checkpoint","checkpoint":` + checkpoint + `}`)
		if _, err := decodeLiveServerMessage(message, 128); err == nil {
			t.Fatalf("malformed checkpoint was accepted: %s", checkpoint)
		}
	}
}

func writeCustomV2Envelope(t *testing.T, privateRoot string, at time.Time, trace []authorresult.TraceStep, selections []authorresult.OutputSelection) (string, string) {
	t.Helper()
	proof := authorresult.GoalProof{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard", Matches: 1}
	envelope, err := authorresult.Build(authorresult.BuildRequest{
		ObservedAt: at, Title: "Member", Goal: "reach the member dashboard and learn how to read account status",
		InitialURL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard", Origins: []string{"https://members.example.test"},
		Contexts: map[string]authorresult.Context{}, Bounds: authorresult.Bounds{NavigationTimeoutMS: 20_000, TotalTimeoutMS: 600_000, MaxRequests: 512, MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128, MaxOutputs: 16},
		Trace: trace, OutputSelections: selections,
		GoalPredicate: authorresult.GoalPredicate{Origin: proof.Origin, Path: proof.Path, Context: proof.Context, Role: proof.Role, Label: proof.Label},
		GoalProof:     proof, AuthenticationProof: proof, HumanConfirmed: true, Diagnostics: []string{"value_free"},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := authorresult.MarshalDeterministic(envelope)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(privateRoot, "custom-v2-envelope.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return path, "sha256:" + hex.EncodeToString(sum[:])
}

func testV2ImportConfig(example, privateRoot string) liveAuthorConfig {
	return liveAuthorConfig{
		ExampleDir: example, PrivateRoot: privateRoot, ProfileID: "member", URL: "https://members.example.test/login",
		DashboardURL: "https://members.example.test/dashboard", Goal: "reach the member dashboard and learn how to read account status",
		Origins: []string{"https://members.example.test"}, GoalContext: "main", GoalRole: "heading", GoalLabel: "Dashboard",
	}
}

func TestLiveOutputCompletionAlwaysCarriesExplicitEmptyList(t *testing.T) {
	var wire bytes.Buffer
	protocol := newLiveProtocol(&wire, strings.NewReader(""))
	outputs := []liveOutputRequest{}
	if err := protocol.send(liveClientMessage{Type: "human_complete", Confirmed: true, Outputs: &outputs}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(wire.String(), `"outputs":[]`) {
		t.Fatalf("completion omitted explicit empty output list: %s", wire.String())
	}
}
