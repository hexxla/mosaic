#!/usr/bin/env sh
# Print the Go version declared by the nearest enclosing module.
set -eu

if ! module_dir=$(CDPATH= cd -P -- "${1:-.}" 2>/dev/null && pwd -P); then
    printf '%s\n' unknown
    exit 0
fi

while [ ! -f "$module_dir/go.mod" ] && [ "$module_dir" != / ]; do
    module_dir=${module_dir%/*}
    module_dir=${module_dir:-/}
done

if [ ! -f "$module_dir/go.mod" ]; then
    printf '%s\n' unknown
    exit 0
fi

awk '$1 == "go" && NF >= 2 { version = $2; exit } END { print version ? version : "unknown" }' "$module_dir/go.mod"
