#!/usr/bin/env bash
# scripts/ci/pre-rebase-protect-main.sh
# Prevent rebasing protected branches

set -euo pipefail

RED='\033[0;31m'
NC='\033[0m'

protected="main master"
current=$(git branch --show-current)

for branch in $protected; do
    if [ "$current" = "$branch" ]; then
        echo -e "${RED}error:${NC} Rebasing the $branch branch is not allowed."
        exit 1
    fi
done
echo
