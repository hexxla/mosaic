# Mosaic config: retention & gates (client-agnostic)

Mosaic does not auto-save chat. The config file describes **retention** intent for `put_cell` and **allow_delete_cell** for `mosaic_hexxla_delete_cell`, plus optional server enforcement.

## Loading

| Mechanism | Precedence |
| --------- | ---------- |
| **`mosaic-mcp -policy PATH`** | Highest |
| **`MOSAIC_POLICY_FILE`** | If `-policy` is omitted |
| **Built-in default** | [`DefaultMosaicConfig`](../../internal/config/mosaic_yaml.go) |

The file is read **once at process start**. Restart the server after edits.

## YAML (`version: 1`)

```yaml
version: 1

retention:
  capture_mode: llm_curates   # required when specifying retention explicitly in section form
  enforcement: false          # optional boolean: false = advisory; true = server rejects conflicting put_cell kinds (legacy: off | reject)
  notes: "optional operator note"

allow_delete_cell: false       # optional; default false — disables mosaic_hexxla_delete_cell when false

# optional — encrypted DB passphrase (prefer env / flag over committing YAML)
# database:
#   passphrase: "use MOSAIC_DB_PASSPHRASE or mosaic-mcp -db-passphrase instead when possible"
```

### `retention`

Owns **capture_mode**, **enforcement**, and **notes** for turn-related writes (`user_message` / `assistant_response`) vs facts.

| `capture_mode` | Meaning |
| -------------- | ------- |
| `llm_curates` | Model chooses what to store (default when retention omitted). |
| `save_all_turns` | Both roles in scope for persistence. |
| `user_only` / `assistant_only` | Only that side’s turn kind in scope. |
| `none` | No chat turns (when enforcement is true, those kinds are blocked). |

| `enforcement` | Meaning |
| ------------- | ------- |
| `false` (or omitted) | Advisory only; server does not block puts by capture mode. |
| `true` | Server returns an error on disallowed `put_cell` kinds for turn-related kinds. |

Strings **`off`** / **`reject`** are still accepted for backward compatibility.

If the **`retention`** key is **omitted** entirely, defaults match [`DefaultRetentionPolicy`](../../internal/config/retention.go) (`llm_curates`, enforcement **false** / `"off"` in JSON). You may also use a **flat** legacy file with `version`, `capture_mode`, `enforcement`, `notes` at the top level (no `retention` key).

**Deprecated:** `persistence_policy:` — same shape as `retention:`. Do not set both `retention` and `persistence_policy` with a `capture_mode` in the same file.

### `allow_delete_cell`

- **Omitted** → **`false`**: `mosaic_hexxla_delete_cell` is rejected until the operator sets `allow_delete_cell: true`.
- Set to **`true`** to allow `DeleteCell` through the MCP and [CellMutationService](../../internal/core/services/cell_mutation.go).

### Encrypted HexxlaDB files (`database`)

New encrypted HexxlaDB v0.6.0 files use authenticated format v3 with XChaCha20-Poly1305 pages, an authenticated header, and keyed WAL records. Existing AES-256-XTS v1/v2 files remain readable through HexxlaDB's legacy compatibility path. Passphrases use Argon2id; raw keys are expanded with HKDF-SHA256.

| Mechanism | Role |
| --------- | ---- |
| **`mosaic-mcp`** | `-db-passphrase` → `MOSAIC_DB_PASSPHRASE` → policy YAML `database.passphrase`; raw `MOSAIC_DB_ENCRYPTION_KEY_HEX` is mutually exclusive. |
| **`mosaic-create-db`** | Same flag → env → policy YAML precedence; prefer env over command arguments or committed YAML. |
| **`mosaic-seed`** | `-db-passphrase` → `MOSAIC_DB_PASSPHRASE`; the policy file configures Ollama only and its `database.passphrase` is currently ignored. The raw-key environment variable remains mutually exclusive. |

If the database file is encrypted and no credential is provided, open fails (see upstream **`ErrEncryptionKeyRequired`** / mismatch errors).

### Creating a database file (layout + encryption)

HexxlaDB **`Open`** creates the file when it does not exist. Mosaic wraps that in:

| Command | Purpose |
| ------- | ------- |
| **`cmd/mosaic-create-db`** | Empty DB only — builds **[`config.NewMosaicDatabaseOptions`](../../internal/config/mosaic_hexxla_db.go)(layout)**, then **[`config.ApplyHexxlaEncryption`](../../internal/config/hexxla_open.go)**, then **`hexxladb.Open`**. |
| **`cmd/mosaic-seed`** | Same layout + optional encryption, then seeds cells/embeddings if the corpus is non-empty. |

**Encryption** is never implicit: credentials come only from the command-specific mechanisms in the table above (`ApplyHexxlaEncryption` / `BuildHexxlaOpenOptions`). **Layout** (MVCC, page size, max value bytes, embedding dimension, distance metric) is optional on both commands—each flag defaults to the Mosaic values in **`DefaultMosaicDatabaseLayout`**. Plaintext MVCC creation uses format v2; supplying official encryption credentials creates authenticated format v3 regardless of the MVCC flag.

**CLI and Task examples:** [DATABASE_CREATION.md](./DATABASE_CREATION.md).

## Runtime gates (code)

[`config.MosaicRuntimeConfig`](../../internal/config/mosaic_runtime.go): **`AllowsPutCell`**, **`PutCellDenied`**, **`AllowsDeleteCell`**, **`DeleteCellDenied`**.

[`cell_mutation_tool.go`](../../internal/adapter/primary/mcpsrv/cell_mutation_tool.go) and [`CellMutationService`](../../internal/core/services/cell_mutation.go) use the same config.

## MCP

- **`mosaic_hexxla_get_persistence_policy`** — Returns JSON: **`retention`** (`capture_mode`, **`enforcement`** as boolean — `true` when the server blocks conflicting turn-related puts), **`allow_delete_cell`**, **`config_file`**, **`is_default`**.

## Example file

| File | Purpose |
| ---- | ------- |
| [`docs/mosaic/MOSAIC_CONFIG.md`](./MOSAIC_CONFIG.md) | Full **`version: 1`** policy YAML reference (all keys) |
| [`configs/config.yaml`](../../configs/config.yaml) | Minimal committed example (copy or adapt) |

A **flat** layout ( `version` plus `capture_mode` / `enforcement` / `notes` at the top level, no `retention` key) is still valid; no separate example file is shipped.

## Agent and IDE instructions

Keep one canonical YAML; point project instructions at “call `mosaic_hexxla_get_persistence_policy`” or the repository file path to avoid drift.
