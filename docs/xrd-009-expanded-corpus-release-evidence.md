# XRD-009 Expanded Corpus Release Evidence

XRD-009 closes the evidence gap left after expanding the eval corpus beyond the original ten-example
real-provider baseline. Ramen owns the release evidence gate and release-note checklist. Provider
behavior remains monitored through XRD-006, and XRD-007 readiness evidence remains local/manual
during active development.

## Release Gate

Use the normal manual release path:

```bash
make release-eval
```

`make release-eval` now passes `--min-briefs` using the current `examples/eval` directory count, so
candidate release evidence must cover the full committed corpus by default. Override
`RAMEN_RELEASE_MIN_BRIEFS` only when intentionally running a smaller emergency or diagnostic gate
and record that exception in the release note.

## Required Evidence

- Eval JSON and Markdown report paths under ignored `eval/runs/`.
- Local readiness JSON path under ignored `eval/readiness/`.
- Provider and model.
- Commit and dirty state.
- Eval corpus size and `--min-briefs` value.
- Pass rate and release-gate result.
- Structured fallback count.
- Maximum attempts for any brief.
- Blocking reference issues.
- Secret-scan failures.
- Provider Drift Watch findings from `docs/xrd-006-provider-drift-watch.md`.

## Acceptance Criteria

- The release gate can fail if fewer than the required number of eval briefs are run.
- `make release-eval` covers the full current corpus by default.
- Release notes have a place to record the minimum brief gate and any exception.
- Real-provider reports remain uncommitted under ignored eval output paths.

## Boundary

This does not enable hosted automation, upload artifacts, relax provider drift handling, or commit
real-provider outputs. It only makes expanded-corpus release evidence explicit and harder to
under-sample by accident.
