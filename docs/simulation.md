# Pure workflow simulation

`openudon simulate` previews the captured UWS package using UWS's public
orchestrator and `mockruntime.Runtime`. It makes no network requests, resolves
no account or environment credentials, invokes no executor, and writes no
package artifacts. Approval and execution remain separate commands.

```bash
openudon simulate --example examples/my-package --input preview-input.json
openudon simulate --example examples/my-package --fixtures fixtures.json
```

Run the CLI from a repository root containing the package. The package must
have its normal review handoff and required artifacts. Stored quality may
fail for an unresolved package; simulation does not turn that quality into a
pass. Captured HCL and YAML must agree. Unsafe paths, inconsistent artifacts,
unsupported expressions, unavailable responses and exceeded bounds produce
a versioned `blocked` result and a nonzero exit code.

The optional input uses `openudon.simulate-input.v1`:

```json
{
  "version": "openudon.simulate-input.v1",
  "inputs": {"city": "Toronto"},
  "responses": {
    "get_weather": {"example": {"statusCode": 200, "body": {"temperature": 18}}}
  }
}
```

Responses are keyed by operation ID for bound steps and by original step ID
for pending steps. Examples use the response envelope expected by workflow
expressions, such as `body` for `$response.body`. Alternatively, `schema`
contains a public UWS ParamSchema for the whole response envelope. API
metadata is never fetched or guessed here; bound steps need an explicit
fixture, example or schema. UWS owns supported expression evaluation and
bounded deterministic schema synthesis. Unsupported formats, references and
composed schemas need an explicit example.

Fixtures use public UWS Mock Fixture Format 1.0. An exact operation/request
match wins over examples and synthesis. A fixture miss refuses unless
`--allow-generated-fallback` is explicitly supplied. Request digests used for
matching stay in memory and never appear in the exported preview.

## Pending and browser steps

Pending contracts receive synthetic operations **only in memory**, with mock
outputs from their declared schema. Reports preserve original step IDs,
`binding: pending`, `hypothetical: true`, and `endpoint_status: unresolved`.
They contain no invented endpoint or operation ID for a pending step. The
projection is never saved, approved or passed to an executor. The original
package remains unresolved, so all approval and run paths continue to refuse
it. Unselected or unused steps remain `not_started` rather than being reported
as successful.

Browser results are labeled `mocked contract only; page state not verified`.
Simulation does not launch Chromium or verify a captured page, selector,
snapshot or replay.

## Result and privacy

`openudon.simulate.v1` has tier `simulation`, status `completed` or `blocked`,
the captured package digest, `package_unchanged`, per-step binding/effect/
simulation outcome and response provenance, and bounded `would_be_requests`.
A simulation outcome is never a real-run outcome or evidence of an external
write. `fixture`, `example`, `synthesized` and `unavailable` identify response
selection; effect labels remain descriptive.

All scalar preview values **and object field names** are redacted; only
bounded object/array shapes are shown. Step/operation identifiers, effects,
provenance and diagnostics retain their declared contract meaning. This
conservative policy prevents private inputs, inline credentials, response
values and private field names from entering exported evidence. Credential
references use symbolic placeholders inside the projection; no resolver is
called. Fixed diagnostics never include raw validation or runtime errors.

Limits: input JSON 256 KiB, public fixtures 16 MiB, captured package 32 MiB
and 1,024 files, each captured file 8 MiB, 256 inventoried steps, 128 mock
requests, each request preview 4 KiB, each output preview 2 KiB and total result
512 KiB. Computation has a five-second context; final package verification has
its own five-second read bound. Cancellation does not grant action authority.
A changed or unavailable final package blocks the report. These checks verify
captured package bytes and digest, not approval validity.
