#!/usr/bin/env bash
# scripts/test-validators.sh
# Fuzz test validation scripts with actual Go files

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "==> Testing validation scripts with actual Go files"
echo

# Test 1: Domain with adapter import (should fail hex-arch-guardrail)
echo "Test 1: Domain with forbidden adapter import"
if [ ! -d "internal/core/domain" ]; then
    echo -e "${YELLOW}SKIP${NC}: No internal/core/domain directory"
else
    TEST_FILE="internal/core/domain/test_violation.go"
    cat > "$TEST_FILE" <<'EOF'
package domain

import "github.com/sploitzberg/go-llm-project-structure/internal/adapter"

type TestEntity struct{}
EOF

    # Run hex-arch-guardrail and check if it catches the violation
    if ./scripts/ci/hex-arch-guardrail.sh 2>&1 | grep -q "core/domain/ must not import"; then
        echo -e "${GREEN}PASS${NC}: Domain with forbidden adapter import detected"
    else
        echo -e "${RED}FAIL${NC}: Domain with forbidden adapter import not detected"
    fi
    rm -f "$TEST_FILE"
fi

# Test 2: Service with adapter import (should fail hex-arch-guardrail)
echo "Test 2: Service with forbidden adapter import"
if [ ! -d "internal/core/services" ]; then
    echo -e "${YELLOW}SKIP${NC}: No internal/core/services directory"
else
    TEST_FILE="internal/core/services/test_violation.go"
    cat > "$TEST_FILE" <<'EOF'
package service

import "github.com/sploitzberg/go-llm-project-structure/internal/adapter"

type TestService struct{}
EOF

    if ./scripts/ci/hex-arch-guardrail.sh 2>&1 | grep -q "core/services/ must not depend on adapter"; then
        echo -e "${GREEN}PASS${NC}: Service with forbidden adapter import detected"
    else
        echo -e "${RED}FAIL${NC}: Service with forbidden adapter import not detected"
    fi
    rm -f "$TEST_FILE"
fi

# Test 3: File with trailing whitespace (should fail file-quality)
echo "Test 3: File with trailing whitespace"
TEST_FILE="test_trailing.go"
cat > "$TEST_FILE" <<'EOF'
package test

func bad() {
    // trailing space
}
EOF

if ./scripts/ci/pre-commit/12-file-quality.sh 2>&1 | grep -q "trailing whitespace"; then
    echo -e "${GREEN}PASS${NC}: Trailing whitespace detected"
else
    echo -e "${RED}FAIL${NC}: Trailing whitespace not detected"
fi
rm -f "$TEST_FILE"

# Test 4: File with merge conflict markers (should fail file-quality)
echo "Test 4: File with merge conflict markers"
TEST_FILE="test_merge.go"
cat > "$TEST_FILE" <<'EOF'
package test

<<<<<<< HEAD
func old() {}
=======
func new() {}
>>>>>>> branch
EOF

if ./scripts/ci/pre-commit/12-file-quality.sh 2>&1 | grep -q "merge conflict"; then
    echo -e "${GREEN}PASS${NC}: Merge conflict markers detected"
else
    echo -e "${RED}FAIL${NC}: Merge conflict markers not detected"
fi
rm -f "$TEST_FILE"

# Test 5: Valid file (should pass file-quality)
echo "Test 5: Valid Go file"
TEST_FILE="test_valid.go"
cat > "$TEST_FILE" <<'EOF'
package test

func valid() string {
    return "ok"
}
EOF

if ./scripts/ci/pre-commit/12-file-quality.sh 2>&1 | grep -q "OK"; then
    echo -e "${GREEN}PASS${NC}: Valid file passes checks"
else
    echo -e "${RED}FAIL${NC}: Valid file failed checks"
fi
rm -f "$TEST_FILE"

# Test 6: Valid hex-arch (should pass)
echo "Test 6: Valid hex-arch structure"
if ./scripts/ci/hex-arch-guardrail.sh 2>&1 | grep -q "PASSED"; then
    echo -e "${GREEN}PASS${NC}: Valid hex-arch structure passes"
else
    echo -e "${RED}FAIL${NC}: Valid hex-arch structure failed"
fi

echo
echo "==> Test summary"
echo "Hex-arch-guardrail now uses go list for accurate import detection."
echo "File-quality checks use grep-based pattern matching."
