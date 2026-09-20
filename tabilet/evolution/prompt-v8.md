# Prompt V8 - Desired-State Conversion Removal

OpenUdon should remove retired desired-state conversion documents, milestone
history, and public navigation from the OpenUdon repo.

Ramen is the desired-state conversion and reconciliation product. OpenUdon
should not preserve an OpenUdon conversion roadmap, status files, fixtures,
command docs, parser imports, or provider/resource mapping tracks. OpenUdon may
review and package UWS-facing artifacts generated elsewhere, but conversion
work itself belongs in Ramen.

Keep negative boundary checks that reject parser/conversion imports so the old
code cannot return by accident.
