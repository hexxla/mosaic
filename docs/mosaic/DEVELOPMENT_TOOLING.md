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

Start Mosaic using the [server quick start](../../README.md#start), then register its local Streamable HTTP endpoint:

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


## CI and version tags

`task ci` runs the validation used on pull requests and pushes to `main`.
It checks module metadata and workflows, lints, runs race tests, builds the
commands, and scans for known Go vulnerabilities. GitHub Actions also scans
the commit range for secrets.

Run `scripts/tool.sh golangci-lint run --enable modernize ./...` for an
optional modernization pass.

Tag a commit from `main` after CI passes. Build new versions from the tagged
source using the [README instructions](../../README.md#start).
Published archives for [v0.3.0](https://github.com/hexxla/mosaic/releases/tag/v0.3.0)
remain available.

## Cross-repository checks

Mosaic CI uses the versions in `go.mod`. To verify coordinated source changes
before publishing dependencies, create a temporary Go workspace outside the
repositories.

```sh
workspace_dir=$(mktemp -d)
(cd "$workspace_dir" && go work init /path/to/mosaic /path/to/mcp-ratchet /path/to/hexxladb)
GOWORK="$workspace_dir/go.work" go test -race ./internal/adapter/primary/mcpsrv
```

Run the test command from the Mosaic root. Once the Ratchet workflow release is
available, update Mosaic's pin and run `task ci` with `GOWORK=off`.
