#!/usr/bin/env bash
# scripts/ci/pre-commit/00-gofmt.sh
# Check Go formatting with gofmt

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v gofmt >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} gofmt not found (should be included with Go)"
    exit 1
fi

echo -e "${CYAN}> Running gofmt check${NC}"

if output=$(gofmt -l -s . 2>&1) && [ -n "$output" ]; then
    echo -e "${RED}error:${NC} The following files are not formatted:"
    echo "$output"
    echo "Fix with: gofmt -l -s -w ."
    exit 1
fi

echo -e "${GREEN}gofmt: OK${NC}"
echo
