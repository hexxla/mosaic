#!/usr/bin/env bash
# scripts/llm/hooks/pre-write-arch-compliance.sh
# Pre-write hook: Validate architectural compliance before writing changes

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT"

file_path=$(echo "$1" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ -z "$file_path" ]] || [[ "$file_path" != *.go ]]; then
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

# Get the content being written
content=$(echo "$1" | jq -r '.content // empty' 2>/dev/null || echo "")

if [[ -z "$content" ]]; then
    exit 0
fi

# Check for forbidden imports based on layer
violations=0

case "$layer" in
    domain)
        # Domain must not import any internal packages
        if echo "$content" | grep -q 'internal/'; then
            echo -e "${RED}VIOLATION${NC}: Domain layer must not import internal packages"
            violations=$((violations + 1))
        fi
        ;;
    port)
        # Port must not import adapter or services
        if echo "$content" | grep -q 'internal/adapter'; then
            echo -e "${RED}VIOLATION${NC}: Port layer must not import adapter/"
            violations=$((violations + 1))
        fi
        if echo "$content" | grep -q 'internal/core/services'; then
            echo -e "${RED}VIOLATION${NC}: Port layer must not import services/"
            violations=$((violations + 1))
        fi
        ;;
    service)
        # Service must not import adapter
        if echo "$content" | grep -q 'internal/adapter'; then
            echo -e "${RED}VIOLATION${NC}: Service layer must not import adapter/"
            violations=$((violations + 1))
        fi
        ;;
esac

# Check for framework imports in core layers
if [[ "$layer" == "domain" || "$layer" == "port" || "$layer" == "service" ]]; then
    # Common framework packages to avoid in core
    frameworks=("github.com/gin-gonic" "github.com/gorilla" "gorm.io" "entgo.io")
    for fw in "${frameworks[@]}"; do
        if echo "$content" | grep -q "$fw"; then
            echo -e "${YELLOW}WARNING${NC}: Framework import '$fw' in core layer ($layer)"
            violations=$((violations + 1))
        fi
    done
fi

if [[ $violations -gt 0 ]]; then
    echo -e "${RED}Architecture compliance check failed with $violations violation(s)${NC}"
    exit 1
fi

echo -e "${GREEN}Architecture compliance check passed${NC}"
exit 0
