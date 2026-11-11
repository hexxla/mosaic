# Roadmap

For completed work, see [`CHANGELOG.md`](../CHANGELOG.md).

## Release qualification

Current repository and stack maturity is alpha. The [release policy](../VERSIONING.md)
owns beta/v1 gates; the [qualification plan](RELEASE_QUALIFICATION.md) owns
candidate selection, distributable dependency checks, the 24-hour beta workload
and the 14-day named-owner operating pilot. Qualification and stabilization
are the next release milestone. No promotion evidence is recorded yet.

## Near-term

- **Large retrieval responses** — Context packs trim against the remaining session budget. Search and diagnostics rely on a final serialized-size check. Profile representative workloads before adding response shaping or pre-estimation.
- **Approximate metering and tokenizer alignment** — `approx_tokens_used` estimates token cost from JSON UTF-8 bytes. Explore clearer labeling or host tokenizer hooks if an agent workflow needs them.

## Future

Spec or direction exists; scheduling TBD.

- **Smarter cap semantics** — Broader follow-up to **oversized one-shot**: optional pre-estimate or tighter per-call clamps so session egress limits can fail **before** expensive DB work when a single response would overshoot.
- **Meter reset / TTL** — Operator or client-controlled boundaries for cumulative metering (per chat, per day, or explicit reset), if production needs diverge from “one MCP session ≈ one counter until session close or 30-minute idle expiry.”

## Future exploration

Interesting but unvalidated; needs product signal.

- **Metrics export** — Prometheus/OpenTelemetry counters for `approx_tokens_used` per session or globally (today: in-process + `mosaic_hexxla_retrieval_budget_status`).
- **Global/ranked continuation** — Regional browsing/export now has snapshot-pinned continuation. Global ranked queries still need a native stable cursor, tie-breaking and work/byte budgets; do not emulate them with offsets over changing results.
- **Bounded native diagnostics** — Snapshot diff materializes its narrow window, and storage statistics/snapshot-name enumeration lack scan cancellation. Add native memory/work budgets before expanding these agent limits.
- **Bounded super-hex summaries** — The native global derived-index rebuild lacks a cell/memory budget; add a bounded upstream contract before exposing it generally.
- **Legacy compatibility cleanup** — The alpha pins now provide typed retained-changefeed, embedding-count and read-sequence accessors. Remove the isolated compatibility scans once the minimum supported dependency policy no longer admits older explicit overrides.
- **Context orchestration** — Evaluate pluggable seed selection, multi-seed ranking, and host-provided packing strategies with representative agent queries.
- **Memory evolution** — Explore application-managed facet derivation and rotation, preference-aware ranking, and reviewable conflict detection and resolution. Measure retrieval coherence, resolution quality, reproducibility, and neighborhood latency before exposing automatic behavior.
- **Inspectability** — Evaluate ring and seam overlays, temporal navigation, and changefeed-backed collaboration against concrete operator workflows.

## Out of Scope

Intentional boundaries for Mosaic as a **local** MCP server.

- **Tokenizer-accurate “LLM context” accounting** — Mosaic meters **JSON egress** with a bytes-per-approx-token model; real tokenizer counts live in the host model runtime.
- **Relevance scoring inside the cap** — Session cumulative limits are **egress / runaway** controls, not substitutes for **per-call** ring and byte budgets or retrieval discipline (see [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](mosaic/MCP_AGENT_BLUEPRINT.md)).
