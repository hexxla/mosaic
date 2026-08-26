# Hexxla API surface ↔ Mosaic MCP coverage

**Purpose:** Map [`github.com/hexxla/hexxladb`](https://pkg.go.dev/github.com/hexxla/hexxladb)’s public capabilities (summarized in upstream [`doc.go`](https://github.com/hexxla/hexxladb/blob/main/doc.go), [`docs/hexxladb/API_REFERENCE.md`](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/API_REFERENCE.md)) to what **Mosaic** exposes through **`cmd/mosaic-mcp`** tools. Use this to spot gaps, avoid duplicating docs, and separate **LLM-callable** surfaces from **operator-only** operations.

**Companion:** [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) (phases / session log), [HEXXLADB_API_NOTES.md](./HEXXLADB_API_NOTES.md) (hexagonal mapping).

---

## Legend

| Mosaic column | Meaning |
| --- | --- |
| MCP tool name | Wired in Mosaic today |
| *(not exposed)* | No Mosaic MCP tool; call Hexxla from Go/CLI/integration |
| Operator / risk | **Intentionally** not MCP-default: destroys layout, compaction, retention, or ambiguous trust — use dedicated operator path |

---

## Engine shell & lifecycle

| Hexxla capability | Mosaic |
| --- | --- |
| `Open`, `Options`, `DB.Close` | Composition root [`cmd/mosaic-mcp/main.go`](../../cmd/mosaic-mcp/main.go); env **`MOSAIC_DB_PATH`** |
| `Compact`, `CompactTo` | *(not exposed)* — **Operator / risk** (I/O heavy, maintenance window) |
| `HealthCheck` | `mosaic_hexxla_health` |
| `DB` layout readers (`PageSize`, `MaxValueBytes`, `EmbeddingDimension`, embedding metric, etc.) | Embedded in **`mosaic_hexxla_health`** JSON (`database_layout`, **`disk`** primary/WAL sizes, **`mvcc_retain_commits_behind_head`**, full report fields) |

---

## Transactions & KV

| Hexxla capability | Mosaic |
| --- | --- |
| `View` / `Update` / transactional `Get`/`Put`/`AscendRange` generic KV | *(not exposed)* — Mosaic stays lattice/embeddings/domain APIs |

---

## Cells, lattice, primitives

| Hexxla capability | Mosaic |
| --- | --- |
| `PutCell`, `FindFreeCellPlacement`, `DeleteCell`, `GetCell` | `mosaic_hexxla_put_cell` supports safe exact placement or bounded `near_anchor` allocation and returns the actual coordinate; `mosaic_hexxla_delete_cell`; reads via **`query_cells`** / **`search_cells`** / **`search_embedding`** |
| `WalkRing` | *(not exposed directly)* |
| `LoadContext` unified assembled retrieval | `mosaic_hexxla_load_context_pack`; HexxlaDB bounds candidates and Mosaic applies provider-neutral byte budgeting ([design](./MCP_AGENT_BLUEPRINT.md)) |
| `LoadContextFOV`, `LoadContextVoronoi` | *(not exposed)* |
| Secondary tag reads: `ListExistingTopics`, `TagCounts` | `mosaic_hexxla_list_tags`, `mosaic_hexxla_tag_counts` |
| `AscendCellsByTag`, `AscendDistinctTags`, other index walks | Partially via **`query_cells`** filters; full tag iteration not a separate tool |
| `QueryCells`, `SearchCells` | `mosaic_hexxla_query_cells`, `mosaic_hexxla_search_cells`; optional **`embed_query_text`** wires **`CellQuery.Embedding`** / **`CellSearchConfig.Embedding`** (Ollama + hybrid ANN with same predicates). |

---

## Embeddings

| Hexxla capability | Mosaic |
| --- | --- |
| `PutEmbedding`, `DeleteEmbedding` | `mosaic_hexxla_put_embedding` (text via Ollama or raw vector); delete not split out — use cell APIs if policy allows |
| `SearchByEmbedding` | `mosaic_hexxla_search_embedding` (Ollama query embed + ANN) |
| `ReindexEmbeddings` | *(not exposed)* — **Operator / risk** |

---

## Seams

| Hexxla capability | Mosaic |
| --- | --- |
| `FindSeams` | `mosaic_hexxla_find_seams` |
| `MarkConflict`, `MarkSupersedes`, `ResolveSeam` | `mosaic_hexxla_mark_conflict`, `mosaic_hexxla_mark_supersedes`, `mosaic_hexxla_resolve_seam` |
| `PutSeam` (raw ULID body) | *(not exposed)* — helpers cover MCP story |
| `FindSeamsAt` | *(not exposed)* |

---

## Facets & edges

| Hexxla capability | Mosaic |
| --- | --- |
| `PutFacet` (via `NewFacetDerived`) | `mosaic_hexxla_put_facet` |
| `LinkCells` / `PutEdge` (via `NewProvenanceWire`) | `mosaic_hexxla_link_cells` |
| `GetFacet` | `mosaic_hexxla_get_facet` |
| `AscendFacetsForCell` | `mosaic_hexxla_list_facets` |
| `GetEdge` | `mosaic_hexxla_get_edge` |
| `AscendEdgesFrom` | `mosaic_hexxla_list_edges_from` (**`max_edges`** default 50 / cap **200**, **`truncated`**) |

---

## MVCC, changelog, encryption, retention

| Hexxla capability | Mosaic |
| --- | --- |
| `StatsMVCC`, `PruneCellVersions`, `SuggestedPruneBeforeSeq`, `PruneScheduler` | *(not exposed)* — **Operator / risk** (data retention / surprise deletes) |
| `SnapshotDiff` | *(not exposed)* |
| `ReadChangelogSince`, logical changefeed | *(not exposed)* — **Operator / risk** if mis-scoped; read-only **peek** could be a future gated tool with strict bounds |
| `RotateEncryption` | *(not exposed)* — **Operator / risk** |

---

## Phase 6 stance (Mosaic)

Planned “operator / advanced” items ([`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md)) include **compaction**, **changelog readers**, and **MVCC hints**. They are **not** shipped on the default MCP surface because:

- **Compaction** alters on-disk layout and timing; wrong automation can starve writers or mask backup timing — keep for **CLI / maintenance** or a separate authenticated control plane.
- **Prune / retention** can remove history the user expected to keep — **never** implied-safe for casual LLM triggers.
- **Changelog** reads are valuable for audits but must be **bounded** (seq ranges, size caps) and ideally **operator-gated** if exposed at all.

**Read-only tag listing** (`list_tags`, `tag_counts`) is the opposite: **View**-only, no mutations, strong value for consistent tagging and search vocabulary.

---

## Maintenance

When Hexxla adds exported APIs or Mosaic ships new MCP tools, update:

1. This table (accuracy).
2. [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) session log when adding tools.
3. [`.cursor/rules/mosaic-mcp-agent.mdc`](../../.cursor/rules/mosaic-mcp-agent.mdc) tool table (agent workflow).
