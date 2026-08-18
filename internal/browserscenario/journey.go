package browserscenario

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	bevidence "github.com/OpenUdon/browsertools/evidence"
	"github.com/OpenUdon/browsertools/guide"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const (
	journeyAuthenticationFlow = "open_journey"
	journeySession            = "journey_session"
)

type JourneyFixture struct {
	manifest     Manifest
	server       *httptest.Server
	mu           sync.Mutex
	mutationPOST int
	requestCount int
	sessions     map[string]int
	nextSession  int
	note         string
	priority     string
	enabled      bool
	archived     bool
}

func NewJourneyFixture(manifest Manifest) (*JourneyFixture, error) {
	if manifest.Suite != SuiteJourney || manifest.Journey == nil {
		return nil, fmt.Errorf("journey fixture requires a journey manifest")
	}
	fixture := &JourneyFixture{manifest: manifest, sessions: map[string]int{}, note: "Initial note", priority: "normal", archived: true}
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	return fixture, nil
}

func (fixture *JourneyFixture) Close()         { fixture.server.Close() }
func (fixture *JourneyFixture) Origin() string { return fixture.server.URL }

func (fixture *JourneyFixture) MutationPOSTs() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.mutationPOST
}

func (fixture *JourneyFixture) Requests() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.requestCount
}

func (fixture *JourneyFixture) SessionCount() int {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return len(fixture.sessions)
}

func (fixture *JourneyFixture) RecordState() (string, string, bool, bool) {
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	return fixture.note, fixture.priority, fixture.enabled, fixture.archived
}

func (fixture *JourneyFixture) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	fixture.mu.Lock()
	fixture.requestCount++
	fixture.mu.Unlock()
	switch {
	case request.URL.Path == "/ready":
		fixture.ready(writer, request)
	case request.URL.Path == "/catalog":
		fixture.catalog(writer)
	case request.URL.Path == "/catalog/results":
		fixture.catalogResults(writer, request)
	case request.URL.Path == "/orders/O-42":
		fixture.order(writer)
	case request.URL.Path == "/records/42" && request.Method == http.MethodGet:
		fixture.record(writer, request.URL.Query().Get("saved") == "1")
	case request.URL.Path == "/records/42/edit" && request.Method == http.MethodGet:
		fixture.recordEdit(writer)
	case request.URL.Path == "/records/42" && request.Method == http.MethodPost:
		fixture.updateRecord(writer, request)
	case request.URL.Path == "/workspace":
		fixture.sessionPage(writer, request)
	default:
		http.NotFound(writer, request)
	}
}

func (fixture *JourneyFixture) ready(writer http.ResponseWriter, request *http.Request) {
	if fixture.manifest.Journey.Kind == "session_lifecycle" {
		fixture.mu.Lock()
		fixture.nextSession++
		id := fixture.nextSession
		name := fmt.Sprintf("journey-%d", id)
		fixture.sessions[name] = id
		fixture.mu.Unlock()
		http.SetCookie(writer, &http.Cookie{Name: "journey_run", Value: name, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	}
	writeJourneyHTML(writer, "Journey ready", `<main><h1>Journey ready</h1><div role="status" aria-label="Application ready">Ready</div></main>`, "")
}

func (fixture *JourneyFixture) catalog(writer http.ResponseWriter) {
	body := `<main><h1>Product catalog</h1><form method="get" action="/catalog/results">
<label>Search <input aria-label="Search products" name="q" type="text"></label>
<label><input aria-label="In stock only" name="stock" type="radio" value="yes"> In stock only</label>
<label><input aria-label="Include archived" name="archived" type="checkbox" value="yes" checked> Include archived</label>
<label>Category <select aria-label="Category" name="category"><option value="all">All</option><option value="tools">Tools</option></select></label>
<button type="submit">Apply filters</button></form></main>`
	writeJourneyHTML(writer, "Product catalog", body, "")
}

func (fixture *JourneyFixture) catalogResults(writer http.ResponseWriter, request *http.Request) {
	page := request.URL.Query().Get("page")
	if page == "2" {
		writeJourneyHTML(writer, "Catalog page two", `<main><h1>Catalog page 2</h1><div role="status" aria-label="Page marker">Page two</div><article aria-label="Widget Gamma">Widget Gamma</article></main>`, "")
		return
	}
	if request.URL.Query().Has("q") && (request.URL.Query().Get("q") != "widget" || request.URL.Query().Get("category") != "tools" || request.URL.Query().Get("stock") != "yes" || request.URL.Query().Has("archived")) {
		writeJourneyHTML(writer, "Invalid filters", `<main><h1>Invalid filters</h1><div role="alert">The submitted filter state did not match the reviewed request.</div></main>`, "")
		return
	}
	body := `<main><h1>Filtered products</h1><div role="status" aria-label="Results ready">Results ready</div><div role="status" aria-label="Match count">2</div><article aria-label="Widget Alpha">Widget Alpha</article><a href="/catalog/results?page=2">Next page</a></main>`
	writeJourneyHTML(writer, "Filtered products", body, "")
}

func (fixture *JourneyFixture) order(writer http.ResponseWriter) {
	head := `<link rel="canonical" href="` + html.EscapeString(fixture.Origin()+"/orders/O-42") + `"><script type="application/ld+json">{"name":"Order O-42"}</script>`
	body := `<main itemscope><h1>Order O-42</h1><div role="status" aria-label="Order label">Order O-42</div><span itemprop="status">ready</span><div class="legacy-code">legacy-42</div></main>`
	writeJourneyHTML(writer, "Order O-42", body, head)
}

func (fixture *JourneyFixture) record(writer http.ResponseWriter, saved bool) {
	fixture.mu.Lock()
	note, priority := fixture.note, fixture.priority
	fixture.mu.Unlock()
	status := ""
	if saved {
		status = `<div role="status" aria-label="Saved">Saved</div>`
	}
	body := `<main><h1>Record 42</h1><a role="button" href="/records/42/edit">Edit record</a>` + status + `<div role="status" aria-label="Record note">` + html.EscapeString(note) + `</div><div role="status" aria-label="Record priority">` + html.EscapeString(priority) + `</div></main>`
	writeJourneyHTML(writer, "Record 42", body, "")
}

func (fixture *JourneyFixture) recordEdit(writer http.ResponseWriter) {
	extraSave := ""
	if fixture.manifest.Journey.Kind == "record_update_ambiguous" {
		extraSave = `<button type="submit">Save record</button>`
	}
	body := `<main><h1>Edit record 42</h1><div role="dialog" aria-label="Edit record"><form method="post" action="/records/42">
<label>Note <input aria-label="Note" name="note" type="text"></label>
<label><input aria-label="Enabled" name="enabled" type="radio" value="yes"> Enabled</label>
<label><input aria-label="Archived" name="archived" type="checkbox" value="yes" checked> Archived</label>
<label>Priority <select aria-label="Priority" name="priority"><option value="normal">Normal</option><option value="high">High</option></select></label>
<div role="status" aria-label="Ready to save">Ready to save</div><button type="submit">Save record</button>` + extraSave + `</form></div></main>`
	writeJourneyHTML(writer, "Edit record", body, "")
}

func (fixture *JourneyFixture) updateRecord(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid", http.StatusBadRequest)
		return
	}
	fixture.mu.Lock()
	fixture.mutationPOST++
	fixture.note = request.Form.Get("note")
	fixture.priority = request.Form.Get("priority")
	fixture.enabled = request.Form.Get("enabled") == "yes"
	fixture.archived = request.Form.Get("archived") == "yes"
	fixture.mu.Unlock()
	http.Redirect(writer, request, "/records/42?saved=1", http.StatusSeeOther)
}

func (fixture *JourneyFixture) sessionPage(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie("journey_run")
	if err != nil {
		http.Error(writer, "missing session", http.StatusUnauthorized)
		return
	}
	fixture.mu.Lock()
	id := fixture.sessions[cookie.Value]
	fixture.mu.Unlock()
	body := fmt.Sprintf(`<main><h1>Run workspace</h1><div role="status" aria-label="Run marker">Run %d</div><a href="/workspace?refresh=1">Refresh workspace</a></main>`, id)
	writeJourneyHTML(writer, "Run workspace", body, "")
}

func writeJourneyHTML(writer http.ResponseWriter, title, body, head string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(writer, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>%s</title>%s</head><body>%s</body></html>", html.EscapeString(title), head, body)
}

type journeyBlueprint struct {
	actions            []journeyAction
	workflow           []synthesize.BrowserScenarioAction
	inputs             []synthesize.BrowserScenarioInput
	values             map[string]any
	approvedOperations []string
	expectedOutputs    map[string]any
}

type journeyAction struct {
	id           string
	description  string
	parameters   []guide.ParameterIntent
	steps        []journeyStep
	outputs      []bevidence.CandidateOutput
	sideEffects  []profile.SideEffect
	confirmation profile.ConfirmationPolicy
}

type journeyStep struct {
	kind           profile.StepKind
	navigate       string
	locator        *bevidence.CandidateLocator
	valueParameter string
	waitLocator    *bevidence.CandidateLocator
	waitNavigation *profile.NavigationWait
}

func journeyScenarioBlueprint(manifest Manifest, origin string) (journeyBlueprint, error) {
	readOnly := []profile.SideEffect{profile.SideEffectReadOnly}
	noConfirmation := profile.ConfirmationPolicy{Required: false}
	switch manifest.Journey.Kind {
	case "catalog_search_filter":
		action := journeyAction{id: "filter_catalog", description: "Filter the product catalog with reviewed controls.",
			parameters: []guide.ParameterIntent{{Name: "query", Type: "string", Required: true}, {Name: "category", Type: "string", Required: true}},
			steps: []journeyStep{
				{kind: profile.StepNavigate, navigate: origin + "/catalog"},
				{kind: profile.StepTypeText, locator: jl("textbox", "Search products"), valueParameter: "query"},
				{kind: profile.StepCheckRadio, locator: jl("radio", "In stock only")},
				{kind: profile.StepUncheck, locator: jl("checkbox", "Include archived")},
				{kind: profile.StepSelectOption, locator: jl("combobox", "Category"), valueParameter: "category"},
				{kind: profile.StepClick, locator: jl("button", "Apply filters"), waitNavigation: nav(profile.NavigationDOMContentLoaded)},
				{kind: profile.StepWaitFor, locator: jl("status", "Results ready")},
			},
			outputs:     []bevidence.CandidateOutput{jo("match_count", "string", "a11y", jl("status", "Match count"), "", ""), jo("first_item", "string", "a11y", jl("article", "Widget Alpha"), "", "")},
			sideEffects: readOnly, confirmation: noConfirmation}
		return blueprint([]journeyAction{action}, map[string]any{"query": "widget", "category": "tools"}, []synthesize.BrowserScenarioAction{{Name: "filter", Operation: action.id, With: map[string]string{"query": "query", "category": "category"}}}, nil, map[string]any{"first_item": "Widget Alpha", "match_count": "2"}), nil
	case "catalog_pagination":
		open := journeyAction{id: "open_results", description: "Open the first result page.", steps: []journeyStep{{kind: profile.StepNavigate, navigate: origin + "/catalog/results"}, {kind: profile.StepWaitFor, locator: jl("status", "Results ready")}}, sideEffects: readOnly, confirmation: noConfirmation}
		next := journeyAction{id: "next_results", description: "Advance to the next result page.", steps: []journeyStep{{kind: profile.StepClick, locator: jl("link", "Next page"), waitNavigation: nav(profile.NavigationLoad)}, {kind: profile.StepWaitFor, locator: jl("status", "Page marker")}}, outputs: []bevidence.CandidateOutput{jo("page", "string", "a11y", jl("status", "Page marker"), "", "")}, sideEffects: readOnly, confirmation: noConfirmation}
		return blueprint([]journeyAction{open, next}, nil, []synthesize.BrowserScenarioAction{{Name: "open_results", Operation: open.id}, {Name: "next_results", Operation: next.id}}, nil, map[string]any{"page": "Page two"}), nil
	case "order_structured_read":
		action := orderReadAction(origin)
		return blueprint([]journeyAction{action}, map[string]any{"order_id": "O-42"}, []synthesize.BrowserScenarioAction{{Name: "read_order", Operation: action.id, With: map[string]string{"order_id": "order_id"}}}, nil, map[string]any{"legacy_code": "legacy-42", "micro_status": "ready", "order_label": "Order O-42", "schema_name": "Order O-42"}), nil
	case "record_update_approved", "record_update_unapproved", "record_update_ambiguous":
		action := recordUpdateAction(origin)
		approvals := []string(nil)
		if manifest.Journey.Kind != "record_update_unapproved" {
			approvals = []string{action.id}
		}
		return blueprint([]journeyAction{action}, map[string]any{"record_id": "42", "note": "Reviewed note", "priority": "high"}, []synthesize.BrowserScenarioAction{{Name: "update_record", Operation: action.id, With: map[string]string{"record_id": "record_id", "note": "note", "priority": "priority"}}}, approvals, map[string]any{"record_note": "Reviewed note", "record_priority": "high", "saved": "Saved"}), nil
	case "parameter_contract_rejected":
		action := journeyAction{id: "open_parameter_target", description: "Open one exact reviewed target.", parameters: []guide.ParameterIntent{{Name: "target_url", Type: "string", Required: true}}, steps: []journeyStep{{kind: profile.StepNavigate, navigate: "{{target_url}}"}, {kind: profile.StepWaitFor, locator: jl("heading", "Order O-42")}}, sideEffects: readOnly, confirmation: noConfirmation}
		return blueprint([]journeyAction{action}, map[string]any{"target_url": origin + "/orders/O-42"}, []synthesize.BrowserScenarioAction{{Name: "open_target", Operation: action.id, With: map[string]string{"target_url": "target_url"}}}, nil, map[string]any{}), nil
	case "session_lifecycle":
		open := journeyAction{id: "open_workspace", description: "Open the execution-local workspace.", steps: []journeyStep{{kind: profile.StepNavigate, navigate: origin + "/workspace"}, {kind: profile.StepWaitFor, locator: jl("status", "Run marker")}}, sideEffects: readOnly, confirmation: noConfirmation}
		refresh := journeyAction{id: "refresh_workspace", description: "Refresh the same named workspace.", steps: []journeyStep{{kind: profile.StepClick, locator: jl("link", "Refresh workspace"), waitNavigation: nav(profile.NavigationNetworkIdle)}, {kind: profile.StepWaitFor, locator: jl("status", "Run marker")}}, outputs: []bevidence.CandidateOutput{jo("run_marker", "string", "a11y", jl("status", "Run marker"), "", "")}, sideEffects: readOnly, confirmation: noConfirmation}
		return blueprint([]journeyAction{open, refresh}, nil, []synthesize.BrowserScenarioAction{{Name: "open_workspace", Operation: open.id}, {Name: "refresh_workspace", Operation: refresh.id}}, nil, nil), nil
	default:
		return journeyBlueprint{}, fmt.Errorf("unknown journey blueprint")
	}
}

func blueprint(actions []journeyAction, values map[string]any, workflow []synthesize.BrowserScenarioAction, approvals []string, outputs map[string]any) journeyBlueprint {
	types := map[string]string{}
	required := map[string]bool{}
	for _, action := range actions {
		for _, parameter := range action.parameters {
			types[parameter.Name] = parameter.Type
			required[parameter.Name] = parameter.Required
		}
	}
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	inputs := make([]synthesize.BrowserScenarioInput, 0, len(names))
	for _, name := range names {
		inputs = append(inputs, synthesize.BrowserScenarioInput{Name: name, Type: types[name], Required: required[name]})
	}
	return journeyBlueprint{actions: actions, workflow: workflow, inputs: inputs, values: values, approvedOperations: approvals, expectedOutputs: outputs}
}

func orderReadAction(origin string) journeyAction {
	return journeyAction{id: "read_order", description: "Read a reviewed order through portable extraction sources.", parameters: []guide.ParameterIntent{{Name: "order_id", Type: "string", Required: true}},
		steps: []journeyStep{{kind: profile.StepNavigate, navigate: origin + "/orders/{{order_id}}"}, {kind: profile.StepWaitFor, locator: jl("status", "Order label")}},
		outputs: []bevidence.CandidateOutput{
			jo("order_label", "string", "a11y", jl("status", "Order label"), "", ""),
			jo("schema_name", "string", "jsonld", nil, "name", ""),
			jo("micro_status", "string", "microdata", nil, "status", ""),
			jo("legacy_code", "string", "css", nil, "", ".legacy-code"),
		}, sideEffects: []profile.SideEffect{profile.SideEffectReadOnly}, confirmation: profile.ConfirmationPolicy{Required: false}}
}

func recordUpdateAction(origin string) journeyAction {
	return journeyAction{id: "update_record", description: "Update one reviewed local record.", parameters: []guide.ParameterIntent{{Name: "record_id", Type: "string", Required: true}, {Name: "note", Type: "string", Required: true}, {Name: "priority", Type: "string", Required: true}},
		steps: []journeyStep{
			{kind: profile.StepNavigate, navigate: origin + "/records/{{record_id}}"},
			{kind: profile.StepClick, locator: jl("button", "Edit record"), waitNavigation: nav(profile.NavigationDOMContentLoaded)},
			{kind: profile.StepTypeText, locator: jl("textbox", "Note"), valueParameter: "note"},
			{kind: profile.StepCheckRadio, locator: jl("radio", "Enabled")},
			{kind: profile.StepUncheck, locator: jl("checkbox", "Archived")},
			{kind: profile.StepSelectOption, locator: jl("combobox", "Priority"), valueParameter: "priority"},
			{kind: profile.StepWaitFor, locator: jl("status", "Ready to save")},
			{kind: profile.StepClick, locator: jl("button", "Save record"), waitNavigation: nav(profile.NavigationNetworkIdle)},
			{kind: profile.StepWaitFor, locator: jl("status", "Saved")},
		},
		outputs:     []bevidence.CandidateOutput{jo("saved", "string", "a11y", jl("status", "Saved"), "", ""), jo("record_note", "string", "a11y", jl("status", "Record note"), "", ""), jo("record_priority", "string", "a11y", jl("status", "Record priority"), "", "")},
		sideEffects: []profile.SideEffect{profile.SideEffectStateChange, profile.SideEffectUpdatesRecord}, confirmation: profile.ConfirmationPolicy{Required: true, Prompt: "Update record {{record_id}}?"}}
}

func jl(role, name string) *bevidence.CandidateLocator {
	return &bevidence.CandidateLocator{Role: role, Name: name}
}

func jo(key, kind, source string, locator *bevidence.CandidateLocator, property, selector string) bevidence.CandidateOutput {
	output := bevidence.CandidateOutput{Key: key, Type: kind, Source: source, Locator: locator, Property: property, Selector: selector}
	if source == "css" {
		output.FallbackReason = string(profile.FallbackNoA11yRegion)
	}
	return output
}

func nav(value profile.NavigationWait) *profile.NavigationWait { return &value }

func buildJourneyBundle(actions []journeyAction, origin string, at time.Time) ([]byte, error) {
	records := make([]bevidence.Record, 0, len(actions))
	for _, action := range actions {
		locators := []bevidence.CandidateLocator{}
		for _, step := range action.steps {
			if step.locator != nil {
				locators = appendUniqueJourneyLocator(locators, *step.locator)
			}
			if step.waitLocator != nil {
				locators = appendUniqueJourneyLocator(locators, *step.waitLocator)
			}
		}
		for _, output := range action.outputs {
			if output.Locator != nil {
				locators = appendUniqueJourneyLocator(locators, *output.Locator)
			}
		}
		record := bevidence.Record{Origin: origin, ObservationKind: bevidence.ObservationA11ySnapshot, ObservedAt: at.Add(-time.Hour).Format(time.RFC3339), ActionHint: action.id, CandidateLocators: locators, CandidateOutputs: action.outputs, RedactionStatus: bevidence.RedactionNotRequired, Provenance: bevidence.Provenance{Tool: "openudon-journey-fixture", Version: "1"}}
		normalized, err := (&bevidence.RawRecord{Record: record}).Normalize()
		if err != nil {
			return nil, err
		}
		records = append(records, normalized)
	}
	catalog, err := guide.NewCatalog(records)
	if err != nil {
		return nil, err
	}
	intents := make([]guide.ActionIntent, 0, len(actions))
	for _, action := range actions {
		recordIDs := journeyRecordIDs(catalog, action.id)
		if len(recordIDs) != 1 {
			return nil, fmt.Errorf("journey action evidence identity is ambiguous")
		}
		steps := make([]guide.StepIntent, 0, len(action.steps))
		for _, step := range action.steps {
			intent := guide.StepIntent{Kind: step.kind, Navigate: step.navigate, ValueParameter: step.valueParameter}
			if step.locator != nil {
				locatorID := journeyLocatorID(catalog, recordIDs[0], *step.locator)
				if step.kind == profile.StepWaitFor {
					intent.Wait = &guide.WaitIntent{LocatorID: locatorID}
				} else {
					intent.LocatorID = locatorID
				}
			}
			if step.waitLocator != nil || step.waitNavigation != nil {
				if intent.Wait == nil {
					intent.Wait = &guide.WaitIntent{}
				}
				intent.Wait.Navigation = step.waitNavigation
				if step.waitLocator != nil {
					intent.Wait.LocatorID = journeyLocatorID(catalog, recordIDs[0], *step.waitLocator)
				}
			}
			steps = append(steps, intent)
		}
		outputIDs := []string{}
		for _, output := range action.outputs {
			outputIDs = append(outputIDs, journeyOutputID(catalog, recordIDs[0], output.Key))
		}
		intents = append(intents, guide.ActionIntent{ID: action.id, Description: action.description, EvidenceIDs: recordIDs, Parameters: action.parameters, Sequence: steps, OutputIDs: outputIDs, SideEffects: action.sideEffects, ConfirmationPolicy: action.confirmation})
	}
	bundle, err := guide.Author(catalog, guide.Intent{Info: profile.Info{Title: "OpenUdon reviewed journey", Provider: "openudon-loopback", Origin: profile.Origins{origin}, LoginStateRequired: true}, ObservationKind: profile.ObservationAccessibilitySnapshot, Confidence: profile.ConfidenceHigh, ExpiresAfter: "P14D", Actions: intents}, at)
	if err != nil {
		return nil, err
	}
	return guide.MarshalDeterministic(bundle)
}

func appendUniqueJourneyLocator(values []bevidence.CandidateLocator, candidate bevidence.CandidateLocator) []bevidence.CandidateLocator {
	for _, value := range values {
		if journeyLocatorEqual(value, candidate) {
			return values
		}
	}
	return append(values, candidate)
}

func journeyLocatorEqual(left, right bevidence.CandidateLocator) bool {
	return left.Role == right.Role && left.Name == right.Name && left.Text == right.Text && left.Value == right.Value
}

func journeyRecordIDs(catalog *guide.Catalog, action string) []string {
	var ids []string
	for _, record := range catalog.Records {
		if record.SourceActionHint == action {
			ids = append(ids, record.ID)
		}
	}
	return ids
}

func journeyLocatorID(catalog *guide.Catalog, record string, locator bevidence.CandidateLocator) string {
	for _, candidate := range catalog.Locators {
		if candidate.RecordID == record && journeyLocatorEqual(candidate.Locator, locator) {
			return candidate.ID
		}
	}
	return ""
}

func journeyOutputID(catalog *guide.Catalog, record, key string) string {
	for _, candidate := range catalog.Outputs {
		if candidate.RecordID == record && candidate.Output.Key == key {
			return candidate.ID
		}
	}
	return ""
}

func stageJourneySources(ctx context.Context, exampleDir, bundlePath, authenticationPath string, at time.Time) (string, string, error) {
	discovery, err := elicitor.DiscoverAuthoringSourcesWithBrowser(ctx, exampleDir, "reviewed browser journey", nil, nil, []elicitor.BrowserSourceInput{{ID: "journey", Path: bundlePath}}, at)
	if err != nil {
		return "", "", err
	}
	if len(discovery.Plans) != 1 || len(discovery.BrowserReport.Candidates) != 1 {
		return "", "", fmt.Errorf("journey source import produced %d plans and %d candidates (rejected=%v ambiguous=%v)", len(discovery.Plans), len(discovery.BrowserReport.Candidates), discovery.BrowserReport.Rejected, discovery.BrowserReport.Ambiguous)
	}
	var capabilityPath string
	for _, plan := range discovery.Plans {
		data, err := elicitor.SourceMaterializationContent(plan, at)
		if err != nil {
			return "", "", err
		}
		if bytesContainJourneyEnvelope(data) {
			return "", "", fmt.Errorf("journey materialization retained private authoring envelope")
		}
		target := filepath.Join(exampleDir, filepath.FromSlash(plan.TargetPath))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return "", "", err
		}
		if strings.HasPrefix(plan.TargetPath, "browser-profiles/") {
			capabilityPath = target
		}
	}
	if capabilityPath == "" {
		return "", "", fmt.Errorf("journey source import omitted its canonical profile")
	}
	authenticationData, err := os.ReadFile(authenticationPath)
	if err != nil || len(authenticationData) == 0 || len(authenticationData) > scenarioCommandOutputLimit || bytesContainJourneyEnvelope(authenticationData) {
		return "", "", fmt.Errorf("journey authentication source is invalid")
	}
	authPath := filepath.Join(exampleDir, "browser-authentication", "journey.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o700); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(authPath, authenticationData, 0o600); err != nil {
		return "", "", err
	}
	return capabilityPath, authPath, nil
}

func bytesContainJourneyEnvelope(data []byte) bool {
	text := string(data)
	return strings.Contains(text, "browsertools.guided-authoring") || strings.Contains(text, `"review"`) || strings.Contains(text, `"decisions"`) || strings.Contains(text, `"spec"`)
}

func writeJourneyAuthentication(path, origin string, at time.Time) error {
	profileValue := map[string]any{
		"profile":         "uws.browser-authentication.1.1",
		"info":            map[string]any{"title": "OpenUdon local journey authentication", "applicationOrigins": []string{origin}, "authenticationOrigins": []string{origin}},
		"observationKind": "accessibility_snapshot",
		"evidence":        map[string]any{"learnedAt": at.Format(time.RFC3339), "source": "reviewed_local_journey_fixture"},
		"confidence":      "high", "expiresAfter": "P14D",
		"verification":    map[string]any{"lastVerifiedAt": at.Format(time.RFC3339), "successfulRuns": 1},
		"credentialSlots": map[string]any{},
		"flows": map[string]any{journeyAuthenticationFlow: map[string]any{
			"sequence": []any{map[string]any{"navigate": origin + "/ready"}, map[string]any{"wait_for": map[string]any{"locator": map[string]any{"role": "heading", "name": "Journey ready"}}}},
			"effects":  []string{"establishes_session"},
			"success":  map[string]any{"origin": origin, "path": "/ready", "locator": map[string]any{"role": "heading", "name": "Journey ready"}},
		}},
	}
	data, err := json.MarshalIndent(profileValue, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func writeJourneyData(path string, values map[string]any) error {
	file := hclwrite.NewEmptyFile()
	block := hclwrite.NewBlock("inputs", nil)
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		var value cty.Value
		switch typed := values[name].(type) {
		case string:
			value = cty.StringVal(typed)
		case bool:
			value = cty.BoolVal(typed)
		case int:
			value = cty.NumberIntVal(int64(typed))
		case float64:
			value = cty.NumberFloatVal(typed)
		default:
			return fmt.Errorf("unsupported journey input %s=%s", name, strconv.Quote(fmt.Sprint(typed)))
		}
		block.Body().SetAttributeValue(name, value)
	}
	file.Body().AppendBlock(block)
	return os.WriteFile(path, file.Bytes(), 0o600)
}
