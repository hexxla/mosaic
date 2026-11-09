# Mosaic MCP boundary

Mosaic adapts the public HexxlaDB API to bounded memory operations. Its default
seven-tool surface covers discovery, exact lookup, context, cell writes,
deletions, tags and health. `advanced_tools: true` exposes all 22 tools.
[Capability mapping](HEXXLA_API_SURFACE_COVERAGE.md) and runtime `tools/list`
are the current API references.

## Request flow

HTTP/SDK parsing leads to Mosaic input validation, optional Ratchet ordering,
application write policy or retrieval budgeting, then typed secondary ports.
Storage adapters own `DB.View` and `DB.Update`; model calls happen through a
separate embedding port. Atomic cell-plus-vector writes compute the vector
before the storage transaction.

Core services do not import the SDK, Ollama client or HexxlaDB implementation.
Commands compose the concrete resources and own their lifetime.
[Architecture](../architecture/architecture.md)

## Transport and security

- Bind to a validated loopback address. General network authentication is outside
  this local deployment contract.
- Accept POST and DELETE. POST responses can use SSE; standalone GET streams
  return 405. Endpoints must be canonical literal paths.
- Apply Go cross-origin protection and SDK localhost Host validation.
- Expire idle HTTP sessions after 30 minutes. Session IDs stay transport-private;
  optional observer streams expose separate correlation IDs.
- Validate every tool against the explicit safety inventory. Enabling Ratchet
  requires coverage for every enabled mutation.
- Drain admitted SDK calls, including calls whose HTTP connection disconnected,
  before closing sessions and storage. Flush pending bounded pruning before DB close.
- Redact credential-bearing URLs and keep model response bodies bounded.

Ratchet wraps the final SDK response and snapshots its wire representation
before recording success. A failed call spends a selected one-time prerequisite
and issues no downstream completion. Ordering does not authorize a resource or
prove that a retrieval inspected a particular target.
[Workflow integration](RATCHET_INTEGRATION.md)

## Retrieval and budgets

`search_cells` has explicit lexical, pure-vector semantic, and filtered/composite
hybrid modes. Search is global unless a radius is supplied. Exact identities use
`get_cell`. Context expands seeds through `LoadContext` and applies one
`max_budget_bytes` ceiling, returning `total_bytes`.

Byte budgeting counts returned content, not the full JSON envelope or a model's
actual tokenizer. Optional session metering bounds serialized JSON egress.
[Agent workflow](MCP_AGENT_BLUEPRINT.md)

## Failure and retry

Preserve inspectable storage/model causes. Return partial search results with
explicit scan/candidate exhaustion flags. Missing exact identities are ordinary
`found: false` results. A committed delete and failed subsequent maintenance
must not be reported as though the delete rolled back.

A durable or uncertain mutation error can mean the write was saved. Reconcile
source/content and the returned coordinate before retrying, especially with
near-anchor placement. Database replacement, encryption, bulk transfer, history
pruning and physical compaction remain explicit operator procedures.
