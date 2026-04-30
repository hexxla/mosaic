# Mosaic policy YAML (`version: 1`)

Reference for the Mosaic MCP server configuration file. Policy affects **`mosaic-mcp`** only (create-db/seed use separate CLI flags unless noted).

**Example (minimal):** [`configs/config.yaml`](../../configs/config.yaml)

**Related:** [Persistence policy semantics](./PERSISTENCE_POLICY.md) · [Database path & encryption](./DATABASE_CREATION.md) · [HEXXLA_TROUBLESHOOTING](./HEXXLA_TROUBLESHOOTING.md) (disk, deletes, MVCC)

---

## Loading

| Mechanism | Notes |
| --------- | ----- |
| **`mosaic-mcp -policy PATH`** | Explicit path |
| **`MOSAIC_POLICY_FILE`** | Env var when `-policy` is not set |
| **`mosaic-mcp` without policy** | Built-in defaults ([`DefaultMosaicConfig`](../../internal/config/mosaic_yaml.go)); no file required |

Restart **`mosaic-mcp`** after editing YAML.

---

## Top-level keys

| Key | Required | Purpose |
| --- | -------- | ------- |
| **`version`** | Yes | Must be **`1`**. |
| **`retention`** | No | Which chat turn kinds may be stored and whether the server **enforces** capture rules. |
| **`allow_delete_cell`** | No | Default **`false`**. When **`false`**, **`mosaic_hexxla_delete_cell`** errors. |
| **`retrieval`** | No | Optional cumulative session read budget (metering / cap). |
| **`ollama`** | No | Optional **`base_url`** and **`embed_model`** — override **`MOSAIC_OLLAMA_URL`** / **`MOSAIC_EMBED_MODEL`** when set (see precedence below). |
| **`database`** | No | Optional passphrase hint, MVCC retention, post-delete prune/compact. |

Legacy/alternate shapes (`persistence_policy`, flat `capture_mode` at root) are accepted by the parser; prefer the **`retention`** block.

---

## `retention`

| Field | Type | Default / notes |
| ----- | ---- | --------------- |
| **`capture_mode`** | string | **`llm_curates`** if omitted. Values: **`llm_curates`**, **`save_all_turns`**, **`user_only`**, **`assistant_only`**, **`none`**. See [PERSISTENCE_POLICY](./PERSISTENCE_POLICY.md). |
| **`enforcement`** | boolean | **`false`** if omitted. **`true`**: conflicting **`put_cell`** kinds return an error. Legacy **`off`** / **`reject`** strings still parse. |
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
| **`bytes_per_approx_token`** | float | **`4`**; must be **2–16** if set |

Tools **`estimate_context_budget`**, **`mosaic_hexxla_get_persistence_policy`**, and writes are outside this budget.

---

## `ollama` (optional)

HTTP root and embedding model for **Ollama** (hybrid query/search, **`mosaic_hexxla_search_embedding`**, text path on **`mosaic_hexxla_put_embedding`**, **`mosaic-seed`** when it embeds).

| Field | Type | Default / notes |
| ----- | ---- | ----------------- |
| **`base_url`** | string | Omit to fall back to **`MOSAIC_OLLAMA_URL`**, then **`http://127.0.0.1:11434`**. |
| **`embed_model`** | string | Omit to fall back to **`MOSAIC_EMBED_MODEL`**, then **`all-minilm`**. |

**Precedence** (URL): **`mosaic-seed -ollama`** (flag) → YAML **`ollama.base_url`** → **`MOSAIC_OLLAMA_URL`** → default. Same for model: **`-embed-model`** → **`ollama.embed_model`** → **`MOSAIC_EMBED_MODEL`** → default.

**`mosaic-mcp`** uses YAML → env → default (no Ollama CLI flags). The [Makefile](../../Makefile) commonly sets **`MOSAIC_OLLAMA_*`**; **`ollama:`** in policy YAML overrides those when the corresponding YAML field is non-empty.

---

## `database` (optional)

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

Runs bounded **`PruneScheduler.Tick`** and/or **`(*DB).Compact`** after a successful cell delete (**`cell_removed: true`**), then swaps the live DB handle ([`LiveDB`](../../internal/adapter/secondary/hexxlastore/livedb.go)).

| Field | Type | Notes |
| ----- | ---- | ----- |
| **`enabled`** | bool | Master switch |
| **`prune`** | bool | Default **`true`** when **`enabled`** |
| **`compact`** | bool | Default **`true`**; exclusive rewrite, briefly blocks readers |
| **`debounce_after_delete_ms`** | int | Milliseconds. **`0`** = run maintain immediately after **each** delete. **Omitted** when **`enabled: true`** defaults to **2000** ms — coalesces rapid deletes into one burst |
| **`prune_profile`** | string | **`balanced`** (default), **`low-latency`**, **`long-history`** → Hexxla **`MVCCPruneProfile`** batch sizes |
| **`max_prune_rounds_per_delete`** | int | Cap on prune **`Tick`** iterations per **maintenance burst** (not per delete when debounced); default **64** if omitted or invalid |

Shutdown **`mosaic-mcp`** flushes pending debounced maintenance before closing the DB.

See [CHANGELOG](../../CHANGELOG.md) and [HEXXLA_TROUBLESHOOTING](./HEXXLA_TROUBLESHOOTING.md) for disk expectations (extend-only file, MVCC tombstones).
