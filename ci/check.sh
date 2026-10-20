#!/usr/bin/env bash
# CI entrypoint. Shell only, no Node.
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

# GitHub Actions: build first and publish it for this commit's Cloudflare preview straight away (in parallel with
# the tests below), so Cloudflare deploys this build instead of compiling it again (kit/previewbuild; issue #7). The deploy tool goes along (static linux/amd64): Cloudflare
# runs it to fetch, verify and deploy. A failing commit still gets a preview; this check stays red.
if [ -n "${GITHUB_ACTIONS:-}" ] && [ -n "${GITHUB_SHA:-}" ]; then
  start=$(date +%s)
  mise run build
  eval "$(mise env -s bash)"   # APP_NAME, APP_REPO, CF_SUBDOMAIN
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/deploy ./cmd/deploy
  bin/deploy -name "$APP_NAME" -assets dist/site -repo "$GITHUB_REPOSITORY" -publish-build "$GITHUB_SHA"
  if [ -n "${CF_SUBDOMAIN:-}" ] && [ "${GITHUB_REF_NAME:-main}" != main ]; then
    slug=$(bin/deploy -name "$APP_NAME" -slug "$GITHUB_REF_NAME")
    bin/deploy -repo "$GITHUB_REPOSITORY" -comment-pr "$GITHUB_REF_NAME" \
      -url "https://$slug-$APP_NAME.$CF_SUBDOMAIN.workers.dev" || echo "· PR comment failed (the preview still deploys)"
  fi
  echo "✓ build published in $(( $(date +%s) - start ))s"
fi

start=$(date +%s)
mise run check
echo "✓ mise run check in $(( $(date +%s) - start ))s"

start=$(date +%s)
mise run e2e
echo "✓ mise run e2e in $(( $(date +%s) - start ))s"
