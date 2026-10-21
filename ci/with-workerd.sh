#!/usr/bin/env bash
# Run a command against a local workerd serving build/ (http://127.0.0.1:8913), then always stop workerd:
#
#   ci/with-workerd.sh smoke go test -C e2e ./smoke     # BASE is set for the command; logs in build/workerd-smoke.log
#
# Used by mise run smoke, e2e and load (unless they're pointed at a deployed URL). Refuses to start when :8913 is
# already taken (a mise run dev or a leftover workerd): stop that first, or two servers answer the same tests.
set -euo pipefail
name=$1
shift
base=http://127.0.0.1:8913
if curl -s -o /dev/null "$base/"; then
  echo "✗ :8913 is already in use (mise run dev, or a leftover workerd: pgrep -l workerd)"
  exit 1
fi
mkdir -p .workerd-state build
log=build/workerd-$name.log
workerd serve worker/local/config.capnp >"$log" 2>&1 &
pid=$!
# workerd can ignore SIGTERM while it drains (seen with hibernated WebSockets): escalate so :8913 is freed.
stop() {
  kill "$pid" 2>/dev/null || return 0
  for _ in $(seq 1 30); do kill -0 "$pid" 2>/dev/null || return 0; sleep 0.1; done
  kill -9 "$pid" 2>/dev/null || true
}
trap stop EXIT
for _ in $(seq 1 80); do
  [ "$(curl -s "$base/healthz")" = ok ] && break
  kill -0 "$pid" 2>/dev/null || { echo "✗ workerd exited:"; cat "$log"; exit 1; }
  sleep 0.25
done
[ "$(curl -s "$base/healthz")" = ok ] || { echo "✗ workerd not healthy after 20 s:"; cat "$log"; exit 1; }
echo "→ $base (workerd, log $log)"
status=0
BASE=$base "$@" || status=$?
# WORKERD_STRICT=1 (smoke): a panic or uncaught exception anywhere in the log fails the run.
if [ "${WORKERD_STRICT:-}" = 1 ] && grep -qiE 'panic|uncaught' "$log"; then echo "✗ errors in $log:"; cat "$log"; status=1; fi
exit $status
