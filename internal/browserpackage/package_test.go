package browserpackage

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	engine "github.com/OpenUdon/openudon/internal/authoringengine"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

func authorFixture(t *testing.T, mode string, totp bool) (string, Request) {
	t.Helper()
	root, err := os.MkdirTemp(".", ".m96-test-")
	if err != nil {
		t.Fatal(err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Truncate(time.Second)
	r := Request{Version: Version, Kind: "request", RequestID: "synthetic", WorkflowName: "review_member", ExpectedTOTP: &totp, AllowOverwrite: true}
	var transaction browsertransaction.Transaction
	files := map[string][]byte{}
	if mode == "authenticated" {
		proof := authorresult.GoalProof{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard", Matches: 1}
		trace := []authorresult.TraceStep{{Kind: "focus_human_input", Phase: "authentication", Context: "main", Role: "textbox", Label: "Username", InputKind: "identifier"}}
		if totp {
			trace = append(trace, authorresult.TraceStep{Kind: "focus_human_input", Phase: "authentication", Context: "main", Role: "textbox", Label: "Code", InputKind: "otp", ChallengeKind: "totp"})
		}
		e, err := authorresult.Build(authorresult.BuildRequest{ObservedAt: at, Title: "Synthetic member", Goal: "Review member dashboard", InitialURL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard", Origins: []string{"https://members.example.test"}, Contexts: map[string]authorresult.Context{}, Bounds: authorresult.Bounds{NavigationTimeoutMS: 20000, TotalTimeoutMS: 600000, MaxRequests: 128, MaxResponseBytes: 8 << 20, MaxObservations: 32, MaxCandidates: 32, MaxOutputs: 8}, Trace: trace, GoalPredicate: authorresult.GoalPredicate{Origin: proof.Origin, Path: proof.Path, Context: proof.Context, Role: proof.Role, Label: proof.Label}, GoalProof: proof, AuthenticationProof: proof, HumanConfirmed: true})
		if err != nil {
			t.Fatal(err)
		}
		auth, err := authprofile.Parse(e.AuthenticationProfile)
		if err != nil {
			t.Fatal(err)
		}
		flow := authprofile.SortedFlowNames(auth)[0]
		var bindings []browsertransaction.CredentialBinding
		for _, s := range auth.Flows[flow].Sequence {
			if s.TypeCredential != nil {
				bindings = append(bindings, browsertransaction.CredentialBinding{Slot: s.TypeCredential.Slot, Binding: s.TypeCredential.Slot})
			}
			if s.Challenge != nil && s.Challenge.Slot != "" {
				bindings = append(bindings, browsertransaction.CredentialBinding{Slot: s.Challenge.Slot, Binding: s.Challenge.Slot})
			}
		}
		ar, _ := json.Marshal(e.AuthenticationReview)
		cr, _ := json.Marshal(e.CapabilityReview)
		candidate, err := browsercandidate.ComposeAuthenticationCapability(browsercandidate.AuthenticationCapabilityRequest{TransactionID: "member", Flow: flow, Session: "member_session", CredentialBindings: bindings, Authentication: e.AuthenticationProfile, AuthenticationReview: ar, Capability: e.CapabilityProfile, CapabilityReview: cr, ResultSHA256: Digest([]byte("synthetic private result")), ObservedAt: e.ObservedAt, Origins: e.Origins, AssessedAt: at})
		if err != nil {
			t.Fatal(err)
		}
		input, err := engine.AuthenticationCapabilityVirtualBrowserTransaction(candidate, true)
		if err != nil {
			t.Fatal(err)
		}
		transaction = input.Transaction
		files["browser-authentication/member-auth.json"] = append(append([]byte{}, e.AuthenticationProfile...), '\n')
		files["browser-profiles/member.json"] = append(append([]byte{}, e.CapabilityProfile...), '\n')
		review := map[string]any{"version": "openudon.authenticated-browser-authoring-review.v3", "profile_id": "member", "authentication_target": "browser-authentication/member-auth.json", "capability_target": "browser-profiles/member.json", "envelope_sha256": transaction.Provenance.ResultSHA256, "observed_at": e.ObservedAt, "goal": e.Goal, "goal_predicate": e.GoalPredicate, "origins": e.Origins, "contexts": e.Contexts, "bounds": e.Bounds, "trace_steps": len(e.Trace), "output_selections": e.OutputSelections, "authentication_review": e.AuthenticationReview, "capability_review": e.CapabilityReview, "diagnostics": e.Diagnostics, "private_envelope_kept_outside_package": true}
		files[".icot/authenticated-browser-authoring.json"], _ = json.Marshal(map[string]any{"version": "openudon.authenticated-browser-authoring-review.v3", "captures": []any{review}})
		start := browsercapture.StartRequest{Version: browsercapture.StartVersion, Mode: browsercapture.Authenticated, Authentication: &browsercapture.AuthenticationStart{ProfileID: "member", URL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard", GoalURL: "https://members.example.test/dashboard", Goal: e.Goal, Origins: e.Origins, GoalRole: "heading", GoalContext: "main", GoalLabel: "Dashboard", AfterAuthentication: "ask_after_authentication"}}
		r.Start, _ = json.MarshalIndent(start, "", "  ")
		r.Flow = flow
		cap, err := profile.ParseJSON(e.CapabilityProfile)
		if err != nil {
			t.Fatal(err)
		}
		r.Action = cap.SortedActionNames()[0]
	} else {
		value, err := registrationprofile.Parse([]byte(`profile: uws.browser-registration.1.0
info:
  title: Synthetic registration
  applicationOrigins: [https://app.example.test]
  registrationOrigins: [https://app.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "` + at.Format(time.RFC3339) + `", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "` + at.Format(time.RFC3339) + `"}
credentialSlots:
  identifier: {kind: identifier}
flows:
  create_account:
    sequence:
      - navigate: https://app.example.test/register
      - type_credential: {locator: {role: textbox}, slot: identifier}
      - submit: {locator: {role: button, name: Register}}
      - wait_for: {locator: {role: status}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: https://app.example.test, locator: {role: status}}
`))
		if err != nil {
			t.Fatal(err)
		}
		source, err := registrationprofile.MarshalJSON(value)
		if err != nil {
			t.Fatal(err)
		}
		review, err := registrationreview.Build(value, at)
		if err != nil {
			t.Fatal(err)
		}
		reviewBytes, _ := json.Marshal(review)
		transaction = browsertransaction.Transaction{Version: browsertransaction.Version, ID: "signup", Kind: browsertransaction.KindRegistration, State: browsertransaction.StateReviewed, Candidates: []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: value.Profile, SourceSHA256: Digest(source), ReviewSHA256: Digest(reviewBytes)}}, Provenance: browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1, ResultSHA256: Digest([]byte("synthetic registration result")), ObservedAt: at.Format(time.RFC3339Nano), ExpiresAt: at.Add(24 * time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://app.example.test"}}, CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "identifier", Binding: "registration_identifier"}}}
		files["browser-registration/signup.json"] = append(source, '\n')
		files["browser-registration/signup.review.json"] = append(reviewBytes, '\n')
		r.Start, _ = json.Marshal(browsercapture.StartRequest{Version: browsercapture.StartVersion, Mode: browsercapture.Registration, Registration: &browsercapture.RegistrationStart{ProfileID: "signup", TransactionID: "signup", URL: "https://app.example.test/register", Origins: []string{"https://app.example.test"}}})
		r.Flow = "create_account"
		r.RegistrationAuthority = "synthetic-requester"
		r.CleanupDisposition = "delete_separately"
	}
	r.ReceiptPath = "expected/browser-capture/" + transaction.ID + ".json"
	r.TransactionSHA256, _ = browsertransaction.Digest(transaction)
	rec := receipt{Version: "openudon.browser-capture-import.v1", StartSHA256: evidencefile.SHA256(r.Start), Transaction: transaction, Effect: "write"}
	for name, data := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0600); err != nil {
			t.Fatal(err)
		}
		rec.Files = append(rec.Files, importedFile{name, evidencefile.SHA256(data)})
	}
	receiptBytes, _ := json.MarshalIndent(rec, "", "  ")
	path := filepath.Join(root, filepath.FromSlash(r.ReceiptPath))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, receiptBytes, 0600); err != nil {
		t.Fatal(err)
	}
	r.ReceiptSHA256 = evidencefile.SHA256(receiptBytes)
	if err := os.WriteFile(filepath.Join(root, "project.md"), []byte("# Synthetic seed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r.InputSHA256, err = InputDigest(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	return root, r
}

func TestReadOnlyPlanThenExactApplyBothModes(t *testing.T) {
	for _, mode := range []string{"authenticated", "registration"} {
		t.Run(mode, func(t *testing.T) {
			root, r := authorFixture(t, mode, mode == "authenticated")
			before := r.InputSHA256
			data, _ := json.Marshal(r)
			plan, err := Prepare(context.Background(), root, data)
			if err != nil {
				t.Fatal("plan", err)
			}
			after, err := InputDigest(context.Background(), root)
			if err != nil || before != after {
				t.Fatal("plan wrote package", err)
			}
			if !plan.Ready || plan.Preview == nil || len(plan.FileActions) == 0 {
				t.Fatalf("plan not ready: %+v", plan.Blockers)
			}
			if _, err := Apply(context.Background(), root, data, plan.PlanSHA256, false); err == nil {
				t.Fatal("unconfirmed apply")
			}
			result, err := Apply(context.Background(), root, data, plan.PlanSHA256, true)
			if err != nil {
				t.Fatalf("apply %v; outcome=%s quality=%s", err, result.Outcome, result.QualityStatus)
			}
			if result.Outcome != "authored" || len(result.Written) == 0 {
				t.Fatal("missing authored evidence")
			}
			if _, err := Apply(context.Background(), root, data, plan.PlanSHA256, true); err == nil {
				t.Fatal("replayed authoring")
			}
		})
	}
}

func TestNativeReceiptDriftAndWrongPlanRefuseWithoutWrites(t *testing.T) {
	for _, mutation := range []string{"wrong-plan", "changed-input", "wrong-receipt", "wrong-start", "no-totp", "symlink"} {
		t.Run(mutation, func(t *testing.T) {
			root, r := authorFixture(t, "authenticated", false)
			data, _ := json.Marshal(r)
			plan, err := Prepare(context.Background(), root, data)
			if err != nil {
				t.Fatal(err)
			}
			expected := plan.PlanSHA256
			switch mutation {
			case "wrong-plan":
				expected = Digest([]byte("other plan"))
			case "changed-input":
				_ = os.WriteFile(filepath.Join(root, "project.md"), []byte("changed"), 0600)
			case "wrong-receipt":
				r.ReceiptSHA256 = strings.Repeat("f", 64)
			case "wrong-start":
				r.Start = append(r.Start, ' ')
			case "no-totp":
				yes := true
				r.ExpectedTOTP = &yes
			case "symlink":
				_ = os.Symlink("project.md", filepath.Join(root, "unsafe"))
			}
			data, _ = json.Marshal(r)
			_, err = Apply(context.Background(), root, data, expected, true)
			if err == nil {
				t.Fatal("unsafe apply accepted")
			}
			if _, err := os.Stat(filepath.Join(root, "workflows/intent.hcl")); !os.IsNotExist(err) {
				t.Fatal("refused apply wrote intent")
			}
		})
	}
}

func TestCommandRefusesAmbiguousOptionsAndDoesNotEcho(t *testing.T) {
	for _, args := range [][]string{
		{"apply", "--example", "x", "--request", "-"},
		{"plan", "--example", "x", "--request", "-", "--confirmed"},
		{"plan", "--example", "x", "--example=y", "--request", "-"},
		{"plan", "--example", "x", "--request", "-", "--credential", "sentinel-secret"},
	} {
		var out, errOut bytes.Buffer
		if RunCommand(context.Background(), args, strings.NewReader("{}"), &out, &errOut) != 2 || strings.Contains(errOut.String(), "sentinel-secret") || out.Len() != 0 {
			t.Fatal("invalid flags accepted or echoed", args)
		}
	}
}
