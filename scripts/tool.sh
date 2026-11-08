#!/usr/bin/env bash
# Run reviewed tool versions with the active Go toolchain.
set -euo pipefail
tool=${1:?tool name required}
shift

gitleaks_module=github.com/zricethezav/gitleaks/v8
gitleaks_version=v8.30.1

case "$tool" in
    actionlint) exec go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 "$@" ;;
    golangci-lint) exec go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 "$@" ;;
    install-gitleaks) exec go install "$gitleaks_module@$gitleaks_version" ;;
    gitleaks)
        binary=$(command -v gitleaks || true)
        if [[ -z "$binary" ]] || ! go version -m "$binary" |
            awk -v module="$gitleaks_module" -v version="$gitleaks_version" '
                $1 == "mod" && $2 == module && $3 == version { found = 1 }
                END { exit !found }
            '; then
            echo "Install the pinned scanner with scripts/tool.sh install-gitleaks and add Go's bin directory to PATH." >&2
            exit 1
        fi
        exec "$binary" "$@"
        ;;
    govulncheck) exec go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 "$@" ;;
    *) printf 'Unknown repository tool: %s\n' "$tool" >&2; exit 2 ;;
esac
