#!/usr/bin/env bash
# Scan staged changes locally or the exact range supplied by GitHub.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ $# == 1 && "$1" == --staged ]]; then
    exec scripts/tool.sh gitleaks git --staged --redact=100
fi

if [[ $# != 1 || "$1" != --github-event ]]; then
    echo "usage: $0 --staged | --github-event" >&2
    exit 2
fi

base=$(jq -er '.pull_request.base.sha // .before' "$GITHUB_EVENT_PATH")
head=$(jq -er '.pull_request.head.sha // .after' "$GITHUB_EVENT_PATH")
if [[ ! "$base" =~ ^[a-f0-9]{40}$ || ! "$head" =~ ^[a-f0-9]{40}$ ]]; then
    echo "Invalid GitHub commit range" >&2
    exit 2
fi

git cat-file -e "$head^{commit}"
if [[ "$base" == 0000000000000000000000000000000000000000 ]]; then
    range="$head"
else
    git cat-file -e "$base^{commit}"
    range="$base..$head"
fi

exec scripts/tool.sh gitleaks git --redact=100 --log-opts="$range --diff-merges=first-parent"
