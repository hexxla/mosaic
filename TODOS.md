# Active Work

Immediate next steps. Update after each session.

## Current

## Pending (next sessions)

- [ ] **Large retrieval responses** — Context packs trim content against the remaining session budget. Query, search, and diagnostic responses still rely on the final JSON size check. Profile representative large responses before adding tool-specific shaping.
- [ ] **Approximate metering vs tokenizer** — Session `approx_tokens_used` is derived from JSON UTF-8 length ÷ `bytes_per_approx_token`. Explore tokenizer alignment if the product needs it.

---

## Recently Completed

- 2026-08-26: **Agent retrieval guidance** — documented lexical vs semantic/hybrid selection, normalized empty lexical results, and recommended tagged preference queries in the MCP blueprint.
- 2026-04-29: **Post-delete maintenance** — **`database.auto_maintain_after_cell_delete.debounce_after_delete_ms`** (default 2s when enabled); **`FlushPostDeleteMaintain`** on **`mosaic-mcp`** shutdown; prune + compact batching ([`internal/adapter/secondary/hexxlastore`](internal/adapter/secondary/hexxlastore)).
- 2026-04-29: **`mosaic_hexxla_health`** — JSON **`disk`** (primary + WAL **`Stat`**) and **`mvcc_retain_commits_behind_head`** (effective policy).
- 2026-04-29: **Policy / onboarding docs** — **[`docs/mosaic/MOSAIC_CONFIG.md`](docs/mosaic/MOSAIC_CONFIG.md)** full YAML reference; **[`configs/config.yaml`](configs/config.yaml)** values-only example; **[`README.md`](README.md)** Get started + feature overview.
- 2026-04-29: **Go module & build** — **`module github.com/sploitzberg/mosaic`**; removed legacy **`cmd/go-llm-project-structure`**; **`task build`** / CI pre-push build **`mosaic-mcp`**.
- 2026-04-29: **Troubleshooting** — **[`docs/mosaic/HEXXLA_TROUBLESHOOTING.md`](docs/mosaic/HEXXLA_TROUBLESHOOTING.md)** disk / **`mosaic_hexxla_health`** section; MCP tool description updates.

## Usage Notes

- **Active**: Work in progress or next immediate task
- **Pending**: Backlog for future sessions
- Move items to [`docs/ROADMAP.md`](docs/ROADMAP.md) when they become formal roadmap themes
- Create GitHub issues for bugs or external collaboration needs
- This file is intentionally lightweight and disposable

For shipped features and fixes see [`CHANGELOG.md`](CHANGELOG.md).
