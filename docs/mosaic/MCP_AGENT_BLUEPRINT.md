# Mosaic MCP — agent blueprint

**Audience:** Humans and AI assistants configuring or using the Mosaic MCP server (`cmd/mosaic-mcp`) against a HexxlaDB file.

**Companion:** Repository-wide agent policy in [`AGENTS.md`](../../AGENTS.md); this document contains the Mosaic-specific retrieval and persistence workflow.

---

## Why two “context” layers exist

- **Retrieval** (`mosaic_hexxla_search_cells` and exact `mosaic_hexxla_get_cell`) answers: *which cells match (similarity, tags, text, filters)?* Outputs are **bounded lists** (top‑K or capped query results).
- **Neighbourhood assembly** (`mosaic_hexxla_load_context_pack`) answers: *given one or more **seed coordinates** on the hex lattice, what **budgeted** slice of nearby cells (and optionally seams / supersession) belongs in a prompt?*

Hexxla upstream demos combine both: retrieve seeds → `Tx.LoadContext`. Embedding-only answers can be sufficient when top‑K hits already contain enough evidence; otherwise **always** consider a context pack.

### HexxlaDB retrieval vs Mosaic budgeting

Use `search_cells` with lexical (default), semantic or hybrid mode, or use
`get_cell` when the exact coordinate is known. Search returns bounded results
and reports scan/candidate limits when reached. Inspect the returned execution
path and limits before widening a search. Semantic mode is ANN-only; structured
filters or non-default sorting belong to lexical/hybrid search.

HexxlaDB **`Tx.LoadContext`** retrieves deterministic, **`MaxCells`**-bounded assembled candidates from one or more seeds. It deliberately does not know about LLM providers or tokenizers. Mosaic’s **`mosaic_hexxla_load_context_pack`** wraps that API, then enforces a UTF-8 byte ceiling by evicting the lowest-confidence cell from the outermost occupied ring. Optional seams, supersession filtering, explanations, and facet text remain available.

The advanced byte-estimation tool provides only a configurable token-to-byte estimate. Exact fit depends on the provider, model, rendered messages, tool schemas, and reserved output; the client that constructs the final request remains responsible for exact tokenizer accounting.

The per-call byte ceiling counts cell **`raw_content`** plus returned **`facet_text`** when `include_facet_text` is enabled. It does not claim to measure the complete JSON envelope or a rendered provider request.

---

## Recommended workflow

1. Use **health** if you need embedding dimension, DB path sanity, or integrity before heavy queries.
2. List **`mosaic_hexxla_list_tags`** before writing; reuse established vocabulary. Tag-frequency browsing is available only with advanced tools enabled.
3. Discover with **`mosaic_hexxla_search_cells`** using lexical, semantic or hybrid mode; fetch an exact coordinate with **`mosaic_hexxla_get_cell`**.
4. Inspect tool JSON for **`retrieval_hint`** when present; it steers toward **`mosaic_hexxla_load_context_pack`** when hits may be incomplete for the user’s question.
5. Call **`mosaic_hexxla_load_context_pack`** with:
   - **`seeds`**: array of `{ "q", "r" }` from retrieval hits (often 1–3 seeds)
   - **`max_budget_bytes`**: a UTF-8 byte ceiling; omitted or zero defaults to **4096**. Explicit values must be **64–100000**.
   - **`max_ring`** (default 3, max 32), **`max_cells`** (default 256, max 8192), optional **`filter_superseded`**, **`include_seams`**, **`include_facet_text`**, **`explain`**.
6. Responses expose **`total_bytes`**, **`max_budget_bytes`**, and **`retrieval_hint`**. Token-named aliases and alternate budget inputs are removed. With advanced tools enabled, `mosaic_hexxla_estimate_context_budget_bytes` can convert approximate tokens into a byte limit before the call.

---

## Session retrieval budget & metering (optional)

Configured in Mosaic YAML under **`retrieval`** — see **[`MOSAIC_CONFIG.md`](./MOSAIC_CONFIG.md)** (`retrieval` section); example values-only file [`configs/config.yaml`](../../configs/config.yaml).

- **Per-call shaping (relevance / waste)** — Use **small rings**, **`max_results` / `max_cells`**, and **byte budgets** on **`mosaic_hexxla_load_context_pack`** (and related read tools) so each response stays tight. That is the primary lever for “don’t pull irrelevant bulk.”
- **Cumulative session cap (egress / runaway)** — **`retrieval.session_approx_token_budget`** (when **> 0**) limits **total approximate JSON output** metered per MCP **session**. Counters are released when the session closes; idle HTTP sessions expire after 30 minutes. It is an **egress and runaway-loop brake**, **not** a relevance filter and **not** a substitute for per-call budgets.
- **Advanced observability** — with `advanced_tools: true`, **`mosaic_hexxla_retrieval_budget_status`** reports **`approx_tokens_used`**, whether **enforcement** is on (**`budgeting_enabled`**), and whether metering is active (**`metering_enabled`**). With **`session_approx_token_budget: 0`**, there is **no hard cap**, but usage can still be **metered** for visibility.

Keep this distinction explicit in docs and agent instructions so expectations do not drift: **small incremental retrieval** vs **optional cumulative cap**.

---

## Persistence: what gets saved

- Mosaic **does not** auto-save chat turns. Long-lived store of user/model text is explicit: tools that **write** cells (**`mosaic_hexxla_put_cell`**, **`mosaic_hexxla_put_embedding`**, etc.) persist data the client chooses to submit (e.g. **`kind`** `user_message` / `assistant_response`, **`source_id`** for session/session key). There is **no** default “record everything”; retrieval/query tools only **read**.
- Decide your product policy (what to store, TTL, PIIs) **above** Mosaic; expose only via mutation calls.

### Ratchet prerequisites when enabled

An operator may enable the supplied Ratchet workflow. In that deployment, use
the prerequisite named by a denial instead of retrying the mutation:

- `put_cell` follows a fresh `list_tags`;
- `put_embedding`, `put_facet`, and `delete_cell` follow a successful cell
  retrieval;
- `link_cells` follows `load_context_pack`; and
- `mark_conflict`, `mark_supersedes`, and `resolve_seam` follow `find_seams`.

Single-prerequisite write tokens are consumed once. Retrieval alternatives are
reusable until their token expires. The state belongs to the current MCP
session and is lost when the server restarts. See
[`RATCHET_INTEGRATION.md`](./RATCHET_INTEGRATION.md) for the exact operator
contract; Ratchet ordering does not replace authorization or persistence
policy.

### Cell placement on writes

`mosaic_hexxla_put_cell` supports two explicit modes:

- **`placement: exact`** (default) treats `(q,r)` as the final coordinate. Updating a live cell requires **`allow_overwrite: true`**. Deleted coordinates remain reserved.
- **`placement: near_anchor`** treats `(q,r)` as an application-chosen anchor and atomically selects the first unused coordinate in ring order, skipping tombstones. **`max_radius`** defaults to 8 and is capped at 32.

Changing the cell's raw content removes its previous embedding in the same
transaction. Set `embed: true` on `put_cell` to compute and commit its new vector atomically. Independent embedding repair remains an advanced operation.

For delayed embedding work, send `expected_content_sha256` to
`mosaic_hexxla_put_embedding`: the 64-digit SHA-256 hex digest of the exact
UTF-8 `raw_content` retrieved earlier. The write atomically compares it with
the live cell and rejects a mismatch, preserving any current vector. Both
text and vector inputs support this guard. Omit it to retain unconditional
embedding replacement; the embedding text may deliberately differ from raw content.

A durable-commit or uncertain finalization error can mean a mutation was
saved. Reconcile by source ID and content before retrying, especially with
near-anchor placement, which can otherwise create another cell. A tool error
alone does not establish rollback.

The response always reports **`coord`**, **`placement`**, **`probes`**, and **`replaced`**. Persist the returned coordinate for later embeddings, facets, edges, seams, and exact retrieval. Mosaic and HexxlaDB resolve bounded geometric collisions but do not infer anchors from content, tags, or embeddings.

### Full-turn persistence (explicit agent workflow)

When the host explicitly authorizes full-turn capture, it can submit the two
messages through `mosaic_hexxla_put_cell`. If every exchange must be captured,
the host should own that workflow and handle failed writes. The active server
policy controls the kinds accepted for submitted calls.

For an authorized client-managed write sequence:

1. Right after the user message is known → **`put_cell`** with **`kind=user_message`** (session **`source_id`**, tags, and either an exact coordinate or `near_anchor` placement); retain the returned coordinate.
2. Generate the reply (retrieval steps from [Recommended workflow](#recommended-workflow) as needed).
3. After the assistant reply is finalized → **`put_cell`** with **`kind=assistant_response`** (same **`source_id`** and an explicit placement choice); retain the returned coordinate.

YAML **`retention.capture_mode: save_all_turns`** matches “both sides in scope”; use **`mosaic_hexxla_get_persistence_policy`** and [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md) for enforcement details. Mosaic MCP keeps durable context outside any particular editor or agent client.

---

## Client-side instructions (non-repo)

Use the retrieval sequence above when configuring a client for Mosaic. Tool descriptions and JSON hints carry the same guidance. For local Codex setup, see the [development tooling guide](./DEVELOPMENT_TOOLING.md#codex-mcp-setup).

---

## Security posture

- Bind MCP **loopback** only in production-style setups; see [`MCP_BLUEPRINT.md`](./MCP_BLUEPRINT.md).
- Never put credentials in MCP tool arguments; DB encryption and paths come from environment (`MOSAIC_DB_PATH`, etc.).

---

## See also

- [`HEXXLA_TROUBLESHOOTING.md`](./HEXXLA_TROUBLESHOOTING.md) — MCP deletes, **`mosaic_hexxla_health`** / **`integrity_ok`**, MVCC index semantics
- [`../ROADMAP.md`](../ROADMAP.md) — roadmap themes; [`../../TODOS.md`](../../TODOS.md) — session scratchpad
- [`DEVELOPMENT_TOOLING.md`](./DEVELOPMENT_TOOLING.md) — repository agent policy, project skill, and editor tasks
- [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md) — YAML `retention` / `allow_delete_cell` (startup file, `mosaic_hexxla_get_persistence_policy`)
- [`HEXXLA_API_SURFACE_COVERAGE.md`](./HEXXLA_API_SURFACE_COVERAGE.md) — Hexxla ↔ MCP capability matrix
