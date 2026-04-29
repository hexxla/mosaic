#!/usr/bin/env bash
# Generic format hook for Go files
# Used by Cursor and similar platforms

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

gofmt -l -s -w "$1" 2>/dev/null || true
goimports -l -w "$1" 2>/dev/null || true

# Run LLM-specific hooks
if [ -f "scripts/llm/hooks/post-write-test-suggestion.sh" ]; then
  ./scripts/llm/hooks/post-write-test-suggestion.sh "$1" 2>/dev/null || true
fi

if [ -f "scripts/llm/hooks/post-write-doc-reminder.sh" ]; then
  ./scripts/llm/hooks/post-write-doc-reminder.sh "$1" 2>/dev/null || true
fi

# Run fast quality checks
go vet ./... 2>/dev/null || true
./scripts/ci/hex-arch-guardrail.sh 2>/dev/null || true
./scripts/ci/pre-commit/18-go-conventions.sh 2>/dev/null || true
