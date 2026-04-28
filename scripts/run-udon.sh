#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -lt 1 ]]; then
  printf 'usage: %s <workflow-file> [workdir]\n' "$0" >&2
  exit 2
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
workflow="$1"
workdir="${2:-$root/.ramen-run}"

if [[ ! -f "$workflow" ]]; then
  printf 'workflow file not found: %s\n' "$workflow" >&2
  exit 1
fi

udon_bin="$root/../udon/dist/udon-linux-amd64"
if [[ ! -x "$udon_bin" ]]; then
  udon_bin="$root/../udon/udon"
fi

if [[ ! -x "$udon_bin" ]]; then
  printf 'udon executable not found. Build ../udon/cmd/udon or use ../udon/dist/udon-linux-amd64.\n' >&2
  exit 1
fi

mkdir -p "$workdir"
exec "$udon_bin" --workdir "$workdir" --workflow "$workflow" --workflow-format auto
