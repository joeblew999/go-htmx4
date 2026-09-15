# Custom domain for the app: go-htmx4.ubuntusoftware.net

**Status:** live on https://go-htmx4.ubuntusoftware.net (2026-09-14 18:30); Search Console left · **Created:** 2026-09-14 09:15

## Goal

Serve the app (Worker `go-htmx4`) on **https://go-htmx4.ubuntusoftware.net**. Attach it with our Go deploy client, with no
dashboard clicks and no wrangler. Make that the one URL search engines index, so `*.workers.dev` redirects to it.

## Decisions

- **Zone:** `ubuntusoftware.net`, your choice. The zone is active, on the Free plan, and in the same Cloudflare account as
  the Worker.
- **Hostname:** `go-htmx4.ubuntusoftware.net`, a subdomain (my default, since the name matches the Worker and D1). The zone
  apex and `www` are the `ubuntu-website` Pages site. Other subdomains belong to other projects (`remy`, `help.remy`,
  `staging-remy`, `ifcsketch*`, `auth`, `dev-remy` tunnel, `plugs` GitHub Pages, mail records), so nothing of theirs is
  touched.
- **workers.dev:** stays enabled for `mise run smoke-remote`/e2e fallbacks, but **301s pages to the custom host** once it's
  attached. Otherwise canonical URLs (built from the request origin) would make both hosts self-canonical duplicates.
  `/healthz`, `/live/*` (WebSocket upgrades can't follow redirects) and non-GET requests are exempt. This joins the
  [search plan](done/2026-09-14_0938_search-indexing-google-gemini.md)'s Phase 6.

## Ground rules

- No Node, no wrangler: Cloudflare REST API via `kit/cfdeploy` / `cmd/deploy`, credentials via fnox.
- **Attaching a domain is outward-facing (DNS + certificate).** You gave the zone; the hostname above is the only record
  created, and nothing else in the zone changes.
- **Shared checkout:** `tasks/app.toml`, `mise.toml` and `main.go` are coordinated with the i18n session (go-htmx4-87);
  stage explicit paths only.

## Upstream API

- **Workers Custom Domains:** `GET /accounts/{account_id}/workers/domains` lists `{id, zone_id, zone_name, hostname,
  service, environment, cert_id, enabled, previews_enabled}` (checked live). `PUT /accounts/{account_id}/workers/domains`
  with `{hostname, service, zone_id, environment}` attaches. `DELETE …/workers/domains/{id}` removes.
- [Custom Domains docs](https://developers.cloudflare.com/workers/configuration/routing/custom-domains/): "Cloudflare will
  create DNS records and issue necessary certificates on your behalf"; you "cannot create a Custom Domain on a hostname
  with an existing CNAME DNS record". `go-htmx4.ubuntusoftware.net` has no record (checked 2026-09-14).
- **Zones visible to the token:** `amplify-cms.{com,net,org}`, `amplifycms.{com,net,org}`, `ubuntudesign.com`,
  `ubuntusoftware.net`.

## Findings (read-only checks, 2026-09-14 17:45)

- **DNS:** `ubuntusoftware.net` → CNAME `ubuntu-website.pages.dev` (apex 301 → `www`), `www` likewise. The remy,
  ifcsketch and auth Workers are Custom Domains (AAAA `100::` placeholders). There are no Worker routes on the zone.
- **Search Console:** the apex has a `google-site-verification` TXT record, so you probably already have a
  **Domain property** for `ubuntusoftware.net`. It would cover `go-htmx4.ubuntusoftware.net` without new verification.
- **`https://go-htmx4.ubuntusoftware.net/`:** doesn't resolve yet (as expected).

## Plan

### Phase 1: deploy client

- [x] `kit/cfdeploy` `Config.Domains` + `cmd/deploy -domain host` (repeatable):
  - lists the account's custom domains; skips when the hostname is already attached to this Worker, and refuses when it's
    attached to another
  - finds the zone by walking the hostname's parents (in this account only), then `PUT`s
  - tested on the fake API: attach once, idempotent redeploy, taken hostname refused, zone not in account refused
  - `-dry-run` prints the domains

### Phase 2: attach (your zone OK given; after the shared deploy)

- [x] `APP_DOMAIN = "go-htmx4.ubuntusoftware.net"` in `mise.toml` `[env]` (`mise run rename` clears it: a renamed copy
      has no domain until its owner sets one). The `deploy` task passes `-domain "$APP_DOMAIN"` when it's set, and
      `.deploy-url` / smoke-remote use the custom host.
- [x] Deploy, wait for the certificate, then run `smoke-remote`, `LOAD_BASE` load (WebSockets over the custom host) and
      `E2E_BASE` e2e against `https://go-htmx4.ubuntusoftware.net`.

### Phase 3: one indexed host (with search plan Phase 6)

- [x] A middleware (with `APP_DOMAIN` as a text binding): requests to any other host 301 to the same path on
      `https://$APP_DOMAIN`, except `/healthz`, `/live/*` and non-GET/HEAD requests. Tests: redirect, exemptions, and no
      redirect when `APP_DOMAIN` is empty (template default). — `host.go` `canonicalHost`, outermost handler; only
      `*.workers.dev` hosts redirect (never localhost, so `mise run run` with `APP_DOMAIN` in the env is unaffected);
      `TestCanonicalHost`. `APP_ENV` on Cloudflare is now plain `cloudflare` (it was `cloudflare (workers.dev)`).
- [x] Canonical, hreflang, sitemap and robots.txt then naturally carry the custom host (they use the request origin).
- [ ] Search Console: confirm the existing `ubuntusoftware.net` Domain property covers the host (or add a URL-prefix
      property). Submit `https://go-htmx4.ubuntusoftware.net/sitemap.xml`.

### Phase 4: docs

- [x] README live URL → custom host; AGENTS.md: `APP_DOMAIN`, the redirect exemptions, "don't take hostnames that belong
      to other Workers in the zone".

## Open questions

- None blocking. If you'd rather use a different subdomain (e.g. `htmx.ubuntusoftware.net`), say so before Phase 2.

## Findings (attach, 2026-09-14 18:30)

- Deploy c117ad6 from a clean worktree: `✓ custom domain go-htmx4.ubuntusoftware.net attached`; Cloudflare issued the
  certificate (`cert_id` set, `enabled`) within seconds.
- **Live:** TLS valid; robots.txt `Sitemap:`, canonical, `og:url` and the sitemap `<loc>`s all on the custom host.
  workers.dev: pages, robots.txt and query strings 301 to it; `/healthz` 200; POST `/greet` 200 (not redirected).
- **Checks on the custom host:** `smoke-remote` 26/26; e2e 11/11 after one test fix; load: 1000/1000 connected and
  delivered, presence 1000, late joiner cached. One check fell short: presence after closing 500 read 509 within its
  5 s wait (1,000 sockets took 17.5 s to connect on this network), so it's timing, not a Room bug; re-check on a
  faster link.
- **This Mac's DNS cache** kept "no such host" from a lookup made before the record existed (zone negative TTL 1800 s).
  Checks ran meanwhile with curl DNS-over-HTTPS and `GODEBUG=netdns=go`. You flushed it with
  `sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder`.
- **e2e write-limit test** hit its 2-minute timeout on the custom host: over a slower path, one-at-a-time writes stay
  under 60 per 10 s, so the limiter never answered. It now sends parallel batches of 20 until one returns 429 (live: 67× 200,
  13× 429, click 429, toast, board unchanged). In one earlier run a 70-write parallel burst from Chrome also got
  32×503, 1×502 and 1×500; it didn't recur in two reruns or with 70 parallel curl writes on either host, and the tail
  showed no Worker errors. Watch for it.
- **Zone security (read-only, `ubuntusoftware.net`):** Bot Fight Mode off, block AI bots / crawler protection off,
  Cloudflare-managed robots.txt off (`cf_robots_variant: policy_only`, only used when the origin has none — ours
  does), security level low, Browser Integrity Check on (verified bots such as Googlebot pass), no custom WAF or
  rate-limit rules. Nothing here blocks Googlebot; proof still has to come from Google (URL Inspection / Rich Results
  Test).
