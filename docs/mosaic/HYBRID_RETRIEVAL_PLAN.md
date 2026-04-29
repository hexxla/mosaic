# Hybrid retrieval, backlog & token budgeting

**Status:** **`embed_query_text`** on **`mosaic_hexxla_query_cells`** and **`mosaic_hexxla_search_cells`** is **shipped**. Ollama embeds the text; **`CellRetrievalService`** passes **`[]float32`** into Hexxla **`CellQuery.Embedding`** / **`CellSearchConfig.Embedding`** (see **`internal/core/services/cell_retrieval.go`**, **`internal/adapter/secondary/hexxlastore/cell_reader.go`**).

**Upstream semantics:** [API_REFERENCE.md — Query planner / Content Search](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/API_REFERENCE.md) — embedding triggers ANN-accelerated candidate selection; other predicates apply as post-filters.

**Requirements:** **`github.com/hexxla/hexxladb` ≥ v0.3.0**, DB **`EmbeddingDimension` > 0**, **`MOSAIC_OLLAMA`** for hybrid mode.

---

## Retrieval backlog (from API audit)

| Priority | Item | Notes |
| --- | --- | --- |
| **P1** | **`TagCooccurrences`** + **`UntaggedCells`** | Read-only `View`; vocabulary / gaps |
| **P2** | **`FindSeamsAt`**, **`UpdateFacet`** | Optional MCP |
| **P3** | **`ViewAt` / `ViewAtTime`**, snapshot tags | MVCC-gated “memory as of…” |
| **—** | `RenderHexGrid*`, bulk JSON I/O | CLI / dev |
| **—** | `ReindexEmbeddings`, compact, changelog, prune | Operator (Phase 6 stance) |

Track execution in **[IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)** and **[HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md)**.

---

## Token budgeting (follow-on)

Hybrid retrieval selects **which** cells rank highly; **`mosaic_hexxla_load_context_pack`** with **`ByteLenBudgeter`** controls **how much** assembled context fills the prompt. Topics for a future **`TOKEN_BUDGETING.md`** or **`MCP_BLUEPRINT.md`** subsection: **`ByteLenBudgeter`** vs **`TokenBudgeter`**, who owns **`max_tokens`**, relation to **`TruncateCellViewsToTokenBudget`**, multi-seed budget splits.

---

## References

- **[HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md)** — capability matrix
- **[IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)** — phases & session log
- Hexxla **`CellQuery.Embedding`** / **`CellSearchConfig.Embedding`** — [query.go](https://github.com/hexxla/hexxladb/blob/main/query.go), [search.go](https://github.com/hexxla/hexxladb/blob/main/search.go)
