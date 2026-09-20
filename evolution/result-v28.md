# Browser Workflow And Evidence Review Remediation Result

OpenUdon's v2 trusted-runner path now supports reviewed browser workflows. It
derives a value-free Browserdriver contract from the current package, includes
browser authentication bindings in the credential inventory, passes the full
Udon browser CLI surface to isolated local or Docker executors, embeds the
contract in run evidence, and makes external runners independently reproduce
it before execution. Credentials, MFA values, sessions, and browser runtime
implementation remain outside OpenUdon.

Browser authoring and synthesis now fail closed over canonical frame labels,
typed approval tuples, exact origins, compatible MFA subsets, stable private
executables, complete process groups, profile discriminators, bounded auth
fields, symbolic bindings, generic outputs, nested document order, strict
review files, safe registry text, and durable create-only staging. The shared
Browsertools producer is pinned at its M25 revision.

The deterministic browser scenario fixture now protects goal pages with a
random session and verifies password plus all eight MFA families server-side.
The shared v2 compatibility lock rejects dirty or revision-mismatched siblings
and checks the actual Playwright package and Chromium browser versions. This
makes a passing replay evidence of authenticated behavior rather than merely
successful navigation.

P03, A14, E07, and M75 record the closure. No release tag was created. A11.5
remains pending until a hosted sandbox-compatible runner records the required
Chromium UI pass.
