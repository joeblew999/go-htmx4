#!/usr/bin/env bash
# Workers Builds (Cloudflare's GitHub app) runs this for every pushed branch except main:
#   build command:  ci/workers-builds.sh build    (fetch the build GitHub CI made and tested for this commit;
#                                                 compile here if it doesn't appear, or PREVIEW_BUILD=compile)
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
  # Normal path: GitHub CI built and tested this commit; fetch that build (~2 MB) instead of compiling again.
  # Fallback, the old way: compile here with TinyGo when that build doesn't appear (GitHub CI down or slow, a
  # forked repo) or when the build variable PREVIEW_BUILD=compile asks for it.
  sha=${WORKERS_CI_COMMIT_SHA:-}
  url="https://github.com/${PREVIEW_BUILDS_REPO:-joeblew999/go-htmx4}/releases/download/preview-builds/$sha.tar.gz"
  fetched=
  if [ "${PREVIEW_BUILD:-}" != compile ] && [ -n "$sha" ]; then
    for i in $(seq 1 48); do   # 8 minutes; GitHub CI normally publishes in ~4
      if curl -fsSL -o build.tar.gz "$url"; then fetched=1; break; fi
      sleep 10
    done
  fi
  if [ -n "$fetched" ]; then
    tar -xzf build.tar.gz && rm build.tar.gz
    echo "✓ build for $sha from GitHub CI: $(wc -c <build/app.wasm | tr -d ' ')B wasm, $(find dist/site -type f | wc -l | tr -d ' ') assets"
    # A prebuilt deploy tool comes with newer builds; without one (or if it won't run), compile it with Go.
    if [ -x bin/deploy ] && bin/deploy -h >/dev/null 2>&1; then echo "✓ prebuilt deploy tool"; else mise install go; fi
  else
    echo "· no GitHub CI build for ${sha:-this commit}${PREVIEW_BUILD:+ (PREVIEW_BUILD=$PREVIEW_BUILD)}: compiling here instead"
    mise install go tinygo github:WebAssembly/binaryen github:tailwindlabs/tailwindcss
    mise run build
  fi
  ;;
deploy)
  branch=${WORKERS_CI_BRANCH:?WORKERS_CI_BRANCH is not set (Workers Builds sets it)}
  if [ -x bin/deploy ] && bin/deploy -h >/dev/null 2>&1; then export DEPLOY_CMD=./bin/deploy; fi
  mise run --skip-deps preview "$branch"
  echo "✓ preview for branch $branch: $(cat .preview-url)"
  ;;
*)
  echo "usage: ci/workers-builds.sh build|deploy" >&2
  exit 2
  ;;
esac
