---
name: seo
description: Use when a gsx app changes what Google reads (a new page, a title or description, a route, its noindex, robots.txt or sitemap rules), after a production deploy, or when asked how Google indexes the site or to fix what Google Search Console reports, with github.com/joeblew999/go-htmx4/seo (go tool seo, mise run seo:*).
---

# How Google sees the site

`seo` checks what the app serves Google and asks Google what it made of it. It produces nothing itself: robots.txt,
the sitemap, canonical URLs, hreflang, `<html lang>`, titles and descriptions are the app's own code (with intl:
`intl.Head`, `intl.Sitemap`, `intl.HTML`, and the catalogs' messages). Fix a problem there, never in `seo`.

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

## 3. What needs the person's OK

`seo:submit` (sends the sitemap to Google) and `seo:setup` (a Google Cloud project, a service account, its key) write
to Google: run them only when asked; both ask before they act. Every other task only reads. The key stays in fnox
(`fnox exec --`): never print it.

## Adding a page Google should index

A title and a description in the catalog, within Google's widths; exactly one `<h1>`; `intl.Head` with the page's
path; the path in the sitemap's pages; not under a path the app marks noindex. Then the audit against a local build,
and `mise run seo:audit` after the deploy. Google finds the page from the sitemap.
