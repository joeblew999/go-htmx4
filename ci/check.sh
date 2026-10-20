#!/usr/bin/env bash
# CI entrypoint. Shell only, no Node.
#
# Installs mise if it's missing (mise's standalone installer, pinned) and the tools, then runs what `mise run check`
# and `mise run e2e` run, without repeating a step: the production build beside the Go tests, the workerd smoke test,
# then headless Chrome (the chrome-headless-shell pinned in mise.toml). Runnable locally too: same command, same tools.
#
# On GitHub it also publishes the build for Cloudflare's preview of this commit, and for a pushed v* tag it releases:
# once everything passed, it moves the production branch to the tagged commit and Workers Builds deploys production
# (ci/workers-builds.sh). Every run ends with a table of where the time went.
set -euo pipefail
cd "$(dirname "$0")/.."

export MISE_YES=1
export MISE_TRUSTED_CONFIG_PATHS="$PWD"

timings=()
# phase NAME COMMAND…: run it, record how long it took.
phase() {
  local name=$1 start
  shift
  start=$(date +%s)
  echo "── $name"
  "$@"
  timings+=("$name|$(($(date +%s) - start))")
}
report() {
  local total=0 row
  echo
  echo "Where the time went:"
  for row in "${timings[@]}"; do
    printf '  %-34s %4ss\n' "${row%%|*}" "${row##*|}"
    total=$((total + ${row##*|}))
  done
  printf '  %-34s %4ss\n' total "$total"
  # mise prints each task's own time ("[build] Finished in 108s"): the build is the big one.
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
      echo "| phase | seconds |"
      echo "| --- | ---: |"
      for row in "${timings[@]}"; do echo "| ${row%%|*} | ${row##*|} |"; done
      echo "| **total** | **$total** |"
    } >>"$GITHUB_STEP_SUMMARY"
  fi
}
trap report EXIT

if ! command -v mise >/dev/null 2>&1; then
  echo "→ installing mise ${MISE_VERSION:=v2026.10.14}"
  curl -fsSL https://mise.run | MISE_VERSION="$MISE_VERSION" MISE_QUIET=1 sh
  export PATH="$HOME/.local/bin:$PATH"
fi
mise --version

# On CI, only the tools CI runs: not fnox (credentials), usage (shell completion) or the gsxui CLI (refreshing
# components; compiling it took ~27 s per run). Locally, everything mise.toml pins. The editor hook skips itself on CI.
if [ -n "${CI:-}" ]; then
  # And don't let `mise run` install the rest before each task (its default): that compiled the gsxui CLI anyway.
  export MISE_TASK_RUN_AUTO_INSTALL=false
  phase "mise install (CI tools)" mise install go tinygo github:WebAssembly/binaryen github:tailwindlabs/tailwindcss \
    workerd http:chrome-headless-shell jq
else
  phase "mise install (tools)" mise install
fi

# GitHub Actions: build first and publish it for this commit's Cloudflare preview straight away (in parallel with
# the tests below), so Cloudflare deploys this build instead of compiling it again (kit/previewbuild; PR #6). The
# deploy tool goes along (static linux/amd64): Cloudflare runs it to fetch, verify and deploy. A failing commit still
# gets a preview; this check stays red.
publish() {
  eval "$(mise env -s bash)" # APP_NAME, APP_REPO, CF_SUBDOMAIN, WORKER_COMPILER
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/deploy ./cmd/deploy
  bin/deploy -name "$APP_NAME" -assets dist/site -repo "$GITHUB_REPOSITORY" -publish-build "$GITHUB_SHA"
  if [ -n "${CF_SUBDOMAIN:-}" ] && [ "${GITHUB_REF_TYPE:-branch}" = branch ] && [ "${GITHUB_REF_NAME:-main}" != main ] &&
    [ "${GITHUB_REF_NAME:-}" != production ] && [[ "${GITHUB_REF_NAME:-}" != release/* ]]; then
    slug=$(bin/deploy -name "$APP_NAME" -slug "$GITHUB_REF_NAME")
    bin/deploy -repo "$GITHUB_REPOSITORY" -comment-pr "$GITHUB_REF_NAME" \
      -url "https://$slug-$APP_NAME.$CF_SUBDOMAIN.workers.dev" || echo "· PR comment failed (the preview still deploys)"
  fi
}
# The production build and the Go tests (formatting, vet, go test) run side by side: the build is mostly one core of
# TinyGo. Then the smoke test on workerd (= mise run check, whose build and tests have just run), then Chrome.
compiler=$(mise env -s bash | sed -n 's/^export WORKER_COMPILER=//p' | tr -d "'\"")
phase "build (${compiler:-tinygo}) + Go tests" mise run build ::: test:go
if [ -n "${GITHUB_ACTIONS:-}" ] && [ -n "${GITHUB_SHA:-}" ]; then
  phase "publish the build" publish
fi
phase "smoke (workerd)" mise run --skip-deps smoke
phase "e2e (Chrome)" mise run --skip-deps e2e
# CPU per request of this build on workerd (charter's workerbench), into the run's summary: a slower page shows on
# every PR. One short run: CI's machines vary, so it is for spotting a jump, not for the numbers in docs/benchmarks.md.
bench_summary() {
  BENCH_N=30 BENCH_RUNS=1 mise run --skip-deps bench | tee build/bench.txt
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    { echo; echo "CPU per request on workerd (ms, mise run bench):"; echo; echo '```'; grep -E "median|CPU per request" build/bench.txt; echo '```'; } >>"$GITHUB_STEP_SUMMARY"
  fi
}
phase "bench (workerd CPU)" bench_summary

# A release: a pushed v* tag (GitHub's Releases page, git push), or a pushed release/v* branch (mise run release: works
# where only branches can be pushed). Only a commit that's on main, and only after everything above passed. For a
# release branch CI makes the tag itself, then deletes the branch.
release() {
  local version=$1 status api="https://api.github.com/repos/$GITHUB_REPOSITORY" auth=(-H "Authorization: Bearer $GITHUB_TOKEN")
  status=$(curl -fsS "${auth[@]}" "$api/compare/$GITHUB_SHA...main" | mise exec -- jq -r .status)
  case $status in
    identical | ahead) ;; # the commit is main, or main is ahead of it: it was merged
    *) echo "✗ $version (${GITHUB_SHA:0:7}) isn't on main ($status): not released"; return 1 ;;
  esac
  if [ "${GITHUB_REF_TYPE:-}" = branch ]; then
    curl -fsS "${auth[@]}" -X POST "$api/git/refs" -d "{\"ref\":\"refs/tags/$version\",\"sha\":\"$GITHUB_SHA\"}" >/dev/null ||
      { echo "✗ couldn't create tag $version (it may exist already)"; return 1; }
    echo "✓ tag $version → ${GITHUB_SHA:0:7}"
  fi
  git push --force "https://x-access-token:$GITHUB_TOKEN@github.com/$GITHUB_REPOSITORY.git" "$GITHUB_SHA:refs/heads/production"
  echo "✓ $version: production → ${GITHUB_SHA:0:7}; Workers Builds deploys it (Cloudflare dashboard → Workers → Builds)"
  if [ "${GITHUB_REF_TYPE:-}" = branch ]; then
    curl -fsS "${auth[@]}" -X DELETE "$api/git/refs/heads/$GITHUB_REF_NAME" >/dev/null || true
  fi
}
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  case "${GITHUB_REF_TYPE:-}:${GITHUB_REF_NAME:-}" in
    tag:v*) phase "release $GITHUB_REF_NAME" release "$GITHUB_REF_NAME" ;;
    branch:release/v*) phase "release ${GITHUB_REF_NAME#release/}" release "${GITHUB_REF_NAME#release/}" ;;
  esac
fi
