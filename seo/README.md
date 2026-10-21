# seo

How Google sees a site that is live, for any app that serves a sitemap: one Go command, `go tool seo`, and
the mise tasks that run it. It produces nothing for Google. What Google reads (robots.txt, the sitemap, canonical URLs,
hreflang, titles and descriptions) is the app's own (in go-htmx4: `app/seo.go`, `views/seo.go` and intl). This checks
it, and asks Google what it made of it.

## The tasks

| Task | Does | Needs |
|---|---|---|
| `seo:audit` | fetches robots.txt and every sitemap URL as Googlebot: 200, indexable, self-canonical, `lang`, a title, a description, one h1, reciprocal hreflang with x-default. `mise run deploy` runs it after every production deploy | nothing |
| `seo:todo` | Google's verdict for every sitemap URL: indexed, waiting, duplicate or a problem; fails on problems | the key |
| `seo:inspect` | one URL (the home page, or e.g. `/de/`): verdict, last crawl, Google's canonical and yours | the key |
| `seo:status` | the sitemap's status, and every URL's coverage, last crawl and canonical | the key |
| `seo:sites` | the properties the service account can use; fails without Full access | the key |
| `seo:submit` | submits `https://$APP_DOMAIN/sitemap.xml`, after a passing audit (asks first) | the key |
| `seo:setup` | once per person: the service account and its key (asks first) | `gcloud auth login`, once |

`--json` prints one JSON document for a program and never opens a browser; `--open` (todo, inspect) opens Google's
page for each problem, where **Test live URL** and **Request indexing** are (indexing has no API). Exit codes: 0 ok,
1 problems or no access. The site is `https://$APP_DOMAIN`, its property the Domain property of its registrable
domain (`app.example.com` → `sc-domain:example.com`); `-base` and `-site` say otherwise.

## The key

The tasks that ask Google use a service account key, `GOOGLE_SEARCH_CONSOLE_KEY` in fnox: one service account for
every site you own. `seo:setup` makes it once (a Google Cloud project, `$APP_NAME-search` unless
`SEO_PROJECT` says otherwise, the Search Console API, the account and its key); with the key already there
it only names the account. Then, for each site: Search Console → the property → Settings → Users and permissions →
add that account with **Full** permission, and `mise run seo:sites` confirms it.

## In an app

```sh
go get -tool github.com/joeblew999/go-htmx4/seo/cmd/seo
go tool seo tasks      # writes .mise/tasks/seo.toml; again after an upgrade
go tool seo skill      # installs the agent skill: .claude/skills/seo
```

The `seo` skill tells Claude Code what Google needs from each page, what each audit problem and each of Google's
verdicts means and where its fix is, and that `seo:submit` and `seo:setup` need a person's OK.

In Go: `seo.Audit(ctx, client, base)` is the audit, for a test against a running server (go-htmx4's smoke
test runs it in every check, so a page Google couldn't index fails before it ships); `seo.SnippetWidth(s)`
measures a title or description in Google's units, for a test that keeps them under `MaxTitleWidth` and
`MaxDescriptionWidth` (go-htmx4: `TestSnippetWidths`).

| Folder | What |
|---|---|
| `.` | package `seo`: the audit, the Search Console API with a service account key, and snippet widths |
| `cmd/seo/` | `go tool seo`: audit, todo, inspect, status, sites, submit, tasks |
| `mise/` | the tasks, embedded in the command |
| `skills/seo/` | the agent skill, embedded in the command |
