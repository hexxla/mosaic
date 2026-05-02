#!/usr/bin/env bash
# Mosaic-specific context injection hook
# Injects Mosaic concepts and patterns before agent reasoning

set -euo pipefail

CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Check if the user prompt mentions mosaic-related keywords
user_prompt=$(echo "$1" | jq -r '.user_prompt // empty' 2>/dev/null || echo "")

if [[ -z "$user_prompt" ]]; then
    exit 0
fi

# Mosaic-related keywords to detect
mosaic_keywords=("hex" "lattice" "cell" "embedding" "facet" "edge" "seam" "tag" "mosaic" "hexxla" "ratchet" "mcp")

mosaic_mentioned=false
for keyword in "${mosaic_keywords[@]}"; do
    if echo "$user_prompt" | grep -qi "$keyword"; then
        mosaic_mentioned=true
        break
    fi
done

if [[ "$mosaic_mentioned" == false ]]; then
    exit 0
fi

echo -e "${CYAN}=== MOSAIC CONTEXT ===${NC}"
echo ""
echo "Mosaic is a local MCP server for structured agent memory:"
echo "- Hex lattice memory layout (not flat pile)"
echo "- Hybrid retrieval (semantic + structured + lexical)"
echo "- Governed writes with ratchet tool flow"
echo "- Budgeted context with byte/token limits"
echo ""
echo "Key Concepts:"
echo "- Hex coordinates (q,r) for spatial memory layout"
echo "- Cells contain raw content, tags, embeddings"
echo "- Facets are derived content slots per cell"
echo "- Edges link cells with relationships"
echo "- Seams record conflicts between cells"
echo ""
echo "Architecture:"
echo "- Hexagonal Architecture (Ports & Adapters)"
echo "- core/domain/ (pure business entities)"
echo "- core/ports/ (interfaces)"
echo "- core/services/ (use case implementations)"
echo "- adapter/ (concrete implementations)"
echo ""
echo "MCP Tools:"
echo "- mosaic_hexxla_* tools for cells, embeddings, facets, edges, seams, tags"
echo "- Ratchet validation enforces tool prerequisites"
echo "- Use list_tags/tag_counts before put_cell for tag hygiene"
echo ""
echo "Documentation:"
echo "- README.md - Quick start"
echo "- docs/mosaic/ - Detailed documentation"
echo "- docs/architecture/ - Hexagonal architecture guide"
echo ""
echo -e "${CYAN}=== END MOSAIC CONTEXT ===${NC}"

exit 0
