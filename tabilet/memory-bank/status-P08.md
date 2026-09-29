# Status P08 — Browser 1.10 v11 trusted execution handoff

**State:** Complete locally after bounded review iteration 1. P08.1 and P08.2
are complete. P07 remains complete
with its original evidence and counters. W8M W24.5 remains blocked with no
accepted campaign count.

**Finding and baseline.** W8M W24.5 attempt 02 selected OpenUdon P07 source
`87df787c7737cc669f98c3b4462d7151db3e6b68` and produced native
`invalid_response`: `browser 1.5 through 1.9 is required by persistent driver
protocol v10`. The protected campaign page was served after sign-in, but Udon
rejected the Browser 1.10 count action before extraction. The attempt and
claim remain failed and consumed. This finding is confirmed at OpenUdon
`ecc3e7b00bcc60bf0107b3e3e60fa18e60ed6bb9` with a clean P07 branch:
trusted-runner dispatch selects v10 for Browser 1.10 and run-config validation
does not accept v11. Local severity P2. No prior P07 result is rewritten.

**Scope and ownership.** OpenUdon owns profile-to-protocol selection and its
external executor run-config. Udon already accepts Browser 1.10 under v11;
UWS and Browserdriver profile/count contracts remain unchanged. W8M owns the
separate source binding, synthetic qualification, adoption and any newly
authorized live count. P08 does not grant a browser or target operation.

| Item | State | Notes |
| --- | --- | --- |
| P08.1 Correct v11 selection and run-config validation | `[+]` | Active Browser 1.10 selects v11; active Browser 1.8/1.9 retain v10; incompatible mixes fail. The persisted run-config validator and local/Docker executor invocation accept v11. Focused tests, `make fast`, `make check`, vet, CLI validation and exact W8M package preparation pass. |
| P08.2 Verify synthetic handoff and bounded review | `[+]` | A fresh Browser 1.10 count loopback smoke passes against pinned local sources; repository and documentation checks pass. Review iteration 1 found no P1/P2. W8M retains its separate source binding, full qualification and adoption gates. |

### P08.1 implementation and checks — 2026-09-29 UTC

The trusted runner selects v11 for active Browser 1.10 and rejects active
older/1.10 mixes before executor handoff. Inactive profile copies cannot
select v11. Run-config validation admits v11 while preserving symbolic
credentials, action/authentication approvals, named sessions, environment
allowlists and registration exclusion. Focused regression tests cover profile
selection, mixed and inactive profiles, persisted evidence validation, and
exact local/Docker CLI arguments. `make fast`, `make check`, vet, CLI check and
UWS validation pass with Go 1.26.6; `check-doc-memory` passes with its expected
milestone-without-evolution warning. P08 changes only the existing trusted
handoff and does not change product or architecture direction.

A fresh private copy of the exact W8M count package from P07's restricted
preparation passed all five preparation gates, including quality/secret scan
and trusted dry-run without executor invocation. The private report is
`/home/peter/.local/state/openudon/p08-package-nk237ut5/prepare.json`
(SHA-256 `a4b62dce257d16d87665654ede97ef5b5f2c533ec47bffcc2abc88aaa6e70528`).
The package report does not expose the protocol choice; the focused source and
run-config tests establish that binding. No browser or target was contacted.

## Acceptance

- Active Browser 1.10 chooses v11 and reaches the external Udon invocation;
  active Browser 1.8/1.9 keep v10, inactive profile copies do not select a
  protocol, and an incompatible active mix fails before execution.
- v11 run configs preserve symbolic credentials, action/authentication approvals,
  named sessions, environment allowlists and registration exclusion in local
  and Docker modes. Unsupported protocol/profile combinations fail closed.
- Focused tests, an affected fresh synthetic journey, repository gates and a
  persisted bounded review pass. No real target contact, runtime adoption,
  deployment or push occurs under P08.

## Review

Whole-milestone review iteration 1/10 passed. The review checked the
profile-to-protocol mapping, inactive and mixed-profile behavior, v11
run-config validation, local/Docker CLI arguments, credential and approval
boundaries, source-owned documentation, exact package preparation and the
fresh loopback smoke. The first smoke command stopped before browser setup
because the Make variable for the external Node modules was empty; a fresh
invocation with the correct variable passed its Browser 1.10 count journey in
51 seconds. The preflight failure and correction are retained here; neither
used a real target. The selected Udon/Browserdriver sources were not modified.
No P1/P2 or higher-severity finding remains. P08 implementation is complete
locally; W8M W24 remains blocked pending its own exact-source qualification,
runtime selection and separately authorized live count. No deployment,
publication or push occurred.
