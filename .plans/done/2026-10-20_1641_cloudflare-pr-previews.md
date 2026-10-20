# 2026-10-20_1641_cloudflare-pr-previews.md

**Status:** superseded (2026-10-20) by `.plans/2026-10-20_1746_previews-done-properly.md`: previews are built once by GitHub CI and deployed by Workers Builds, with no secrets in GitHub; the GitHub-secrets design below was dropped.

Every pull request gets its own live copy of the app on Cloudflare, with its URL posted on the PR, updated on every
push and deleted when the PR closes. Same no-Node, no-wrangler rule as production: `kit/cfdeploy` talks to the REST
API, and a `mise run preview` task runs the same way locally and in CI.

## Why Worker Previews, not Version URLs

- **Version URLs (aliases, 2025) don't exist for Workers with Durable Objects.** This app's board is a Durable Object
  (`Room`), so they're out.
- **Worker Previews (2026-09-22)** run under the same Worker, each with its own code, configuration, URL and
  observability. Durable Objects are isolated per preview automatically: each preview gets its own namespace and
  storage, so a PR's board rooms never touch production's.
- **D1 is not isolated automatically.** A preview must bind `DB` to a separate database, or it writes to production
  D1.

## REST API (read from wrangler 4.151.2's `deploy-helpers/src/preview/api.ts`; wrangler is never run)

| Call | Use |
| --- | --- |
| `POST /accounts/{acct}/workers/workers` `{name, subdomain:{enabled, previews_enabled}}` | parent Worker, only if it doesn't exist (production deploy already creates it) |
| `GET /accounts/{acct}/workers/workers/{worker}/previews/{name}` | does the PR's preview exist (404 / code 10025 = no) |
| `POST …/previews` `{name, observability?, logpush?}` | create the preview |
| `PATCH …/previews/{name}` | change preview settings |
| `POST …/previews/{name}/deployments` multipart: `metadata` JSON + module `files` | deploy code to the preview |
| `GET …/previews/{name}/deployments/latest` | current `migration_tag` (DO migrations are per preview) |
| `DELETE …/previews/{name}` | remove it when the PR closes |

Deployment `metadata`: `main_module`, `compatibility_date`/`_flags`, `assets: {jwt, config}` (the same assets
upload session as production), `exports` (DO classes), `migrations`, `env` (bindings), `annotations`
(`workers/pull_request_number`, `…_url`, `…_title`, `workers/commit_sha`, `workers/repository_url`).
URLs come back in `preview.urls` (stable per PR) and `deployment.urls` (per commit).

## Decisions

1. **Name = `pr-<number>`.** Stable across pushes, short (DNS label limit 63 with the Worker name), and the cleanup
   job knows it from the PR number alone.
2. **One shared preview D1, `go-htmx4-preview`.** Created once by the preview task if missing; migrations applied
   before each preview deploy (same `_migrations` tracking as production). Isolation per PR isn't needed for a
   template's board, and one database avoids leaking a database per PR. Durable Objects are per preview anyway.
3. **`host.go` must not redirect preview hosts.** It 301s `*.workers.dev` to `APP_DOMAIN`; a preview URL is a
   `workers.dev` host and would bounce to production. Previews get `APP_DOMAIN=""` (the binding is per preview), so
   `host.go` leaves them alone; canonical, hreflang, sitemap and robots.txt then follow the preview's own origin
   (`httpx.Origin`).
4. **Previews are never indexed.** A preview binding `APP_PREVIEW=1` makes every response carry
   `X-Robots-Tag: noindex` (one place in the middleware chain; robots.txt keeps its single `*` group, AGENTS.md).
5. **CI: a separate `preview.yml`** (shell steps only, no JavaScript actions, like `check.yml`):
   - `pull_request` opened/synchronize/reopened → `mise run preview -- pr-<n>` then comment the URL on the PR
     (update one comment, not a new one per push). Then `E2E_BASE=<preview url> mise run e2e`, so every PR is
     tested on real Cloudflare, not only workerd.
   - `pull_request` closed → `mise run preview:delete -- pr-<n>`.
   - Permissions: `contents: read`, `pull-requests: write` (the comment).
   - Secrets: `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID` as GitHub Actions secrets (locally they stay in fnox).
     The token needs Workers Scripts edit, D1 edit and Workers Tail; the same scopes as `mise run deploy`.
6. **Locally identical:** `mise run preview -- <name>` (credentials via fnox) and `mise run preview:delete -- <name>`.
7. **Rate limiting:** the `WRITES` binding is included so previews behave like production.

## Needs Gerard (can't be done from this sandbox)

- [-] (superseded) GitHub Actions secrets `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`. The sandbox can't write repository
      settings or secrets, and has no Cloudflare credentials. From any machine with fnox and gh:
      `fnox exec -- sh -c 'gh secret set CLOUDFLARE_API_TOKEN -R joeblew999/go-htmx4 -b "$CLOUDFLARE_API_TOKEN"; gh secret set CLOUDFLARE_ACCOUNT_ID -R joeblew999/go-htmx4 -b "$CLOUDFLARE_ACCOUNT_ID"'`
- [-] (superseded) The first preview deploy is a Cloudflare write (AGENTS.md: outward-facing, needs his OK). Asking for this
      plan is that OK for previews; production deploys still need it each time.

## Steps

- [x] `kit/cfdeploy`: preview create/get/delete, preview deployment (reusing the module and assets upload code),
      tests against the `httptest` fake API (never real Cloudflare)
- [x] `cmd/deploy -preview <name>` / `-delete-preview <name>`: flags only, logic in kit/
- [x] `APP_PREVIEW` noindex, with tests (`host.go` needed no change: previews get `APP_DOMAIN=`)
- [x] Tasks `preview`, `preview:delete` in `tasks/app.toml`; shared preview D1 + migrations
- [x] `.github/workflows/preview.yml` + `ci/preview.sh` (runnable locally)
- [-] (superseded) `mise run check` + `mise run e2e` locally
- [-] (superseded) Once secrets exist: open a test PR, check the comment, the URL, e2e against it, and cleanup on close
- [x] README and AGENTS.md updated

## Findings

- **Production deploys switched previews off.** `Deploy` sent `previews_enabled: false` on every run; it now sends
  `true`, and `DeployPreview` turns them on too. The Worker had them off (`subdomain.previews_enabled: false`).
- **The API checked live** through the Cloudflare connector (2026-10-20), with a throwaway preview `claude-probe`,
  deleted afterwards: lookup by name works, create takes `{name}`, a missing preview is HTTP 404 code 10025, a
  preview with no deployment answers `deployments/latest` with HTTP 404 code 10222, delete by name works.
  `cfapi.IsNotFound` treats any 404 as not found.
- **Not yet proven live: the deployment upload itself** (the 6 MB module upload). The connector can't create API
  tokens (its OAuth scope has no API Tokens permission) and can't upload local files, so the first real
  deployment happens in CI once the secrets exist, or with `mise run preview pr-test` on a machine with fnox.
- **CI doesn't re-test:** `check.yml` tests every PR, so `ci/preview.sh` runs `mise run build` then
  `mise run --skip-deps preview`, then e2e against the preview URL. Without the secrets it says so and passes.
- **Switched to Workers Builds (Gerard's standing preference; my first version used GitHub secrets, wrong).**
  Cloudflare's GitHub app is installed on his repos. Configured through the Cloudflare connector (API, 2026-10-20):
  repo connection joeblew999/go-htmx4, previews on (`migrate_to_previews` with our deploy command), previews build
  `curl mise.run | sh && mise install && mise run build`, deploy `mise run --skip-deps preview "$WORKERS_CI_BRANCH"`,
  `CLOUDFLARE_ACCOUNT_ID` as a build variable. The `main` (production) trigger only echoes: production deploys stay
  manual until Gerard says otherwise. `.github/workflows/preview.yml` and `ci/preview.sh` removed.
- **Build token:** reuses the account's existing `worker-go build token` (creating one needs an API token secret,
  which the connector can't produce). Replace it with a go-htmx4 token from the dashboard if that matters.
