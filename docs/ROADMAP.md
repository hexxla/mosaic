# Roadmap

For completed work, see [`CHANGELOG.md`](../CHANGELOG.md).

## Near-term

Engineering polish aligned with Mosaic as the MCP orchestration layer over HexxlaDB. Details and checkboxes live in [`TODOS.md`](../TODOS.md); echoed here as roadmap themes.

- **Large retrieval responses** — Context packs already trim against the remaining session budget. Query, search, and diagnostics still rely on a final serialized-size check. Profile representative workloads before adding response shaping or pre-estimation. See [`TODOS.md`](../TODOS.md).
- **Approximate metering vs tokenizer** — `approx_tokens_used` is JSON UTF‑8 length ÷ `bytes_per_approx_token`, not the host tokenizer. **Explore:** labeling vs optional hooks. [`TODOS.md`](../TODOS.md) *(Pending)*.

## Future

Spec or direction exists; scheduling TBD.

- **Smarter cap semantics** — Broader follow-up to **oversized one-shot**: optional pre-estimate or tighter per-call clamps so session egress limits can fail **before** expensive DB work when a single response would overshoot.
- **Meter reset / TTL** — Operator or client-controlled boundaries for cumulative metering (per chat, per day, or explicit reset), if production needs diverge from “one MCP session ≈ one counter until session close or 30-minute idle expiry.”

## Future exploration

Interesting but unvalidated; needs product signal.

- **Metrics export** — Prometheus/OpenTelemetry counters for `approx_tokens_used` per session or globally (today: in-process + `mosaic_hexxla_retrieval_budget_status`).
- **Bounded taxonomy analytics (recommended first read-only experiment)** — Evaluate `TagCooccurrences(min_count)` and radius-bounded `UntaggedCells` tools. They directly improve tag reuse and memory hygiene, require no Ollama, and can be hard-capped by minimum count, radius, and result count.
- **Bounded relationship traversal (recommended second read-only experiment)** — Evaluate `FindEdgePath` and `WalkEdgeCoords` behind strict `max_hops`, `max_nodes`/`max_coords`, relation filters, and response-size caps. Validate a real agent navigation workflow before exposing weighted-path details.
- **Lattice density diagnostics** — Evaluate a small `RingDensityMap` response for placement/coverage decisions. Keep rendered grids and super-hex derived indexes in native/operator tooling unless a concrete agent workflow demonstrates value.
- **Snapshot/time-travel reads** — Evaluate list/read/diff operations only after specifying retained-history availability, authorization, maximum sequence/time span, and response caps. Snapshot creation/deletion remains operator-only.
- **Bounded changelog peek** — Evaluate `ReadChangelogFiltered` only for a demonstrated audit workflow with strict sequence, operation, record-count, and byte limits. Consumer cursor mutation and retention coordination remain operator-only.

## Out of Scope

Intentional boundaries for Mosaic as a **local** MCP server.

- **Tokenizer-accurate “LLM context” accounting** — Mosaic meters **JSON egress** with a bytes-per-approx-token model; real tokenizer counts live in the host model runtime.
- **Relevance scoring inside the cap** — Session cumulative limits are **egress / runaway** controls, not substitutes for **per-call** ring and byte budgets or retrieval discipline (see [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](mosaic/MCP_AGENT_BLUEPRINT.md)).
