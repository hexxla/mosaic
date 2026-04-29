# Mosaic MCP — agent blueprint

**Audience:** Humans and AI assistants configuring or using the Mosaic MCP server (`cmd/mosaic-mcp`) against a HexxlaDB file.

**Companion:** Cursor rule [`.cursor/rules/mosaic-mcp-agent.mdc`](../../.cursor/rules/mosaic-mcp-agent.mdc) (concise); this document expands rationale and references.

---

## Why two “context” layers exist

- **Retrieval** (`mosaic_hexxla_search_embedding`, `mosaic_hexxla_query_cells`, `mosaic_hexxla_search_cells`) answers: *which cells match (similarity, tags, text, filters)?* Outputs are **bounded lists** (top‑K or capped query results).
- **Neighbourhood assembly** (`mosaic_hexxla_load_context_pack`) answers: *given one or more **seed coordinates** on the hex lattice, what **budgeted** slice of nearby cells (and optionally seams / supersession) belongs in a prompt?*

Hexxla upstream demos combine both: retrieve seeds → `LoadContextPackFrom`. Embedding-only answers can be sufficient when top‑K hits already contain enough evidence; otherwise **always** consider a context pack.

### `LoadContext` / `LoadContextAt` vs `LoadContextPackFrom`

Hexxla also exposes **`Tx.LoadContext`** and **`Tx.LoadContextAt`** — simpler ring walks with **count** caps and raw records, optional validity-at-time (**`LoadContextAt`**). Mosaic does **not** wrap these in MCP; **`mosaic_hexxla_load_context_pack`** is the intended tool for prompts (**token/byte budgeting**, **`CellView`** assembly, seams, **`FilterSuperseded`**). Use the low-level APIs only from custom Go around Hexxla, not via MCP unless a dedicated tool is added later.

---

## Recommended workflow

1. Use **health** if you need embedding dimension, DB path sanity, or integrity before heavy queries.
2. Optionally list **distinct tags** or **tag frequencies** (**`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`**) before writing new cells—reuse existing vocabulary—or when retrieval queries keep missing due to wording.
3. **Discover** with the narrowest tool:
   - semantic ANN-only → **`mosaic_hexxla_search_embedding`**
   - structured filters (+ optional **`embed_query_text`** for hybrid ANN + predicates) → **`mosaic_hexxla_query_cells`**
   - lexical relevance (+ optional **`embed_query_text`** for hybrid) → **`mosaic_hexxla_search_cells`**
4. Inspect tool JSON for **`retrieval_hint`** when present; it steers toward **`mosaic_hexxla_load_context_pack`** when hits may be incomplete for the user’s question.
5. Call **`mosaic_hexxla_load_context_pack`** with:
   - **`seeds`**: array of `{ "q", "r" }` from retrieval hits (often 1–3 seeds)
   - **Budget (UTF‑8 bytes, Hexxla `ByteLenBudgeter`)** — pick one mode (see tool JSON for exclusivity rules):
     - **`omit_budget`**: sparse “no tight cap” semantics → server resolves to the maximum allowed clamp (100000 UTF‑8 bytes).
     - **`budget_tokens_approx`** (+ optional **`bytes_per_approx_token`**, default 4): approximate LM tokens → bytes (same math as **`mosaic_hexxla_estimate_context_budget_bytes`**).
     - **`max_budget_bytes`** or legacy **`max_tokens`**: explicit byte ceiling (both mean the same field to Hexxla; do not send conflicting values).
     - If none of the above: default **4096** bytes.
   - Optional **`mosaic_hexxla_estimate_context_budget_bytes`** to preview **`budget_bytes`** before loading a pack.
   - **`max_ring`**, **`max_cells`**, optional **`filter_superseded`**, **`include_seams`**, **`explain`**
6. Response includes **`retrieval_hint`** reminding when **global** semantic search vs **local** pack applies.

---

## Session retrieval budget & metering (optional)

Configured in Mosaic YAML under **`retrieval`** (see [`configs/config.yaml`](../../configs/config.yaml)).

- **Per-call shaping (relevance / waste)** — Use **small rings**, **`max_results` / `max_cells`**, and **byte or approximate-token budgets** on **`mosaic_hexxla_load_context_pack`** (and related read tools) so each response stays tight. That is the primary lever for “don’t pull irrelevant bulk.”
- **Cumulative session cap (egress / runaway)** — **`retrieval.session_approx_token_budget`** (when **> 0**) limits **total approximate JSON output** metered per MCP **session** over the life of the server process (until reconnect/restart or a new session id). It is an **egress and runaway-loop brake**, **not** a relevance filter and **not** a substitute for per-call budgets.
- **Observability** — **`mosaic_hexxla_retrieval_budget_status`** reports **`approx_tokens_used`**, whether **enforcement** is on (**`budgeting_enabled`**), and whether metering is active (**`metering_enabled`**). With **`session_approx_token_budget: 0`**, there is **no hard cap**, but usage can still be **metered** for visibility.

Keep this distinction explicit in docs and agent instructions so expectations do not drift: **small incremental retrieval** vs **optional cumulative cap**.

---

## Persistence: what gets saved

- Mosaic **does not** auto-save chat turns. Long-lived store of user/model text is explicit: tools that **write** cells (**`mosaic_hexxla_put_cell`**, **`mosaic_hexxla_put_embedding`**, etc.) persist data the client chooses to submit (e.g. **`kind`** `user_message` / `assistant_response`, **`source_id`** for session/session key). There is **no** default “record everything”; retrieval/query tools only **read**.
- Decide your product policy (what to store, TTL, PIIs) **above** Mosaic; expose only via mutation calls.

### Full-turn persistence (explicit agent workflow)

When product policy is to **store every user/assistant exchange**, the agent should treat **`mosaic_hexxla_put_cell`** as the **bookends** of each turn:

1. Right after the user message is known → **`put_cell`** with **`kind=user_message`** (session **`source_id`**, project **`(q,r)`** / tags).
2. Generate the reply (retrieval steps from [Recommended workflow](#recommended-workflow) as needed).
3. After the assistant reply is finalized → **`put_cell`** with **`kind=assistant_response`** (same **`source_id`**).

YAML **`retention.capture_mode: save_all_turns`** matches “both sides in scope”; use **`mosaic_hexxla_get_persistence_policy`** and [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md) for enforcement details. Cursor does not provide built-in cross-session Memories (see [external memory note](https://omegamax.co/blog/cursor-removed-memories)); Mosaic MCP is one way to keep durable context **outside** the editor.

---

## Client-side instructions (non-repo)

Cursor / IDE **rules** and team **playbooks** should repeat the short chain: **retrieve → read hints → load context pack if needed**. Tool descriptions and JSON hints in Mosaic already encode this; repeating it in the **system** or **project instructions** layer improves compliance from frontier models.

---

## Security posture

- Bind MCP **loopback** only in production-style setups; see [`MCP_BLUEPRINT.md`](./MCP_BLUEPRINT.md).
- Never put credentials in MCP tool arguments; DB encryption and paths come from environment (`MOSAIC_DB_PATH`, etc.).

---

## See also

- [`../ROADMAP.md`](../ROADMAP.md) — roadmap themes; [`../../TODOS.md`](../../TODOS.md) — session scratchpad
- [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md) — YAML `retention` / `allow_delete_cell` (startup file, `mosaic_hexxla_get_persistence_policy`)
- [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md) — roadmap and shipped tools
- [`HEXXLA_API_ROADMAP.md`](./HEXXLA_API_ROADMAP.md) — Hexxla API coverage
- [`HYBRID_RETRIEVAL_PLAN.md`](./HYBRID_RETRIEVAL_PLAN.md) — hybrid **`embed_query_text`** backlog and token-budget pointers
