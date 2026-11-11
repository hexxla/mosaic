# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Common Changelog](https://common-changelog.org/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Publish the `v0.4.0-alpha.1` snapshot against HexxlaDB `v0.8.0-alpha.1` and MCP Ratchet `v0.4.0-alpha.1`, resolving both through normal Go modules with local workspaces disabled. This alpha snapshot does not establish beta/v1 qualification.

- Mark the repository and combined stack as alpha; preserve published tags, define independent v0 prerelease milestones and evidence-gated beta/v1 promotion, and link the shared practical qualification plan.

### Breaking usability/API changes

- External query/write vectors must declare the database's `embedding_model` identity. Existing custom Ratchet policies enabling advanced tools must guard `put_cells`, `put_seam`, `delete_embedding` and `rebuild_embedding_index`.

- Near-anchor `put_cell` calls require an `idempotency_key`. Retries with the same key and arguments return the original result.

- Consolidate current API documentation and remove the superseded implementation tracker and historical tool-description audit.

- Replace separate query and embedding-discovery tools with `search_cells` modes: lexical, pure-vector semantic, and filtered/composite hybrid. Search is global by default; positive radius opts into spatial restriction. Add exact `get_cell` with explicit presence.
- Publish seven core tools by default; `advanced_tools: true` exposes all 43. Advanced facet/edge/seam operations, independent embeddings and policy/budget diagnostics retain their capabilities.
- Context accepts `max_budget_bytes` and reports `total_bytes` only. Remove token-named byte aliases, `omit_budget`, and inline approximate-token alternatives; the advanced conversion tool remains available.
- Use one strict retention shape with boolean enforcement. Sample configuration selects selective capture, disables deletion, and leaves maintenance opt-in.
- Use the simplified Ratchet engine, typed events and shared MCP success boundary. `ratchet: true` enables the bundled grouped policy; custom policies use the same single schema. Observer correlation IDs remain separate from transport IDs.

### Added

- Expand advanced MCP coverage with bounded graph/FOV/Voronoi exploration, density/grid/taxonomy tools, retained snapshots/diffs, changefeed inspection, vector read/delete/rebuild, storage diagnostics, snapshot-pinned regional browsing/export and record chunks, and atomic idempotent capture batches.
- Add optional Ollama digest pinning and lexical-only startup for native databases with unknown vector provenance. Expose ANN/candidate controls, historical lexical/context reads, graph context and richer metadata.
- Forward cache size, primary fdatasync and lazy changefeed settings from database configuration.

- Pin the simplified Ratchet API and cancellation fixes to the published `v0.4.0-alpha.1` tag.
- Optional `put_cell` `embed: true` computes a vector before the transaction and commits cell/vector atomically. Model or vector-write failure saves neither; responses report `embedded`.


- **Ratchet observer** — Emit separate observer correlation IDs in place of
  transport session IDs. Use the correlation ID for stream filters.
- **Ratchet enforcement** — Delegate to the current executor, including
  one-time alternatives, and reject policies naming unknown Mosaic tools.

### Fixed

- Preserve omitted tags, creation timestamps, validity and cluster hints on cell overwrite; support explicit metadata overrides, exact tags and system-prompt templates. Guard facet derivation against concurrent source edits when a content hash is supplied.
- Prefer typed upstream changefeed/vector/snapshot introspection when available while retaining compatibility with the published dependency pin.

- Return database close failures from the MCP server command, preserving any serve or shutdown error.

- Honor the shutdown context while waiting for active post-delete maintenance.

- **Changefeed continuity** — Continue existing default HexxlaDB changefeeds on MCP startup. Use `-changefeed` to start a feed or `-changefeed-path` to select a custom sidecar. Startup rejects a missing sidecar when the primary retains changefeed state, even without registered consumers.
- **Ratchet profiles** — Keep usable prerequisite alternatives in policy order and reject enabled guarded tools only when all alternatives are unavailable.
- **Embedding compatibility** — Persist the configured embedding model and reject same-width model swaps until `mosaic-mcp -reembed` regenerates the vectors.
- **Mutation recovery** — Distinguish known-durable and uncertain HexxlaDB commit errors in MCP mutation responses.

- **Configuration safety** — Reject unknown YAML fields, extra documents, invalid Ratchet paths, nonliteral HTTP endpoint paths, and nonfinite retrieval ratios. Validate Ollama HTTP(S) bases and preserve path prefixes.
- **Credential diagnostics** — Redact URL credentials and query values from startup and HTTP errors while retaining inspectable error causes.
- **MCP lifecycle** — Reject standalone GET streams, preserve POST SSE responses, validate cross-origin mutations, drain active HTTP requests, and close idle sessions before storage shutdown.
- **Retrieval correctness** — Preserve requested lexical scan limits, validate and cap bounded hybrid scans, avoid byte-budget integer overflow, and reject ANN results canceled while waiting or searching.
- **Storage adapter safety** — Report corrupt edge coordinates, bound Ollama response bodies, and reject embedding components outside finite float32 range.

- **Configuration** — Reject files that mix retention configuration forms. Use YAML encryption credentials when seeding a database.
- **Retrieval budgets** — Release counters when MCP sessions close and pass the remaining budget into context assembly.

### Changed

- Consolidate Mosaic architecture, API mapping, database troubleshooting, and roadmap guidance. Correct the demo encryption guidance and the persistence-policy tool description.

- **Go and dependencies** — Require Go 1.27.4 and update imported Go modules.

- **Storage compatibility** — Pin HexxlaDB to `v0.8.0-alpha.1` for current WAL/header integrity, encrypted history recovery and typed capability inspection. Back up the complete recovery set before upgrading; older writers cannot maintain the updated framing. See [database upgrade instructions](docs/mosaic/DATABASE_CREATION.md#upgrade-an-existing-database).
- **Embedding race protection** — `mosaic_hexxla_put_embedding` accepts optional `expected_content_sha256`; compare the retrieved raw content atomically before storing a delayed embedding. Omission preserves existing behavior.
- **Context eviction** — Sort eviction priorities once and compact retained cells once, preserving ring/confidence/tie order and explanations. The 8,192-cell eviction benchmark improves roughly 40–55×, with two extra allocations when eviction is needed; packs that already fit retain the allocation-free path.

- **Health tool** — Report layout and disk usage by default. Pass `full_check: true` for the integrity scan and counts.

- **Development workflow** — Build future tagged versions from source. Remove automatic binary publishing and narrow routine lint checks.

## [0.3.0] - 2026-11-09

### Breaking

- **Database lifecycle** — MCP startup requires an existing primary. Creation and seeding validate a separate database before copying it to an unused destination. The in-place replacement flags and destructive reseed task are retired. [Upgrade and replacement instructions](docs/mosaic/DATABASE_CREATION.md)

### Added

- **Optional Ratchet enforcement** — `mosaic-mcp` can load an `mcp-ratchet` workflow with `-ratchet-config` or `MOSAIC_RATCHET_CONFIG_FILE`. Central MCP middleware isolates state by actual client session, enforces the supplied tag/retrieval prerequisites, and keeps tokens and state process-local without public observability endpoints.
- **MCP tool safety inventory** — all 23 tools publish explicit read-only, destructive, idempotent, and open-world hints from one fail-closed registration inventory; tests prevent production registrations from bypassing it.
- **Secure Ratchet live observability** — an opt-in, bearer-authenticated WebSocket stream shares the loopback MCP listener and emits a redacted live event shape without Ratchet capability tokens, metadata, or history. The bounded `mosaic-observe` CLI provides a self-contained operator workflow.

### Fixed

- **Cell identity** — near-anchor placement skips tombstones, exact placement refuses deleted identities, vector writes require a live cell, and changing raw content invalidates the previous embedding.
- **Bounded retrieval** — preserve partial query hits, expose scan and vector candidate limits, and validate bounded exact hybrid queries before embedding.

- Reject unsupported one-time Ratchet alternatives with the pinned dependency. Delegate to Ratchet workflow execution when available.
- Report committed deletes separately from pruning errors. Live database replacement is an offline operator procedure. Existing configurations that enable live compaction must remove `compact: true` before startup.

- **Dependency security** — build Mosaic with the fixed Go crypto module version used by HexxlaDB.

### Changed

- **API documentation** — align the MCP tool inventory, seed workflow, and persistence guide with the pinned HexxlaDB dependency.
- Pin HexxlaDB `v0.7.0`. MVCC is intrinsic to both current storage profiles. Convert engine v1 and AES-XTS files offline before upgrading.

- Move write policy into application values and compose text embedding with vector search through separate ports.

- **Go toolchain** — require Go 1.27.3 or newer. GitHub Actions uses the version in `go.mod`.

- **CI and releases** — validate tagged commits before publishing and include the server, database creator, and observer in release archives.

- **Editor configuration** — keep Zed and VS Code settings local and use the shared Taskfile for development commands.

- **Codex development setup** — document local MCP registration and verification, clarify host ownership of transcript capture, and use the JetBrains Modern Go Guidelines plugin for version-specific idioms.
- **Cell placement safety** ⚠️ **breaking write behavior** — `mosaic_hexxla_put_cell` now defaults to safe exact placement: writing over a live coordinate requires explicit `allow_overwrite: true`. New `placement: near_anchor` atomically composes HexxlaDB `FindUnusedCellPlacement` with `PutCell` using a caller-owned anchor and bounded radius (default 8, maximum 32). Responses report the actual coordinate, placement mode, occupied probes, and whether an exact write replaced a live cell.

- **HexxlaDB v0.6.0 compatibility** — context assembly now retrieves count-bounded candidates with `Tx.LoadContext`; Mosaic owns approximate-token conversion, UTF-8 byte accounting, outer-ring/low-confidence eviction, and compatibility byte aliases in MCP responses.
- **Toolchain and storage defaults** — require Go **1.27.0**, depend on `github.com/hexxla/hexxladb` **v0.6.0**, and use HexxlaDB’s validated **4096-byte** page profile for newly created databases (existing files retain their persisted layout).
- **MCP SDK** — update `github.com/modelcontextprotocol/go-sdk` to **v1.7.0** while retaining Mosaic's stateful Streamable HTTP session model.
- **Repository agent tooling** — replace generated Cursor/Windsurf rules, workflow installers, and platform hook scripts with shared `AGENTS.md` guidance.
- **Build tooling** — replace GNU Make with Task while retaining build, cross-compile, test, integration, install, database setup, MCP development, and maintenance commands.
- **Context responses** — add authoritative `total_bytes` / `max_budget_bytes` fields and optional returned facet text; deprecated `total_tokens` / `max_tokens_budget` remain byte-valued compatibility aliases.
- **Documentation, agent guidance, and seed corpus** — align current HexxlaDB retrieval, embeddings, authenticated-v3 encryption, changefeed, page allocation, retention, and tokenizer ownership; reconcile branch-divergent Ratchet documentation with the implemented opt-in enforcement contract; distinguish runtime MCP schemas from historical and aspirational blueprints; and document the database creation and seed credential contracts.
- **Ratchet observability boundary** — the development branch's unauthenticated HTTP event/statistics handlers and cross-origin WebSocket were replaced by an explicit opt-in live stream that rejects browser origins, requires a private bearer-token file, redacts credential-bearing fields, retains no event history, and never blocks enforcement for slow observers.
- **Ratchet mutation coverage** — when Ratchet is enabled, startup now fails unless every Mosaic mutation has a real prerequisite and no free-pass rule. The supplied policy governs all eight mutation tools through taxonomy, retrieval, context, or seam-inspection workflows.
- **HexxlaDB API coverage documentation** — map every v0.6.0 public feature family to an MCP, composition, native, candidate, or operator-only disposition; record bounded read-only experiments without exposing maintenance, retention, migration, encryption, or raw-KV operations.

## [0.2.0] - 2026-05-05

### Changed

- **README · Get started** — Primary build **`make build-mosaic-mcp build-mosaic-create-db`** (**`bin/<os>-<arch>/`**); **`build-mosaic-seed`** framed as developer-only (**`make seed`** uses **`go run`**). **`go install`** for **`mosaic-mcp`** / **`mosaic-create-db`** only unless using the seed binary.

- **README** — Version badge and release link updated to **`v0.2.0`**.
- **README & changelog links** — GitHub Actions badges, **releases**, clone URL, and **[Unreleased]** compare links use **`github.com/hexxla/mosaic`** (canonical repo). **pkg.go.dev** and **Go Report Card** remain **`github.com/sploitzberg/mosaic`** (Go module path).
- **Dependencies** — **`github.com/modelcontextprotocol/go-sdk`** **v1.6.0** (was v1.5.0).
- **Release workflow** — Drop mandatory **GPG** import (was failing when **`GPG_PRIVATE_KEY`** / **`GPG_PASSPHRASE`** secrets are unset); **`.goreleaser.yml`** does not define **`signs`**. Restore **`crazy-max/ghaction-import-gpg`** and **`GPG_FINGERPRINT`** when you add signing to GoReleaser and repository secrets.
- **README** — Remove trailing spaces (file quality / **Get started** hard breaks merged into single lines).

### Fixed

- **`LiveDB.WithRead`** — wrap callback errors with **`%w`** for consistent wrapping.
- **Post-delete prune** — wrap **`ctx.Err()`** with **`%w`**; MVCC prune loop uses **`for range rounds`** (satisfies **`modernize`**).
- **CI** — **`scripts/ci/pre-commit/16-error-wrapping.sh`** and **`18-go-conventions.sh`** increment warning/error counters with **`var=$((var + 1))`** so **`set -e`** does not abort on the first **`((var++))`** (exit status quirk).

## [0.1.0] - 2026-04-29

First tagged release of **Mosaic**: local **Model Context Protocol** server backed by **[HexxlaDB](https://github.com/hexxla/hexxladb)** for structured agent memory (hex lattice, hybrid retrieval, policy-governed writes, budgeted context).

### Breaking changes

- Go module path **`github.com/sploitzberg/mosaic`** (replaces **`github.com/sploitzberg/go-llm-project-structure`**). Bootstrap **`cmd/go-llm-project-structure`** removed — use **`mosaic-mcp`**, **`mosaic-seed`**, **`mosaic-create-db`**.

### MCP server (`mosaic-mcp`)

- **Transport:** Streamable HTTP on loopback (`MOSAIC_MCP_ADDR` / `MOSAIC_MCP_PATH`); official Go MCP SDK.
- **Reads:** **`mosaic_hexxla_health`**, **`mosaic_hexxla_query_cells`**, **`mosaic_hexxla_search_cells`** (optional hybrid **`embed_query_text`**), **`mosaic_hexxla_search_embedding`**, **`mosaic_hexxla_load_context_pack`**, **`mosaic_hexxla_estimate_context_budget_bytes`**, **`mosaic_hexxla_retrieval_budget_status`**, facet/edge **`get`** / **`list`**, **`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`**, **`mosaic_hexxla_get_persistence_policy`**.
- **Writes:** **`mosaic_hexxla_put_cell`**, **`mosaic_hexxla_put_embedding`**, **`mosaic_hexxla_delete_cell`** ( **`DeleteCellMutationResult`** with **`cell_removed`** ), seam and facet/edge mutations as documented in the README tool map.

### Configuration YAML (`version: 1`)

- **`retention`** (capture mode, boolean enforcement, agent **`notes`**), **`allow_delete_cell`**, optional **`retrieval`** session metering, optional **`ollama`** (**`base_url`**, **`embed_model`** — override **`MOSAIC_OLLAMA_URL`** / **`MOSAIC_EMBED_MODEL`** when set), optional **`database`** (passphrase hint, **`mvcc_retain_commits_behind_head`**, **`auto_maintain_after_cell_delete`** with debounced prune/compact). See **`docs/mosaic/MOSAIC_CONFIG.md`**.

### CLIs & database paths

- **`mosaic-create-db`** — empty Mosaic-compatible **`*.hexxla`** files; **`mosaic-seed`** — demo corpus + Ollama embeddings; optional **`-policy`**, **`MOSAIC_POLICY_FILE`**; **`-replace`** / **`-force`** alias.
- **`ResolveMosaicDBPath`**, **`-db`**, **`-name`**, **`MOSAIC_DB_DIR`**, **`MOSAIC_DB_PATH`**; default file **`mosaic.hexxla`** in the process cwd when path omitted (create/seed). At-rest encryption via env flags or YAML (**`DATABASE_CREATION.md`**).

### Documentation & CI

- Operator docs: **`docs/mosaic/`** (DATABASE_CREATION, MCP blueprints, troubleshooting, Hexxla mapping, roadmap), **`configs/config.yaml`**, **`README`** with badges aligned to HexxlaDB style.

### Depends on

- **`github.com/hexxla/hexxladb v0.3.0`**

### Repository / automation

- GitHub Actions **`actions/checkout@v6`**, **`actions/setup-go@v6`**; **`MOSAIC_POLICY_FILE`** forwarded in **`Makefile`** for seed/MCP/create-db when set.

[Unreleased]: https://github.com/hexxla/mosaic/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/hexxla/mosaic/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/hexxla/mosaic/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/hexxla/mosaic/releases/tag/v0.1.0
