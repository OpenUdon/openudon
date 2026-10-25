#!/usr/bin/env python3
"""Offline source inventory. No builds, tests, network, or sibling writes."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[2]
MODULE = 'github.com/OpenUdon/openudon/'
PUBLIC = 'approval authority browsercontract browsertransaction browserverify credentialpolicy digest handoff packagev3 runevidence trust udonreport wire'.split()
DELETE = ('browserauthor browserauthoring browsercandidate browsercapture browserpackage browserscenario browsersystem browserintegrationeval browsertransactioneval capturequalification browserworkflow browsercheck synthesize workflowintent elicitor projectwizard udonrunner trustedrunner packagepipeline authoringcli authoringengine artifactwriter stepauthoring eval smokematrix releaseevidence').split()
A31 = 'synthesize workflowintent elicitor projectwizard udonrunner trustedrunner'.split()
ENV = dict(os.environ, GOWORK='off', GOPROXY='off', GOSUMDB='off', GOAUTH='off', GOTOOLCHAIN='local', GOFLAGS='-mod=readonly')


def run(args, cwd=ROOT):
    return subprocess.check_output(args, cwd=cwd, env=ENV, timeout=110).decode()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def stream(data):
    decoder = json.JSONDecoder()
    while data.strip():
        value, end = decoder.raw_decode(data.lstrip())
        yield value
        data = data.lstrip()[end:]


def imports(text):
    # Go import declarations only; include tagged files omitted by default go list.
    declarations = re.findall(r'(?m)^import\s+(?:\([\s\S]*?\)|[^\n]+)', text)
    return sorted(set(re.findall(r'"([^"\n]+)"', '\n'.join(declarations))))


def generate(workspace, w8m):
    listed = list(stream(run(['go', 'list', '-json', './...'])))
    public = list(stream(run(['go', 'list', '-deps', '-json'] + ['./' + p for p in PUBLIC])))
    tracked = run(['git', 'ls-files', '-z']).split('\0')
    files = sorted(p for p in tracked if p and (ROOT / p).is_file())
    graph = []
    for p in listed:
        graph.append({k: sorted(p.get(k, [])) for k in ['Imports', 'Deps', 'TestImports', 'XTestImports']} | {'package': p['ImportPath']})
    graph.sort(key=lambda p: p['package'])
    source_edges, go_files = [], {}
    hcl = []
    for name in files:
        if not name.endswith('.go'):
            continue
        data = (ROOT / name).read_bytes()
        go_files[name] = digest(data)
        text = data.decode()
        for imported in imports(text):
            if imported.startswith(MODULE):
                source_edges.append({'file': name, 'kind': 'test' if name.endswith('_test.go') else 'production', 'import': imported})
        for line, value in enumerate(text.splitlines(), 1):
            if re.search(r'\b(?:HCLToJSON|UnmarshalHCL|hcl\.Import)\b', value):
                hcl.append({'file': name, 'line': line, 'test': name.endswith('_test.go'), 'source': value.strip()})
    guard_references = [ref for ref in hcl if ref['file'] == 'tests/publicapi/legacy_closure_test.go']
    hcl = [ref for ref in hcl if ref['file'] != 'tests/publicapi/legacy_closure_test.go']
    closure = sorted({p['ImportPath'] for p in public})
    forbidden = [p for p in closure if p.startswith(MODULE + 'internal/') or p.startswith(MODULE + 'cmd/') or (p.startswith('github.com/genelet/') and not p.startswith('github.com/genelet/horizon/'))]
    if forbidden:
        raise SystemExit('unsupported public closure: ' + repr(forbidden))
    targets = []
    for name in A31:
        target = MODULE + 'internal/' + name
        targets.append({'package': target,
            'direct_consumers': [p['package'] for p in graph if target in p['Imports']],
            'transitive_consumers': [p['package'] for p in graph if target in p['Deps']],
            'test_consumers': [p['package'] for p in graph if target in p['TestImports'] + p['XTestImports']]})
    deleted = {MODULE + 'internal/' + n for n in DELETE}
    # Nested engine/presentation and their reverse consumers belong to the removal closure.
    deleted.update(p['package'] for p in graph if p['package'].startswith(MODULE + 'internal/browsertransaction/'))
    deleted.add(MODULE + 'cmd/udon-runner')
    reverse = sorted({p['package'] for p in graph if set(p['Imports']) & deleted} - deleted)
    refs = []
    for name in files:
        if name.startswith('tabilet/') or name == 'AGENTS.md' or name.startswith(('docs/a31-', 'docs/a32-')):
            continue  # Historical/shared ledgers are immutable, not migration inputs.
        if name.endswith('.go') or name.startswith('docs/fixtures/'):
            continue
        if not (name.endswith(('.md', '.yml', '.yaml', '.sh')) or name in ['Makefile', 'Dockerfile'] or name.startswith('bin/')):
            continue
        text = (ROOT / name).read_text(errors='replace')
        hits = [i for i, line in enumerate(text.splitlines(), 1) if re.search(r'cmd/(?:openudon|udon-runner)|(?:^|[ /`])openudon\s|internal/(?:' + '|'.join(DELETE) + r')\b|intent\.hcl|workflow\.hcl|ghcr.io/.*/openudon', line)]
        if hits:
            refs.append({'file': name, 'lines': hits})
    main = ROOT / 'cmd/openudon/main.go'
    cases = []
    if main.exists():
        text = main.read_text().split('switch command {', 1)[1].split('\nfunc ', 1)[0]
        cases = sorted(set(re.findall(r'"([^"\n]+)"', '\n'.join(re.findall(r'(?m)^\s*case (.*):$', text)))))
    retain = {name: sha for name, sha in go_files.items() if name.split('/')[0] in PUBLIC or name.startswith('docs/schemas/')}
    fixtures = {name: digest((ROOT / name).read_bytes()) for name in files if name.startswith(('docs/fixtures/', 'docs/schemas/')) or name in ['docs/p10-public-surface.json'] or (name.split('/')[0] in PUBLIC and not name.endswith('.go'))}
    def pin(repo, revision):
        return {'revision': revision, 'tree': run(['git', 'rev-parse', revision + '^{tree}'], repo).strip(), 'local_commit_object': run(['git', 'cat-file', '-t', revision], repo).strip() == 'commit', 'remote_fetchability': 'unverified; A32.5 publication gate'}
    w8m_revision = 'b23fd63772e0d68f4e5ba1acb59d31b3c757d391'
    locks = {}
    for path in ['browser-workflows/browser-system.lock.json', 'browser-workflows/candidates/kinet-browser/stack.lock.json']:
        raw = subprocess.check_output(['git', 'show', w8m_revision + ':' + path], cwd=w8m, timeout=5)
        locks[path] = {'sha256': digest(raw), 'contents': json.loads(raw)}
    frozen = {'openudon_system_history': pin(ROOT, '01998d6f9d2d2bfd5e6bcbb2c5be9c0ec908b864'), 'openudon_browser': pin(ROOT, 'c2f161d762bc9f2217bbf0c34b00cdef64b0f7d0'), 'w8m': pin(w8m, w8m_revision) | {'locks': locks}}
    ramen = workspace / 'ramen'
    # Freeze the inspected exact snapshot; never infer an OpenUdon module pin where absent.
    ramen_revision = 'eab764c4c353c58311e1da94fbca706645ba995f'
    ramen_mod = subprocess.check_output(['git', 'show', ramen_revision + ':go.mod'], cwd=ramen, timeout=5)
    frozen['ramen'] = pin(ramen, ramen_revision) | {'go_mod_sha256': digest(ramen_mod), 'openudon_module_requirements': [line.strip() for line in ramen_mod.decode().splitlines() if 'github.com/OpenUdon/openudon ' in line], 'policy': 'no OpenUdon imports; conversion/runtime ownership remains Ramen'}
    handoff_path = workspace / 'kinet/docs/m53-stage12-handoff.md'
    handoff = handoff_path.read_bytes()
    baseline_path = ROOT / 'docs/a32-consumer-inventory.json'
    baseline = json.loads(baseline_path.read_text())
    for key in ['retained_public_source_sha256', 'retained_history_wire_fixture_sha256']:
        for name, expected in baseline[key].items():
            if digest((ROOT / name).read_bytes()) != expected:
                raise SystemExit('retained public source/fixture changed: ' + name)
    legacy_packages = [p['package'] for p in graph if p['package'].startswith((MODULE + 'internal/', MODULE + 'cmd/'))]
    legacy_edges = [e for e in source_edges if e['import'].startswith((MODULE + 'internal/', MODULE + 'cmd/'))]
    build_refs = [r for r in refs if r['file'] == 'Makefile' or r['file'].startswith('.github/workflows/')]
    return {'version': 'openudon.a32-final-consumer-inventory.v1',
        'baseline_inventory_sha256': digest(baseline_path.read_bytes()),
        'baseline_retained_hashes_match': True,
        'source_revision': run(['git', 'log', '-1', '--format=%H', '--', ':(glob)**/*.go', 'go.mod', 'go.sum']).strip(),
        'method': {'owner': 'go list -json ./...; no execution', 'public': 'go list -deps -json public roots; no execution', 'source_edges': 'all tracked .go import declarations, including tagged files', 'environment': {k: ENV[k] for k in ['GOWORK', 'GOPROXY', 'GOSUMDB', 'GOAUTH', 'GOTOOLCHAIN', 'GOFLAGS']}, 'go_version': run(['go', 'version']).strip()},
        'packages': len(graph), 'owner_graph': graph, 'owner_graph_sha256': digest(json.dumps(graph, sort_keys=True).encode()),
        'source_files_sha256': go_files, 'source_edges': source_edges, 'a31_targets': targets,
        'a31_inventory_sha256': digest((ROOT / 'docs/a31-consumer-inventory.json').read_bytes()),
        'declared_removal_packages': sorted(deleted), 'remaining_reverse_consumers': reverse,
        'other_internal_packages': sorted(p['package'] for p in graph if p['package'].startswith(MODULE + 'internal/') and p['package'] not in deleted),
        'command_cases': cases, 'command_build_doc_references': refs,
        'current_build_command_references': build_refs,
        'reference_scope': 'command_build_doc_references includes retained historical guide/example strings; current_build_command_references covers Make/CI only',
        'legacy_packages': legacy_packages, 'legacy_source_imports': legacy_edges,
        'empty_legacy_owned_closure': not (legacy_packages or legacy_edges or cases or hcl or build_refs),
        'developer_commands': [p['ImportPath'] for p in listed if p.get('Name') == 'main'],
        'hcl_input_references': hcl,
        'hcl_guard_fixture_references': guard_references,
        'hcl_reference_scope': 'lexical candidates exclude synthetic guard fixture strings; tests/publicapi AST guard proves no owned HCL input references including inactive build tags',
        'public_roots': PUBLIC, 'public_closure': closure, 'public_closure_forbidden': forbidden,
        'retained_public_source_sha256': retain, 'retained_history_wire_fixture_sha256': fixtures,
        'frozen_consumers': frozen,
        'kinet_m53': {'handoff': 'kinet/docs/m53-stage12-handoff.md', 'sha256': digest(handoff), 'legacy_cmd_image_internal_consumers': 'none in accepted compiled/caller proof; public libraries remain LIVE', 'unqualified': 'extra physical/observer/forced-crash methods; historical UNKNOWN/no replay retained'},
        'limitations': ['Source inventory only; semantic no-HCL-input/public-closure guards are separately qualified by make check.', 'No consumer build, runtime, physical or remote-fetchability claim; exact frozen objects were inspected locally.', 'Retained historical guides/examples and guard fixture strings are not current CLI/build consumers. Consumer/publication/whole-review acceptance is recorded separately.']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--workspace', type=Path, required=True)
    parser.add_argument('--w8m', type=Path, required=True)
    parser.add_argument('--output', type=Path, default=ROOT / 'docs/a32-final-consumer-inventory.json')
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    if args.output.resolve() == (ROOT / 'docs/a32-consumer-inventory.json').resolve():
        raise SystemExit('A32.1 baseline is immutable; use a32-final-consumer-inventory.json')
    result = (json.dumps(generate(args.workspace, args.w8m), indent=2) + '\n').encode()
    if args.check:
        if args.output.read_bytes() != result:
            raise SystemExit('inventory differs: regenerate and review')
        print('inventory current; public closure contains no legacy/private imports')
    else:
        args.output.write_bytes(result)
        print('wrote', args.output)


if __name__ == '__main__':
    main()
