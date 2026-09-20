# Interactive Phase C Authoring And Browser Qualification Result

OpenUdon A10 turns the embedded local shell into an accessible Phase C
authoring and review experience over the existing experimental API v2. The
whole current frontier is rendered as required labelled controls, optional
recommendations require an explicit fill action, and one round submission
carries every answer plus the exact displayed revision. Client-side omissions
focus the first missing answer, successful mutations install their returned
state directly, and POST requests are never automatically retried.

Snapshots now expose a sorted read-only `write_conflicts` preflight derived
from the exact prepared artifact transaction. The shell shows selected sources,
readiness, top issue, both artifact previews, proposed actions, and conflicts
before approval. Review acknowledgement is distinct from overwrite permission,
and separate final and incomplete buttons preserve the engine's exact approval
flags and frozen write result.

The client has explicit failure reconciliation. Domain rejection leaves the
form editable with its request ID; retryable failure performs a fresh snapshot
check and offers manual retry only when the revision is unchanged; stale state
keeps unsent answers until explicit adoption and archives displaced values;
workspace drift shows restart-required lockout; indeterminate failure locks
mutation; and completed state remains inspectable. Visible polling remains two
seconds with hidden pause, immediate visibility refresh, conditional requests,
and exponential failure backoff capped at 30 seconds.

A11 adds a build-tagged Playwright-Go v0.6201.0 Chromium suite against the real
loopback handler. It proves capability bootstrap, accessible names and keyboard
order, visible focus, complete rounds, preview/conflict approval, both
completion modes, lifecycle failures, polling timing, `304` behavior, and
360-pixel plus 200-percent layout. `make icot-ui-browser-check` is required by
the provider-free SaaS and tag release gates. Playwright remains test-only,
release Chromium is configured to stay sandboxed, and production remains
embedded plain HTML, CSS, and JavaScript with no expanded serving or execution
authority. A11 implementation is complete, but hosted sandboxed release-runner
verification remains pending; the restricted local sandbox-disabled run is not
recorded as hosted success.

The Phase C closure remains part of A10. Artifact preparation, conflict
inspection, and commit now share one read-only plan validator that runs before
directory, temporary-file, or backup creation. Source materialization cannot
target `.icot/**`, `project.md`, or either intent path, and ambiguous duplicate,
case-folded, ancestor/descendant, or remove/write plans fail without filesystem
residue. Engine mutation observation no longer walks the workspace: it covers
only watched paths and current local/registry candidate targets, streams
cancellable SHA-256 while checking stable file identity and metadata, and
treats newly produced unobserved targets as missing. The shell announces
mutation progress/results politely and focuses the next question, proposal
review, or completion after success while preserving all existing failure and
polling behavior.
