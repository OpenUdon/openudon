// Package browsersystem composes existing qualification owners. It has no
// browser action, credential resolver, target URL or general executor surface.
package browsersystem

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const Version = "openudon.browser-system-qualification.v2"
const legacyVersion = "openudon.browser-system-qualification.v1"

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)

type Source struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
	SHA256 string `json:"source_sha256"`
}
type Stage struct {
	ID       string          `json:"id"`
	Status   string          `json:"status"`
	SHA256   string          `json:"evidence_sha256,omitempty"`
	Evidence json.RawMessage `json:"evidence,omitempty"`
}
type Pass struct {
	Number int     `json:"number"`
	Stages []Stage `json:"stages"`
}
type Toolchains struct {
	Go   string `json:"go"`
	Node string `json:"node"`
}

type Report struct {
	Toolchains   Toolchains                        `json:"toolchains"`
	Version      string                            `json:"version"`
	Suite        string                            `json:"suite"`
	Status       string                            `json:"status"`
	FailureStage string                            `json:"failure_stage"`
	Sources      []Source                          `json:"sources"`
	Baseline     browserscenario.CompatibilityLock `json:"baseline"`
	PlaywrightGo string                            `json:"playwright_go"`
	NetworkClaim string                            `json:"network_claim"`
	Passes       []Pass                            `json:"passes"`
}
type Tests struct {
	Passed          int    `json:"passed"`
	Skipped         int    `json:"skipped"`
	InventorySHA256 string `json:"inventory_sha256"`
}

var offlineStages = []string{"openudon_unit", "browsertools_unit", "driver_unit", "application_lifecycle"}
var legacyLoopbackStages = []string{"ui_browser", "registration_ui", "supervised_control", "build_inputs", "udon_browser_contract", "udon_browser_cli", "registration_driver", "loopback_scenarios", "journey_scenarios", "bap_bcp_transaction", "registration_ui_handoff"}
var loopbackStages = append(append([]string(nil), legacyLoopbackStages...), "supervised_registration_package", "supervised_authenticated_package")

func inventory(suite string) []string {
	if suite == "offline" {
		return offlineStages
	}
	return loopbackStages
}
func hash(data []byte) string { d := sha256.Sum256(data); return hex.EncodeToString(d[:]) }
func evidenceHash(data []byte) string {
	var compact bytes.Buffer
	if json.Compact(&compact, data) != nil {
		return ""
	}
	return hash(compact.Bytes())
}
func proof(id string, value any) Stage {
	data, _ := json.Marshal(value)
	return Stage{ID: id, Status: "pass", SHA256: hash(data), Evidence: data}
}
func canonical(value any) []byte {
	data, _ := json.MarshalIndent(value, "", "  ")
	return append(data, '\n')
}

// Verify validates exact inventory, ordering and component evidence. Integrity
// is independently checkable; authenticity still requires trusted execution and
// comparison of source digests to the reviewed local trees.
func validToolchains(value Toolchains, lock browserscenario.CompatibilityLock) bool {
	return value.Go == lock.GoVersion && regexp.MustCompile(`^`+regexp.QuoteMeta(lock.NodeVersion)+`\.[0-9]+\.[0-9]+$`).MatchString(value.Node)
}

func Validate(r *Report) error {
	bad := errors.New("browser system report is invalid")
	if r == nil || (r.Version != Version && r.Version != legacyVersion) || (r.Suite != "offline" && r.Suite != "loopback") || (r.Status != "pass" && r.Status != "fail") || r.PlaywrightGo != "v0.6201.0" || r.NetworkClaim != "application_request_allowlists_not_network_wide_containment" {
		return bad
	}
	lock, err := browserscenario.LoadCompatibilityLock()
	if err != nil || !bytes.Equal(canonical(lock), canonical(r.Baseline)) || !validToolchains(r.Toolchains, lock) {
		return bad
	}
	names := []string{"openudon", "browsertools", "uws", "udon", "browserdriver"}
	expectedBuildCommits := map[string]string{}
	if r.Suite == "loopback" {
		build, err := browserscenario.LoadQualificationBuildInputLock(lock)
		if err != nil {
			return bad
		}
		for _, component := range build.Components {
			name := "udon_build_" + component.Name
			names = append(names, name)
			expectedBuildCommits[name] = component.Commit
		}
	}
	if len(r.Sources) != len(names) {
		return bad
	}
	for i, s := range r.Sources {
		if s.Name != names[i] || !commitPattern.MatchString(s.Commit) || !digestPattern.MatchString(s.SHA256) || expectedBuildCommits[s.Name] != "" && s.Commit != expectedBuildCommits[s.Name] {
			return bad
		}
	}
	for _, component := range lock.Components {
		for _, source := range r.Sources {
			if source.Name == component.Name && source.Commit != component.Commit {
				return bad
			}
		}
	}
	count := 1
	if r.Suite == "loopback" {
		count = 3
	}
	if len(r.Passes) < 1 || len(r.Passes) > count {
		return bad
	}
	failed := ""
	for n, pass := range r.Passes {
		ids := inventory(r.Suite)
		if r.Version == legacyVersion && r.Suite == "loopback" {
			ids = legacyLoopbackStages
		}
		if pass.Number != n+1 || len(pass.Stages) < 1 || len(pass.Stages) > len(ids) {
			return bad
		}
		for i, stage := range pass.Stages {
			if stage.ID != ids[i] || failed != "" {
				return bad
			}
			if stage.Status == "fail" {
				if len(stage.Evidence) != 0 || stage.SHA256 != "" {
					return bad
				}
				failed = stage.ID
				continue
			}
			if stage.Status != "pass" || !digestPattern.MatchString(stage.SHA256) || stage.SHA256 != evidenceHash(stage.Evidence) {
				return bad
			}
			if err := validateProof(stage, r.Suite); err != nil {
				return bad
			}
			if stage.ID == "loopback_scenarios" || stage.ID == "journey_scenarios" {
				var component browserscenario.Report
				if evidencefile.DecodeStrict(stage.Evidence, &component) != nil {
					return bad
				}
				for i, revision := range component.Repositories {
					if revision.Name != r.Sources[i].Name || revision.Commit != r.Sources[i].Commit {
						return bad
					}
				}
				for _, dependency := range component.Dependencies {
					matched := false
					for _, pinned := range lock.Components {
						if dependency.Module == pinned.Module && dependency.Version == pinned.Version {
							matched = true
						}
					}
					if !matched {
						return bad
					}
				}
			}
		}
		if failed == "" && len(pass.Stages) != len(ids) {
			return bad
		}
	}
	if r.Status == "pass" {
		if failed != "" || r.FailureStage != "none" || len(r.Passes) != count {
			return bad
		}
	} else if failed == "" || failed != r.FailureStage {
		return bad
	}
	return nil
}
func decodeProof(data []byte, target any) error {
	if err := evidencefile.DecodeStrict(data, target); err != nil {
		return err
	}
	normalized, err := json.Marshal(target)
	if err != nil || hash(normalized) != evidenceHash(data) {
		return errors.New("component evidence is not canonical")
	}
	return nil
}
func validateProof(stage Stage, suite string) error {
	bad := errors.New("component evidence is invalid")
	switch stage.ID {
	case "loopback_scenarios", "journey_scenarios":
		var r browserscenario.Report
		if decodeProof(stage.Evidence, &r) != nil {
			return bad
		}
		if browserscenario.ValidateLocalQualificationReport(&r) != nil || r.Status != "pass" || r.Summary.Skipped != 0 || r.Summary.Quarantined != 0 {
			return bad
		}
		expected := browserscenario.SuiteLoopback
		if stage.ID == "journey_scenarios" {
			expected = browserscenario.SuiteJourney
		}
		manifests, err := browserscenario.LoadManifests(time.Now())
		if err != nil {
			return bad
		}
		selected, err := browserscenario.SelectManifests(manifests, expected, nil)
		if err != nil || len(selected) != len(r.Scenarios) {
			return bad
		}
		for i, manifest := range selected {
			if manifest.ID != r.Scenarios[i].ID {
				return bad
			}
		}
		if r.Suite != expected {
			return bad
		}
	case "bap_bcp_transaction":
		var e browserscenario.BAPBCPQualificationEvidence
		if decodeProof(stage.Evidence, &e) != nil {
			return bad
		}
		return browserscenario.ValidateBAPBCPQualificationEvidence(e)
	case "registration_ui_handoff":
		var e browserscenario.BRPQualificationEvidence
		if decodeProof(stage.Evidence, &e) != nil {
			return bad
		}
		return browserscenario.ValidateBRPQualificationEvidence(e)
	case "build_inputs":
		var e browserscenario.QualificationBuildInputLock
		if decodeProof(stage.Evidence, &e) != nil {
			return bad
		}
		lock, _ := browserscenario.LoadCompatibilityLock()
		expected, err := browserscenario.LoadQualificationBuildInputLock(lock)
		if err != nil || !bytes.Equal(canonical(e), canonical(expected)) {
			return bad
		}
	default:
		var e Tests
		if decodeProof(stage.Evidence, &e) != nil || e.Passed < 1 || e.Skipped < 0 || !digestPattern.MatchString(e.InventorySHA256) || suite == "loopback" && e.Skipped != 0 {
			return bad
		}
	}
	return nil
}
func Write(path string, r *Report) error {
	if err := Validate(r); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("report_output_not_regular")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return atomicfile.Write(path, canonical(r), 0600)
}
func Verify(path string) (*Report, error) {
	data, _, err := evidencefile.ReadRegular(path, 16<<20)
	if err != nil {
		return nil, err
	}
	var r Report
	if err = evidencefile.DecodeStrict(data, &r); err != nil {
		return nil, err
	}
	if err = Validate(&r); err != nil {
		return nil, err
	}
	if !bytes.Equal(data, canonical(&r)) {
		return nil, errors.New("browser system report is not canonical")
	}
	return &r, nil
}
