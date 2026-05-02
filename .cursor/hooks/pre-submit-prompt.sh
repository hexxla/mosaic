#!/usr/bin/env bash
# Cursor pre-submit-prompt hook
# Injects mosaic context before agent reasoning

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

# Read JSON input from stdin
input=$(cat)

# Parse user prompt from cursor input
user_prompt=$(echo "$input" | jq -r '.prompt // empty' 2>/dev/null || echo "")

if [[ -z "$user_prompt" ]]; then
    # Allow the action if no prompt
    echo '{"continue": true, "permission": "allow"}'
    exit 0
fi

# Check if the user prompt mentions mosaic-related keywords
mosaic_keywords=("hex" "lattice" "cell" "embedding" "facet" "edge" "seam" "tag" "mosaic" "hexxla" "ratchet" "mcp")

mosaic_mentioned=false
for keyword in "${mosaic_keywords[@]}"; do
    if echo "$user_prompt" | grep -qi "$keyword"; then
        mosaic_mentioned=true
        break
    fi
done

if [[ "$mosaic_mentioned" == true ]]; then
    # Run the windsurf pre-user-prompt hook
    if [ -f ".windsurf/hooks/pre-user-prompt.sh" ]; then
        ./.windsurf/hooks/pre-user-prompt.sh "{\"user_prompt\": \"$user_prompt\"}" 2>/dev/null || true
    fi
fi

# Allow the action
echo '{"continue": true, "permission": "allow"}'

exit 0
