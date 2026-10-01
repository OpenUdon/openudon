package elicitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/draft"
	bevidence "github.com/OpenUdon/browsertools/evidence"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/review"
)

var guidedBundleTestTime = time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)

func TestDiscoverExplicitGuidedAuthoringBundleStagesOnlyVerifiedProfile(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "example")
	bundlePath := filepath.Join(root, "reviewed-guided.json")
	writeGuidedBundleFixture(t, bundlePath, false)

	discovery, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), example, "read member status", nil, nil, []BrowserSourceInput{{ID: "member-status", Path: bundlePath}}, guidedBundleTestTime)
	if err != nil {
		t.Fatalf("discover guided bundle: %v", err)
	}
	if len(discovery.BrowserReport.Candidates) != 1 || discovery.BrowserReport.Candidates[0].Kind != browserGuidedSourceKind {
		t.Fatalf("guided candidates = %#v", discovery.BrowserReport.Candidates)
	}
	if len(discovery.Plans) != 1 || discovery.Plans[0].SourceKind != string(browserGuidedSourceKind) || discovery.Plans[0].TargetPath != "browser-profiles/member-status.json" {
		t.Fatalf("guided plans = %#v", discovery.Plans)
	}
	if strings.Contains(string(discovery.Plans[0].MaterializedContent), browserGuidedAuthoringVersion) || strings.Contains(string(discovery.Plans[0].MaterializedContent), `"spec"`) || strings.Contains(string(discovery.Plans[0].MaterializedContent), `"review"`) || strings.Contains(string(discovery.Plans[0].MaterializedContent), `"decisions"`) {
		t.Fatalf("materialization retained guided envelope/evidence: %s", discovery.Plans[0].MaterializedContent)
	}
	var materialized profile.Profile
	if err := json.Unmarshal(discovery.Plans[0].MaterializedContent, &materialized); err != nil {
		t.Fatalf("decode materialized profile: %v", err)
	}
	if materialized.Schema != "uws.browser.1.5" || materialized.SortedActionNames()[0] != "read_status" {
		t.Fatalf("materialized profile = %#v", materialized)
	}

	resumed := discovery.Plans[0]
	resumed.MaterializedContent = nil
	content, err := SourceMaterializationContent(resumed, guidedBundleTestTime.Add(time.Hour))
	if err != nil {
		t.Fatalf("revalidate resumed guided bundle: %v", err)
	}
	if string(content) != string(discovery.Plans[0].MaterializedContent) {
		t.Fatalf("resumed materialization drifted\nwant: %s\ngot: %s", discovery.Plans[0].MaterializedContent, content)
	}
}

func TestDiscoverExplicitGuidedAuthoringBundleFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name    string
		tamper  bool
		unknown bool
		want    string
	}{
		{name: "profile tamper", tamper: true, want: "embedded profile does not match"},
		{name: "unknown field", unknown: true, want: "unknown field"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "guided.json")
			writeGuidedBundleFixture(t, path, test.tamper)
			if test.unknown {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				data = []byte(strings.Replace(string(data), `"version":`, `"unexpected":true,"version":`, 1))
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", nil, nil, []BrowserSourceInput{{ID: "status", Path: path}}, guidedBundleTestTime)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGuidedAuthoringAdapterRejectsStaleUnsupportedAndTrailingInput(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]byte) []byte
		at     time.Time
		want   string
	}{
		{name: "stale", mutate: func(data []byte) []byte { return data }, at: guidedBundleTestTime.Add(15 * 24 * time.Hour), want: "expired"},
		{name: "unsupported version", mutate: func(data []byte) []byte {
			return []byte(strings.Replace(string(data), browserGuidedAuthoringVersion, "browsertools.guided-authoring.v2", 1))
		}, at: guidedBundleTestTime, want: "unsupported"},
		{name: "trailing JSON", mutate: func(data []byte) []byte { return append(data, []byte("{}\n")...) }, at: guidedBundleTestTime, want: "trailing JSON"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "guided.json")
			writeGuidedBundleFixture(t, path, false)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, test.mutate(data), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", nil, nil, []BrowserSourceInput{{ID: "status", Path: path}}, test.at)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGuidedAuthoringAdapterRejectsUnsignedSafetyBypasses(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*browserGuidedBundle)
		want   string
	}{
		{
			name: "secret shaped evidence",
			mutate: func(bundle *browserGuidedBundle) {
				bundle.Evidence[0].Provenance.Session = "Bearer abcdefghijklmnopqrstuvwxyz"
			},
			want: "secret, credential, session",
		},
		{
			name: "literal type text",
			mutate: func(bundle *browserGuidedBundle) {
				locator := bevidence.CandidateLocator{Role: "textbox", Name: "Member number"}
				bundle.Evidence[0].CandidateLocators = []bevidence.CandidateLocator{locator}
				action := bundle.Spec.Actions["read_status"]
				action.Sequence = []profile.Step{{Kind: profile.StepTypeText, TypeText: &profile.TypeTextStep{Locator: profile.Locator{Role: profile.RoleTextbox, Name: "Member number"}, Value: "literal-member-number"}}}
				bundle.Spec.Actions["read_status"] = action
			},
			want: "declared parameter template",
		},
		{
			name: "literal select option",
			mutate: func(bundle *browserGuidedBundle) {
				locator := bevidence.CandidateLocator{Role: "combobox", Name: "Member type"}
				bundle.Evidence[0].CandidateLocators = []bevidence.CandidateLocator{locator}
				action := bundle.Spec.Actions["read_status"]
				action.Sequence = []profile.Step{{Kind: profile.StepSelectOption, SelectOption: &profile.SelectOptionStep{Locator: profile.Locator{Role: profile.RoleCombobox, Name: "Member type"}, Value: "premium"}}}
				bundle.Spec.Actions["read_status"] = action
			},
			want: "declared parameter template",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "guided.json")
			writeGuidedBundleFixture(t, path, false)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var bundle browserGuidedBundle
			if err := json.Unmarshal(data, &bundle); err != nil {
				t.Fatal(err)
			}
			test.mutate(&bundle)
			rebuilt, err := draft.Build(bundle.Evidence, bundle.Spec)
			if err != nil {
				t.Fatalf("rebuild crafted bundle: %v", err)
			}
			bundle.Profile = *rebuilt.Profile
			bundle.Decisions = append([]bevidence.LocatorDecision(nil), rebuilt.Decisions...)
			bundle.Spec.Decisions = append([]bevidence.LocatorDecision(nil), rebuilt.Decisions...)
			reviewed, err := review.Build(rebuilt.Profile, bundle.Evidence, bundle.Decisions, guidedBundleTestTime)
			if err != nil {
				t.Fatalf("review crafted bundle: %v", err)
			}
			bundle.Review = *reviewed
			encoded, err := json.MarshalIndent(bundle, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", nil, nil, []BrowserSourceInput{{ID: "status", Path: path}}, guidedBundleTestTime)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGuidedAuthoringAdapterRejectsDecisionMismatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "guided.json")
	writeGuidedBundleFixture(t, path, false)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var bundle browserGuidedBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	bundle.Decisions = append(bundle.Decisions, bevidence.LocatorDecision{
		ActionHint: "read_status",
		Locator:    bevidence.CandidateLocator{Role: "button", Name: "Read status"},
		Rationale:  "invented downstream decision",
	})
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", nil, nil, []BrowserSourceInput{{ID: "status", Path: path}}, guidedBundleTestTime)
	if err == nil || !strings.Contains(err.Error(), "ambiguity decisions do not match") {
		t.Fatalf("decision mismatch error = %v", err)
	}
}

func TestDiscoverExplicitGuidedAuthoringBundleDeduplicatesAndOverridesBroadRootAmbiguity(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "a-guided.json")
	second := filepath.Join(root, "b-guided.json")
	writeGuidedBundleFixture(t, first, false)
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, data, 0o600); err != nil {
		t.Fatal(err)
	}

	discovery, err := discoverBrowserAuthoringSources(context.Background(), filepath.Join(root, "example"), []BrowserSourceInput{{ID: "first", Path: first}, {ID: "second", Path: second}}, []string{root}, guidedBundleTestTime)
	if err != nil {
		t.Fatalf("discover duplicate guided bundles: %v", err)
	}
	if len(discovery.Report.Candidates) != 1 || discovery.Report.Candidates[0].Path != first {
		t.Fatalf("guided candidates = %#v", discovery.Report.Candidates)
	}
	if len(discovery.Plans) != 1 || discovery.Plans[0].ID != "first" {
		t.Fatalf("guided plans = %#v", discovery.Plans)
	}
	for _, diagnostic := range discovery.Report.Ambiguous {
		if diagnostic.Path == first || diagnostic.Path == second {
			t.Fatalf("explicit guided path remained ambiguous: %#v", diagnostic)
		}
	}
	if len(discovery.Report.Rejected) != 1 || discovery.Report.Rejected[0].Code != "duplicate" || discovery.Report.Rejected[0].Path != second {
		t.Fatalf("guided duplicate diagnostics = %#v", discovery.Report.Rejected)
	}
}

func TestGuidedAuthoringAdapterBoundsReplayWork(t *testing.T) {
	bundle := &browserGuidedBundle{
		Evidence: make([]bevidence.Record, browserGuidedMaxRecords+1),
		Spec: draft.Spec{
			Info:    profile.Info{Origin: profile.Origins{"https://example.test"}},
			Actions: map[string]draft.ActionSpec{"read": {Sequence: []profile.Step{{Kind: profile.StepNavigate, Navigate: "https://example.test"}}}},
		},
	}
	if err := validateBrowserGuidedBounds(bundle); err == nil || !strings.Contains(err.Error(), "evidence count") {
		t.Fatalf("oversized evidence error = %v", err)
	}
	bundle.Evidence = make([]bevidence.Record, browserGuidedMaxCatalogRecords+1)
	bundle.Spec.Actions = map[string]draft.ActionSpec{}
	for index := range bundle.Evidence {
		name := fmt.Sprintf("action_%d", index/browserGuidedMaxPerAction)
		bundle.Evidence[index].ActionHint = name
		bundle.Spec.Actions[name] = draft.ActionSpec{Sequence: []profile.Step{{Kind: profile.StepNavigate, Navigate: "https://example.test"}}}
	}
	if err := validateBrowserGuidedBounds(bundle); err != nil {
		t.Fatalf("valid cross-action catalog reuse was rejected: %v", err)
	}
	bundle.Evidence = make([]bevidence.Record, 1)
	bundle.Spec.Actions = map[string]draft.ActionSpec{}
	for index := 0; index <= browserGuidedMaxActions; index++ {
		bundle.Spec.Actions["action_"+strings.Repeat("x", index)] = draft.ActionSpec{Sequence: []profile.Step{{Kind: profile.StepNavigate, Navigate: "https://example.test"}}}
	}
	if err := validateBrowserGuidedBounds(bundle); err == nil || !strings.Contains(err.Error(), "action count") {
		t.Fatalf("oversized action error = %v", err)
	}

	locators := make([]bevidence.CandidateLocator, 5_000)
	outputLocators := make([]bevidence.CandidateOutput, 4_000)
	for index := range outputLocators {
		outputLocators[index].Locator = &bevidence.CandidateLocator{Role: "button", Name: fmt.Sprintf("output-%d", index)}
	}
	bundle.Evidence = []bevidence.Record{{ActionHint: "read", CandidateLocators: locators, CandidateOutputs: outputLocators}}
	bundle.Spec.Actions = map[string]draft.ActionSpec{"read": {Sequence: []profile.Step{{Kind: profile.StepNavigate, Navigate: "https://example.test"}}}}
	if err := validateBrowserGuidedBounds(bundle); err == nil || !strings.Contains(err.Error(), "locator/output bounds") {
		t.Fatalf("nested locator bound error = %v", err)
	}

	bundle.Evidence = []bevidence.Record{{ActionHint: "read", CandidateLocators: make([]bevidence.CandidateLocator, browserGuidedMaxLocators)}}
	sequence := make([]profile.Step, browserGuidedMaxStepsPerAction)
	for index := range sequence {
		sequence[index] = profile.Step{Kind: profile.StepClick, Click: &profile.LocatorStep{Locator: profile.Locator{Role: profile.RoleButton}}}
	}
	bundle.Spec.Actions = map[string]draft.ActionSpec{"read": {Sequence: sequence}}
	bundle.Decisions = make([]bevidence.LocatorDecision, browserGuidedMaxDecisions)
	bundle.Spec.Decisions = append([]bevidence.LocatorDecision(nil), bundle.Decisions...)
	if err := validateBrowserGuidedBounds(bundle); err == nil || !strings.Contains(err.Error(), "bounded matching work") {
		t.Fatalf("matching-work bound error = %v", err)
	}
}

func writeGuidedBundleFixture(t *testing.T, path string, tamper bool) {
	t.Helper()
	record := bevidence.Record{
		Origin: "https://example.test", ObservationKind: bevidence.ObservationA11ySnapshot,
		ObservedAt:      guidedBundleTestTime.Add(-time.Hour).Format(time.RFC3339),
		RedactionStatus: bevidence.RedactionNotRequired, ActionHint: "read_status",
		CandidateLocators: []bevidence.CandidateLocator{{Role: "button", Name: "Read status"}},
		Provenance:        bevidence.Provenance{Tool: "synthetic-test", Version: "1"},
	}
	raw := bevidence.RawRecord{Record: record}
	normalized, err := raw.Normalize()
	if err != nil {
		t.Fatalf("normalize evidence: %v", err)
	}
	records := []bevidence.Record{normalized}
	spec := draft.Spec{
		Info:            profile.Info{Title: "Member status", Provider: "example", Origin: profile.Origins{"https://example.test"}},
		ObservationKind: profile.ObservationAccessibilitySnapshot,
		Confidence:      profile.ConfidenceHigh, ExpiresAfter: "P14D",
		Actions: map[string]draft.ActionSpec{
			"read_status": {
				Sequence: []profile.Step{{Kind: profile.StepClick, Click: &profile.LocatorStep{Locator: profile.Locator{Role: profile.RoleButton, Name: "Read status"}}}},
				Outputs:  map[string]profile.Output{}, SideEffects: []profile.SideEffect{profile.SideEffectReadOnly},
				ConfirmationPolicy: profile.ConfirmationPolicy{Required: false},
			},
		},
	}
	result, err := draft.Build(records, spec)
	if err != nil {
		t.Fatalf("build guided fixture: %v", err)
	}
	reviewed, err := review.Build(result.Profile, records, result.Decisions, guidedBundleTestTime)
	if err != nil {
		t.Fatalf("review guided fixture: %v", err)
	}
	bundle := browserGuidedBundle{
		Version: browserGuidedAuthoringVersion, Spec: spec, Profile: *result.Profile,
		Evidence: records, Decisions: result.Decisions, Review: *reviewed,
	}
	if tamper {
		bundle.Profile.Info.Title = "Tampered title"
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatalf("marshal guided fixture: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write guided fixture: %v", err)
	}
}
