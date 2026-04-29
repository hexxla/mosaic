#!/usr/bin/env bash
# scripts/ci/pre-commit/16-error-wrapping.sh
# Validate that errors are wrapped with %w where appropriate

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running error wrapping validation${NC}"

# Find all Go files
go_files=$(find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" 2>/dev/null || true)

errors=0
warnings=0

for file in $go_files; do
    # Find functions that return error
    error_funcs=$(grep -n "func.*error" "$file" || true)

    while IFS= read -r line; do
        line_num=$(echo "$line" | cut -d: -f1)
        func_line=$(echo "$line" | cut -d: -f2-)

        # Check if function returns error
        if [[ ! "$func_line" =~ error ]]; then
            continue
        fi

        # Extract function body (simple approach)
        func_name=$(echo "$func_line" | sed 's/func //' | sed 's/(.*//' || true)

        # Look for error returns in the function
        # This is a simplified check - a proper implementation would parse AST
        error_returns=$(sed -n "${line_num},/^}/p" "$file" | grep -E "return.*fmt\.Errorf" || true)

        if [ -n "$error_returns" ]; then
            # Check if using %w for wrapping
            if echo "$error_returns" | grep -q "fmt.Errorf.*%w"; then
                continue
            fi

            # Check if it's a new error (not wrapping)
            if echo "$error_returns" | grep -q "fmt\.Errorf.*%"; then
                # This is OK - creating a new error
                continue
            fi

            # Warn about potential missing %w
            echo -e "${YELLOW}warning:${NC} $file:$line_num Function $func_name returns error but may not wrap with %w"
            ((warnings++))
        fi
    done <<< "$error_funcs"
done

# Check for error returns that should be wrapped
# Look for patterns like: return err
for file in $go_files; do
    # Find lines that return err directly without wrapping
    bare_err_returns=$(grep -n "return err" "$file" || true)

    while IFS= read -r line; do
        line_num=$(echo "$line" | cut -d: -f1)

        # Check if this is in a function that should wrap errors
        # This is heuristic - in practice you'd need AST parsing
        if echo "$line" | grep -q "return err$"; then
            # Skip if it's the last return in the function (common pattern)
            next_line=$(sed -n "$((line_num + 1))p" "$file")
            if [[ "$next_line" =~ ^}$ ]] || [[ "$next_line" =~ ^return ]]; then
                continue
            fi

            echo -e "${YELLOW}warning:${NC} $file:$line_num Bare 'return err' - consider wrapping with context"
            ((warnings++))
        fi
    done <<< "$bare_err_returns"
done

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Error wrapping validation failed with $errors error(s)"
    exit 1
elif ((warnings > 0)); then
    echo -e "${YELLOW}warning:${NC} Error wrapping validation passed with $warnings warning(s)"
    exit 0
fi

echo -e "${GREEN}Error wrapping: OK${NC}"
echo
