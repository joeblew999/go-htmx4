#!/usr/bin/env bash
# The app's deploys in one place (mise run deploy / staging / preview, and Workers Builds): which modules, assets,
# bindings and D1 each one gets. charter's workerdeploy does the uploading (Go, no wrangler).
#
#   ci/deploy.sh production        $APP_NAME, custom domain $APP_DOMAIN if set → .deploy-url
#   ci/deploy.sh staging           $APP_NAME-staging, its own D1, noindex → .staging-url
#   ci/deploy.sh preview <name>    a Worker Preview of $APP_NAME, the shared preview D1, noindex → .preview-url
#
# Credentials: CLOUDFLARE_API_TOKEN/CLOUDFLARE_ACCOUNT_ID from the environment (Workers Builds' build token), else
# fnox. DEPLOY_CMD: a prebuilt workerdeploy (Workers Builds gets one from GitHub CI); default: go tool workerdeploy.
# The URL files live outside build/: workers-assets-gen wipes build/.
set -euo pipefail
kind=${1:?production, staging or preview <name>}
name=${2:-}
cf() { if [ -n "${CLOUDFLARE_API_TOKEN:-}" ] && [ -n "${CLOUDFLARE_ACCOUNT_ID:-}" ]; then "$@"; else fnox exec -- "$@"; fi; }
read -r -a deploy <<<"${DEPLOY_CMD:-go tool workerdeploy}"
# The compatibility date the Worker runs on in production is the one local workerd tests it with.
compat=$(sed -n 's/.*compatibilityDate = "\(.*\)".*/\1/p' workerd/config.capnp | sort -u)
[ "$(echo "$compat" | wc -l | tr -d ' ')" = 1 ] && [ -n "$compat" ] || { echo "✗ workerd/config.capnp: want one compatibilityDate, got: $compat"; exit 1; }

# Every deploy: the entry module, every other worker/*.mjs next to it, Static Assets, D1 with migrations, the Room
# Durable Object and the write limit.
common=(-compat "$compat" -main worker/index.mjs -assets dist/site -migrations migrations -d1-create
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
    [ -z "${APP_REPO:-}" ] || ann+=(-annotation "workers/repository_url=https://github.com/$APP_REPO")
    cf "${deploy[@]}" -name "$APP_NAME" -preview "$name" "${common[@]}" -d1 DB="$APP_NAME-preview" \
      -var "APP_ENV=preview $name" -var "APP_DOMAIN=" -var "APP_PREVIEW=1" "${ann[@]}" | tee .preview-url
    ;;
  *)
    echo "✗ $kind: production, staging or preview <name>"
    exit 1
    ;;
esac
