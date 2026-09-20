# A15 Engine Acquisition Lifecycle

Item | State | Notes
--- | --- | ---
A15.1 Journey starter evidence | `[+]` | `api`, `existing_account_sign_in`, `authenticated_action`, `existing_reviewed_capability`, and `freeform_mixed` plus the operator goal persist through the engine as human decisions; sessions created before starters remain valid.
A15.2 Private-root authority | `[+]` | Upload and browser capture require an absolute, non-symlink, mode-0700 root outside the example; ordinary authoring remains usable without a private root.
A15.3 API upload admission | `[+]` | The private inbox accepts at most 20 MiB, uses Apitools to require exactly one unambiguous supported source family, rejects secret-like content, and reports the canonical package target before staging.
A15.4 API source stage and removal | `[+]` | Revision-protected staging is atomic and collision-safe, refreshes discovery in the same mutation, and records digest ownership so removal rejects drift and non-UI files.
A15.5 Browser profile-pair staging | `[+]` | Independently reconstructed authentication and capability profiles are staged create-only under collision-free names, support multiple captures, and refresh discovery without selecting the final workflow source, flow, action, session, or approval.
A15.6 Safe review collection | `[+]` | `openudon.authenticated-authoring-review.v3` is append-only and bounded to 128 entries; the next stage migrates one valid v2 singleton deterministically while malformed, duplicate, changed, or colliding records fail closed.
A15.7 Failure and restart coverage | `[+]` | Focused engine/import tests cover private-root policy, secrets, ambiguity, upload cleanup, staging collisions, removal drift, restart, v2 migration, a genuine second profile, review-file workspace drift, and non-overwrite behavior.
A15.8 Unified source refresh | `[+]` | Interactive, complete-session, agent, progressive, restart, and engine/UI flows derive local discovery, inactive/ambiguous/truncated blockers, registry triggers, selected-registry digest revalidation, source-plan synchronization, and verification attachment from one refresh routine while retaining surface-specific error/readiness presentation.
A15.9 Step-scoped browser authority | `[+]` | Every browser step receives its own source/action frontier, symbolic external session name when login state is otherwise opaque, mutation approval, and revision clearing; collision-free exact question IDs and exact slot parsing prevent distinct or substring-shaped step names from intercepting other answers.

Implementation and focused regression coverage are complete in the coordinated
workspace. No registration, profile replacement/relearning, browser-state
transfer, credential intake, or trusted execution was added.
