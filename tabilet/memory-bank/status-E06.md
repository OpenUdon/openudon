# E06 Honest And Reproducible Evidence

Item | State | Notes
--- | --- | ---
E06.1 Honest browser status | `[+]` | Suites with no executed scenario report `not_run`; structural inspection accepts the wire but passing/release verification does not.
E06.2 Bounded subprocess trees | `[+]` | Probes use 30 seconds, builds two minutes, and scenarios three minutes; cancellation terminates full Unix or Windows process trees.
E06.3 Portable deterministic reports | `[+]` | Quality/refinement artifacts persist package-relative labels and stripped candidates; map-derived diagnostics and evidence are sorted.
E06.4 Self-cleaning eval workspaces | `[+]` | Temporary generated workspaces are removed normally; `generated_dir` is empty unless explicit archive retention records an archive-relative path.
E06.5 Minimal fixtures | `[+]` | Duplicate full Slack specifications were removed in favor of the minimal per-example fixture and deterministic expected artifacts were regenerated.

Browser release targets still require readiness; `not_run` is never evidence of a pass.
