# intl in this repo

Written by `mise-intl init`, for the version this repo's `go.mod` pins: don't edit it. After upgrading intl,
`mise-intl init` writes it again, and `mise-intl check` fails until it has. intl is
[mise-intl](https://github.com/joeblew999/mise-intl).

Every language a Go app ships, with [gsx](https://github.com/gsxhq/gsx) or without, through the runtime's own `Intl`. Messages are ICU
MessageFormat catalogs. Dates, numbers, plurals, lists, names, collation and segmentation are the real `Intl` APIs,
written in markup the gsx way. The workflows a multi-country app lives with are mise tasks: translating (by Claude
Code, by people, or by both), finding stale translations, adding a language, and checking all of it in CI.

```gsx
<h1>{ locales.HomeHeading }</h1>
<p title={ locales.BoardHint }>{ locales.BoardNotes(len(notes)) }</p>
<p>{ order.Placed |> date("dateStyle", "long") } · { order.Total |> number("style", "currency", "currency", "EUR") }</p>
<p>{ note.Created |> since } · { "JP" |> intl("DisplayNames", "of", "type", "region") }</p>
```

For a German request that renders `4. Juli 2026 · 1.234,50 €` and `vor 3 Minuten · Japan`, formatted by V8's
Intl: a Cloudflare Worker's own, or workerd's on an ordinary Go server. Each value arrives in an element that carries its call, and the browser redoes it with the same
code in the viewer's own time zone and clock:

```html
<time datetime="2026-07-04T15:30:00Z" data-intl="DateTimeFormat" data-opts='{"dateStyle":"long"}'
      data-args="[1783179000000]">4. Juli 2026</time>
```

Set its `data-opts` or `data-args` in the browser and the browser's Intl redoes it: a menu that changes a date's
style, a slider whose value is written in kilometres, need no request and no Go:

```gsx
<ui.Slider hx-on:input=js`document.querySelector("#km [data-intl]").dataset.args = JSON.stringify([this.value])`/>
<output id="km">{ 42 |> number("style", "unit", "unit", "kilometer", "unitDisplay", "long") }</output>
```

Go decides nothing about a language: no CLDR tables, no formatting of its own. Outside a Worker (an ordinary Go
server, `go test`, the tooling) the Intl is workerd's, the same V8, run as the repo's intl daemon (`intl/workerd`).

## What the repo needs: mise

Every tool comes from mise, intl's command among them: `mise-intl`, a binary on
[mise-intl's releases](https://github.com/joeblew999/mise-intl/releases), which mise installs at the release
`.mise/config.toml` pins (`init` writes the pin), as mise-cloudflare's repos get theirs.

workerd, which intl's Intl runs on outside a Worker, is an npm package: the repo's `package.json` pins it and aube
installs it (`mise run setup`: `aube install`), with node and aube from `mise.toml`, as the template's app has them. It
runs as a daemon, which [pitchfork](https://pitchfork.jdx.dev) runs (`mise.toml` pins it too): `pitchfork.toml`'s
`intl` runs `mise run intl:daemon`, workerd serving the files `mise-intl gen` writes beside the catalogs
(`intl-workerd/`), restarted when gen rewrites them. A repo with an app of its own adds them: `aube add -D
workerd@1.20261104.1`, `pitchfork` in its `mise.toml`, and the template's `pitchfork.toml`.

| | How | Checked by |
|---|---|---|
| The command | `mise-intl <command>`: the binary of the release `.mise/config.toml` pins (`"github:joeblew999/mise-intl"`) | mise (the release's checksums) |
| The packages an app imports | `github.com/joeblew999/mise-intl/intl` (and `gsxintl`, in a gsx app), the same release, which `init` and `intl:template` require in `go.mod` | `go.sum` and the compiler |
| Intl, off Cloudflare | workerd, from npm: the repo's `package.json` pins it, aube installs it (`INTL_WORKERD` names one of your own); pitchfork runs it, `pitchfork.toml`'s `intl` daemon | the lockfile |
| Go itself | `go.mod`'s `toolchain` line (or its `go` line), which `.mise/config.toml` has mise install | mise |

Each command is a task too, named for it: `mise run intl:gen` runs `mise-intl gen`. The table below has them in the order
a repo meets them, and `mise run intl` says which is next.

## Using it day to day

Every task runs with no arguments (a click in an editor's mise panel), taking what it needs from `intl.toml`;
arguments only narrow it, straight after the task name. `mise run <task> --help` explains any task.

```sh
mise run intl                       # what's done, and what to run next (mise-intl)
mise run intl:check              # every problem, for CI: catalogs, plurals, unused messages, intl's files
mise run intl:gen                # compile the catalogs after editing them (gsx generate needs this first)
mise run intl:status             # how far along each language is
mise run intl:add-locale fr      # a new language: intl.toml and its catalog, the id checked by Intl
mise run intl:todo               # what is left to translate, for a person (--json for an agent)
mise run intl:translate          # Claude Code translates everything missing or stale, in every language
mise run intl:export             # xliff/out/<locale>.xlf for human translators: what is left (--all: everything)
mise run intl:import             # their work back, from xliff/in
mise run intl:confirm            # the source changed but not its meaning (a typo fixed): keep the stale translations
mise run intl:prune              # remove keys the source no longer has from every catalog
mise run intl:add preferences       # in a gsx app: one of intl's screens (language-list, language-offer, preferences)
mise run intl:init               # after upgrading intl: intl's files written again

mise run intl:translate de ja         # only German and Japanese
mise run intl:confirm de:cart.items   # only one translation
```

Each runs the command of its name: `mise run intl:gen` runs `mise-intl gen`. The usual loop: write
or change messages in the source catalog, `mise run intl:translate`, `mise run intl:check`. A new language:
`mise run intl:add-locale fr`, then `mise run intl:translate`. (Adding its id to `intl.toml`'s `locales` by
hand works too: the next `mise run intl:gen` writes its catalog.)

## With gsx, or without

intl works in any Go app. Without gsx, everything is in `intl`: the messages are typed Go values (`locales.HomeTitle`,
`locales.HomeNotes(n)`), `intl.Text(ctx, x)` is a message's or a value's text in the request's language, and
`v.Element(ctx)` is a value's element (its tag, attributes and text) for your own templates. The template's app,
`mise-intl template`, is that: a page in plain Go, `net/http`, `html/template`, `intl.Site` and `intl.Text`.

With gsx, `gsxintl` is the one package that imports gsx (`mise-intl init` writes it into `gsx.toml` when the app
has gsx: gsxui, a `gsx.toml`, or `.gsx` files). intl uses gsx's own mechanisms, so `gsx generate`, the gsx language
server and `gsx info` see it like any project code:

- **Messages:** typed values and their renderers, as gsx's
  [Translation keys](https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/i18n-keys.md) pattern writes them.
  `{ locales.NavHome }` is a value; a message with arguments is a typed constructor, `{ locales.BoardNotes(n) }`.
  Both render in text and in attributes.
- **Intl calls:** [pipeline filters](https://github.com/gsxhq/gsx/blob/main/docs/guide/syntax/pipelines.md) (`date`,
  `number`, `since`, …). They return a `Value`, which a
  [package renderer](https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/package-renderers.md) written in gsx
  renders (`gsxintl.Render`).
- **gsxui:** its components' own text (`i18n.T`, the English as the id) is generated from the catalogs.
- **gsx's own checks:** `intl:check` runs `gsx generate`, `gsx fmt` and the `gsx` skill's `find-smells.sh` on the
  app's markup, and fails on anything they report. Make gsx's lints errors in the app's `gsx.toml` (`[lint]`) so
  `gsx generate` fails on them too. `gsx info` lists intl's filters and renderers, and each lint's level.

## Set up an app

intl needs Go 1.26 or newer, the same as gsx. [mise](https://mise.jdx.dev) brings every tool: mise-intl (the release
`.mise/config.toml` pins), Go (from `go.mod`, which `init` tells mise to use), gsx, node and aube, which install
[workerd](https://github.com/cloudflare/workerd) from `package.json`, and pitchfork, which runs it: its Intl is what the
tooling, the tests and an ordinary Go server run on.
Translating with Claude also needs [Claude Code](https://code.claude.com) (`claude`) installed.

At the repository's root (GitHub and Claude Code read `.github/` and `.claude/` only there), with mise installed once
on each machine ([mise's getting started](https://mise.jdx.dev/getting-started.html): `curl https://mise.run | sh`, or
`brew install mise`):

1. Add intl to the repo and set it up:

   ```sh
   mise exec github:joeblew999/mise-intl@latest -- mise-intl init --locales en,de,ar   # pins itself in .mise/config.toml
   mise run intl:template  # only in a repo with no app yet: the template's little app, its go.mod, and its tools
   ```

   `init` writes:
   - intl's tasks, `.mise/tasks/mise-intl.toml`, where mise finds them with no configuration
     ([Task discovery](https://mise.jdx.dev/tasks/task-discovery.html)), and `.mise/config.toml`, pinning mise-intl at
     the release that ran `init`, with the Go `go.mod` names, and the settings that make `latest` mise-intl's newest
     release the minute it's out (`minimum_release_age_excludes`, `use_versions_host = false`; a repo's own file gets
     them added);
   - in a Go module, intl's packages required in `go.mod` at the same release;
   - the `intl` skill into `.claude/skills`, and Claude Code's hook for cloud sessions;
   - these docs, `docs/intl/README.md`, and the GitHub workflow, `.github/workflows/intl.yml`;
   - `intl.toml`, the source catalog and a catalog per language;
   - in a gsx app, intl's filters and renderers into `gsx.toml`;
   - the generated files into `.gitignore`;
   - [mise-check](https://github.com/joeblew999/mise-check)'s own files, through its own init: its tasks
     (`mise run mise-check`), its workflow, skill and docs, and its pin beside mise-intl's.

   `@latest` is the newest release once it's a day old (mise's minimum release age); name one instead (`@0.2.0`:
   they're all on [its releases](https://github.com/joeblew999/mise-intl/releases)). `intl:template` never writes
   over a file. Commit what both wrote; `mise run serve` serves the template's app at http://localhost:8080, `/de/` in German.

   It picks up gsxui from `gsxui.json`, and then compiles the catalogs. Running it again leaves what is the repo's
   alone. To upgrade intl, run it again at the newer release (`mise exec github:joeblew999/mise-intl@latest --
   mise-intl init`), which moves the pin in `.mise/config.toml` and `go.mod`'s requirement forward to it, and writes
   intl's own files again; `mise-intl check` says when they're out of date.

2. Serve every request through `intl.Site`. It handles the locale URLs (`/about`, `/de/about`), the locale cookie,
   `Content-Language`, noindex while a catalog is incomplete, and the viewer's time zone and clock. `IsPage` names
   the pages, by their path without the locale prefix: a GET of one is a page view.

   ```go
   pages := []string{"/", "/about"}
   site := intl.Site{IsPage: func(path string) bool { return slices.Contains(pages, path) }}
   // …
   log.Fatal(http.ListenAndServe(addr, search.Handler(site.Handler(mux)))) // search is seo.Site; without seo, site.Handler(mux)
   ```

   On Cloudflare Workers, with [charter](https://github.com/joeblew999/charter), give it Cloudflare's guess of the
   viewer's zone too, `ConnectionTimeZone: func(r *http.Request) string { return transport.CFOf(r).Timezone }`
   (charter's `transport`), and register plain paths, checking the method in the handler: TinyGo's `net/http` has the
   ServeMux from before Go 1.22, which takes no `GET /x` patterns.

   The server renders in the reader's zone, so text doesn't change after it arrives: the zone they chose; else the one
   their browser reported on its first page (`intl-browser.mjs` sets the `browser_tz` cookie); else Cloudflare's guess
   from their IP; else UTC. `intl.Suggested(ctx)` is the page in the language the reader wants, when the site has it
   and the page is in another: the one they chose, else the one their browser asks for (`Accept-Language`, matched by
   Intl's likely subtags: `zh-TW` is zh-Hant). intl's `language-offer` screen offers it as a link, written in that
   language, `{ locales.IntlAlsoIn(s.Name).In(s.Lang) }`, on every page until they choose: following a link from a page in one
   language to one in another (the language list, the offer) is choosing, and so is `intl.Decline(ctx)`, the "No
   thanks" beside the offer. Never a redirect: Googlebot sends no `Accept-Language`, and a redirect would hide the
   other language versions from it. Every response is `Cache-Control: private`: it is written for one reader.

3. In the layout, `<html>` gets the page's `lang` and `dir` and the viewer's `data-tz` and `data-hc`, and
   `gsxintl.Head` writes the canonical URL, the hreflang alternates, `og:url` and `og:locale` of `path`, the page
   without its locale prefix:

   ```gsx
   <html { gsxintl.HTML(ctx)... }>
   	<head>
   		// …
   		<gsxintl.Head page={ path }/>
   		<script type="module" src="/static/intl-browser.mjs"></script>
   ```

   No `//` comment after an element's `>` on the same line: in gsx markup that is text, and the page shows it.

   The browser redoes each `[data-intl]` element in the viewer's zone and clock, keeps `since` fresh, and redoes what
   htmx swaps in. A language list is `intl.Links(ctx)` (each locale's name in its own language, and the page's URL
   in it), and intl ships one ready to use ([Its screens](#its-screens)); `intl.Sitemap(origin, pages)` is a
   sitemap.xml with every alternate.

4. Give intl its Intl.

   On an ordinary Go server, set workerd's before serving. It is the repo's intl daemon: `Start` starts it when it
   isn't running (`pitchfork start intl`, through the repo's own mise: gen, then workerd on gen's files, from the
   `node_modules` `aube install` filled), and it stays for every process after, restarting when gen rewrites the
   catalogs; `pitchfork stop intl` stops it. A server deployed without the repo runs a workerd of its own on the same
   files (`workerd serve intl-workerd/intl-workerd.capnp`) and names it with `INTL_WORKERD_URL`. Each formatted value
   is a request to it on 127.0.0.1, about 0.1 ms:

   ```go
   if err := workerd.Start(); err != nil { // github.com/joeblew999/mise-intl/intl/workerd
   	log.Fatal(err)
   }
   ```

   On Cloudflare Workers, the Worker's own Intl: its entry module puts intl's formatter where Go calls it, and the
   build copies the JavaScript (embedded in the module):

   ```js
   import { goIntl } from "./build/intl-worker.mjs";
   globalThis.goIntl = goIntl;
   ```

   ```sh
   mise-intl assets -worker build -browser dist/static
   ```

   The template's app runs on a Worker as it is: `app/worker.go` serves the same handler with
   [workers-go](https://github.com/syumai/workers-go) (`GOOS=js GOARCH=wasm go build`, its `workers-assets-gen -mode=go`
   for the glue), and an entry module that sets `globalThis.goIntl` and exports workers-go's `worker.mjs`.

   Either way, the browser needs `intl-browser.mjs` and the `intl.mjs` it imports: `mise-intl assets -browser
   static` copies them. Copy them in your build and keep them out of git, like other generated files: they come
   with the intl version in `go.mod`.

5. Test with workerd's Intl, the same daemon, in every package that renders:

   ```go
   func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }
   ```

6. Day to day: `mise run intl:gen` before `gsx generate` (make your generate task depend on it), and
   `mise run intl:check` in CI (the workflow `init` wrote runs `mise run intl:verify`). On a pull request (GitHub Actions' `pull_request` runs), `mise-intl check` only
   reports missing and stale translations, without failing, so a contributor can leave them and you translate them
   after merging; pushes and releases stay strict. `INTL_MISSING=fail` or `warn` (or `--missing`) says otherwise.

`intl.toml` holds the languages, where the catalogs are, notes for translators and, if you want one, a README badge:

```toml
locales = ["en", "de", "ar"]   # the source first
catalogs = "locales"
gsxui = "ui"
ui = "views/intl"              # where mise-intl add puts intl's screens (the default)
badge = "docs/badges/intl.svg" # "languages | 3, all translated": gen writes it, check says when it's stale

[notes]
"*" = "A shop. Keep brand names in English."
de = "Address the reader as Sie."
```

## Its screens

intl ships the screens a site with languages needs, the way gsxui ships components: gsx files copied into the app,
made of gsxui components, with their text already translated into intl's 13 languages.

| Screen | Component | What it is |
|---|---|---|
| `language-list` | `<intlui.LanguageList/>` | the page in every language, each named in its own: plain links, crawlable, for a footer |
| `language-offer` | `<intlui.LanguageOffer/>` | the page in the reader's language, offered in that language until they choose (`intl.Suggested`) |
| `preferences` | `<intlui.PreferencesDialog/>` | a header button and the time zone and clock dialog it opens |

```sh
mise-intl add language-list language-offer preferences   # or mise run intl:add …
```

`add` runs `gsxui add` for the components a screen needs that the app doesn't have (Button, Alert, Dialog, Field,
NativeSelect), writes each screen to `views/intl/<screen>.gsx` (package `intlui`; `intl.toml`'s `ui` moves it) with
its imports pointing at the app's catalogs and gsxui components, and adds their text to the catalogs: `[intl]`, and
`[gsxui]` for what those components write themselves, in every language the app has that intl ships. The source
catalog gets the comments translators read; a language intl doesn't ship goes through `mise run intl:translate` like
the app's own text. Then it compiles the catalogs, runs `gsx generate`, and says where each screen goes.

The files are the app's own: change them as you like. `add` records what it wrote in `intl.sum`, and never replaces a
file the app changed: `--diff` shows how the app's copy differs from intl's, `--overwrite` replaces it. A message the
app already has keeps its text.

`add` is a registry any module can use for components of its own: `gsxintl/registry` copies gsx templates made of
gsxui components, with their translated text and the browser files they load, into an app. intl's screens are one
registry (`gsxintl/screens`); [gsxkit](https://github.com/joeblew999/go-htmx4/blob/main/gsxkit/README.md#components)'s
components (a map, live video) are another, added with `go tool gsxkit add`.

The dialog loads its form from two routes, which the app serves:

```go
mux.HandleFunc("GET /fragments/preferences", func(w http.ResponseWriter, r *http.Request) {
	intlui.PreferencesForm(intl.ReturnTo(r), site.ConnectionZone(r)).Render(r.Context(), w)
})
mux.HandleFunc("POST /preferences", site.Preferences)
```

## In markup

| Write | Renders |
|---|---|
| `{ locales.Key }`, `{ locales.Key(args…) }` | a message, in text or an attribute; text arguments are bidi-isolated |
| `{ locales.Placed(intl.Date(t, intl.Options("dateStyle", "long"))) }` | a formatted value inside a message. A text argument takes a `string`, another message or a Value; a `{n, number}` or `plural` one takes any Go number. gsx allows no pipeline inside a call's arguments, so the Value is written in Go |
| `{ t \|> date }`, `{ t \|> date("dateStyle", "full", "calendar", "hebrew") }` | `Intl.DateTimeFormat`, in a `<time>` |
| `{ start \|> dateRange(end, opts) }` | its `formatRange` |
| `{ t \|> since }` | `Intl.RelativeTimeFormat` in the largest readable unit, kept fresh by the browser |
| `{ n \|> number }`, `{ n \|> number("style", "percent") }` | `Intl.NumberFormat`, in a `<data>`; an int, a float or an exact decimal string |
| `{ d \|> duration }` | `Intl.DurationFormat` |
| `{ items \|> list(opts) }` | `Intl.ListFormat` |
| `{ id \|> zone }` | a time zone's name: `Europe/Berlin · Mitteleuropäische Zeit (GMT+2)` |
| `{ v \|> intl(api, method, opts…) }` | any call: `new Intl[api](lang, opts)[method](v)`; api `"Intl"` for `Intl.supportedValuesOf` and the like |
| `{ intl.Args{a, b} \|> intl(api, method, opts…) }` | a call with several arguments, or none (`intl.Args{}`): `Collator.compare`, `formatRange`, `Locale.getWeekInfo` |
| `title={ v \|> plain }` | a Value's text alone: attributes take no markup |
| `title={ locales.Key \|> text }` | a message's text, for a component input typed `string` (gsxui's `Toast title`) |
| `<option { gsxintl.Attrs(v)... }>{ v \|> plain }</option>` | an element that can't hold the Value's own, still redone by the browser |
| `<option { gsxintl.Attrs(intl.Clock("h12"))... }>{ intl.Clock("h12") \|> plain }</option>` | a clock choice labelled by a time in that clock (`""`: the language's own) |
| `href={ "/about" \|> localize }`, `hx-post={ f`/cart/add?id=@{id}` \|> localize }` | an in-app path in the request's locale: `/de/about` |
| `href={ pages.About{} \|> url }` | a page's address in the request's locale, for an app whose pages are Go types with a `Path() string` (a page table, structpages' shape, TinyGo-safe): `/de/about` |

Options are Intl's own option names and values, in pairs (`"maximumFractionDigits", "0"`: a number or true/false
is one); `v.JSOptions()` shows them as JavaScript writes them. Results that aren't strings (`resolvedOptions`, `select`, `compare`, `segment`) render as
JSON.

In Go, the same values are `intl.Date(t, opts)`, `intl.Of(api, method, opts, args…)` and the rest, with options
from `intl.Options(pairs…)`, and `intl.Text(ctx, x)` is the text (an HTTP error, a log line). For what markup can't
say:

| Go | Does |
|---|---|
| `intl.Order(ctx, names, opts)` | the order `Intl.Collator` sorts names in: sort rows by a name (Ägypten before Österreich in German, after it in Swedish) |
| `intl.Segments(ctx, text, "grapheme" \| "word" \| "sentence")` | text split by `Intl.Segmenter`: count what a reader sees as characters, take initials, find words in Japanese |
| `intl.CalendarOf(ctx)`, `intl.WeekStart(ctx)` | a month calendar's words and its first day: `<ui.Calendar locale={ ui.CalendarLocale(intl.CalendarOf(ctx)) } weekStartsOn={ intl.WeekStart(ctx) }/>` (gsxui e23aef1 or later: months inside a date, Russian "4 июля") |
| `intl.Current(ctx).ParseNumber(text)` | a number typed the language's way, as a plain decimal (`window.intl.parseNumber` in the browser) |

## Translating

| When | Run |
|---|---|
| You added or changed messages in the source catalog | `mise run intl:gen`, then `mise run intl:translate` (Claude Code) or translate what `mise run intl:todo` lists |
| A translation's source changed | it is **stale**: `intl:todo` shows the old translation beside the new source. Update it, or keep it: `mise run intl:confirm de:order.items` |
| You add a language | `mise run intl:add-locale fr`, then `mise run intl:translate`. A regional one can take what it doesn't translate from another: `[inherit] fr-CA = "fr"` |
| You delete or rename a source message | `intl:check` names the keys left in the translations; `mise run intl:prune` removes them |
| `gsxui add pagination` | `mise run intl:gen` names each new text with the line to add under `[gsxui]`; add them, then translate |
| A person translates | `mise run intl:export` writes XLIFF 2.0 per locale into `xliff/out` for their tool; put what comes back in `xliff/in`, and `mise run intl:import` reads it, checking each target (one translated from older English is refused) |
| CI | `mise run intl:check` |
| Leftover English in the UI | `intl.Pseudo = true` renders the `en-XA` catalog: accented, bracketed (`⟦Ĥéļļó⟧`), 40% longer |

`intl:translate` runs Claude Code headless with the `intl` skill, one worker per language at once
(`intl.toml`'s `[translate] workers`, default 6, on `[translate] model`). A worker reads `intl:todo --json` for its
language (each message's source, context and arguments, the plural categories from Intl, the notes from
`intl.toml`). It may edit only that language's catalog, and it checks its work with `mise-intl check <locale>`
until that passes. Then `gen` and the full `intl:check` run once. The skill is in `.claude/skills/intl` and works the same in an
interactive session ("translate the new messages").

`intl:check` fails on any of these:
- an ICU syntax error, or an argument or argument kind the source doesn't have;
- a plural or selectordinal without a case for a category the language uses (Russian: one, few, many, other);
- a missing or stale translation, or a key the source no longer has;
- an `[inherit]` between locales Intl writes in different scripts;
- a message no Go or gsx file uses;
- gsxui text without a message;
- an `intl.sum` that is out of date.

### The catalogs

- One `<locale>.toml` per shipped locale, with the source's tables and keys, in ICU MessageFormat 1 (`plural`,
  `selectordinal`, `select`, `{n, number}`). A key a translation lacks comes from the locale `[inherit]` names
  (`en-IN = "en"`). Otherwise it is missing, and the source text shows.
- `intl.sum` records, for every translation, the source text it was translated from (as hashes). That's how a
  changed source shows up as stale. `intl:gen` writes it; editing a translation updates it.
- A locale is complete when nothing is missing: `intl.Complete(id)`, so `intl.Site` serves an incomplete one
  `noindex`. Stale translations don't make it incomplete (they still say the right thing in the language); `intl:check`
  fails on them until they are updated or confirmed.

## The tasks

Numbered in the order a repo meets them; each runs the command of its name.

| Task (with mise) | Command (without) | What it does |
|---|---|---|
| `mise run intl` | `mise-intl` | What's done, and what to run next |
| `intl:init` | `mise-intl init` | intl's files, as this version writes them |
| `intl:check` | `mise-intl check` | Every problem, for CI (`--missing warn`; or only the languages named) |
| `intl:show` | `mise-intl show` | What's done, and what to run next (`--json`) |
| `intl:gen` | `mise-intl gen` | The catalogs compiled into the app's messages package |
| `intl:status` | `mise-intl status` | Per language: translated, inherited, stale, missing |
| `intl:add-locale` | `mise-intl add-locale` | A language added |
| `intl:todo` | `mise-intl todo` | What's left to translate (`--json`) |
| `intl:translate` | `mise-intl translate` | Claude Code translates it |
| `intl:export` | `mise-intl export` | XLIFF 2.0 for translators (`--all`, `--out`) |
| `intl:import` | `mise-intl import` | Their XLIFF read back |
| `intl:confirm` | `mise-intl confirm` | Stale translations that are still right, kept |
| `intl:prune` | `mise-intl prune` | Keys the source no longer has, removed |
| `intl:verify` | `mise-intl verify` | What GitHub runs: the repo's setup and test tasks, then check |
| `intl:template` | `mise-intl template` | The template's little app, in a repo with no app yet |
| `intl:add` | `mise-intl add` | In a gsx app: intl's screens (`--diff`, `--overwrite`) |
| `intl:daemon` | `mise-intl daemon` | Intl outside a Worker: workerd on gen's files, in the foreground, which `pitchfork.toml`'s `intl` daemon runs |
| `intl:assets` | `mise-intl assets` | The JavaScript a Worker and a page import (`--worker`, `--browser`) |
| `intl:show-help` | `mise-intl show-help` | Every command |

GitHub runs `.github/workflows/intl.yml` on every push: it checks out, installs mise, then runs `mise run intl:verify`.
Run that task here and you see what GitHub sees.

## The packages

| Package (`github.com/joeblew999/mise-intl/…`) | For | Imports gsx? |
|---|---|---|
| `intl` | every app: messages, languages, the middleware (`intl.Site`), `intl.Text` and the formatting calls | no |
| `intl/workerd` | apps not on Cloudflare: Intl from workerd | no |
| `intl/intltest` | apps' tests | no |
| `intl/config` | tools that read an app's `intl.toml` | no |
| `gsxintl` | gsx apps: `Head`, `HTML`, `Attrs` and the renderer `gsx.toml` names | **yes, the only one** |
| `gsxintl/filters` | gsx apps' `gsx.toml`: `date`, `number`, `since`, … | no |
| `gsxintl/registry`, `gsxintl/screens` | `mise-intl add`: intl's screens, and any module's gsx components | no |
| `cmd/mise-intl` | `mise-intl` | no |
