# Cross-Package Browser-Profile Transactions

Unify OpenUdon's browser-authentication, browser-capability, and browser-
registration authoring around one public, value-free transaction contract
without allocating a new UWS discriminator or moving live browser/runtime
authority into OpenUdon.

Preserve Browsertools' existing authenticated author-session v2 contracts and
add a separate no-submit, exact-origin, GET/HEAD-only registration producer.
Adopt its private reviewed candidates through stable digest/provenance checks;
compose authentication plus capability profiles with one execution-local named
session while registration remains session-free and fails before executor
invocation.

Separate pure package preparation from restrictive scratch qualification and
atomic promotion, then expose the same typed lifecycle through the engine,
experimental loopback UI API v4, accessible UI, and compatible terminal flow.
Prove the seams with value-free loopback and adversarial evidence, publish only
the reviewed Browsertools producer commit, keep OpenUdon local, leave UWS
unchanged, and finish with an offline-only private W8M reconciliation. Defer
this prompt's result until every milestone through W01 passes.
