#!/usr/bin/env bash
# scripts/ci/commit-msg/01-length.sh
# Validate commit message length
# Usage: ./scripts/ci/commit-msg/01-length.sh <commit-message-file>

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

msg_file="$1"
msg=$(cat "$msg_file")

echo -e "${CYAN}> Validating commit message length${NC}"

# Remove commented lines (like # Branch: ...)
# Handle case where grep returns nothing
msg=$(echo "$msg" | grep -v '^#' || echo "$msg")

# Skip if message is empty after removing comments
if [ -z "$msg" ]; then
    exit 0
fi

# Get first line (subject)
subject=$(echo "$msg" | head -n 1)

# Check subject length (max 72 chars)
if [ ${#subject} -gt 72 ]; then
    echo -e "${RED}error:${NC} Commit subject line is too long (${#subject} chars). Max 72 chars."
    echo "Subject: $subject"
    exit 1
fi

# Check body line length (max 72 chars per line)
while IFS= read -r line; do
    if [ ${#line} -gt 72 ] && [ -n "$line" ]; then
        echo -e "${RED}error:${NC} Commit message line is too long (${#line} chars). Max 72 chars per line."
        echo "Line: $line"
        exit 1
    fi
done <<< "$msg"

# Check total message size (excluding comments) for conciseness
total_chars=${#msg}
MAX_TOTAL_CHARS=500
if [ $total_chars -gt $MAX_TOTAL_CHARS ]; then
    echo -e "${RED}error:${NC} Commit message is too long ($total_chars chars). Max $MAX_TOTAL_CHARS chars."
    echo "Please be concise. Use multiple commits for unrelated changes."
    exit 1
fi

echo -e "${GREEN}Commit message length: OK${NC}"
echo
