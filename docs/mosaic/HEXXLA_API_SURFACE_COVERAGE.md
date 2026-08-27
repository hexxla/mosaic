# HexxlaDB API surface ↔ Mosaic MCP coverage

This document maps the public feature families in
`github.com/hexxla/hexxladb` **v0.6.0** (the version pinned by Mosaic) to the
23 tools registered by `cmd/mosaic-mcp`. It is feature-family complete: related
symbols are grouped when they share one Mosaic disposition. Runtime MCP
`tools/list` remains authoritative for exact input/output schemas.

Companions: [HEXXLADB_API_NOTES.md](./HEXXLADB_API_NOTES.md) explains the
hexagonal adapter boundary; [MCP_AGENT_BLUEPRINT.md](./MCP_AGENT_BLUEPRINT.md)
describes agent workflows.

## Dispositions

| Disposition | Meaning |
| --- | --- |
| **MCP tool** | Available to the general local agent surface today |
| **Composition** | Used internally to construct or operate Mosaic, not callable as a tool |
| **Candidate** | Read-only capability worth a bounded product experiment; see the roadmap |
| **Native API** | Deliberately left to trusted Go integrations because the generic MCP abstraction would be weak or unsafe |
| **Operator only** | Backup, maintenance, retention, migration, encryption, or raw administration; keep outside the general agent surface |

## Engine lifecycle, health, and physical operations

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `Open`, `Options`, `Close`; `PageSize`, `MaxValueBytes`, `EmbeddingDimension`, `EmbeddingMetric` | **Composition**. The layout readers are returned by `mosaic_hexxla_health`. |
| `HealthCheck` | `mosaic_hexxla_health`; Mosaic also reports primary/WAL file sizes and its effective MVCC retention setting. |
| `WriteStats`, `GroupWALStats`, `StorageStats` | **Operator only**. The MCP health response does not claim these full reports. |
| `BackupTo` | **Operator only**; backups need operator-selected destinations and lifecycle control. |
| `Compact`, `CompactWithOptions`, `CompactTo`, `CompactToWithOptions`, `PreflightCompactTo`, `ReclaimTail` | **Operator only**; I/O-heavy physical maintenance. |
| `BatchPutCells`, `ImportCellsJSON`, `Tx.ExportCellsJSON` | **Operator only** for bulk transfer/ingestion. MCP writes stay individually governed. |

## Transactions and generic KV

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `View`, `Update`, `Batch` | **Composition** inside secondary adapters; transaction handles never cross a Mosaic port. |
| `Tx.Get`, `Tx.Put`, `Tx.AscendRange`, `Tx.Writable` | **Native API**. Mosaic exposes product-shaped cells, seams, facets, edges, and embeddings—not arbitrary byte keys. |
| `AfterPutCellHook`, `AfterPutSeamHook`, `CellValidator` options | **Native API** configuration seams; not runtime MCP tools. |

## Cells, placement, and structured retrieval

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `PutCell`, `FindFreeCellPlacement` | `mosaic_hexxla_put_cell`: safe exact placement or bounded `near_anchor`, with the actual coordinate returned. Mosaic does not infer the semantic anchor. |
| `DeleteCell`, `DeleteCellWithOutcome` | `mosaic_hexxla_delete_cell`, additionally controlled by `allow_delete_cell`. |
| `GetCell` | No coordinate-only tool. Cell records are returned through bounded query/search/context tools. |
| Cell templates (`NewFactCell`, `NewUserMessageCell`, `NewAssistantResponseCell`) | `mosaic_hexxla_put_cell` selects the corresponding product template through `kind`. `NewSystemPromptCell` remains **Native API**. |
| `QueryCells`, `SearchCells` | `mosaic_hexxla_query_cells`, `mosaic_hexxla_search_cells`; optional `embed_query_text` supplies hybrid ANN candidate selection. |
| `AscendCellsByTag`, `AscendCellsBySource`, `AscendCellsInTimeBucket`, `AscendDistinctTags` | Covered by higher-level query filters or tag tools where useful; raw index iteration remains **Native API**. |

## Context, spatial views, and lattice geometry

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `LoadContext`, `AssembleCellView` | `mosaic_hexxla_load_context_pack`; HexxlaDB bounds candidates and Mosaic applies provider-neutral UTF-8 byte budgeting. |
| `WalkRing`, `WalkRingAt`, `WalkRingFacets`, `ScanContextRaw`, `ScanContextAtRaw` | **Native API**. The context-pack tool is the safer bounded agent abstraction. |
| `LoadContextFOV`, `LoadContextVoronoi` | **Native API** for specialized spatial applications; no demonstrated general-agent need yet. |
| `Coord`, `PackedCoord`, `Pack`, `Unpack`, `Ring`, `WalkRings` | **Native API** geometry primitives. MCP schemas use axial `q,r` fields. |
| `RingDensityMap`, `TotalDensity` | **Candidate** for bounded placement/coverage diagnostics; no current MCP tool. |
| `RenderHexGrid`, `RenderHexGridFromDB`, `SuperHexSummaryIndex` | **Native API** visualization/derived-index facilities. |

## Tags and graph traversal

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `ListExistingTopics`, `TagCounts` | `mosaic_hexxla_list_tags`, `mosaic_hexxla_tag_counts`. |
| `TagCooccurrences`, `UntaggedCells` | **Candidate** for bounded taxonomy-quality tools; no current MCP tool. |
| `FindEdgePath`, `WalkEdges`, `WalkEdgeCoords` | **Candidate** for bounded relationship traversal with strict hop/node/result caps; no current MCP tool. |

## Embeddings

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `PutEmbedding`, `PutEmbeddingWithOptions` | `mosaic_hexxla_put_embedding` accepts either a raw vector or text embedded through configured Ollama. Advanced write options remain **Native API**. |
| `GetEmbedding`, `DeleteEmbedding` | Not exposed independently. Search returns matches rather than raw stored vectors; cell deletion performs HexxlaDB's cell-associated cleanup when policy allows. |
| `SearchByEmbedding` | `mosaic_hexxla_search_embedding`; query text is embedded through configured Ollama. |
| `SearchByEmbeddingWithStats` | Search behavior is exposed, but detailed execution-path statistics remain **Native API**. |
| `ReindexEmbeddings`, `RebuildEmbeddingIndex` | **Operator only**; potentially expensive derived-index maintenance. |

## Seams

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `FindSeams` | `mosaic_hexxla_find_seams`. |
| `MarkConflict`, `MarkSupersedes`, `ResolveSeam` | `mosaic_hexxla_mark_conflict`, `mosaic_hexxla_mark_supersedes`, `mosaic_hexxla_resolve_seam`. |
| `PutSeam` | **Native API**. Mosaic exposes validated lifecycle operations instead of arbitrary seam records. |
| `FindSeamsAt`, `AscendSeamsBySource`, `AscendSeamsInTimeBucket` | **Native API**; no separate temporal/index-walk tool. |

## Facets and edges

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `PutFacet`, `NewFacetDerived` | `mosaic_hexxla_put_facet`. It does not expose `UpdateFacet`'s derivation-hash guard. |
| `UpdateFacet` | **Native API** when guarded derived-content replacement is required. |
| `GetFacet`, `AscendFacetsForCell` | `mosaic_hexxla_get_facet`, `mosaic_hexxla_list_facets`. |
| `LinkCells`, `PutEdge`, `NewProvenanceWire` | `mosaic_hexxla_link_cells`; Mosaic constructs provenance and does not accept arbitrary edge records. |
| `GetEdge`, `AscendEdgesFrom` | `mosaic_hexxla_get_edge`, `mosaic_hexxla_list_edges_from` (`max_edges` default 50, cap 200, explicit `truncated`). |

## MVCC, snapshots, and time travel

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `ViewAt`, `ViewAtTime`, `ViewAtTag`; `TagSnapshot`, `ListSnapshotTags`, `DeleteSnapshotTag` | Read-only portions are a **Candidate** only after authorization, retention, and response bounds are specified. Snapshot creation/deletion stays **Operator only**. |
| `SnapshotDiff` | **Candidate** for a bounded read-only comparison tool; no current MCP tool. |
| `StatsMVCC`, `SuggestedPruneBeforeSeq`, `MVCCPrunePlan` | **Operator only** diagnostics/planning. |
| `PruneCellVersions`, `PruneCellVersionsByProfile`, `PruneScheduler` | **Operator only**; can remove retained history. Mosaic may run configured post-delete maintenance internally, but exposes no direct tool. |

## Changelog and consumers

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| `ReadChangelogSince`, `ReadChangelogFiltered` | A strictly bounded read-only peek is a **Candidate** for audit workflows; the full changefeed remains **Operator only**. |
| `GetChangelogConsumerCursor`, `ListChangelogConsumers`, `ChangelogRetentionFloor` | **Operator only** observability and retention coordination. |
| `AdvanceChangelogConsumer`, `DeleteChangelogConsumer` | **Operator only** state mutation; never a general-agent tool. |

## Encryption, migration, and recovery

| HexxlaDB v0.6.0 capability | Mosaic disposition |
| --- | --- |
| Passphrase/key options, `DeriveKeyFromPassphrase` | **Composition** from protected operator configuration; never exposed through MCP payloads. |
| `RotateEncryption`, `RotateEncryptionWithOptions`, `RecoverInterruptedRotation` | **Operator only** key-management/recovery workflow. |
| `PreflightMigrateV1ToV2`, `MigrateV1ToV2`, `PreflightMigrateToAuthenticated`, `MigrateToAuthenticated` | **Operator only** offline format migration. |

## Default-surface boundary

The general MCP surface intentionally excludes arbitrary KV access, raw record
construction, bulk import/export, physical maintenance, retention mutation,
changefeed cursor mutation, migration, and encryption operations. Those actions
need explicit operator authority, destinations, credentials, maintenance
windows, or recovery plans that an ordinary local agent call cannot safely
represent.

Read-only candidates are not assumed to be safe merely because they do not
write. Each needs hard request/response bounds, data-exposure review, and a
demonstrated agent workflow before implementation. Their current evaluation and
priority are recorded in [the roadmap](../ROADMAP.md).

## Maintenance

When Mosaic changes HexxlaDB versions or registers a tool:

1. compare the pinned package's public `DB`, `Tx`, and package-level exports;
2. update this feature-family map and the runtime agent guidance;
3. add the tool to the explicit safety-policy inventory; and
4. classify every mutation in the enabled Ratchet policy before startup.
