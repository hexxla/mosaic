<div align="center">

<img src="assets/images/mosaic_logo_shadow.svg" alt="Mosaic" width="240">

# Mosaic

**Local MCP server for structured agent memory — hex lattice, hybrid retrieval, governed writes, and budgeted context — backed by [HexxlaDB](https://github.com/hexxla/hexxladb).**

[![CI](https://github.com/sploitzberg/mosaic/actions/workflows/ci.yml/badge.svg)](https://github.com/sploitzberg/mosaic/actions/workflows/ci.yml)
[![Integration](https://github.com/sploitzberg/mosaic/actions/workflows/integration.yml/badge.svg)](https://github.com/sploitzberg/mosaic/actions/workflows/integration.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sploitzberg/mosaic.svg)](https://pkg.go.dev/github.com/sploitzberg/mosaic)
[![Go Report Card](https://goreportcard.com/badge/github.com/sploitzberg/mosaic)](https://goreportcard.com/report/github.com/sploitzberg/mosaic)
[![Go 1.26](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](https://go.dev/doc/go1.26)
[![Version](https://img.shields.io/github/v/tag/sploitzberg/mosaic?label=version&color=7c3aed)](https://github.com/sploitzberg/mosaic/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

</div>

---

## Get started

**You need:** [Go 1.26+](https://go.dev/dl/). **Run [Ollama](https://ollama.com)** on your machine (defaults match the [Makefile](Makefile): **`MOSAIC_OLLAMA_URL`**, **`MOSAIC_EMBED_MODEL`**). Mosaic calls it for **`mosaic_hexxla_search_embedding`**, hybrid **`embed_query_text`** on structured and lexical search tools, **`mosaic_hexxla_put_embedding`** when given text (not raw vectors), and for **`mosaic-seed`** embeddings.

1. **Clone and enter the repo**
   ```bash
   git clone https://github.com/sploitzberg/mosaic.git
   cd mosaic
   ```

2. **Create a database** — you choose where the **`.hexxla`** file lives:
   - **`-db path/to/file.hexxla`** — full path (relative or absolute).
   - **`-name myworkspace`** — writes **`<dir>/myworkspace.hexxla`**. **`<dir>`** is **`-db-dir`**, or **`MOSAIC_DB_DIR`**, or **`.tmp`** when you only pass **`-name`**.
   - If you omit **`-db`**, **`-name`**, and **`MOSAIC_DB_PATH`**, **`mosaic-create-db`** and **`mosaic-seed`** create **`mosaic.hexxla` in the shell’s current working directory** (typically wherever you ran the command).

   **Empty database (fast, no corpus):** `go run ./cmd/mosaic-create-db -name myworkspace`, or **`make create-db`** (the Makefile sets **`MOSAIC_DB_PATH`** — override with **`MOSAIC_DB_PATH=/path/to/db.hexxla`**).

   **Seed demo corpus** *(development samples only — fills a demo lattice + embeddings; requires Ollama)* — **`make seed`** skips if the DB file already exists; **`make reseed`** replaces it. Use **`MOSAIC_DB_PATH`** or **`go run ./cmd/mosaic-seed -db …`** to put the file wherever you want.

3. **Start the MCP server** using the **same** path resolution (**`-db`**, **`-name`** / **`MOSAIC_DB_DIR`**, or **`MOSAIC_DB_PATH`**):
   ```bash
   make run-mosaic-mcp MOSAIC_MCP_FLAGS='-policy configs/config.yaml -name myworkspace'
   ```

4. **Attach your MCP client** to **`http://127.0.0.1:8787/mcp`** (defaults from the [Makefile](Makefile): **`MOSAIC_MCP_ADDR`**, **`MOSAIC_MCP_PATH`**).

**Policy YAML:** [configs/config.yaml](configs/config.yaml) is a minimal example. All keys → **[docs/mosaic/MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md)**. Paths, **`MOSAIC_DB_*`**, **`-replace`**, **encrypted databases**, and matching **`mosaic-mcp`** config → **[docs/mosaic/DATABASE_CREATION.md](docs/mosaic/DATABASE_CREATION.md)**.

---

## Features

| Area | What Mosaic does |
| ---- | ---------------- |
| **Hex lattice memory** | Cells live on axial coordinates **{q,r}**; retrieve by walking **rings** and neighbourhoods, not only flat nearest-neighbour vectors. |
| **Hybrid retrieval** | **Semantic** (**`mosaic_hexxla_search_embedding`**), **structured filters** (**`mosaic_hexxla_query_cells`**), **lexical** (**`mosaic_hexxla_search_cells`**) — compose discovery before expanding context. |
| **Context packs** | **`mosaic_hexxla_load_context_pack`** expands from **seed** coords under a **byte or token budget** — structured neighbourhoods + optional seams, not blind top‑K truncation. |
| **Persistence policy** | YAML **`retention`** (what turn kinds to save, **enforcement**, agent-facing **`notes`**) loaded at boot; **`mosaic_hexxla_get_persistence_policy`** exposes it to the model. |
| **Writes & embeddings** | **`mosaic_hexxla_put_cell`**, **`mosaic_hexxla_put_embedding`**; optional **`allow_delete_cell`** + **`mosaic_hexxla_delete_cell`**. |
| **MVCC & disk hygiene** | HexxlaDB **MVCC** for honest versioning; optional **`database.auto_maintain_after_cell_delete`** (prune + compact, debounced) + shallow **`mvcc_retain_commits_behind_head`** — see [MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md). |
| **Health & footprint** | **`mosaic_hexxla_health`** — cells, seams, index checks, **`mvcc`** stats, **`disk`** (primary + WAL bytes), effective **`mvcc_retain_commits_behind_head`**. |
| **Contradictions** | **Seams**: **`mosaic_hexxla_find_seams`**, **`mosaic_hexxla_mark_conflict`**, **`mosaic_hexxla_mark_supersedes`**, **`mosaic_hexxla_resolve_seam`** — disagreements stay explicit. |
| **Facets & edges** | **`mosaic_hexxla_put_facet`**, **`mosaic_hexxla_link_cells`**, reads **`get_facet`**, **`list_facets`**, **`get_edge`**, **`list_edges_from`** — lightweight graph over cells. |
| **Tags** | **`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`** for taxonomy-aware writes. |
| **Retrieval metering** | Optional YAML **`retrieval`** session cap + **`mosaic_hexxla_retrieval_budget_status`** (egress brake, not a relevance filter — see [MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md)). |
| **Privacy** | Runs **localhost** [MCP](https://modelcontextprotocol.io); optional **AES‑XTS**-style encryption for the DB (passphrase / env — [DATABASE_CREATION](docs/mosaic/DATABASE_CREATION.md)). |
| **Budget helper** | **`mosaic_hexxla_estimate_context_budget_bytes`** for planning pack sizes. |

Under the hood: **[HexxlaDB](https://github.com/hexxla/hexxladb)** — embedded store with **B+tree**, optional **HNSW** for embeddings, **MVCC**, **DEFLATE** for large values.

---

## MCP tools (quick map)

| Cluster | Tools |
| ------- | ----- |
| **Reads** | **`mosaic_hexxla_health`**, **`mosaic_hexxla_query_cells`**, **`mosaic_hexxla_search_cells`**, **`mosaic_hexxla_search_embedding`**, **`mosaic_hexxla_load_context_pack`**, **`mosaic_hexxla_estimate_context_budget_bytes`**, **`mosaic_hexxla_retrieval_budget_status`** |
| **Writes** | **`mosaic_hexxla_put_cell`**, **`mosaic_hexxla_put_embedding`**, **`mosaic_hexxla_delete_cell`** |
| **Policy** | **`mosaic_hexxla_get_persistence_policy`** |
| **Seams** | **`mosaic_hexxla_find_seams`**, **`mosaic_hexxla_mark_conflict`**, **`mosaic_hexxla_mark_supersedes`**, **`mosaic_hexxla_resolve_seam`** |
| **Facets / edges** | **`mosaic_hexxla_put_facet`**, **`mosaic_hexxla_link_cells`**, **`mosaic_hexxla_get_facet`**, **`mosaic_hexxla_list_facets`**, **`mosaic_hexxla_get_edge`**, **`mosaic_hexxla_list_edges_from`** |
| **Tags** | **`mosaic_hexxla_list_tags`**, **`mosaic_hexxla_tag_counts`** |

Full wiring and agent narrative → **[docs/mosaic/MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md)** · API ↔ coverage → **[docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md](docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md)**

---

## Why Mosaic

Agents often get **stateless** tool loops and **flat** similarity search. Mosaic is **opinionated**: memory is **placement** on a lattice, **provenance**, **operator policy**, and **assembled** context — not “whatever vectors matched.” Same ideas as serious retrieval stacks, exposed as **[Model Context Protocol](https://modelcontextprotocol.io/)** tools so agents call one coherent layer instead of bespoke glue.

---

## Compared to “RAG only”

| Without a layer like Mosaic | With Mosaic |
| --------------------------- | ----------- |
| Nearest vectors only | **Hybrid** semantic + structured query + lexical search |
| No contract for writes | **YAML policy**: capture mode, enforcement, delete gates |
| Truncate context at N tokens | **Budgeted** packs from **seed** coordinates |
| Conflicts disappear on re-embed | **Seams** + **supersession** stay visible |
| Secrets only in chat | **Passphrase / env / encrypted DB** options |

---

## Configuration

Load optional policy YAML at **`mosaic-mcp`** startup: **`-policy PATH`** or **`MOSAIC_POLICY_FILE`**. **`version:`** must be **`1`**.

**Keys:** **`retention`**, **`allow_delete_cell`**, optional **`retrieval`**, optional **`ollama`** (**`base_url`**, **`embed_model`** — overrides Makefile / env **`MOSAIC_OLLAMA_*`** when set), optional **`database`** (encryption hint, MVCC retention, post-delete maintenance).

**References:** **[docs/mosaic/MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md)** · **[docs/mosaic/PERSISTENCE_POLICY.md](docs/mosaic/PERSISTENCE_POLICY.md)** · **[configs/config.yaml](configs/config.yaml)**

---

## Commands

| Binary | Role |
| ------ | ---- |
| **`mosaic-mcp`** | MCP server — DB via **`MOSAIC_DB_PATH`** or **`-db`** / **`-name`**; policy via **`-policy`** / **`MOSAIC_POLICY_FILE`** |
| **`mosaic-create-db`** | Create empty HexxlaDB (layout + optional encryption) |
| **`mosaic-seed`** | Create DB + optional seeded corpus + Ollama embeddings |

```bash
go run ./cmd/mosaic-mcp -help
go run ./cmd/mosaic-create-db -help
go run ./cmd/mosaic-seed -help
```

Make wrappers: **`make run-mosaic-mcp`**, **`make create-db`**, **`make seed`**, **`make ci`**.

---

## Documentation

| Doc | Contents |
| --- | -------- |
| [docs/mosaic/MOSAIC_CONFIG.md](docs/mosaic/MOSAIC_CONFIG.md) | Policy YAML reference (`retention`, **`database`** encryption hint, MVCC, post-delete maintenance) |
| [configs/config.yaml](configs/config.yaml) | Minimal example policy |
| [docs/mosaic/DATABASE_CREATION.md](docs/mosaic/DATABASE_CREATION.md) | DB paths (cwd default), **encrypted DB + MCP**, **`-replace`**, Make flags |
| [docs/mosaic/PERSISTENCE_POLICY.md](docs/mosaic/PERSISTENCE_POLICY.md) | Retention semantics, **`mosaic_hexxla_get_persistence_policy`** |
| [docs/mosaic/MCP_AGENT_BLUEPRINT.md](docs/mosaic/MCP_AGENT_BLUEPRINT.md) | Agent workflows, retrieval vs budgeting |
| [docs/mosaic/HEXXLA_TROUBLESHOOTING.md](docs/mosaic/HEXXLA_TROUBLESHOOTING.md) | Deletes, **`integrity_ok`**, disk / MVCC |
| [docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md](docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md) | Hexxla ↔ MCP capability matrix |
| [docs/mosaic/IMPLEMENTATION_PLAN.md](docs/mosaic/IMPLEMENTATION_PLAN.md) | Phased MCP exposure checklist + session log |
| [docs/mosaic/MCP_BLUEPRINT.md](docs/mosaic/MCP_BLUEPRINT.md) | Local MCP server architecture (hexagonal) |
| [docs/mosaic/MOSAIC.md](docs/mosaic/MOSAIC.md) | Broader lattice-memory vision (aspirational; shipped stack is MCP + HexxlaDB) |
| [docs/mosaic/AGENT_CLIENT_WORKFLOWS.md](docs/mosaic/AGENT_CLIENT_WORKFLOWS.md) | Cursor, Windsurf, rules & workflows |
| [README.md#reliable-tool-use-from-agents](#reliable-tool-use-from-agents) | Rules, **`retention.notes`**, best practices |
| [AGENTS.md](AGENTS.md) | Contributor architecture guide |
| [CHANGELOG.md](CHANGELOG.md) | Shipped changes |
| [docs/architecture/architecture.md](docs/architecture/architecture.md) | Dependency layout |

---

## Reliable tool use from agents

MCP **does not force** models to call tools — reinforce with **project rules**, **`retention.notes`** in YAML (injected into server instructions), and **[`.cursor/rules/mosaic-mcp-agent.mdc`](.cursor/rules/mosaic-mcp-agent.mdc)** or **[docs/mosaic/AGENT_CLIENT_WORKFLOWS.md](docs/mosaic/AGENT_CLIENT_WORKFLOWS.md)**. Typical flow: discover with **`search_embedding`** / **`query_cells`** → expand **`load_context_pack`** → **`put_cell`** when policy requires persistence.

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

## License

MIT — see [LICENSE](LICENSE).
