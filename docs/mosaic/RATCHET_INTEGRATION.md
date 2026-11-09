# Ratchet integration

Mosaic optionally enforces session-local tool prerequisites with the shared
[mcp-ratchet](https://github.com/hexxla/mcp-ratchet) receiving middleware. Mosaic
selects governed tools and validates policy coverage; Ratchet owns execution,
completion expiry and consumption, and final-response serialization.

## Activation and profiles

Set `ratchet: true` in the normal Mosaic config and load it with `-policy` or
`MOSAIC_POLICY_FILE`. This enables the embedded
[`ratchet_policy.yaml`](../../internal/config/ratchet_policy.yaml); no separate
workflow file is needed. In-process defaults leave Ratchet disabled. The shipped
[`configs/config.yaml`](../../configs/config.yaml) enables it.

A custom `-ratchet-config PATH` or `MOSAIC_RATCHET_CONFIG_FILE` also enables
Ratchet, even when `ratchet` is false or omitted. The flag takes precedence.
Policies are loaded once at startup. Missing files, unknown tool names, invalid
rules, and missing protection for enabled mutations fail startup.

The default profile exposes seven tools: health, search_cells, get_cell,
load_context_pack, put_cell, delete_cell, and list_tags (all with the
`mosaic_hexxla_` prefix). `advanced_tools: true` adds raw embedding writes,
facets/edges, seams, tag counts, policy inspection, retrieval-budget status and
byte-budget estimation. Policy coverage follows the enabled profile; inactive
guards are removed before the engine is constructed. A retained guard must
still have an enabled prerequisite.

## Bundled workflow

| Tool | Recent successful prerequisite | Consumption |
| --- | --- | --- |
| `put_cell` | `list_tags` | Once per write |
| `put_embedding`, `put_facet`, `delete_cell`, `load_context_pack` | `get_cell` or `search_cells` | Reusable |
| `link_cells` | `load_context_pack` | Once per link |
| `mark_conflict`, `mark_supersedes`, `resolve_seam` | `find_seams` | Once per mutation |

Completions last ten minutes by default. Completions issued by
`load_context_pack` last five minutes. Only tools needed as prerequisites retain
completion state. A failed call spends any reserved one-time completion and
issues no downstream completion, including SDK validation or serialization
failures. Same-session execution is serialized; independent sessions remain
isolated. State is bounded, process-local and lost on restart.

A custom policy uses grouped alternatives:

```yaml
completion_ttl: 10m
rules:
  - tool: mosaic_hexxla_put_cell
    after_any: [mosaic_hexxla_list_tags]
    consume: true
  - tool: mosaic_hexxla_delete_cell
    after_any: [mosaic_hexxla_get_cell, mosaic_hexxla_search_cells]
```

`after_any` alternatives are OR requirements. `consume` spends the selected
completion before the SDK call. `ttl` changes the lifetime of completions issued
by that tool; `message` optionally supplies a denial message. The old
`prerequisite`, `expiry`, `one_time_use` and `error_message` keys are rejected.

## Security and live observability

Ratchet controls workflow order. Authentication, resource authorization,
argument validation, retention policy and the deletion gate remain the host's
responsibility. Completions attest to successful calls in one session; they do
not bind a retrieved coordinate to later mutation arguments.

Mosaic keeps its loopback-only listener, Host and Origin checks, idle session
expiry, and explicit tool safety inventory. Shutdown stops new tool admissions,
drains active SDK handlers even after their POST connection disconnects, closes
sessions, and flushes pending maintenance before closing the database.

The optional live WebSocket observer requires Ratchet and a private bearer-token
file:

```sh
umask 077
openssl rand -hex 32 > ./observer.token
mosaic-mcp -db ./data/mosaic.hexxla -policy configs/config.yaml \
  -ratchet-observability-token-file ./observer.token
mosaic-observe -token-file ./observer.token
```

The default URL is `ws://127.0.0.1:8787/observability/stream`. Server
`-ratchet-observability-path` and observer `-url` override it. The stream rejects
browser Origins and accepts credentials only in the Authorization header. Keep
the file private (`0600` on Unix-like systems); restart after rotating it.

Events contain `id`, `type`, a process-local `session_id` correlation label,
optional `tool_name`, and `timestamp`. Raw MCP session IDs and internal event
error/details are excluded. Filter with `-session-id` using a label from a live
event. The observer receives immutable event values, retains no history, allows
at most 16 clients, and drops events when a client's bounded buffer is full.
