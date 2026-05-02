#!/usr/bin/env bash
# Mosaic-specific context injection for file reads
# Injects Mosaic-specific context when reading mosaic-related files

set -euo pipefail

CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

file_path=$(echo "$1" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ -z "$file_path" ]]; then
    exit 0
fi

# Check if file is mosaic-related
mosaic_related=false
if [[ "$file_path" == cmd/mosaic-* ]] || \
   [[ "$file_path" == internal/adapter/primary/mcpsrv/* ]] || \
   [[ "$file_path" == internal/core/domain/* ]] || \
   [[ "$file_path" == internal/core/ports/* ]] || \
   [[ "$file_path" == internal/core/services/* ]] || \
   [[ "$file_path" == internal/adapter/* ]] || \
   [[ "$file_path" == internal/config/* ]] || \
   [[ "$file_path" == configs/* ]] || \
   [[ "$file_path" == docs/mosaic/* ]] || \
   [[ "$file_path" == docs/architecture/* ]]; then
    mosaic_related=true
fi

if [[ "$mosaic_related" == false ]]; then
    exit 0
fi

echo -e "${CYAN}=== MOSAIC FILE CONTEXT ===${NC}"
echo "File: $file_path"
echo ""

# Determine which part of mosaic this is
if [[ "$file_path" == cmd/mosaic-* ]]; then
    echo "Mosaic Entry Point"
    echo "- Main entry points for mosaic binaries"
    echo "- mosaic-mcp: MCP server for HexxlaDB"
    echo "- mosaic-create-db: Database initialization"
    echo ""
elif [[ "$file_path" == internal/adapter/primary/mcpsrv/* ]]; then
    echo "MCP Server Adapter (Primary)"
    echo "- Implements MCP tool handlers"
    echo "- Wraps HexxlaDB operations with MCP protocol"
    echo "- Integrates ratchet validation"
    echo ""
elif [[ "$file_path" == internal/core/domain/* ]]; then
    echo "Domain Layer"
    echo "- Pure business entities and value objects"
    echo "- Zero internal dependencies (only standard library)"
    echo "- Core hex lattice and seam concepts"
    echo ""
elif [[ "$file_path" == internal/core/ports/* ]]; then
    echo "Port Layer"
    echo "- Interface definitions (contracts)"
    echo "- Primary ports: inbound use cases"
    echo "- Secondary ports: outbound dependencies"
    echo ""
elif [[ "$file_path" == internal/core/services/* ]]; then
    echo "Service Layer"
    echo "- Application use case implementations"
    echo "- Orchestrates domain logic"
    echo "- Can depend on domain and ports only"
    echo ""
elif [[ "$file_path" == internal/adapter/* ]]; then
    echo "Adapter Layer"
    echo "- Concrete implementations of ports"
    echo "- Primary adapters: HTTP, gRPC, CLI"
    echo "- Secondary adapters: Database, APIs, cache"
    echo ""
elif [[ "$file_path" == internal/config/* ]]; then
    echo "Config Layer"
    echo "- Configuration structures and loading"
    echo "- Mosaic runtime configuration"
    echo "- Ratchet configuration"
    echo ""
elif [[ "$file_path" == configs/* ]]; then
    echo "Configuration Files"
    echo "- YAML configuration for mosaic"
    echo "- ratchet.yaml: Tool flow governance rules"
    echo "- config.yaml: Runtime configuration"
    echo ""
elif [[ "$file_path" == docs/mosaic/* ]]; then
    echo "Mosaic Documentation"
    echo "- Detailed mosaic documentation"
    echo "- MCP tool documentation"
    echo "- Configuration guides"
    echo ""
elif [[ "$file_path" == docs/architecture/* ]]; then
    echo "Architecture Documentation"
    echo "- Hexagonal architecture guides"
    echo "- Design flow documentation"
    echo "- Dependency graphs"
    echo ""
fi

echo "Key Principles:"
echo "- Hexagonal Architecture (Ports & Adapters)"
echo "- All dependencies point inward toward domain"
echo "- External concerns pushed to adapter layer"
echo "- Core business logic remains pure"
echo ""
echo -e "${CYAN}=== END MOSAIC CONTEXT ===${NC}"

exit 0
