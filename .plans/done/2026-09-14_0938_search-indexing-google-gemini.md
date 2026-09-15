# Get indexed by Google Search and Gemini

**Status:** Phases 1–6 done (live on https://go-htmx4.ubuntusoftware.net, sitemap submitted by API); left: Google's live tests (UI), a week-later index check, docs · **Created:** 2026-09-14 09:38 · **Refreshed:** 2026-09-14 16:40

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
      POST. [x] Registered in `main.go` next to `/sitemap.xml` by i18n Phase 8 (c87ac81).
- [x] `sitemap.xml`: done by i18n Phase 8 (c87ac81), generated in Go from the locale data, 56 URLs with alternates.
- [x] `httpx.NoIndex(h)` (sets `X-Robots-Tag: noindex`) and `httpx.Origin(r)` in `kit/httpx`, with tests.
- [x] `noindex` on non-page responses, as **one middleware** in `robots.go` wrapped around the mux inside `withLocale`
      (so `/de/fragments/…` is matched on its unprefixed path): `/fragments/*`, `/healthz` and every request that isn't
      GET/HEAD (`/greet`, `/preferences`, `/board/add`, `/board/note`, and any future form post). One rule instead of
      wrapping each handler, so a new fragment or POST route can't forget it. `/live/` isn't a page. Pages, `/board` for
      every topic, `/robots.txt` and `/sitemap.xml` stay without the header. Main.go edit agreed with go-htmx4-87 first.
      — `noindexNonPages` (design by go-htmx4-12), `return withLocale(noindexNonPages(mux))`. `/robots.txt` also 404s under
      locale prefixes like `/sitemap.xml`.

### Phase 2: Open Graph (after i18n Phase 8)

- [-] Canonical, description, `hreflang` — moved to i18n Phase 8. Note for it: keep `?topic=` in the board's canonical for
      non-default topics (`?topic=lobby` → `/board`), since each topic is its own indexable page.
- [x] `og:title`, `og:description`, `og:url` (= Phase 8's canonical), `og:locale` and `og:locale:alternate` from the same
      locale data, in `Layout` only (never in `BoardFragment`).
- [x] No JSON-LD (Google says it isn't needed for AI features) and no llms.txt.

### Phase 3: tests

- [x] Go tests: `/robots.txt` served natively, `noindex` on every fragment/POST route (a table test over the mux), OG
      tags on each page (after Phase 2). Sitemap tests belong to i18n Phase 8.
- [x] Smoke under workerd (TinyGo): `/robots.txt` has our `User-agent` line, a fragment carries `X-Robots-Tag: noindex`.
      `mise run check` green.

### Phase 4: deploy and verify (⚠ needs OK)

- [x] `mise run deploy`, then `smoke-remote`: `/robots.txt` is ours (no Cloudflare notice, no prepended `Disallow`), and
      `/sitemap.xml` returns 200 once i18n Phase 8 has shipped. — 7d6915f deployed by go-htmx4-87 (smoke-remote 26/26,
      live e2e 11/11); again on the custom host with c117ad6 (26/26).
- [-] Not needed: Googlebot crawled and indexed 52 of 56 sitemap URLs within a day (2026-09-15, `mise run search:status`), so Google's fetcher gets the pages, not a Cloudflare challenge. Originally: **Real-Google check (UI only, no API):** `/`, `/board` and `/de/` on `https://go-htmx4.ubuntusoftware.net` through Google's
      [Rich Results Test](https://search.google.com/test/rich-results), which fetches from Google's machines. The rendered HTML
      must be our page, not a Cloudflare challenge.

### Phase 5: Search Console (by API, plus the UI for what has no API)

- [-] URL-prefix property for workers.dev + HTML verification file — superseded: the app moved to the custom domain, which
      your existing **Domain property `sc-domain:ubuntusoftware.net`** covers (no new verification).
- [x] **Search Console API tooling** (go-htmx4-87, commits 7937f11, b727f75): `kit/searchconsole` (service-account JWT
      auth, sitemaps submit/get, URL Inspection), `cmd/searchconsole`, `mise run search:sites|submit|status`
      (`tasks/search.toml`). Service account `search-console@go-htmx4-search.iam.gserviceaccount.com` in GCP project
      `go-htmx4-search` (no billing), key in fnox as `GOOGLE_SEARCH_CONSOLE_KEY`, added by you with Full access on
      `sc-domain:ubuntusoftware.net`.
- [x] Submit `https://go-htmx4.ubuntusoftware.net/sitemap.xml` — submitted and downloaded (~2026-09-14 17:10): 56 URLs,
      0 errors, 0 warnings. URL Inspection by API: 46 "Discovered – currently not indexed", 10 "URL is unknown to Google",
      no canonical conflicts.
- [-] Not needed: the submitted sitemap gets the remaining URLs crawled; Request indexing only speeds that up. `mise run search:todo` flags real problems (and opens only those). Originally: *Test live URL* and *Request indexing* have no API: in the Search Console UI for `/`, `/board`, `/de/` (the live
      test is the real-Googlebot fetch that would show a Cloudflare block).
- [-] Moved to routine use, not a plan step: `mise run search:todo` (URLs not yet indexed) and `mise run search:google` (Crawl stats → Host status). Originally: A week later (~2026-09-21): `mise run search:status` for index coverage, plus **Crawl stats → Host status** in the UI
      (no API), which is where a Cloudflare block would show. Note results here.

### Phase 6: custom domain (done with the [custom-domains plan](2026-09-14_0915_workers-custom-domains.md))

- [x] https://go-htmx4.ubuntusoftware.net attached (c117ad6). No static files to update: robots.txt, sitemap, canonical,
      hreflang and og:url use the request origin. `*.workers.dev` 301s pages to it in Go (`host.go`; `/healthz`, `/live/*`
      and form posts exempt). Domain property already existed. [-] *Change of address*: not needed (workers.dev was never
      indexed: 0 indexed pages before the move).
- [x] Zone security read (2026-09-14 18:25, `ubuntusoftware.net`): Bot Fight Mode off, block AI bots and crawler
      protection off, `ai_training`/`ai_search`/`ai_user` "disabled" (no blocking rule), managed robots.txt off
      (`policy_only`), security level low, Browser Integrity Check on (verified bots pass), no custom WAF or rate-limit
      rules. Nothing blocks Googlebot as configured. [-] Superseded: the zone's AI Crawl Control crawler table isn't needed, since Googlebot's 52 indexed pages show it gets through. Originally: look at the zone's AI Crawl Control crawler table in the
      dashboard after Google has crawled, to see Googlebot / Google-Agent requests let through.

### Phase 7: docs

- [x] README (indexing status) and AGENTS.md crawl rules:
  - `robots.txt` is a Go route (one `*` group, origin from the request); the sitemap is generated (i18n Phase 8)
  - fragments live under `/fragments/`, so the middleware marks them `noindex`; form posts are covered by method
  - canonical + description on new pages
  - new pages appear in the generated sitemap
  - `mise run rename` keeps the host right

## Open questions

- None blocking.
- If real Google requests turn out to be blocked on workers.dev, we can't change that zone. The fix is bringing a custom
  domain forward (Phase 6), where we control AI Crawl Control.

## Findings (implementation)

### 2026-09-14 17:40: Phases 1–3

- **robots.txt:** a Go route (`robots.go`) because `Sitemap:` must be absolute and the host differs per environment,
  account and after `mise run rename`; `httpx.Origin(r)` supplies it. 404 under locale prefixes (only the root is read).
- **noindex:** one middleware (`noindexNonPages`) inside `withLocale`: `/fragments/*` (any locale prefix), `/healthz`,
  every non-GET/HEAD request. Pages, every `/board` topic, `/robots.txt`, `/sitemap.xml` stay indexable; the i18n
  middleware's own incomplete-catalog noindex is untouched ("translated page indexable" smoke still green).
- **Open Graph** in `views/Layout`: `og:type`, `og:site_name`, `og:title` (= `<title>`), `og:description` (= the translated
  meta description), `og:url` (= canonical), `og:locale` in language_TERRITORY from the likely-subtags maximum
  (`OGLocale`: en → en_US, en-IN → en_IN, zh-Hant → zh_TW, zh-Hans → zh_CN, ar → ar_EG), `og:locale:alternate` for the
  other 13 locales.
- **Tests:** `TestNoIndexNonPages` (20 requests, prefixed fragment included), `TestRobotsTxt` + `TestRobotsTxtOnlyAtRoot`,
  `TestOpenGraph` (9 pages/locales: og values equal canonical/title/description, 13 unique alternates), plus the i18n
  session's `TestCanonicalAndHreflang`, `TestSitemap`, `TestNoHardcodedText`. Smoke (local + remote): robots.txt is
  ours, fragment noindex, Open Graph url + locale. `mise run check` 49 ✓, `mise run e2e` 11/11.
- **Three sessions coordinated** on these files: go-htmx4-87 (i18n Phase 8, registered the route), go-htmx4-12 (the
  middleware design and plan edits, then stood down), this session (implementation).

### Follow-ups

- [x] Phase 4: deploy by go-htmx4-87 (authorized for the shared i18n + search deploy), then Rich Results Test on `/`,
      `/board`, `/de/`; go-htmx4-12 runs independent read-only live checks. — deployed; go-htmx4-12's live checks 24/24
      on the custom host (workers.dev /about 301s to it). Rich Results Test is still open in Phase 4 (UI only).
- [-] Phase 5: Search Console (superseded: `sc-domain:ubuntusoftware.net` covers the custom host; see Phase 5). Note: `ubuntusoftware.net` already has a `google-site-verification` TXT record, so you may
      already own a Domain property there; once the app moves to a subdomain of it (custom-domains plan) that property
      covers it. For `go-htmx4.gedw99.workers.dev` a URL-prefix property + HTML file (a root route) is still needed.

### 2026-09-15: Google's verdict, live audit, snippet widths

- **Index coverage (Search Console API, `mise run search:status`):** 52 of 56 sitemap URLs "Submitted and indexed".
  Not yet: `/` and `/zh-hans/formats` "Discovered – currently not indexed" (never crawled), `/zh-hans/board` "URL is
  unknown to Google", `/en-in/about` "Duplicate, Google chose different canonical" (Google picked `/about`).
- **Live audit as Googlebot** (new `mise run search:audit`, `cmd/searchconsole audit`): robots.txt and all 56 URLs answer
  200, indexable, self-canonical, `lang`, one `<h1>`, reciprocal hreflang with x-default. Only failure: meta descriptions
  too long for a result on `/` (10 locales, 163–211) and `/formats` (5 locales), because both pages reused their visible
  intro as the description.
- [x] Fix: dedicated `home.description` and `formats.description` keys in all 13 full catalogs (`en-IN` inherits `en`),
      ≤ 155 wide; views use them. `TestSnippetWidths` renders every sitemap URL and fails on a title > 60 or description
      > 160 wide (`kit/searchconsole.SnippetWidth`, CJK counts double); mutation-checked (the old intros fail 11 pages).
- [x] Tooling so Google's own view is one command: `search:todo` (URLs not indexed, why, opens their Search Console
      inspection pages + Rich Results Test), `search:google` (site: search, Pages, Sitemaps, Crawl stats, Inspection,
      Rich Results Test, PageSpeed Insights). `jq` pinned in `mise.toml`. `mise run check` green.
- [x] Deployed 912dc70 with your OK (2026-09-15): `smoke-remote` 26/26; `mise run search:audit` live: robots.txt and all
      56 sitemap URLs pass.
- [-] Not needed (see Phase 5): Google crawls the 3 remaining URLs from the sitemap. Originally: `mise run search:todo`: *Request indexing* for `/`, `/zh-hans/formats`, `/zh-hans/board` and *Test live URL* /
      Rich Results Test for `/`, `/board`, `/de/` (UI only: the task opens the pages).
- [x] `/en-in/about` decision: accept. Its text is the English page (en-IN only differs in formats), and Google folds
      same-language copies with the same content into one canonical while hreflang still serves `/en-in/` URLs to India
      (https://developers.google.com/search/docs/specialty/international/localized-versions). Not a defect to fix here.
- [x] PageSpeed Insights API: anonymous daily quota was exhausted (HTTP 429); `search:google` opens the PageSpeed page
      instead of calling the API (an API key would be a GCP write).

### Closed 2026-09-15

Done: robots.txt, sitemap, noindex, Open Graph, snippet widths, custom domain, Search Console API tooling, live audit.
Google's verdict at close: 52 of 56 sitemap URLs indexed; the rest are queued from the sitemap (`/en-in/about` folded
into `/about` by design). Ongoing checks: `mise run search:audit` after deploys, `search:todo` and `search:google` any
time.
