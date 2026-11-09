# HexxlaDB capability coverage

Mosaic exposes seven core tools, or 22 tools with `advanced_tools: true`.
All names below omit `mosaic_hexxla_`. Runtime `tools/list` is authoritative.
The [policy inventory](../../internal/adapter/primary/mcpsrv/tool_policy.go)
classifies every tool and defines both startup profiles.

## Core

| Tool | Public database/application operation |
| --- | --- |
| `health` | Layout readers and file sizes; opt-in `HealthCheck` for full integrity/counts |
| `search_cells` | Lexical/hybrid `QueryCellsWithStats`; pure-vector `SearchByEmbedding` in semantic mode |
| `get_cell` | Exact current `GetCell`; explicit missing/tombstoned result |
| `load_context_pack` | `LoadContext`, followed by Mosaic byte budgeting |
| `put_cell` | `FindUnusedCellPlacement` and `PutCell` in one transaction; optional atomic `PutEmbedding` |
| `delete_cell` | `DeleteCellWithOutcome`, additionally controlled by write policy |
| `list_tags` | `ListExistingTopics` |

The previous query and embedding-discovery tools are replaced by search modes.
There is no implicit 32-ring search scope. Semantic mode retains pure vector
scores; filtered semantic discovery uses hybrid mode with its native composite
ranking. [Detailed mapping](HEXXLADB_API_NOTES.md)

## Advanced

| Tools | Purpose |
| --- | --- |
| `put_embedding` | Independent text/vector maintenance; optional atomic content-hash guard |
| `tag_counts` | `TagCounts` |
| `find_seams`, `mark_conflict`, `mark_supersedes`, `resolve_seam` | Bounded seam discovery and validated lifecycle operations |
| `put_facet`, `get_facet`, `list_facets` | Derived facet writes/reads; native guarded `UpdateFacet` remains outside MCP |
| `link_cells`, `get_edge`, `list_edges_from` | Typed graph records with bounded read results |
| `get_persistence_policy` | Application capture/deletion policy |
| `retrieval_budget_status` | Current session's JSON-egress meter |
| `estimate_context_budget_bytes` | Optional approximate-token conversion; context itself accepts bytes only |

## Native and operator boundaries

| Capability | Disposition |
| --- | --- |
| `Open`, `OpenExisting`, `Close`, `Options` | Command composition; existing-primary server startup |
| `View`, `Update` | Adapter-owned transaction boundaries |
| Raw KV and physical index walks | Native API, never arbitrary MCP byte keys |
| `BackupTo`, compaction, `ReclaimTail`, pruning, key rotation/recovery | Operator procedures with explicit destinations and recovery plans |
| Bulk cell transfer and batch ingestion | Operator/native API; generic MCP writes remain individually governed |
| FOV, Voronoi, super-hex summaries and rendering | Native specialized spatial/visualization APIs |
| `FindEdgePath`, edge walks, density and taxonomy analytics | Native; bounded agent experiments require a demonstrated use case |
| MVCC snapshots, diffs and changelog consumers | Native/operator; retention and consumer mutation are not general-agent actions |
| Hooks, record constructors, guarded facet update | Native integration options |

Mosaic intentionally does not mirror every database method. The general agent
surface excludes physical maintenance, retention mutation, migrations,
encryption, raw KV and changefeed cursor mutation.

## Maintenance

When changing a dependency or tool, verify the pinned public APIs, update this
map and runtime instructions, classify the tool in the inventory, and validate
Ratchet coverage for every enabled mutation. Core and advanced profiles must
both pass real MCP session tests. [Roadmap](../ROADMAP.md)
