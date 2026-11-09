# Database API ownership

Mosaic uses the public `github.com/hexxla/hexxladb` root package. Database
transactions, record encoding, storage formats and retrieval algorithms belong
to HexxlaDB. Mosaic owns application write policy, model calls, result bounds,
UTF-8 budgeting and MCP schemas. No database implementation types cross inward
through a Mosaic port.

## Retrieval

`mosaic_hexxla_search_cells` is the sole discovery tool:

- `lexical` translates filters into `Tx.QueryCellsWithStats` without vectors.
- `semantic` embeds the query and uses context-aware `Tx.SearchByEmbedding`,
  retaining pure vector scores. Its supported controls are result and score bounds.
- `hybrid` embeds `query` and uses `QueryCellsWithStats` with structured filters
  and optional independent `lexical_query`. Its scores include native confidence
  and optional lexical contributions; they are not pure vector scores.

Discovery is global unless a positive spatial radius is requested. Lexical
scans default to 20,000 rows; adaptive hybrid search retains native candidate
bounds and reports exhaustion. A supported filter plus positive row cap enables
bounded exact hybrid scoring. Partial hits remain visible with their status.

`get_cell` uses `Tx.GetCell` directly. Missing and tombstoned identities return
`found: false`, with no fabricated empty record.

## Writes and context

Exact or near-anchor placement and `PutCell` share one `DB.Update`. When
`embed: true`, Mosaic computes the vector before that transaction and writes
both records within it. A failed vector write aborts the cell write. Independent
advanced embedding writes can compare `expected_content_sha256` atomically.

`Tx.LoadContext` owns candidate selection. Mosaic applies one
`max_budget_bytes` limit and returns `total_bytes`. It evicts outer-ring,
low-confidence cells while preserving deterministic tie order and explanations.
The byte limit does not represent full JSON egress or model tokenizer accounting.

## Boundaries

Secondary adapters own snapshots and transactions. Services depend on Mosaic
ports, including separate model and vector-search effects. Commands open and
close resources. Ratchet's shared MCP middleware handles its final-result
success boundary; Mosaic adds inventory and enabled-mutation coverage checks.

See [complete capability coverage](HEXXLA_API_SURFACE_COVERAGE.md),
[database upgrades](DATABASE_CREATION.md#upgrade-an-existing-database), and
[the architecture contract](../architecture/architecture.md).
