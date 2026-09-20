# Prompt V7 - Ramen/OpenUdon Governance Sharing Review

OpenUdon should add a milestone to compare its approval, review-handoff,
run-config, and run-evidence contracts with Ramen's governance, plan approval,
and run approval artifacts.

The direction is contract review first, not shared-code extraction. Keep
command code and product logic in each repo. Do not make OpenUdon import Ramen,
do not move Ramen `run`, and do not create a broad shared module before a
stable common approval/governance core is proven.

The first implementation slice should be documentation and milestone tracking:
create the next OpenUdon milestone, record task rows for the comparison, and
clarify sibling boundaries. Code, schemas, command behavior, and module
dependencies should remain unchanged.
