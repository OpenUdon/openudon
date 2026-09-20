# UWS 1.9.1 Content-Trust Adoption Result

OpenUdon now accepts an optional operator-authored `content_trust` registry in
`workflows/intent.hcl` for reviewed source descriptions, operation outputs and
defaults, triggers, and external `main` workflow inputs. It lowers those
declarations to UWS 1.9.1. Declaration-free packages retain their prior version
and shape, including UWS 1.9.0 for otherwise unchanged browser 1.7 packages;
the LLM intent schema does not author trust declarations.

Package assessment explicitly runs the UWS analyzer when the registry is
present. Contained browser profiles use Browsertools M28 contracts. Stable
codes, analyzer severity, document paths, and fixed messages appear as
non-failing quality warnings and value-free review evidence. Runtime values,
content excerpts, resolver details, credentials, and private paths are not
copied. Ordinary validation, passing quality status, package approval,
trusted-runner authorization, executor reports, planning, and execution remain
unchanged. Provenance remains separate from capability: constrained attacker
input is still untrusted even when it cannot carry free text.

E12 passes the offline mail-to-LLM data, untrusted instruction,
model-output-to-authority, constrained control, trigger-default, unknown
entry/opaque extension, resolver failure/conflict, and declaration-free legacy
matrix. `make content-trust-qualification` binds UWS
`9e676eaa469e9168225a7dcee75eb309e3499637`, Browsertools
`75fd5c3ab81f904243f8c2650c61ba1cd8c00540`, and Udon M37
`207e7f163ff24603138953d82ee68d55e4345394`, then invokes M37's public
analyzer tests without adding Udon to OpenUdon's module dependencies.

The content-trust-qualified OpenUdon commit is
`cc378beec5bf268754a1020e61a354f4e8ebdd4d`. Complete workspace,
standalone, race, vet, fast release, tagged compatibility, offline evaluation,
strict documentation, module, secret, and diff gates pass. The broader
registration-oriented SaaS gate stopped at its unrelated historical
Browserdriver lock because the clean sibling is now `2122806`; this result does
not rewrite that lock. The cumulative review found no unresolved
P1/P2-or-higher issue.

The user subsequently authorized ordered publication. OpenUdon
`2c99fdea575ac7514520ae8081534773fafef44c` makes the public-module tests
hermetic in clean checkouts by using tracked or synthetic temporary fixtures
and an explicit synthetic Playwright driver directory; it does not change the
content-trust or production contract. Hosted Actions run `33104475699` passes
the public-module job and all six release-build jobs. No repository was tagged
or released, and no live provider, browser target, account, executor,
deployment, or W8M action occurred.
