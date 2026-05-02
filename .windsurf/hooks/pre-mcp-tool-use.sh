#!/usr/bin/env bash
# Mosaic-specific MCP tool guidance hook
# Injects Mosaic-specific guidance when calling MCP tools

set -euo pipefail

CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Get MCP tool info from input
mcp_server_name=$(echo "$1" | jq -r '.mcp_server_name // empty' 2>/dev/null || echo "")
mcp_tool_name=$(echo "$1" | jq -r '.mcp_tool_name // empty' 2>/dev/null || echo "")

# Only process mosaic MCP tools
if [[ "$mcp_server_name" != "mosaic" ]] && [[ "$mcp_server_name" != "mosaic-mcp" ]]; then
    exit 0
fi

echo -e "${CYAN}=== MOSAIC MCP TOOL GUIDANCE ===${NC}"
echo ""
echo "Tool: $mcp_tool_name"
echo ""

# Provide tool-specific guidance
case "$mcp_tool_name" in
    mosaic_hexxla_put_cell)
        echo "Prerequisites:"
        echo "- Call mosaic_hexxla_list_tags or mosaic_hexxla_tag_counts before put_cell"
        echo "- Call a retrieval tool (search_embedding, query_cells, search_cells) before put_cell"
        echo ""
        echo "Tag Hygiene:"
        echo "- Review available tags to prevent fragmentation"
        echo "- Reuse existing tags when relevant"
        echo "- Use semantic tags: fact, opinion, idea, code, signal, task, project, etc."
        echo ""
        echo "Required Fields:"
        echo "- q, r: Hex coordinates (axial)"
        echo "- raw_content: Cell body text"
        echo "- source_id: Provenance source or session id"
        echo "- confidence: 0..1"
        ;;
    mosaic_hexxla_list_tags | mosaic_hexxla_tag_counts)
        echo "Purpose:"
        echo "- Review available tags before writing cells"
        echo "- Understand tag frequency and usage patterns"
        echo ""
        echo "Use Before:"
        echo "- mosaic_hexxla_put_cell (for tag hygiene)"
        ;;
    mosaic_hexxla_search_embedding | mosaic_hexxla_query_cells | mosaic_hexxla_search_cells)
        echo "Purpose:"
        echo "- Retrieve cells by semantic similarity or structured filters"
        echo "- Locate relevant cells before writes"
        echo ""
        echo "Use Before:"
        echo "- mosaic_hexxla_put_cell (to verify cell existence and context)"
        echo "- mosaic_hexxla_put_embedding (to verify cell exists)"
        echo "- mosaic_hexxla_delete_cell (to verify cell exists)"
        ;;
    mosaic_hexxla_load_context_pack)
        echo "Prerequisites:"
        echo "- Call mosaic_hexxla_estimate_context_budget_bytes before large context loads"
        echo ""
        echo "Purpose:"
        echo "- Expand hex-neighbourhood context from seed coordinates"
        echo "- Retrieve adjacent lattice context (seams, neighbours)"
        ;;
    mosaic_hexxla_mark_conflict | mosaic_hexxla_mark_supersedes | mosaic_hexxla_resolve_seam)
        echo "Purpose:"
        echo "- Manage seams (conflicts between cells)"
        echo ""
        echo "Use After:"
        echo "- Retrieval tools to understand current state"
        ;;
    mosaic_hexxla_put_facet)
        echo "Purpose:"
        echo "- Write derived content to facet slots"
        echo ""
        echo "Use After:"
        echo "- Retrieval tools to verify cell exists"
        ;;
    mosaic_hexxla_link_cells)
        echo "Purpose:"
        echo "- Create relationships between cells"
        echo ""
        echo "Use After:"
        echo "- Retrieval tools to verify both cells exist"
        ;;
esac

echo ""
echo "Ratchet Validation:"
echo "- Tool prerequisites are enforced by ratchet service"
echo "- One-time use tokens may require fresh prerequisite calls"
echo ""
echo -e "${CYAN}=== END GUIDANCE ===${NC}"

exit 0
