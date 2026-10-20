#!/usr/bin/env bash
# Worker Preview for a pull request (plan: .plans/2026-10-20_1641_cloudflare-pr-previews.md). Shell only, no Node.
#
#   ci/preview.sh deploy <pr-number>   build, deploy preview pr-<n>, comment its URL on the PR, run e2e against it
#   ci/preview.sh delete <pr-number>   delete preview pr-<n> (the PR closed)
#
# Needs CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID (GitHub Actions secrets). Without them it says so and exits 0,
# so pull requests don't fail before the secrets exist. Runnable locally too: with the variables unset, the preview
# tasks take the credentials from fnox, and the PR comment is skipped unless GITHUB_REPOSITORY and gh are there.
# check.yml tests every pull request; this only builds and deploys (mise run --skip-deps), then runs e2e remotely.
set -euo pipefail
cd "$(dirname "$0")/.."

action=${1:?usage: ci/preview.sh deploy|delete <pr-number>}
pr=${2:?usage: ci/preview.sh deploy|delete <pr-number>}
name="pr-$pr"

if [ -n "${CI:-}" ] && { [ -z "${CLOUDFLARE_API_TOKEN:-}" ] || [ -z "${CLOUDFLARE_ACCOUNT_ID:-}" ]; }; then
  echo "· no Cloudflare credentials (repository secrets CLOUDFLARE_API_TOKEN, CLOUDFLARE_ACCOUNT_ID): no preview"
  exit 0
fi

export MISE_YES=1
export MISE_TRUSTED_CONFIG_PATHS="$PWD"
if ! command -v mise >/dev/null 2>&1; then
  echo "→ installing mise ${MISE_VERSION:=v2026.10.14}"
  curl -fsSL https://mise.run | MISE_VERSION="$MISE_VERSION" MISE_QUIET=1 sh
  export PATH="$HOME/.local/bin:$PATH"
fi
mise install

if [ "$action" = delete ]; then
  mise run preview:delete "$name"
  exit 0
fi

mise run build
PREVIEW_PR="$pr" mise run --skip-deps preview "$name"
url=$(cat .preview-url)
echo "✓ preview $name: $url"

# One comment per PR, edited on every push (marker line), so the PR doesn't fill up with links.
if [ -n "${GITHUB_REPOSITORY:-}" ] && command -v gh >/dev/null 2>&1; then
  marker="<!-- worker-preview -->"
  body="$marker
**Preview:** $url

Commit \`$(git rev-parse --short HEAD)\`. Its own Durable Object storage; shared preview D1; not indexed. Deleted when the PR closes."
  id=$(gh api "repos/$GITHUB_REPOSITORY/issues/$pr/comments" --paginate --jq ".[] | select(.body | startswith(\"$marker\")) | .id" | head -1)
  if [ -n "$id" ]; then
    gh api -X PATCH "repos/$GITHUB_REPOSITORY/issues/comments/$id" -f body="$body" >/dev/null
  else
    gh api -X POST "repos/$GITHUB_REPOSITORY/issues/$pr/comments" -f body="$body" >/dev/null
  fi
  echo "✓ PR #$pr comment"
fi

# The same browser checks as CI's local e2e, against real Cloudflare.
E2E_BASE="$url" mise run --skip-deps e2e
