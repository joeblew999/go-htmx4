# mise-seo in this repo

This repo answers search engines with [mise-seo](https://github.com/joeblew999/mise-seo): the middleware that answers
them, an audit of what they read, Lighthouse on Google's servers, the README's badges, and Google Search Console.
`.mise/config.toml` pins mise-seo, from its GitHub releases: `mise install` gets it. Every command is a task of
`.mise/tasks/mise-seo.toml`, numbered in the order a repo meets them: `mise run seo:<nn>-<command>` runs `mise-seo
<command>`. Start with `mise run seo`: what's done, and what to run next.

## A new repo, or one with an app of its own

At the repository's root (GitHub and Claude Code read `.github/` and `.claude/` only there), with mise installed once
on each machine ([mise's getting started](https://mise.jdx.dev/getting-started.html)):

```sh
mise exec "github:joeblew999/mise-seo[minimum_release_age=0s]@latest" -- mise-seo init --site https://app.example.com
mise run seo:template   # only in a repo with no Go program yet: the template's little app
mise run seo            # what's done, and what to run next
```

`init` runs mise-seo's newest release (or the one you name instead of `latest`: they're all on
[its releases](https://github.com/joeblew999/mise-seo/releases)); mise holds back a release younger than a day as
`latest`, so `init` asks for none of that, for itself alone (`[minimum_release_age=0s]`). It writes the repo's
`.mise/` folder (its tasks, and mise-seo pinned at exactly that release), `seo.toml` with the live site (`--site`), its
GitHub workflow, its Claude Code hook and skill, this page and its `.gitignore`; in a repo with a `go.mod`, it requires
mise-seo's packages at the same release. Of a file the repo has already (`.mise/config.toml`, `.gitignore`,
`.claude/settings.json`, `seo.toml`) it writes only its parts: mise-seo's pin, its name in
`minimum_release_age_excludes` and `use_versions_host = false`, so `latest` is its newest release the minute it's out;
the `.gitignore` lines; the hook's entry; a site when `seo.toml` sets none. Run again after upgrading, it writes its
files as that release does, and moves the pins forward.

`seo:template` never writes over a file: two pages in plain Go behind `seo.Site` (`app/`), their test auditing them as
Googlebot reads them, the `mise.toml` of their tasks (`setup`, `serve`, `test`), a README, and the app's site on this
machine in `seo.toml` when it sets none. Commit what `init` and `template` wrote.

## What Google reads, and who makes it

Google ranks what the site serves. Every piece is made by the app, live on each request; nothing is generated ahead:

| Google reads | What it's for | Made by |
|---|---|---|
| `/robots.txt` | may it crawl, and where the sitemap is | `seo.Site` |
| `/sitemap.xml` | every page, with its other-language versions | `seo.Site`, writing `seo.Sitemap` of the app's pages (in several languages, mise-intl's `intl.Sitemap`) |
| `/llms.txt` | for agents and language models: what the site is and where to read more ([llmstxt.org](https://llmstxt.org)) | `seo.Site`'s `LLMs`, written with `seo.LLMsTxt` |
| `X-Robots-Tag: noindex` | keeps fragments, posts, previews and endless query pages out | `seo.Site` (the app's `IsPage`, and `Preview`) |
| one address per page | other hosts and plain http move to `https://` the domain | `seo.Site` (`Host`) |
| `<html lang>` | the page's language | the layout (mise-intl's `gsxintl.HTML`) |
| `<title>`, `<meta name="description">`, `og:title`, `og:description` | the blue link and text in a result, and how a shared link looks | `gsxseo.Head` in a gsx app, the page's own `<head>` otherwise |
| `<link rel="canonical">`, `og:url`, hreflang | which URL is the page, and its versions in other languages | the page's origin (`seo.Origin`) and path; with mise-intl, `gsxintl.Head` |
| `<link rel="icon">` | the icon beside the site's results | `gsxseo.Head` |
| `WebSite` structured data | the site's name in results | `gsxseo.Head`, on the home page |
| one `<h1>`, real text | what the page is about; a thin page gets crawled and not indexed | each page |

`mise run seo:06-audit` checks all of it, before a deploy and after, and the tasks after it ask Google what it made
of it.

## In an app

`pages` are the app's paths:

```go
search := seo.Site{
	Host:    os.Getenv("APP_DOMAIN"), // the one address; "" in previews and locally
	Sitemap: func(origin string) string { return seo.Sitemap(origin, pages) },
	IsPage:  func(path string) bool { return slices.Contains(pages, path) }, // the request's path and query
	Preview: os.Getenv("APP_PREVIEW") != "",                                // everything noindex
}
log.Fatal(http.ListenAndServe(addr, search.Handler(mux))) // outside everything else: it sees what crawlers ask for
```

With mise-intl, `Sitemap` is `intl.Sitemap(origin, pages)`, `IsPage` has the shape of `intl.Site`'s so one page table
answers both, and `seo.Site` goes outside `intl.Site`. On Cloudflare Workers, read `APP_DOMAIN` and `APP_PREVIEW` from
the Worker's variables rather than the process environment.

In a gsx app's layout, in `<head>`: the page's own title and description in its language, the site's name and icon,
and on the home page its absolute URL (`""` on every other page):

```gsx
<gsxseo.Head
	title={ title }
	description={ description }
	siteName="Shop"
	icon="/static/favicon.svg"
	home={ homeURL }
	lang="en"
/>
```

`gsxseo` (`github.com/joeblew999/mise-seo/gsxseo`) is mise-seo's only package that imports gsx: an app without gsx
imports `seo` alone, writes the same tags in its own `<head>` (as the template's app does), and never gets gsx in its
`go.mod`. And in the app's tests, the audit on a local build, so a page Google couldn't index fails before it ships:

```go
srv := httptest.NewServer(handler())
defer srv.Close()
_, problems, err := audit.Audit(context.Background(), srv.Client(), srv.URL) // github.com/joeblew999/mise-seo/seo/audit
```

| Package (`github.com/joeblew999/mise-seo/…`) | For | Imports gsx? |
|---|---|---|
| `seo` | every app: `Site`, `Sitemap`, `LLMsTxt`, `Origin`, `SnippetWidth` | no |
| `seo/audit` | an app's tests: the site as Googlebot reads it | no |
| `seo/config` | tools that read an app's `seo.toml` | no |
| `gsxseo` | gsx apps: `Head` | **yes, the only one** |

## seo.toml

The settings the tasks read, so none needs an argument: the live site (no default: `mise run seo:03-init --site`
writes it, and every task prints which site it works on), its Search Console property (default: the site's Domain
property, `app.example.com` → `sc-domain:example.com`), the pages, device and minimum score for Lighthouse (default:
every page of the live sitemap, mobile, 90), and where `seo:08-badges` writes (default `docs/badges`, and `README.md`'s
lines between `<!-- seo scores -->` and `<!-- /seo scores -->`). `--site` on `seo:06-audit` audits another server for
one run; `seo:07-lighthouse` and `seo:14-inspect` take one page.

## The tasks

Every task runs with no arguments, on `seo.toml`'s site, and says which site first. Exit codes: 0 ok, 1 problems or no
access. `--json` prints one JSON document for a program (`seo`, `seo:04-check`, `seo:05-show`, `seo:06-audit`,
`seo:07-lighthouse`, `seo:10-sites`, `seo:12-status`, `seo:13-todo`, `seo:14-inspect`); `--open` opens Google's page of
each problem (`seo:13-todo`) or of the page (`seo:14-inspect`); `--desktop` scores the desktop page (`seo:07-lighthouse`);
`--site` is init's live site, or another server for the audit. Steps 00 to 02 come before them: the repo's git, `mise
install`, its own `setup`.

| Task | Run it | What it does |
|---|---|---|
| `mise run seo` | first, and whenever you're unsure | what's done for the repo, and what to run next |
| `mise run seo:03-init` | once, then after upgrading mise-seo | mise-seo's files (its tasks, the workflow, the skill, the docs, the hook), seo.toml when the repo has none or no site (--site, the live site), and go.mod requiring mise-seo's packages at this release |
| `mise run seo:04-check` | after a change to seo.toml or mise-seo's files; when it fails, run what it names | the repo's rules: mise-seo's files as this version writes them, its pin, seo.toml with a site, the packages in go.mod at the binary's release |
| `mise run seo:05-show` | to see where the repo is | the same as mise run seo |
| `mise run seo:06-audit` | after every production deploy, and on a local build before it | the site as Googlebot reads it, robots.txt, the sitemap and every page in it: whatever would keep a page out of the index or look wrong in a result (--site: another server, a local build or a preview) |
| `mise run seo:07-lighthouse` | after a production deploy, and to fix a score | Lighthouse on the live site on Google's servers: performance, accessibility, best practices and SEO out of 100 for every page or the one named, what fails and on which elements, real users' Core Web Vitals; fails under seo.toml's min (--desktop: the desktop page) |
| `mise run seo:08-badges` | after a production release, then commit them | Lighthouse on every page and the audit, then the README's badges and its scores table |
| `mise run seo:09-setup` | once per person, before the tasks that ask Google | the Google Cloud project, its Search Console and PageSpeed Insights APIs, and a service account key in fnox (asks first; needs gcloud auth login) |
| `mise run seo:10-sites` | after setup, and after adding the account to a property | the Search Console properties the service account can use; fails without Full access to seo.toml's |
| `mise run seo:11-submit` | once the site is live, and after its pages change | the sitemap submitted to Search Console, after a passing audit (asks first) |
| `mise run seo:12-status` | to see what Google has of the site | the sitemap's status in Search Console, and Google's index status of every URL in it |
| `mise run seo:13-todo` | days after submitting, and whenever Search Console reports a problem | Google's verdict for every sitemap URL: indexed, waiting, duplicate or a problem; fails on a problem (--open: each problem's Search Console page) |
| `mise run seo:14-inspect` | to look into one page | one URL, the home page or the one named: Google's verdict, last crawl and canonical, and its Search Console page (--open opens it) |
| `mise run seo:verify` | before every push | what the repo's GitHub workflow runs: its setup and test tasks, then check |
| `mise run seo:template` | only in a repo with no Go program yet | the template's little app, its pages answering search engines as the audit wants; it never writes over a file |
| `mise run seo:show-version` | to say which mise-seo this is | its version |
| `mise run seo:show-help` | to list every command | every command and its flags |

What the audit fails on, for every page in the sitemap: not 200 itself; noindex; a canonical that isn't the page; no
`lang`; no title or description, or one wider than Google shows (`seo.SnippetWidth`); invisible characters in them;
not exactly one `<h1>`; hreflang that isn't reciprocal, misses x-default or points off the site; no icon; no
`og:title`, `og:description`, or an `og:url` that isn't the page. For the site: robots.txt that keeps anything out; a
page that doesn't exist answering 200 (a soft 404); plain http not moving to https; no `WebSite` name on the home
page; an llms.txt, when there is one, without a title and summary or with a link to the site that doesn't answer 200.

Lighthouse's servers can't open WebSockets: a page with a live connection logs a console error there, which costs it
a few best-practice points that real visitors don't see.

## The keys

The tasks that ask Google read each key from the environment, else from [fnox](https://github.com/jdx/fnox), which
each of them brings (its `tools`), so a repo needs only mise. A key set empty in the environment is none, and fnox
isn't asked.

**Lighthouse** (`seo:07-lighthouse`, `seo:08-badges`) needs only a PageSpeed Insights API key, `PAGESPEED_API_KEY`: a
free key from the Google Cloud console (APIs & Services → Credentials → Create credentials → API key; restrict it to
the PageSpeed Insights API), then `fnox set PAGESPEED_API_KEY`. Without one, Lighthouse uses the service account key
below. Google's anonymous quota is too small to rely on.

**Search Console** (`seo:10-sites` to `seo:14-inspect`) uses one service account key, `GOOGLE_SEARCH_CONSOLE_KEY`, for
every site you own. `mise run seo:09-setup` makes it once per person, with your gcloud login (`mise exec -- gcloud auth
login`; the task brings gcloud): a Google Cloud project (`SEO_PROJECT`, else the key's own, else the site's first label
and `-search`), its Search Console and PageSpeed Insights APIs, the account, and its key in fnox. Then, for each site:
Search Console → the property → Settings → Users and permissions → add that account with **Full** permission, and
`mise run seo:10-sites` confirms it. `seo:09-setup` and `seo:11-submit` write to Google and ask first.

## The skill and the workflow

The `seo` skill (`.claude/skills/seo/SKILL.md`) tells Claude Code what Google needs from each page, what each audit
problem, Lighthouse failure and Google verdict means and where its fix is, and that `seo:11-submit` and `seo:09-setup`
need a person's OK. `.github/workflows/seo.yml` runs `mise run seo:verify` on every push and pull request: the repo's
`setup` and `test` tasks (an app's tests auditing its local build), then `seo:04-check`. It asks Google nothing. In a
Claude Code cloud session, `.claude/hooks/mise-seo-session-start.sh` installs mise and every pinned tool first.
