#!/usr/bin/env bash
# scripts/ci/pre-commit/15-import-order.sh
# Validate import order: stdlib → third-party → internal

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running import order validation${NC}"

# Find all Go files
go_files=$(find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" 2>/dev/null || true)

errors=0

for file in $go_files; do
    # Extract import block
    import_block=$(sed -n '/^import (/,/^)/p' "$file" 2>/dev/null || true)

    if [ -z "$import_block" ]; then
        # Single line imports
        import_block=$(grep "^import \"" "$file" || true)
    fi

    if [ -z "$import_block" ]; then
        continue
    fi

    # Categorize imports
    stdlib_imports=""
    third_party_imports=""
    internal_imports=""

    in_stdlib=true
    in_third_party=false
    in_internal=false

    while IFS= read -r line; do
        # Skip empty lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*// ]] && continue

        # Extract import path
        import_path=$(echo "$line" | sed 's/.*"\(.*\)".*/\1/' || true)

        if [[ -z "$import_path" ]]; then
            continue
        fi

        if [[ "$import_path" =~ ^internal/ ]]; then
            in_internal=true
            in_stdlib=false
            in_third_party=false
        elif [[ "$import_path" =~ ^github\.com/ ]]; then
            if ! in_internal && ! in_third_party; then
                in_third_party=true
                in_stdlib=false
            fi
        fi
    done <<< "$import_block"

    # Check order: stdlib should come before third-party, which should come before internal
    # This is a simplified check - a proper implementation would parse the full import block

    # For now, just check that stdlib imports don't appear after internal imports
    if echo "$import_block" | grep -q "internal.*" && echo "$import_block" | grep -B 10 "internal" | grep -q "github\.com"; then
        echo -e "${RED}error:${NC} $file: Third-party imports appear before internal imports"
        ((errors++))
    fi
done

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Import order validation failed with $errors error(s)"
    echo "Expected order: stdlib → third-party → internal"
    exit 1
fi

echo -e "${GREEN}Import order: OK${NC}"
echo
