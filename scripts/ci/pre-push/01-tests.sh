#!/usr/bin/env bash
# scripts/ci/pre-push/01-tests.sh
# Run full test suite

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} go not found"
    exit 1
fi
echo -e "${CYAN}> Running full test suite${NC}"

go test ./...
echo -e "${GREEN}Tests: OK${NC}"
echo
