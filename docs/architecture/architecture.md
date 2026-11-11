# Mosaic architecture

Mosaic is a local MCP application. HexxlaDB owns the database format and transactions. Ratchet owns workflow prerequisites. Mosaic owns memory write policy, retrieval budgets, and the MCP tool contracts.

## Dependency direction

- `cmd/mosaic-mcp` composes startup configuration, services and adapters. It owns the server and database lifetimes.
- `internal/adapter/primary/mcpsrv` translates MCP requests, classifies tools and meters structured retrieval output. It adapts Ratchet to the transport session.
- `internal/core/services` owns application decisions. It receives write policy as an application value and composes the text embedder with vector search through separate ports.
- `internal/core/domain` owns application inputs, results and policy values. It is independent of other Mosaic layers. Primary ports describe the use cases called by entry points. Secondary ports describe the storage and model operations the core needs.
- `internal/adapter/secondary/hexxlastore` uses the public HexxlaDB API for database effects. Exact or near-anchor placement and the write share one transaction. Optional pruning runs after a committed delete.
- `internal/adapter/secondary/ollamaembed` adapts the Ollama client to the application's text embedding port.
- `internal/config` parses startup inputs and maps them into application settings. Core services depend on the domain and ports.

A put request travels from SDK parsing through the MCP adapter and mutation service to the storage adapter. Ratchet checks prerequisites before the handler runs. The mutation service validates the arguments and applies the operator's write policy. Storage adapters own `DB.View` and `DB.Update` and translate database errors. [Database operations](../mosaic/DATABASE_CREATION.md)

Run `task ci` for import boundaries and tests. Exercise MCP schemas, cross-session enforcement, atomic cell placement and policy rejections when their owners change.

## Current API boundary

Mosaic publishes one global-by-default discovery operation with lexical,
pure-vector semantic, and filtered/composite hybrid modes. Exact coordinate
lookup is separate. Model effects precede atomic cell-and-vector transactions;
placement, overwrite checks and both writes share the adapter's `DB.Update`.

The default MCP inventory contains seven everyday tools. Specialist tools are
registered only when `advanced_tools` is enabled. The enabled inventory drives
Ratchet mutation coverage. Mosaic selects configured calls and delegates their
execution/result serialization to shared Ratchet middleware. A bundled grouped
policy is enabled with `ratchet: true`; custom YAML remains an optional adapter.

Context assembly owns one canonical UTF-8 byte budget. Approximate-token
conversion is an advanced convenience; legacy token-named byte aliases and
parallel query APIs are removed.

## MCP transport and lifecycle

- The server binds to a validated loopback address and uses Go cross-origin protection and SDK localhost Host validation. Endpoints must be canonical literal paths. General network authentication is outside the local deployment contract.
- The server accepts POST and DELETE. POST responses can use SSE. Standalone GET streams return 405.
- Idle HTTP sessions expire after 30 minutes. The optional observer exposes separate correlation labels for transport sessions. [Ratchet integration](../mosaic/RATCHET_INTEGRATION.md)
- Shutdown drains admitted SDK calls, including calls whose HTTP connections have closed, before closing sessions and storage. The server flushes pending pruning before closing the database.
- Ollama diagnostics redact URL credentials and the client bounds response bodies. Configuration loading and environment variables are documented in the [configuration reference](../mosaic/MOSAIC_CONFIG.md).

The [capability matrix](../mosaic/HEXXLA_API_SURFACE_COVERAGE.md) maps MCP tools to the database API. The [agent guide](../mosaic/MCP_AGENT_BLUEPRINT.md) owns retrieval and retry behavior, including partial results and commit outcomes.
