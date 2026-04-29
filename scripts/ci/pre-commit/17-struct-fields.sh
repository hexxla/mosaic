#!/usr/bin/env bash
# scripts/ci/pre-commit/17-struct-fields.sh
# Validate that domain structs don't have external dependencies in fields

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running struct field validation${NC}"

if [ ! -d "internal/core/domain" ]; then
    echo -e "${GREEN}No domain directory found, skipping${NC}"
    exit 0
fi

# Find all Go files in domain
domain_files=$(find internal/core/domain -name "*.go" 2>/dev/null || true)

errors=0

for file in $domain_files; do
    # Find struct definitions
    struct_defs=$(grep -n "^type [A-Z].* struct" "$file" || true)

    while IFS= read -r line; do
        line_num=$(echo "$line" | cut -d: -f1)
        struct_line=$(echo "$line" | cut -d: -f2-)
        struct_name=$(echo "$struct_line" | awk '{print $2}')

        # Extract struct body
        struct_body=$(sed -n "${line_num},/^}/p" "$file" || true)

        # Check for field types that might be external dependencies
        # Look for types from other internal packages
        if echo "$struct_body" | grep -E "internal/(adapter|core/services|core/ports)" | grep -q .; then
            echo -e "${RED}error:${NC} $file:$line_num Struct $struct_name has fields from adapter/services/ports packages"
            echo "$struct_body" | grep -E "internal/(adapter|core/services|core/ports)"
            ((errors++))
        fi

        # Check for framework types in domain structs
        if echo "$struct_body" | grep -E "(github\.com/gin|github\.com/gorilla|database/sql|http\.ResponseWriter|http\.Request)" | grep -q .; then
            echo -e "${RED}error:${NC} $file:$line_num Struct $struct_name has framework types (forbidden in domain)"
            echo "$struct_body" | grep -E "(github\.com/gin|github\.com/gorilla|database/sql|http\.ResponseWriter|http\.Request)"
            ((errors++))
        fi
    done <<< "$struct_defs"
done

# Also check service and port for framework leaks
for dir in internal/core/services internal/core/ports; do
    if [ ! -d "$dir" ]; then
        continue
    fi

    files=$(find "$dir" -name "*.go" 2>/dev/null || true)

    for file in $files; do
        struct_defs=$(grep -n "^type [A-Z].* struct" "$file" || true)

        while IFS= read -r line; do
            line_num=$(echo "$line" | cut -d: -f1)
            struct_body=$(sed -n "${line_num},/^}/p" "$file" || true)

            # Check for framework types in services/ports
            if echo "$struct_body" | grep -E "(github\.com/gin|github\.com/gorilla|database/sql)" | grep -q .; then
                echo -e "${RED}error:${NC} $file:$line_num Struct in $dir has framework types (forbidden)"
                echo "$struct_body" | grep -E "(github\.com/gin|github\.com/gorilla|database/sql)"
                ((errors++))
            fi
        done <<< "$struct_defs"
    done
done

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Struct field validation failed with $errors error(s)"
    exit 1
fi

echo -e "${GREEN}Struct fields: OK${NC}"
echo
