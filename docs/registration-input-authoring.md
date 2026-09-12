# Registration 1.1 in iCoT

Start `icot ui` with an example workspace, private producer root, package scope,
scratch directory and package store. In registration authoring, choose profile
version 1.1. Observe the public form, review suggested definitions, and declare
each ordinary field's type, requirement, public choices and bounds. Conditional
requirements reference a required public choice field. Credential bindings
remain symbolic.

Review each public preview before selecting a choice or navigating a wizard.
The producer permits only observed choices and reviewed transitions under its
GET/HEAD guard. Preview cannot grant consent, answer verification, fill private
values or submit an account. Observations from successive generations remain
available for exact step selection.

Begin the recipe with an input checkpoint. Add native credential and ordinary
fill/check/select actions, reviewed navigation/click transitions, later named
input checkpoints, one submit, and the exact success proof. A deferred success
proof requires explicit operator review. Checkpoints require the profile's
human-verification effect under the published UWS 1.1 validation rules.

Review the BRP, its transaction and the UWS package separately, then promote the
package. The UWS call carries the selected flow and a symbolic `inputBinding`.
Actual values are entered only in the separate Udon private runtime form.
Initial Start is untimed; later Apply and Stop retain the running deadline.
Account creation still requires the exact reviewed attestation and submit
approval. `openudon run --browser-registration-input-service URL` can connect
to a prepared Udon form using private canonical token and expected-identity
environment variables. No private input endpoint is proxied through iCoT.

A consumer can start `icot ui` or `icot control` with
`--registration-authority FILE`. The private authority fixes profile/version,
initial URL, exact origins, allowed navigation URLs and a maximum twenty-minute
expiry. The shared application enforces those constraints before consuming its
one authoring attempt. HTTP shutdown joins worker teardown just as control does.

The existing BRP 1.1 vocabulary supports scalar inputs and one submit. Files,
arbitrary widgets, multi-POST wizards and native spinbutton locators need future
public contracts. Numeric values can use supported textbox locators. W8M is a
consumer test case; the editor and runtime contain no W8M-specific rules.
