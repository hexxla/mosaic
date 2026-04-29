#!/usr/bin/env bash
# scripts/llm/hooks/pre-read-context-injection.sh
# Pre-read hook: Inject architecture context before LLM reads code

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT"

file_path=$(echo "$1" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ -z "$file_path" ]]; then
    exit 0
fi

# Determine which layer the file is in
layer=""
if [[ "$file_path" == internal/core/domain/* ]]; then
    layer="domain"
elif [[ "$file_path" == internal/core/ports/* ]]; then
    layer="port"
elif [[ "$file_path" == internal/core/services/* ]]; then
    layer="service"
elif [[ "$file_path" == internal/adapter/* ]]; then
    layer="adapter"
elif [[ "$file_path" == internal/config/* ]]; then
    layer="config"
fi

if [[ -z "$layer" ]]; then
    exit 0
fi

# Output context for the LLM
echo "=== ARCHITECTURE CONTEXT ==="
echo "File Layer: $layer"
echo ""

case "$layer" in
    domain)
        echo "DOMAIN LAYER RULES:"
        echo "- Must have ZERO internal imports (only standard library)"
        echo "- Pure business entities and value objects"
        echo "- No framework dependencies"
        echo "- No dependencies on core/ports/, core/services/, or adapter/"
        echo ""
        echo "Related Documentation: internal/core/domain/README.md"
        ;;
    port)
        echo "PORT LAYER RULES:"
        echo "- Defines interfaces (contracts)"
        echo "- Can only depend on core/domain/"
        echo "- Must NOT depend on adapter/ or core/services/"
        echo "- Primary ports: inbound use cases"
        echo "- Secondary ports: outbound dependencies"
        echo ""
        echo "Related Documentation: internal/core/ports/README.md"
        ;;
    service)
        echo "SERVICE LAYER RULES:"
        echo "- Implements application use cases"
        echo "- Can only depend on core/domain/ and core/ports/"
        echo "- Must NOT depend on adapter/"
        echo "- Orchestrates domain logic"
        echo ""
        echo "Related Documentation: internal/core/services/README.md"
        ;;
    adapter)
        echo "ADAPTER LAYER RULES:"
        echo "- Concrete implementations of ports"
        echo "- Primary adapters: HTTP, gRPC, CLI (call primary ports)"
        echo "- Secondary adapters: Database, APIs (implement secondary ports)"
        echo "- Can depend on core/ports/ and core/domain/"
        echo ""
        echo "Related Documentation: internal/adapter/README.md"
        ;;
    config)
        echo "CONFIG LAYER RULES:"
        echo "- Configuration structures and loading"
        echo "- No dependencies on other internal packages"
        echo ""
        echo "Related Documentation: internal/config/README.md"
        ;;
esac

echo ""
echo "=== DEPENDENCY RULES ==="
echo "Allowed dependencies for $layer layer:"
case "$layer" in
    domain)
        echo "- Standard library only"
        ;;
    port)
        echo "- Standard library"
        echo "- internal/core/domain/"
        ;;
    service)
        echo "- Standard library"
        echo "- internal/core/domain/"
        echo "- internal/core/ports/"
        ;;
    adapter)
        echo "- Standard library"
        echo "- internal/core/domain/"
        echo "- internal/core/ports/"
        ;;
    config)
        echo "- Standard library"
        ;;
esac

echo ""
echo "=== END CONTEXT ==="
