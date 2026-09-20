# Prompt V13 - Additive Browser Authentication

Extend API-first browser fallback authoring so a reviewed UI-only workflow may
include an explicit website sign-in and MFA flow before a protected browser
action. Consume the public `uws.browser-authentication.1.0` and
`uws.browser-authentication-call.1.0` supplements plus Browsertools local
validation. Preserve existing `uws.browser.1.5` workflows and legacy opaque
session binding.

OpenUdon owns only package-local profile/flow selection, symbolic credential
mapping, named-session intent, bounded timeout, exact authoring approval, safe
review evidence, UWS lowering, quality, and handoff inventory. Udon and its
private Browserdriver retain credentials, MFA interaction, live sessions,
driver lifecycle, and separate runtime approval. Do not add membership,
authentication-profile publication, enrollment, recovery, password changes,
consent, logout, account creation, or CAPTCHA solving.
