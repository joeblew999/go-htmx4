#!/usr/bin/env bash
# Workers Builds (Cloudflare's GitHub app) runs this for every pushed branch except main:
#   build command:  ci/workers-builds.sh build    (fetch the build GitHub CI made and tested for this commit)
#   deploy command: ci/workers-builds.sh deploy   (mise run preview <branch>: a Worker Preview, URL in the log)
# Cloudflare compiles nothing: GitHub CI (ci/check.sh) builds once and publishes build/ + dist/site as
# preview-builds/<commit>.tar.gz on this public repo's "preview-builds" pre-release. No secrets in GitHub.
# Workers Builds provides CLOUDFLARE_API_TOKEN (its build token), WORKERS_CI_BRANCH and WORKERS_CI_COMMIT_SHA;
# CLOUDFLARE_ACCOUNT_ID is a build variable. Plan: .plans/2026-10-20_1641_cloudflare-pr-previews.md
set -euo pipefail
cd "$(dirname "$0")/.."

# Only this repo's mise.toml. The build image has mise config of its own (node, ruby, python, hugo, …) in the
# global/system config, in the clone's parent directory (/opt/buildhome) and in MISE_*_VERSION variables; `mise
# install` would install all of it too (and its hugo pin fails with current mise).
empty=$(mktemp --suffix .toml)
export MISE_GLOBAL_CONFIG_FILE="$empty" MISE_SYSTEM_CONFIG_FILE="$empty" MISE_CEILING_PATHS="$(dirname "$PWD")"
for v in $(env | grep -oE '^MISE_[A-Z0-9_]+_VERSION' || true); do unset "$v"; done
# MISE_TASK_RUN_AUTO_INSTALL=false: `mise run` would otherwise install every tool in mise.toml first.
export MISE_YES=1 MISE_TRUSTED_CONFIG_PATHS="$PWD" MISE_TASK_RUN_AUTO_INSTALL=false PATH="$HOME/.local/bin:$PATH"

if ! command -v mise >/dev/null 2>&1; then
  curl -fsSL https://mise.run | MISE_VERSION="${MISE_VERSION:-v2026.10.14}" MISE_QUIET=1 sh
fi
mise --version
mise config ls

case "${1:-}" in
build)
  sha=${WORKERS_CI_COMMIT_SHA:?WORKERS_CI_COMMIT_SHA is not set (Workers Builds sets it)}
  url="https://github.com/${PREVIEW_BUILDS_REPO:-joeblew999/go-htmx4}/releases/download/preview-builds/$sha.tar.gz"
  # GitHub CI builds and tests this commit in ~3 minutes; wait for it (a failed check publishes nothing).
  for i in $(seq 1 90); do
    if curl -fsSL -o build.tar.gz "$url"; then break; fi
    [ "$i" = 90 ] && { echo "✗ no build for $sha after 15 minutes: did GitHub CI (check) fail?"; exit 1; }
    sleep 10
  done
  tar -xzf build.tar.gz && rm build.tar.gz
  echo "✓ build for $sha from GitHub CI: $(wc -c <build/app.wasm | tr -d ' ')B wasm, $(find dist/site -type f | wc -l | tr -d ' ') assets"
  mise install go   # the deploy tool (go run ./cmd/deploy)
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
