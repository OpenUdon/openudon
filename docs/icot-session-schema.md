# iCoT Session And Answers Files

`cmd/icot --answers <file>` accepts YAML or JSON. The preferred shape is an iCoT session because it
can carry both the human project brief and the structured `workflows/intent.hcl` contract. The legacy
answers shape is still accepted for older scripts, but it cannot fully describe operation IDs,
request mappings, step bindings, or typed API source documents.

## Preferred Session Shape

Use this shape when you want `icot` to resume or render a mostly complete workflow without asking for
details that are already known:

```yaml
project:
  project_name: Weather Toronto Gmail
  goal: Resolve Toronto weather and send a reviewed Gmail report.
  side_effect_scope: sandbox-only
  credentials:
    - weather_appid
    - gmail_oauth_token
  safety: Generate and validate artifacts only; Gmail send requires approved sandbox credentials.
  fallback: Stop if geocoding, weather lookup, report rendering, or Gmail send fails.
intent:
  source: openapi/openweathermap-one-call-3-overlay.json
  workflow:
    name: weather_toronto_gmail
    description: Resolve Toronto weather and send a reviewed Gmail report.
  input:
    - name: recipient_email
      type: string
      required: true
  step:
    - name: openweathermap
      type: http
      source: openapi/openweathermap-one-call-3-overlay.json
      operation: getOpenWeatherMapOneCall3
      with:
        lat: geocode.received_body[0].lat
        lon: geocode.received_body[0].lon
        appid: credentials.weather_appid
    - name: gmail
      type: http
      source: google-discovery/gmail-discovery-v1.json
      operation: gmail_users_messages_send
      with:
        userId: me
        raw: render_report.received_body.raw
  output:
    - name: result
      from: gmail.received_body
credentials:
  - weather_appid
  - gmail_oauth_token
credentials_set: true
safety: Generate and validate artifacts only; Gmail send requires approved sandbox credentials.
safety_set: true
fallback: Stop if any API step fails.
fallback_set: true
side_effect_scope: sandbox-only
```

The session fields mirror OpenUdon's authoring state:

- `project`: human-facing `project.md` values.
- `intent`: structured workflow intent rendered to `workflows/intent.hcl`.
- `credentials`: symbolic credential binding names only.
- `credentials_set`, `safety_set`, `fallback_set`: markers that preserve explicit user answers.
- `side_effect_scope`: one of `read-only`, `sandbox-only`, or `after-approval`.
- `annotations`, `assumptions`, `classifications`: optional review evidence produced by iCoT.
- `decision_evidence`: compact user-visible rationale and confidence for selected sources,
  operations, mappings, outputs, side-effect scope, and flow-review findings. It is not hidden
  model chain-of-thought.

Decision evidence entries use:

```yaml
decision_evidence:
  - stage: request_mapping
    slot: steps.gmail.with.raw
    value: render_report.received_body.raw
    source: deterministic
    confidence: review
    reason: Flow review suggested connecting Gmail raw to the rendered report.
    evidence: Gmail send step must consume the report content.
    requires_confirmation: true
```

Valid confidence values are `high`, `review`, `low`, and `conflict`. `normal` and `fast` prompt
modes may auto-accept `high` and `review` defaults, but `low` and `conflict` evidence forces an
operator question.

In `--agent` mode, `low` and `conflict` evidence prevents final artifact writes and appears as a
`needs_input` report instead of prompting interactively.

Do not put credential values, API tokens, OAuth refresh tokens, or private endpoints in this file.

## Legacy Answers Shape

Legacy answer files seed only the human project brief:

```yaml
project_name: Weather Toronto Gmail
goal: Resolve Toronto weather and send a reviewed Gmail report.
inputs: "recipient_email:string"
outputs: "result from gmail.received_body"
data_flow: "Resolve Toronto coordinates; use them for weather; send report through Gmail."
function_contracts: "render_report: inputs weather response; outputs Gmail raw message."
uses_openapi: true
openapi: openapi/openweathermap-one-call-3-overlay.json
cmd_approved: false
ssh_approved: false
side_effect_scope: sandbox-only
credentials:
  - weather_appid
  - gmail_oauth_token
safety: Generate and validate artifacts only; Gmail send requires approved sandbox credentials.
fallback: Stop if any API step fails.
```

When `uses_openapi` is true, a legacy file normally still needs interactive completion because it has
no native place for operation IDs, typed API source refs, request mappings, or step dependencies.
