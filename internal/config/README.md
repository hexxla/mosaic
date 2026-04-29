# Config Layer

Centralized configuration structures and loading logic.

**MCP (Streamable HTTP):** `LoadMCPFromEnv` reads `MOSAIC_MCP_ADDR` (default `127.0.0.1:8787`) and `MOSAIC_MCP_PATH` (default `/mcp`). Only loopback listen addresses are accepted.

## Best Practices

- Define configuration as structs with validation
- Support multiple sources (env vars, YAML, flags, etc.)
- Keep configuration simple and immutable where possible
- Used primarily by primary adapters (e.g. `main.go`)
