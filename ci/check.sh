#!/usr/bin/env bash
# CI entrypoint. Shell only, no Node.
#
# Installs mise if it's missing (mise's standalone installer, pinned) and the tools, then runs what `mise run check`
# and `mise run e2e` run, without repeating a step: the production build beside the Go tests, the workerd smoke test,
# then headless Chrome (the chrome-headless-shell pinned in mise.toml). Runnable locally too: same command, same tools.
#
# On GitHub, for a push, it also publishes the build for Cloudflare's preview of this commit, and for a pushed v* tag
# it releases: once everything passed, it moves the production branch to the tagged commit and Workers Builds deploys
# production (ci/workers-builds.sh). A fork's pull request gets the checks only. Every run ends with a table of where
# the time went.
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
  echo "→ installing mise ${MISE_VERSION:=v$(sed -n 's/^min_version = "\(.*\)"/\1/p' mise.toml)}"
  curl -fsSL https://mise.run | MISE_VERSION="$MISE_VERSION" MISE_QUIET=1 sh
  export PATH="$HOME/.local/bin:$PATH"
fi
mise --version

# On CI, only the tools CI runs: not fnox (credentials), usage (shell completion) or the gsxui CLI (refreshing
# components; compiling it took ~27 s per run). Locally, everything mise.toml pins. The editor hook skips itself on CI.
if [ -n "${CI:-}" ]; then
  # And don't let `mise run` install the rest before each task (its default): that compiled the gsxui CLI anyway.
  export MISE_TASK_RUN_AUTO_INSTALL=false
  # go@1.26: the oldest Go intl, seo and badge declare, which test:go builds them with too.
  phase "mise install (CI tools)" mise install go go@1.26 tinygo github:WebAssembly/binaryen github:tailwindlabs/tailwindcss \
    workerd http:chrome-headless-shell jq
else
  phase "mise install (tools)" mise install
fi

# GitHub Actions: build first and publish it for this commit's Cloudflare preview straight away (in parallel with
# the tests below), so Cloudflare deploys this build instead of compiling it again: go tool charter ci publish, from a
# static linux/amd64 charter, which goes along so Workers Builds runs it to fetch, verify and deploy (go tool charter
# builds, ci/workers-builds.sh). A failing commit still gets a preview; this check stays red.
publish() {
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/charter github.com/joeblew999/charter/go/lib/cmd/charter
  bin/charter ci publish
}
# The production build and the Go tests (formatting, vet, go test) run side by side: the build is mostly one core of
# TinyGo. Then the smoke test on workerd (= mise run check, whose build and tests have just run), then Chrome.
compiler=$(mise env -s bash | sed -n 's/^export WORKER_COMPILER=//p' | tr -d "'\"")
phase "build (${compiler:-tinygo}) + Go tests + fresh app" mise run build ::: test:go ::: fresh
# Only a push publishes: a fork's pull request runs read-only.
pushed=; [ -n "${GITHUB_ACTIONS:-}" ] && [ "${GITHUB_EVENT_NAME:-push}" = push ] && pushed=1
if [ -n "$pushed" ] && [ -n "${GITHUB_SHA:-}" ]; then
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
# where only branches can be pushed). go tool charter ci release, after everything above passed: only a commit that's
# on main; a release branch gets its tag, then goes. The hook tags intl, seo and badge, which other apps use, with
# their next version when they changed, before production moves (Workers Builds deploys it).
release() {
  bin/charter ci release -hook "ci/module-tags.sh $GITHUB_SHA" "$1"
}
if [ -n "$pushed" ]; then
  case "${GITHUB_REF_TYPE:-}:${GITHUB_REF_NAME:-}" in
    tag:v*) phase "release $GITHUB_REF_NAME" release "$GITHUB_REF_NAME" ;;
    branch:release/v*) phase "release ${GITHUB_REF_NAME#release/}" release "${GITHUB_REF_NAME#release/}" ;;
  esac
fi
