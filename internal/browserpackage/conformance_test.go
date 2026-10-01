package browserpackage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/artifactwriter"
	"github.com/OpenUdon/openudon/internal/browserauthoring"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

func TestCatalogAndUnboundPlanHaveNoApplyAuthority(t *testing.T) {
	for _, mutation := range []string{"selection", "inventory", "overwrite"} {
		t.Run(mutation, func(t *testing.T) {
			root, r := authorFixture(t, "authenticated", true)
			switch mutation {
			case "selection":
				r.Flow = ""
				r.Action = ""
			case "inventory":
				r.InputSHA256 = ""
			case "overwrite":
				r.AllowOverwrite = false
			}
			data, _ := json.Marshal(r)
			before, _ := InputDigest(context.Background(), root)
			plan, err := Prepare(context.Background(), root, data)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Candidates) == 0 || len(plan.Operations) == 0 {
				t.Fatal("catalog absent")
			}
			if mutation != "inventory" && (plan.Ready || len(plan.Blockers) == 0) {
				t.Fatal("unready plan grants authority")
			}
			if _, err := Apply(context.Background(), root, data, plan.PlanSHA256, true); err == nil {
				t.Fatal("unbound plan applied")
			}
			after, _ := InputDigest(context.Background(), root)
			if before != after {
				t.Fatal("refusal mutated package")
			}
		})
	}
}

func TestUnsafePackageModesAndHardlinksAreRefused(t *testing.T) {
	for _, mutation := range []string{"root-mode", "parent-mode", "file-mode", "hardlink", "outside-root"} {
		t.Run(mutation, func(t *testing.T) {
			root, r := authorFixture(t, "authenticated", false)
			switch mutation {
			case "root-mode":
				if err := os.Chmod(root, 0770); err != nil {
					t.Fatal(err)
				}
			case "parent-mode":
				if err := os.Chmod(filepath.Join(root, "expected"), 0777); err != nil {
					t.Fatal(err)
				}
			case "file-mode":
				if err := os.Chmod(filepath.Join(root, "project.md"), 0666); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(filepath.Join(root, "project.md"), filepath.Join(root, "extra.md")); err != nil {
					t.Fatal(err)
				}
			case "outside-root":
				root = t.TempDir()
			}
			data, _ := json.Marshal(r)
			if _, err := Prepare(context.Background(), root, data); err == nil {
				t.Fatal("unsafe package accepted")
			}
		})
	}
}

func TestReboundReceiptCannotChangeNativeReviewOrStartPolicy(t *testing.T) {
	for _, mutation := range []string{"expired", "unreviewed", "start-goal", "start-origins", "start-login", "start-dashboard", "profile", "extra-file", "source-bytes", "totp", "review"} {
		t.Run(mutation, func(t *testing.T) {
			root, r := authorFixture(t, "authenticated", false)
			recPath := filepath.Join(root, filepath.FromSlash(r.ReceiptPath))
			recBytes, err := os.ReadFile(recPath)
			if err != nil {
				t.Fatal(err)
			}
			var rec receipt
			if err := json.Unmarshal(recBytes, &rec); err != nil {
				t.Fatal(err)
			}
			switch mutation {
			case "expired":
				rec.Transaction.Provenance.ExpiresAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
			case "unreviewed":
				rec.Transaction.State = browsertransaction.StatePrepared
			case "start-goal", "start-origins", "start-login", "start-dashboard", "profile":
				var start browsercapture.StartRequest
				if err := json.Unmarshal(r.Start, &start); err != nil {
					t.Fatal(err)
				}
				if mutation == "start-goal" {
					start.Authentication.Goal = "A different outcome"
				} else if mutation == "start-origins" {
					start.Authentication.Origins = append(start.Authentication.Origins, "https://unexpected.example.test")
				} else if mutation == "start-login" {
					start.Authentication.URL = "https://members.example.test/other-login"
				} else if mutation == "start-dashboard" {
					start.Authentication.DashboardURL = "https://members.example.test/other-dashboard"
				} else {
					start.Authentication.ProfileID = "other"
				}
				r.Start, _ = json.Marshal(start)
				rec.StartSHA256 = evidencefile.SHA256(r.Start)
			case "extra-file":
				rec.Files = append(rec.Files, importedFile{"project.md", evidencefile.SHA256([]byte("# Synthetic seed\n"))})
			case "source-bytes":
				for _, f := range rec.Files {
					if strings.HasPrefix(f.Path, "browser-profiles/") {
						if err := os.WriteFile(filepath.Join(root, f.Path), []byte("{}"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
			case "totp":
				yes := true
				r.ExpectedTOTP = &yes
			case "review":
				for i, f := range rec.Files {
					if strings.HasPrefix(f.Path, ".icot/") {
						data, err := os.ReadFile(filepath.Join(root, f.Path))
						if err != nil {
							t.Fatal(err)
						}
						data = bytes.Replace(data, []byte("Review member dashboard"), []byte("A different outcome"), 1)
						if err := os.WriteFile(filepath.Join(root, f.Path), data, 0600); err != nil {
							t.Fatal(err)
						}
						rec.Files[i].SHA256 = evidencefile.SHA256(data)
					}
				}
			}
			r.TransactionSHA256, _ = browsertransaction.Digest(rec.Transaction)
			recBytes, _ = json.Marshal(rec)
			if err := os.WriteFile(recPath, recBytes, 0600); err != nil {
				t.Fatal(err)
			}
			r.ReceiptSHA256 = evidencefile.SHA256(recBytes)
			r.InputSHA256, err = InputDigest(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := json.Marshal(r)
			if _, err := Prepare(context.Background(), root, data); err == nil {
				t.Fatal("native constraint substitution accepted")
			}
			after, _ := InputDigest(context.Background(), root)
			if after != r.InputSHA256 {
				t.Fatal("refusal changed package")
			}
		})
	}
}

func TestOriginalReceiptBindsExactStartInBothModes(t *testing.T) {
	for _, mode := range []string{"authenticated", "registration"} {
		t.Run(mode, func(t *testing.T) {
			root, r := authorFixture(t, mode, false)
			before := r.InputSHA256
			var start browsercapture.StartRequest
			if err := json.Unmarshal(r.Start, &start); err != nil {
				t.Fatal(err)
			}
			if mode == "authenticated" {
				start.Authentication.URL = "https://members.example.test/other-login"
			} else {
				start.Registration.ProfileID = "other"
				start.Registration.URL = "https://app.example.test/other-register"
			}
			r.Start, _ = json.Marshal(start)
			data, _ := json.Marshal(r)
			if _, err := Prepare(context.Background(), root, data); err == nil {
				t.Fatal("changed start accepted against original approved receipt")
			}
			after, err := InputDigest(context.Background(), root)
			if err != nil || after != before {
				t.Fatal("start refusal changed original package", err)
			}
		})
	}
}

func TestPreReplaceInventoryIgnoresOnlyReportedNativeTransients(t *testing.T) {
	for _, sideWrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "own-staging", true: "unexpected-side-file"}[sideWrite], func(t *testing.T) {
			root, r := authorFixture(t, "authenticated", false)
			data, _ := json.Marshal(r)
			prepared, err := prepare(context.Background(), root, data)
			if err != nil {
				t.Fatal(err)
			}
			stopped := errors.New("stop before any replacement")
			observed := false
			_, err = artifactwriter.CommitCheckedObserved(prepared.files, true, func(paths []string) error {
				if len(paths) == 0 {
					t.Fatal("native transients not observed")
				}
				ignored := map[string]bool{}
				for _, p := range paths {
					rel, err := filepath.Rel(root, p)
					if err != nil || !safeRelative(filepath.ToSlash(rel)) {
						t.Fatal("transient escaped root")
					}
					ignored[p] = true
				}
				if sideWrite {
					if err := os.WriteFile(filepath.Join(root, "unexpected.tmp"), []byte("unapproved"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				actual, err := inputDigest(context.Background(), root, ignored)
				if err != nil {
					t.Fatal(err)
				}
				if (actual != r.InputSHA256) != sideWrite {
					t.Fatal("guard masks drift or own staging")
				}
				observed = true
				return stopped
			})
			if !errors.Is(err, stopped) || !observed {
				t.Fatal("pre-replace stop not retained", err)
			}
			if sideWrite {
				if err := os.Remove(filepath.Join(root, "unexpected.tmp")); err != nil {
					t.Fatal(err)
				}
			}
			actual, err := InputDigest(context.Background(), root)
			if err != nil || actual != r.InputSHA256 {
				t.Fatal("interrupted commit changed original or leaked staging", err)
			}
		})
	}
}

type lostOutput struct{}

func (lostOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestLostOutputAfterCommitRequiresInspectionAndRejectsReplay(t *testing.T) {
	root, r := authorFixture(t, "registration", false)
	data, _ := json.Marshal(r)
	plan, err := Prepare(context.Background(), root, data)
	if err != nil {
		t.Fatal(err)
	}
	var errOut bytes.Buffer
	args := []string{"apply", "--example", root, "--request", "-", "--expected-plan", plan.PlanSHA256, "--confirmed"}
	if RunCommand(context.Background(), args, bytes.NewReader(data), lostOutput{}, &errOut) != 1 || !strings.Contains(errOut.String(), "output lost; inspect") {
		t.Fatal("lost output did not require inspection", errOut.String())
	}
	if _, err := os.Stat(filepath.Join(root, "workflows/intent.hcl")); err != nil {
		t.Fatal("committed authoring missing", err)
	}
	if _, err := os.Stat(filepath.Join(root, "expected/quality.json")); err != nil {
		t.Fatal("built evidence missing", err)
	}
	after, err := InputDigest(context.Background(), root)
	if err != nil || after == r.InputSHA256 {
		t.Fatal("commit not evidenced", err)
	}
	if _, err := Apply(context.Background(), root, data, plan.PlanSHA256, true); err == nil {
		t.Fatal("output loss allowed replay")
	}
}

func TestCancelledContextBeforeCommitPreservesPackage(t *testing.T) {
	root, r := authorFixture(t, "authenticated", true)
	data, _ := json.Marshal(r)
	plan, err := Prepare(context.Background(), root, data)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Apply(ctx, root, data, plan.PlanSHA256, true); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	after, err := InputDigest(context.Background(), root)
	if err != nil || after != r.InputSHA256 {
		t.Fatal("cancelled authoring wrote package", err)
	}
}

func TestImportedPolicyRetainsNativeCanonicalStartForms(t *testing.T) {
	root, r := authorFixture(t, "authenticated", true)
	var start browsercapture.StartRequest
	if err := json.Unmarshal(r.Start, &start); err != nil {
		t.Fatal(err)
	}
	start.Authentication.GoalRole = " HEADING "
	start.Authentication.GoalContext = " main "
	start.Authentication.GoalLabel = " Dashboard "
	start.Authentication.Origins = []string{" https://members.example.test/ "}
	// The actual native capture config accepts and canonicalizes these values.
	startData, _ := json.Marshal(start)
	if _, err := browsercapture.DecodeStart(startData); err != nil {
		t.Fatal("native start schema rejected fixture", err)
	}
	private := t.TempDir()
	if err := os.Chmod(private, 0700); err != nil {
		t.Fatal(err)
	}
	_, cfg, err := start.AuthenticationConfig(root, private, "")
	if err != nil {
		a := start.Authentication
		diagnostic := browserauthoring.LiveConfig{ExampleDir: root, PrivateRoot: private, URL: a.URL, DashboardURL: a.DashboardURL, GoalURL: a.GoalURL, Goal: a.Goal, Origins: a.Origins, ProfileID: a.ProfileID, AfterAuthentication: a.AfterAuthentication, GoalRole: a.GoalRole, GoalLabel: a.GoalLabel, GoalContext: a.GoalContext, NoLLM: true}
		t.Fatal("native config rejected fixture", err, browserauthoring.NormalizeLiveConfig(&diagnostic))
	}
	if cfg.GoalRole != "heading" || cfg.GoalContext != "main" || cfg.ProfileID != "member" {
		t.Fatal("native normalization changed")
	}
	r.Start, _ = json.Marshal(start)
	path := filepath.Join(root, filepath.FromSlash(r.ReceiptPath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rec receipt
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatal(err)
	}
	rec.StartSHA256 = evidencefile.SHA256(r.Start)
	data, _ = json.Marshal(rec)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	r.ReceiptSHA256 = evidencefile.SHA256(data)
	r.InputSHA256, err = InputDigest(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = json.Marshal(r)
	plan, err := Prepare(context.Background(), root, data)
	if err != nil || !plan.Ready {
		t.Fatal("native canonical equivalents refused", err)
	}
}
