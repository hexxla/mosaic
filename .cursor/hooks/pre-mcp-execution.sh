#!/usr/bin/env bash
# Cursor pre-mcp-execution hook
# Provides mosaic MCP tool guidance

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Read JSON input from stdin
input=$(cat)

# Parse MCP tool info from cursor input
mcp_server_name=$(echo "$input" | jq -r '.mcp_server_name // empty' 2>/dev/null || echo "")
mcp_tool_name=$(echo "$input" | jq -r '.mcp_tool_name // empty' 2>/dev/null || echo "")

# Only process mosaic MCP tools
if [[ "$mcp_server_name" != "mosaic" ]] && [[ "$mcp_server_name" != "mosaic-mcp" ]]; then
    echo '{"continue": true, "permission": "allow"}'
    exit 0
fi

# Run the windsurf pre-mcp-tool-use hook
if [ -f ".windsurf/hooks/pre-mcp-tool-use.sh" ]; then
    ./.windsurf/hooks/pre-mcp-tool-use.sh "{\"mcp_server_name\": \"$mcp_server_name\", \"mcp_tool_name\": \"$mcp_tool_name\"}" 2>/dev/null || true
fi

# Allow the action
echo '{"continue": true, "permission": "allow"}'

exit 0
