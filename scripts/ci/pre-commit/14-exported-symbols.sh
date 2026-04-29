#!/usr/bin/env bash
# scripts/ci/pre-commit/14-exported-symbols.sh
# Validate that domain only exports necessary types

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running exported symbol validation${NC}"

if [ ! -d "internal/core/domain" ]; then
    echo -e "${YELLOW}warning:${NC} No domain directory found, skipping"
    exit 0
fi

errors=0
warnings=0

# Find all exported types in domain
domain_files=$(find internal/core/domain -name "*.go" 2>/dev/null || true)

for file in $domain_files; do
    # Find exported types
    exported_types=$(grep -E "^type [A-Z]" "$file" | awk '{print $2}' || true)

    for type in $exported_types; do
        # Check if type is struct (common for domain entities)
        if grep -q "^type $type struct" "$file"; then
            # Check if struct has unexported fields (this is OK for domain)
            # But warn if it has exported fields that might be implementation details
            exported_fields=$(grep -A 20 "^type $type struct" "$file" | grep -E "^\s+[A-Z]" | awk '{print $1}' || true)

            if [ -n "$exported_fields" ]; then
                echo -e "${YELLOW}warning:${NC} Domain struct $type in $file has exported fields: $exported_fields"
                ((warnings++))
            fi
        fi
    done

    # Check for exported functions that might be implementation details
    exported_funcs=$(grep -E "^func [A-Z]" "$file" | awk '{print $2}' | sed 's/(.*//' || true)

    for func in $exported_funcs; do
        func_name=$(echo "$func" | sed 's/(.*//')
        # Skip common constructors and public API functions
        if [[ "$func_name" =~ ^(New|Create|Get|Update|Delete|Find|List) ]]; then
            continue
        fi

        # Warn about other exported functions that might be internal
        echo -e "${YELLOW}warning:${NC} Domain file $file has exported function $func_name that might be an implementation detail"
        ((warnings++))
    done
done

# Check that domain doesn't export implementation patterns
if grep -r "internal.*adapter\|internal.*core/services" internal/core/domain 2>/dev/null | grep -q .; then
    echo -e "${RED}error:${NC} Domain has references to adapter or services packages"
    ((errors++))
fi

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Exported symbol validation failed with $errors error(s)"
    exit 1
elif ((warnings > 0)); then
    echo -e "${YELLOW}warning:${NC} Exported symbol validation passed with $warnings warning(s)"
    exit 0
fi

echo -e "${GREEN}Exported symbols: OK${NC}"
echo
