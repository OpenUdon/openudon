# Lessons

Keep concise, reusable lessons that still affect decisions. Consult the topics
relevant to the current task before substantial changes; this is not a session
log or a requirement to produce one lesson per milestone.

Maintain relevant lessons when ordinary work produces reusable, evidence-backed
learning, and consolidate them during milestone closure. A still-applicable
lesson stays here even after its supporting milestone retires; closing a task
or reaching an arbitrary file size is not a reason to discard useful knowledge.

For each lesson, use a descriptive heading and record when it applies, the
lesson, why it matters, and links to supporting tasks, verification, or retired
records. Keep product facts in `product.md`, system contracts in
`architecture.md`, and commands in `tech-stack.md` rather than repeating them.

Merge duplicates. Before materially replacing or removing an obsolete lesson,
append its previous wording, source, reason, and replacement reference to
`tabilet/docs/history/knowledge.md` under the retirement rules in
[milestone.md](milestone.md#long-term-memory-and-retirement). Preserve evidence
links when merging lessons. Revalidate historical evidence before applying it
to current work. Routine wording edits need no journal entry or archive run.

## Validate task tables with the installed runner

When repairing or adding a status ledger, use outer `|` table delimiters and
the exact backticked markers defined in `AGENTS.md`. A visually readable table
without outer delimiters was invisible to the installed API runner, while
`[x]` was rejected as an unknown state. This left historical rows uncounted
and hid in-progress work. The repaired [A12 ledger](../docs/history/status-A12.md) and
[A21 ledger](../docs/history/status-A21.md) show the accepted syntax; the runner entry point is
recorded in [tech-stack.md](tech-stack.md#harness-runner). Check the full active
ledger with that runner's read-only parser before launching an execution loop.

## Keep qualification chronology with its evidence owner

When a cross-repo qualification closes, retain exact pins, failed attempts,
report digests, and review history in its milestone or status record. Product,
architecture, and tech-stack should state the current fact each owns and link
to that evidence. Copying release chronology into all routine-read documents
repeated long passages and left older "current" headings beside newer work.
The [knowledge journal](../docs/history/knowledge.md) preserves the wording
removed during the September 24 consolidation, and [E20](../docs/history/status-E20.md)
illustrates the owning task evidence.
