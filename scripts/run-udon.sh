#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 2 || "${1:-}" != "--config" ]]; then
  printf 'usage: %s --config <run-config.json>\n' "$0" >&2
  exit 2
fi

config="$2"
if [[ ! -f "$config" ]]; then
  printf 'run config not found: %s\n' "$config" >&2
  exit 1
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

read_config() {
  python3 - "$config" <<'PY'
import json
import os
import sys
import pathlib

with open(sys.argv[1], "r", encoding="utf-8") as f:
    cfg = json.load(f)

if cfg.get("version") != "ramen.executor-run.v1":
    raise SystemExit("unsupported run config version: %s" % cfg.get("version"))

package_root = cfg.get("package_root") or ""
workflow_path = cfg.get("workflow_path") or ""
workflow_format = cfg.get("workflow_format") or "uws-yaml"
workdir = cfg.get("workdir") or ""
openapi_paths = cfg.get("openapi_paths") or []
credential_bindings = cfg.get("credential_bindings") or []

if not package_root or not workflow_path or not workdir:
    raise SystemExit("run config requires package_root, workflow_path, and workdir")
if not isinstance(openapi_paths, list):
    raise SystemExit("openapi_paths must be a list")
if not isinstance(credential_bindings, list):
    raise SystemExit("credential_bindings must be a list")

package_root = os.path.abspath(package_root)
workdir = os.path.abspath(workdir)
workflow_rel = workflow_path
if not os.path.isabs(workflow_path):
    workflow_path = os.path.join(package_root, workflow_path)
else:
    try:
        workflow_rel = os.path.relpath(workflow_path, package_root)
    except ValueError:
        workflow_rel = pathlib.Path(workflow_path).name

workflow_rel = os.path.normpath(workflow_rel)
if workflow_rel.startswith(".."):
    raise SystemExit("workflow_path must be inside package_root")

def env_name(binding):
    out = ["UDON_CREDENTIAL_"]
    last_underscore = False
    for ch in binding.strip():
        if ch.isalnum():
            out.append(ch.upper())
            last_underscore = False
        elif not last_underscore:
            out.append("_")
            last_underscore = True
    return "".join(out).rstrip("_")

openapi_rel = []
for path in openapi_paths:
    if not path:
        continue
    if os.path.isabs(path):
        try:
            rel = os.path.relpath(path, package_root)
        except ValueError:
            raise SystemExit("openapi path escapes package_root: %s" % path)
    else:
        rel = os.path.normpath(path)
    if rel.startswith(".."):
        raise SystemExit("openapi path escapes package_root: %s" % path)
    openapi_rel.append(rel)

credential_env = []
seen_env = set()
for binding in credential_bindings:
    binding = str(binding).strip()
    if not binding:
        continue
    name = env_name(binding)
    if name == "UDON_CREDENTIAL":
        raise SystemExit("credential binding does not produce a valid env var: %s" % binding)
    if name not in seen_env:
        credential_env.append(name)
        seen_env.add(name)

print(package_root)
print(workflow_path)
print(workflow_format)
print(workdir)
print(workflow_rel)
print(len(openapi_rel))
for rel in openapi_rel:
    print(rel)
print(len(credential_env))
for name in credential_env:
    print(name)
PY
}

mapfile -t values < <(read_config)
package_root="${values[0]}"
workflow="${values[1]}"
workflow_format="${values[2]}"
workdir="${values[3]}"
workflow_rel="${values[4]}"
openapi_count="${values[5]}"
openapi_paths=("${values[@]:6:openapi_count}")
credential_count_index=$((6 + openapi_count))
credential_count="${values[$credential_count_index]}"
credential_env_start=$((credential_count_index + 1))
credential_env_names=("${values[@]:credential_env_start:credential_count}")

if [[ ! -f "$workflow" ]]; then
  printf 'workflow file not found: %s\n' "$workflow" >&2
  exit 1
fi

mkdir -p "$workdir"
stage="$(mktemp -d "$workdir/stage.XXXXXX")"
staged_workflow="$stage/$workflow_rel"
mkdir -p "$(dirname "$staged_workflow")"
cp "$workflow" "$staged_workflow"
for rel in "${openapi_paths[@]}"; do
  src="$package_root/$rel"
  dst="$stage/$rel"
  if [[ -L "$src" ]]; then
    printf 'openapi file must not be a symlink: %s\n' "$src" >&2
    exit 1
  fi
  if [[ ! -f "$src" ]]; then
    printf 'openapi file not found: %s\n' "$src" >&2
    exit 1
  fi
  mkdir -p "$(dirname "$dst")"
  cp "$src" "$dst"
done

docker_env_args=()
for env_name in "${credential_env_names[@]}"; do
  if [[ -z "${!env_name:-}" ]]; then
    printf 'required credential env var is not set: %s\n' "$env_name" >&2
    exit 1
  fi
  docker_env_args+=("-e" "$env_name")
done

if [[ -n "${RAMEN_UDON_IMAGE:-}" ]]; then
  exec docker run --rm \
    -v "$stage:/workspace" \
    -w /workspace \
    "${docker_env_args[@]}" \
    "$RAMEN_UDON_IMAGE" \
    --workdir /workspace \
    --workflow "/workspace/$workflow_rel" \
    --workflow-format "$workflow_format"
fi

executor="${RAMEN_EXECUTOR:-${RAMEN_UDON_BIN:-}}"
if [[ -z "$executor" ]]; then
  executor="$root/../udon/dist/udon-linux-amd64"
fi
if [[ ! -x "$executor" ]]; then
  executor="$root/../udon/udon"
fi
if [[ ! -x "$executor" ]]; then
  printf 'trusted executor not found. Set RAMEN_EXECUTOR, RAMEN_UDON_BIN, RAMEN_UDON_IMAGE, or build ../udon.\n' >&2
  exit 1
fi

exec "$executor" --workdir "$stage" --workflow "$staged_workflow" --workflow-format "$workflow_format"
