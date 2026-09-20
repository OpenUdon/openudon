# Prompt V10 - Adaptive Evidence-Grounded Workflow Authoring

iCoT should replace its fixed one-blocker interview with a dependency-aware
design tree. It should select one active workflow from broad requests, preserve
later workflows as unnumbered candidates, inspect bounded local source evidence
before questioning, and ask every dependency-ready decision as one frontier
round without a fixed breadth ceiling.

Authoring should own the generic versioned interview graph and round engine;
apitools should own safe multi-family API-source discovery; OpenUdon should own
workflow graph construction, source selection/materialization, v2 wires,
proposal/draft/final lifecycle, and agent/report behavior. `tfconfig` should not
change because Terraform facts are outside this authoring boundary.

Only resumable `.icot/` state may be written before approval. Final proposal
approval must atomically cover project, intent, selected sources, collisions,
backups, and promotion. Agent mode must return the whole frontier and file plan
without prompting or writing. Public rationale should remain concise evidence,
never hidden chain-of-thought.
