---
name: seo
description: Use when a Go web app changes what Google reads (a new page, a title or description, a route, its noindex, robots.txt or sitemap rules), after a production deploy, or when asked how Google indexes the site, to fix what Google Search Console reports, or to get Lighthouse (performance, accessibility, best practices, SEO) right, with mise-seo (github.com/joeblew999/mise-seo; mise run seo and its seo tasks).
---

# How Google sees the site

`seo.Site`, middleware around the app (package `github.com/joeblew999/mise-seo/seo`), answers robots.txt, the sitemap
and llms.txt, marks what isn't a page noindex, and moves other hosts and plain http to the one https address; the app
says which paths are pages. The rest of what Google reads is the app's own: canonical URLs and hreflang (with
mise-intl, `gsxintl.Head` and `intl.Sitemap`), `<html lang>`, titles and descriptions (`gsxseo.Head` in a gsx app, the
page's own `<head>` otherwise), the icon, structured data, the pages' text. Fix a problem there. The tasks check it and
ask Google; `docs/seo/README.md` has the whole map.

Run every task from the repo's root. The site is `seo.toml`'s `site`; every task prints it first, so check it. Start
with `mise run seo`: what's done, and what to run next. Exit codes: 0 ok, 1 problems or no access; `--json` prints one
JSON document, so read that, never the text.

## 1. Check what the app serves

```sh
mise run seo:audit --json
mise run seo:audit --json --site http://localhost:8080   # a local build (mise run serve), a preview
```

It fetches robots.txt, the sitemap and every URL in it as Googlebot, and needs no Google account. `ok: false` comes
with `problems`, one line each, naming the URL. In a Go test, `audit.Audit(ctx, client, base)`
(`github.com/joeblew999/mise-seo/seo/audit`) returns the same problems: the template's app audits itself in its test.

| Problem | Where the fix is |
|---|---|
| `HTTP 3xx/4xx` | A sitemap URL must answer 200 itself: fix the route, or take the path out of the sitemap's pages |
| `X-Robots-Tag: noindex`, `meta robots` | The app's noindex rule (`seo.Site`'s `IsPage`) matched a page. With mise-intl, `intl.Site` also sends noindex while a language's catalog is incomplete: translate it |
| `canonical … is not the page's own URL` | The page's `<link rel="canonical">`: its origin (`seo.Origin`) and path, as the sitemap lists it; with mise-intl the page path given to `gsxintl.Head` |
| `<html> has no lang` | The layout's `<html lang>` |
| `no <title>`, `no meta description`, `… characters wide` | The page's title and description: write them, or shorten them (Google's widths: `seo.MaxTitleWidth`, `seo.MaxDescriptionWidth`, measured by `seo.SnippetWidth`) |
| `N <h1> elements` | Exactly one `<h1>` per page |
| `hreflang …` | Every language version lists every other and itself, plus x-default, as absolute URLs on the site's own scheme and host |
| `robots.txt …` | One `User-agent: *` group that allows everything, and `Sitemap:` with the absolute sitemap URL: `seo.Site` writes it |
| `… has invisible characters` | Bidi marks in a title: a message argument isolated where it needn't be |
| `no <link rel="icon">` | The layout's `<link rel="icon">` (`gsxseo.Head`'s icon) |
| `no og:title or og:description`, `og:url …` | The layout's Open Graph tags; og:url is the canonical URL |
| `HTTP 200 for a page that doesn't exist` | A catch-all route answers 200: unknown paths must be 404 |
| `not a redirect to https` | `seo.Site`'s `Host`: the production domain |
| `no WebSite structured data` | The home page's `<script type="application/ld+json">` with `"@type": "WebSite"` and `name` (`gsxseo.Head` with `home`) |
| `llms.txt …` | `seo.Site`'s `LLMs`, written with `seo.LLMsTxt`: a title, a summary, links on the site that answer 200 |

## 2. Lighthouse

```sh
mise run seo:lighthouse --json             # every page of the live sitemap (or seo.toml's pages), mobile
mise run seo:lighthouse --json /de/board   # one page; --desktop for the desktop one
```

It runs on Google's servers (PageSpeed Insights), so only on a live site: production, or a branch preview (whose SEO
score reads noindex, by design). Each category under seo.toml's min (90) fails; `failed` lists the checks with
`items`, the elements or files they failed on. Real users' Core Web Vitals come in `field` once Chrome has enough
visits. After a production release, `mise run seo:badges` writes the README's Lighthouse and audit badges and its
scores table (between `<!-- seo scores -->` and `<!-- /seo scores -->`) from a fresh run: commit them, never edit them
by hand.

| Fails | Where the fix is |
|---|---|
| Render-blocking requests | A script in `<head>` without `defer` or `type="module"`. Only what must run before first paint (the theme) stays blocking |
| Minify JavaScript, CSS | The build minifies what it ships |
| ARIA, names, contrast, touch targets | In the app's markup: fix it. In a vendored component (gsxui's, never edited): give it what it takes (an `aria-label`, an `id` and `aria-controls`, a class), else report it upstream with Lighthouse's items as evidence |
| Errors in the console | Read the items. A WebSocket that can't connect is Lighthouse's sandbox, not the site |

## 3. Ask Google

These need `GOOGLE_SEARCH_CONSOLE_KEY` (in the environment, else in fnox) and that account's Full access to the
property: `mise run seo:sites` says whether it has it.

```sh
mise run seo:todo --json           # every sitemap URL; exits 1 when one is a problem
mise run seo:inspect --json /de/   # one URL: verdict, coverage, last crawl, Google's canonical and the app's
mise run seo:status --json         # the sitemap's status, and every URL's coverage
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

## 4. What needs the person's OK

Lighthouse needs only `PAGESPEED_API_KEY` (a free PageSpeed Insights API key, in fnox); the service account key is
for Search Console. `mise run seo:submit` (sends the sitemap to Google) and `mise run seo:setup` (a Google Cloud
project, a service account, its key in fnox) write to Google: run them only when asked; both ask before they act
(agents pass `mise run --yes` only with the person's OK). Every other task only reads. The keys stay in fnox: never
print them.

## Adding a page Google should index

A title and a description within Google's widths; exactly one `<h1>`; its canonical URL; the path in the sitemap's
pages and in `IsPage`; not under a path the app marks noindex. Then the audit against a local build (`mise run
seo:audit --site http://localhost:8080`, or the app's own test with `audit.Audit`), and `mise run seo:audit`
after the deploy. Google finds the page from the sitemap; `mise run seo:submit` sends it again.

## Setting the repo up, and keeping it so

`mise run seo:init` writes mise-seo's files again after upgrading it (`--site https://…` sets seo.toml's live site
when it has none); `mise run seo:check` is the repo's rules; `mise run seo:verify` is what its GitHub workflow runs;
`mise run seo:template` writes the template's little app into a repo with no Go program yet. `mise run seo:show`
is `mise run seo`; `mise run seo:show-version` says which mise-seo this is, and `mise run seo:show-help` lists every
command.

## Every task

| Task | When | What it does |
|---|---|---|
| `mise run seo` | first, and whenever you're unsure | what's done for the repo, and what to run next |
| `mise run seo:init` | once, then after upgrading mise-seo | mise-seo's files (its tasks, the workflow, the skill, the docs, the hook), seo.toml when the repo has none or no site (--site, the live site), and go.mod requiring mise-seo's packages at this release |
| `mise run seo:check` | after a change to seo.toml or mise-seo's files; when it fails, run what it names | the repo's rules: mise-seo's files as this version writes them, its pin, seo.toml with a site, the packages in go.mod at the binary's release |
| `mise run seo:show` | to see where the repo is | the same as mise run seo |
| `mise run seo:audit` | after every production deploy, and on a local build before it | the site as Googlebot reads it, robots.txt, the sitemap and every page in it: whatever would keep a page out of the index or look wrong in a result (--site: another server, a local build or a preview) |
| `mise run seo:lighthouse` | after a production deploy, and to fix a score | Lighthouse on the live site on Google's servers: performance, accessibility, best practices and SEO out of 100 for every page or the one named, what fails and on which elements, real users' Core Web Vitals; fails under seo.toml's min (--desktop: the desktop page) |
| `mise run seo:badges` | after a production release, then commit them | Lighthouse on every page and the audit, then the README's badges and its scores table |
| `mise run seo:setup` | once per person, before the tasks that ask Google | the Google Cloud project, its Search Console and PageSpeed Insights APIs, and a service account key in fnox (asks first; needs gcloud auth login) |
| `mise run seo:sites` | after setup, and after adding the account to a property | the Search Console properties the service account can use; fails without Full access to seo.toml's |
| `mise run seo:submit` | once the site is live, and after its pages change | the sitemap submitted to Search Console, after a passing audit (asks first) |
| `mise run seo:status` | to see what Google has of the site | the sitemap's status in Search Console, and Google's index status of every URL in it |
| `mise run seo:todo` | days after submitting, and whenever Search Console reports a problem | Google's verdict for every sitemap URL: indexed, waiting, duplicate or a problem; fails on a problem (--open: each problem's Search Console page) |
| `mise run seo:inspect` | to look into one page | one URL, the home page or the one named: Google's verdict, last crawl and canonical, and its Search Console page (--open opens it) |
| `mise run seo:verify` | before every push | what the repo's GitHub workflow runs: its setup and test tasks, then check |
| `mise run seo:template` | only in a repo with no Go program yet | the template's little app, its pages answering search engines as the audit wants; it never writes over a file |
| `mise run seo:show-version` | to say which mise-seo this is | its version |
| `mise run seo:show-help` | to list every command | every command and its flags |
