#!/usr/bin/env bash
# Cursor pre-read hook
# Injects architecture and mosaic context before reading files

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Read JSON input from stdin
input=$(cat)

# Parse file path from cursor input
file_path=$(echo "$input" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ -z "$file_path" ]]; then
    # Allow the action if no file path
    echo '{"continue": true, "permission": "allow"}'
    exit 0
fi

# Run Mosaic-specific context injection
if [ -f ".windsurf/hooks/pre-read-mosaic-context.sh" ]; then
    ./.windsurf/hooks/pre-read-mosaic-context.sh "{\"file_path\": \"$file_path\"}" 2>/dev/null || true
fi

# Run context injection
if [ -f "scripts/llm/hooks/pre-read-context-injection.sh" ]; then
  ./scripts/llm/hooks/pre-read-context-injection.sh "{\"file_path\": \"$file_path\"}" 2>/dev/null || true
fi

# Run layer awareness
if [ -f "scripts/llm/hooks/pre-read-layer-awareness.sh" ]; then
  ./scripts/llm/hooks/pre-read-layer-awareness.sh "{\"file_path\": \"$file_path\"}" 2>/dev/null || true
fi

# Allow the action
echo '{"continue": true, "permission": "allow"}'

exit 0
