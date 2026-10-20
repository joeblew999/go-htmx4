#!/usr/bin/env bash
# The app's deploys in one place (mise run deploy / staging / preview / preview:delete): which modules, assets,
# bindings and D1 each one gets. cmd/deploy does the uploading (stdlib Go, no wrangler).
#
#   ci/deploy.sh production        $APP_NAME, custom domain $APP_DOMAIN if set → .deploy-url
#   ci/deploy.sh staging           $APP_NAME-staging, its own D1, noindex → .staging-url
#   ci/deploy.sh preview <name>    a Worker Preview of $APP_NAME, the shared preview D1, noindex → .preview-url
#   ci/deploy.sh delete-preview <name>
#
# Credentials: CLOUDFLARE_API_TOKEN/CLOUDFLARE_ACCOUNT_ID from the environment (Workers Builds' build token), else
# fnox. DEPLOY_CMD: a prebuilt cmd/deploy (Workers Builds gets one from GitHub CI); default: go run ./cmd/deploy.
# The URL files live outside build/: workers-assets-gen wipes build/.
set -euo pipefail
kind=${1:?production, staging, preview <name> or delete-preview <name>}
name=${2:-}
cf() { if [ -n "${CLOUDFLARE_API_TOKEN:-}" ] && [ -n "${CLOUDFLARE_ACCOUNT_ID:-}" ]; then "$@"; else fnox exec -- "$@"; fi; }
read -r -a deploy <<<"${DEPLOY_CMD:-go run ./cmd/deploy}"

# Every deploy: the entry module, every other worker/*.mjs next to it, Static Assets, D1 with migrations, the Room
# Durable Object and the write limit.
common=(-main worker/index.mjs -assets dist/site -migrations migrations -d1-create
  -do ROOM=Room -migration-tag v1 -new-sqlite-class Room -ratelimit WRITES=1001:60/10)
for m in worker/*.mjs; do [ "$m" = worker/index.mjs ] || common+=(-module "$m"); done

case $kind in
  production)
    domain=()
    [ -z "${APP_DOMAIN:-}" ] || domain=(-domain "$APP_DOMAIN")
    url=$(cf "${deploy[@]}" -name "$APP_NAME" -allow-existing "${common[@]}" -d1 DB="$APP_NAME" \
      -var "APP_ENV=cloudflare" -var "APP_DOMAIN=${APP_DOMAIN:-}" "${domain[@]}")
    echo "$url"
    # smoke-remote, e2e and load target the custom domain when there is one (workers.dev redirects to it).
    if [ -n "${APP_DOMAIN:-}" ]; then echo "https://$APP_DOMAIN"; else echo "$url"; fi >.deploy-url
    ;;
  staging)
    # Its own Worker, exactly what's on main. Never production: no custom domain, noindex, its own D1 and Durable Objects.
    cf "${deploy[@]}" -name "$APP_NAME-staging" -allow-existing "${common[@]}" -d1 DB="$APP_NAME-staging" \
      -var "APP_ENV=staging" -var "APP_DOMAIN=" -var "APP_PREVIEW=1" | tee .staging-url
    ;;
  preview)
    [ -n "$name" ] || { echo "✗ preview needs a name, e.g. pr-7"; exit 1; }
    ann=(-annotation "workers/commit_sha=$(git rev-parse HEAD)")
    gh=${GITHUB_SERVER_URL:-https://github.com}
    [ -z "${PREVIEW_PR:-}" ] || ann+=(-annotation "workers/pull_request_number=$PREVIEW_PR")
    [ -z "${GITHUB_REPOSITORY:-}" ] || ann+=(-annotation "workers/repository_url=$gh/$GITHUB_REPOSITORY")
    [ -z "${GITHUB_REPOSITORY:-}" ] || [ -z "${PREVIEW_PR:-}" ] ||
      ann+=(-annotation "workers/pull_request_url=$gh/$GITHUB_REPOSITORY/pull/$PREVIEW_PR")
    cf "${deploy[@]}" -name "$APP_NAME" -preview "$name" "${common[@]}" -d1 DB="$APP_NAME-preview" \
      -var "APP_ENV=preview $name" -var "APP_DOMAIN=" -var "APP_PREVIEW=1" "${ann[@]}" | tee .preview-url
    ;;
  delete-preview)
    [ -n "$name" ] || { echo "✗ delete-preview needs a name"; exit 1; }
    cf "${deploy[@]}" -name "$APP_NAME" -delete-preview "$name"
    ;;
  *)
    echo "✗ $kind: production, staging, preview <name> or delete-preview <name>"
    exit 1
    ;;
esac
