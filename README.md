<div align="center">

<img src="assets/images/mosaic_logo_shadow.svg" alt="Mosaic" width="240">

# Mosaic

**Local MCP server for structured agent memory — hex lattice, hybrid retrieval, governed writes, and budgeted context — backed by [HexxlaDB](https://github.com/hexxla/hexxladb).**

[![CI](https://github.com/hexxla/mosaic/actions/workflows/ci.yml/badge.svg)](https://github.com/hexxla/mosaic/actions/workflows/ci.yml)
[![Integration](https://github.com/hexxla/mosaic/actions/workflows/integration.yml/badge.svg)](https://github.com/hexxla/mosaic/actions/workflows/integration.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sploitzberg/mosaic.svg)](https://pkg.go.dev/github.com/sploitzberg/mosaic)
[![Go Report Card](https://goreportcard.com/badge/github.com/sploitzberg/mosaic)](https://goreportcard.com/report/github.com/sploitzberg/mosaic)
[![Go 1.27](https://img.shields.io/badge/go-1.27-00ADD8?logo=go)](https://go.dev/doc/go1.27)
[![Version](https://img.shields.io/badge/version-v0.2.0-7c3aed)](https://github.com/hexxla/mosaic/releases/tag/v0.2.0)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## Why Mosaic

Mosaic keeps agent memory **on infrastructure you operate**: [MCP](https://modelcontextprotocol.io/) on **localhost**, optional **encryption at rest**, and code you can **inspect and extend**. Retention guidance and delete permission follow **policy you define**, so context is not outsourced by default.

Agents fail in production when recalled facts drift or sessions read as unrelated reruns. Mosaic gives memory that **accumulates cleanly across sessions**, so the assistant can anchor on durable state instead of re-deriving intent from prompts alone.

It is backed by **[HexxlaDB](https://github.com/hexxla/hexxladb)**: a **hex lattice** lays out related cells for **spatial, bounded expansion** from a seed; **hybrid retrieval** combines similarity with structured constraints; callers can record conflicting updates as explicit **seams** rather than losing the disagreement in embedding space. Operators describe behaviour in **YAML**; callers receive **budgeted context** within explicit limits you can trace and revise.

---

## Get started

**What Mosaic needs**

- **[Ollama](https://ollama.com)** only for operations that turn text into vectors, including semantic search, text embedding writes, and `mosaic-seed`. Database creation, health, lexical/structured reads, raw-vector writes, and the other non-embedding tools do not require it. The sample **[configs/config.yaml](configs/config.yaml)** expects Ollama on **`http://127.0.0.1:11434`** with the **`all-minilm`** model; adjust it or see **[MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md)**.

---

**Step 1 — Clone the repo**

```bash
git clone https://github.com/hexxla/mosaic.git
cd mosaic
```

---

**Step 2 — Build**

You need **[Go 1.27+](https://go.dev/dl/)**. From the repo root:

```bash
go mod download
make build-mosaic-mcp build-mosaic-create-db
```

After **`make`**, your programs are under **`bin/<platform>/`** (the command prints the paths). Later steps assume **`bin/linux-amd64/`** — use whatever folder **`make`** created on your machine (add **`.exe`** on Windows).

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

Cell writes make placement explicit. `mosaic_hexxla_put_cell` defaults to an exact coordinate and refuses to replace a live cell unless `allow_overwrite` is set. With `placement: near_anchor`, `(q,r)` is a caller-chosen semantic anchor and Mosaic atomically selects the first free coordinate in deterministic ring order within a bounded radius. The response returns the actual coordinate; Mosaic does not infer semantic meaning or silently relocate existing cells.

### Retrieval that stacks

Semantic similarity, structured filters, and lexical search **coexist**. You find candidates with the signal that fits the question, then **assemble** a context pack under a UTF-8 **byte budget** (optionally estimated from a token target) so the model sees a **curated slice** of the lattice. Mosaic stays provider-neutral; exact tokenizer accounting belongs with the client that renders the final model request.

### Contradictions you can keep

When two memories disagree, callers can record a **conflict seam** with `mosaic_hexxla_mark_conflict`; an intentional replacement uses `mosaic_hexxla_mark_supersedes`. Neither Mosaic nor HexxlaDB infers semantic disagreement automatically. These visible relationships let the model reason about conflict instead of silently letting a newer embedding win.

### Policy your operators can sign

**YAML** describes capture guidance, whether deletes are permitted, embedding settings, housekeeping after deletes, and encryption hints—not scattered conventions in prose prompts. Delete permission and configured maintenance are enforced by the server; capture modes and notes guide clients but do not auto-save turns.

### Telemetry without theatre

Insight into footprint, versioning, integrity — grounded in MVCC-aware checks — sits alongside optional **automatic prune and compact** after deletes so conscientious workloads do not choke on dormant history unless you intend that trade-off.

For command names, YAML keys, and troubleshooting, begin with **[`docs/mosaic/`](docs/mosaic/)** and the local **[`docs/architecture/`](docs/architecture/)**. HexxlaDB’s storage/API docs live in the **[upstream `docs/hexxladb` tree](https://github.com/hexxla/hexxladb/tree/main/docs/hexxladb)**. **[AGENTS.md](AGENTS.md)** and **[CHANGELOG.md](CHANGELOG.md)** cover contribution layout and shipped changes.

---

## Reliable tool use from agents

MCP **does not force** models to call tools. Reinforce behavior with **project rules**, **`retention.notes`** in your policy YAML (they are injected into server instructions), and **[`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc)** or **[docs/mosaic/AGENT_CLIENT_WORKFLOWS.md](docs/mosaic/AGENT_CLIENT_WORKFLOWS.md)**. The steady pattern is **discover** candidates, **assemble** a budgeted context pack, and **persist** only where policy permits. Runtime `tools/list` schemas are the authoritative payload contract.

The repository also contains a **[Ratchet integration proposal](docs/mosaic/RATCHET_INTEGRATION.md)** and sample **[`configs/ratchet.yaml`](configs/ratchet.yaml)**. They are inactive design artifacts: `mosaic-mcp` does not load that file, enforce its prerequisites, or expose Ratchet observability endpoints.

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
