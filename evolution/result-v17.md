# Result V17 - Authenticated Goal-Directed Browser Authoring

OpenUdon now exposes the explicit `icot browser-author live` command. It
reviews a typed continuation and goal predicate, preserves API preference,
starts an absolute Browsertools binary with a minimal child environment,
strictly consumes reduced `browsertools.author-session.v1` observations, and
requires disclosure, origin/action/POST, authentication, typed-plus-human
completion, and final staging gates that `--yes` cannot bypass. Browsertools
alone owns the non-persistent Playwright-Go context and human-entered
credentials/MFA.

The private digest-bound Browsertools envelope is stable-read and independently
validated for time, bounds, origins, contexts, trace, goal proof, reviews,
freshness, schemas, and secret absence. It remains outside the package; one
atomic approval stages only canonical authentication/capability profiles and
safe `.icot` metadata. Synthesis keeps old main-page sources on UWS 1.7/call
1.0 and selects UWS 1.8/call 1.1 for the additive context/path contracts.

The completed cross-repository path is bound to clean commits OpenUdon
`ea81570`, Browsertools `b73e1d4`, UWS `73e8aa7`, Udon `89eac9b`, and
Browserdriver `d7d2ca1`. The provider-free matrix passed all 11 required gates;
the three unrequested installed/headed loopbacks skipped honestly. Full
workspace and standalone Go tests/vet, strict docs, repository checks,
Browserdriver npm tests/audit, matrix verification, and diff checks passed.
Published module pins remain on reachable compatibility revisions until the
new upstream commits are published; the integrated workspace uses no committed
local replacement.
