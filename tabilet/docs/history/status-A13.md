# Retired milestone A13 - A13 iCoT UI Review Remediation

**Milestone.** A13
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-A13.md
**Source status SHA-256.** 43781cc6faa991620eef368b0e3aece2b4ce72540a35c2b264f770b8fe4559d3
**Source milestone snapshot.** tabilet/docs/history/milestone-before-legacy-retirement.md.txt
**Snapshot SHA-256.** 26884eeda9af4ded30e6d545a33dca84c401d2320c22355fe4fb0336d5dcac71
**Evidence.** 71a4f78afbcf2180fc478ffa89c53544c9160648
**Worktree.** includes uncommitted changes
**Review.** not established
**Review iterations.** not recorded
**Verification.** Original status bytes and full earlier milestone bytes preserved by SHA-256; no fresh acceptance claim.
**Consolidated into.** Current milestone dashboard and maintained memory-bank guidance; full earlier text remains in the frozen snapshot.

## Status record

~~~~~~~~~~~~~~~~~~~~markdown
# A13 iCoT UI Review Remediation

| Item | State | Notes |
| --- | --- | --- |
| A13.1 Review scope and engineering ledger | `[+]` | The verified review is now ordered into testable correctness, control, revision, recovery, evidence, protocol, and qualification work without widening the single-workspace, single-operator, non-executing boundary. |
| A13.2 Effective answers and field-addressable rejection | `[+]` | Commit `c19589a` makes goal correction replace stale state, rejects answers that leave their decision unresolved, and carries authoritative question identity through safe 422 responses so the shell marks and focuses the rejected field. |
| A13.3 Structured controls and explicit deferral | `[+]` | Commit `eb7e8c0` adds engine-owned closed choices, syntax guidance, stable security-alternative values, and deferrability plus accessible select/deferral controls and structured request validation. Its browser audit also found and fixed a P1 where `no-referrer` made Chromium send `Origin: null`; `same-origin` retains cross-origin privacy and restores the exact-Origin bootstrap. |
| A13.4 Settled-decision revision | `[+]` | Commit `d788853` adds a bounded exact-revision reopen mutation for eligible human answers, persists the cleared non-approvable state through normalization and autosave, returns replacements through the ordinary complete-frontier contract, and exposes impact-aware accessible controls. The first diff audit fixed side-effect default restoration, deferral replacement, and numeric answer-order edge cases before closure. |
| A13.5 Bootstrap recovery and review evidence | `[+]` | Commit `7414c1d` adds separately throttled terminal-only code rotation after use or expiry, redirects a lost scoped shell session to the tokenless recovery page, and renders candidate workflows, prompt-safe evidence summaries/references, and discovery candidates/blockers beside approval. Recommendation fill remains empty-only from A13.3; no evidence value or attribute is added to the visible review surface. |
| A13.6 Conditional HTTP, performance evidence, and documentation | `[+]` | Commits `aa15517` and `2eb87d0` implement wildcard ETags and frozen-before-stale conflicts, add a reproducible 1 MiB exact-hash benchmark plus a same-metadata byte-change regression, and document volatile answers, serialized mutation reads, recovery, settled revision, visible evidence, and the CLI-first AI handoff. The reference Haswell VM measured about 3.2 ms per poll (roughly 0.16% of one core at two seconds), so the unsafe metadata-only shortcut was rejected. |
| A13.7 Qualification and review closure | `[+]` | Commits `1b91157` and `755f36c` close the final review loop: decisions without safe dependent-state clearing are not advertised, recovery fails before rotation when no terminal sink exists, stale browser fixtures use the tokenless bootstrap and exact question IDs, and rejected-field focus waits for mutation unlock. Focused/full/race/vet tests, `make check`, strict MkDocs, doc-memory and repository-boundary checks, CGO-disabled Linux/Windows/macOS builds, silent pinned `deadcode v0.47.0`, the complete iCoT browser suite with the documented test-only local sandbox override, corpus/variant/scorecard/UWS/n8n/lint gates, and trusted sandbox dry-run demos pass. Repeated full-range audits found no unresolved P1/P2. The sandbox-enabled Chromium gate still fails on this host with `No usable sandbox`; the separate scenario lane reaches headed authoring under Xvfb but is blocked by the same required sandbox policy. Neither result is claimed as hosted evidence, so A11.5 remains `[~]`. |

OpenUdon remains CLI-first for v0.2. The UI remains an experimental embedded
loopback client over one engine and gains no execution, account, remote-hosting,
or UI-owned LLM authority.
~~~~~~~~~~~~~~~~~~~~
