# HexxlaDB API coverage roadmap (Mosaic MCP)

**Purpose:** Track progress exposing [`github.com/hexxla/hexxladb`](https://github.com/hexxla/hexxladb) through Mosaic’s hexagonal stack (`domain` → `ports` → `services` → adapters → MCP tools). Order is iterative—finish vertical slices before widening surfaces.

**Working checklist & session tracker:** [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) — use it to record phase progress and what shipped each session.

**Canonical API:** [doc.go](https://github.com/hexxla/hexxladb/blob/main/doc.go), [API_REFERENCE.md](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/API_REFERENCE.md).

**Coverage matrix (Hexxla capability ↔ Mosaic tool):** [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md).

---

## Done

| Area | MCP tool | Notes |
| --- | --- | --- |
| Health / integrity | `mosaic_hexxla_health` | Full `HealthCheck` + `database_layout` |
| Structured reads | `mosaic_hexxla_query_cells` | `Tx.QueryCells` — tags, source, time (RFC3339), axial radius, sort, explain, max_scan_rows; optional **`embed_query_text`** → hybrid **`CellQuery.Embedding`** |
| Lexical search | `mosaic_hexxla_search_cells` | `Tx.SearchCells` — relevance scoring + filters, optional max_scan_radius; optional **`embed_query_text`** → hybrid **`CellSearchConfig.Embedding`** |
| Embeddings / ANN | `mosaic_hexxla_search_embedding` | Ollama query embed + `SearchByEmbedding` + cell payload via `GetCell`; response includes **`retrieval_hint`** to chain **`mosaic_hexxla_load_context_pack`** |
| Context pack | `mosaic_hexxla_load_context_pack` | `Tx.LoadContextPackFrom` — seeds from retrieval tools, byte budget, optional seams / supersession |
| Cell write | `mosaic_hexxla_put_cell` | `Tx.PutCell` — `kind`-selected templates + merged tags |
| Embedding write | `mosaic_hexxla_put_embedding` | `Tx.PutEmbedding` — Ollama text path or raw `[]float32` (DB dimension) |
| Cell delete | `mosaic_hexxla_delete_cell` | `Tx.DeleteCell` — MVCC tombstone semantics |
| Seams (read) | `mosaic_hexxla_find_seams` | `Tx.FindSeams` — optional `unresolved_only`; radius default 3 |
| Seams (write) | `mosaic_hexxla_mark_conflict` | `Tx.MarkConflict` — ULID seam, canonical endpoints |
| Seams (write) | `mosaic_hexxla_mark_supersedes` | `Tx.MarkSupersedes` — `SeamTypeSupersedes` for context-pack filtering |
| Seams (write) | `mosaic_hexxla_resolve_seam` | `Tx.ResolveSeam` — by ULID |
| Facets | `mosaic_hexxla_put_facet` | `Tx.PutFacet` — facet_id 0..5; uses **`NewFacetDerived`** (HexxlaDB export) |
| Facets (read) | `mosaic_hexxla_get_facet` | `Tx.GetFacet` — read slot 0..5 (read-only `View`) |
| Facets (read) | `mosaic_hexxla_list_facets` | `Tx.AscendFacetsForCell` — all facets at (q,r) (read-only `View`) |
| Edges | `mosaic_hexxla_link_cells` | `Tx.LinkCells` → `PutEdge` — relation_type, weight, **`NewProvenanceWire`** |
| Edges (read) | `mosaic_hexxla_get_edge` | `Tx.GetEdge` — exact (from,to,relation_type) (read-only `View`) |
| Edges (read) | `mosaic_hexxla_list_edges_from` | `Tx.AscendEdgesFrom` — capped `max_edges` (read-only `View`) |
| Tag discovery | `mosaic_hexxla_list_tags` | `Tx.ListExistingTopics` — distinct sorted tags (read-only `View`) |
| Tag analytics | `mosaic_hexxla_tag_counts` | `Tx.TagCounts` — per-tag frequencies (read-only `View`) |

---

## Planned (methodical pass)

| Priority | Group | Examples (Hexxla) | Direction |
| --- | --- | --- | --- |
| 1 | Reads | *(shipped: `QueryCells` + `SearchCells` via MCP)* | — |
| 2 | Context | *(shipped: `LoadContextPackFrom` via MCP)* — optional: other `LoadContext*` entry points | — |
| 3 | Writes | *(shipped via `mosaic_hexxla_put_cell` / `mosaic_hexxla_put_embedding` / `mosaic_hexxla_delete_cell`)* | — |
| 4 | Seams | *(shipped: `FindSeams`, `MarkConflict`, `MarkSupersedes`, `ResolveSeam` via MCP)* — optional: `FindSeamsAt`, bespoke `PutSeam` | — |
| 5 | Facets / edges | *(writes + reads shipped via MCP; optional: `UpdateFacet` hash-guard)* | — |
| 6 | Ops | `Compact`, changelog readers, MVCC prune hints | **Default MCP excludes** — compaction/retention are maintenance risks; keep for operator CLI / separate auth. See [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md) § Phase 6 stance |

---

## Conventions

- **Ports** stay free of `hexxladb` types in signatures where feasible; DTOs live in `domain` or small command structs.
- **Secondary adapter** is the only layer that imports `github.com/hexxla/hexxladb`.
- **Ollama** stays at the edge (`internal/ollama`, env `MOSAIC_OLLAMA_URL` / `MOSAIC_EMBED_MODEL`), never in `core/domain`.

See also [HEXXLADB_API_NOTES.md](./HEXXLADB_API_NOTES.md) and [MCP_BLUEPRINT.md](./MCP_BLUEPRINT.md).
