# Project Instructions

## Architecture and security

- Follow the [architecture guide](docs/architecture/architecture.md) for changes under `internal/`. Keep the domain independent, dependencies pointing toward the core, and external I/O behind ports. Follow `.golangci.yml` and [SECURITY.md](SECURITY.md).
- Keep MCP transport and policy wiring at the adapter boundary. Preserve tool schemas, session isolation, retrieval budgets, and write gates when changing MCP behavior. See the [MCP transport and lifecycle contract](docs/architecture/architecture.md#mcp-transport-and-lifecycle) and [Ratchet integration](docs/mosaic/RATCHET_INTEGRATION.md) for their contracts.
- For Mosaic retrieval, choose a bounded search, inspect `retrieval_hint`, and load a context pack when the question needs neighbourhood or seam context. The [MCP agent blueprint](docs/mosaic/MCP_AGENT_BLUEPRINT.md) owns the full workflow. The [persistence policy](docs/mosaic/PERSISTENCE_POLICY.md) describes the host's role in capture and the server's write gates.

## Go and verification

- Honor the target module's `go.mod` and any file-specific build constraints. In Codex, use the [JetBrains Modern Go Guidelines plugin](https://github.com/JetBrains/go-modern-guidelines#codex) when available. Preserve tool and JSON contracts, match surrounding style, format Go changes, and preserve inspectable error causes with `%w`.
- Run the narrowest relevant checks first. Use `task test` for race tests, including the MCP integration cases, and `task ci` before pushing.
- See the [development tooling guide](docs/mosaic/DEVELOPMENT_TOOLING.md) for editor and local Codex MCP setup.

## Repository workflow

- Preserve unrelated work and keep commits focused.
- Track pending work in the [roadmap](docs/ROADMAP.md) and user-visible changes in [CHANGELOG.md](CHANGELOG.md).
- Keep repository skills under `.agents/skills/` and reserve new skills for recurring Mosaic-specific work.

## Release maturity

The repository and combined stack are alpha. Follow [VERSIONING.md](VERSIONING.md):
preserve published tags, keep versions independent, publish deliberate v0
prerelease milestones and require recorded beta/v1 qualification before promotion.
