<div align="center">

<img src="assets/images/mosaic_logo_shadow.svg" alt="Mosaic" width="240">

# Mosaic

**Local MCP server for structured agent memory — hex lattice, hybrid retrieval, governed writes, and budgeted context — backed by [HexxlaDB](https://github.com/hexxla/hexxladb).**

[![CI](https://github.com/hexxla/mosaic/actions/workflows/ci.yml/badge.svg)](https://github.com/hexxla/mosaic/actions/workflows/ci.yml)
[![Integration](https://github.com/hexxla/mosaic/actions/workflows/integration.yml/badge.svg)](https://github.com/hexxla/mosaic/actions/workflows/integration.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sploitzberg/mosaic.svg)](https://pkg.go.dev/github.com/sploitzberg/mosaic)
[![Go Report Card](https://goreportcard.com/badge/github.com/sploitzberg/mosaic)](https://goreportcard.com/report/github.com/sploitzberg/mosaic)
[![Go 1.26](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](https://go.dev/doc/go1.26)
[![Version](https://img.shields.io/badge/version-v0.1.0-7c3aed)](https://github.com/hexxla/mosaic/releases/tag/v0.1.0)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## Why Mosaic

Mosaic keeps agent memory **on infrastructure you operate**: [MCP](https://modelcontextprotocol.io/) on **localhost**, optional **encryption at rest**, and code you can **inspect and extend**. Retention and access follow **policy you define**, so context is not outsourced by default.

Agents fail in production when recalled facts drift or sessions read as unrelated reruns. Mosaic gives memory that **accumulates cleanly across sessions**, so the assistant can anchor on durable state instead of re-deriving intent from prompts alone.

It is backed by **[HexxlaDB](https://github.com/hexxla/hexxladb)**: a **hex lattice** lays out related cells for **spatial, bounded expansion** from a seed; **hybrid retrieval** combines similarity with structured constraints; conflicting updates surface as **seams** rather than disappearing into the embedding space. Operators describe behaviour in **YAML**; callers receive **budgeted context** within explicit limits you can trace and revise.

---

## Get started

**What Mosaic needs**

- **[Ollama](https://ollama.com)** running on your machine. Mosaic needs it for **embeddings** (semantic search and anything else that turns text into vectors). The sample **[configs/config.yaml](configs/config.yaml)** expects Ollama on **`http://127.0.0.1:11434`** with the **`all-minilm`** model; adjust in that file or see **[MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md)** for env vars and options.

---

**Step 1 — Clone the repo**

```bash
git clone https://github.com/hexxla/mosaic.git
cd mosaic
```

---

**Step 2 — Build**

You need **[Go 1.26+](https://go.dev/dl/)**. From the repo root:

```bash
go mod download
make build-mosaic-mcp build-mosaic-create-db
```

After **`make`**, your programs are under **`bin/<platform>/`** (the command prints the paths). Later steps assume **`bin/linux-amd64/`** — use whatever folder **`make`** created on your machine (add **`.exe`** on Windows). There are no downloadable release binaries yet.

**Developers:** **`make seed`** / **`make reseed`**; **`make build-mosaic-seed`** builds a **`mosaic-seed`** binary.

---

**Step 3 — Create a database**

Pick a path for the DB file (**`-db`**). Use the **same** **`-db`** value (or **`MOSAIC_DB_PATH`**) again when you start **`mosaic-mcp`** (Step 5).

```bash
./bin/linux-amd64/mosaic-create-db -db ./data/mosaic.db
```

**Encrypted databases:** If you pass **`-db-passphrase`** or **`MOSAIC_DB_PASSPHRASE`** to **`mosaic-create-db`**, the new database is **encrypted on disk**. Omit both for an unencrypted file. Full behavior: **[DATABASE_CREATION.md](docs/mosaic/DATABASE_CREATION.md#encrypted-database-passphrase-or-raw-key)** (**`MOSAIC_DB_PASSPHRASE`** is usually safer than **`-db-passphrase`** where **`ps`** lists process arguments).

```bash
./bin/linux-amd64/mosaic-create-db -db ./data/mosaic.db -db-passphrase 'use-a-strong-secret'
```

---

**Step 4 — Policy file (YAML)**

An example **`version: 1`** policy ships at **[configs/config.yaml](configs/config.yaml)** — copy or edit from there for **`mosaic-mcp`** ( **`-policy`** or **`MOSAIC_POLICY_FILE`**). All keys → **[MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md)**.

---

**Step 5 — Run the MCP server**

```bash
./bin/linux-amd64/mosaic-mcp -policy configs/config.yaml -db ./data/mosaic.db
```

---

**Step 6 — Point your MCP client at the URL**

By default Mosaic serves Streamable HTTP at **`http://127.0.0.1:8787/mcp`**. Tune **`MOSAIC_MCP_ADDR`** / **`MOSAIC_MCP_PATH`** if needed ([Makefile](Makefile)).

**Cursor:** add a **`mcpServers`** entry (often **`.cursor/mcp.json`** next to your project):

```json
{
  "mcpServers": {
    "mosaic": {
      "url": "http://127.0.0.1:8787/mcp"
    }
  }
}
```

Other clients ship different config files or UIs — use their MCP documentation rather than copying this verbatim.

---

## How Mosaic works

Mosaic fronts **[HexxlaDB](https://github.com/hexxla/hexxladb)** — a single embedded engine with B+tree pages for structure, optional **HNSW** for embeddings, **MVCC** for truthful versions, and sensible handling of large payloads. You do not assemble that yourself; you expose it through one **MCP** surface your agent can learn once.

### Lattice memory, not a flat pile

Memories sit on **hex coordinates**; **related** items can be **near** in the same way they are near on disk. Expansion from a seed is **spatial and bounded** — you pull context in **rings** with intent, not by hoping the top similarity hits cohere. The benefit: **reproducible, explainable** neighbourhoods instead of a black-box vector grab bag.

### Retrieval that stacks

Semantic similarity, structured filters, and lexical search **coexist**. You find candidates with the signal that fits the question, then **assemble** a context pack under a **byte or token budget** so the model sees a **curated slice** of the lattice — not a blunt truncation that happens to fit the window.

### Contradictions you can keep

When two memories disagree, HexxlaDB records **seams** — visible relationships the model can reason about — instead of silently letting the newer embedding win. **Supersession** tracks how preferences evolve without pretending history never happened. The benefit: **auditability and honest dialogue** when knowledge conflicts.

### Policy your operators can sign

**YAML** describes what to retain, whether deletes are permitted, embeddings settings, housekeeping after deletes, encryption hints — not scattered conventions in prose prompts. Agents still improvise reasoning; persistence becomes **enforceable**.

### Telemetry without theatre

Insight into footprint, versioning, integrity — grounded in MVCC-aware checks — sits alongside optional **automatic prune and compact** after deletes so conscientious workloads do not choke on dormant history unless you intend that trade-off.

For command names, YAML keys, and troubleshooting, explore the **`docs/`** tree — begin with **[`docs/mosaic/`](docs/mosaic/)**, then **[`docs/hexxladb/`](docs/hexxladb/)** or **[`docs/architecture/`](docs/architecture/)** as needed. **[AGENTS.md](AGENTS.md)** and **[CHANGELOG.md](CHANGELOG.md)** cover contribution layout and shipped changes.

---

## Ratchet: Tool flow governance

Mosaic integrates **[mcp-ratchet](https://github.com/hexxla/mcp-ratchet)** to enforce tool call sequences and prerequisites. This prevents agents from making unsafe or context-free mutations by requiring specific discovery or validation steps before write operations.

### Features

- **Prerequisite validation**: Tools can require other tools to be called first (e.g., `put_cell` requires `list_tags` to prevent tag fragmentation)
- **One-time use tokens**: Prerequisite tokens can be consumed after use, forcing fresh discovery for each operation
- **Time-bound sessions**: Tokens expire after a configurable duration, ensuring context remains current
- **Flexible rules**: Multiple prerequisite rules per tool (e.g., `put_cell` accepts `list_tags`, `tag_counts`, or retrieval tools)
- **Session management**: Session-based token tracking across tool calls
- **Observability**: Real-time event capture, WebSocket streaming, and HTTP endpoints for monitoring tool usage, session state, and token issuance

### Benefits

- **Tag hygiene**: Forces agents to review available tags before writing, preventing fragmentation
- **Context awareness**: Requires retrieval tools before mutations to verify cell existence and understand context
- **Budget enforcement**: Requires budget estimation before large context loads
- **Auditability**: Tool call sequences are logged and validated against policy
- **Safety**: Prevents blind writes by requiring discovery steps

### Configuration

Ratchet rules are defined in **`configs/ratchet.yaml`**. Each rule specifies:

- `tool`: The tool being governed
- `prerequisite`: Required prerequisite tool (empty string means no prerequisite)
- `error_message`: Custom error message shown when validation fails
- `one_time_use`: Whether the prerequisite token is consumed after use
- `expiry`: Time limit for token validity (ignored when `one_time_use: true`)

Example rule:

```yaml
- tool: "mosaic_hexxla_put_cell"
  prerequisite: "mosaic_hexxla_list_tags"
  error_message: "Before saving a cell, review available tags to prevent fragmentation. Call mosaic_hexxla_list_tags to see current vocabulary."
  one_time_use: true
```

#### Observability

Ratchet observability is configured in the same file under the `observability` section:

```yaml
observability:
  enabled: true
  storage_type: memory # memory or hexxladb
  retention_days: 0 # 0 = keep all events
```

When enabled, Mosaic exposes HTTP endpoints for monitoring:

- **`GET /observability/stats`** - Aggregate statistics (total events, tokens issued, active sessions)
- **`GET /observability/events?session_id=<id>&limit=<n>`** - Paginated event history
- **`WS /observability/stream`** - WebSocket endpoint for real-time event streaming

For detailed configuration options and examples, see **[`docs/mosaic/RATCHET_INTEGRATION.md`](docs/mosaic/RATCHET_INTEGRATION.md)**.

---

## Reliable tool use from agents

MCP **does not force** models to call tools. Reinforce behavior with **project rules**, **`retention.notes`** in your policy YAML (they are injected into server instructions), and **[`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc)** or **[docs/mosaic/AGENT_CLIENT_WORKFLOWS.md](docs/mosaic/AGENT_CLIENT_WORKFLOWS.md)**. The steady pattern is **discover** candidates, **assemble** a budgeted context pack, and **persist** only where policy permits — **`docs/mosaic/`** spells out names and payloads.

---

## Development

```bash
make ci          # full pipeline (same as CI)
make test        # unit tests
make integration # tagged integration tests
```

---

## Roadmap

Exploration and limits → **[TODOS.md](TODOS.md)** · themes → **[docs/ROADMAP.md](docs/ROADMAP.md)**

---

## Sponsorship

Mosaic is open source and under active development. If it's useful to your work — or you want to accelerate the roadmap (distributed replication, materialized views, richer seam semantics) — sponsorship is the most direct way to help.

- **GitHub Sponsors:** [github.com/sponsors/hexxla](https://github.com/sponsors/hexxla)
- **Monero (XMR):** `46shAhAihZ3dmVHGU4V6H2ZZt21ex8xydB7Awkxaheq4U1VZFoK53K92tsqhnL8roV2bV8pQWCryR3yNRJJd5gAeBsZUXPF`
- **Open Collective:** _coming soon_

Sponsors get early access to roadmap discussions, priority issue triage, and attribution in release notes.

---

## License

MIT — see [LICENSE](LICENSE).
