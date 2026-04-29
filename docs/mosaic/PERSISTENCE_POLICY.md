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
- Set to **`true`** to allow `DeleteCell` through the MCP and [CellMutationService](internal/core/services/cell_mutation.go).

## Runtime gates (code)

[`config.MosaicRuntimeConfig`](../../internal/config/mosaic_runtime.go): **`AllowsPutCell`**, **`PutCellDenied`**, **`AllowsDeleteCell`**, **`DeleteCellDenied`**.

[`cell_mutation_tool.go`](../../internal/adapter/primary/mcpsrv/cell_mutation_tool.go) and [`CellMutationService`](../../internal/core/services/cell_mutation.go) use the same config.

## MCP

- **`mosaic_hexxla_get_persistence_policy`** — Returns JSON: **`retention`**, **`allow_delete_cell`**, **`config_file`**, **`is_default`**.

## Example file

| File | Purpose |
| ---- | ------- |
| [`configs/config.yaml`](../../configs/config.yaml) | Canonical — `retention` + `allow_delete_cell` for dev |

A **flat** layout ( `version` plus `capture_mode` / `enforcement` / `notes` at the top level, no `retention` key) is still valid; no separate example file is shipped.

## Cursor / IDE rules

Keep one canonical YAML; point rules at “call `mosaic_hexxla_get_persistence_policy`” or the repo file path to avoid drift.
