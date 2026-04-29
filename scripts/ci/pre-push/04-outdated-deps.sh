#!/usr/bin/env bash
# scripts/ci/pre-push/04-outdated-deps.sh
# Check for outdated Go dependencies

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} go not found"
    exit 1
fi

echo -e "${CYAN}> Checking for outdated dependencies${NC}"

# Only fail on outdated *direct* modules (not every transitive in the build graph).
outdated=$(go list -m -u -f '{{if and .Update (not .Indirect)}}{{.Path}} {{.Version}} [{{.Update.Version}}]{{end}}' all 2>/dev/null | grep '.' || true)

if [ -n "$outdated" ]; then
    echo -e "${RED}error:${NC} Outdated dependencies found:"
    echo "$outdated"
    echo ""
    echo "Update with: go get -u ./..."
    echo "Or run: go list -u -m all"
    exit 1
fi

echo -e "${GREEN}Dependencies: OK${NC}"
echo
