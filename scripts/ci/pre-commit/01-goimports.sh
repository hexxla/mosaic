#!/usr/bin/env bash
# scripts/ci/pre-commit/01-goimports.sh
# Check import ordering with goimports

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v goimports >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} goimports not found (should be included with Go)"
    exit 1
fi

echo -e "${CYAN}> Running goimports check${NC}"

if output=$(goimports -l . 2>&1) && [ -n "$output" ]; then
    echo -e "${RED}error:${NC} The following files need goimports:"
    echo "$output"
    echo "Fix with: goimports -l -w ."
    exit 1
fi

echo -e "${GREEN}goimports: OK${NC}"
echo
