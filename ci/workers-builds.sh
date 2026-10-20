#!/usr/bin/env bash
# Workers Builds (Cloudflare's GitHub app) runs this for every pushed branch except main:
#   build command:  ci/workers-builds.sh build    (mise + the pinned tools, then the TinyGo build and assets)
#   deploy command: ci/workers-builds.sh deploy   (mise run preview <branch>: a Worker Preview, URL in the log)
# Workers Builds provides CLOUDFLARE_API_TOKEN (its build token) and the branch in WORKERS_CI_BRANCH;
# CLOUDFLARE_ACCOUNT_ID is a build variable. Plan: .plans/2026-10-20_1641_cloudflare-pr-previews.md
set -euo pipefail
cd "$(dirname "$0")/.."

# Only this repo's mise.toml: the build image has a global mise config of its own (node, ruby, python, hugo, …)
# that `mise install` would otherwise install too (and its hugo pin fails with current mise).
empty=$(mktemp)
export MISE_GLOBAL_CONFIG_FILE="$empty" MISE_SYSTEM_CONFIG_FILE="$empty"
export MISE_YES=1 MISE_TRUSTED_CONFIG_PATHS="$PWD" PATH="$HOME/.local/bin:$PATH"

if ! command -v mise >/dev/null 2>&1; then
  curl -fsSL https://mise.run | MISE_VERSION="${MISE_VERSION:-v2026.10.14}" MISE_QUIET=1 sh
fi
mise --version

case "${1:-}" in
build)
  mise install
  mise run build
  ;;
deploy)
  branch=${WORKERS_CI_BRANCH:?WORKERS_CI_BRANCH is not set (Workers Builds sets it)}
  mise run --skip-deps preview "$branch"
  echo "✓ preview for branch $branch: $(cat .preview-url)"
  ;;
*)
  echo "usage: ci/workers-builds.sh build|deploy" >&2
  exit 2
  ;;
esac
