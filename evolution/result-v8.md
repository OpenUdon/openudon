# Result V8 - Desired-State Conversion Removal

OpenUdon now treats desired-state conversion as Ramen-owned with no retained
OpenUdon conversion track.

The stale provider conversion corpus page, old conversion status files,
conversion milestone sections, conversion navigation, and the retired
API-source conversion evolution record were removed. Product, architecture,
tech-stack, related-project, README, AGENTS, and release docs now describe
OpenUdon as a UWS authoring/review/package/handoff tool only.

The repository boundary check still rejects parser/conversion imports. That is
a negative guard, not an OpenUdon conversion feature.
