#!/usr/bin/env bash
# scripts/ci/pre-commit/06-go-vet.sh
# Run go vet static analysis

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} go not found"
    exit 1
fi

echo -e "${CYAN}> Running go vet${NC}"

go vet ./...
echo -e "${GREEN}go vet: OK${NC}"
echo
echo
