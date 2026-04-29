#!/usr/bin/env bash
# scripts/llm/hooks/post-write-test-suggestion.sh
# Post-write hook: Suggest test cases for new code

set -euo pipefail

YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/../../../.." && pwd)"
cd "$ROOT"

file_path=$(echo "$1" | jq -r '.file_path // empty' 2>/dev/null || echo "")

if [[ -z "$file_path" ]] || [[ "$file_path" != *.go ]] || [[ "$file_path" == *_test.go ]]; then
    exit 0
fi

# Check if test file exists
test_file="${file_path%.go}_test.go"
if [[ -f "$test_file" ]]; then
    exit 0
fi

# Get the package name
package=$(grep -m1 '^package ' "$file_path" | awk '{print $2}')

echo -e "${CYAN}=== TEST SUGGESTION ===${NC}"
echo ""
echo -e "${YELLOW}No test file found for:${NC} $file_path"
echo ""
echo "Consider adding tests for:"
echo ""
# Extract exported functions
echo "Exported functions to test:"
grep -E '^func [A-Z]' "$file_path" | sed 's/func //' | sed 's/(.*$//' | while read -r func; do
    echo "  - Test$func"
done
echo ""
echo "Test file location: $test_file"
echo ""
echo "Example test structure:"
cat <<'EOF'
```go
package $package

import "testing"

func TestFuncName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name: "test case 1",
            input: "input",
            want:  "expected",
            wantErr: false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```
EOF
echo ""
echo -e "${CYAN}=== END SUGGESTION ===${NC}"
