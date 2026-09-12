package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	"github.com/mxschmitt/playwright-go"
)

// This adapter is fixture-only: synthetic decisions enter through actual DOM
// controls and the shipped JavaScript. It never synthesizes a mutation request.
// Snapshot inspection remains supplemental to the browser journey.
type registrationBrowserQualification struct {
	server      *Server
	http        *httptest.Server
	pw          *playwright.Playwright
	browser     playwright.Browser
	page        playwright.Page
	failed      atomic.Bool
	stage       string
	failedStage string
}

func newRegistrationBrowserQualification(ctx context.Context, handler http.Handler) (*registrationBrowserQualification, error) {
	q := &registrationBrowserQualification{server: handler.(*Server)}
	q.http = httptest.NewUnstartedServer(handler)
	q.server.authority = q.http.Listener.Addr().String()
	q.server.origin = "http://" + q.server.authority
	q.http.Start()
	fail := func(stage string) (*registrationBrowserQualification, error) {
		_ = q.Close()
		return nil, errors.New("registration_ui_" + stage)
	}
	var err error
	q.pw, err = playwright.Run()
	if err != nil {
		return fail("playwright")
	}
	q.browser, err = q.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true), ChromiumSandbox: playwright.Bool(true)})
	if err != nil {
		return fail("browser")
	}
	q.page, err = q.browser.NewPage()
	if err != nil {
		return fail("page")
	}
	q.page.SetDefaultTimeout(15000)
	q.page.OnPageError(func(error) { q.failed.Store(true) })
	// Request interception is an application request allowlist, not a claim of
	// network-wide isolation. Workers and drivers retain their own origin guards.
	if err = q.page.Route("**/*", func(route playwright.Route) {
		if !strings.HasPrefix(route.Request().URL(), q.http.URL+"/") {
			q.failed.Store(true)
			if route.Abort() != nil {
				q.failed.Store(true)
			}
			return
		}
		if route.Continue() != nil {
			q.failed.Store(true)
		}
	}); err != nil {
		return fail("route")
	}
	if _, err = q.page.Goto(q.http.URL); err != nil {
		return fail("navigation")
	}
	if err = q.page.Locator(`input[name="code"]`).Fill(registrationQualificationCode); err != nil {
		return fail("access_code")
	}
	if err = q.button("Continue").Click(); err != nil {
		return fail("access_click")
	}
	if err = q.page.Locator("#registration-start").WaitFor(); err != nil {
		return fail("ready")
	}
	return q, nil
}
func (q *registrationBrowserQualification) Close() error {
	bad := q.failed.Load()
	if q.browser != nil {
		if q.browser.Close() != nil {
			bad = true
		}
	}
	if q.pw != nil {
		if q.pw.Stop() != nil {
			bad = true
		}
	}
	if q.http != nil {
		q.http.Close()
	}
	if bad || q.failed.Load() {
		return errors.New("registration_ui_teardown")
	}
	if q.failedStage != "" {
		return errors.New("registration_ui_" + q.failedStage)
	}
	return nil
}
func (q *registrationBrowserQualification) button(name string) playwright.Locator {
	return q.page.GetByRole("button", playwright.PageGetByRoleOptions{Name: name, Exact: playwright.Bool(true)})
}
func selectQualificationValue(locator playwright.Locator, value string) error {
	v := []string{value}
	_, err := locator.SelectOption(playwright.SelectOptionValues{Values: &v})
	return err
}
func (q *registrationBrowserQualification) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		r.Host = q.server.authority
		r.URL.Host = q.server.authority
		q.server.ServeHTTP(w, r)
		return
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
	if err != nil || len(data) > MaxRequestBytes {
		w.WriteHeader(400)
		return
	}
	path := r.URL.Path
	// The callback enters an entire reviewed form through the DOM before its
	// final request. Keep per-control waits short, but budget that full sequence
	// separately from the response wait so larger typed forms remain testable.
	response, err := q.page.ExpectResponse(func(url string) bool { return strings.HasSuffix(url, path) }, func() error { return q.act(path, data) }, playwright.PageExpectResponseOptions{Timeout: playwright.Float(120000)})
	if err != nil || q.failed.Load() || response.Request().Method() != http.MethodPost {
		q.failedStage = q.stage
		w.WriteHeader(500)
		return
	}
	body, err := response.Body()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(response.Status())
	_, _ = w.Write(body)
}
func (q *registrationBrowserQualification) act(path string, data []byte) error {
	q.stage = strings.ReplaceAll(strings.TrimPrefix(path, "/api/v4/"), "/", "_")
	fill := func(id, value string) error { return q.page.Locator(id).Fill(value) }
	click := func(id string) error { return q.page.Locator(id).Click() }
	check := func(id string) error { return q.page.Locator(id).Check() }
	switch path {
	case "/api/v4/registration-authoring/start":
		var request registrationAuthoringStartRequest
		if json.Unmarshal(data, &request) != nil {
			return errors.New("input")
		}
		if request.ProfileVersion != "" {
			if err := selectQualificationValue(q.page.Locator("#registration-profile-version"), request.ProfileVersion); err != nil {
				return err
			}
		}
		for _, v := range [][2]string{{"#registration-profile-id", request.ProfileID}, {"#registration-title", "Synthetic dedicated test registration"}, {"#registration-provider", "Synthetic loopback"}, {"#registration-url", request.URL}, {"#registration-origins", strings.Join(request.Origins, "\n")}} {
			if err := fill(v[0], v[1]); err != nil {
				return err
			}
		}
		return click("#registration-start")
	case "/api/v4/registration-authoring/command":
		var request registrationAuthoringCommandRequest
		if json.Unmarshal(data, &request) != nil {
			return errors.New("input")
		}
		switch request.Type {
		// These are closed fixture commands; no page text enters stage codes.
		case "preview":
			if request.Preview == nil {
				return errors.New("preview")
			}
			var candidate registrationauthorsession.Candidate
			for _, item := range q.server.registrationAuthoring.Observation.Candidates {
				if item.ID == request.Preview.CandidateID {
					candidate = item
				}
			}
			button := q.button("Preview " + candidate.Label)
			group := button.Locator("..")
			if request.Preview.Option != nil {
				if err := selectQualificationValue(group.GetByRole("combobox"), *request.Preview.Option); err != nil {
					return err
				}
			}
			if request.Preview.Checked != nil {
				if err := selectQualificationValue(group.GetByRole("combobox"), fmt.Sprint(*request.Preview.Checked)); err != nil {
					return err
				}
			}
			if err := group.GetByRole("checkbox").Check(); err != nil {
				return err
			}
			return button.Click()
		case "observe":
			return q.button("Observe current page").Click()
		case "navigate":
			if err := selectQualificationValue(q.page.GetByLabel("Navigation method"), request.Method); err != nil {
				return err
			}
			if err := q.page.GetByLabel("Exact navigation URL").Fill(request.URL); err != nil {
				return err
			}
			return q.button("Navigate without submitting").Click()
		case "draft":
			return q.draft(*request.Draft)
		case "review":
			q.stage = "review"
			if err := check("#registration-draft-confirmed"); err != nil {
				return err
			}
			return click("#registration-review")
		case "finish":
			q.stage = "finish"
			return click("#registration-finish")
		}
	case "/api/v4/browser-transactions/review":
		if err := check("#transaction-review-confirmed"); err != nil {
			return err
		}
		return click("#transaction-review")
	case "/api/v4/browser-transactions/prepare":
		if err := check("#transaction-prepare-confirmed"); err != nil {
			return err
		}
		return click("#transaction-prepare")
	case "/api/v4/browser-transactions/promote":
		if err := check("#transaction-promote-confirmed"); err != nil {
			return err
		}
		return click("#transaction-promote")
	case "/api/v4/round":
		var request roundRequest
		if json.Unmarshal(data, &request) != nil {
			return errors.New("input")
		}
		for i, answer := range request.Answers {
			locator := q.page.Locator(fmt.Sprintf("#frontier-answer-%d", i+1))
			tag, err := locator.Evaluate("element => element.tagName", nil)
			if err != nil {
				return err
			}
			if tag == "SELECT" {
				err = selectQualificationValue(locator, answer.Value)
			} else {
				err = locator.Fill(answer.Value)
			}
			if err != nil {
				return err
			}
		}
		return click("#round-submit")
	case "/api/v4/author/approve":
		if err := check("#review-confirmed"); err != nil {
			return err
		}
		return click("#approve-final")
	case "/api/v4/package/build":
		if err := check("#package-confirmed"); err != nil {
			return err
		}
		return click("#package-build")
	}
	return errors.New("unsupported_ui_operation")
}
func (q *registrationBrowserQualification) draft(d registrationDraftRequest) error {
	q.stage = "draft_ready"
	if err := q.page.Locator("#registration-draft-form").WaitFor(); err != nil {
		return err
	}
	slots := q.page.Locator(".registration-slot-row")
	for {
		n, err := slots.Count()
		if err != nil {
			return err
		}
		if n <= len(d.CredentialSlots) {
			break
		}
		if err := slots.Last().GetByRole("button").Click(); err != nil {
			return err
		}
	}
	for i, slot := range d.CredentialSlots {
		q.stage = fmt.Sprintf("draft_credential_%d", i)
		row := slots.Nth(i)
		if err := row.Locator(`[data-registration-slot="slot"]`).Fill(slot.Slot); err != nil {
			return err
		}
		if err := row.Locator(`[data-registration-slot="binding"]`).Fill(slot.Binding); err != nil {
			return err
		}
		if err := selectQualificationValue(row.Locator(`[data-registration-slot="kind"]`), slot.Kind); err != nil {
			return err
		}
	}
	steps := q.page.Locator(".registration-step-row")
	for {
		n, err := steps.Count()
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		if err := steps.Last().GetByRole("button", playwright.LocatorGetByRoleOptions{Name: "Remove step", Exact: playwright.Bool(true)}).Click(); err != nil {
			return err
		}
	}
	if err := q.inputDefinitions(d); err != nil {
		return err
	}
	for i, step := range d.Flow.Steps {
		q.stage = fmt.Sprintf("draft_step_%d_%s", i, step.Type)
		if err := q.page.Locator("#registration-add-step").Click(); err != nil {
			return err
		}
		row := steps.Nth(i)
		if err := selectQualificationValue(row.Locator(`[data-registration-step="type"]`), step.Type); err != nil {
			return err
		}
		if step.Type == "navigate" {
			if err := row.Locator(`[data-registration-step="navigate"]`).Fill(step.Navigate); err != nil {
				return err
			}
		} else if step.CandidateID != "" {
			if err := selectQualificationValue(row.Locator(`[data-registration-step="candidate"]`), step.CandidateID); err != nil {
				return err
			}
			if step.Slot != "" {
				if err := selectQualificationValue(row.Locator(`[data-registration-step="slot"]`), step.Slot); err != nil {
					return err
				}
			}
		}
		if step.Type == "input_checkpoint" {
			if err := row.Locator(`[data-registration-step="checkpoint_id"]`).Fill(step.CheckpointID); err != nil {
				return err
			}
			if err := row.Locator(`[data-registration-step="slots"]`).Fill(strings.Join(step.Slots, ",")); err != nil {
				return err
			}
		}
		if step.Type == "fill_input" {
			if err := selectQualificationValue(row.Locator(`[data-registration-step="control"]`), step.Control); err != nil {
				return err
			}
		}
		if step.Type == "human_checkpoint" {
			if err := selectQualificationValue(row.Locator(`[data-registration-step="checkpoint"]`), step.CheckpointKind); err != nil {
				return err
			}
		}
	}
	for _, effect := range d.Flow.Effects {
		q.stage = "draft_effects"
		if effect == "requires_human_verification" {
			if err := q.page.Locator("#registration-effect-human").Check(); err != nil {
				return err
			}
		}
		if effect == "sends_verification" {
			if err := q.page.Locator("#registration-effect-verification").Check(); err != nil {
				return err
			}
		}
	}
	for _, v := range [][2]string{{"#registration-flow-name", d.Flow.Name}, {"#registration-flow-description", d.Flow.Description}, {"#registration-confirmation-prompt", d.Flow.ConfirmationPrompt}, {"#registration-success-path", d.Flow.Success.Path}, {"#registration-success-name", d.Flow.Success.Locator.Name}} {
		if err := q.page.Locator(v[0]).Fill(v[1]); err != nil {
			return err
		}
	}
	if err := selectQualificationValue(q.page.Locator("#registration-success-origin"), d.Flow.Success.Origin); err != nil {
		return err
	}
	q.stage = "draft_success_role"
	if err := selectQualificationValue(q.page.Locator("#registration-success-role"), d.Flow.Success.Locator.Role); err != nil {
		return err
	}
	if err := q.page.Locator("#registration-success-reviewed").Check(); err != nil {
		return err
	}
	q.stage = "draft_build"
	missing, err := q.page.Evaluate(`() => { const d=collectRegistrationDraft(); if(!d.title || !d.expires_after) return 'metadata'; if(d.credential_slots.some(s=>!s.slot||!s.binding)) return 'credentials'; const i=d.flow.steps.findIndex(s=>s.type==='navigate'?!s.navigate:s.type==='input_checkpoint'?!s.checkpoint_id||!s.slots.length:!s.candidate_id&&s.type!=='human_checkpoint'); if(i>=0)return 'step_'+i; if(d.flow.steps.some(s=>['type_credential','fill_input'].includes(s.type)&&!s.slot))return 'slot'; if(!d.flow.success.origin || !d.flow.success.locator.role || !d.flow.success.locator.name)return 'success'; return ''; }`)
	if err != nil {
		q.stage = "draft_inspection"
		return errors.New("draft_inspection")
	}
	if missing != "" {
		q.stage = "draft_missing_" + fmt.Sprint(missing)
		return errors.New("draft_incomplete")
	}
	return q.button("Build canonical draft for review").Click()
}
