#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="${1:-$root/examples}"
schema_dir="$root/../uws/versions"

if [[ ! -d "$schema_dir" ]]; then
  printf 'missing UWS schema directory: %s\n' "$schema_dir" >&2
  exit 1
fi

if [[ ! -e "$target" ]]; then
  printf 'target does not exist: %s\n' "$target" >&2
  exit 1
fi

mapfile -t files < <(
  find "$target" -type f \( \
    -name '*.uws.json' -o \
    -name '*.uws.yaml' -o \
    -name '*.uws.yml' \
  \) | sort
)

if [[ "${#files[@]}" -eq 0 ]]; then
  printf 'no UWS artifacts found under %s\n' "$target"
  exit 0
fi

printf 'found %d UWS artifact(s); schema selected from document version\n' "${#files[@]}"
for file in "${files[@]}"; do
  go run ./cmd/ramen validate "$file"
done
