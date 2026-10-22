# Deploying

Three kinds of deploy, all through `ci/deploy.sh` (which bindings, D1 and Durable Objects each gets) and charter's
`go tool workerdeploy` (Cloudflare's REST API; no wrangler):

| Kind | Worker | When | URL |
| --- | --- | --- | --- |
| Preview | a Worker Preview of `$APP_NAME`, shared preview D1, noindex | every pushed branch, automatically | `https://<branch>-$APP_NAME.$CF_SUBDOMAIN.workers.dev` (posted on the PR) |
| Staging | `$APP_NAME-staging`, its own D1, noindex | every merge to `main`, automatically | `https://$APP_NAME-staging.$CF_SUBDOMAIN.workers.dev` |
| Production | `$APP_NAME`, custom domain `$APP_DOMAIN` | a release | `https://$APP_DOMAIN` |

`APP_NAME`, `APP_DOMAIN` and `CF_SUBDOMAIN` are in `mise.toml` `[env]`; `mise run rename` resets them.

## One build per commit

1. GitHub CI (`ci/check.sh`) builds the commit, publishes the build and the deploy tool to the `preview-builds`
   pre-release, and runs the tests.
2. Workers Builds (Cloudflare's GitHub app, `ci/workers-builds.sh`) downloads them, verifies the build against its
   manifest and deploys: the branch's preview, or staging for `main`, or production for `production`. It then
   deletes previews of branches that no longer exist.
3. If CI's build never comes (CI down, a fork, or `PREVIEW_BUILD=compile`), Workers Builds compiles it itself.

No Cloudflare credential is in GitHub: Workers Builds deploys with its own build token. `mise run cf:builds` sets up
its triggers once.

## Releasing

```sh
mise run release v1.3    # or push a v* tag from GitHub's Releases page
```

CI tests the commit, checks it is on `main`, tags it, and moves the `production` branch to it; Workers Builds then
deploys production. The same release tags `badge/`, `intl/` and `seo/` with their next version when they changed
since their last tag (`ci/module-tags.sh`), so other apps get every fix with `go get`. Once the Go proxy has those
tags (a few minutes), `mise run fresh:published` builds the READMEs' app from them, as someone running `go get` would.
`mise run release` releases the tip of `main`; to roll back, push a v* tag on an older commit of `main`. Never push to
`production` by hand.

`mise run deploy` deploys production from a machine with fnox credentials instead, after the full test and a token
check, and then audits the live site as Googlebot.

## Credentials

`CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` come from fnox (`fnox exec -- …`), or from the environment in
Workers Builds. Nothing reads a `.env` file.

## After a deploy

- `SMOKE_BASE=$(cat .deploy-url) mise run smoke`: every route of the deployed Worker.
- `E2E_BASE=… mise run e2e` and `LOAD_BASE=… mise run load`: the browser checks and the 1,000-socket check against it.
- `mise run tail`: its live logs (`go tool workertail`).
