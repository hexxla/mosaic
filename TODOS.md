# Active Work

Immediate next steps. Update after each session.

## Current

## Pending (next sessions)

- [ ] **Oversized one-shot (session cap)** — **What it is:** With `retrieval.session_approx_token_budget` > 0, enforcement uses **prior** cumulative usage only in `BeforeRead`. If the session is **under** the cap but **one** read returns a structured payload so large that `used + approx_tokens(this JSON)` would exceed the cap, HexxlaDB / assembly **still runs** for that call; we only fail when **recording** the response (`RecordStructuredOutput`). The client gets an error and usage is **not** incremented—so you paid DB work without delivering a “successful” retrieval under the cap. **Explore:** stricter per-call defaults (`max_results`, rings, context-pack budgets), truncation policy, size pre-estimates, or reject-before-query where bounds are known.
- [ ] **Session meter lifetime** — Usage is in-memory, keyed by MCP session id (process lifetime). Fine for dev and many deployments; for “per chat” or “per day” semantics, add explicit reset/TTL, an operator story, or a documented client reconnect boundary.
- [ ] **Approximate metering vs tokenizer** — Session `approx_tokens_used` is derived from JSON UTF-8 length ÷ `bytes_per_approx_token`, not the host model tokenizer. Explore clearer docs-only labeling vs optional alignment hooks if product needs it.
- [ ] **Cursor MCP server id vs `mcp.json` name** — **Why:** Hosts (e.g. Cursor) may expose an internal server id (e.g. `project-0-mosaic-mosaic`) while a local `.cursor/mcp.json` uses a short key (`mosaic`). Tool dispatch can fail with “server does not exist” if callers use the wrong identifier. **How:** Document both in [`README.md`](README.md) / [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](docs/mosaic/MCP_AGENT_BLUEPRINT.md); link [Cursor MCP docs](https://cursor.com/docs/mcp); if the product allows a single stable name, prefer that in examples.
- [ ] **Refuse missing MCP database paths** — `mosaic-mcp` currently calls `hexxladb.Open`, which creates a file when the path is absent. Add an explicit regular-file existence check before open so typos cannot start an empty database without Mosaic's embedding layout; test missing, directory, and valid-file paths.
- [ ] **Make seed policy credentials explicit** — `mosaic-seed -policy` currently loads Ollama fields but ignores YAML `database.passphrase`. Either wire the YAML value into `ApplyHexxlaEncryption` with flag → env → YAML precedence and regression tests, or deliberately remove the policy credential promise from the command surface.

---

## Recently Completed

- 2026-08-26: **Agent retrieval guidance** — documented lexical vs semantic/hybrid selection, normalized empty lexical results, and recommended tagged preference queries in the MCP blueprint and Cursor rule.
- 2026-04-29: **Post-delete maintenance** — **`database.auto_maintain_after_cell_delete.debounce_after_delete_ms`** (default 2s when enabled); **`FlushPostDeleteMaintain`** on **`mosaic-mcp`** shutdown; prune + compact batching ([`internal/adapter/secondary/hexxlastore`](internal/adapter/secondary/hexxlastore)).
- 2026-04-29: **`mosaic_hexxla_health`** — JSON **`disk`** (primary + WAL **`Stat`**) and **`mvcc_retain_commits_behind_head`** (effective policy).
- 2026-04-29: **Policy / onboarding docs** — **[`docs/mosaic/MOSAIC_CONFIG.md`](docs/mosaic/MOSAIC_CONFIG.md)** full YAML reference; **[`configs/config.yaml`](configs/config.yaml)** values-only example; **[`README.md`](README.md)** Get started + feature overview.
- 2026-04-29: **Go module & build** — **`module github.com/sploitzberg/mosaic`**; removed legacy **`cmd/go-llm-project-structure`**; **`make build`** / CI pre-push build **`mosaic-mcp`**; **[`.goreleaser.yml`](.goreleaser.yml)** publishes **`mosaic-mcp`**.
- 2026-04-29: **Troubleshooting** — **[`docs/mosaic/HEXXLA_TROUBLESHOOTING.md`](docs/mosaic/HEXXLA_TROUBLESHOOTING.md)** disk / **`mosaic_hexxla_health`** section; MCP tool description updates.

## Usage Notes

- **Active**: Work in progress or next immediate task
- **Pending**: Backlog for future sessions
- Move items to [`docs/ROADMAP.md`](docs/ROADMAP.md) when they become formal roadmap themes
- Create GitHub issues for bugs or external collaboration needs
- This file is intentionally lightweight and disposable

For shipped features and fixes see [`CHANGELOG.md`](CHANGELOG.md).
