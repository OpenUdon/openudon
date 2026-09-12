package ui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/uws/browserregistration"
	"github.com/mxschmitt/playwright-go"
)

// SyntheticRegistrationForm is a fixed loopback fixture, without private data.
const SyntheticRegistrationForm = `<!doctype html><html><body><form method="post" action="/registration-complete">
<div id="identity"><label>Email<input type="email" role="textbox" name="email" required></label><label>Password<input type="password" role="textbox" name="password" required></label>
<label>Contact name<input name="contact" required></label>
<label for="kind">Account kind</label><select id="kind" name="kind" required onchange="document.getElementById('company').hidden=this.value!=='business'"><option value="individual">Individual</option><option value="business">Business</option></select>
<div id="company" hidden><label>Company name<input name="company"></label></div>
<button type="button" onclick="document.getElementById('identity').hidden=true;document.getElementById('contact').hidden=false">Next</button></div>
<div id="contact" hidden><label>Phone<input type="tel" name="phone"></label><label>Product updates<input type="checkbox" name="updates"></label><label>Quantity<input type="text" inputmode="numeric" name="quantity" required></label><label>Ratio<input type="text" inputmode="decimal" name="ratio" required></label><button type="submit">Register</button></div></form></body></html>`

func typedRegistrationQualificationDraft(ctx context.Context, options RegistrationQualificationOptions, handler http.Handler, first Response) (Response, registrationDraftRequest, error) {
	bad := errors.New("typed registration qualification evidence")
	id := func(state Response, label string) string {
		if state.RegistrationAuthoring == nil || state.RegistrationAuthoring.Observation == nil {
			return ""
		}
		for _, c := range state.RegistrationAuthoring.Observation.Candidates {
			if c.Label == label && c.Matches == 1 {
				return c.ID
			}
		}
		return ""
	}
	preview := func(current Response, label, action string, option *string) (Response, error) {
		request := registrationAuthoringCommandRequest{Revision: current.Revision, RegistrationRevision: current.RegistrationRevision, Type: "preview", Confirmed: true,
			Preview: &registrationauthorsession.PreviewRequest{CandidateID: id(current, label), Generation: current.RegistrationAuthoring.Observation.Generation, Action: action, Option: option, Purpose: "public_form_preview"}}
		if _, err := registrationQualificationJSON(ctx, handler, http.MethodPost, "/api/v4/registration-authoring/command", request, http.StatusAccepted); err != nil {
			return Response{}, bad
		}
		return registrationQualificationWait(ctx, handler, "observation")
	}
	choice := "business"
	second, err := preview(first, "Account kind", "select", &choice)
	if err != nil {
		return Response{}, registrationDraftRequest{}, err
	}
	third, err := preview(second, "Next", "click", nil)
	if err != nil {
		return Response{}, registrationDraftRequest{}, err
	}
	yes, no := true, false
	zero, ten, one := float64(0), float64(10), float64(1)
	fields := map[string]browserregistration.InputSlot{
		"contact_name": {Type: "string", Label: "Contact name", Required: &yes},
		"account_kind": {Type: "string", Label: "Account kind", Required: &yes, Enum: []any{"individual", "business"}},
		"company":      {Type: "string", Label: "Company name", RequiredWhen: &browserregistration.InputCondition{Slot: "account_kind", Equals: "business"}},
		"phone":        {Type: "string", Label: "Phone", Required: &no}, "updates": {Type: "boolean", Label: "Product updates", Required: &no},
		"quantity": {Type: "integer", Label: "Quantity", Required: &yes, Minimum: &zero, Maximum: &ten}, "ratio": {Type: "number", Label: "Ratio", Required: &yes, Minimum: &zero, Maximum: &one},
	}
	draft := registrationDraftRequest{Title: "Synthetic typed registration", Provider: "Synthetic loopback", Confidence: "high", ExpiresAfter: "P30D", InputsReviewed: true, InputSlots: fields,
		CredentialSlots: []registrationDraftSlot{{Slot: "identifier", Kind: "identifier", Binding: "registration_identifier"}, {Slot: "password", Kind: "password", Binding: "reg_password"}},
		Flow: registrationDraftFlow{Name: "create_dedicated_test_user", Description: "Create one synthetic member through reviewed typed checkpoints.", ConfirmationPrompt: "Approve one synthetic registration.", Effects: []string{"creates_account", "requires_human_verification", "sends_verification"},
			Steps: []registrationDraftStep{
				{Type: "input_checkpoint", CheckpointID: "identity", Slots: []string{"identifier", "password", "contact_name", "account_kind", "company"}},
				{Type: "navigate", Navigate: options.InitialURL},
				{Type: "type_credential", Slot: "identifier", CandidateID: id(first, "Email")}, {Type: "type_credential", Slot: "password", CandidateID: id(first, "Password")},
				{Type: "fill_input", Slot: "contact_name", Control: "fill", CandidateID: id(first, "Contact name")},
				{Type: "fill_input", Slot: "account_kind", Control: "select", CandidateID: id(first, "Account kind")},
				{Type: "fill_input", Slot: "company", Control: "fill", CandidateID: id(second, "Company name")},
				{Type: "click", CandidateID: id(second, "Next")},
				{Type: "input_checkpoint", CheckpointID: "contact", Slots: []string{"phone", "updates", "quantity", "ratio"}},
				{Type: "fill_input", Slot: "phone", Control: "fill", CandidateID: id(third, "Phone")},
				{Type: "fill_input", Slot: "updates", Control: "check", CandidateID: id(third, "Product updates")},
				{Type: "fill_input", Slot: "quantity", Control: "fill", CandidateID: id(third, "Quantity")},
				{Type: "fill_input", Slot: "ratio", Control: "fill", CandidateID: id(third, "Ratio")},
				{Type: "submit", CandidateID: id(third, "Register")},
				{Type: "human_checkpoint", CheckpointKind: "email_verification"},
			},
			Success: registrationDraftSuccess{Origin: options.Origin, Path: "/registration-complete", Proof: registrationSuccessProofOperatorReviewedDeferred, OperatorReviewed: true, Locator: registrationDraftSuccessLocator{Role: "status", Name: "Registration complete"}},
		}, CallControls: registrationDraftCallControls{Approval: "browser_registration_submit", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately"},
	}
	return third, draft, nil
}

func (q *registrationBrowserQualification) inputDefinitions(d registrationDraftRequest) error {
	if len(d.InputSlots) == 0 {
		return nil
	}
	keys := make([]string, 0, len(d.InputSlots))
	for key := range d.InputSlots {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		q.stage = "draft_input_" + key
		field := d.InputSlots[key]
		if err := q.button("Add field definition").Click(); err != nil {
			return err
		}
		row := q.page.Locator("#registration-input-list > fieldset").Last()
		fill := func(name, value string) error { return row.Locator(`[data-input="` + name + `"]`).Fill(value) }
		choose := func(name, value string) error {
			return selectQualificationValue(row.Locator(`[data-input="`+name+`"]`), value)
		}
		if err := fill("slot", key); err != nil {
			return err
		}
		if err := fill("label", field.Label); err != nil {
			return err
		}
		if err := choose("type", field.Type); err != nil {
			return err
		}
		requirement := "optional"
		if field.Required != nil && *field.Required {
			requirement = "required"
		}
		if field.RequiredWhen != nil {
			requirement = "conditional"
		}
		if err := choose("required", requirement); err != nil {
			return err
		}
		choices := []string{}
		for _, choice := range field.Enum {
			choices = append(choices, fmt.Sprint(choice))
		}
		if err := fill("enum", strings.Join(choices, "\n")); err != nil {
			return err
		}
		if field.RequiredWhen != nil {
			if err := fill("condition_slot", field.RequiredWhen.Slot); err != nil {
				return err
			}
			if err := fill("condition_equals", fmt.Sprint(field.RequiredWhen.Equals)); err != nil {
				return err
			}
		}
		for name, value := range map[string]*float64{"minimum": field.Minimum, "maximum": field.Maximum} {
			if value != nil {
				if err := fill(name, fmt.Sprint(*value)); err != nil {
					return err
				}
			}
		}
		for name, value := range map[string]*int{"minLength": field.MinLength, "maxLength": field.MaxLength} {
			if value != nil {
				if err := fill(name, fmt.Sprint(*value)); err != nil {
					return err
				}
			}
		}
	}
	return q.page.Locator("#registration-inputs-reviewed").Check(playwright.LocatorCheckOptions{})
}
