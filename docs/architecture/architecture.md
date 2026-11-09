# Mosaic architecture

Mosaic is a local MCP application. HexxlaDB owns the database format and transactions. Ratchet owns workflow prerequisites. Mosaic owns memory write policy, retrieval budgets, and the MCP tool contracts.

## Dependency direction

- `cmd/mosaic-mcp` composes startup configuration, services and adapters. It owns the server and database lifetimes.
- `internal/adapter/primary/mcpsrv` translates MCP requests, classifies tools and meters structured retrieval output. It adapts Ratchet to the transport session.
- `internal/core/services` owns application decisions. It receives write policy as an application value and composes the text embedder with vector search through separate ports.
- `internal/core/domain` owns application inputs, results and policy values. The primary and secondary ports refer to these types.
- `internal/adapter/secondary/hexxlastore` uses the public HexxlaDB API for database effects. Exact or near-anchor placement and the write share one transaction. Optional pruning runs after a committed delete.
- `internal/adapter/secondary/ollamaembed` adapts the Ollama client to the application's text embedding port.
- `internal/config` parses startup inputs and maps them into application settings. Core services depend on the domain and ports.

A put request travels through the MCP adapter and mutation service to the storage adapter. The mutation service applies the operator's write policy. Ratchet independently controls prerequisites before the handler runs. Database replacement is a separate offline operator procedure.

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
