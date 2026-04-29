# Roadmap

For completed work, see [`CHANGELOG.md`](../CHANGELOG.md) and [`TODOS.md`](../TODOS.md) (Recently Completed).

## Near-term

Engineering polish aligned with Mosaic as the MCP orchestration layer over HexxlaDB. Details and checkboxes live in [`TODOS.md`](../TODOS.md); echoed here as roadmap themes.

- **Oversized one-shot (session cap)** — When `retrieval.session_approx_token_budget` is on, `BeforeRead` only sees **prior** cumulative usage. A single call can still run HexxlaDB and build a full result that **would** exceed the cap on record; we fail at metering time (no usage bump, client error). **Explore:** tighter per-call defaults, truncation, pre-estimates, reject-before-query. Full definition: [`TODOS.md`](../TODOS.md) *(Pending)*.
- **Session meter lifetime** — In-memory counters keyed by MCP session id (process lifetime). **Explore:** reset/TTL, per-chat boundaries, operator story. [`TODOS.md`](../TODOS.md) *(Pending)*.
- **Approximate metering vs tokenizer** — `approx_tokens_used` is JSON UTF‑8 length ÷ `bytes_per_approx_token`, not the host tokenizer. **Explore:** labeling vs optional hooks. [`TODOS.md`](../TODOS.md) *(Pending)*.

### Agent experience & host integration

These reduce friction for assistants and humans wiring Mosaic into MCP clients (especially Cursor).

- **MCP server identifier mismatch** — Cursor’s bridge may use an internal id (e.g. `project-0-mosaic-mosaic`) while config uses `mosaic`; mis-invocation causes “server does not exist.” **Improves Mosaic** by cutting setup/debug time and failed tool runs. **How:** Surface both ids in docs + optional tiny **Cursor** doc; see [`TODOS.md`](../TODOS.md) *(Pending)*.
- **Lexical vs embedding semantics** — Agents should pick **`search_cells`** vs **`search_embedding`** (and hybrid `embed_query_text`) on purpose; empty lexical + non-empty ANN is **expected** when text is not literal in the DB. **Improves** answer quality and fewer false “no data” conclusions. **How:** Blueprint + rule decision table; [`TODOS.md`](../TODOS.md) *(Pending)*.
- **Preferences = structured query** — Prefer **`query_cells`** with **`require_tags`** including **`preference`** over a loose full-text query when loading user prefs from seeded/tagged cells. **Improves** precision and aligns with taxonomy. **How:** Document pattern in blueprint + agent rule; [`TODOS.md`](../TODOS.md) *(Pending)*.

## Future

Spec or direction exists; scheduling TBD.

- **Smarter cap semantics** — Broader follow-up to **oversized one-shot**: optional pre-estimate or tighter per-call clamps so session egress limits can fail **before** expensive DB work when a single response would overshoot.
- **Meter reset / TTL** — Operator or client-controlled boundaries for cumulative metering (per chat, per day, or explicit reset), if production needs diverge from “one MCP session ≈ one counter until reconnect/restart.”

## Future exploration

Interesting but unvalidated; needs product signal.

- **Metrics export** — Prometheus/OpenTelemetry counters for `approx_tokens_used` per session or globally (today: in-process + `mosaic_hexxla_retrieval_budget_status`).

## Out of Scope

Intentional boundaries for Mosaic as a **local** MCP server.

- **Tokenizer-accurate “LLM context” accounting** — Mosaic meters **JSON egress** with a bytes-per-approx-token model; real tokenizer counts live in the host model runtime.
- **Relevance scoring inside the cap** — Session cumulative limits are **egress / runaway** controls, not substitutes for **per-call** ring and byte budgets or retrieval discipline (see [`docs/mosaic/MCP_AGENT_BLUEPRINT.md`](mosaic/MCP_AGENT_BLUEPRINT.md)).
