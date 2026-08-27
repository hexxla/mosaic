#!/usr/bin/env sh
# Print the Go language version declared by a module (default: current directory).
set -eu

module_dir=${1:-.}
go_mod="$module_dir/go.mod"

if [ ! -f "$go_mod" ]; then
	printf '%s\n' unknown
	exit 0
fi

awk '$1 == "go" { print $2; exit }' "$go_mod"
