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

exec python3 - "$config" "$root" <<'PY'
import json
import os
import shutil
import string
import sys
import tempfile

RUN_CONFIG_VERSION = "ramen.executor-run.v1"


def fail(message):
    raise SystemExit(message)


def require_string(cfg, key):
    value = cfg.get(key) or ""
    if not isinstance(value, str):
        fail("run config field %s must be a string" % key)
    if not value.strip():
        fail("run config requires %s" % key)
    reject_control_chars(key, value)
    return value


def reject_control_chars(name, value):
    if any(ord(ch) < 0x20 or ord(ch) == 0x7F for ch in value):
        fail("%s must not contain control characters" % name)


def reject_backslash(name, value):
    if "\\" in value:
        fail("%s must use slash separators: %s" % (name, value))


def path_inside(base, path):
    try:
        return os.path.commonpath([base, path]) == base
    except ValueError:
        return False


def package_relative_path(package_root, name, value):
    reject_control_chars(name, value)
    reject_backslash(name, value)
    if os.path.isabs(value):
        absolute = os.path.abspath(value)
        if not path_inside(package_root, absolute):
            fail("%s escapes package_root: %s" % (name, value))
        rel = os.path.relpath(absolute, package_root)
    else:
        rel = os.path.normpath(value)
        absolute = os.path.abspath(os.path.join(package_root, rel))
        if not path_inside(package_root, absolute):
            fail("%s escapes package_root: %s" % (name, value))
    if rel in ("", ".") or rel.startswith(".." + os.sep) or rel == "..":
        fail("%s escapes package_root: %s" % (name, value))
    return rel, absolute


def validate_regular_package_file(package_root, rel, absolute, label):
    current = package_root
    for index, segment in enumerate(rel.split(os.sep)):
        if segment in ("", "."):
            fail("%s path is invalid: %s" % (label, rel))
        if segment == "..":
            fail("%s escapes package_root: %s" % (label, rel))
        current = os.path.join(current, segment)
        try:
            info = os.lstat(current)
        except OSError as err:
            fail("%s file not found: %s: %s" % (label, absolute, err))
        if os.path.islink(current):
            fail("%s file must not be a symlink: %s" % (label, absolute))
        last = index == len(rel.split(os.sep)) - 1
        if last:
            if not os.path.isfile(current):
                fail("%s file must be a regular file: %s" % (label, absolute))
        elif not os.path.isdir(current):
            fail("%s parent must be a directory: %s" % (label, absolute))


def env_name(binding):
    out = ["UDON_CREDENTIAL_"]
    last_underscore = False
    for ch in binding.strip():
        if ch in string.ascii_letters or ch in string.digits:
            out.append(ch.upper())
            last_underscore = False
        elif not last_underscore:
            out.append("_")
            last_underscore = True
    return "".join(out).rstrip("_")


def credential_env_names(bindings):
    out = []
    seen = set()
    for binding in bindings:
        binding = str(binding).strip()
        if not binding:
            continue
        reject_control_chars("credential binding", binding)
        name = env_name(binding)
        if name == "UDON_CREDENTIAL":
            fail("credential binding does not produce a valid env var: %s" % binding)
        if name not in seen:
            out.append(name)
            seen.add(name)
    return out


def load_config(path):
    with open(path, "r", encoding="utf-8") as f:
        cfg = json.load(f)
    if cfg.get("version") != RUN_CONFIG_VERSION:
        fail("unsupported run config version: %s" % cfg.get("version"))
    openapi_paths = cfg.get("openapi_paths", [])
    credential_bindings = cfg.get("credential_bindings", [])
    if openapi_paths is None:
        openapi_paths = []
    if credential_bindings is None:
        credential_bindings = []
    if not isinstance(openapi_paths, list):
        fail("openapi_paths must be a list")
    if not isinstance(credential_bindings, list):
        fail("credential_bindings must be a list")
    return cfg, openapi_paths, credential_bindings


def validate_openapi_paths(package_root, openapi_paths):
    out = []
    for raw in openapi_paths:
        if not isinstance(raw, str):
            fail("openapi path must be a string")
        if not raw.strip():
            fail("openapi path must be non-empty")
        rel, src = package_relative_path(package_root, "openapi path", raw)
        validate_regular_package_file(package_root, rel, src, "openapi")
        out.append((rel, src))
    return out


def stage_package(workdir, workflow_rel, workflow_path, openapi_files):
    os.makedirs(workdir, mode=0o755, exist_ok=True)
    stage = tempfile.mkdtemp(prefix="stage.", dir=workdir)

    staged_workflow = os.path.join(stage, workflow_rel)
    os.makedirs(os.path.dirname(staged_workflow), mode=0o755, exist_ok=True)
    shutil.copy2(workflow_path, staged_workflow)

    for rel, src in openapi_files:
        dst = os.path.join(stage, rel)
        os.makedirs(os.path.dirname(dst), mode=0o755, exist_ok=True)
        shutil.copy2(src, dst)

    return stage, staged_workflow


def executor_argv(root, stage, staged_workflow, workflow_format):
    image = os.environ.get("RAMEN_UDON_IMAGE", "")
    if image:
        argv = [
            "docker",
            "run",
            "--rm",
            "-v",
            stage + ":/workspace",
            "-w",
            "/workspace",
        ]
        for name in docker_env_names:
            argv.extend(["-e", name])
        argv.extend([
            image,
            "--workdir",
            "/workspace",
            "--workflow",
            "/workspace/" + os.path.relpath(staged_workflow, stage),
            "--workflow-format",
            workflow_format,
        ])
        return argv

    executor = os.environ.get("RAMEN_EXECUTOR") or os.environ.get("RAMEN_UDON_BIN") or ""
    if not executor:
        executor = os.path.join(root, "..", "udon", "dist", "udon-linux-amd64")
    if not os.access(executor, os.X_OK):
        executor = os.path.join(root, "..", "udon", "udon")
    if not os.access(executor, os.X_OK):
        fail("trusted executor not found. Set RAMEN_EXECUTOR, RAMEN_UDON_BIN, RAMEN_UDON_IMAGE, or build ../udon.")
    return [
        executor,
        "--workdir",
        stage,
        "--workflow",
        staged_workflow,
        "--workflow-format",
        workflow_format,
    ]


config_path = sys.argv[1]
repo_root = sys.argv[2]
cfg, openapi_paths, credential_bindings = load_config(config_path)

package_root = os.path.abspath(require_string(cfg, "package_root"))
workdir = os.path.abspath(require_string(cfg, "workdir"))
workflow_format = cfg.get("workflow_format") or "uws-yaml"
if not isinstance(workflow_format, str):
    fail("run config field workflow_format must be a string")
reject_control_chars("workflow_format", workflow_format)

workflow_raw = require_string(cfg, "workflow_path")
workflow_rel, workflow_path = package_relative_path(package_root, "workflow_path", workflow_raw)
validate_regular_package_file(package_root, workflow_rel, workflow_path, "workflow")
openapi_files = validate_openapi_paths(package_root, openapi_paths)

docker_env_names = credential_env_names(credential_bindings)
for name in docker_env_names:
    if not os.environ.get(name):
        fail("required credential env var is not set: %s" % name)

stage, staged_workflow = stage_package(workdir, workflow_rel, workflow_path, openapi_files)
argv = executor_argv(repo_root, stage, staged_workflow, workflow_format)
try:
    os.execvp(argv[0], argv)
except OSError as err:
    fail("invoke trusted executor: %s" % err)
PY
