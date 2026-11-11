# Mosaic policy YAML (`version: 1`)

Reference for the Mosaic MCP server configuration file. Policy affects **`mosaic-mcp`** only (create-db/seed use separate CLI flags unless noted).

**Example (minimal):** [`configs/config.yaml`](../../configs/config.yaml)

**Related:** [Persistence policy semantics](./PERSISTENCE_POLICY.md) · [Database path & encryption](./DATABASE_CREATION.md) · [Database diagnostics](./DATABASE_CREATION.md#diagnose-deletes-and-disk-usage)

---

## Loading

| Mechanism | Notes |
| --------- | ----- |
| **`mosaic-mcp -policy PATH`** | Explicit path |
| **`MOSAIC_POLICY_FILE`** | Env var when `-policy` is not set |
| **`mosaic-mcp` without policy** | Built-in defaults ([`DefaultMosaicConfig`](../../internal/config/mosaic_yaml.go)); no file required |

Restart **`mosaic-mcp`** after editing YAML.

## MCP listener

`MOSAIC_MCP_ADDR` defaults to `127.0.0.1:8787`. The listener accepts only
loopback addresses. `MOSAIC_MCP_PATH` defaults to `/mcp` and must be a
canonical literal HTTP path. [Transport and lifecycle](../architecture/architecture.md#mcp-transport-and-lifecycle)

## Policy validation

Files must contain exactly one YAML document. Unknown fields are rejected,
including misspelled retention and encryption settings. Use only the canonical
retention block with boolean enforcement; legacy aliases and string enforcement
are rejected.

---

## Top-level keys

| Key | Required | Purpose |
| --- | -------- | ------- |
| **`version`** | Yes | Must be **`1`**. |
| **`ratchet`** | No | Default **`false`**. **`true`** enables the embedded workflow policy; a custom Ratchet path also enables it. |
| **`advanced_tools`** | No | Default **`false`**. Enables the additional low-level operations and diagnostic tools. |
| **`retention`** | No | Which chat turn kinds may be stored and whether the server **enforces** capture rules. |
| **`allow_delete_cell`** | No | Default **`false`**. When **`false`**, **`mosaic_hexxla_delete_cell`** errors. |
| **`retrieval`** | No | Optional cumulative session read budget (metering / cap). |
| **`ollama`** | No | Optional **`base_url`**, **`embed_model`** and **`expected_digest`** — override **`MOSAIC_OLLAMA_URL`** / **`MOSAIC_EMBED_MODEL`** when set (see precedence below). |
| **`database`** | No | Optional credentials, cache/durability settings, MVCC retention and post-delete pruning. |

The default seven tools are health, search_cells, get_cell, load_context_pack,
put_cell, delete_cell and list_tags, with the `mosaic_hexxla_` prefix. Advanced
tools expand the inventory to 43, adding graph/spatial exploration, retained
history, changefeed inspection, regional export, atomic batches and vector
maintenance. See
[Ratchet integration](./RATCHET_INTEGRATION.md) for activation and mutation coverage.

---

## `retention`

| Field | Type | Default / notes |
| ----- | ---- | --------------- |
| **`capture_mode`** | string | **`llm_curates`** if omitted. Values: **`llm_curates`**, **`save_all_turns`**, **`user_only`**, **`assistant_only`**, **`none`**. See [PERSISTENCE_POLICY](./PERSISTENCE_POLICY.md). |
| **`enforcement`** | boolean | **`false`** if omitted. **`true`**: conflicting **`put_cell`** kinds return an error. Only YAML booleans are accepted. |
| **`notes`** | string | Injected into MCP server instructions and **`mosaic_hexxla_get_persistence_policy`**. Operator-facing guidance for agents. |

---

## `allow_delete_cell`

Boolean. Default **`false`**. Set **`true`** when **`mosaic_hexxla_delete_cell`** should be allowed (e.g. local development or governed production workflows).

---

## `retrieval` (optional)

Cumulative approximate-token metering on structured JSON outputs from HexxlaDB **read** tools (same MCP session). Not a relevance filter — use per-call rings and context-pack budgets for shaping retrieval ([MCP_AGENT_BLUEPRINT](./MCP_AGENT_BLUEPRINT.md)).

| Field | Type | Default |
| ----- | ---- | ------- |
| **`session_approx_token_budget`** | int | **`0`** = no enforcement cap; metering still recorded where applicable |
| **`bytes_per_approx_token`** | float | **`4`**; **`0`** selects the default; otherwise finite **2–16** |

Mosaic discards usage when the MCP session closes. Idle HTTP sessions close
after 30 minutes. Context packs use the remaining budget to trim content
before Mosaic checks the cost of the complete serialized response.

Advanced tools **`mosaic_hexxla_estimate_context_budget_bytes`**, **`mosaic_hexxla_get_persistence_policy`**, and writes are outside this budget.

Context packs accept only `max_budget_bytes`: omitted or zero selects 4096;
explicit limits must be 64–100000 UTF-8 bytes. Results expose `total_bytes`
and `max_budget_bytes`. Approximate-token input aliases and legacy token-named
outputs have been removed. `max_ring` is capped at 32 and `max_cells` at 8192.
The advanced byte-estimate tool remains available for token-to-byte planning.

---

## `ollama` (optional)

HTTP root and embedding model for **Ollama** (semantic/hybrid search, atomic cell embeddings, advanced embedding writes, and **`mosaic-seed`** when it embeds).

| Field | Type | Default / notes |
| ----- | ---- | ----------------- |
| **`base_url`** | string | Omit to fall back to **`MOSAIC_OLLAMA_URL`**, then **`http://127.0.0.1:11434`**. |
| **`embed_model`** | string | Omit to fall back to **`MOSAIC_EMBED_MODEL`**, then **`all-minilm`**. |
| **`expected_digest`** | string | Optional 64-hex SHA256 (optional `sha256:` prefix), falling back to `MOSAIC_EMBED_DIGEST`. Pins provider model content. |

**Precedence** (URL): **`mosaic-seed -ollama`** (flag) → YAML **`ollama.base_url`** → **`MOSAIC_OLLAMA_URL`** → default. Same for model: **`-embed-model`** → **`ollama.embed_model`** → **`MOSAIC_EMBED_MODEL`** → default.

**`mosaic-mcp`** uses YAML → env → default (no Ollama CLI flags). The [Taskfile](../../Taskfile.yml) supplies **`MOSAIC_OLLAMA_*`** defaults to its seed tasks; **`ollama:`** in policy YAML overrides those when the corresponding YAML field is non-empty.

The base must use HTTP or HTTPS with a host. Path prefixes and URL userinfo
are supported; query strings and fragments are rejected. Diagnostics omit
userinfo, queries, and fragments. Embedding responses are limited to 4 MiB;
error-body diagnostics are limited to 4 KiB. Components must fit finite float32.

---

The recorded identity is the configured model name, optionally followed by
`@sha256:<digest>`. Adding, removing or changing a digest on an existing vector
store requires `mosaic-mcp -reembed` just like a model-name change. When pinned,
Mosaic checks Ollama's [model-list digest](https://docs.ollama.com/api/tags)
before and after each inference and rejects a changed result before a write.
This trusts the provider's metadata; unpinned mutable names cannot detect a
same-name retag. Raw query/write vectors must declare the complete recorded
identity through `embedding_model`; declaration cannot verify how an external
vector was computed.

Use `mosaic-mcp -lexical-only` to serve a native database without adopting
unknown vector provenance. This disables semantic/hybrid queries, embedding
generation/writes and vector mutation/rebuild; lexical/history reads and ordinary
cell operations remain available. It neither adopts nor changes model identity.
It cannot be combined with `-reembed`.

## `database` (optional)

### Runtime engine settings

| Field | Type | Default / behavior |
| --- | --- | --- |
| `page_cache_bytes` | int | `0`: native 4 MiB default; `-1` disables caching; positive byte budget up to 1 GiB |
| `use_primary_fdatasync` | bool | `false`; native platform-specific durability option; review HexxlaDB durability guidance before enabling |
| `changelog_lazy` | bool | `false`; when a feed is enabled, defer feed fsync while durable primary outbox intents protect delivery; a crash can redeliver events |

Enable/select the feed with `-changefeed` / `-changefeed-path`. Existing default
sidecars continue automatically. These are runtime options, not persisted
layout overrides. Changefeed inspection never acknowledges a consumer.

### Passphrase (encryption)

Optional **`passphrase`** under **`database:`** — human-readable secret for encrypted HexxlaDB files.

**Precedence** for passphrase (single winner): **`mosaic-mcp -db-passphrase`** → **`MOSAIC_DB_PASSPHRASE`** → YAML **`database.passphrase`**.

Alternatively **`MOSAIC_DB_ENCRYPTION_KEY_HEX`** (raw key); do not combine with passphrase sources.

Prefer env or flags over committing secrets in Git. Plain (non-encrypted) DBs: omit passphrase everywhere.

Full precedence and rotation notes: [DATABASE_CREATION](./DATABASE_CREATION.md), [PERSISTENCE_POLICY](./PERSISTENCE_POLICY.md).

### `mvcc_retain_commits_behind_head`

Unsigned integer forwarded into **`hexxladb.Options.MVCCRetention`** at open. Bounds how far **`ViewAt`** can see behind head after pruning; drives **`SuggestedPruneBeforeSeq`**.

Keep this **below** typical peak **`CommitSeq`** for your workload: while **`CommitSeq ≤ retain`**, suggested automatic prune watermarks can be ineffective (see HexxlaDB **`docs/hexxladb/OPERATIONS.md`**). When **`database.auto_maintain_after_cell_delete`** enables prune but YAML omits **`mvcc_retain_commits_behind_head`**, Mosaic injects a small default ([`DefaultAutoMaintainRetainCommits`](../../internal/config/delete_auto_maintain.go)).

### `auto_maintain_after_cell_delete`

Runs bounded **`PruneScheduler.Tick`** after a successful cell delete. A prune failure is logged separately from the committed delete result. Stop Mosaic and follow the [HexxlaDB operations guide](https://github.com/hexxla/hexxladb/blob/main/docs/hexxladb/OPERATIONS.md) for offline compaction and validated replacement.

| Field | Type | Notes |
| ----- | ---- | ----- |
| **`enabled`** | bool | Master switch |
| **`prune`** | bool | Default **`true`** when **`enabled`** |
| **`debounce_after_delete_ms`** | int | Milliseconds. **`0`** = run pruning immediately after **each** delete. **Omitted** when **`enabled: true`** defaults to **2000** ms — coalesces rapid deletes into one burst |
| **`prune_profile`** | string | **`balanced`** (default), **`low-latency`**, **`long-history`** → Hexxla **`MVCCPruneProfile`** batch sizes |
| **`max_prune_rounds_per_delete`** | int | Cap on prune **`Tick`** iterations per **maintenance burst** (not per delete when debounced); default **64** if omitted or invalid |

The removed `compact` field is rejected even when false; compaction remains an
offline operation. The shipped configuration leaves automatic pruning disabled.

Shutdown **`mosaic-mcp`** drains active SDK calls, including disconnected POSTs, closes sessions, and flushes
pending debounced maintenance before closing the DB. Prune rounds are bounded
by count; an individual storage operation is synchronous and has no hard deadline.

See [database diagnostics](./DATABASE_CREATION.md#diagnose-deletes-and-disk-usage) for disk usage and MVCC tombstones.
