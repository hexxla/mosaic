# HexxlaA Hexagonal Spatial Memory Operating System for LLMs

**Version 2.0**
**Date:** April 2026
**Project Name:** HexxlaVision

Hexxla is a deterministic hexagonal lattice that serves as a spatial operating layer for long-term LLM memory. It combines a purpose-built embedded database (HexxlaDB) with a higher-level runtime that orchestrates seed selection, neighborhood expansion, explicit contradiction handling, and token-efficient context assembly.

The system delivers inspectable organization, manageable knowledge conflicts, and reproducible memory retrieval. Semantic or lexical methods propose one or more seed coordinates; the lattice then governs all subsequent structure and loading behavior.

This hybrid approach provides token-efficient context packing, visible contradictions via first-class seams, bi-temporal evolution, and natural hierarchical clustering — making it a true memory operating system rather than another vector or graph store.

## Why Hexagonal?

A hexagonal lattice offers 6-neighbor connectivity, natural ring enumeration, exact deterministic distance, and hierarchical super-hex regions. These properties make locality, neighborhood traversal, summarization, and collaborative evolution first-class operations.

## Relationship to HexxlaDB

HexxlaDB is the production-ready embedded storage engine (v0.2+). It provides the hex-native keyspace, Morton-packed coordinates, MVCC snapshots, optional HNSW embeddings, WAL durability, changefeed, and all core primitives (`PutCell`, `WalkRing`, `LoadContextPackFrom`, `FindSeams`, `ResolveSeam`, etc.).

The Hexxla Runtime (this document) builds the agent-facing memory OS on top of HexxlaDB. It owns orchestration, tool surfaces, facet policies, seam auto-detection, context budgeting, and higher-level workflows.

For a **narrower, local-first milestone** — an MCP server (Streamable HTTP on loopback) whose tools talk directly to HexxlaDB, without the full runtime — see [MCP_BLUEPRINT.md](./MCP_BLUEPRINT.md).

## Core Architecture

### Geometric Model

Memory cells are addressed by axial coordinates `(q, r)` with implicit cube coordinate `s = -q - r`. Distance, neighbors, and rings are computed deterministically using standard axial hex formulas.

All core operations after seed selection remain purely geometric and inspectable.

### Seed Selection Layer

Seed selection is the only fuzzy step. The runtime provides a pluggable `SeedSelector` interface with built-in strategies:

- Embedding-based (using HexxlaDB HNSW when enabled)
- Lexical / tag-based
- Explicit coordinate
- Multi-seed merging for composite queries

Multiple seeds can be combined and passed to `LoadContextPackFrom` for unified budgeted assembly.

## Core Objects

### Coord

Axial `(q, r)` with full lattice methods (Distance, Neighbors, Ring, etc.)

### Cell

- Coord
- RawContent (immutable anchor)
- Provenance
- ValidityWindow
- Tags
- ClusterHint
- Facets (map)
- ActiveFacet
- Edges (read aggregate)
- Seams (read aggregate)

### FacetView

Derived views of the raw content (summary, conflict notes, procedural, custom, etc.)

### Seam

First-class contradiction record with ULID ID, participating cells, reason, confidence delta, resolution status, and full audit trail.

ValidityWindow and Provenance remain as defined in HexxlaDB.

## Facets

Facet IDs are fixed.

| Facet ID | Purpose                         | Lifecycle Notes                 |
| -------- | ------------------------------- | ------------------------------- |
| 0        | Raw verbatim source (immutable) | Created on put; never updated   |
| 1        | Semantic summary                | Derived; hash-validated         |
| 2        | Conflict notes and seams        | Auto-populated on seam creation |
| 3        | Temporal validity window        | Mirrors cell validity           |
| 4        | Procedural or action-oriented   | LLM-derived                     |
| 5        | User or project-specific lens   | Custom                          |

The runtime manages facet derivation, rotation policies, and hash validation.

## Retrieval and Context Orchestration

Retrieval follows a clean four-phase flow:

1. Seed selection (fuzzy)
2. Deterministic spatial expansion (rings or bounded radius)
3. Filtering and ranking (validity, provenance, tags, seams, confidence)
4. Token-budget-aware packing

The primary runtime primitive is:

LoadContextPackFrom(seeds, maxTokens, options)

It supports:

- Concentric ring loading
- Spiral ordering within rings
- Supersession filtering
- Seam highlighting
- Pluggable budget strategies (byte length, token estimation, hybrid score)

**Ordering rule:** concentric rings from center(s) outward, axial spiral within each ring starting from positive-q direction. Low-confidence or superseded items are dropped from outer rings first when the budget is exceeded.

Hierarchical super-hex clustering provides natural zoom levels for large memories.

## Contradiction and Evolution Engine

Conflicts are modeled as explicit, queryable Seams. The runtime strengthens this with:

- Automatic lightweight seam detection during `PutMemory`
- Manual `mark_conflict` / `mark_supersedes` tools
- LLM-guided resolution strategies: Supersede, Merge (new cell + linking seams), Archive
- Full audit trail preserved via MVCC and changefeed

Preference Cells (tagged `"preference"`) receive elevated priority in all context packs and strong supersession semantics.

Evolution is first-class through:

- Facet rotation
- Validity windows
- Seam resolution
- Temporal snapshots (`ViewAtTime`)
- Provenance tracking

## Agent Tooling Surface

The runtime exposes a clean, LLM-friendly tool interface:

- `put_memory(raw_content, tags, coord_hint?, provenance?, validity?)`
- `search_and_load_context(query, max_tokens, strategy?)`
- `mark_conflict(coord_a, coord_b, reason)`
- `mark_supersedes(old_coord, new_coord, reason)`
- `resolve_seam(seam_id, strategy, note?)`
- `get_temporal_view(as_of_time)`
- `explain_seams(center, radius)`

All tools return structured observations suitable for the next LLM reasoning step. The runtime handles validation, provenance injection, and HexxlaDB transaction boundaries.

## Time and Evolution

Every cell and seam carries explicit timestamps and validity windows. MVCC snapshots enable reproducible `as_of` views and `SnapshotDiff` operations. A rotating time-wheel concept supports temporal navigation and slicing in UIs and agents.

## Non-Goals (v1 Runtime)

- Automatic global semantic clustering or auto-placement
- Full autonomous agent loop (belongs in consuming frameworks)
- Built-in embedding model hosting (use external providers like Ollama)
- General-purpose graph or vector database replacement
- Experimental dynamics (pollen, crystallization, resonance, buckling) — reserved for post-v1

## Baseline Comparison

| Aspect                      | Hexxla (v2)                               | Pure Vector DB   | Pure Graph DB       |
| --------------------------- | ----------------------------------------- | ---------------- | ------------------- |
| Token efficiency            | High via budgeted ring packing            | Medium via top-k | Variable            |
| Contradiction visibility    | Explicit first-class seams + supersession | Implicit         | Links only          |
| Locality and inspectability | Deterministic rings + spatial hierarchy   | Approximate      | High traversal cost |
| Update cost                 | Local immutable cells + seams             | Re-embedding     | Often global        |
| Temporal support            | Native MVCC + validity windows            | Add-on           | Add-on              |
| Collaboration readiness     | Changefeed + super-hex foundation         | Limited          | Limited             |

## Implementation Notes

- Language: Go
- Storage: HexxlaDB (embedded, hex-native B+-tree, MVCC, optional HNSW)
- Architecture: Hexagonal (ports and adapters) — runtime depends on HexxlaDB via stable public API
- Visualization: Honeycomb TUI/dashboard with seam highlighting, ring overlays, and temporal controls
- Extensibility: Pluggable SeedSelectors, BudgetStrategies, and changefeed consumers

## v1 Scope (Runtime)

- Seed selection strategies
- `LoadContextPackFrom` with multi-seed and budget control
- Full tool surface for LLM agents
- Facet management and rotation policies
- Seam detection, resolution, and preference cell handling
- Temporal views and changefeed integration
- Basic honeycomb visualization patterns

## Evaluation

Focus on:

- Token efficiency and context coherence
- Contradiction detection and resolution quality
- Agent tool ergonomics and reproducibility
- Neighborhood operation latency
- Dashboard and TUI interpretability
- Collaboration readiness via changefeed

## Final Positioning

Hexxla is a hexagonal spatial memory operating system for LLMs and agents. HexxlaDB provides the durable lattice foundation; the runtime adds orchestration, agent tooling, and collaborative evolution primitives. Together they deliver inspectable, token-efficient, contradiction-aware long-term memory that scales from personal use to federated knowledge fabrics.

This architecture trades global semantic fluidity for determinism, transparency, and efficiency — exactly the properties needed for reliable agent cognition.
