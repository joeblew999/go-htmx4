# intl

Every language a [gsx](https://github.com/gsxhq/gsx) app ships, through the runtime's own `Intl`. Messages are ICU
MessageFormat catalogs. Dates, numbers, plurals, lists, names, collation and segmentation are the real `Intl` APIs,
written in markup the gsx way. The workflows a multi-country app lives with are mise tasks: translating (by Claude
Code, by people, or by both), finding stale translations, adding a language, and checking all of it in CI.

```gsx
<h1>{ locales.HomeHeading }</h1>
<p title={ locales.BoardHint }>{ locales.BoardNotes(len(notes)) }</p>
<p>{ order.Placed |> date("dateStyle", "long") } · { order.Total |> number("style", "currency", "currency", "EUR") }</p>
<p>{ note.Created |> since } · { "JP" |> intl("DisplayNames", "of", "type", "region") }</p>
```

For a German request that renders `4. Juli 2026 · 1.234,50 €` and `vor 3 Minuten · Japan`, formatted by the
Worker's V8 Intl. Each value arrives in an element that carries its call, and the browser redoes it with the same
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

Go decides nothing about a language: no CLDR tables, no formatting of its own. Outside a Worker, `go test` and the
tooling use workerd's Intl.

## Using it day to day

Every task runs with no arguments (a click in an editor's mise panel), taking what it needs from `intl.toml`;
arguments only narrow it, straight after the task name. `mise run <task> --help` explains any task.

```sh
mise run intl                # compile the catalogs after editing them (gsx generate needs this first)
mise run intl:status         # how far along each language is
mise run intl:translate      # Claude Code translates everything missing or stale, in every language
mise run intl:check          # everything, for CI: catalogs, plurals, unused messages, gsx's own checks
mise run intl:todo           # what is left to translate, for a person (--json for an agent)
mise run intl:confirm        # the source changed but not its meaning (a typo fixed): keep the stale translations
mise run intl:export         # xliff/out/<locale>.xlf for human translators: what is left (--all: everything)
mise run intl:import         # their work back, from xliff/in
mise run intl:prune          # remove keys the source no longer has from every catalog
mise run intl:tasks          # after upgrading intl: these tasks again
mise run intl:skill          # after upgrading intl: Claude's translation skill again

mise run intl:translate de ja         # only German and Japanese
mise run intl:confirm de:cart.items   # only one translation
```

The usual loop: write or change messages in the source catalog, `mise run intl:translate`, `mise run intl:check`.
A new language is a line: add its id to `locales` in `intl.toml`, then `mise run intl:translate`.

## It is gsx

intl uses gsx's own mechanisms, so `gsx generate`, the gsx language server and `gsx info` see it like any project
code:

- **Messages:** typed values and their renderers, as gsx's
  [Translation keys](https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/i18n-keys.md) pattern writes them.
  `{ locales.NavHome }` is a value; a message with arguments is a typed constructor, `{ locales.BoardNotes(n) }`.
  Both render in text and in attributes.
- **Intl calls:** [pipeline filters](https://github.com/gsxhq/gsx/blob/main/docs/guide/syntax/pipelines.md) (`date`,
  `number`, `since`, …). They return a `Value`, which a
  [package renderer](https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/package-renderers.md) written in gsx
  renders (`render.gsx`).
- **gsxui:** its components' own text (`i18n.T`, the English as the id) is generated from the catalogs.
- **gsx's own checks:** `intl:check` runs `gsx generate`, `gsx fmt` and the `gsx` skill's `find-smells.sh` on the
  app's markup. The smell finder can't tell one shape from a component call, a message with arguments alone on a line
  (`{ locales.Notes(n) }`), and `intl:check` lets that one through. `gsx info` lists intl's filters and renderers.

## Set up an app

[mise](https://mise.jdx.dev) brings every tool: Go (from `go.mod`'s `toolchain`, which `init` tells mise to use),
gsx (a `go tool`), and [workerd](https://github.com/cloudflare/workerd), whose Intl the tooling and the tests run on.
Translating with Claude also needs [Claude Code](https://code.claude.com) (`claude`) installed.

1. Add intl's command and set the app up, from the directory of its `go.mod`:

   ```sh
   go get -tool github.com/joeblew999/go-htmx4/intl/cmd/intl
   go tool intl init -locales en,de,ar
   ```

   `init` writes:
   - `intl.toml`, the source catalog and a catalog per language;
   - intl's filters and renderers into `gsx.toml`;
   - the generated files into `.gitignore`;
   - intl's mise tasks to `.mise/tasks/intl.toml`, where mise finds them with no configuration
     ([Task discovery](https://mise.jdx.dev/tasks/task-discovery.html));
   - the `intl-translate` skill into `.claude/skills`.

   It picks up gsxui from `gsxui.json`, and then compiles the catalogs. Running it again leaves what is there
   alone. After upgrading intl, `mise run intl:tasks` and `mise run intl:skill` write those two again, and
   `intl:check` says when they're out of date.

2. Serve every request through `intl.Site`. It handles the locale URLs (`/about`, `/de/about`), the locale cookie,
   `Content-Language`, noindex while a catalog is incomplete, and the viewer's time zone and clock:

   ```go
   site := intl.Site{
   	IsPage:             func(path string) bool { return !strings.HasPrefix(path, "/api/") },
   	ConnectionTimeZone: func(r *http.Request) string { return transport.CFOf(r).Timezone }, // Cloudflare's guess
   }
   mux.HandleFunc("POST /preferences", site.Preferences) // tz, hc, return: the viewer's choice
   http.ListenAndServe(addr, site.Handler(mux))
   ```

3. In the layout:

   ```gsx
   <html { intl.HTML(ctx)... }>                  // lang, dir, and the viewer's data-tz / data-hc
   <head>
   	<intl.Head page="/about"/>                 // canonical, hreflang alternates, og:url, og:locale
   	<script type="module" src="/static/intl-browser.mjs"></script>
   ```

   The browser redoes each `[data-intl]` element in the viewer's zone and clock, keeps `since` fresh, and redoes what
   htmx swaps in. A language list is `intl.Links(ctx)` (each locale's name in its own language, and the page's URL
   in it); `intl.Sitemap(origin, pages)` is a sitemap.xml with every alternate.

4. Wire the Worker: its entry module puts intl's formatter where Go calls it, and the build copies the JavaScript
   (embedded in the module):

   ```js
   import { goIntl } from "./build/intl-worker.mjs";
   globalThis.goIntl = goIntl;
   ```

   ```sh
   go tool intl assets -worker build -browser dist/static
   ```

5. Test with workerd's Intl, in every package that renders:

   ```go
   func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }
   ```

6. Day to day: `mise run intl` before `gsx generate` (make your generate task depend on it), and
   `mise run intl:check` in CI.

`intl.toml` holds the languages, where the catalogs are, and notes for translators:

```toml
locales = ["en", "de", "ar"]   # the source first
catalogs = "locales"
gsxui = "ui"

[notes]
"*" = "A shop. Keep brand names in English."
de = "Address the reader as Sie."
```

## In markup

| Write | Renders |
|---|---|
| `{ locales.Key }`, `{ locales.Key(args…) }` | a message, in text or an attribute; text arguments are bidi-isolated |
| `{ t \|> date }`, `{ t \|> date("dateStyle", "full", "calendar", "hebrew") }` | `Intl.DateTimeFormat`, in a `<time>` |
| `{ start \|> dateRange(end, opts) }` | its `formatRange` |
| `{ t \|> since }` | `Intl.RelativeTimeFormat` in the largest readable unit, kept fresh by the browser |
| `{ n \|> number }`, `{ n \|> number("style", "percent") }` | `Intl.NumberFormat`, in a `<data>`; an int, a float or an exact decimal string |
| `{ d \|> duration }` | `Intl.DurationFormat` |
| `{ items \|> list(opts) }` | `Intl.ListFormat` |
| `{ id \|> zone }` | a time zone's name: `Europe/Berlin · Mitteleuropäische Zeit (GMT+2)` |
| `{ v \|> intl(api, method, opts…) }` | any call: `new Intl[api](lang, opts)[method](v)`; api `"Intl"` for `Intl.supportedValuesOf` and the like |
| `title={ v \|> plain }` | a Value's text alone: attributes take no markup |
| `title={ locales.Key \|> text }` | a message's text, for a component input typed `string` (gsxui's `Toast title`) |
| `<option { v.Attrs()... }>{ v \|> plain }</option>` | an element that can't hold the Value's own, still redone by the browser |
| `href={ "/about" \|> localize }`, `hx-post={ f`/cart/add?id=@{id}` \|> localize }` | an in-app path in the request's locale: `/de/about` |

Options are Intl's own option names and values, in pairs (`"maximumFractionDigits", "0"`: a number or true/false
is one), or one JSON object. Results that aren't strings (`resolvedOptions`, `select`, `compare`, `segment`) render as
JSON.

In Go, the same values are `intl.Date(t, opts)`, `intl.Of(api, method, opts, args…)` and the rest, with options
from `intl.Options(pairs…)`, and `intl.Text(ctx, x)` is the text (an HTTP error, a log line). For what markup can't
say:

| Go | Does |
|---|---|
| `intl.Order(ctx, names, opts)` | the order `Intl.Collator` sorts names in: sort rows by a name (Ägypten before Österreich in German, after it in Swedish) |
| `intl.Segments(ctx, text, "grapheme" \| "word" \| "sentence")` | text split by `Intl.Segmenter`: count what a reader sees as characters, take initials, find words in Japanese |
| `intl.CalendarOf(ctx)`, `intl.WeekStart(ctx)` | a month calendar's words and its first day: `<ui.Calendar locale={ ui.CalendarLocale(intl.CalendarOf(ctx)) } weekStartsOn={ intl.WeekStart(ctx) }/>` |
| `intl.Current(ctx).ParseNumber(text)` | a number typed the language's way, as a plain decimal (`window.intl.parseNumber` in the browser) |

## Translating

| When | Run |
|---|---|
| You added or changed messages in the source catalog | `mise run intl`, then `mise run intl:translate` (Claude Code) or translate what `mise run intl:todo` lists |
| A translation's source changed | it is **stale**: `intl:todo` shows the old translation beside the new source. Update it, or keep it: `mise run intl:confirm de:order.items` |
| You add a language | add its id to `intl.toml`'s `locales`, then `mise run intl:translate`. A regional one can take what it doesn't translate from another: `[inherit] fr-CA = "fr"` |
| You delete or rename a source message | `intl:check` names the keys left in the translations; `mise run intl:prune` removes them |
| `gsxui add pagination` | `mise run intl` names each new text with the line to add under `[gsxui]`; add them, then translate |
| A person translates | `mise run intl:export` writes XLIFF 2.0 per locale into `xliff/out` for their tool; put what comes back in `xliff/in`, and `mise run intl:import` reads it, checking each target (one translated from older English is refused) |
| CI | `mise run intl:check` |
| Leftover English in the UI | `intl.Pseudo = true` renders the `en-XA` catalog: accented, bracketed (`⟦Ĥéļļó⟧`), 40% longer |

`intl:translate` runs Claude Code headless with the `intl-translate` skill, one worker per language at once
(`intl.toml`'s `[translate] workers`, default 6, on `[translate] model`). A worker reads `intl:todo -json` for its
language (each message's source, context and arguments, the plural categories from Intl, the notes from
`intl.toml`). It may edit only that language's catalog, and it checks its work with `go tool intl check <locale>`
until that passes. Then `gen` and the full `intl:check` run once. The skill is in `skills/` and works the same in an
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

## Tasks

| Task | Does |
|---|---|
| `intl` | compile the catalogs (`go tool intl gen`) |
| `intl:status` | per locale: translated, inherited, stale, missing; exit 1 when one is incomplete |
| `intl:todo [--json] [locale…]` | what is left to translate, with what a translator needs |
| `intl:translate [locale…]` | translate it with Claude Code |
| `intl:check` | every problem, for CI |
| `intl:confirm [locale[:key]…]` | keep stale translations that are still right (all by default) |
| `intl:export [--all] [locale…]`, `intl:import [file…]` | XLIFF 2.0 for human translators: out to `xliff/out`, back from `xliff/in` |
| `intl:prune` | remove keys the source no longer has |
| `intl:skill` | install or update the agent skill |
| `intl:tasks` | write these tasks again, after upgrading intl |

Each one is a `go tool intl` command, so they also run without mise.

## The module

| Path | What |
|---|---|
| `*.go`, `render.gsx`, `head.gsx` | the package: `Msg`, `Value`, `Locale`, `Request`, the renderers, `Site` (URLs, preferences, SEO), `Head` |
| `*.x.go` | `render.gsx` and `head.gsx` generated, committed: apps import intl as a Go package |
| `filters/` | the gsx filters |
| `js/intl.mjs` | the formatter, in the Worker and the browser |
| `js/intl-worker.mjs`, `js/intl-browser.mjs` | what Go calls; what redoes `[data-intl]` in the browser |
| `catalog/` | the catalogs: read, checked, compiled, what is left, intl.sum, XLIFF |
| `cmd/intl/` | `go tool intl` |
| `config/` | `intl.toml` |
| `setup/` | `go tool intl init` |
| `intltest/` | workerd's Intl for `go test` and the tooling |
| `skills/intl-translate/` | the agent skill |
| `mise/tasks.toml` | the mise tasks |
