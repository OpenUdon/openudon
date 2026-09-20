# Prompt V9 - CLI-First Public Beta

OpenUdon should publish its first public release as v0.1.0 after repairing the
standalone lifecycle dependency failure.

The release should support the deterministic package, approval,
trusted-handoff, and run-evidence CLI/artifact boundary through v0.1.x while
keeping iCoT/LLM behavior, prompts, catalog/eval/readiness/smoke helpers, and
provider behavior experimental. OpenUdon should not claim a supported public
Go-library API while its implementation remains internal.

Release archives should bundle `openudon`, `icot`, and `udon-runner` for Linux,
macOS, and Windows on amd64 and arm64, publish checksums, and prove a
credential-free author/build/assess/approval/dry-run path. Tagging requires
standalone public-module gates plus the provider-free local udon smoke; live
provider calls are not required.
