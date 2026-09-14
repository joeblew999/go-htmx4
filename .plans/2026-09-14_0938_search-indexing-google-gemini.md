# Get indexed by Google Search and Gemini

**Status:** go (2026-09-14 17:00, your answer here); Phase 1 in progress, sequenced with i18n Phase 8 · **Created:** 2026-09-14 09:38 · **Refreshed:** 2026-09-14 16:40

Written this morning against the two demos. It was never committed, and those demos, their Workers and `page.html` are
gone. The decisions below are kept; everything else is rewritten for the app as it is now: one Worker at
https://go-htmx4.gedw99.workers.dev, gsx `views/Layout`, 14 locales with URL prefixes. Nothing has been executed.

## Decisions (2026-09-14, unchanged)

- **Index `*.workers.dev` now.** Custom domains come later ([parked plan](2026-09-14_0915_workers-custom-domains.md));
  Phase 6 covers the move then.
- **Allow Gemini training and grounding:** no `Google-Extended` restriction, `Content-Signal: … ai-train=yes`.
- **The shared board is indexable**, including other topics and their notes.

## Split with the i18n plan (agreed with the i18n session, go-htmx4-87)

[Full i18n plan](2026-09-14_1106_full-i18n.md) Phase 8 owns (agreed 2026-09-14 16:50):
- **`/sitemap.xml`**, generated in Go from `cldr.Data.Locales` + `views.LocaleLinks`: every page in all 14 locales with
  reciprocal `xhtml:link` alternates. A static file would drift from `I18N_LOCALES`.
- `<link rel="canonical">`, `hreflang` + `x-default`, and a translated `<meta name="description">` message in `Layout`.
- Search Console checks of localized URLs.

This plan owns:
- `robots.txt` pointing at `/sitemap.xml` (a small Go route, see Phase 1)
- `noindex` on non-page routes
- Open Graph tags built on Phase 8's head (`og:locale` / `og:locale:alternate` from the same locale data)
- the Cloudflare question, Search Console setup, a later custom domain

Order: search Phase 1 can go before or after i18n Phase 8; search Phase 2 (OG) goes after it.

Current i18n facts (from the i18n session):
- Locale URLs are `/<lowercase id>/…`; English is unprefixed.
- Every locale's catalog is complete, so there's no `noindex` stopgap any more.
- `views.LocaleLinks(ctx)` lists the current page in every locale.

## Goal

The live app gets crawled and indexed by Googlebot and shows up in Google Search (and so in AI Overviews / AI Mode). It
also stays open to Gemini:
- training and grounding via `Google-Extended`
- the user-triggered `Google-Agent` and `Google-GeminiNotebook` fetchers

Everything is set up in Go, with no Node and no dashboard clicks, except the Search Console steps only you can do.

## Ground rules

- **Same as the app:** no Node, TinyGo for everything on Workers, plain-path routes + `allow()`, credentials via fnox.
- **Tests:** crawl files and headers are covered by `mise run check`, and by `mise run e2e` where a browser matters.
- **Outward-facing:** deploys and Search Console changes need your explicit OK.
- **Shared checkout:** it's shared with the i18n session. `views/`, `main.go`, `tasks/app.toml` and `e2e/` are currently
  its files, so agree before editing and stage explicit paths only.

## Upstream guidance

- **Google crawlers:** [common crawlers](https://developers.google.com/crawling/docs/crawlers-fetchers/google-common-crawlers)
  (updated 2026-07-14), [user-triggered fetchers](https://developers.google.com/crawling/docs/crawlers-fetchers/google-user-triggered-fetchers)
  (updated 2026-08-19).
  - `Googlebot` (smartphone + desktop): Search, Discover, Images, News. Respects robots.txt.
  - `Google-Extended`: a robots.txt token only (no user agent of its own; fetches are Googlebot's). Controls "training future
    generations of Gemini models" **and** "grounding … in Gemini Apps and Grounding with Google Search on Vertex AI".
    "Does not impact a site's inclusion in Google Search nor is it used as a ranking signal."
  - `GoogleOther`, `Google-CloudVertexBot`: R&D crawling and site-owner Vertex AI agents. Respect robots.txt.
  - `Google-Agent` and `Google-GeminiNotebook`: user-triggered, **ignore robots.txt**. They only need pages that load without
    bot challenges and work as plain HTML.
- **AI features:** [AI features and your website](https://developers.google.com/search/docs/appearance/ai-features) says
  "There are no additional requirements to appear in AI Overviews or AI Mode". A page must be "indexed and eligible to be
  shown in Google Search with a snippet". No llms.txt and no special schema.org needed.
- **Crawl control and verification:**
  - [robots.txt](https://developers.google.com/search/docs/crawling-indexing/robots/create-robots-txt)
  - [sitemaps](https://developers.google.com/search/docs/crawling-indexing/sitemaps/build-sitemap) (absolute URLs)
  - [canonical URLs](https://developers.google.com/search/docs/crawling-indexing/consolidate-duplicate-urls)
  - [`X-Robots-Tag`](https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag)
  - [Search Console verification](https://support.google.com/webmasters/answer/9008080)
- **Cloudflare:** [managed robots.txt / Content Signals Policy](https://developers.cloudflare.com/bots/additional-configurations/managed-robots-txt/).
  A host without its own robots.txt gets Cloudflare's policy text; serving our own replaces it.

## Findings (live check, 2026-09-14 16:35, https://go-htmx4.gedw99.workers.dev)

- **What works:** pages (`/`, `/about`, `/board`, `/formats`, `/de/…`) return 200 with full server-rendered HTML, `<html lang
  dir>`, `<title>` and `Content-Language`, and no `X-Robots-Tag`. Nav is boosted real `<a href>` links, and the locale
  switcher is plain links with `hreflang`. Good base.
- **`/robots.txt`** is still **Cloudflare's Content Signals notice**: comments only, with no `User-agent` and no `Sitemap:`.
- **`/sitemap.xml`:** 404. There's no `<meta name="description">`, `<link rel="canonical">`, `<link rel="alternate" hreflang>`
  or Open Graph in `<head>` (Phase 8 of the i18n plan adds the localized parts).
- **Non-page routes return 200 without `noindex`:** `/fragments/server-info`, `/fragments/stats`, `/fragments/preferences`,
  `/healthz`. The POST endpoints (`/greet`, `/board/add`, `/board/note`, `/preferences`) aren't crawled as GETs but should
  still send `noindex`. `/live/{topic}` answers 426 without an upgrade.
- **Where the files would go:** Static Assets serve `dist/site`, and `static/` is published under `/static/`, so a
  root-level `robots.txt` needs a new source folder copied to the `dist/site` root by `mise run assets` (e.g. `web/root/`).
- **Search Console:** `*.workers.dev` can only be a **URL-prefix** property (HTML file or meta tag). A **Domain** property
  (DNS TXT) needs a custom domain.
- **User-generated content:** the board has topics and anonymous notes on arbitrary topic URLs. Abuse limits are in place
  (rate limit, 50 notes per topic).

## Cloudflare blocking risk

Cloudflare can block Google before a request ever reaches the Worker, so our tests can't see it.

- **2026-09-15 defaults** ([announcement](https://blog.cloudflare.com/content-independence-day-ai-options/)):
  - AI crawlers are split into Search / Agent / Training.
  - For new domains, Training and Agent are blocked by default **on pages that display ads**; Search stays allowed.
  - "Multi-purpose crawlers such as Googlebot … will be blocked by customers who have selected to block Training."

  We show no ads and don't block Training, so as written Googlebot, Google-Agent and Google-Extended use are allowed.
- **[Managed robots.txt](https://developers.cloudflare.com/bots/additional-configurations/managed-robots-txt/)** (off by
  default) would put `User-agent: Google-Extended` / `Disallow: /` in front of our robots.txt. It must stay off.
- **workers.dev is Cloudflare's zone, not ours:** we can't see or change its AI Crawl Control or bot settings.
- **A spoofed user agent proves nothing:** Cloudflare identifies Googlebot by IP, so only requests from Google's own machines
  count (Phases 4–5).

## Reuse, don't build

| Need | Use | Not building |
| --- | --- | --- |
| robots.txt | A 10-line Go route using the request's origin (`httpx.Origin`) | A static file with a hard-coded host, or a `SITE_URL` setting |
| Search Console verification file | A **static file** in Static Assets (`web/root/`, copied by `mise run assets`) | — |
| Sitemap | Generated by the app from the locale data (i18n Phase 8) | A static sitemap that drifts, or a sitemap service |
| Canonical host | The request host now; a 301 to the custom domain later | A `SITE_URL` binding |
| "Does real Google get through Cloudflare?" | Google's **URL Inspection live test**, **Rich Results Test**, **Crawl stats** | A crawler log or IP matching |
| Keep fragments out of the index | An `X-Robots-Tag: noindex` header on those routes (one helper in `kit/httpx`) | — |

## Plan

### Phase 1: crawl files + noindex (locale-independent)

- [x] `robots.txt` as a Go route (`robots.go`), not a static file: the `Sitemap:` line must be an absolute URL, and a
      static file would hard-code `go-htmx4.gedw99.workers.dev` (wrong locally, after `mise run rename`, on another
      account's subdomain and on a custom domain). The route builds it from `httpx.Origin(r)`; Static Assets has no
      robots.txt, so the request reaches the Worker. Content:
      ```
      User-agent: *
      Content-Signal: search=yes, ai-input=yes, ai-train=yes
      Allow: /

      Sitemap: <origin>/sitemap.xml
      ```
      One `*` group only: a named group (e.g. `User-agent: Googlebot`) would make that bot ignore `*`. No group for
      `Google-Extended` means it's allowed. Google doesn't read `Content-Signal`; it replaces Cloudflare's neutral notice
      for other crawlers. `TestRobotsTxt` pins one group, no `Disallow`, no named Google groups, the sitemap URL, 405 on
      POST. [ ] Registered in `main.go` (`mux.HandleFunc("/robots.txt", robotsTxt)`) after i18n Phase 8, which owns
      main.go right now.
- [-] `sitemap.xml` — moved to i18n Phase 8 (generated in Go from the locale data). If search Phase 1 ships first,
      robots.txt's `Sitemap:` line points at a 404 until then; harmless, or hold the line back until Phase 8 lands.
- [x] `httpx.NoIndex(h)` (sets `X-Robots-Tag: noindex`) and `httpx.Origin(r)` in `kit/httpx`, with tests.
- [ ] Apply `NoIndex` to `/fragments/*`, `/healthz` and every POST route (`/greet`, `/preferences`, `/board/add`,
      `/board/note`) after i18n Phase 8 (main.go, board.go). `/live/` isn't a page. `/board` stays indexable for every
      topic.

### Phase 2: Open Graph (after i18n Phase 8)

- [-] Canonical, description, `hreflang` — moved to i18n Phase 8. Note for it: keep `?topic=` in the board's canonical for
      non-default topics (`?topic=lobby` → `/board`), since each topic is its own indexable page.
- [ ] `og:title`, `og:description`, `og:url` (= Phase 8's canonical), `og:locale` and `og:locale:alternate` from the same
      locale data, in `Layout` only (never in `BoardFragment`).
- [ ] No JSON-LD (Google says it isn't needed for AI features) and no llms.txt.

### Phase 3: tests

- [ ] Go tests: `/robots.txt` served natively, `noindex` on every fragment/POST route (a table test over the mux), OG
      tags on each page (after Phase 2). Sitemap tests belong to i18n Phase 8.
- [ ] Smoke under workerd (TinyGo): `/robots.txt` has our `User-agent` line, a fragment carries `X-Robots-Tag: noindex`.
      `mise run check` green.

### Phase 4: deploy and verify (⚠ needs OK)

- [ ] `mise run deploy`, then `smoke-remote`: `/robots.txt` is ours (no Cloudflare notice, no prepended `Disallow`), and
      `/sitemap.xml` returns 200 once i18n Phase 8 has shipped.
- [ ] **Real-Google check, no account needed:** `/`, `/board` and `/de/` through Google's
      [Rich Results Test](https://search.google.com/test/rich-results), which fetches from Google's machines. The rendered HTML
      must be our page, not a Cloudflare challenge.

### Phase 5: Search Console (you, in your Google account)

- [ ] Add a URL-prefix property for `https://go-htmx4.gedw99.workers.dev/`. Verify with the **HTML file** method: you give
      me `google<token>.html`, it goes in `web/root/`, then redeploy (⚠ OK).
- [ ] Submit `sitemap.xml`. URL Inspection → *Test live URL* (real Googlebot; shows blocked fetches) → *Request indexing*.
- [ ] A week later, note here the Pages report (indexed / excluded reasons) and **Crawl stats → Host status**, which is
      where a Cloudflare block would show.

### Phase 6: after a custom domain (optional; domains are parked)

- [ ] Point the static files at the custom hostname. 301 `*.workers.dev` → custom host in Go (keep `/healthz` exempt), or
      disable workers.dev. Add a Domain property (DNS TXT, ⚠ OK), and run *Change of address* if workers.dev was indexed.
- [ ] In that zone's **AI Crawl Control**:
  - Search, Agent and Training **allowed**
  - managed robots.txt **off**
  - "Block AI bots" **off**

  Its crawler table then shows Googlebot / Google-Agent requests and whether Cloudflare let them through.

### Phase 7: docs

- [ ] README (indexing status) and AGENTS.md crawl rules:
  - static `robots.txt` in `web/root/`, one `*` group; the sitemap is generated (i18n Phase 8)
  - `noindex` on new fragment/POST routes
  - canonical + description on new pages
  - new pages appear in the generated sitemap
  - `mise run rename` keeps the host right

## Open questions

- None blocking.
- If real Google requests turn out to be blocked on workers.dev, we can't change that zone. The fix is bringing a custom
  domain forward (Phase 6), where we control AI Crawl Control.
