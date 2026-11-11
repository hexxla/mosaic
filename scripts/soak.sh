#!/usr/bin/env bash
# Linux qualification stages are explicit. Never chains or promotes stages.
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ "${1:-}" == --inside-cgroup ]]; then
  shift
  export GOMAXPROCS=2 GOMEMLIMIT=512MiB GOWORK=off
  go build -o .tmp/mosaic-soak ./cmd/mosaic-soak
  exec .tmp/mosaic-soak "$@"
fi
command -v systemd-run >/dev/null
mkdir -p .tmp/qualification
stage=30m
verify=false
args=("$@")
for ((i=0; i<${#args[@]}; i++)); do
  case "${args[i]}" in
    --duration) ((i+=1)); stage="${args[i]:?--duration requires a value}";;
    --duration=*) stage="${args[i]#--duration=}";;
    --verify|--verify=true) verify=true;;
    --verify=false) verify=false;;
  esac
done
# Exact stages keep the OS timeout consistent with the runner's duration.
case "$stage" in
  30m) ceiling=2400;;
  1h) ceiling=4200;;
  24h) ceiling=87000;;
  *)
    if [[ "$verify" != true ]]; then
      echo 'Select an explicit stage: --duration 30m, 1h, or 24h.' >&2
      exit 2
    fi
    ceiling=720;;
esac
run="$(date -u +%Y%m%dT%H%M%SZ)-$$"
exec systemd-run --user --wait --pipe --collect --unit="mosaic-soak-$run" \
  --working-directory="$PWD" \
  -p MemoryHigh=768M -p MemoryMax=1G -p MemorySwapMax=0 \
  -p CPUQuota=100% -p TasksMax=128 -p LimitFSIZE=134217728 -p RuntimeMaxSec="$ceiling" \
  /usr/bin/env "PATH=$PATH" "GOTOOLCHAIN=${GOTOOLCHAIN:-go1.27.4}" \
  bash "$PWD/scripts/soak.sh" --inside-cgroup \
  --dir "$PWD/.tmp/qualification/soak-$run" --owner void "$@"
