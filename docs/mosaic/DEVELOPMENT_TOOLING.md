# Development tooling

Mosaic keeps shared engineering instructions and editor automation small and repository-native. It does not generate per-client rule trees, hooks, slash-command workflows, or local MCP configuration.

## Sources of truth

| Path | Responsibility |
| --- | --- |
| [`AGENTS.md`](../../AGENTS.md) | Architecture, security, Go, testing, documentation, and repository workflow agreements for humans and agents. |
| [`MCP_AGENT_BLUEPRINT.md`](./MCP_AGENT_BLUEPRINT.md) | Mosaic-specific retrieval, context assembly, persistence, and client guidance. |
| [`.agents/skills/use-modern-go/SKILL.md`](../../.agents/skills/use-modern-go/SKILL.md) | Reusable Codex guidance keyed to the Go version declared in `go.mod`. |
| Runtime MCP `tools/list` | Authoritative tool names, input schemas, output schemas, and descriptions. |

MCP client connection settings remain local because configuration locations and schemas differ between hosts. Point the chosen client at Mosaic's Streamable HTTP URL, normally `http://127.0.0.1:8787/mcp`.

## VS Code

Tracked files under [`.vscode/`](../../.vscode/) provide recommended Go, YAML, ShellCheck, and Markdown extensions; format/import settings; and tasks for CI, tests, builds, and integration tests. VS Code is configured to consume `AGENTS.md` rather than a duplicated editor-specific ruleset.

## Zed

Tracked files under [`.zed/`](../../.zed/) configure Go formatting and `gopls`, exclude generated or temporary trees from scanning, and expose the same Mosaic CI, test, build, and integration commands.

## Agent policy versus runtime policy

Repository instructions guide development. Mosaic runtime policy is separate: `retention.notes` from the operator YAML is injected into MCP server instructions, while Ratchet can enforce configured tool prerequisites. Neither mechanism makes an MCP host call tools automatically.

For the recommended tool sequence, see [`MCP_AGENT_BLUEPRINT.md`](./MCP_AGENT_BLUEPRINT.md). For configuration enforcement, see [`PERSISTENCE_POLICY.md`](./PERSISTENCE_POLICY.md) and [`RATCHET_INTEGRATION.md`](./RATCHET_INTEGRATION.md).
