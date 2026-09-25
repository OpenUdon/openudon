# Status E22 — Browser 1.10 campaign-row count workflow

**State:** Active. Approved planning; implementation is pending.

**Goal.** Add a synthetic OpenUdon workflow that returns a bounded count of
rendered campaign rows from the default first action=topics page using Browser
1.10 selector match-count output.

**Dependencies.** Published UWS M05, Browsertools M32, Browserdriver M15 and Udon
M43 exact reviewed revisions.

**Compatibility.** Preserve E21 current-stack locks and report readers v1–v3.
Add a current-stack report v4 reader. No live target operation, runtime
adoption, registration, deployment, or campaign mutation is included.

## Tasks

| Item | State | Notes |
| --- | --- | --- |
| E22.1 Pin published Browser 1.10 dependency chain | [ ] | Require clean exact commits and matching module/profile/protocol pins before qualification. |
| E22.2 Add isolated count workflow and current-stack scenarios | [ ] | Emit only campaign_count in the range 0–100; cover zero, one, multiple, missing, ambiguous, invalid and over-bound cases. |
| E22.3 Preserve old reports and add strict versioned reader | [ ] | Keep v1–v3 readers and their lock semantics unchanged; independently reject malformed or cross-version evidence. |
| E22.4 Verify, qualify and review | [ ] | Run focused checks and current-stack synthetic qualification; verify report bindings and source cleanliness; complete bounded review before publication. |
