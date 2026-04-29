#!/usr/bin/env bash
# scripts/ci/pre-commit/05-gosec.sh
# Run gosec security scan

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v gosec >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} gosec not found. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    exit 1
fi

echo -e "${CYAN}> Running gosec security scan${NC}"

gosec -quiet ./...
echo -e "${GREEN}gosec: OK${NC}"
echo
