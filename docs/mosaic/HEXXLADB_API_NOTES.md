# HexxlaDB API surface → Mosaic hexagonal mapping

**Purpose:** Inform `internal/core/domain`, `ports`, and secondary adapter design without leaking engine types inward.

**Canonical references (hexxladb repo):** [doc.go](https://github.com/hexxla/hexxladb/blob/main/doc.go) overview · [docs/hexxladb/API_REFERENCE.md](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/API_REFERENCE.md) full inventory · [TX.md](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/TX.md) transactions / MVCC.

Architecture rule from hexxladb: outbound code calls **`package hexxladb` only**, not `internal/engine` ([HEXAGONAL_ARCHITECTURE.md](https://github.com/hexxla/hexxladb/blob/main/docs/architecture/HEXAGONAL_ARCHITECTURE.md)).

**Concrete MCP ↔ Hexxla capability matrix:** [HEXXLA_API_SURFACE_COVERAGE.md](./HEXXLA_API_SURFACE_COVERAGE.md).

---

## 1. API shape (groups)

| Group | Main symbols | Mosaic relevance |
| --- | --- | --- |
| **Lifecycle** | `Open`, `Options`, `(*DB).Close`, `Compact`, encryption options | Composition root opens DB once; encryption keys from env/config. |
| **Transactions** | `View`, `Update`, `Batch`, `ViewAt`, `ViewAtTime`; `Tx` | Secondary adapter wraps Tx boundaries; MOSAIC use cases choose read vs write snapshot. |
| **Cells & coords** | `PutCell`, `GetCell`, `DeleteCell`; `Coord`, `PackedCoord`, `Pack`, `Unpack`; ring walks | Domain models **logical** coordinates/value; adapter maps ↔ `PutCell` / `GetCell` payloads. |
| **Context assembly** | `LoadContext` with seeds, result bounds, validity, graph/ring dispatch, seams, and supersession | Secondary adapter retrieves provider-neutral candidates; Mosaic services own ranking and byte-budget policy. |
| **Seams** | `PutSeam`, `FindSeams`, `FindSeamsAt`, `ResolveSeam`, `MarkConflict`, supersession helpers | Mosaic MCP exposes `FindSeams`, `MarkConflict`, `MarkSupersedes`, `ResolveSeam` ([seam_tools.go](../../internal/adapter/primary/mcpsrv/seam_tools.go)); validation in `SeamLifecycleService`. |
| **Facets & edges** | `PutFacet`, `PutEdge`, `LinkCells`, reads: `GetFacet`, `AscendFacetsForCell`, `GetEdge`, `AscendEdgesFrom` | Writes: `mosaic_hexxla_put_facet`, `mosaic_hexxla_link_cells` ([facet_edge_tools.go](../../internal/adapter/primary/mcpsrv/facet_edge_tools.go)); reads: `mosaic_hexxla_get_facet`, `mosaic_hexxla_list_facets`, `mosaic_hexxla_get_edge`, `mosaic_hexxla_list_edges_from` ([facet_edge_read_tools.go](../../internal/adapter/primary/mcpsrv/facet_edge_read_tools.go)); **`NewFacetDerived`** / **`NewProvenanceWire`** ([templates.go](https://github.com/hexxla/hexxladb/blob/main/templates.go)). |
| **Query / search** | `QueryCells`, `SearchCells`, tag/source/time scans; embeddings + `SearchByEmbedding` when enabled | MCP **`mosaic_hexxla_query_cells`**, **`mosaic_hexxla_search_cells`**, **`mosaic_hexxla_search_embedding`** ([coverage](./HEXXLA_API_SURFACE_COVERAGE.md)). |
| **Tags (analytics)** | `ListExistingTopics`, `TagCounts` (`View`) | MCP **`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`** ([`tag_tools.go`](../../internal/adapter/primary/mcpsrv/tag_tools.go)). |
| **Ops / observability** | `HealthCheck`, `StatsMVCC`, changelog readers, prune | Good first **ping**/health vertical slice. The MCP tool **`mosaic_hexxla_health`** returns the full `HealthCheck` report **and** header-derived **`PageSize` / `MaxValueBytes` / `EmbeddingDimension`** for operators. |
| **Raw KV** | `Tx.Get` / `Put` / `AscendRange` | Prefer lattice primitives unless you intentionally bypass them. |

---

## 2. Tight hexagonal adherence (mosaic rules)

| Layer | May depend on HexxlaDB / `record.*` types? |
| --- | --- |
| **`internal/core/domain`** | **No.** Pure types (e.g. axial `(q,r)` validation, semantic errors, invariants). |
| **`internal/core/ports/*`** | **Prefer no** `import hexxladb` in interfaces—use mosaic DTOs / domain types so primary ports stay product-shaped. |
| **`internal/core/services`** | **No** imports of `adapter/` or concrete `hexxladb`; only **ports**. |
| **`internal/adapter/secondary`** | **Yes** — translate domain/DTO ↔ `Open`/`Tx`/`PutCell`/… |

**Practical note:** HexxlaDB v0.6.0 exports root aliases such as **`hexxladb.CellRecord`** and lattice types. Mosaic still keeps those storage-facing types **inside the secondary adapter** so ports remain product-shaped and MCP JSON schemas are not coupled to the database package.

---

## 3. Suggested secondary port evolution (thin → fat)

Start with a **minimal** driven port aligned to one vertical slice, e.g.:

- `Ping(ctx)` or `Health(ctx)` → maps to `DB` stats / health APIs only.

Then widen intentionally:

- `WithTx(ctx, readOnly, func(LatticeTxn) error)` **where `LatticeTxn` is a mosaic-defined interface** implemented only by wrapping `*hexxladb.Tx`—**or** small methods like `GetCell(ctx, coord) (...)` without exposing `Tx` to services if you prefer.

Avoid copy-pasting the entire HexxlaDB surface into ports; expose **only what MOSAIC tools need** per milestone.

---

## 4. Current implementation

The initial vertical slices are complete across domain DTOs, ports, services, the HexxlaDB secondary adapter, and MCP registration. Use [`HEXXLA_API_SURFACE_COVERAGE.md`](./HEXXLA_API_SURFACE_COVERAGE.md) for the current mapping and runtime `tools/list` for exact schemas. New capabilities should still be added inside-out through the existing boundaries rather than exposing HexxlaDB types directly from MCP handlers.

---

## 5. Local seeded database for Mosaic (`cmd/mosaic-seed`)

Seeding follows the same **ingestion pipeline** as Hexxla’s [`examples/llm_context_engine`](https://github.com/hexxla/hexxladb/tree/main/examples/llm_context_engine): for each conversation turn, **Ollama** (`POST /api/embeddings`, default model **`all-minilm`**, **384-dimensional** vectors) then **`Update` → `PutCell` + `PutEmbedding`** on the same **packed coordinate**. The **spiral coordinate layout** matches [`examples/conversational_memory`](https://github.com/hexxla/hexxladb/tree/main/examples/conversational_memory) (`NewUserMessageCell` / `NewAssistantResponseCell`, extra tags).

Requirements: **Ollama** running (e.g. `ollama pull all-minilm`). The database is created with **`EmbeddingDimension: 384`**, **`DistanceCosine`**, MVCC, and the validated 4096-byte HexxlaDB page profile.

**Why `Update` per turn:** HexxlaDB v0.6.0 accepts `[]hexxladb.CellRecord` in `BatchPutCells`, but Mosaic generates each embedding over HTTP before opening the database transaction and then atomically stores that turn's cell and vector at one coordinate. A bulk cell-only batch would change those failure and durability semantics without helping the current seed workflow.

**Typical flow** (run from **repository root**):

1. **`MOSAIC_OLLAMA_URL`** (default `http://127.0.0.1:11434`) and **`MOSAIC_EMBED_MODEL`** (default `all-minilm`) configure Ollama. Override with **`MOSAIC_OLLAMA_URL`**, **`MOSAIC_EMBED_MODEL`**, or flags **`-ollama`**, **`-embed-model`**.
2. **`MOSAIC_DB_PATH`** / **`-db`** select the Hexxla file (**`mosaic.hexxla`** in the shell cwd if unset for create/seed).
3. First successful run creates parent dirs and the DB file. Re-runs **skip** unless **`-force`**.

```bash
cd /path/to/mosaic
ollama pull all-minilm   # once
go run ./cmd/mosaic-seed
export MOSAIC_DB_PATH=./mosaic.hexxla
go run ./cmd/mosaic-mcp
```

Replace an existing seeded file:

```bash
go run ./cmd/mosaic-seed -force
```
