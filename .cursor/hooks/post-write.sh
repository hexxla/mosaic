#!/usr/bin/env bash
# Cursor post-write hook for Go files
# Receives JSON input via stdin

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Read JSON input from stdin
input=$(cat)

# Parse file path from cursor input
file_path=$(echo "$input" | jq -r '.file_path // empty' 2>/dev/null || echo "")

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
    ./scripts/llm/hooks/post-write-test-suggestion.sh "{\"file_path\": \"$file_path\"}" 2>/dev/null || true
  fi

  if [ -f "scripts/llm/hooks/post-write-doc-reminder.sh" ]; then
    ./scripts/llm/hooks/post-write-doc-reminder.sh "{\"file_path\": \"$file_path\"}" 2>/dev/null || true
  fi
fi

# Return success
echo '{"continue": true, "permission": "allow"}'

exit 0
