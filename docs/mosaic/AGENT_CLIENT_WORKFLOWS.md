# Agent clients: rules, workflows, and Mosaic

Different editors expose **repeatable agent instructions** in different places. Mosaic ships **rules** and **Windsurf workflows** in-repo; use this page to pick the right mechanism for your client.

---

## Concepts

| Mechanism | Typical role |
| --------- | ------------ |
| **Rules** (instructions injected into context) | Always-on or scoped guidance: retrieval chaining, `put_cell` order, tag reuse. |
| **Workflows** (explicit step lists the user or agent runs) | Same content as a playbook, often invoked by **slash command** in tools that support it. |
| **MCP server instructions** | Mosaic reads YAML at startup (e.g. **`retention.notes`**) and includes it in MCP server metadata—good for operator policy text. |
| **Repo docs** | [`AGENTS.md`](../../AGENTS.md), [`MCP_AGENT_BLUEPRINT.md`](./MCP_AGENT_BLUEPRINT.md), this file—human-readable and shareable. |

**Nothing** in a client guarantees the model will call MCP tools; combine layers (see [README § Reliable tool use from agents](../../README.md#reliable-tool-use-from-agents)).

Mosaic does **not** currently ship a reusable `SKILL.md`. The committed rules and workflows guide tool selection and chaining; the runtime MCP `tools/list` schemas are the authoritative names, inputs, outputs, and descriptions. [`HEXXLA_API_SURFACE_COVERAGE.md`](./HEXXLA_API_SURFACE_COVERAGE.md) maps those tools to the underlying HexxlaDB API without duplicating every runtime schema.

---

## Common MCP-capable clients (summary)

Details change with product versions—verify in each product’s docs.

| Client | Rules / instructions | “Workflow”-style repetition | MCP config |
| ------ | -------------------- | ----------------------------- | ---------- |
| **Windsurf** | [`.windsurf/rules/`](../../.windsurf/rules/) Markdown | [`.windsurf/workflows/*.md`](../../.windsurf/workflows/) — Markdown + YAML frontmatter; Cascade discovers them under `.windsurf/workflows/` and related scopes; invoke with **`/<workflow-name>`** (derived from the filename, e.g. **`/mosaic-save-turns`**). Global templates: **`~/.codeium/windsurf/global_workflows/`**. Vendor overview: [Windsurf — Workflows](https://codeium.mintlify.app/plugins/cascade/workflows). | Windsurf MCP settings |
| **Cursor** | [`.cursor/rules/*.mdc`](../../.cursor/rules/) (and optional [`AGENTS.md`](../../AGENTS.md)) | No workspace `.cursor/workflows/` convention like Windsurf’s; combine **project rules**, **Plan mode**, Composer prompts, or an in-repo checklist (this doc). Official: [Rules](https://cursor.com/docs/context/rules). | `.cursor/mcp.json` (often gitignored locally—ship a template or document env setup) |
| **Claude Code** | Project instructions, `CLAUDE.md` | Slash commands and skills (team-dependent); MCP via CLI/config | `claude mcp` / project MCP entries |
| **VS Code + MCP** | Rules extensions / repo docs | Tasks, prompts, or Copilot instructions—no single standard | MCP extension host config |
| **Cline / OpenClaw / other** | Varies | Often custom prompts + MCP | Per-tool JSON |

Windsurf workflow files are limited to **~12k characters** each (per vendor docs).

---

## Mosaic artifacts in this repo

| Path | Purpose |
| ---- | ------- |
| [`.cursor/rules/mosaic-mcp-agent.mdc`](../../.cursor/rules/mosaic-mcp-agent.mdc) | Cursor: retrieval chain, turns, tags, budgets (`alwaysApply`). |
| [`.windsurf/rules/mosaic-turn-persistence.md`](../../.windsurf/rules/mosaic-turn-persistence.md) | Windsurf rules: short mandate for full-turn `put_cell`. |
| [`.windsurf/workflows/mosaic-save-turns.md`](../../.windsurf/workflows/mosaic-save-turns.md) | Windsurf Cascade: slash **`/mosaic-save-turns`** — step-by-step save turns. |
| [`configs/config.yaml`](../../configs/config.yaml) | Example **`retention`** / **`notes`** for MCP instructions. |

Copy or adapt these patterns into another client’s instruction format if your team standardizes elsewhere.

---

## See also

- [`MCP_AGENT_BLUEPRINT.md`](./MCP_AGENT_BLUEPRINT.md)
- [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md)
- [README — Reliable tool use from agents](../../README.md#reliable-tool-use-from-agents)
