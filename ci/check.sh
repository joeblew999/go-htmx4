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

# Publish this commit's build for its Cloudflare preview (ci/workers-builds.sh fetches it), so Cloudflare deploys
# what was just tested instead of compiling it again. Only in GitHub Actions; keeps the newest 40 builds.
if [ -n "${GITHUB_ACTIONS:-}" ] && [ -n "${GITHUB_SHA:-}" ]; then
  export GH_REPO="$GITHUB_REPOSITORY"   # the checkout has no git remote for gh to read
  tar -czf "/tmp/$GITHUB_SHA.tar.gz" build/worker.mjs build/wasm_exec.js build/runtime.mjs build/app.wasm dist/site
  gh release view preview-builds >/dev/null 2>&1 ||
    gh release create preview-builds --prerelease --title "Preview builds" \
      --notes "Build output per commit for Cloudflare previews (ci/check.sh, ci/workers-builds.sh). Not a release." >/dev/null
  gh release upload preview-builds "/tmp/$GITHUB_SHA.tar.gz" --clobber
  echo "✓ build published: preview-builds/$GITHUB_SHA.tar.gz"
  gh release view preview-builds --json assets --jq '.assets | sort_by(.createdAt) | reverse | .[40:] | .[].name' |
    while read -r old; do gh release delete-asset preview-builds "$old" --yes; done
fi

start=$(date +%s)
mise run e2e
echo "✓ mise run e2e in $(( $(date +%s) - start ))s"
