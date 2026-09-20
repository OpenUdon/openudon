# Complementary Real-Browser Scenario Evaluation

Add two deliberately separate suites for the implemented Playwright-browser
workflow. The release suite must be deterministic, external-network-free, and
exercise real Browsertools author-session v2 through OpenUdon staging, UWS
synthesis, Udon v3 lowering, and Browserdriver v3 replay. The public suite must
be an explicit-network, credential-free, read-only drift canary over a small
fixed anonymous inventory and must not gate releases.

Use strict embedded scenario manifests and an exact compatibility lock. Cover
all reviewed MFA kinds, contexts, locator modes, typed/presence outputs, bounds,
and closed negative cases. Retain only a digest-bound value-free report with
exact revisions and closed outcomes; never archive credentials, page content,
browser state, or subprocess output. Keep ordinary tests browser-free.
