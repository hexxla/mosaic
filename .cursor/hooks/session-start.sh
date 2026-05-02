#!/usr/bin/env bash
# Cursor session-start hook
# Injects mosaic context at session start

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Read JSON input from stdin
input=$(cat)

# Parse session info
session_id=$(echo "$input" | jq -r '.session_id // empty' 2>/dev/null || echo "")
is_background_agent=$(echo "$input" | jq -r '.is_background_agent // false' 2>/dev/null || echo "false")

# Inject mosaic context as additional_context
additional_context="Mosaic is a local MCP server for structured agent memory with hex lattice layout, hybrid retrieval, governed writes with ratchet, and budgeted context. Key concepts: hex coordinates (q,r), cells, embeddings, facets, edges, seams, tags. Architecture: Hexagonal (Ports & Adapters) with core/domain/, core/ports/, core/services/, adapter/, config/ layers. MCP tools: mosaic_hexxla_* for cells, embeddings, facets, edges, seams, tags. Ratchet validation enforces tool prerequisites. See README.md and docs/mosaic/ for details."

# Return JSON with additional_context
cat <<EOF
{
  "continue": true,
  "permission": "allow",
  "additional_context": "$additional_context"
}
EOF

exit 0
