#!/usr/bin/env bash
# CI entrypoint (plan: .plans/done/2026-09-14_0915_ci-mise-check.md). Shell only, no Node.
#
# Installs mise if it's missing (mise's standalone installer, pinned), the tools pinned in mise.toml, then runs
# `mise run check` (TinyGo on workerd + gsx fmt + go vet + go test), then `mise run e2e` (headless Chrome against local
# workerd, with the chrome-headless-shell pinned in mise.toml). Runnable locally too: same command, same tools.
set -euo pipefail
cd "$(dirname "$0")/.."

export MISE_YES=1
export MISE_TRUSTED_CONFIG_PATHS="$PWD"

if ! command -v mise >/dev/null 2>&1; then
  echo "→ installing mise ${MISE_VERSION:=v2026.10.14}"
  curl -fsSL https://mise.run | MISE_VERSION="$MISE_VERSION" MISE_QUIET=1 sh
  export PATH="$HOME/.local/bin:$PATH"
fi
mise --version

start=$(date +%s)
mise install
echo "✓ mise install in $(( $(date +%s) - start ))s"

start=$(date +%s)
mise run check
echo "✓ mise run check in $(( $(date +%s) - start ))s"

start=$(date +%s)
mise run e2e
echo "✓ mise run e2e in $(( $(date +%s) - start ))s"
