# Mosaic config: retention & gates (client-agnostic)

Mosaic does not auto-save chat. The config file describes **retention** intent for `put_cell` and **allow_delete_cell** for `mosaic_hexxla_delete_cell`, plus optional server enforcement.

Load the policy at startup and restart Mosaic after editing it. See the
[configuration reference](MOSAIC_CONFIG.md#loading) for file selection.

## YAML (`version: 1`)

```yaml
version: 1

retention:
  capture_mode: llm_curates   # default when omitted
  enforcement: false          # optional boolean: false = advisory; true = server rejects conflicting put_cell kinds
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

Only YAML boolean enforcement is accepted. The canonical `retention` block is
required for explicit retention settings; flat root keys and
`persistence_policy` aliases are rejected. Omitting retention uses
`llm_curates` with enforcement false.

### `allow_delete_cell`

- **Omitted** → **`false`**: `mosaic_hexxla_delete_cell` is rejected until the operator sets `allow_delete_cell: true`.
- Set to **`true`** to allow `DeleteCell` through the MCP and [CellMutationService](../../internal/core/services/cell_mutation.go).

## Runtime gates (code)

[`config.MosaicRuntimeConfig`](../../internal/config/mosaic_runtime.go) maps startup settings into the application mutation policy.

[`cell_mutation_tool.go`](../../internal/adapter/primary/mcpsrv/cell_mutation_tool.go) and [`CellMutationService`](../../internal/core/services/cell_mutation.go) use the same config.

## MCP

- With `advanced_tools: true`, **`mosaic_hexxla_get_persistence_policy`** — Returns JSON: **`retention`** (`capture_mode`, **`enforcement`** as boolean — `true` when the server blocks conflicting turn-related puts), **`allow_delete_cell`**, **`config_file`**, **`is_default`**.

## Example file

| File | Purpose |
| ---- | ------- |
| [`docs/mosaic/MOSAIC_CONFIG.md`](./MOSAIC_CONFIG.md) | Full **`version: 1`** policy YAML reference (all keys) |
| [`configs/config.yaml`](../../configs/config.yaml) | Minimal committed example (copy or adapt) |

## Agent and IDE instructions

Keep one canonical YAML. The server supplies active retention and deletion settings in initialize instructions; advanced profiles also expose the policy-inspection tool. The shipped configuration selects `llm_curates`, disables deletion and automatic pruning, and enables bundled Ratchet prerequisites.
