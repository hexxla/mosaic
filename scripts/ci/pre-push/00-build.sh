#!/usr/bin/env bash
# scripts/ci/pre-push/00-build.sh
# Build the project

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}error:${NC} go not found"
    exit 1
fi

echo -e "${CYAN}> Running go build${NC}"

# Detect host OS and architecture for output directory naming
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)
BINDIR="bin/${GOOS}-${GOARCH}"
EXE=""
if [[ "$GOOS" == "windows" ]]; then
    EXE=".exe"
fi

BINARY_NAME="go-llm-project-structure"

# Create output directory and build
mkdir -p "$BINDIR"
go build -o "${BINDIR}/${BINARY_NAME}${EXE}" ./cmd/go-llm-project-structure

echo -e "${GREEN}Build: OK${NC}"
echo
