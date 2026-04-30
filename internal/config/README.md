# Config Layer

Centralized configuration structures and loading logic.

**MCP (Streamable HTTP):** `LoadMCPFromEnv` reads `MOSAIC_MCP_ADDR` (default `127.0.0.1:8787`) and `MOSAIC_MCP_PATH` (default `/mcp`). Only loopback listen addresses are accepted.

**Ollama:** **`ResolveOllama`** (`embed.go`) merges optional policy **`ollama:`** (**`base_url`**, **`embed_model`**) with **`MOSAIC_OLLAMA_URL`**, **`MOSAIC_EMBED_MODEL`**, and defaults (YAML overrides env for fields that are set in YAML; **`mosaic-seed`** flags override YAML).

**Database paths:** **`ResolveMosaicDBPath`** (`mosaic_db_path.go`) resolves **`MOSAIC_DB_PATH`**, optional flags **`-db`**, **`-name`**, **`-db-dir`**, and **`MOSAIC_DB_DIR`**. With **`defaultWhenUnset`** ( **`mosaic-create-db`** / **`mosaic-seed`** only), if nothing selects a path, the default file is **`mosaic.hexxla`** in the process working directory. **`LoadDBFromEnv`** still reads only **`MOSAIC_DB_PATH`** (tests and simple env-only use). **`cmd/mosaic-mcp`** uses **`ResolveMosaicDBPath`** so **`MOSAIC_DB_PATH`** is optional when **`-name`** or **`-db`** is set.

## Best Practices

- Define configuration as structs with validation
- Support multiple sources (env vars, YAML, flags, etc.)
- Keep configuration simple and immutable where possible
- Used primarily by primary adapters (e.g. `main.go`)
