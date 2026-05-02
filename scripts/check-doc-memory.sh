#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

missing=0
for file in \
  memory-bank/product.md \
  memory-bank/architecture.md \
  memory-bank/tech-stack.md \
  memory-bank/milestone.md \
  memory-bank/status.md \
  evolution/prompt-v1.md \
  evolution/result-v1.md
do
  if [[ -f "$file" ]]; then
    printf 'ok: %s\n' "$file"
  else
    printf 'missing: %s\n' "$file" >&2
    missing=1
  fi
done

if [[ "$missing" -ne 0 ]]; then
  exit "$missing"
fi

stale_pattern='ICOT\.md|SYMPHONY_WRAPPER\.md|WORKFLOW\.md|openudon\.md|TODO\.md|migrate\.md'
stale_hits="$(
  grep -RInE "$stale_pattern" . \
    --exclude-dir=.git \
    --exclude-dir=readiness \
    --exclude-dir=runs \
    --exclude-dir=artifacts \
    --exclude=check-doc-memory.sh || true
)"
if [[ -n "$stale_hits" ]]; then
  printf 'stale removed-doc references found:\n%s\n' "$stale_hits" >&2
  exit 1
fi
printf 'ok: no stale removed-doc references\n'

changed_files="$(git diff --name-only HEAD -- 2>/dev/null || true)"
untracked_evolution="$(git ls-files --others --exclude-standard evolution 2>/dev/null || true)"
if printf '%s\n' "$changed_files" | grep -qx 'memory-bank/milestone.md'; then
  if ! printf '%s\n%s\n' "$changed_files" "$untracked_evolution" | grep -q '^evolution/'; then
    printf 'warning: memory-bank/milestone.md changed without evolution/ changes; confirm no new evolution version is needed\n' >&2
  fi
fi
