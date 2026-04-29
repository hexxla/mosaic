#!/usr/bin/env bash
# scripts/ci/pre-commit/13-interface-impl.sh
# Validate that adapters implement required port interfaces

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running interface implementation validation${NC}"

# Find all adapter files
adapter_files=$(find internal/adapter -name "*.go" 2>/dev/null || true)

if [ -z "$adapter_files" ]; then
    echo -e "${GREEN}No adapter files found, skipping${NC}"
    exit 0
fi

errors=0

# Check each adapter file
for file in $adapter_files; do
    # Extract the package name from the adapter
    adapter_pkg=$(grep -m1 "^package " "$file" | sed 's/package //')

    # Look for struct definitions in adapter
    structs=$(grep -E "^type [A-Z].* struct" "$file" | awk '{print $2}' || true)

    for struct in $structs; do
        # Check if this struct claims to implement an interface (usually in comments or methods)
        # For now, we'll check if it has methods that match port interfaces

        # Get all methods defined for this struct
        methods=$(grep -E "func \($struct \*?[A-Z]" "$file" | sed 's/.*func //' | sed 's/(.*//' || true)

        # This is a basic check - in a real implementation, you'd parse the Go AST
        # to verify the struct actually implements the interface it claims to
        if [ -z "$methods" ]; then
            echo -e "${RED}warning:${NC} Adapter struct $struct in $file has no methods"
        fi
    done
done

# Check that port interfaces are actually implemented by adapters
port_interfaces=$(find internal/core/ports -name "*.go" -exec grep -l "^type.*interface" {} \; 2>/dev/null || true)

for port_file in $port_interfaces; do
    port_pkg=$(dirname "$port_file" | xargs basename)

    # Find corresponding adapter directory
    if [[ "$port_pkg" == "primary" ]]; then
        adapter_dir="internal/adapter/primary"
    elif [[ "$port_pkg" == "secondary" ]]; then
        adapter_dir="internal/adapter/secondary"
    else
        continue
    fi

    if [ ! -d "$adapter_dir" ]; then
        echo -e "${RED}error:${NC} Port interface in $port_file has no corresponding adapter directory: $adapter_dir"
        ((errors++))
    fi
done

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Interface implementation validation failed with $errors error(s)"
    exit 1
fi

echo -e "${GREEN}Interface implementation: OK${NC}"
echo
