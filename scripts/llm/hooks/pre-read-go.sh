#!/usr/bin/env bash
# Generic pre-read hook for Go files
# Used by Windsurf and similar platforms

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Run context injection
if [ -f "scripts/llm/hooks/pre-read-context-injection.sh" ]; then
  ./scripts/llm/hooks/pre-read-context-injection.sh "$1" 2>/dev/null || true
fi

# Run layer awareness
if [ -f "scripts/llm/hooks/pre-read-layer-awareness.sh" ]; then
  ./scripts/llm/hooks/pre-read-layer-awareness.sh "$1" 2>/dev/null || true
fi

exit 0
