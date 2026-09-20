# Status A10 - Interactive Phase C Authoring And Review Shell

## Goal

Turn the A09 polling status shell into an accessible, revision-protected
authoring and review experience while retaining one loopback workspace for one
trusted operator and no execution authority.

## State

Complete.

## Dependencies

- A07 headless authoring engine and shared artifact writer.
- A08/A09 loopback transport, API v2, transactional mutations, and workspace
  drift protection.
- M70 adaptive iCoT v2 plus A01-A06/P01 reviewed source contracts.

## Task Ledger

| Item | State | Notes |
|---|---|---|
| A10.1 exact conflict inspection contract | `[+]` | Snapshots expose a sorted `write_conflicts` preflight derived from the prepared transaction without creating directories or mutating outputs. Every prepared entry is path-chain and file-type validated before removal/authorized-overwrite entries skip collision reporting, so unsafe paths and symlinks retain writer-consistent handling. |
| A10.2 accessible current-frontier controls | `[+]` | Separate embedded HTML/CSS/JavaScript renders each current question as a required labelled textarea with rationale, slot context, optional recommendation fill, invalid-state focus, keyboard-visible focus, live notices, responsive reflow, and no unsafe HTML insertion. |
| A10.3 complete revision-protected rounds | `[+]` | The shell posts every frontier answer with the exact rendered revision, pauses polling during mutation, never automatically retries a POST, installs successful responses directly, and leaves domain-rejected controls editable with the request ID visible. |
| A10.4 preview, conflict, and explicit approval | `[+]` | Selected sources, readiness, top issue, project/intent previews, proposed actions, and conflicts appear before approval. Review and overwrite acknowledgements are separate, and final versus explicitly incomplete approval sends exact independent flags. |
| A10.5 lifecycle reconciliation | `[+]` | Dirty stale state preserves unsent values until explicit adoption and archives displaced answers. A poll generation invalidates every snapshot request overlapping a mutation, including late responses after rejection or success. Retryable failure reconciles through a forced snapshot before offering explicit retry; workspace drift requires restart; indeterminate failure locks mutation; and frozen completion retains write results and cleanup warnings. |
| A10.6 provider-free contract coverage | `[+]` | Go tests cover exact conflict derivation, read-only preflight, symlink rejection, snapshot/revision inclusion, and embedded asset semantics while retaining A09 HTTP and direct-engine regressions. |
| A10.7 docs and operator guidance | `[+]` | README, CLI help, iCoT/operator docs, product/architecture/tech-stack/milestone memory, and evolution v25 describe the interactive Phase C contract and unchanged safety boundary. |
| A10.8 Phase C review remediation | `[+]` | The shared plan preflight now rejects reserved source targets from the final normalized/revalidated source plan and rejects duplicate, case-folded, ancestor/descendant, or remove/write collisions before mutation; a root-targeted output is distinguished from a genuinely out-of-root path. Workspace observation is bounded and streaming; successful mutations announce status and restore focus to the next question, proposal review, or completion. Regression coverage proves read-only rejection, candidate drift, cancellable hashing, precise path rejection, and the restored browser focus lifecycle. |

## Acceptance Criteria

- Browser authoring remains an experimental local view over the same
  transactional engine and API v2; the server remains loopback-only and
  single-workspace.
- The operator reviews exact preview/action/conflict state and explicitly
  chooses final or incomplete approval; no UI path executes a workflow.
- The production frontend remains plain embedded HTML, CSS, and JavaScript
  without React, Node, accounts, folder browsing, or live Browsertools
  orchestration.

## Verification

- Focused artifact-writer, engine, UI, CLI, and static embedded-asset tests
  pass.
- Full standalone, race, vet, repository, documentation, cross-build, and
  scorecard gates pass. The Phase C browser journeys pass with the documented
  local sandbox-disable override; hosted sandboxed evidence remains tracked as
  pending in A11.
