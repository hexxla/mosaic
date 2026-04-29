<div align="center">

<img src="assets/images/mosaic_logo_shadow.svg" alt="Mosaic" width="300">

# Mosaic

**The memory and context layer your agent deserves — not a vector dump, a real orchestration surface.**

[![CI](https://github.com/sploitzberg/go-llm-project-structure/actions/workflows/ci.yml/badge.svg)](https://github.com/sploitzberg/go-llm-project-structure/actions/workflows/ci.yml)
[![Integration](https://github.com/sploitzberg/go-llm-project-structure/actions/workflows/integration.yml/badge.svg)](https://github.com/sploitzberg/go-llm-project-structure/actions/workflows/integration.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sploitzberg/go-llm-project-structure.svg)](https://pkg.go.dev/github.com/sploitzberg/go-llm-project-structure)
[![Go Report Card](https://goreportcard.com/badge/github.com/sploitzberg/go-llm-project-structure)](https://goreportcard.com/report/github.com/sploitzberg/go-llm-project-structure)
[![Go 1.26](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](https://go.dev/doc/go1.26)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## Why Mosaic

Agents drown in **stateless** calls and **flat** similarity search. Real work needs **durable** memory, **governed** writes, and **orchestrated** context: what to retrieve, how to expand, when to pack a prompt, and what the operator allows to touch disk.

**Mosaic** is a **local MCP server** that sits between your agent and **[HexxlaDB](https://github.com/hexxla/hexxladb)** — an embedded engine built for **structured** memory on a **hex grid**: spatial ring walks, **hybrid** semantic + lexical + tag filters, **token- or byte-budgeted** context packs, explicit **seams** for contradictions, **facets** and **edges** for relationships, and **MVCC** when you need honest snapshots.

You get **one** coherent layer:

| Capability | What it means for your agent |
| ---------- | ------------------------------ |
| **Orchestrated retrieval** | Embedding search, structured queries, and lexical search — composed, not three unrelated APIs |
| **Context packs** | Expand from seed coordinates on the lattice inside a **budget** — not blind top‑K truncation |
| **Operator policy** | YAML at startup: **what** chat turns may be persisted, **whether** enforcement blocks bad writes, **whether** deletes are allowed |
| **Retrieval metering (optional cap)** | Per-call **rings and byte budgets** shape each answer; optional YAML **`retrieval`** adds a **cumulative session egress** ceiling plus **`mosaic_hexxla_retrieval_budget_status`** for observability — see [MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md) *(session cap is a runaway brake, not a relevance filter)* |
| **Contradiction-aware memory** | Seams surface disagreement; supersession chains evolve preferences without silent overwrite |
| **Privacy-first deployment** | Runs **localhost** MCP over **your** file; optional **AES‑XTS** encryption for the database |
| **Open protocol** | **[Model Context Protocol](https://modelcontextprotocol.io/)** — wire your favorite MCP-capable client |

Mosaic is opinionated: **memory is not “whatever embeddings matched.”** It is **placement**, **provenance**, **policy**, and **assembly** — the same ideas serious retrieval stacks use, packaged so **agents** can call tools instead of reinventing glue code.

---

## The contrast

| Without a layer like Mosaic | With Mosaic |
| --------------------------- | ----------- |
| RAG = nearest vectors only | **Hybrid** retrieval + filters + ring-shaped context expansion |
| No contract for what may be written | **Policy** at boot: capture mode, enforcement, delete gates |
| Context = cut at N tokens | **Budgeted** packs with seeds from real hits |
| Conflicts vanish in the next embed | **Seams** and **supersession** stay visible to the model |
| Secrets in chat | **Passphrase / env / encrypted file** options for the DB |

---

## Quick start

**Requirements:** Go **1.26+**, and **Ollama** if you use embedding-heavy tools or **`mosaic-seed`**.

```bash
git clone https://github.com/sploitzberg/go-llm-project-structure.git
cd go-llm-project-structure

# Optional: empty HexxlaDB — default path .tmp/mosaic-seed.hexxla, or name it:
make create-db MOSAIC_CREATE_DB_FLAGS='-name myworkspace'

# Or seed demo data (needs Ollama; skips if that file exists — `make reseed` replaces it)
make seed

# Run MCP — use the same -name / MOSAIC_DB_PATH as create-db so the client opens that file:
make run-mosaic-mcp MOSAIC_MCP_FLAGS='-policy configs/config.yaml -name myworkspace'
```

Point your MCP client at **`http://127.0.0.1:8787/mcp`** (defaults from the Makefile). **[configs/config.yaml](configs/config.yaml)** is an example policy file: **`retention`** (capture mode / enforcement / deletes), optional **`database.passphrase`**, and optional **`retrieval`** (session metering — commented defaults leave the hard cap **off**).

See **[docs/mosaic/DATABASE_CREATION.md](docs/mosaic/DATABASE_CREATION.md)** for encrypted databases, **`-replace`** / **`-force`**, and **`MOSAIC_CREATE_DB_FLAGS`**, **`MOSAIC_SEED_FLAGS`**, **`MOSAIC_MCP_FLAGS`**.

---

## Commands (overview)

| Binary | Role |
| ------ | ---- |
| **`mosaic-mcp`** | Serve MCP tools; DB path via **`MOSAIC_DB_PATH`** or **`-db`** / **`-name`**; **`-policy`** / **`MOSAIC_POLICY_FILE`** for YAML |
| **`mosaic-create-db`** | Create empty DB (layout flags + optional encryption) |
| **`mosaic-seed`** | Create + optional demo corpus + Ollama embeddings |

Full CLI flags: **`go run ./cmd/<name> -help`**.

---

## Documentation

| Doc | Contents |
| --- | -------- |
| [docs/mosaic/DATABASE_CREATION.md](docs/mosaic/DATABASE_CREATION.md) | Creating DBs, encryption, Make wrappers |
| [docs/mosaic/PERSISTENCE_POLICY.md](docs/mosaic/PERSISTENCE_POLICY.md) | YAML retention, MCP policy tool |
| [docs/mosaic/MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md) | Agent workflow, context packs, optional retrieval budgeting |
| [configs/config.yaml](configs/config.yaml) | Example **`retention`** / **`retrieval`** / **`allow_delete_cell`** (comments inline) |
| [AGENTS.md](AGENTS.md) | Instructions for AI coding assistants (architecture, CI, doc index) |
| [TODOS.md](TODOS.md) | Active / pending session notes |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Roadmap themes and out-of-scope boundaries |
| [CHANGELOG.md](CHANGELOG.md) | Notable shipped changes |
| [docs/architecture/architecture.md](docs/architecture/architecture.md) | Dependency layout and design overview |

---

## Development

```bash
make ci          # full pipeline (same as CI)
make test        # unit tests
make integration # tagged integration tests
```

Quality checks (tests, lint, dependency policy) run in CI — same targets locally via **`make ci`**.

---

## Roadmap and rough edges

Exploration items and known limitations (e.g. **oversized one-shot** under session cap, meter lifetime) are tracked in **[TODOS.md](TODOS.md)**; higher-level themes live in **[docs/ROADMAP.md](docs/ROADMAP.md)**.

---

## Assistant / agent workflow (optional)

This repo includes machine-readable workflow hints for tools that call Mosaic MCP—tool roles, retrieval chaining, and when to use **`mosaic_hexxla_load_context_pack`**. The canonical copy lives at **[`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc)** (Markdown with optional front matter). You can lift the content into **whatever rule or instructions format your editor or agent host supports** (rules files, project docs, or MCP client configuration)—nothing here assumes a specific IDE.

For the full narrative, see **[docs/mosaic/MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md)**.

---

## License

MIT — see [LICENSE](LICENSE).
