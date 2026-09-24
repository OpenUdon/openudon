# Retired milestone M26 - Status M26 - iCoT Catalog Planning And Operation-Detail Mapping Pipeline

**Milestone.** M26
**Outcome.** legacy-preserved
**Retired.** 2026-09-24
**Source status.** tabilet/memory-bank/status-M26.md
**Source status SHA-256.** 6825e1d382a7cd0c3c249685364c1cd02a5ac6ad869948cf3d09d6a0d5d8f206
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
# Status M26 - iCoT Catalog Planning And Operation-Detail Mapping Pipeline

Task states use `[ ]` not started, `[/]` in progress, `[+]` complete, and `[!]`
blocked.

| Item | State | Notes |
|---|---|---|
| Early catalog planning interface added | `[+]` | Added bounded extractor `CatalogPlan(ctx, CatalogPlanRequest)` and compact `catalog_plan` prompt for provider/artifact selection without full API document contents. |
| Compact catalog-plan payload added | `[+]` | Payload is built from deterministic `BuildCatalogHints` and `CatalogMigrationCandidates`, capped to compact artifact metadata: provider, auth status, artifact key, relative path, kind/protocol, and follow-up text. |
| Local catalog-plan validation added | `[+]` | Returned provider/artifact tuples are validated against the deterministic shortlist; unknown providers, unknown artifact keys, invented paths, and non-migratable selections are rejected locally. |
| Catalog-selected artifact migration added | `[+]` | Valid selected artifacts are migrated, advisory overlays for selected providers are included when available, and matching apitools security overlays are materialized as package-local sidecars beside migrated API sources. No valid selection falls back to deterministic catalog migration behavior. |
| Provider-level step seeding added | `[+]` | Early catalog-plan output may seed only `name`, `type: http`, `provider`, optional local API document path, `do`, and safe `depends_on`; operation IDs, request mappings, credentials, response paths, and secrets remain disallowed at this stage. |
| Catalog-plan transcript events added | `[+]` | iCoT records `catalog_plan_call`, `catalog_plan_result`, and `catalog_plan_rejected` events in `.icot/transcript.json`. |
| Catalog-plan malformed output hardening added | `[+]` | Tolerates malformed `depends_on` values such as numeric items without discarding the entire catalog plan; unsafe values are still filtered before step seeding. |
| Operation-detail mapping draft pass added | `[+]` | When readiness reports missing required request values after operation selection, iCoT gives the LLM one focused draft pass with selected operation details before asking the operator for manual mappings. |
| Deterministic prework step insertion added | `[+]` | When a selected operation needs values another listed local operation can produce, iCoT inserts a legal dependency step before field mapping; OpenWeatherMap geocoding now feeds One Call weather latitude/longitude instead of hardcoded coordinates. |
| Pre-final flow review added | `[+]` | LLM-assisted iCoT runs one structured advisory `ReviewDraft` pass before the final draft summary. It reports `llm_flow_review_*` warnings for cross-step data-flow mistakes, records transcript evidence, and never mutates or blocks the draft by itself. |
| Review follow-up hardening added | `[+]` | Added focused tests for API source refs, draft-review sanitization and HCL comments, readiness code triggers, session validation, drift comparison, and draft-detail caps; added public docs for iCoT session files, transcripts, and project brief schema; expanded `icot --help` stage/subcommand guidance. |
| Future review-repair mode graduated | `[+]` | M26 parked `--review-repair`; M28 implements it as an experimental opt-in mode with bounded deterministic repairs and unchanged-draft stop behavior. |
| Prompt/docs aligned | `[+]` | Updated iCoT prompt guidance, README, public iCoT/SaaS authoring docs, weather tutorial, architecture memory, and milestone/status guidance for the new pipeline. |
| Regression coverage added | `[+]` | Added tests for compact catalog payloads, valid and invalid catalog selections, malformed `depends_on`, deterministic fallback, catalog-plan transcript events, LLM request-mapping draft before manual prompt fallback, prompt modes, and pre-final flow review warnings. |
| Memory-bank status-file restructure completed | `[+]` | Merged the root status dashboard and status-file index into `memory-bank/milestone.md`, removed `memory-bank/status.md`, updated `AGENTS.md`, and kept future milestone task history in individual `status-Mx.md` files. |
| Verification completed | `[+]` | `go test ./internal/icot/elicitor`, `go test ./...`, `git diff --check`, `check-doc-memory`, and `mkdocs build --strict` passed after the iCoT pipeline and memory-bank status-file restructure. |
~~~~~~~~~~~~~~~~~~~~
