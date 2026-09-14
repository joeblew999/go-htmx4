#!/usr/bin/env bash
# Scheduled/manual CI for kit/i18n's reference data (kit/i18n/README.md). Shell only, no Node.
#
# 1. mise run i18n:verify — pins match the pinned workerd release; tables regenerate byte for byte from pinned
#    cldr-json/CLDR/Chromium ICU; goldens re-record identically with the pinned workerd; known-difference evidence holds.
# 2. i18npins -latest -strict — fails when the newest workerd release builds with a different Chromium ICU: time to
#    move the pins (README, "Moving a pin"). It changes nothing.
# 3. mise run i18n:browsers — Chrome/Firefox Intl drift report (informational; GitHub's Ubuntu image has both).
set -euo pipefail
cd "$(dirname "$0")/.."

export MISE_YES=1
export MISE_TRUSTED_CONFIG_PATHS="$PWD"
if ! command -v mise >/dev/null 2>&1; then
  echo "→ installing mise ${MISE_VERSION:=v2026.9.5}"
  curl -fsSL https://mise.run | MISE_VERSION="$MISE_VERSION" MISE_QUIET=1 sh
  export PATH="$HOME/.local/bin:$PATH"
fi
mise install

status=0
mise run i18n:verify || status=1
mise exec -- go run ./cmd/i18npins -latest -strict || status=1
mise run i18n:browsers || echo "· browser drift report failed (informational)"
exit $status
