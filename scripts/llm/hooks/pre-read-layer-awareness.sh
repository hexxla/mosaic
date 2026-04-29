#!/usr/bin/env bash
# scripts/llm/hooks/pre-read-layer-awareness.sh
# Pre-read hook: Detect layer and provide layer-specific guidance

set -euo pipefail

CYAN='\033[0;36m'
NC='\033[0m'

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
elif [[ "$file_path" == internal/core/ports/primary/* ]]; then
    layer="port-primary"
elif [[ "$file_path" == internal/core/ports/secondary/* ]]; then
    layer="port-secondary"
elif [[ "$file_path" == internal/core/services/* ]]; then
    layer="service"
elif [[ "$file_path" == internal/adapter/primary/* ]]; then
    layer="adapter-primary"
elif [[ "$file_path" == internal/adapter/secondary/* ]]; then
    layer="adapter-secondary"
elif [[ "$file_path" == internal/config/* ]]; then
    layer="config"
fi

if [[ -z "$layer" ]]; then
    exit 0
fi

echo -e "${CYAN}=== LAYER AWARENESS ===${NC}"
echo "Current layer: $layer"
echo ""

case "$layer" in
    domain)
        echo "Focus: Pure business logic"
        echo "Pattern: Value objects, entities, domain rules"
        echo "No: Frameworks, external dependencies"
        ;;
    port-primary)
        echo "Focus: Use case interfaces"
        echo "Pattern: Service interfaces defining what app can do"
        echo "No: Implementation details"
        ;;
    port-secondary)
        echo "Focus: Dependency interfaces"
        echo "Pattern: Repository interfaces for external systems"
        echo "No: Concrete implementations"
        ;;
    service)
        echo "Focus: Application orchestration"
        echo "Pattern: Use case implementation, workflow logic"
        echo "Uses: Domain entities, ports"
        ;;
    adapter-primary)
        echo "Focus: Entry point implementation"
        echo "Pattern: HTTP handlers, gRPC servers, CLI"
        echo "Calls: Primary ports"
        ;;
    adapter-secondary)
        echo "Focus: Infrastructure implementation"
        echo "Pattern: Database repos, API clients, cache"
        echo "Implements: Secondary ports"
        ;;
    config)
        echo "Focus: Configuration"
        echo "Pattern: Config structs, loading logic"
        echo "No: Business logic"
        ;;
esac

echo ""
echo -e "${CYAN}=== END AWARENESS ===${NC}"
