# HexxlaDB API surface → Mosaic hexagonal mapping

**Purpose:** Inform `internal/core/domain`, `ports`, and secondary adapter design without leaking engine types inward.

Mosaic pins HexxlaDB `v0.7.0`. Its upstream references are [doc.go](https://github.com/hexxla/hexxladb/blob/v0.7.0/doc.go) overview · [docs/hexxladb/API_REFERENCE.md](https://github.com/hexxla/hexxladb/blob/v0.7.0/docs/hexxladb/API_REFERENCE.md) task-oriented guide · [TX.md](https://github.com/hexxla/hexxladb/blob/v0.7.0/docs/hexxladb/TX.md) transactions / MVCC.

Architecture rule from hexxladb: outbound code calls **`package hexxladb` only**, not `internal/engine` ([HEXAGONAL_ARCHITECTURE.md](https://github.com/hexxla/hexxladb/blob/v0.7.0/docs/architecture/HEXAGONAL_ARCHITECTURE.md)).

**Concrete MCP ↔ Hexxla capability matrix:** [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md).

---

## 1. API shape (groups)

| Group | Main symbols | Mosaic relevance |
| --- | --- | --- |
| **Lifecycle** | `OpenExisting`, `Options`, `(*DB).Close`, `Compact`, encryption options | Composition root opens DB once; encryption keys from env/config. |
| **Transactions** | `View`, `Update`, `Batch`, `ViewAt`, `ViewAtTime`; `Tx` | Secondary adapter wraps Tx boundaries; Mosaic use cases choose read vs write snapshot. |
| **Cells & coords** | `PutCell`, `GetCell`, `DeleteCell`; `Coord`, `PackedCoord`, `Pack`, `Unpack`; ring walks | Domain models **logical** coordinates/value; adapter maps ↔ `PutCell` / `GetCell` payloads. |
| **Context assembly** | `LoadContext` with radial seeds, result bounds, seams, and supersession | Secondary adapter retrieves provider-neutral candidates; Mosaic services own ranking and byte-budget policy. |
| **Seams** | `PutSeam`, `FindSeams`, `FindSeamsAt`, `ResolveSeam`, `MarkConflict`, supersession helpers | Mosaic MCP exposes `FindSeams`, `MarkConflict`, `MarkSupersedes`, `ResolveSeam` ([seam_tools.go](../../internal/adapter/primary/mcpsrv/seam_tools.go)); validation in `SeamLifecycleService`. |
| **Facets & edges** | `PutFacet`, `PutEdge`, `LinkCells`, reads: `GetFacet`, `AscendFacetsForCell`, `GetEdge`, `AscendEdgesFrom` | Writes: `mosaic_hexxla_put_facet`, `mosaic_hexxla_link_cells` ([facet_edge_tools.go](../../internal/adapter/primary/mcpsrv/facet_edge_tools.go)); reads: `mosaic_hexxla_get_facet`, `mosaic_hexxla_list_facets`, `mosaic_hexxla_get_edge`, `mosaic_hexxla_list_edges_from` ([facet_edge_read_tools.go](../../internal/adapter/primary/mcpsrv/facet_edge_read_tools.go)); **`NewFacetDerived`** / **`NewProvenanceWire`** ([templates.go](https://github.com/hexxla/hexxladb/blob/v0.7.0/templates.go)). |
| **Query / search** | `QueryCells`, `SearchCells`, tag/source/time scans; embeddings + `SearchByEmbedding` when enabled | MCP **`mosaic_hexxla_query_cells`**, **`mosaic_hexxla_search_cells`**, **`mosaic_hexxla_search_embedding`** ([coverage](./HEXXLA_API_SURFACE_COVERAGE.md)). |
| **Tags (analytics)** | `ListExistingTopics`, `TagCounts` (`View`) | MCP **`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`** ([`tag_tools.go`](../../internal/adapter/primary/mcpsrv/tag_tools.go)). |
| **Ops / observability** | `HealthCheck`, `StatsMVCC`, changelog readers, prune | `mosaic_hexxla_health` maps the integrity report to Mosaic’s health schema and includes persisted layout settings and file sizes. |
| **Raw KV** | `Tx.Get` / `Put` / `AscendRange` | Mosaic exposes typed records. Application-owned raw keys stay outside the MCP surface. |

---

## 2. Tight hexagonal adherence (mosaic rules)

| Layer | May depend on HexxlaDB / `record.*` types? |
| --- | --- |
| **`internal/core/domain`** | **No.** Pure types (e.g. axial `(q,r)` validation, semantic errors, invariants). |
| **`internal/core/ports/*`** | **No.** Ports use Mosaic domain types. |
| **`internal/core/services`** | **No** imports of `adapter/` or concrete `hexxladb`; only **ports**. |
| **`internal/adapter/secondary`** | **Yes.** The storage adapter translates domain values and owns transactions. Startup opens the database in `cmd/mosaic-mcp`. |

**Practical note:** HexxlaDB exports root aliases such as **`hexxladb.CellRecord`** and lattice types. Mosaic still keeps those storage-facing types **inside the secondary adapter** so ports remain product-shaped and MCP JSON schemas are not coupled to the database package.

---

## Local seeded database

[`mosaic-seed`](../../cmd/mosaic-seed/main.go) embeds each conversation turn
through Ollama, then writes its cell and vector at the same packed coordinate
in one `DB.Update`. The [local corpus and spiral layout](../../cmd/mosaic-seed/seed_data.go)
supply the turn order and coordinates.

Requirements: **Ollama** running (e.g. `ollama pull all-minilm`). The database is created with **`EmbeddingDimension: 384`**, **`DistanceCosine`**, MVCC, and the validated 4096-byte HexxlaDB page profile.

**Typical flow** (run from **repository root**):

1. Configure Ollama with `-ollama` and `-embed-model`, policy YAML, or `MOSAIC_OLLAMA_URL` and `MOSAIC_EMBED_MODEL`. Defaults are `http://127.0.0.1:11434` and `all-minilm`.
2. **`MOSAIC_DB_PATH`** / **`-db`** select the Hexxla file (**`mosaic.hexxla`** in the shell cwd if unset for create/seed).
3. The first successful run creates the validated database. Re-runs skip an existing primary.

```bash
cd /path/to/mosaic
ollama pull all-minilm   # once
go run ./cmd/mosaic-seed
export MOSAIC_DB_PATH=./mosaic.hexxla
go run ./cmd/mosaic-mcp
```

To refresh an existing fixture, [seed a separate candidate](./DATABASE_CREATION.md#replace-a-database-offline).
