#!/usr/bin/env bash
# scripts/ci/pre-push/02-hex-arch-guardrail.sh
# Check hexagonal architecture rules

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Checking hexagonal architecture rules${NC}"

if [[ -f ./scripts/ci/hex-arch-guardrail.sh ]]; then
    ./scripts/ci/hex-arch-guardrail.sh
else
    echo -e "${YELLOW}warning:${NC} hex-arch-guardrail.sh not found (skipped)"
fi
echo
