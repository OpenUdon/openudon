# Ramen Symphony Wrapper Plan

Ramen now owns a local trusted execution wrapper for Symphony-managed handoff packages. The wrapper
validates generated artifacts, current quality gates, reviewer approval JSON, package digest, and
execution tier before invoking udon through `scripts/run-udon.sh`.

This keeps `../symphony` untouched. Symphony may still own reviewer identity, audit history, and
work-item routing, but Ramen no longer depends on an unactionable external enforcement hook before a
trusted local operator can run an approved package.

## Operator Flow

Generate or update artifacts:

```bash
go run ./cmd/ramen synthesize --example examples/support-email --provider gemini --model gemini-2.5-flash
```

Create approval JSON from the current handoff package digest:

```bash
mkdir -p approvals
go run ./cmd/ramen approval-template \
  --example examples/support-email \
  --state approved_for_sandbox \
  --reviewer "Reviewer Name" \
  > approvals/support-email-sandbox.json
```

Validate gates and run udon:

```bash
go run ./cmd/ramen run \
  --example examples/support-email \
  --tier sandbox \
  --approval approvals/support-email-sandbox.json
```

Use `--dry-run` to validate all wrapper gates without invoking udon.

## CLI

```bash
ramen run --example <dir> --tier sandbox|production --approval <file> [--workdir <dir>] [--dry-run]
```

The wrapper:

- reads `expected/symphony-handoff.json`
- reads `expected/quality.json`
- recomputes current quality in memory without rewriting tracked files
- reads and validates the approval JSON
- computes the canonical handoff package digest
- checks approval scope, expiry, digest, and tier/state compatibility
- rejects manifests that allow credential values in artifacts or direct production execution
- invokes `scripts/run-udon.sh` by argv, not by shell-evaluating generated command text

```bash
ramen approval-template --example <dir> --state approved_for_sandbox|approved_for_production --reviewer <name> [--notes <text>]
```

The helper validates the current handoff package and prints approval JSON to stdout.

## Approval Schema

```json
{
  "version": "ramen.approval.v1",
  "scope": "examples/support-email",
  "state": "approved_for_sandbox",
  "reviewer": "Reviewer Name",
  "approved_at": "2026-04-29T12:00:00Z",
  "expires_at": "2026-05-06T12:00:00Z",
  "package_sha256": "<current handoff package digest>",
  "notes": "optional"
}
```

`expires_at` and `notes` are optional. The wrapper does not cryptographically prove reviewer
identity; it records the reviewer string supplied by the trusted operator.

## Tier Rules

- `sandbox` accepts `approved_for_sandbox` or `approved_for_production`
- `production` accepts only `approved_for_production`
- expired approvals fail
- scope mismatch fails
- package digest mismatch fails
- stored or current quality failures fail
- malformed handoff manifests fail
- credential-value artifacts and direct production execution remain prohibited

## Implementation Notes

The trusted-runner implementation lives under `internal/trustedrunner`. Ramen-owned synthesis and
assessment logic remains in `internal/synthesize`; the wrapper uses the non-writing quality
assessment path so approval checks do not modify tracked artifacts.
