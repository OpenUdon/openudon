# Realistic Playwright-Browser Journey Qualification Result

OpenUdon `5a825a41b298b4cb589756adbdef812573fc7522` now has a third
`browser-scenario-eval` suite with eight realistic local cases. Each case
authors a deterministic
`browsertools.guided-authoring.v1` bundle from normalized evidence, re-imports
it through OpenUdon's strict source boundary, materializes only the canonical
browser 1.5 profile, synthesizes ordered parameterized UWS 1.8 operations, and
replays them through external Udon and Browserdriver v3 in real headless
Chromium.

The corpus covers a search/filter form, pagination, accessibility plus
JSON-LD/microdata/CSS extraction, an exactly approved record update, rejection
without approval, an ambiguous Save locator, missing/undeclared/wrong-type/
origin-escape parameters, and fresh isolation across complete executions.
Positive cases compare typed outputs and fixture state exactly; negative cases
prove the mutation endpoint was not reached.

The suite uses strict `openudon.browser-journey.v1` manifests and a dedicated
value-free `openudon.browser-journey-eval.v1` report with the existing exact
compatibility lock and SHA-256 sidecar. `make browser-scenario-journey` is
required by local SaaS release qualification and the tag release workflow. It
needs Browserdriver's pinned headless Chromium but no display, provider,
credential, or external network. The coordinated run passed 8/8 without any
upstream repository change.
