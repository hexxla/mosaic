#!/usr/bin/env bash
# scripts/ci/pre-commit/03-tests.sh
# Run tests with configurable flags

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} go not found"
    exit 1
fi

echo -e "${CYAN}> Running tests${NC}"

# Use GO_TEST_FLAGS env var or default to -short for pre-commit
FLAGS=${GO_TEST_FLAGS:--short}

go test $FLAGS ./...
echo -e "${GREEN}Tests: OK${NC}"
echo
