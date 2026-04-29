#!/usr/bin/env bash
# scripts/ci/commit-msg/00-conventional-commit.sh
# Validate conventional commit format
# Usage: ./scripts/ci/commit-msg/00-conventional-commit.sh <commit-message-file>

set -euo pipefail

RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

msg_file="$1"
msg=$(cat "$msg_file")

echo -e "${CYAN}> Validating conventional commit format${NC}"

if ! echo "$msg" | grep -qE '^(feat|fix|chore|docs|style|refactor|perf|test|build|ci|revert)(\([a-z0-9-]+\))?!?: '; then
    echo -e "${RED}error:${NC} Commit message does not follow Conventional Commits format."
    echo "Example: feat(auth): add login endpoint"
    echo "  - build: Changes that affect the build system or external dependencies"
    echo "  - ci: Changes to CI configuration files and scripts"
    echo "  - revert: Reverts a previous commit"
    exit 1
fi
echo
