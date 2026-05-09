#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
parent="$(cd "$root/.." && pwd)"

missing=0
for name in uws apitools; do
  path="$parent/$name"
  if [[ -d "$path" ]]; then
    printf 'ok: %s\n' "$path"
  else
    printf 'missing: %s\n' "$path" >&2
    missing=1
  fi
done

exit "$missing"
