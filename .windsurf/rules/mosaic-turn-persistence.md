# Mosaic — full-turn persistence (MCP)

Applies when this repo’s Mosaic MCP server is enabled and **operator policy** requires storing **every** user/assistant exchange (YAML `retention.capture_mode: save_all_turns` or explicit “persist all turns” product policy). Canonical detail: [`.cursor/rules/mosaic-mcp-agent.mdc`](../../.cursor/rules/mosaic-mcp-agent.mdc).

## Bookend every exchange with `put_cell`

1. **Immediately after** the user message is available: call **`mosaic_hexxla_put_cell`** with **`kind=user_message`**, **`raw_content`** = user text, stable **`source_id`** (e.g. session id), and explicit exact or `near_anchor` placement; retain the returned coordinate.
2. Reason, retrieve, and write the assistant reply.
3. **Before ending the turn:** **`mosaic_hexxla_put_cell`** with **`kind=assistant_response`**, same **`source_id`**, assistant text in **`raw_content`**, and an explicit placement choice; retain the returned coordinate.

If taxonomy is unknown, call **`mosaic_hexxla_list_tags`** / **`mosaic_hexxla_tag_counts`** before writes and **reuse** existing tags. Call **`mosaic_hexxla_get_persistence_policy`** when unsure which kinds the server allows.

**Note:** Cursor and other editors do not replace this with built-in long-lived memory; durable context is **your** HexxlaDB via MCP. See [Cursor 3.0 and external memory](https://omegamax.co/blog/cursor-removed-memories).
