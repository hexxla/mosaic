# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Common Changelog](https://common-changelog.org/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **README · Get started** — Primary build **`make build-mosaic-mcp build-mosaic-create-db`** (**`bin/<os>-<arch>/`**); **`build-mosaic-seed`** framed as developer-only (**`make seed`** uses **`go run`**). **`go install`** for **`mosaic-mcp`** / **`mosaic-create-db`** only unless using the seed binary.

- **README** — Version badge uses a **static** **`v0.1.0`** shield (linked to **`/releases/tag/v0.1.0`**) because **`img.shields.io/github/v/tag/hexxla/mosaic`** resolves to **no tag** until **`v*`** tags exist on the GitHub repo; switch back to **`github/v/tag`** after the first tag is published if you prefer a dynamic badge.
- **README & changelog links** — GitHub Actions badges, **releases**, clone URL, and **[Unreleased]** compare links use **`github.com/hexxla/mosaic`** (canonical repo). **pkg.go.dev** and **Go Report Card** remain **`github.com/sploitzberg/mosaic`** (Go module path).
- **Dependencies** — **`github.com/modelcontextprotocol/go-sdk`** **v1.6.0** (was v1.5.0).
- **Release workflow** — Drop mandatory **GPG** import (was failing when **`GPG_PRIVATE_KEY`** / **`GPG_PASSPHRASE`** secrets are unset); **[`.goreleaser.yml`](.goreleaser.yml)** does not define **`signs`**. Restore **`crazy-max/ghaction-import-gpg`** and **`GPG_FINGERPRINT`** when you add signing to GoReleaser and repository secrets.
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

[Unreleased]: https://github.com/hexxla/mosaic/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/hexxla/mosaic/releases/tag/v0.1.0
