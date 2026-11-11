# Mosaic

**Status: alpha.** This repository and the HexxlaDB–Mosaic–MCP Ratchet stack
are under active development. Use controlled evaluation with recoverable data;
general production sign-off has not been given. Existing tags remain intact.
[Release policy and beta/v1 gates](VERSIONING.md)

Mosaic is a local MCP server for persistent agent memory backed by HexxlaDB.
It stores cells on a hex lattice, discovers matching memories, and assembles
bounded neighbourhood context. You operate the database and optional Ollama
embedding service.

## Start

Use Go 1.27.4 or newer and run from the repository root:

```sh
export MOSAIC_DB_PATH="$PWD/.tmp/mosaic.hexxla"
go run ./cmd/mosaic-create-db -db "$MOSAIC_DB_PATH"
go run ./cmd/mosaic-mcp -db "$MOSAIC_DB_PATH"
```

Connect your MCP client to `http://127.0.0.1:8787/mcp`. The server opens an
existing primary; creation and replacement are explicit operator actions.
For a demo corpus, run `ollama pull all-minilm`, then use `task seed` on an
unused database path. [Creation, seeding and upgrades](docs/mosaic/DATABASE_CREATION.md)

Ollama is needed only for semantic/hybrid discovery or `embed: true` writes.
Its defaults are `http://127.0.0.1:11434` and model `all-minilm`. Lexical reads,
exact lookup, context loading, and ordinary writes work without it.
[Configuration](docs/mosaic/MOSAIC_CONFIG.md)

For encryption, inject `MOSAIC_DB_PASSPHRASE` when creating and opening the
file. [Credentials and recovery](docs/mosaic/DATABASE_CREATION.md#encrypted-database-passphrase-or-raw-key)

## Everyday tools

The default surface has seven tools. Names below omit the shared
`mosaic_hexxla_` prefix:

| Tool | Purpose |
| --- | --- |
| `search_cells` | Global-by-default lexical, semantic, or hybrid discovery |
| `get_cell` | Exact coordinate read, with explicit `found` status |
| `load_context_pack` | Expand known coordinates under a UTF-8 byte budget |
| `put_cell` | Exact or near-anchor placement; optional atomic embedding |
| `delete_cell` | Delete when operator policy permits |
| `list_tags` | Reuse the existing vocabulary |
| `health` | Inspect database layout; request a full check explicitly |

`search_cells` uses `mode: lexical` by default. Pure-vector `semantic` mode
accepts a query and result/score bounds. Use `hybrid` for filtered semantic
retrieval; its optional `lexical_query` adds an independent text predicate.
Scores are mode-specific. A positive `radius` opts into spatial scope; omission
searches globally. Inspect scan/candidate-limit flags before assuming completeness.

Use returned coordinates as context seeds, or call `get_cell` for an identity
you already know. Context accepts one `max_budget_bytes` value (default 4096,
maximum 100000); it counts raw cell and requested facet text in UTF-8 bytes.
The host remains responsible for the final model request's tokenizer accounting.

`put_cell` with `embed: true` computes a vector before opening the write
transaction and commits cell and vector together. Model failure saves nothing.
The returned coordinate is authoritative. Changing raw content without a new
embedding invalidates the old vector.

## Policy and specialist tools

The [sample policy](configs/config.yaml) uses selective capture and leaves
deletion and automatic maintenance disabled. Pass it with:

```sh
go run ./cmd/mosaic-mcp -policy configs/config.yaml -db "$MOSAIC_DB_PATH"
```

Mosaic does not automatically capture transcripts. The host must explicitly
submit the turns it is authorized to persist. Capture policy constrains those
writes; it is separate from database history retention.
[Persistence contract](docs/mosaic/PERSISTENCE_POLICY.md)

Set `ratchet: true` in the policy file to enable the bundled prerequisites.
Custom grouped workflows are optional. Ratchet enforces successful call order;
it does not prove that a query inspected the target or that a write chose
appropriate tags. [Workflow contract](docs/mosaic/RATCHET_INTEGRATION.md)

Set `advanced_tools: true` to expose all 43 tools: graph/spatial exploration,
retained-history and changefeed inspection, resumable regional export, atomic
batch capture, guarded facets, vector maintenance and diagnostics. Physical
maintenance, raw KV and encryption recovery remain operator workflows.
[Complete API mapping](docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md)

## Development

`task ci` runs the shared lint, race tests, command builds, and vulnerability
checks. The server supports POST and DELETE, including POST SSE responses;
standalone GET streams return 405. It binds to loopback and checks foreign
Origins. [Architecture](docs/architecture/architecture.md) ·
[Agent workflow](docs/mosaic/MCP_AGENT_BLUEPRINT.md) ·
[Development tooling](docs/mosaic/DEVELOPMENT_TOOLING.md)
