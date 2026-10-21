---
name: seo
description: Use when a gsx app changes what Google reads (a new page, a title or description, a route, its noindex, robots.txt or sitemap rules), after a production deploy, or when asked how Google indexes the site, to fix what Google Search Console reports, or to get Lighthouse (performance, accessibility, best practices, SEO) right, with github.com/joeblew999/go-htmx4/seo (go tool seo, mise run seo:*).
---

# How Google sees the site

`seo.Site`, middleware around the app, answers robots.txt and the sitemap, marks what isn't a page noindex, and moves
other hosts and plain http to the one https address; the app says which paths are pages. The rest of what Google
reads is the app's own: canonical URLs, hreflang, `<html lang>`, titles and descriptions (with intl: `intl.Head`,
`intl.Sitemap`, `intl.HTML`, the catalogs' messages), the icon, structured data, the pages' text. Fix a problem there.
The `seo` tools check it and ask Google; see its README for the whole map.

Run every command from the app's root. The site is `https://$APP_DOMAIN`.

## 1. Check what the app serves

```sh
mise run seo:audit --json
```

It fetches robots.txt, the sitemap and every URL in it as Googlebot, and needs no Google account. `ok: false` comes
with `problems`, one line each, naming the URL. For another server (a local build, a preview) run
`go tool seo audit -json -base <url>`; in a Go test, `seo.Audit(ctx, client, base)` returns the same problems.

| Problem | Where the fix is |
|---|---|
| `HTTP 3xx/4xx` | A sitemap URL must answer 200 itself: fix the route, or take the path out of the sitemap's pages |
| `X-Robots-Tag: noindex`, `meta robots` | The app's noindex rule matched a page. intl.Site also sends noindex while a language's catalog is incomplete: translate it |
| `canonical … is not the page's own URL` | The page path given to `intl.Head` |
| `<html> has no lang` | The layout's `<html { intl.HTML(ctx)... }>` |
| `no <title>`, `no meta description`, `… characters wide` | The page's title and description messages: write them, or shorten them in the source catalog and every translation (Google's widths: `seo.MaxTitleWidth`, `seo.MaxDescriptionWidth`) |
| `N <h1> elements` | Exactly one `<h1>` per page |
| `hreflang …` | Every language version lists every other and itself, plus x-default, as absolute URLs on the site's own scheme and host: `intl.Head` does, unless a page is missing in a language |
| `robots.txt …` | One `User-agent: *` group that allows everything, and `Sitemap:` with the absolute sitemap URL |
| `… has invisible characters` | Bidi marks in a title: a message argument isolated where it needn't be (pass another message, not text, as the argument) |
| `no <link rel="icon">` | The layout's `<link rel="icon">` |
| `no og:title or og:description`, `og:url …` | The layout's Open Graph tags; og:url is the canonical (`intl.Head`) |
| `HTTP 200 for a page that doesn't exist` | A catch-all route answers 200: unknown paths must be 404 |
| `not a redirect to https` | `seo.Site`'s Host: the production domain |
| `no WebSite structured data` | The home page's `<script type="application/ld+json">` with `"@type": "WebSite"` and `name` |

## 2. Ask Google

These need `GOOGLE_SEARCH_CONSOLE_KEY` in fnox and that account's Full access to the property:
`mise run seo:sites` says whether it has it.

```sh
mise run seo:todo --json            # every sitemap URL; exits 1 when one is a problem
mise run seo:inspect /de/ --json    # one URL: verdict, coverage, last crawl, Google's canonical and the app's
mise run seo:status --json          # the sitemap's status, and every URL's coverage
```

`todo` gives `total`, `indexed` and `findings` (`url`, `state`, `note`, `problem`, `link`). Only `problem: true`
needs work:

| `state` | What to do |
|---|---|
| `URL is unknown to Google`, `Discovered - currently not indexed` | Nothing: Google crawls it from the sitemap, in days or weeks |
| `Duplicate, Google chose different canonical than user` | Usually nothing: the same content under another URL, often a regional copy of a language (en-IN of en). Google sends that language's readers to the right URL through hreflang |
| `Crawled - currently not indexed` | Google chose not to index it, usually for thin or duplicate-looking content: make the page's text, title and description its own |
| A failed verdict (404, noindex, blocked by robots.txt, a server error) | Find the cause with step 1, fix it, deploy, audit again |

Google re-crawls on its own schedule. **Test live URL** and **Request indexing** have no API: give the person each
problem's `link` (`--open` opens them), don't try to do it for them.

## 3. Lighthouse

```sh
mise run seo:lighthouse --json             # every page of the live sitemap (or seo.toml's pages), mobile
mise run seo:lighthouse /de/board --json   # one page; --desktop for the desktop one
```

The settings (the site, the property, Lighthouse's pages, device and minimum score, where badges go) are the app's
`seo.toml`, each with a default; an argument overrides one for one run. After a production release,
`mise run seo:badges` writes the README's Lighthouse and audit badges and its scores table from a fresh run: commit
them, never edit them by hand.

It runs on Google's servers (PageSpeed Insights), so only on a live site: production, or a branch preview (whose
SEO score reads noindex, by design). Each category under 90 fails; `failed` lists the checks with `items`, the
elements or files they failed on. Real users' Core Web Vitals come in `field` once Chrome has enough visits.

| Fails | Where the fix is |
|---|---|
| Render-blocking requests | A script in `<head>` without `defer` or `type="module"`. Only what must run before first paint (the theme) stays blocking |
| Minify JavaScript, CSS | The build minifies what it ships |
| ARIA, names, contrast, touch targets | In the app's markup: fix it. In a vendored component (gsxui's `ui/`, never edited): give it what it takes (an `aria-label`, an `id` and `aria-controls`, a class), else report it upstream with Lighthouse's items as evidence |
| Errors in the console | Read the items. A WebSocket that can't connect is Lighthouse's sandbox, not the site |

## 4. What needs the person's OK

`seo:submit` (sends the sitemap to Google) and `seo:setup` (a Google Cloud project, a service account, its key) write
to Google: run them only when asked; both ask before they act. Every other task only reads. The key stays in fnox
(`fnox exec --`): never print it.

## Adding a page Google should index

A title and a description in the catalog, within Google's widths; exactly one `<h1>`; `intl.Head` with the page's
path; the path in the sitemap's pages; not under a path the app marks noindex. Then the audit against a local build,
and `mise run seo:audit` after the deploy. Google finds the page from the sitemap.
