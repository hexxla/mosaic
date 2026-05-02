# Roadmap

For completed work, see [`CHANGELOG.md`](../CHANGELOG.md) and [`TODOS.md`](../TODOS.md) (Recently Completed).

## Recently shipped

Documented in **`CHANGELOG.md`**; session log in **`TODOS.md` → Recently Completed**. Includes: post-delete **debounced** prune/compact + shutdown flush; **`mosaic_hexxla_health`** **`disk`** + **`mvcc_retain_commits_behind_head`**; **`MOSAIC_CONFIG.md`** and minimal **`configs/config.yaml`**; **`README`** refresh; Go module **`github.com/sploitzberg/mosaic`** and removal of the old bootstrap CLI.

## Near-term

Engineering polish aligned with Mosaic as the MCP orchestration layer over HexxlaDB. Details and checkboxes live in [`TODOS.md`](../TODOS.md); echoed here as roadmap themes.

- **Oversized one-shot (session cap)** — When `retrieval.session_approx_token_budget` is on, `BeforeRead` only sees **prior** cumulative usage. A single call can still run HexxlaDB and build a full result that **would** exceed the cap on record; we fail at metering time (no usage bump, client error). **Explore:** tighter per-call defaults, truncation, pre-estimates, reject-before-query. Full definition: [`TODOS.md`](../TODOS.md) _(Pending)_.
- **Session meter lifetime** — In-memory counters keyed by MCP session id (process lifetime). **Explore:** reset/TTL, per-chat boundaries, operator story. [`TODOS.md`](../TODOS.md) _(Pending)_.
- **Approximate metering vs tokenizer** — `approx_tokens_used` is JSON UTF‑8 length ÷ `bytes_per_approx_token`, not the host tokenizer. **Explore:** labeling vs optional hooks. [`TODOS.md`](../TODOS.md) _(Pending)_.

### Agent experience & host integration

These reduce friction for assistants and humans wiring Mosaic into MCP clients (especially Cursor).

- **MCP server identifier mismatch** — Cursor’s bridge may use an internal id (e.g. `project-0-mosaic-mosaic`) while config uses `mosaic`; mis-invocation causes “server does not exist.” **Improves Mosaic** by cutting setup/debug time and failed tool runs. **How:** Surface both ids in docs + optional tiny **Cursor** doc; see [`TODOS.md`](../TODOS.md) _(Pending)_.
- **Lexical vs embedding semantics** — Agents should pick **`search_cells`** vs **`search_embedding`** (and hybrid `embed_query_text`) on purpose; empty lexical + non-empty ANN is **expected** when text is not literal in the DB. **Improves** answer quality and fewer false “no data” conclusions. **How:** Blueprint + rule decision table; [`TODOS.md`](../TODOS.md) _(Pending)_.
- **Preferences = structured query** — Prefer **`query_cells`** with **`require_tags`** including **`preference`** over a loose full-text query when loading user prefs from seeded/tagged cells. **Improves** precision and aligns with taxonomy. **How:** Document pattern in blueprint + agent rule; [`TODOS.md`](../TODOS.md) _(Pending)_.

## Future

Spec or direction exists; scheduling TBD.

- **Smarter cap semantics** — Broader follow-up to **oversized one-shot**: optional pre-estimate or tighter per-call clamps so session egress limits can fail **before** expensive DB work when a single response would overshoot.
- **Meter reset / TTL** — Operator or client-controlled boundaries for cumulative metering (per chat, per day, or explicit reset), if production needs diverge from “one MCP session ≈ one counter until reconnect/restart.”

## Future exploration

Interesting but unvalidated; needs product signal.

- **Metrics export** — Prometheus/OpenTelemetry counters for `approx_tokens_used` per session or globally (today: in-process + `mosaic_hexxla_retrieval_budget_status`).

## Competitive Capability Roadmap

These features position Mosaic as the most capable local agent memory system across retrieval performance, automation, flexibility, and developer experience.

### Retrieval Performance

- **LongMemEval benchmark** — Achieve 96%+ R@5 on LongMemEval (500 questions) without LLM reranking, 98%+ with hybrid retrieval. Include held-out validation set and reproducible benchmark scripts.
- **Hybrid BM25+vector search** — BM25 keyword matching as fallback when vector similarity misses exact terms. Okapi-BM25 with tunable k1/b parameters, integrated with ANN for candidate re-ranking.
- **Time-decay scoring** — Temporal-proximity boosting where recent memories surface before older ones. Configurable decay functions integrated with context pack assembly.
- **Additional benchmark suites** — LoCoMo (target 88%+ R@10), ConvoMo (target 92%+ avg recall), MemBench (target 80%+ R@5). Reproducible benchmark infrastructure.

### Knowledge & Structure

- **Entity abstraction layer** — Entity-first query interface on top of Hexxladb edges (cells represent entities with typed relationships). Leverage existing ValidFrom/ValidTo validity windows and MVCC time-travel for temporal queries. No separate database needed.
- **Time-decay scoring** — Integrate temporal proximity into Hexxladb's composite relevance score calculation. Add configurable decay functions that boost recent cell scores during retrieval using existing ValidFrom timestamps.
- **Local NLP entity extraction** — On-device entity extraction, relationship detection, topic classification without external APIs. Feature-flagged, consumer hardware support (GPU-accelerated when available).
- **Entity/topic compression** — Compression scheme for names, repeated words, concepts, and key moments into AI-readable shorthand. Index layer points to full content for fast scanning.
- **Cross-wing navigation tunnels** — Tunnels/links between related content across wings. Creation, listing, following, deletion. Graph traversal with cross-wing queries.

### Automation & Tooling

- **Background automation hooks** — Auto-save hooks for IDE integration (Claude Code, Cursor). Periodic save and pre-context-compression triggers. Subagent pipeline for background filing.
- **CLI tooling expansion** — Mine, search, wake-up commands. Project file scanning, conversation history ingestion, direct palace management without MCP.
- **Agent diaries** — Per-agent memory spaces with automatic wing/drawer assignment. Runtime discoverable via MCP tools. Agent-specific preferences and context.
- **Expanded MCP tool coverage** — Drawer CRUD, paginated export, hook settings, health checks, maintenance operations. Target 30+ tools.

### Infrastructure & Flexibility

- **Pluggable storage backend abstraction** — Backend interface (BaseCollection, BaseBackend) for drop-in replacements beyond HexxlaDB. PostgreSQL backend (ACID guarantees), LanceDB backend (multi-device sync).
- **Query sanitization** — System prompt contamination mitigation with input validation and query normalization. Injection attack protection.
- **Stale index detection** — Automatic reconnection when HNSW index changes on disk. Corruption detection, rebuild triggers, repair utilities.

## Out of Scope

Intentional boundaries for Mosaic as a **local** MCP server.

- **Tokenizer-accurate "LLM context" accounting** — Mosaic meters **JSON egress** with a bytes-per-approx-token model; real tokenizer counts live in the host model runtime.
- **Relevance scoring inside the cap** — Session cumulative limits are **egress / runaway** controls, not substitutes for **per-call** ring and byte budgets or retrieval discipline (see [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](mosaic/MCP_AGENT_BLUEPRINT.md)).
