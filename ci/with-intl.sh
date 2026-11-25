#!/usr/bin/env bash
# ci/with-intl.sh <command…>: the command with Intl, as a Worker has it. mise-intl's tests and its check format through
# workerd's Intl (the same V8 Intl a Worker runs on); outside a Worker that is a small workerd service. This starts it
# on a free port (mise-intl daemon), hands its address to the command (INTL_WORKERD_URL) and stops it after.
#
# mise-intl starts that service through pitchfork by itself (pitchfork.toml) when nothing names one. Naming it here
# needs no pitchfork supervisor, which on a Mac may have started with the machine, without mise on its PATH.
set -euo pipefail
log=$(mktemp)
for _ in 1 2 3 4 5; do
  port=$((20000 + RANDOM % 20000))
  PORT=$port mise-intl daemon >"$log" 2>&1 &
  pid=$!
  for _ in $(seq 1 150); do
    grep -q "workerd answers" "$log" && break 2
    kill -0 "$pid" 2>/dev/null || break # the port was taken: try another
    sleep 0.2
  done
  kill "$pid" 2>/dev/null || true
  pid=""
done
[ -n "${pid:-}" ] || { cat "$log"; echo "✗ mise-intl daemon didn't start"; exit 1; }
trap 'pkill -P "$pid" 2>/dev/null; kill "$pid" 2>/dev/null; rm -f "$log"' EXIT
INTL_WORKERD_URL="http://127.0.0.1:$port" "$@"
