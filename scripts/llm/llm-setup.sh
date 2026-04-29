#!/usr/bin/env bash
# scripts/llm/llm-setup.sh
# Robust, cross-platform LLM tool setup script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PLATFORMS_DIR="$SCRIPT_DIR/platforms"

echo "=== LLM Tool Setup ==="

# Check dependencies
if ! command -v yq >/dev/null 2>&1; then
    echo "❌ Error: 'yq' is required but not installed."
    echo "   Install with: go install github.com/mikefarah/yq/v4@latest"
    echo "   or use your package manager (brew install yq, etc.)"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "❌ Error: 'jq' is required but not installed."
    echo "   Install jq for your OS."
    exit 1
fi

# List platforms
echo
echo "Available platforms:"
platforms=()
i=1
for dir in "$PLATFORMS_DIR"/*/ ; do
    if [ -d "$dir" ]; then
        name=$(basename "$dir")
        platforms+=("$name")
        printf "  %d) %s\n" $i "$name"
        ((i++))
    fi
done

echo
read -p "Select platform (number): " choice

if ! [[ "$choice" =~ ^[0-9]+$ ]] || (( choice < 1 || choice > ${#platforms[@]} )); then
    echo "❌ Error: Invalid selection"
    exit 1
fi

PLATFORM="${platforms[$((choice-1))]}"
CONFIG_FILE="$PLATFORMS_DIR/$PLATFORM/config.yaml"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Error: config.yaml not found for $PLATFORM"
    exit 1
fi

FOLDER=$(yq -r '.folder' "$CONFIG_FILE")
TARGET_DIR="$PROJECT_ROOT/$FOLDER"

echo "Setting up $PLATFORM → $FOLDER"
mkdir -p "$TARGET_DIR"

echo "Creating files and directories..."

# Process structure with better error handling
yq -o=json '.structure' "$CONFIG_FILE" | jq -c '.[]' | while read -r item; do
    path=$(echo "$item" | jq -r '.path')
    type=$(echo "$item" | jq -r '.type')
    content=$(echo "$item" | jq -r '.content // ""')
    source=$(echo "$item" | jq -r '.source // ""')

    fullpath="$TARGET_DIR/$path"

    if [ "$type" = "dir" ] || [ "$type" = "directory" ]; then
        mkdir -p "$fullpath"
        echo "✓ Created directory: $path"
    elif [ "$type" = "file" ]; then
        mkdir -p "$(dirname "$fullpath")"

        # Handle source field (copy from source file)
        if [ -n "$source" ]; then
            source_path="$PROJECT_ROOT/$source"
            if [ -f "$source_path" ]; then
                cp "$source_path" "$fullpath"
                echo "✓ Copied file: $path (from $source)"
            else
                echo "⚠️  Source file not found: $source_path (skipped)"
                touch "$fullpath"
            fi
        # Handle content field (inline content)
        elif [ -n "$content" ]; then
            echo "$content" > "$fullpath"
            echo "✓ Created file: $path"
        # No content or source, create empty file
        else
            touch "$fullpath"
            echo "✓ Created empty file: $path"
        fi

        # Set executable permission for shell scripts
        if [[ "$path" == *.sh ]]; then
            chmod +x "$fullpath"
        fi
    else
        echo "⚠️  Unknown type '$type' for path '$path' (skipped)"
    fi
done

echo
echo "✅ Successfully set up $PLATFORM!"
echo "Location: $FOLDER/"
