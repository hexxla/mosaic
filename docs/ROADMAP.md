# Roadmap

For completed work, see [`CHANGELOG.md`](../CHANGELOG.md).

## Release qualification

Current repository and stack maturity is alpha. The [release policy](../VERSIONING.md)
owns beta/v1 gates; the [qualification plan](RELEASE_QUALIFICATION.md) owns
candidate selection, distributable dependency checks, the 24-hour beta workload
and the 14-day named-owner operating pilot. Qualification and stabilization
are the next release milestone.

As of 2026-11-11, the three-repository baseline/recovery checks and the
30-minute / 1-hour lexical soaks passed. The owner is void. The short stages
completed 2,144 / 2,084 successful MCP calls, with three verified maintenance
cycles each, and the 1-hour stage verified session expiry after 31 minutes idle.
See the [dated evidence report](qualification/2026-11-11-baseline-short-soaks.md)
for exact sources, results, limitations and local artifact custody. The 24-hour
stage, live embeddings and 14-day pilot remain pending; beta gates remain unmet.

## Near-term

Recommended order; implementation remains separate from this roadmap update.

1. **Complete scoped qualification** — Run/review the 24-hour workload. Before
   vector-enabled claims, configure the bounded Ollama service, pin provider/model
   identity and repeat short stages with real embeddings. Exercise representative
   concurrent callers/readers, advanced operations, custom policies and failure
   outcomes required by the beta gates. The small serial lexical runner alone
   does not establish those claims. Complete applicable capacity evidence, then
   the named-owner pilot and v1 contract review.
2. **Sustainable maintenance integration** — Track upstream consumer-coordinated
   feed archival/reclamation and history-preserving encrypted compaction. Current
   primary compaction does not shrink the append-only feed; the encrypted
   candidate cannot adopt its old sidecar. Keep these limits explicit and require
   matched backups/oracle checks, resource bounds and preserved consumer progress;
   never reset history implicitly to satisfy a disk budget.
3. **Legacy compatibility cleanup** — Decide and document the minimum supported
   HexxlaDB library version. The alpha pin already provides typed retained-feed,
   embedding-count and read-sequence APIs. Once older explicit overrides are
   unsupported, remove fallback interface checks/physical-key scans and stale
   comments. Keep accepted on-disk-format recovery fixtures; that compatibility
   is separate from old library overrides.
4. **Bounded retrieval responses** — Consolidate large-response handling and
   smarter cap semantics: context packs trim to remaining session budget, while
   search/diagnostics still rely on final serialization checks. Profile real
   workloads, then add response shaping, pre-estimates or tighter per-call clamps
   where they avoid expensive work for a response that cannot fit. Broader native
   diagnostics/rebuild exposure first requires upstream work/memory budgets.
5. **Metering clarity** — Label `approx_tokens_used` consistently as an estimate
   derived from JSON UTF-8 bytes. Improve operational visibility when a workflow
   needs it; model-tokenizer-accurate accounting remains the host's responsibility.

## Future

Spec or direction exists; scheduling TBD.

- **Meter reset / TTL** — Operator or client-controlled boundaries for cumulative metering (per chat, per day, or explicit reset), if production needs diverge from “one MCP session ≈ one counter until session close or 30-minute idle expiry.”

## Future exploration

Deferred and unscheduled. Require a concrete use case, bounded experiment and
measurable benefit before promoting an item to implementation.

- **Metrics export** — Prometheus/OpenTelemetry counters for `approx_tokens_used` per session or globally (today: in-process + `mosaic_hexxla_retrieval_budget_status`).
- **Global/ranked continuation** — Regional browsing/export now has snapshot-pinned continuation. Global ranked queries still need a native stable cursor, tie-breaking and work/byte budgets; do not emulate them with offsets over changing results.
- **Bounded native diagnostics** — Snapshot diff materializes its narrow window, and storage statistics/snapshot-name enumeration lack scan cancellation. Add native memory/work budgets before expanding these agent limits.
- **Bounded super-hex summaries** — The native global derived-index rebuild lacks a cell/memory budget; add a bounded upstream contract before exposing it generally.
- **Context orchestration** — Evaluate pluggable seed selection, multi-seed ranking, and host-provided packing strategies with representative agent queries.
- **Memory evolution** — Explore application-managed facet derivation and rotation, preference-aware ranking, and reviewable conflict detection and resolution. Measure retrieval coherence, resolution quality, reproducibility, and neighborhood latency before exposing automatic behavior.
- **Inspectability** — Evaluate ring and seam overlays, temporal navigation, and changefeed-backed collaboration against concrete operator workflows.

## Out of Scope

Intentional boundaries for Mosaic as a **local** MCP server.

- **Tokenizer-accurate “LLM context” accounting** — Mosaic meters **JSON egress** with a bytes-per-approx-token model; real tokenizer counts live in the host model runtime.
- **Relevance scoring inside the cap** — Session cumulative limits are **egress / runaway** controls, not substitutes for **per-call** ring and byte budgets or retrieval discipline (see [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](mosaic/MCP_AGENT_BLUEPRINT.md)).
