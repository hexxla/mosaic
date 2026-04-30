# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Common Changelog](https://common-changelog.org/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Policy YAML **`ollama:`** with **`base_url`** and **`embed_model`** — **`mosaic-mcp`** and **`mosaic-seed`** resolve Ollama settings with precedence: **`mosaic-seed`** CLI **`-ollama`** / **`-embed-model`** → YAML → **`MOSAIC_OLLAMA_URL`** / **`MOSAIC_EMBED_MODEL`** → defaults. **`mosaic-mcp`**: YAML → env → defaults. **[`config.ResolveOllama`](internal/config/embed.go)** centralizes resolution; **[`MOSAIC_POLICY_FILE`](internal/config/mosaic_yaml.go)** is forwarded from the [Makefile](Makefile) for **`seed`** / **`run-mosaic-mcp`** / **`create-db`** when set.
- **`mosaic_hexxla_health`** — JSON includes **`disk`** (**`primary_path`**, **`primary_bytes`**, **`wal_bytes`**, **`total_bytes`**) and **`mvcc_retain_commits_behind_head`** (effective policy). See **`docs/mosaic/HEXXLA_TROUBLESHOOTING.md`**.
- **`database.auto_maintain_after_cell_delete.debounce_after_delete_ms`** — coalesce rapid deletes into one prune/compact burst (**`0`** = maintain immediately after each delete; **omit** when **`enabled: true`** defaults to **2000** ms). **`mosaic-mcp`** flushes pending debounced maintenance on shutdown.

- **Post-delete MVCC prune + compact (optional)** — YAML **`database.auto_maintain_after_cell_delete`** (**`enabled`**, **`prune`**, **`compact`**, optional **`prune_profile`**, **`max_prune_rounds_per_delete`**, **`debounce_after_delete_ms`**). After a **`cell_removed: true`** delete (or after the debounce window), Mosaic runs bounded **`PruneScheduler.Tick`** and/or **`(*hexxladb.DB).Compact`**, swapping the DB handle behind **`LiveDB`** in **`internal/adapter/secondary/hexxlastore`**. Default injected **`mvcc_retain_commits_behind_head`** when pruning is on and YAML omits it is **`8`** — shallow **`ViewAt`** history, tighter disk vs cell count once prune runs — see **`configs/config.yaml`** and **`docs/hexxladb/OPERATIONS.md`** (while **`CommitSeq` ≤ retain**, suggested prune is ineffective).

### Changed

- **Default database file** when **`mosaic-create-db`** / **`mosaic-seed`** omit **`-db`**, **`-name`**, and **`MOSAIC_DB_PATH`**: **`mosaic.hexxla`** in the shell’s current working directory (was **`.tmp/mosaic-seed.hexxla`**). **Makefile** **`MOSAIC_DB_PATH`** default is **`./mosaic.hexxla`**; **`.gitignore`** includes **`/mosaic.hexxla`** at repo root.
- **Documentation** — README **Get started** (Ollama expected; seed demos called out as dev-only; path choice); **[`DATABASE_CREATION.md`](docs/mosaic/DATABASE_CREATION.md)** adds encrypted DB + MCP setup; removed **`POLICY_AND_RETENTION.md`** (obsolete operator gates), **`HYBRID_RETRIEVAL_PLAN.md`** / **`HEXXLA_API_ROADMAP.md`** (superseded by **IMPLEMENTATION_PLAN**, **MCP_AGENT_BLUEPRINT**, **HEXXLA_API_SURFACE_COVERAGE**).
- **Breaking:** Go module path is **`github.com/sploitzberg/mosaic`** (was `github.com/sploitzberg/go-llm-project-structure`). Update imports and **`go.mod`** `replace` directives. Bootstrap **`cmd/go-llm-project-structure`** removed — use **`mosaic-mcp`**, **`mosaic-seed`**, **`mosaic-create-db`**.
- **Documentation** — **[`docs/mosaic/MOSAIC_CONFIG.md`](docs/mosaic/MOSAIC_CONFIG.md)** documents every policy YAML key; **[`configs/config.yaml`](configs/config.yaml)** is values-only (comments removed); **[`README.md`](README.md)** restructured (**Get started** first, feature matrix, MCP tool map).

- **`mosaic_hexxla_delete_cell`** — tool result is **`DeleteCellMutationResult`**: **`ok`** plus **`cell_removed`** (**`true`** only when a live cell existed and was removed; **`false`** with **`ok`** means empty coord / typo / duplicate delete, not an error). Implemented with **`Tx.GetCell` + `Tx.DeleteCell`** in one **`DB.Update`** (same transactional visibility as a single-delete round-trip).
- **[README.md](README.md)** — Retrieval metering row, **`configs/config.yaml`** callout in Quick start, expanded documentation index (**`TODOS.md`**, **`docs/ROADMAP.md`**, **`CHANGELOG.md`**), and **Roadmap and rough edges** pointer.
- **[`docs/ROADMAP.md`](docs/ROADMAP.md)** — Near-term echoes **[`TODOS.md`](TODOS.md)** pending items (oversized one-shot, meter lifetime, approximate vs tokenizer).
- **[`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](docs/mosaic/MCP_AGENT_BLUEPRINT.md)** and **[`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc)** — document that the optional **cumulative session retrieval cap** is an **egress / runaway** control, not a **relevance** mechanism (per-call rings and byte budgets remain primary).
- **MCP retrieval metering** — approximate retrieval totals are **always recorded** per MCP session (when read tools run); **`session_approx_token_budget: 0`** disables **enforcement** only. **`mosaic_hexxla_retrieval_budget_status`** exposes **`metering_enabled`**, **`budgeting_enabled`**, and **`approx_tokens_used`**.
- **Mosaic config YAML** — `retention.enforcement` is now a **boolean** (`true` = server returns an error for conflicting turn-related `put_cell` kinds; `false` = advisory only). Legacy strings **`off`** / **`reject`** remain accepted. See [PERSISTENCE_POLICY.md](docs/mosaic/PERSISTENCE_POLICY.md).
- **MCP persistence policy** — `mosaic_hexxla_get_persistence_policy` JSON now exposes **`retention.enforcement` as a boolean**; server instructions and retention errors use the same wording (no legacy “reject” in user-facing text).
- **`mosaic-mcp` startup log** — logs **`enforcement_enabled`** (boolean) instead of the internal **`off`/`reject`** string; [`RetentionPolicy.EnforcementEnabled`](internal/config/retention.go).
- **`mosaic-create-db` / `mosaic-seed`** — **`-replace`** is an alias for **`-force`** (overwrite existing DB path); help text clarifies both mean replace-on-disk, not “encryption force”.
- **Database path resolution** — **`config.ResolveMosaicDBPath`**, **`MOSAIC_DB_DIR`**, CLI **`-name`** / **`-db-dir`** / **`-db`**; **`mosaic-mcp`** can use **`-name`** or **`-db`** without **`MOSAIC_DB_PATH`**.

### Added

- **[`docs/mosaic/HEXXLA_TROUBLESHOOTING.md`](docs/mosaic/HEXXLA_TROUBLESHOOTING.md)** — MCP iterative deletes (`allow_delete_cell`, policy reload, retrieval cap), and **`mosaic_hexxla_health`** / MVCC tag–source index semantics (vs real orphan seams).
- **[`TODOS.md`](TODOS.md)** and **[`docs/ROADMAP.md`](docs/ROADMAP.md)** — session scratchpad and roadmap (pattern aligned with HexxlaDB); [`AGENTS.md`](AGENTS.md) links both.
- **MCP retrieval session budget** — optional YAML **`retrieval.session_approx_token_budget`** (0 = unlimited) and **`retrieval.bytes_per_approx_token`**; per‑MCP‑session cumulative approximate tokens on JSON outputs from HexxlaDB read tools; tool **`mosaic_hexxla_retrieval_budget_status`**; see [configs/config.yaml](configs/config.yaml).
- Root **[README.md](README.md)** (Mosaic + HexxlaDB MCP overview; logo **`assets/images/mosaic_logo_shadow.svg`**).
- **Encrypted HexxlaDB**: [`BuildHexxlaOpenOptions`](internal/config/hexxla_open.go) / **`MOSAIC_DB_PASSPHRASE`**, **`MOSAIC_DB_ENCRYPTION_KEY_HEX`**, **`mosaic-mcp -db-passphrase`**, optional YAML **`database.passphrase`**; **`mosaic-seed -db-passphrase`**. See [PERSISTENCE_POLICY.md](docs/mosaic/PERSISTENCE_POLICY.md).
- **`config.MosaicRuntimeConfig`**: `AllowsPutCell` / `PutCellDenied` / `DeleteCellDenied`; wired through mutation MCP tools and [`CellMutationService`](internal/core/services/cell_mutation.go)
- **Mosaic config YAML** (`version: 1`; `mosaic-mcp -policy` / **`MOSAIC_POLICY_FILE`**): **`retention`** block (capture_mode, enforcement, notes); deprecated **`persistence_policy`** alias; **`allow_delete_cell`** (default **false**). [`LoadMosaicConfigFromFile`](internal/config/mosaic_yaml.go), [`RetentionPolicy`](internal/config/retention.go). See [PERSISTENCE_POLICY.md](docs/mosaic/PERSISTENCE_POLICY.md), [configs/config.yaml](configs/config.yaml).
- **`embed_query_text`** on **`mosaic_hexxla_query_cells`** and **`mosaic_hexxla_search_cells`**: Ollama embeds the string, then **`CellRetrievalService`** passes **`[]float32`** into **[`hexxlastore.CellReaderAdapter`](internal/adapter/secondary/hexxlastore/cell_reader.go)** for Hexxla **`CellQuery.Embedding`** / **`CellSearchConfig.Embedding`** (hybrid ANN + filters). Requires non-zero DB **`EmbeddingDimension`** and **`MOSAIC_OLLAMA`**; see [MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md).
- MCP JSON for returned **cells** includes **`created_at`**, **`updated_at`** (provenance) and optional **`valid_from`**, **`valid_to`** (validity window), RFC3339Nano UTC, on query/search cells, embedding hits, and context-pack cell rows — see [`wire_time.go`](internal/adapter/secondary/hexxlastore/wire_time.go); no HexxlaDB engine changes required (**wire already stores** [`ProvenanceWire`](https://github.com/hexxla/hexxladb/blob/main/internal/record/types.go) timestamps on **`PutCell`**).
- MCP **read facet/edge snapshots** (**`mosaic_hexxla_get_facet`**, **`mosaic_hexxla_list_facets`**, **`mosaic_hexxla_get_edge`**, **`mosaic_hexxla_list_edges_from`**) wrapping **`Tx.GetFacet`** / **`Tx.AscendFacetsForCell`** / **`Tx.GetEdge`** / **`Tx.AscendEdgesFrom`** (`View`‑only): `primary.FacetEdgeBrowse` → **`FacetEdgeReadService`** → **`FacetEdgeReader`** on **[`FacetEdgeStoreAdapter`](internal/adapter/secondary/hexxlastore/facet_edge_reads.go)**; MCP [`facet_edge_read_tools.go`](internal/adapter/primary/mcpsrv/facet_edge_read_tools.go). Edge **list** responses cap **`max_edges`** default **50** / max **200** with **`truncated`** when **`AscendEdgesFrom`** stopped early — requires **HexxlaDB exports** **`FacetWalkRecord`** / **`EdgeWalkRecord`** ([`walk_export_aliases.go`](https://github.com/hexxla/hexxladb/blob/main/walk_export_aliases.go)).
- MCP **`mosaic_hexxla_list_tags`** and **`mosaic_hexxla_tag_counts`**: HexxlaDB **`Tx.ListExistingTopics`** / **`Tx.TagCounts`** (read-only `View`) via `primary.TagBrowse` → **`TagCatalogService`** → **`secondary.TagCatalog`** → [`TagCatalogAdapter`](internal/adapter/secondary/hexxlastore/tag_catalog.go); [`tag_tools.go`](internal/adapter/primary/mcpsrv/tag_tools.go). [HEXXLA_API_SURFACE_COVERAGE.md](docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md) documents full Hexxla ↔ MCP mapping and **Phase 6** operator-only stance (Compaction / MVCC prune / changelog not MCP-default).
- MCP **`mosaic_hexxla_put_facet`** and **`mosaic_hexxla_link_cells`**: **`Tx.PutFacet`** and **`Tx.LinkCells`** via `primary.FacetEdge` → **`FacetEdgeService`** → **`FacetEdgeStoreAdapter`** ([`facet_edge_store.go`](internal/adapter/secondary/hexxlastore/facet_edge_store.go)); domain [`facet_edge.go`](internal/core/domain/facet_edge.go). Uses **`NewFacetDerived`** / **`NewProvenanceWire`** from **`github.com/hexxla/hexxladb`** ([`templates.go`](https://github.com/hexxla/hexxladb/blob/main/templates.go)); **`go.mod`** pins **`github.com/hexxla/hexxladb v0.3.0`** (no **`replace`** — optional local symlink / clone for upstream development only).
- MCP tools **`mosaic_hexxla_find_seams`**, **`mosaic_hexxla_mark_conflict`**, **`mosaic_hexxla_mark_supersedes`**, **`mosaic_hexxla_resolve_seam`**: HexxlaDB **`FindSeams`**, **`MarkConflict`**, **`MarkSupersedes`**, **`ResolveSeam`** via `primary.SeamLifecycle` → **`SeamLifecycleService`** → **`secondary.SeamStore`** → [`hexxlastore.SeamStoreAdapter`](internal/adapter/secondary/hexxlastore/seam_store.go); domain [`internal/core/domain/seam.go`](internal/core/domain/seam.go); MCP [`seam_tools.go`](internal/adapter/primary/mcpsrv/seam_tools.go); wired in [`cmd/mosaic-mcp/main.go`](cmd/mosaic-mcp/main.go)
- MCP tools **`mosaic_hexxla_put_cell`**, **`mosaic_hexxla_put_embedding`**, **`mosaic_hexxla_delete_cell`**: HexxlaDB **`PutCell`** / **`PutEmbedding`** / **`DeleteCell`** via `primary.CellMutation` → `services.CellMutationService` → **`secondary.CellWriter`** → [`hexxlastore.CellWriterAdapter`](internal/adapter/secondary/hexxlastore/cell_writer.go); **`secondary.TextEmbedder`** implemented by [`ollamaembed.NewTextEmbedder`](internal/adapter/secondary/ollamaembed/embedder.go) for the text embedding path; domain types in [`internal/core/domain/cell_write.go`](internal/core/domain/cell_write.go); wired in [`cmd/mosaic-mcp/main.go`](cmd/mosaic-mcp/main.go)
- [`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc) and [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](docs/mosaic/MCP_AGENT_BLUEPRINT.md) — agent workflow (retrieval → `retrieval_hint` → context pack); [`AGENTS.md`](AGENTS.md) links to both
- MCP tool **`mosaic_hexxla_load_context_pack`** (budget: **`omit_budget`**, **`budget_tokens_approx`** / **`bytes_per_approx_token`**, **`max_budget_bytes`** or legacy **`max_tokens`** as explicit UTF‑8 bytes; default **4096**; domain [`ApproximateByteBudgetFromTokens`](internal/core/domain/context_budget.go)): HexxlaDB **`LoadContextPackFrom`** (multi-seed, `ByteLenBudgeter`) via `primary.ContextAssembly` → [`ContextAssemblyService`](internal/core/services/context_assembly.go) → [`hexxlastore.ContextPackAdapter`](internal/adapter/secondary/hexxlastore/context_pack.go); domain [`context_pack.go`](internal/core/domain/context_pack.go)
- MCP tool **`mosaic_hexxla_estimate_context_budget_bytes`**: previews UTF‑8 byte budget from approximate tokens (same math/clamps as token-based **`load_context_pack`**); [`context_budget_estimate_tool.go`](internal/adapter/primary/mcpsrv/context_budget_estimate_tool.go); [MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md) (**persistence**: no chat auto-save — only explicit **`put_cell`** path)
- Domain [**`retrieval_hints.go`**](internal/core/domain/retrieval_hints.go): shared hint strings; **`retrieval_hint`** on **`EmbeddingSearchResponse`** and **`CellHitsResponse`** so LLM clients know when to call **`mosaic_hexxla_load_context_pack`** after ANN/lexical retrieval
- MCP tools **`mosaic_hexxla_query_cells`** and **`mosaic_hexxla_search_cells`**: HexxlaDB **`QueryCells`** / **`SearchCells`** via `primary.CellRetrieval` → `services.CellRetrievalService` → `secondary.CellReader` → [`hexxlastore.CellReaderAdapter`](internal/adapter/secondary/hexxlastore/cell_reader.go); domain types in [`internal/core/domain/cell_reads.go`](internal/core/domain/cell_reads.go)
- [IMPLEMENTATION_PLAN.md](docs/mosaic/IMPLEMENTATION_PLAN.md) — phased checklist and progress tracker for HexxlaDB MCP exposure
- MCP tool **`mosaic_hexxla_search_embedding`**: Ollama (`MOSAIC_OLLAMA_URL`, **`MOSAIC_EMBED_MODEL`**) query embedding + HexxlaDB **`SearchByEmbedding`** + **`GetCell`** payloads; **`internal/ollama`** HTTP client; **`secondary.EmbeddingANN`** / **`EmbeddingANNAdapter`**; **`primary.EmbeddingSearch`** / **`EmbeddingSearchService`**; config **`LoadOllamaFromEnv`**
- [HEXXLA_API_SURFACE_COVERAGE.md](docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md) mapping HexxlaDB API exposure through MCP
- **`cmd/mosaic-create-db`**: creates an **empty** Mosaic-compatible HexxlaDB file (same open options as seed, no Ollama); optional flags **`mvcc`**, **`page-size`**, **`max-value-bytes`**, **`embedding-dim`**, **`distance-metric`** default to Mosaic layout; **`make run-mosaic-create-db`** / **`make create-db`** / **`make build-mosaic-create-db`**; Makefile **`MOSAIC_CREATE_DB_FLAGS`**, **`MOSAIC_SEED_FLAGS`**, **`MOSAIC_MCP_FLAGS`** append extra CLI args — see [DATABASE_CREATION.md](docs/mosaic/DATABASE_CREATION.md)
- **`cmd/mosaic-seed`**: same optional layout flags as **`mosaic-create-db`** (defaults unchanged)
- **`cmd/mosaic-seed`**: seeds a HexxlaDB file with a conversational corpus **`PutCell`** + **Ollama embeddings** (**`PutEmbedding`**, **384‑dim** **`all-minilm`**, **`DistanceCosine`**) matching hexxladb **`examples/llm_context_engine`**; spiral coords + cell templates aligned with **`examples/conversational_memory`**; env **`MOSAIC_OLLAMA_URL`**, **`MOSAIC_EMBED_MODEL`**; **`make seed`** / **`make reseed`** / **`make mosaic-dev`**; seed uses **`internal/ollama`**
- MCP tool **`mosaic_hexxla_health`**: HexxlaDB **`HealthCheck`** via `primary.Health` → `services.HealthService` → `secondary.EngineHealth` → [`internal/adapter/secondary/hexxlastore`](internal/adapter/secondary/hexxlastore); **`MOSAIC_DB_PATH`** for `cmd/mosaic-mcp`; dependency **`github.com/hexxla/hexxladb`**
- [HEXXLADB_API_NOTES.md](docs/mosaic/HEXXLADB_API_NOTES.md) linking HexxlaDB public API groups to Mosaic hexagonal layers
- `cmd/mosaic-mcp`: local MCP Streamable HTTP server using the official Go MCP SDK; loopback bind (`MOSAIC_MCP_ADDR` / `MOSAIC_MCP_PATH`) and HexxlaDB path (`MOSAIC_DB_PATH`)
- [MCP_BLUEPRINT.md](docs/mosaic/MCP_BLUEPRINT.md) describing the local Streamable HTTP MCP server plan for HexxlaDB-backed tools
- Initial project structure with hexagonal architecture (domain, port, service, adapter)
- Makefile with common development targets
- CI pipeline with formatting, linting, tests, and architecture guardrails
- Comprehensive documentation for each architectural layer

### Changed

- **Docs:** Hybrid **`embed_query_text`** summary lives in [MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md); **`LoadContext`** vs pack rationale there; removed [`load_context_sketch.go`](internal/adapter/secondary/hexxlastore/load_context_sketch.go) (obsolete commented sketch).

- **`go.mod` / CI**: **`require github.com/hexxla/hexxladb v0.3.0`** with **no** **`replace`**; workflows resolve the module from the **Go module proxy** (no second checkout of **`hexxla/hexxladb`** in **CI**, **integration**, or **release**).
- **`mosaic_hexxla_search_embedding`**, **`mosaic_hexxla_query_cells`**, **`mosaic_hexxla_search_cells`**: tool descriptions clarify top-K vs lattice-expanded context; embedding search response includes **`retrieval_hint`** from [`EmbeddingSearchService`](internal/core/services/embedding_search.go)
- **`mosaic_hexxla_health`**: response includes **`database_layout`** (`page_size_bytes`, `max_value_bytes`, `embedding_dimension`, `embedding_metric`) from HexxlaDB **`DB.PageSize` / `MaxValueBytes` / `EmbeddingDimension` / `EmbeddingMetric`**, plus **`integrity_ok`** (no tag/source index errors, no orphan seams); still returns full **`HealthCheck`** report fields
- CI: `14-exported-symbols.sh` uses safe warning counters under `set -e` (`warnings=$((warnings + 1))`)
- CI scripts: import-order validation now compares MCP SDK vs module-internal imports correctly; outdated-deps flags only outdated direct modules
- Updated folder structure to clearly separate primary and secondary ports/adapters

## [0.1.0] - 2025-04-25

### Added

- ~~Bootstrapper CLI skeleton (`cmd/go-llm-project-structure`)~~ — removed; module is **`github.com/sploitzberg/mosaic`**, primary binary **`mosaic-mcp`**
- Hexagonal architecture guardrail script
- AGENTS.md with instructions for LLMs and contributors
- Layer-specific README.md files explaining responsibilities

### Changed

- Standardized project layout for LLM-friendly hexagonal architecture templates

[Unreleased]: https://github.com/sploitzberg/mosaic/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/sploitzberg/mosaic/releases/tag/v0.1.0
