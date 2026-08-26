# Mosaic MCP — implementation plan & progress tracker

**Purpose:** Single working checklist for exposing HexxlaDB through Mosaic’s hexagonal stack and MCP tools. Update this file as slices land so sessions stay aligned.

**Companion docs:** [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md) (capability matrix), [HEXXLADB_API_NOTES.md](./HEXXLADB_API_NOTES.md) (hexagonal mapping), [MCP_BLUEPRINT.md](./MCP_BLUEPRINT.md) (MCP server architecture), [MCP_AGENT_BLUEPRINT.md](./MCP_AGENT_BLUEPRINT.md) (agent workflows; hybrid **`embed_query_text`**).

---

## Status at a glance

| Field | Value |
| --- | --- |
| **Active phase** | Phase 6 — operator-only candidates (Compaction / prune / changelog remain off MCP) |
| **Last meaningful update** | 2026-08-26 — HexxlaDB v0.6.0 compatibility, provider-neutral byte budgeting, and bounded cell placement |
| **MCP tools shipped** | 23 tools: `mosaic_hexxla_health`, `mosaic_hexxla_query_cells`, `mosaic_hexxla_search_cells`, `mosaic_hexxla_search_embedding`, `mosaic_hexxla_load_context_pack`, `mosaic_hexxla_estimate_context_budget_bytes`, `mosaic_hexxla_retrieval_budget_status`, `mosaic_hexxla_put_cell`, `mosaic_hexxla_put_embedding`, `mosaic_hexxla_delete_cell`, `mosaic_hexxla_find_seams`, `mosaic_hexxla_mark_conflict`, `mosaic_hexxla_mark_supersedes`, `mosaic_hexxla_resolve_seam`, `mosaic_hexxla_put_facet`, `mosaic_hexxla_link_cells`, `mosaic_hexxla_get_facet`, `mosaic_hexxla_list_facets`, `mosaic_hexxla_get_edge`, `mosaic_hexxla_list_edges_from`, `mosaic_hexxla_list_tags`, `mosaic_hexxla_tag_counts`, `mosaic_hexxla_get_persistence_policy` |
| **Offline / CLI** | `cmd/mosaic-seed` — 82-turn `conversational_memory` corpus + Ollama embeddings |

---

## How to use this document

1. Before starting work: set **Active phase** and optionally add a one-line note under **Session log**.
2. When a checkbox is done: mark `[x]`, adjust **Active phase** if you move on.
3. After user-visible behavior changes: add a line under [CHANGELOG.md](../../CHANGELOG.md) `[Unreleased]` and bump **Last meaningful update**.
4. Prefer **one vertical slice** per PR/session (port → service → adapter → MCP tool → tests), not half-finished layers.

---

## Phase 0 — Baseline (complete)

Foundation already in the repo; keep these stable when adding phases.

- [x] Hexagonal layout (`core/domain`, `ports`, `services`, `adapter/primary/mcpsrv`, `adapter/secondary/hexxlastore`)
- [x] MCP Streamable HTTP server (`cmd/mosaic-mcp`), loopback config (`internal/config`)
- [x] `mosaic_hexxla_health` → `HealthCheck` + layout summary
- [x] `mosaic_hexxla_search_embedding` → Ollama embed + `SearchByEmbedding` + `GetCell`
- [x] `cmd/mosaic-seed` — spiral coords, `PutCell` + `PutEmbedding`, env `MOSAIC_OLLAMA_URL` / `MOSAIC_EMBED_MODEL`
- [x] CI (`make ci`) and tests on touched packages

---

## Phase 1 — Reads (priority)

**Goal:** LLM-callable **structured retrieval** without relying only on embedding search.

**Hexxla API (indicative):** `QueryCells`, `SearchCells`, tag/source/time oriented scans — confirm signatures against current [`API_REFERENCE.md`](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/API_REFERENCE.md).

### Design

- [x] Domain DTOs: `CellQueryCommand`, `CellSearchCommand`, `CellHit`, `CellHitsResponse` in `internal/core/domain/cell_reads.go`
- [x] `secondary.CellReader` — `QueryCells` / `SearchCells` (pointer command args for gocritic)
- [x] `hexxlastore.CellReaderAdapter` — `DB.View` + `Tx.QueryCells` / `Tx.SearchCells`
- [x] `primary.CellRetrieval` + `services.CellRetrievalService` (sort validation, max-results cap 100, defaults)

### MCP

- [x] **`mosaic_hexxla_query_cells`** → `QueryCells` (tags, time RFC3339, spatial center+radius, sort, explain, max_scan_rows)
- [x] **`mosaic_hexxla_search_cells`** → `SearchCells` (lexical relevance + filters, max_scan_radius)
- [x] Documented in [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md)

### Quality

- [x] Service tests with stub `CellReader`; MCP helpers tested (`cell_time_test.go`); `make ci` green
- [x] **HEXXLA_API_SURFACE_COVERAGE** matrix updated
- [x] Changelog `[Unreleased]` updated

**Phase 1 exit criteria:** At least one MCP read path exercises **non-embedding** cell discovery against the seeded DB; `make ci` green.

---

## Phase 2 — Context assembly

**Goal:** Multi-seed context packs with storage-neutral retrieval and application-owned budgeting.

- [x] One MCP tool **`mosaic_hexxla_load_context_pack`** (`Tx.LoadContext` candidates; Mosaic UTF-8 byte budgeting)
- [x] Domain: `LoadContextPackCommand`, `ContextPackResponse`, stats/cells/seams summaries; hints in [`retrieval_hints.go`](../../internal/core/domain/retrieval_hints.go)
- [x] `secondary.ContextPackLoader` / `primary.ContextAssembly` / `ContextAssemblyService` (defaults: ring 3, 4096 bytes; caps seeds 32, budget 100k bytes)
- [x] Retrieval tools populate **`retrieval_hint`** JSON + expanded **tool descriptions** so models chain embed/query → context pack when needed

HexxlaDB remains provider-neutral: it bounds assembled candidates by count. Mosaic owns approximate-token conversion, UTF-8 byte accounting, and outer-ring/low-confidence eviction; exact final-request tokenization remains a client responsibility. See [MCP_AGENT_BLUEPRINT.md](./MCP_AGENT_BLUEPRINT.md).

**Exit criteria:** Caller passes seed coords (from retrieval) and receives a budgeted pack — **done**.

---

## Phase 3 — Writes (non-seed)

**Goal:** Controlled mutations outside `mosaic-seed`.

- [x] `PutCell` path — validation, tags, source id, `kind` (`fact` \| `user_message` \| `assistant_response`); safe exact writes require explicit overwrite, while bounded `near_anchor` placement composes `FindFreeCellPlacement` + `PutCell` atomically and returns the actual coordinate
- [x] `PutEmbedding` path — **text** (`secondary.TextEmbedder` / Ollama) or raw **vector**; length must match `DB.EmbeddingDimension()`
- [x] `DeleteCell` — semantics in MCP tool description (MVCC tombstone)
- [x] MCP tools ([`cell_mutation_tool.go`](../../internal/adapter/primary/mcpsrv/cell_mutation_tool.go)) + service tests + changelog

**Exit criteria:** Documented, tested write/delete tools with clear error surfaces; no secrets in logs. **Done.**

---

## Phase 4 — Seams

**Goal:** Contradiction and supersession as first-class operations.

- [x] Domain rules for seam intent — [`internal/core/domain/seam.go`](../../internal/core/domain/seam.go) header comment + DTOs
- [x] **`FindSeams`** (read), **`MarkConflict`**, **`MarkSupersedes`**, **`ResolveSeam`** (writes) via [`hexxlastore.SeamStoreAdapter`](../../internal/adapter/secondary/hexxlastore/seam_store.go)
- [x] MCP tools [`seam_tools.go`](../../internal/adapter/primary/mcpsrv/seam_tools.go) + [`SeamLifecycleService`](../../internal/core/services/seam_lifecycle.go) tests + changelog

Raw `PutSeam` with a caller-supplied ULID body is **not** exposed — Hexxla helpers assign ids for conflict/supersession; callers resolve by id from **`mosaic_hexxla_find_seams`**.

---

## Phase 5 — Facets & edges (optional)

- [x] **`PutFacet`** — [`mosaic_hexxla_put_facet`](../../internal/adapter/primary/mcpsrv/facet_edge_tools.go); Hexxla **`NewFacetDerived`** builds the wire record ([`FacetEdgeStoreAdapter`](../../internal/adapter/secondary/hexxlastore/facet_edge_store.go))
- [x] **`LinkCells`** (`PutEdge`) — [`mosaic_hexxla_link_cells`](../../internal/adapter/primary/mcpsrv/facet_edge_tools.go); **`NewProvenanceWire`** for timestamps on edge provenance
- [x] **Facet/edge reads (View‑only MCP)** — [`mosaic_hexxla_get_facet`](../../internal/adapter/primary/mcpsrv/facet_edge_read_tools.go), **`mosaic_hexxla_list_facets`**, **`mosaic_hexxla_get_edge`**, **`mosaic_hexxla_list_edges_from`**: `primary.FacetEdgeBrowse` → **`FacetEdgeReadService`** → **`FacetEdgeReader`** on **same adapter** `[facet_edge_reads.go](../../internal/adapter/secondary/hexxlastore/facet_edge_reads.go)` (`GetFacet`, `AscendFacetsForCell`, `GetEdge`, `AscendEdgesFrom`)
- Raw **`PutEdge`** without **`LinkCells`** is not duplicated — the helper covers the MCP write surface

---

## Phase 6 — Operator / advanced

**Included in Mosaic today (safe, read-only MCP):**

- [x] **Tag vocabulary** — `mosaic_hexxla_list_tags` (`Tx.ListExistingTopics`), `mosaic_hexxla_tag_counts` (`Tx.TagCounts`) — **`DB.View`** only; no mutations. Helps LLMs reuse consistent tags on `mosaic_hexxla_put_cell` and refine searches.
- [x] **Facet/edge lookups** — `mosaic_hexxla_get_facet`, `mosaic_hexxla_list_facets`, `mosaic_hexxla_get_edge`, `mosaic_hexxla_list_edges_from` — **`DB.View`** only; verify writes and traverse graphs before mutations.

**Excluded from default LLM MCP (documented stance — see [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md)):**

- [ ] **`Compact` / compaction** — layout/maintenance operation; starvation and backup-timing concerns → **operator CLI / control plane**, not casual agent triggers.
- [ ] **`PruneCellVersions` / MVCC retention** — **data loss surface** if mis-invoked → operator-only.
- [ ] **Changelog / `ReadChangelogSince`** — useful for audits when **bounded + gated**; not shipped via MCP until those guarantees exist.

---

## Session log (optional)

Append a line per focused session:

| Date | Focus | Outcome |
| --- | --- | --- |
| 2026-04-29 | Planning | Added this tracker; **Active phase** = Phase 1 (reads) |
| 2026-04-29 | Phase 1 reads | `mosaic_hexxla_query_cells`, `mosaic_hexxla_search_cells`; ports/adapters/services wired in `cmd/mosaic-mcp`; **Active phase** → Phase 2 |
| 2026-04-29 | Phase 2 context + UX | `mosaic_hexxla_load_context_pack`; `retrieval_hint` on embed/query/search JSON; tool descriptions steer chaining |
| 2026-04-29 | Phase 3 writes | `mosaic_hexxla_put_cell`, `mosaic_hexxla_put_embedding`, `mosaic_hexxla_delete_cell`; `CellMutationService` + `CellWriterAdapter`; **Active phase** → Phase 4 |
| 2026-04-29 | Phase 4 seams | `mosaic_hexxla_find_seams`, `mosaic_hexxla_mark_conflict`, `mosaic_hexxla_mark_supersedes`, `mosaic_hexxla_resolve_seam`; **Active phase** → Phase 5 (optional facets/edges) |
| 2026-04-29 | Phase 5 facets / edges | `mosaic_hexxla_put_facet`, `mosaic_hexxla_link_cells`; **`go.mod`** pinned **`hexxladb v0.3.0`** (no **`replace`**); **Active phase** → Phase 6 |
| 2026-04-29 | Tag discovery + audit | **`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`**; [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md); Phase 6 narrowed to operator-only ops (Compaction / prune / changelog) |
| 2026-04-29 | Facet / edge reads | **`mosaic_hexxla_get_facet`**, **`mosaic_hexxla_list_facets`**, **`mosaic_hexxla_get_edge`**, **`mosaic_hexxla_list_edges_from`** (+ Hexxla **`FacetWalkRecord`** / **`EdgeWalkRecord`**) |
| 2026-04-29 | Hybrid retrieval | **`embed_query_text`** on **`mosaic_hexxla_query_cells`** / **`mosaic_hexxla_search_cells`** → **`CellQuery.Embedding`** / **`CellSearchConfig.Embedding`**; see [MCP_AGENT_BLUEPRINT.md](./MCP_AGENT_BLUEPRINT.md) |

---

## Quick links — code map

| Area | Location |
| --- | --- |
| MCP tool registration | `internal/adapter/primary/mcpsrv/` (`health_tool.go`, `cell_query_tool.go`, `cell_search_tool.go`, `embedding_search_tool.go`, `context_pack_tool.go`, `cell_mutation_tool.go`, `seam_tools.go`, `facet_edge_tools.go`, `facet_edge_read_tools.go`, `tag_tools.go`) |
| Composition root | `cmd/mosaic-mcp/main.go` |
| Hexxla adapter | `internal/adapter/secondary/hexxlastore/` |
| Primary ports | `internal/core/ports/primary/` |
| Secondary ports | `internal/core/ports/secondary/` |
| Services | `internal/core/services/` |
| Ollama client | `internal/ollama/` |
| Seed CLI | `cmd/mosaic-seed/` |
