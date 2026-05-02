#!/usr/bin/env bash
# Generic post-write hook for Go files
# Used by all LLM platforms

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

file_path=$(echo "$1" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ "$file_path" == *.go ]]; then
  # Format Go files
  gofmt -l -s -w "$file_path" 2>/dev/null || true
  goimports -l -w "$file_path" 2>/dev/null || true

  # Run fast quality checks
  go vet ./... 2>/dev/null || true
  ./scripts/ci/hex-arch-guardrail.sh 2>/dev/null || true
  ./scripts/ci/pre-commit/18-go-conventions.sh 2>/dev/null || true

  # Run LLM-specific hooks
  if [ -f "scripts/llm/hooks/post-write-test-suggestion.sh" ]; then
    ./scripts/llm/hooks/post-write-test-suggestion.sh "$1" 2>/dev/null || true
  fi

  if [ -f "scripts/llm/hooks/post-write-doc-reminder.sh" ]; then
    ./scripts/llm/hooks/post-write-doc-reminder.sh "$1" 2>/dev/null || true
  fi
fi

exit 0
