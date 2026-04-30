#!/usr/bin/env bash
# scripts/ci/pre-commit/15-import-order.sh
# Validate import order: stdlib → third-party (e.g. MCP SDK) → module-internal packages.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running import order validation${NC}"

go_files=$(find . -name "*.go" ! -path "./vendor/*" ! -path "./.git/*" 2>/dev/null || true)

errors=0

for file in $go_files; do
	import_block=$(sed -n '/^import (/,/^)/p' "$file" 2>/dev/null || true)

	if [ -z "$import_block" ]; then
		import_block=$(grep "^import \"" "$file" || true)
	fi

	if [ -z "$import_block" ]; then
		continue
	fi

	# When both MCP SDK and this repo's internal/ packages appear, MCP SDK imports must come first.
	mpc_line=$(echo "$import_block" | grep -n "github.com/modelcontextprotocol" | head -1 | cut -d: -f1 || true)
	mod_line=$(echo "$import_block" | grep -n "github.com/sploitzberg/mosaic/internal" | head -1 | cut -d: -f1 || true)
	if [[ -n "$mpc_line" && -n "$mod_line" && "$mpc_line" -gt "$mod_line" ]]; then
		echo -e "${RED}error:${NC} $file: github.com/modelcontextprotocol imports must precede module internal imports"
		((errors++))
	fi
done

if ((errors > 0)); then
	echo -e "${RED}error:${NC} Import order validation failed with $errors error(s)"
	echo "Expected order: stdlib → third-party (MCP SDK) → module-internal"
	exit 1
fi

echo -e "${GREEN}Import order: OK${NC}"
echo
