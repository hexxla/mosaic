# HexxlaDB capability coverage

Mosaic exposes seven core tools, or 43 tools with `advanced_tools: true`.
Names below omit `mosaic_hexxla_`. Runtime `tools/list` is authoritative.
The [policy inventory](../../internal/adapter/primary/mcpsrv/tool_policy.go)
classifies every tool and defines both startup profiles. New mutation tools
must appear in the bundled Ratchet policy and in custom policies that enable them.

## Core

| Tool | Native/application coverage |
| --- | --- |
| `health` | Layout and disk sizes; opt-in native integrity/count checks |
| `search_cells` | Lexical/hybrid `QueryCellsWithStats`, semantic `SearchByEmbeddingWithStats`; structured predicates, ordering, explanations, row/candidate caps and ANN breadth |
| `get_cell` | Exact current read with presence, timestamps, validity and cluster hint |
| `load_context_pack` | Radial/multi-seed or relation-filtered graph `LoadContext`; validity time, edges, facets, seams and supersession, followed by Mosaic byte budgeting |
| `put_cell` | Exact or unused-near-anchor placement and optional embedding in one transaction; metadata overrides, exact tags, fact/user/assistant/system templates |
| `delete_cell` | `DeleteCellWithOutcome`, controlled by deletion policy |
| `list_tags` | Native topic discovery |

Search defaults to 20 results and caps at 100. Lexical scans default to 20,000
rows. Semantic scores remain pure vector scores; structured filters use hybrid
mode. `embedding_candidate_limit` selects a hybrid candidate ceiling up to
10,000 and cannot accompany `max_scan_rows`; `ef_search` selects semantic ANN
breadth up to 10,000. Raw query vectors require the recorded `embedding_model`
identity. Hybrid responses report candidate exhaustion; semantic responses
report native execution path and effective ANN breadth.

Search and context accept optional `snapshot` with exactly one of `seq`, `tag`
or RFC3339 `at`. Historical search is lexical; historical context cannot
traverse or enrich edges. `as_of` separately filters domain validity. Native
snapshot expiry and historical-data errors remain errors, never current-data
fallbacks. Context text budgeting counts raw/facet text and included relation
labels; session metering counts complete serialized JSON separately.

On overwrite, omitted tags, creation time, validity and cluster hints survive.
`metadata` permits explicit overrides; `clear_validity` and `clear_cluster_hint`
remove those fields, and `replace` clears validity/cluster metadata before
applying overrides. `exact_tags` suppresses template tags. Updated time defaults
to now; explicit timestamps must fit native nanosecond storage. Near-anchor
writes require durable idempotency keys. [Write/retry contract](MCP_AGENT_BLUEPRINT.md#cell-placement-on-writes)

## Advanced

| Tools | Coverage and operational limits |
| --- | --- |
| `put_embedding`, `get_embedding`, `delete_embedding` | Current vector lifecycle; raw writes declare model identity, writes optionally guard source content hash; delete preserves the cell |
| `rebuild_embedding_index` | Derived HNSW rebuild, default 1000 vectors/128 MiB, ceilings 10000/512 MiB; concurrent changes reject publication |
| `tag_counts` | Native global counts; collection may scan the database |
| `tag_cooccurrences` | Bounded sample of at most 1000 cells and 64 tags/cell; at most 10000 distinct counted pairs and 100 returned; `partial` identifies sampling/truncation |
| `untagged_cells` | Regional discovery, radius at most 16, configurable scan cap and at most 100 returned coordinates |
| `find_seams`, `mark_conflict`, `mark_supersedes`, `resolve_seam`, `put_seam` | Regional discovery including validity/source filtering, convenience lifecycle and explicit caller-identified custom seam records |
| `put_facet`, `get_facet`, `list_facets` | Native facet slots and metadata; optional source hash invokes guarded `UpdateFacet`; omission explicitly writes unconditionally |
| `link_cells`, `get_edge`, `list_edges_from` | Directed typed graph records; outgoing listing is bounded and reports truncation |
| `walk_graph`, `find_path` | Directed exploration with node and adjacency-row budgets; BFS additionally limits hops; weighted paths use native edge costs |
| `load_fov`, `load_voronoi` | Native spatial context with declarative blockers/uniform region costs, radius at most 16 and at most 100 returned cells; up to 8 Voronoi seeds |
| `ring_density`, `render_grid` | Native occupancy and textual diagrams, radius ceilings 16 and 10 |
| `list_snapshots`, `get_cell_at` | Up to 500 names plus retention floor; exact reads by sequence, tag or commit time |
| `snapshot_diff` | Retained cell/seam diagnostics over at most 64 commits, at most 100 returned changes, five-second cooperative deadline; native collection materializes the selected window |
| `read_changes` | Up to 128 examined records (32 with inline payloads), operation filters and advancing sequence; no consumer acknowledgement |
| `browse_cells`, `export_cells`, `read_cell_chunk` | Resumable regional enumeration and complete cell-record/content chunks at one retained snapshot; see transfer contract below |
| `put_cells` | Atomic batch of 1–32 individually keyed near-anchor captures, at most 1 MiB text/tags; same policy, metadata, embedding and receipt rules as individual puts |
| `storage_diagnostics` | Typed physical/live/reusable storage accounting, write-phase timing, WAL counters, model identity and oldest retained sequence |
| `get_persistence_policy`, `retrieval_budget_status`, `estimate_context_budget_bytes` | Capture policy, session meter and optional approximate-token conversion |

Every new read goes through session metering. Storage statistics and snapshot
name enumeration use native APIs that lack cooperative scan cancellation;
they can hold the read lock and delay writers. A diff response reports
`history_pruned` when its start predates the current retention floor. A
truncated diff has no continuation; narrow the window. It is not complete CDC.
Changefeed records use opaque base64 keys/payloads, at-least-once semantics,
and hash/length-only representation when the native feed omits large payloads.

## Transfer and continuation

Regional browsing/export enumerates nearest rings, radius 0–128, without
retaining a result set or writing to a network client under the DB lock. Pages
bound cells/deferred identities (default 50, max 100), coordinate probes
(default 1000, max 2000) and cell JSON bytes (default 64000, 1024–100000).
Empty pages may advance through sparse regions. This does not paginate ranked
search or claim to enumerate coordinates outside the requested region.

Continuation is authenticated, bound to its MCP session and tool, and expires
after five minutes or a server restart. Repeating a cursor is safe; concurrent
writes do not change the selected snapshot, but pruning can expire it. When a
cell exceeds the page budget, its identity appears in `deferred_cells` rather
than being silently lost. `read_cell_chunk` transfers up to 8192 raw bytes in
base64 at an explicit snapshot: `content` or a complete native `record_json`.
Collect chunks and verify their full-document SHA256. Cell transfer excludes
facets, seams, edges, vectors, application keys and physical recovery state;
use native backup for a complete database copy.

## Configuration and compatibility

Database creation supports native page/value limits, vector dimensions and all
three metrics. Runtime configuration exposes cache size, primary fdatasync,
changelog lazy mode, credentials and retention/pruning policy. Compression and
authenticated encryption are native engine behavior.

Mosaic persists model identity, requires explicit `-reembed` on change, supports
optional provider digest pinning, and permits safe `-lexical-only` access to
unknown native vector stores. [Configuration](MOSAIC_CONFIG.md#ollama-optional)

The published HexxlaDB dependency pin remains supported. New HexxlaDB builds
provide `HasRetainedChangefeed`, `CountEmbeddings` and `ReadSequence`; Mosaic
uses these typed APIs when present. The older pin retains isolated compatibility
scans for changefeed/vector status and a full MVCC count to acquire the initial
browse sequence. Remove those fallbacks after publishing and adopting the new
minimum dependency. A local workspace is not required to build Mosaic.

## Native and operator boundaries

| Capability | Disposition and reason |
| --- | --- |
| Open/close and transactions | Command/adapter-owned lifecycle; no remote transaction handles or callbacks |
| Raw KV, physical index keys, hooks/validators | Native embedding contracts; no arbitrary agent-supplied keys or executable callbacks |
| Backup, compaction, repair, reclaim, key rotation/recovery | HexxlaDB operator tooling with explicit destinations, exclusive ownership and recovery plans |
| Snapshot-tag and changefeed-consumer mutations/history reset | Operator-owned retention and coordination; MCP exposes inspection only |
| Native import and arbitrary large batch ingestion | Operator APIs; MCP supplies bounded atomic captures and regional transfer |
| Super-hex derived indexes | Native; current global rebuild has no cell/memory budget and needs a bounded API before general-agent exposure |
| Ranked/global query cursors | Not available natively; regional transfer has separate honest continuation semantics |
| Historical edges/vectors | Unavailable natively; Mosaic preserves the rejection |
| FOV opacity callbacks / Voronoi weight callbacks | MCP exposes declarative blockers/uniform weights, not executable code |

## Verification

Tests cover metadata round-trips, guarded facet rejection, model swaps/digest
retagging, lexical-only admission, graph context/validity, historical search,
filtered changefeed progress, cursor tampering/session isolation/snapshot
continuity, oversized record chunks, atomic batch rollback/replay, tool schemas
and Ratchet-governed mutations. Run `task ci` with published pins, and run the
cross-repository suite with an explicit temporary Go workspace when changing
native capability APIs. Both dependency paths must remain usable.
