---
description: Save each conversation turn to Mosaic per retention policy (save_all_turns)
---

# Mosaic Save Turns Workflow

Ensures compliance when operator policy requires persisting **every** user and assistant turn via **`mosaic_hexxla_put_cell`** (see [`configs/config.yaml`](../../configs/config.yaml) **`retention`**, [`docs/mosaic/PERSISTENCE_POLICY.md`](../../docs/mosaic/PERSISTENCE_POLICY.md), and repo rule [`.windsurf/rules/mosaic-turn-persistence.md`](../rules/mosaic-turn-persistence.md)).

## When to use

Use for **every** turn when Mosaic MCP is connected and **`capture_mode`** (with **`enforcement`**, if enabled) requires both turn kinds—typically **`save_all_turns`**. If unsure, call **`mosaic_hexxla_get_persistence_policy`** once per session.

## Steps

1. **Receive user message** — Read the incoming user text (and keep the same **`source_id`** for the whole exchange, e.g. session key).

2. **Save user message to Mosaic** — Call **`mosaic_hexxla_put_cell`** with:
   - **`kind`**: `user_message`
   - **`raw_content`**: exact user message text
   - **`source_id`**: stable session identifier (same for steps 2 and 4)
   - **`confidence`**: `1.0` (or policy-appropriate value)
   - **`tags`**: topic-specific tags when known; prefer **`mosaic_hexxla_list_tags`** / **`mosaic_hexxla_tag_counts`** when taxonomy is unknown and **reuse** existing tags instead of near-duplicates. Include discriminators such as `preference`, `question`, `fact` when they apply, plus broad tags like `conversation` / `user-message` if your project uses them.
   - Placement: use default **`exact`** for a known free coordinate, or **`placement: near_anchor`** with `(q,r)` as the application-selected anchor. Retain the returned coordinate; set **`allow_overwrite: true`** only for intentional exact replacement.

3. **Generate response** — Answer the user (retrieval: embedding / query / search → **`mosaic_hexxla_load_context_pack`** when neighbourhood context is needed; see [`.cursor/rules/mosaic-mcp-agent.mdc`](../../.cursor/rules/mosaic-mcp-agent.mdc)).

4. **Save assistant response to Mosaic** — Call **`mosaic_hexxla_put_cell`** with:
   - **`kind`**: `assistant_response`
   - **`raw_content`**: your full reply text
   - **`source_id`**: **same** as step 2
   - **`confidence`**: `1.0` (or policy-appropriate)
   - **`tags`**: aligned with the topic of the turn; reuse vocabulary from **`list_tags`** when possible; include `assistant-response` / `conversation` if your taxonomy uses them.
   - Placement: use an explicit exact coordinate or bounded `near_anchor` allocation and retain the returned coordinate.

5. **Display response** — Show the reply in chat.

## Critical rules

- **Do not** ask the user whether to store history when policy mandates persistence—it must be **automatic** for every turn.
- Both **`user_message`** and **`assistant_response`** must be written for each exchange when **`save_all_turns`** + enforcement (or equivalent operator mandate) applies.
- **`source_id`** must match across both puts in the same exchange.
- Use **specific, searchable tags** where they improve **`mosaic_hexxla_query_cells`** / filter retrieval—not **only** generic tags.

## Notes

- Active policy is loaded at MCP startup; **`configs/config.yaml`** in this repo is an example (`capture_mode`, **`enforcement`**, **`notes`** surfaced in server instructions).
- Tag strategy complements semantic search: lexical or tag-filtered queries can recover cells when embedding search alone misses.
