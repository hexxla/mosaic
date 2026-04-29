#!/usr/bin/env bash
# Secret scanning for CI and local development
# Uses high-signal patterns and falls back gracefully

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}> Running secret scanning${NC}"

errors=0

die() {
    echo -e "${RED}error:${NC} $*" >&2
    ((errors++))
}

# ======================
# High-signal secret patterns
# ======================

echo "Checking for AWS credentials"
if grep -rI --exclude-dir={.git,.cursor,.claude,.continue,bin,dist,node_modules,vendor} \
    -E 'AKIA[0-9A-Z]{16}' . 2>/dev/null | grep -q .; then
    die "Possible AWS Access Key ID found"
fi

echo "Checking for GitHub tokens"
if grep -rI --exclude-dir={.git,.cursor,.claude,.continue,bin,dist,node_modules,vendor} \
    -E 'gh[pousr]_[A-Za-z0-9_]{20,}' . 2>/dev/null | grep -q .; then
    die "Possible GitHub Personal Access Token found"
fi

echo "Checking for private keys"
if grep -rI --exclude-dir={.git,.cursor,.claude,.continue,bin,dist,node_modules,vendor} \
    -E '-----BEGIN (RSA |OPENSSH |EC |PGP )?PRIVATE KEY-----' . 2>/dev/null | grep -q .; then
    die "Private key material found"
fi

echo "Checking for OpenAI/Anthropic keys"
if grep -rI --exclude-dir={.git,.cursor,.claude,.continue,bin,dist,node_modules,vendor} \
    -E 'sk-[A-Za-z0-9]{48}' . 2>/dev/null | grep -q .; then
    die "Possible OpenAI or Anthropic API key found"
fi

# Add more patterns here as needed

# ======================
# Try gitleaks if available (more comprehensive)
# ======================
if command -v gitleaks >/dev/null 2>&1; then
    echo "Running gitleaks for comprehensive secret detection"
    if ! gitleaks detect --source . --no-git --verbose 2>/dev/null; then
        die "Gitleaks detected potential secrets"
    fi
fi

# ======================
# Summary
# ======================

if ((errors > 0)); then
    echo -e "${RED}error:${NC} Secret scanning FAILED with $errors finding(s)"
    echo "Please remove any secrets before committing."
    exit 1
else
    echo -e "${GREEN}Secret scanning: OK${NC}"
fi
echo
