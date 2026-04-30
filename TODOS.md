# Active Work

Immediate next steps. Update after each session.

## Current

## Pending (next sessions)

- [ ] **Oversized one-shot (session cap)** — **What it is:** With `retrieval.session_approx_token_budget` > 0, enforcement uses **prior** cumulative usage only in `BeforeRead`. If the session is **under** the cap but **one** read returns a structured payload so large that `used + approx_tokens(this JSON)` would exceed the cap, HexxlaDB / assembly **still runs** for that call; we only fail when **recording** the response (`RecordStructuredOutput`). The client gets an error and usage is **not** incremented—so you paid DB work without delivering a “successful” retrieval under the cap. **Explore:** stricter per-call defaults (`max_results`, rings, context-pack budgets), truncation policy, size pre-estimates, or reject-before-query where bounds are known.
- [ ] **Session meter lifetime** — Usage is in-memory, keyed by MCP session id (process lifetime). Fine for dev and many deployments; for “per chat” or “per day” semantics, add explicit reset/TTL, an operator story, or a documented client reconnect boundary.
- [ ] **Approximate metering vs tokenizer** — Session `approx_tokens_used` is derived from JSON UTF-8 length ÷ `bytes_per_approx_token`, not the host model tokenizer. Explore clearer docs-only labeling vs optional alignment hooks if product needs it.
- [ ] **Cursor MCP server id vs `mcp.json` name** — **Why:** Hosts (e.g. Cursor) may expose an internal server id (e.g. `project-0-mosaic-mosaic`) while `.cursor/mcp.json` uses a short key (`mosaic`). Tool dispatch can fail with “server does not exist” if callers use the wrong identifier. **How:** Document both in [`README.md`](README.md) / [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](docs/mosaic/MCP_AGENT_BLUEPRINT.md) (and optionally a comment in [`.cursor/mcp.json`](.cursor/mcp.json) or `docs/mosaic/CURSOR_MCP.md`); link [Cursor MCP docs](https://cursor.com/docs/mcp); if the product allows a single stable name, prefer that in examples.
- [ ] **Lexical vs embedding retrieval (agent guidance)** — **Why:** `mosaic_hexxla_search_cells` matches **substrings**; `mosaic_hexxla_search_embedding` finds **semantic** neighbors. Expect **zero lexical hits** when the phrase is not stored literally, while ANN can still return related cells—agents may wrongly treat empty lexical as “DB empty.” **How:** Extend [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](docs/mosaic/MCP_AGENT_BLUEPRINT.md) and [`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc) with a short decision table (when to use lexical, hybrid `embed_query_text`, pure embedding) and explicitly normalize “empty lexical + non-empty ANN.”
- [ ] **Tagged preferences via `query_cells`** — **Why:** “Fetch preferences” is ambiguous as plain full-text search. Cells tagged **`preference`** give a **stable, filterable** slice. **How:** Document **`mosaic_hexxla_query_cells`** with **`require_tags`: [`preference`]** (and `sort_by` / `max_results`) as the recommended pattern in the blueprint and agent rule; contrast with vague `query`-only search.

---

## Recently Completed

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
