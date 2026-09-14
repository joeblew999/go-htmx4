# Get indexed by Google Search and Gemini

**Status:** refreshed for the one-app repo, waiting for go · **Created:** 2026-09-14 09:38 · **Refreshed:** 2026-09-14 16:40

Written this morning against the two demos. It was never committed, and those demos, their Workers and `page.html` are
gone. The decisions below are kept; everything else is rewritten for the app as it is now: one Worker at
https://go-htmx4.gedw99.workers.dev, gsx `views/Layout`, 14 locales with URL prefixes. Nothing has been executed.

## Decisions (2026-09-14, unchanged)

- **Index `*.workers.dev` now.** Custom domains come later ([parked plan](2026-09-14_0915_workers-custom-domains.md));
  Phase 6 covers the move then.
- **Allow Gemini training and grounding:** no `Google-Extended` restriction, `Content-Signal: … ai-train=yes`.
- **The shared board is indexable**, including other topics and their notes.

## Split with the i18n plan (agreed with the i18n session, go-htmx4-87)

[Full i18n plan](2026-09-14_1106_full-i18n.md) Phase 8 owns everything per-locale:
- translated `<title>` and description
- per-locale canonical
- reciprocal `hreflang` + `x-default`
- sitemap `xhtml:link` alternates
- Search Console checks of localized URLs

This plan owns what is locale-independent:
- robots.txt and the sitemap file itself
- `noindex` on non-page routes
- the Cloudflare question
- Search Console setup

The two plans meet in `views/Layout` and the sitemap, so whoever goes second builds on the other's work.

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
| robots.txt, sitemap.xml, Search Console verification file | **Static files** in Static Assets (and the native file server) | Go handlers or `SITE_URL` plumbing for them |
| Localized sitemap entries | Generated once from the i18n locale list (`I18N_LOCALES`) by a small `mise run` step, or hand-written if the i18n plan prefers | A sitemap service |
| Canonical host | The request host now; a 301 to the custom domain later | A `SITE_URL` binding |
| "Does real Google get through Cloudflare?" | Google's **URL Inspection live test**, **Rich Results Test**, **Crawl stats** | A crawler log or IP matching |
| Keep fragments out of the index | An `X-Robots-Tag: noindex` header on those routes (one helper in `kit/httpx`) | — |

## Plan

### Phase 1: crawl files + noindex (locale-independent)

- [ ] `web/root/robots.txt`, copied to the `dist/site` root by `mise run assets`:
      ```
      User-agent: *
      Content-Signal: search=yes, ai-input=yes, ai-train=yes
      Allow: /

      Sitemap: https://go-htmx4.gedw99.workers.dev/sitemap.xml
      ```
      Use one `*` group only: a named group (e.g. `User-agent: Googlebot`) would make that bot ignore the `*` group. No group
      for `Google-Extended` means it's allowed. Google doesn't read `Content-Signal`, but it replaces Cloudflare's neutral
      notice and states the same intent to other crawlers. The host comes from `APP_NAME` + subdomain at build time, so
      `mise run rename` stays correct.
- [ ] `web/root/sitemap.xml` with the pages `/`, `/about`, `/board` and `/formats`. Board topics are unbounded; Google finds
      linked ones. Leave out `lastmod` unless it's real. Per-locale URLs + `xhtml:link` alternates come from i18n Phase 8;
      until then this lists only the English URLs.
- [ ] `httpx.NoIndex(h)` (sets `X-Robots-Tag: noindex`) on `/fragments/*`, `/healthz` and every POST route. `/live/` isn't
      a page. `/board` stays indexable for every topic.

### Phase 2: page head (joins i18n Phase 8)

- [ ] `views/Layout` gets `description` and `canonical`, and renders `<meta name="description">`, `<link rel="canonical">`
      and minimal Open Graph (`og:title`, `og:description`, `og:url`).
  - Canonical is scheme + host + path with no query string, except the board: keep `?topic=` for non-default topics
    (`?topic=lobby` → `/board`).
  - i18n Phase 8 makes these per-locale and adds `hreflang`/`x-default`. Do this in one pass with that phase if it's close.
- [ ] No JSON-LD (Google says it isn't needed for AI features) and no llms.txt. Head tags only in `Layout`, never in
      `BoardFragment` (wire format untouched).

### Phase 3: tests

- [ ] Go tests: `/robots.txt` and `/sitemap.xml` served natively (the sitemap parses with `encoding/xml`, absolute URLs
      on the app host), `noindex` on every fragment/POST route (a table test over the mux), canonical + description on
      each page.
- [ ] Smoke under workerd (TinyGo): `/robots.txt` has our `User-agent` line, `/sitemap.xml` 200 `application/xml`, a
      fragment carries `X-Robots-Tag: noindex`. `mise run check` green.

### Phase 4: deploy and verify (⚠ needs OK)

- [ ] `mise run deploy`, then `smoke-remote`: `/robots.txt` is ours (no Cloudflare notice, no prepended `Disallow`), and
      `/sitemap.xml` returns 200.
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
  - static `robots.txt`/`sitemap.xml` in `web/root/`, one `*` group
  - `noindex` on new fragment/POST routes
  - canonical + description on new pages
  - new pages added to the sitemap
  - `mise run rename` keeps the host right

## Open questions

- None blocking.
- If real Google requests turn out to be blocked on workers.dev, we can't change that zone. The fix is bringing a custom
  domain forward (Phase 6), where we control AI Crawl Control.
