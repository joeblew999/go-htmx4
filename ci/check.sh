#!/usr/bin/env bash
# CI entrypoint. Shell only, no Node.
#
# Installs mise if it's missing (mise's standalone installer, pinned), the tools pinned in mise.toml, then runs
# `mise run check` (TinyGo on workerd + gsx fmt + go vet + go test), then `mise run e2e` (headless Chrome against local
# workerd, with the chrome-headless-shell pinned in mise.toml). Runnable locally too: same command, same tools.
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

phase "mise install (tools)" mise install

# GitHub Actions: build first and publish it for this commit's Cloudflare preview straight away (in parallel with
# the tests below), so Cloudflare deploys this build instead of compiling it again (kit/previewbuild; PR #6). The
# deploy tool goes along (static linux/amd64): Cloudflare runs it to fetch, verify and deploy. A failing commit still
# gets a preview; this check stays red.
publish() {
  mise run build
  eval "$(mise env -s bash)" # APP_NAME, APP_REPO, CF_SUBDOMAIN
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/deploy ./cmd/deploy
  bin/deploy -name "$APP_NAME" -assets dist/site -repo "$GITHUB_REPOSITORY" -publish-build "$GITHUB_SHA"
  if [ -n "${CF_SUBDOMAIN:-}" ] && [ "${GITHUB_REF_TYPE:-branch}" = branch ] && [ "${GITHUB_REF_NAME:-main}" != main ] &&
    [ "${GITHUB_REF_NAME:-}" != production ]; then
    slug=$(bin/deploy -name "$APP_NAME" -slug "$GITHUB_REF_NAME")
    bin/deploy -repo "$GITHUB_REPOSITORY" -comment-pr "$GITHUB_REF_NAME" \
      -url "https://$slug-$APP_NAME.$CF_SUBDOMAIN.workers.dev" || echo "· PR comment failed (the preview still deploys)"
  fi
}
if [ -n "${GITHUB_ACTIONS:-}" ] && [ -n "${GITHUB_SHA:-}" ]; then
  phase "build (TinyGo) + publish" publish
fi

phase "mise run check" mise run check
phase "mise run e2e (Chrome)" mise run e2e

# A pushed v* tag releases: only a commit that's on main, and only after everything above passed.
release() {
  local tag=$GITHUB_REF_NAME status
  status=$(curl -fsS -H "Authorization: Bearer $GITHUB_TOKEN" \
    "https://api.github.com/repos/$GITHUB_REPOSITORY/compare/$GITHUB_SHA...main" | mise exec -- jq -r .status)
  case $status in
    identical | ahead) ;; # the tagged commit is main, or main is ahead of it: it was merged
    *) echo "✗ $tag (${GITHUB_SHA:0:7}) isn't on main ($status): not released"; return 1 ;;
  esac
  git push --force "https://x-access-token:$GITHUB_TOKEN@github.com/$GITHUB_REPOSITORY.git" "$GITHUB_SHA:refs/heads/production"
  echo "✓ $tag: production → ${GITHUB_SHA:0:7}; Workers Builds deploys it (Cloudflare dashboard → Workers → Builds)"
}
if [ -n "${GITHUB_ACTIONS:-}" ] && [ "${GITHUB_REF_TYPE:-}" = tag ] && [[ "${GITHUB_REF_NAME:-}" == v* ]]; then
  phase "release $GITHUB_REF_NAME" release
fi
