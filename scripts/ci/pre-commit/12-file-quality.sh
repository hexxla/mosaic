#!/usr/bin/env bash
# scripts/ci/pre-commit/12-file-quality.sh
# Check for file quality issues (trailing whitespace, merge conflicts, large files)

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running file quality checks${NC}"

# Check for trailing whitespace
trailing_whitespace=$(grep -Rl ' $' --include='*.go' --include='*.md' --include='*.yaml' --include='*.yml' --include='*.json' . 2>/dev/null || true)
if [ -n "$trailing_whitespace" ]; then
    echo -e "${RED}error:${NC} Files with trailing whitespace:"
    echo "$trailing_whitespace"
    echo "Fix by running: find . -type f -exec sed -i 's/[[:space:]]*$//' {} +"
    exit 1
fi

# Check for merge conflict markers
merge_conflicts=$(grep -Rl '^[<>]{7}' --include='*.go' --include='*.md' --include='*.yaml' --include='*.yml' --include='*.json' . 2>/dev/null || true)
if [ -n "$merge_conflicts" ]; then
    echo -e "${RED}error:${NC} Files with merge conflict markers:"
    echo "$merge_conflicts"
    echo "Resolve merge conflicts before committing"
    exit 1
fi

# Check for large files (max 512kb) - only check source/text files
large_files=$(find . -type f -size +512k \( -name '*.go' -o -name '*.md' -o -name '*.yaml' -o -name '*.yml' -o -name '*.json' -o -name '*.txt' \) ! -path './.git/*' ! -path './.tmp/*' 2>/dev/null || true)
if [ -n "$large_files" ]; then
    echo -e "${RED}error:${NC} Large files found (>512kb):"
    echo "$large_files"
    echo "Consider using git-lfs or removing these files"
    exit 1
fi

echo -e "${GREEN}File quality: OK${NC}"
echo
