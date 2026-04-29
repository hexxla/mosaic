#!/usr/bin/env bash
# scripts/ci/pre-commit/18-go-conventions.sh
# Validate adherence to modern Go conventions from Effective Go and Google Style Guide

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running modern Go conventions validation${NC}"

errors=0
warnings=0

# Find all Go files
go_files=$(find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" 2>/dev/null || true)

for file in $go_files; do
    # Check 1: No context.Background() in exported functions (should accept context)
    if grep -q "func [A-Z]" "$file" && grep -q "context.Background()" "$file"; then
        echo -e "${YELLOW}warning:${NC} $file: Exported function uses context.Background() - consider accepting context.Context as parameter"
        ((warnings++))
    fi

    # Check 2: No TODO/FIXME/HACK comments in production code
    if grep -iE "TODO|FIXME|HACK|XXX" "$file" | grep -v "test" | grep -q .; then
        echo -e "${YELLOW}warning:${NC} $file: Contains TODO/FIXME/HACK comments - consider resolving or creating an issue"
        ((warnings++))
    fi

    # Check 3: No panic() in production code (except in init or tests)
    if [[ ! "$file" =~ _test\.go$ ]] && ! grep -q "func init()" "$file"; then
        if grep -q "panic(" "$file"; then
            echo -e "${RED}error:${NC} $file: Contains panic() in production code - use errors instead"
            ((errors++))
        fi
    fi

    # Check 4: No empty struct{} for channels (use struct{}{} instead)
    if grep -q "chan struct{}" "$file"; then
        echo -e "${YELLOW}warning:${NC} $file: Using chan struct{} - prefer chan struct{}{} for clarity"
        ((warnings++))
    fi

    # Check 5: No time.Sleep() in production code (use proper timing/timeout)
    if [[ ! "$file" =~ _test\.go$ ]]; then
        if grep -q "time.Sleep" "$file"; then
            echo -e "${YELLOW}warning:${NC} $file: Contains time.Sleep() - consider using context.WithTimeout or proper timing"
            ((warnings++))
        fi
    fi

    # Check 6: No bare returns in complex functions
    if grep -q "^func [A-Z]" "$file"; then
        # Check for bare return in functions with multiple return values
        func_lines=$(grep -n "^func [A-Z]" "$file" | cut -d: -f1)
        for line_num in $func_lines; do
            func_end=$(sed -n "$((line_num + 1)),/^}/p" "$file" | wc -l)
            if grep -q "^return$" "$file"; then
                echo -e "${YELLOW}warning:${NC} $file: Contains bare return - prefer explicit return values for clarity"
                ((warnings++))
            fi
        done
    fi

    # Check 7: No exported errors without Error() method
    if grep -q "type.*Error struct" "$file"; then
        if ! grep -q "func.*Error()" "$file"; then
            echo -e "${RED}error:${NC} $file: Exported error type without Error() method - implement error interface"
            ((errors++))
        fi
    fi

    # Check 8: No string() conversion on errors (use .Error() or type assertion)
    if grep -q 'string(err' "$file"; then
        echo -e "${YELLOW}warning:${NC} $file: Converting error to string - use .Error() or type assertion"
        ((warnings++))
    fi

    # Check 9: No os.Exit() in non-main packages
    if [[ ! "$file" =~ cmd/ ]] && [[ ! "$file" =~ _test\.go$ ]]; then
        if grep -q "os.Exit" "$file"; then
            echo -e "${RED}error:${NC} $file: Contains os.Exit() in non-main package - return error instead"
            ((errors++))
        fi
    fi

    # Check 10: No log.Fatal() in production code
    if [[ ! "$file" =~ _test\.go$ ]] && [[ ! "$file" =~ cmd/ ]]; then
        if grep -q "log.Fatal" "$file"; then
            echo -e "${RED}error:${NC} $file: Contains log.Fatal() in production code - return error instead"
            ((errors++))
        fi
    fi
done

# Check 11: No main packages outside cmd/
main_files=$(find . -name "main.go" ! -path "./cmd/*" ! -path "./vendor/*" ! -path "./.git/*" 2>/dev/null || true)
for file in $main_files; do
    if grep -q "^package main" "$file"; then
        echo -e "${RED}error:${NC} $file: Main package outside cmd/ directory - move to cmd/"
        ((errors++))
    fi
done

# Check 12: No init() functions in production code (use proper initialization)
init_files=$(find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" 2>/dev/null || true)
for file in $init_files; do
    if [[ ! "$file" =~ _test\.go$ ]] && grep -q "^func init()" "$file"; then
        echo -e "${YELLOW}warning:${NC} $file: Contains init() function - consider explicit initialization"
        ((warnings++))
    fi
done

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Go conventions validation failed with $errors error(s)"
    exit 1
elif ((warnings > 0)); then
    echo -e "${YELLOW}warning:${NC} Go conventions validation passed with $warnings warning(s)"
    exit 0
fi

echo -e "${GREEN}Go conventions: OK${NC}"
echo
