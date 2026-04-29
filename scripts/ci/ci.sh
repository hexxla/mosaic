#!/usr/bin/env bash
# Single entry point for all quality checks.
# Run locally with: ./scripts/ci/ci.sh or `make ci`

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

RED='\033[0;31m'
NC='\033[0m'

die() {
    echo -e "${RED}error:${NC} $*" >&2
    exit 1
}

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

need_cmd go
need_cmd git

# Run pre-commit checks with CI-specific configurations
echo "==> Running pre-commit checks"
./scripts/ci/pre-commit/00-gofmt.sh
./scripts/ci/pre-commit/01-goimports.sh
./scripts/ci/pre-commit/06-go-vet.sh
./scripts/ci/pre-commit/12-file-quality.sh

# Run tests with race detection for CI
echo "==> Running tests"
GO_TEST_FLAGS="-count=1 -race -parallel=1" ./scripts/ci/pre-commit/03-tests.sh

# Run linter with longer timeout for CI
echo "==> Running golangci-lint"
GOLANGCI_LINT_TIMEOUT="5m" ./scripts/ci/pre-commit/02-golangci-lint.sh

# CI-specific: go mod tidy validation
echo "==> Checking dependencies"
go mod tidy

if [[ "${CI:-}" == "true" ]] || git ls-files --error-unmatch go.mod >/dev/null 2>&1; then
    if [[ -n "$(git diff --name-only -- go.mod go.sum 2>/dev/null)" ]]; then
        echo "go.mod or go.sum is not up to date"
        git diff -- go.mod go.sum >&2 || true
        die "Run: go mod tidy && git add go.mod go.sum"
    fi
fi

# CI-specific: vulnerability check
echo "==> Checking for known vulnerabilities"
go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Run architecture and secret checks
./scripts/ci/pre-commit/04-hex-arch-guardrail.sh
./scripts/ci/pre-commit/07-secrets.sh

# Run additional quality checks
./scripts/ci/pre-commit/12-file-quality.sh
./scripts/ci/pre-commit/13-interface-impl.sh
./scripts/ci/pre-commit/14-exported-symbols.sh
./scripts/ci/pre-commit/15-import-order.sh
./scripts/ci/pre-commit/16-error-wrapping.sh
./scripts/ci/pre-commit/17-struct-fields.sh
./scripts/ci/pre-commit/18-go-conventions.sh

# Run pre-push checks for CI
./scripts/ci/pre-push/00-build.sh
./scripts/ci/pre-push/04-outdated-deps.sh

echo "==> OK"
