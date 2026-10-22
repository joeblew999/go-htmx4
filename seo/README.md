# seo

Search engines, for a Go web app: the middleware that answers them, an audit of what they read, Lighthouse, and
Google Search Console, from one Go command, `go tool seo`, and the mise tasks and agent skill that run it.

## What Google reads, and who makes it

Google ranks what the site serves. Every piece is made by the app, live on each request; nothing is generated ahead:

| Google reads | What it's for | Made by (in go-htmx4) |
|---|---|---|
| `/robots.txt` | may it crawl, and where the sitemap is | `seo.Site` |
| `/sitemap.xml` | every page, in every language, with its other-language versions | `seo.Site`, writing `intl.Sitemap` of the app's pages (`app/site.go`) |
| `/llms.txt` | for agents and language models: what the site is and where to read more ([llmstxt.org](https://llmstxt.org)) | `seo.Site`'s `LLMs`, written with `seo.LLMsTxt` (a title, a summary, sections of links) |
| `X-Robots-Tag: noindex` | keeps fragments, posts, previews, incomplete languages and endless query pages out | `seo.Site` (the app's `isPage`, `indexable`) and `intl.Site` |
| one address per page | other hosts and plain http move to `https://` the domain | `seo.Site` |
| `<html lang>` | the page's language | `intl.HTML` |
| `<title>`, `<meta name="description">`, `og:title`, `og:description` | the blue link and text in a result, and how a shared link looks | `seo.Head`, from the catalogs (`views/layout.gsx`) |
| `<link rel="canonical">`, `hreflang` links, `og:url`, `og:locale` | which URL is the page, and its versions in other languages | `intl.Head`, with the page key (`views/views.go`) |
| `<link rel="icon">` | the icon beside the site's results | `seo.Head` |
| `WebSite` structured data | the site's name in results | `seo.Head`, on the home page |
| one `<h1>`, real text | what the page is about; a thin page gets crawled and not indexed | each page |

`seo` checks all of it, before a deploy and after, and asks Google what it made of it.

## In an app

`pages` are the app's pages, and `site` its [`intl.Site`](../intl/README.md):

```go
search := seo.Site{
	Host:    os.Getenv("APP_DOMAIN"), // the one address; "" in previews and locally
	Sitemap: func(origin string) string { return intl.Sitemap(origin, pages) },
	IsPage: func(path string) bool { // the request's path and query: "/de/about?ref=x"
		path, _, _ = strings.Cut(intl.Unprefixed(path), "?")
		return slices.Contains(pages, path)
	},
	Preview: os.Getenv("APP_PREVIEW") != "", // everything noindex
}
// …
log.Fatal(http.ListenAndServe(addr, search.Handler(site.Handler(mux)))) // outside intl.Site: it sees what crawlers ask for
```

On Cloudflare Workers, read `APP_DOMAIN` and `APP_PREVIEW` from the Worker's variables (workers-go's
`cloudflare.Getenv`) rather than the process environment.

`IsPage` has the shape of `intl.Site`'s, so one page table can answer both, as go-htmx4's does (its `pages`
package: each page a Go type, with the routes, the links, the sitemap and both hooks coming from it).

And in the layout's `<head>`, beside `intl.Head`, what the audit checks on every page: the page's own title and
description in its language, the site's name and icon, and on the home page its absolute URL (`homeURL` is the
app's: `intl.Canonical(ctx, "/")` there, `""` on every other page):

```gsx
<seo.Head
	title={ locales.PageTitle(title) |> text }
	description={ description |> text }
	siteName="Shop"
	icon="/static/favicon.svg"
	home={ homeURL(ctx, path) }
	lang={ intl.Current(ctx).ID() }
/>
<intl.Head page={ path }/>
```

seo needs Go 1.26 or newer, the same as gsx. Package `seo` is small enough for a Worker. The rest are tools: `seo/audit`, `seo/lighthouse`, `seo/searchconsole`
and `go tool seo`.

```sh
go get -tool github.com/joeblew999/go-htmx4/seo/cmd/seo
go tool seo init -base https://app.example.com   # seo.toml with your site, the mise tasks (.mise/tasks/seo.toml)
                                                  # and the agent skill (.claude/skills/seo)
```

`seo.toml` holds the settings the tasks read: the live site (no default: `init -base` writes it, and every run prints
which site it works on), its Search Console property, and the pages, device and minimum score for Lighthouse
(default: every page of the live sitemap, mobile, 90). An argument still overrides it for one run:
`mise run seo:lighthouse /de/board --desktop`. After upgrading, `go tool seo tasks` and `go tool seo skill` write the
tasks and the skill again.

## The tasks

Every task runs with no arguments, on `seo.toml`'s site, and says which site first. Exit codes: 0 ok, 1 problems or
no access.
`--json` prints one JSON document for a program.

| Task | Does | Needs |
|---|---|---|
| `seo:audit` | fetches robots.txt, the sitemap and every page in it as Googlebot, and fails on whatever would keep a page out or look wrong in a result (below). `mise run deploy` runs it after every production deploy | nothing |
| `seo:lighthouse` | Lighthouse on the live site (every page of the sitemap, `seo.toml`'s pages, or one: `/de/board`), on Google's servers through PageSpeed Insights: performance, accessibility, best practices and SEO out of 100, each failing check with the elements it failed on, and real Chrome users' Core Web Vitals once there are enough visits; fails under 90. `--desktop` for the desktop page | a PageSpeed API key (or the service account key) |
| `seo:badges` | Lighthouse on every page and the audit, then badges for a README (an SVG per Lighthouse category, the lowest page's score; one for the audit) into `seo.toml`'s `[badges] dir`, and every page's scores into the README between `<!-- seo:scores -->` and `<!-- /seo:scores -->`. Run it after a release, and commit | a PageSpeed API key (or the service account key) |
| `seo:todo` | Google's verdict for every sitemap URL: indexed, waiting, duplicate or a problem; fails on problems; `--open` opens each problem's page | the key |
| `seo:inspect` | one URL (the home page, or e.g. `/de/`): verdict, last crawl, Google's canonical and the app's | the key |
| `seo:status` | the sitemap's status, and every URL's coverage, last crawl and canonical | the key |
| `seo:sites` | the Search Console properties the service account can use; fails without Full access | the key |
| `seo:submit` | submits the sitemap, after a passing audit (asks first) | the key |
| `seo:setup` | once per person: the Google Cloud project, its Search Console and PageSpeed Insights APIs, the service account and its key (asks first) | `gcloud auth login` |

What the audit fails on, for every page in the sitemap: not 200 itself; noindex; a canonical that isn't the page;
no `lang`; no title or description, or one wider than Google shows (`seo.SnippetWidth`); invisible characters in
them; not exactly one `<h1>`; hreflang that isn't reciprocal, misses x-default or points off the site; no icon; no
`og:title`, `og:description`, or an `og:url` that isn't the page. For the site: robots.txt that keeps anything out;
a page that doesn't exist answering 200 (a soft 404); plain http not moving to https; no `WebSite` name on the home
page; an llms.txt, when there is one, without a title and summary or with a link to the site that doesn't answer 200. In a Go test, `audit.Audit(ctx, client, base)` returns the same problems: go-htmx4's smoke test audits the local
build in every check, so a page Google couldn't index fails before it ships.

Lighthouse's servers can't open WebSockets: a page with a live connection logs a console error there, which costs
it a few best-practice points that real visitors don't see.

## The keys

**Lighthouse** (`seo:lighthouse`, `seo:badges`) needs only a PageSpeed Insights API key, `PAGESPEED_API_KEY` in fnox:
a free key from the Google Cloud console (APIs & Services → Credentials → Create credentials → API key; restrict it to
the PageSpeed Insights API), then `fnox set PAGESPEED_API_KEY`. Without one, Lighthouse uses the service account key
below if there is one. Google's anonymous quota is too small to rely on.

**Search Console** (every other task that asks Google) uses one service account key, `GOOGLE_SEARCH_CONSOLE_KEY` in
fnox, for every site you own.
`seo:setup` makes it once: a Google Cloud project (`$SEO_PROJECT`, else the key's own, else `$APP_NAME-search`), its
Search Console and PageSpeed Insights APIs, the account and its key. Then, for each site: Search Console → the
property → Settings → Users and permissions → add that account with **Full** permission, and `mise run seo:sites`
confirms it.

## The skill

The `seo` skill tells Claude Code what Google needs from each page, what each audit problem, Lighthouse failure and
Google verdict means and where its fix is, and that `seo:submit` and `seo:setup` need a person's OK.

| Folder | What |
|---|---|
| `.` | package `seo`: `Site`, `Head` (`head.gsx`, with its generated `head.x.go` committed: apps import seo as a Go package), and `SnippetWidth` |
| `audit/` | the site as Googlebot reads it |
| `lighthouse/` | Lighthouse through PageSpeed Insights |
| `searchconsole/` | the Search Console API with a service account key |
| `config/` | `seo.toml` |
| `cmd/seo/` | `go tool seo`: audit, lighthouse, badges, todo, inspect, status, sites, submit, init, tasks, skill |
| `mise/` | the tasks, embedded in the command |
| `skills/seo/` | the agent skill, embedded in the command |
