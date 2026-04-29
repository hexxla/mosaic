#!/usr/bin/env bash
# scripts/ci/prepare-commit-msg-add-branch-name.sh
# Add branch name to commit message
# Usage: ./scripts/ci/prepare-commit-msg-add-branch-name.sh <commit-message-file>

set -euo pipefail

msg_file="$1"

# Only run on new commits (not amend)
if [ -z "${2-}" ]; then
    branch=$(git branch --show-current)
    if [ -n "$branch" ] && [ "$branch" != "main" ] && [ "$branch" != "master" ]; then
        echo "" >> "$msg_file"
        echo "# Branch: $branch" >> "$msg_file"
    fi
fi
