#!/usr/bin/env bash
# scripts/ci/pre-commit/04-hex-arch-guardrail.sh
# Check hexagonal architecture rules

set -euo pipefail

echo "Checking hexagonal architecture rules..."

if [[ -f ./scripts/ci/hex-arch-guardrail.sh ]]; then
    ./scripts/ci/hex-arch-guardrail.sh
else
    echo "Warning: hex-arch-guardrail.sh not found (skipped)"
fi
