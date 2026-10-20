#!/usr/bin/env bash
# Workers Builds (Cloudflare's GitHub app) runs this for every pushed branch; main's preview is named "staging"
# (PR #6; `mise run cf:builds` sets the trigger up):
#   build command:  ci/workers-builds.sh build    fetch the deploy tool CI published for this commit, then with it
#                                                 the build (verified against its manifest). No compiling.
#   deploy command: ci/workers-builds.sh deploy   deploy the preview <branch slug>, then delete previews of
#                                                 branches that no longer exist.
# Fallback, the old way: no tool from CI within 8 minutes (CI down, a fork) or PREVIEW_BUILD=compile → install the
# tools and compile here. Workers Builds provides CLOUDFLARE_API_TOKEN, WORKERS_CI_BRANCH, WORKERS_CI_COMMIT_SHA;
# CLOUDFLARE_ACCOUNT_ID is a build variable. APP_NAME and APP_REPO come from mise.toml.
set -euo pipefail
cd "$(dirname "$0")/.."

# Only this repo's mise.toml: the build image has mise config of its own (node, ruby, python, hugo, …) in the
# global/system config, in the clone's parent directory and in MISE_*_VERSION variables.
empty=$(mktemp --suffix .toml)
export MISE_GLOBAL_CONFIG_FILE="$empty" MISE_SYSTEM_CONFIG_FILE="$empty" MISE_CEILING_PATHS="$(dirname "$PWD")"
for v in $(env | grep -oE '^MISE_[A-Z0-9_]+_VERSION' || true); do unset "$v"; done
# task.run_auto_install off: `mise run` would otherwise install every tool in mise.toml first.
export MISE_YES=1 MISE_TRUSTED_CONFIG_PATHS="$PWD" MISE_TASK_RUN_AUTO_INSTALL=false PATH="$HOME/.local/bin:$PATH"
if ! command -v mise >/dev/null 2>&1; then
  curl -fsSL https://mise.run | MISE_VERSION="${MISE_VERSION:-v2026.10.14}" MISE_QUIET=1 sh
fi
eval "$(mise env -s bash)"   # APP_NAME, APP_REPO (no tools installed)
sha=${WORKERS_CI_COMMIT_SHA:-}

case "${1:-}" in
build)
  # The tool is published after the build, so once it's there the build is too.
  tool="https://github.com/$APP_REPO/releases/download/preview-builds/deploy-$sha"
  if [ "${PREVIEW_BUILD:-}" != compile ] && [ -n "$sha" ]; then
    for _ in $(seq 1 48); do   # 8 minutes; CI normally publishes in ~3
      if curl -fsSL -o deploy.tmp "$tool" 2>/dev/null; then chmod +x deploy.tmp && mkdir -p bin && mv deploy.tmp bin/deploy; break; fi
      # A newer push cancels CI for this commit, so its build never comes: stop instead of compiling a stale commit.
      head=$(git ls-remote "https://github.com/$APP_REPO" "refs/heads/${WORKERS_CI_BRANCH:-}" 2>/dev/null | cut -f1)
      if [ -n "$head" ] && [ "$head" != "$sha" ]; then
        echo "· ${WORKERS_CI_BRANCH} moved on to ${head:0:7}: ${sha:0:7} is superseded, nothing to deploy"
        touch .superseded
        exit 0
      fi
      sleep 10
    done
  fi
  if [ -x bin/deploy ] && bin/deploy -h >/dev/null 2>&1; then
    bin/deploy -name "$APP_NAME" -repo "$APP_REPO" -fetch-build "$sha" -wait 2m
  else
    echo "· no deploy tool from CI for ${sha:-this commit}${PREVIEW_BUILD:+ (PREVIEW_BUILD=$PREVIEW_BUILD)}: compiling here"
    rm -f bin/deploy
    mise install go tinygo github:WebAssembly/binaryen github:tailwindlabs/tailwindcss
    mise run build
  fi
  ;;
deploy)
  [ ! -e .superseded ] || { echo "· superseded by a newer push: no deploy"; exit 0; }
  branch=${WORKERS_CI_BRANCH:?WORKERS_CI_BRANCH is not set (Workers Builds sets it)}
  if [ -x bin/deploy ]; then deploy=(bin/deploy); else mise install go; deploy=(mise exec -- go run ./cmd/deploy); fi
  # main is staging: https://staging-$APP_NAME.<subdomain>.workers.dev, exactly what's merged. Other branches: their slug.
  if [ "$branch" = main ]; then slug=staging; else slug=$("${deploy[@]}" -name "$APP_NAME" -slug "$branch"); fi
  DEPLOY_CMD="${deploy[*]}" mise run --skip-deps preview "$slug"
  echo "✓ preview for branch $branch: $(cat .preview-url)"
  "${deploy[@]}" -name "$APP_NAME" -repo "$APP_REPO" -prune-previews -keep-preview staging || echo "· pruning previews failed (the preview is live)"
  ;;
*)
  echo "usage: ci/workers-builds.sh build|deploy" >&2
  exit 2
  ;;
esac
