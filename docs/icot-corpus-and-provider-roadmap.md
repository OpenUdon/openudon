# iCoT Corpus And Provider Roadmap

This note records the next iCoT reliability direction without assigning it to a
milestone. The focus is iCoT itself: better natural-language coverage, broader
provider evidence, and stronger provider catalog coupling before adding another
agent integration surface.

iCoT should continue to be described as a review-first authoring assistant. Its
output is a candidate `project.md` and `workflows/intent.hcl`; executability is
only established after `icot lint`, `openudon build`, `openudon assess`, and the
normal review/handoff gates pass.

## Reliability Goal

The useful product claim is not that iCoT can convert arbitrary language into a
safe executable workflow. The target claim is narrower:

> iCoT can produce reviewable API-backed workflow intents from realistic operator
> language when local provider metadata, source artifacts, or curated examples
> are available, and it reports blockers instead of guessing when they are not.

The corpus should measure that claim directly.

## Corpus Expansion

The current eval corpus is strong for seed/build and curated fixture coverage,
but it does not yet represent enough natural-language variation. Each important
fixture should grow from one canonical brief to a small family of realistic
operator phrasings.

For each fixture, keep variants across these dimensions:

| Dimension | Examples |
| --- | --- |
| Direct command | "Post this release note to Slack." |
| Business phrasing | "Notify the incident channel that the release passed smoke." |
| Provider-as-verb phrasing | "Slack the sandbox channel with this message." |
| Output expectation | "Return the Slack timestamp and channel." |
| Safety phrasing | "Prepare this for sandbox approval only." |
| Missing detail | "Send the report to the team" without a recipient or channel. |
| Negative/unsafe wording | "Use my token from this prompt" or "skip review and send it." |

Each variant should record:

- the natural-language input;
- expected provider family or explicit blocker;
- expected operation family when known;
- required user clarification, if any;
- whether `intent.hcl` should be buildable;
- allowed failure family for intentionally incomplete or unsafe inputs.

## Provider Coverage Priorities

Prioritize providers that exercise different API and policy shapes rather than
only adding popular brand names.

| Provider family | Why it matters |
| --- | --- |
| Slack | Message post, channel input, bearer auth, response metadata. |
| Gmail | OAuth-style send, helper-rendered body, side-effect review. |
| OpenWeatherMap | API key query auth, read-only data, prework geocoding. |
| Jira | Issue creation and lookup, multi-service incident workflows. |
| Google Drive | File upload/archive behavior and OAuth policy. |
| PagerDuty | Incident/contact lookup and operational alert workflows. |
| Trello | Board/list/card style resource identifiers. |
| Airtable/HubSpot | Advisory n8n-derived patterns and pagination/list cases. |
| Generic internal APIs | Header keys, bearer auth, path/query/body mapping, local stubs. |

Provider additions should include both source artifacts and expected iCoT
behavior. A provider with no usable source artifact is still useful if the
expected behavior is a clear blocker or capability-gap intent.

## Catalog Coupling

iCoT should rely more on `apitools` metadata before invoking an LLM or asking
the operator to guess.

High-value improvements:

- rank provider candidates from natural-language mentions and known aliases;
- materialize canonical source artifacts when available;
- prefer reviewed advisory overlays when both upstream and advisory artifacts
  exist;
- expose compact operation details: operation ID, summary, required fields,
  security schemes, request body shape, and response fields;
- reject unknown operation IDs and fields before draft generation;
- make "no catalog/source artifact available" a first-class blocker.

The LLM should receive a constrained menu of provider artifacts and operation
details. It should not invent source files, operation IDs, credential bindings,
or response paths.

## Measurement

Every corpus run should be summarized as a pipeline:

```text
natural language
  -> iCoT draft
  -> intent parse
  -> icot lint
  -> openudon build
  -> openudon assess
```

Record the first failing family:

| Failure family | Meaning |
| --- | --- |
| `missing_api_source` | No local or catalog source artifact is available. |
| `missing_operation` | No selected operation can satisfy the requested action. |
| `bad_request_mapping` | Required path/query/header/body values are missing or invalid. |
| `bad_response_path` | Output or downstream binding references unavailable response data. |
| `credential_binding_gap` | Credential binding does not match provider security metadata. |
| `side_effect_policy_gap` | Project policy does not approve the requested side effect. |
| `ambiguous_user_intent` | The operator request needs a clarification before drafting. |
| `runtime_profile_gap` | The task needs a runtime/profile capability not approved locally. |

The release-quality metric should separate:

- deterministic seeded/reference pass rate;
- natural-language variant pass rate;
- required clarification rate;
- safe blocker rate;
- unsafe false-positive rate.

A safe blocker is a good outcome when source metadata or user intent is
insufficient. The dangerous failure is a generated intent that looks executable
but binds the wrong operation, field, credential, or side effect.

Use `icot scorecard` for the provider-free baseline. Use `icot --agent --json`
and `icot lint --json` for per-example structured authoring and lint reports.
Use `icot repair --dry-run --json` to inspect bounded repair candidates before
allowing edits.

## Acceptance Shape

Before promoting this as product readiness evidence, iCoT should show:

- broad natural-language variants for the major provider families;
- clear pass/fail/blocker summaries by provider and failure family;
- zero known unsafe false positives in the curated negative set;
- stable seeded/reference behavior through the existing seed/build matrix;
- provider catalog retrieval and operation ranking evidence for catalog-backed
  providers;
- documented cases where iCoT correctly refuses to guess.

## Non-Goals

- Do not add MCP or another agent integration surface as part of this work.
- Do not treat live provider execution as an iCoT reliability gate.
- Do not move provider execution, credential resolution, or SDK behavior into
  iCoT.
- Do not claim arbitrary natural-language workflow generation until measured
  corpus evidence supports it.
