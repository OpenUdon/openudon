package browsercapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authprofile"
	engine "github.com/OpenUdon/openudon/internal/authoringengine"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/elicitor"
)

func nativeImportInput(t *testing.T) elicitor.VirtualBrowserTransactionInput {
	t.Helper()
	at := time.Now().UTC().Truncate(time.Second)
	proof := authorresult.GoalProof{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard", Matches: 1}
	envelope, err := authorresult.Build(authorresult.BuildRequest{
		ObservedAt: at, Title: "Disposable member profile", Goal: "reach member dashboard",
		InitialURL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard", Origins: []string{"https://members.example.test"}, Contexts: map[string]authorresult.Context{},
		Bounds:        authorresult.Bounds{NavigationTimeoutMS: 20_000, TotalTimeoutMS: 600_000, MaxRequests: 128, MaxResponseBytes: 8 << 20, MaxObservations: 32, MaxCandidates: 32, MaxOutputs: 8},
		GoalPredicate: authorresult.GoalPredicate{Origin: proof.Origin, Path: proof.Path, Context: proof.Context, Role: proof.Role, Label: proof.Label}, GoalProof: proof, AuthenticationProof: proof, HumanConfirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	authentication, err := authprofile.Parse(envelope.AuthenticationProfile)
	if err != nil {
		t.Fatal(err)
	}
	flow := authprofile.SortedFlowNames(authentication)[0]
	var bindings []browsertransaction.CredentialBinding
	for _, step := range authentication.Flows[flow].Sequence {
		if step.TypeCredential != nil {
			bindings = append(bindings, browsertransaction.CredentialBinding{Slot: step.TypeCredential.Slot, Binding: step.TypeCredential.Slot})
		}
	}
	authReview, _ := json.Marshal(envelope.AuthenticationReview)
	capReview, _ := json.Marshal(envelope.CapabilityReview)
	privateSHA := sha256.Sum256([]byte("synthetic private envelope, not native qualification"))
	candidate, err := browsercandidate.ComposeAuthenticationCapability(browsercandidate.AuthenticationCapabilityRequest{
		TransactionID: "member", Flow: flow, Session: "member_session", CredentialBindings: bindings,
		Authentication: envelope.AuthenticationProfile, AuthenticationReview: authReview, Capability: envelope.CapabilityProfile, CapabilityReview: capReview,
		ResultSHA256: "sha256:" + hex.EncodeToString(privateSHA[:]), ObservedAt: envelope.ObservedAt, Origins: envelope.Origins, AssessedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	input, err := engine.AuthenticationCapabilityVirtualBrowserTransaction(candidate, true)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func TestReviewedImportCommitsNativeProfilesAndReceiptTogether(t *testing.T) {
	for _, outcome := range []string{"approve", "workspace_drift", "target_collision", "expired"} {
		t.Run(outcome, func(t *testing.T) {
			root := t.TempDir()
			ctx := context.Background()
			guard, err := engine.ObserveWorkspace(ctx, root)
			if err != nil {
				t.Fatal(err)
			}
			input := nativeImportInput(t)
			if outcome == "expired" {
				input.Transaction.Provenance.ExpiresAt = time.Now().Add(time.Second).UTC().Format(time.RFC3339Nano)
			}
			admission, err := prepareProfileImport(ctx, root, input, nil, strings.Repeat("a", 64), guard)
			if err != nil {
				t.Fatal(err)
			}
			receipt := filepath.Join(root, "expected", "browser-capture", "member.json")
			if _, err := os.Stat(receipt); !os.IsNotExist(err) {
				t.Fatal("receipt written before approval")
			}
			switch outcome {
			case "workspace_drift":
				if err := os.WriteFile(filepath.Join(root, "project.md"), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "target_collision":
				if err := os.Mkdir(filepath.Join(root, "browser-profiles"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "browser-profiles", "member.json"), []byte("do not overwrite"), 0600); err != nil {
					t.Fatal(err)
				}
			case "expired":
				deadline, _ := time.Parse(time.RFC3339Nano, input.Transaction.Provenance.ExpiresAt)
				time.Sleep(time.Until(deadline) + time.Millisecond)
			}
			err = admission.commit(ctx)
			if outcome != "approve" {
				if err == nil {
					t.Fatal("invalid import committed")
				}
				if _, err := os.Stat(receipt); !os.IsNotExist(err) {
					t.Fatal("failed import left receipt")
				}
				if _, err := os.Stat(filepath.Join(root, "browser-authentication", "member-auth.json")); !os.IsNotExist(err) {
					t.Fatal("failed import left partial authentication profile")
				}
				if outcome == "target_collision" {
					data, _ := os.ReadFile(filepath.Join(root, "browser-profiles", "member.json"))
					if string(data) != "do not overwrite" {
						t.Fatal("collision overwritten")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(receipt)
			if err != nil {
				t.Fatal(err)
			}
			var decoded importReceipt
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Transaction.State != browsertransaction.StateReviewed || decoded.Effect != "write" || len(decoded.Files) != 2 {
				t.Fatal("reviewed receipt incomplete")
			}
			for _, file := range decoded.Files {
				content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
				if err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(content)
				if hex.EncodeToString(sum[:]) != file.SHA256 {
					t.Fatal("receipt does not bind exact materialization")
				}
			}
			if err := admission.commit(ctx); err == nil {
				t.Fatal("completed import replay overwrote a profile")
			}
		})
	}
}
