# Development tooling

Mosaic keeps shared engineering instructions and development commands in the repository. Editor and MCP client settings are local.

## Sources of truth

| Path | Responsibility |
| --- | --- |
| [`AGENTS.md`](../../AGENTS.md) | Architecture, security, Go, testing, documentation, and repository workflow agreements for humans and agents. |
| [`MCP_AGENT_BLUEPRINT.md`](./MCP_AGENT_BLUEPRINT.md) | Mosaic-specific retrieval, context assembly, persistence, and client guidance. |
| [JetBrains Modern Go Guidelines](https://github.com/JetBrains/go-modern-guidelines#codex) | Optional Codex plugin for version-specific Go idioms. |
| Runtime MCP `tools/list` | Authoritative tool names, input schemas, output schemas, and descriptions. |

MCP client connection settings remain local because configuration locations and schemas differ between hosts. Point the chosen client at Mosaic's Streamable HTTP URL, normally `http://127.0.0.1:8787/mcp`.

## Codex MCP setup

Start Mosaic using the [server quick start](../../README.md#step-5--run-the-mcp-server), then register its local Streamable HTTP endpoint:

```bash
codex mcp add mosaic --url http://127.0.0.1:8787/mcp
codex mcp list
```

Start a new Codex session and use `/mcp` to confirm Mosaic is connected.
For a read-only smoke check, call `mosaic_hexxla_get_persistence_policy` and
confirm the active policy matches the intended deployment.
In the desktop app, the same endpoint can be added under Settings → MCP servers.
See [Codex MCP configuration](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)
for project-scoped settings and tool allowlists.

## Editors

Use any editor with the shared formatting conventions in [`.editorconfig`](../../.editorconfig).
Run `task build`, `task test`, or `task ci` from the repository root.
See [`Taskfile.yml`](../../Taskfile.yml) for the full command list.

## Agent policy versus runtime policy

Repository instructions guide development. Mosaic runtime policy is separate: `retention.notes` from the operator YAML is injected into MCP server instructions, while Ratchet can enforce configured tool prerequisites. Neither mechanism makes an MCP host call tools automatically.

For the recommended tool sequence, see [`MCP_AGENT_BLUEPRINT.md`](./MCP_AGENT_BLUEPRINT.md). For configuration enforcement, see [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md) and [`RATCHET_INTEGRATION.md`](./RATCHET_INTEGRATION.md).


## Releases

The [Release workflow](../../.github/workflows/release.yml) validates the exact
tagged commit and rehearses packaging before the publishing job can run.
The archives contain `mosaic-mcp`, `mosaic-create-db`, and `mosaic-observe`.
Run the workflow manually on a candidate reachable from `main` to download
the rehearsal archives and checksums.
