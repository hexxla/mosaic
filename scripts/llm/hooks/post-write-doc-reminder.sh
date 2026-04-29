#!/usr/bin/env bash
# scripts/llm/hooks/post-write-doc-reminder.sh
# Post-write hook: Remind to add documentation for exported symbols

set -euo pipefail

YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT"

file_path=$(echo "$1" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ -z "$file_path" ]] || [[ "$file_path" != *.go ]]; then
    exit 0
fi

# Check for exported functions without comments
exported_funcs=$(grep -E '^func [A-Z]' "$file_path" || true)

if [[ -z "$exported_funcs" ]]; then
    exit 0
fi

missing_docs=0
while IFS= read -r line; do
    func_name=$(echo "$line" | sed 's/func //' | sed 's/(.*$//')
    # Check if the line before has a comment
    line_num=$(grep -n "^func $func_name" "$file_path" | cut -d: -f1)
    prev_line=$((line_num - 1))
    prev_content=$(sed "${prev_line}q;d" "$file_path")
    
    if [[ ! "$prev_content" =~ ^// ]]; then
        echo -e "${YELLOW}Missing documentation:${NC} $func_name"
        missing_docs=$((missing_docs + 1))
    fi
done <<< "$exported_funcs"

if [[ $missing_docs -gt 0 ]]; then
    echo -e "${CYAN}=== DOCUMENTATION REMINDER ===${NC}"
    echo ""
    echo -e "${YELLOW}Found $missing_docs exported function(s) without documentation${NC}"
    echo ""
    echo "Please add godoc comments for exported symbols:"
    echo ""
    echo "Example:"
    cat <<'EOF'
```go
// FuncName does X and returns Y.
//
// Parameters:
//   - ctx: context for cancellation
//   - input: the input value
//
// Returns:
//   - result: the result of the operation
//   - err: error if operation failed
func FuncName(ctx context.Context, input string) (result string, err error) {
    // implementation
}
```
EOF
    echo ""
    echo -e "${CYAN}=== END REMINDER ===${NC}"
fi

exit 0
