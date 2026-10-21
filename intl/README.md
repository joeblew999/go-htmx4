# intl

Every language a [gsx](https://github.com/gsxhq/gsx) app ships, through the runtime's own `Intl`. Messages are ICU
MessageFormat catalogs. Dates, numbers, plurals, lists, names, collation and segmentation are the real `Intl` APIs,
written in markup the gsx way. The workflows a multi-country app lives with are mise tasks: translating (by Claude
Code, by people, or by both), finding stale translations, adding a language, and checking all of it in CI.

```gsx
<h1>{ locales.HomeHeading }</h1>
<p title={ locales.BoardHint }>{ locales.BoardNotes(len(notes)) }</p>
<p>{ order.Placed |> date(`{"dateStyle":"long"}`) } · { order.Total |> number(`{"style":"currency","currency":"EUR"}`) }</p>
<p>{ note.Created |> since } · { "JP" |> intl("DisplayNames", "of", `{"type":"region"}`) }</p>
```

For a German request that renders `4. Juli 2026 · 1.234,50 €` and `vor 3 Minuten · Japan`, formatted by the
Worker's V8 Intl. Each value arrives in an element that carries its call, and the browser redoes it with the same
code in the viewer's own time zone and clock:

```html
<time datetime="2026-07-04T15:30:00Z" data-intl="DateTimeFormat" data-opts='{"dateStyle":"long"}'
      data-args="[1783179000000]">4. Juli 2026</time>
```

Go decides nothing about a language: no CLDR tables, no formatting of its own. Outside a Worker, `go test` and the
tooling use workerd's Intl.

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
- **gsx's own checks:** markup written this way passes the `gsx` skill's `find-smells.sh`. `gsx info` lists intl's
  filters and renderers.

## Set up an app

[mise](https://mise.jdx.dev) brings every tool: Go (from `go.mod`'s `toolchain`), gsx (a `go tool`), and
[workerd](https://github.com/cloudflare/workerd), whose Intl the tooling and the tests run on.

1. Add the module and its command:

   ```sh
   go get github.com/joeblew999/go-htmx4/intl
   go get -tool github.com/joeblew999/go-htmx4/intl/cmd/intl
   ```

2. Include intl's tasks in `mise.toml` ([mise Task discovery](https://mise.jdx.dev/tasks/task-discovery.html):
   a `git::` include pinned to a tag). The list replaces mise's default task directories, so list your own too:

   ```toml
   [task_config]
   includes = ["git::https://github.com/joeblew999/go-htmx4.git//intl/tasks.toml?ref=<tag>", "mise-tasks"]
   ```

3. Name the languages in `intl.toml` at the module root, the source first:

   ```toml
   locales = ["en", "de", "ar"]
   catalogs = "locales"   # locales/<locale>.toml; go tool intl gen writes the Go package there
   gsxui = "ui"           # optional: where gsxui add vendors components

   [notes]                # optional: guidance for translators, people and Claude
   "*" = "A shop. Keep brand names in English."
   de = "Address the reader as Sie."
   ```

4. Write the source catalog. A comment above a key or its table tells translators where the text appears:

   ```toml
   # locales/en.toml
   page_title = "{title} · My shop"

   # The order page.
   [order]
   # Under the order's items; n is how many.
   items = "{n, plural, one {# item} other {# items}}"

   [gsxui]
   close = "Close"
   ```

5. Register intl in `gsx.toml`:

   ```toml
   filter_packages = ["github.com/joeblew999/go-htmx4/intl/filters"]

   [renderers]
   "github.com/joeblew999/go-htmx4/intl.Msg" = "github.com/joeblew999/go-htmx4/intl.Translate"
   "github.com/joeblew999/go-htmx4/intl.Value" = "github.com/joeblew999/go-htmx4/intl.Render"
   "github.com/joeblew999/go-htmx4/intl.Plain" = "github.com/joeblew999/go-htmx4/intl.PlainText"
   "example.com/app/ui/i18n.T" = "example.com/app/ui/i18n.Translate"   # with gsxui
   ```

6. Run `mise run intl` before `gsx generate` (make your generate task depend on it). It writes, beside the
   catalogs, the `locales` package (`intl_gen.go`), the catalogs the formatter reads (`intl-messages.mjs`), and
   `intl.sum`. With `gsxui` set, it also writes `ui/i18n/intl_gen.go`. Commit `intl.sum`; the others are generated
   output.

7. Put the request's language in the context, in middleware:

   ```go
   ctx := intl.WithRequest(r.Context(), intl.Request{
   	Locale:   locales.Shipped.Find("de"), // from the URL, a cookie, Accept-Language
   	TimeZone: "Europe/Berlin",            // the viewer's choice, else the connection's guess
   })
   ```

   `intl.Current(ctx)` is that locale: `<html lang={ intl.Current(ctx).ID() } dir={ intl.Current(ctx).Dir() }>`.

8. Wire the Worker and the page. `go tool intl assets` copies the JavaScript (embedded in the module) where they
   load it:

   ```sh
   go tool intl assets -worker build -browser dist/static
   ```

   ```js
   // the Worker's entry module, before Go runs
   import { goIntl } from "./build/intl-worker.mjs";
   globalThis.goIntl = goIntl;
   ```

   ```gsx
   <html lang={ intl.Current(ctx).ID() } data-tz={ chosenZone } data-hc={ chosenClock }>
   <script type="module" src="/static/intl-browser.mjs"></script>
   ```

   The browser redoes each `[data-intl]` element in the viewer's zone (or `<html data-tz>`) and clock
   (`data-hc`), keeps `since` fresh, and redoes what htmx swaps in.

9. Test with workerd's Intl, in every package that renders:

   ```go
   func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }
   ```

10. For translating with Claude Code: `mise run intl:skill` installs the `intl-translate` skill into
    `.claude/skills`. Commit it.

## In markup

| Write | Renders |
|---|---|
| `{ locales.Key }`, `{ locales.Key(args…) }` | a message, in text or an attribute; text arguments are bidi-isolated |
| `{ t \|> date }`, `{ t \|> date(opts) }` | `Intl.DateTimeFormat`, in a `<time>` |
| `{ start \|> dateRange(end, opts) }` | its `formatRange` |
| `{ t \|> since }` | `Intl.RelativeTimeFormat` in the largest readable unit, kept fresh by the browser |
| `{ n \|> number }`, `{ n \|> number(opts) }` | `Intl.NumberFormat`, in a `<data>`; an int, a float or an exact decimal string |
| `{ d \|> duration }` | `Intl.DurationFormat` |
| `{ items \|> list(opts) }` | `Intl.ListFormat` |
| `{ id \|> zone }` | a time zone's name: `Europe/Berlin · Mitteleuropäische Zeit (GMT+2)` |
| `{ v \|> intl(api, method, opts) }` | any call: `new Intl[api](lang, opts)[method](v)`; api `"Intl"` for `Intl.supportedValuesOf` and the like |
| `title={ v \|> plain }` | a Value's text alone: attributes take no markup |
| `title={ locales.Key \|> text }` | a message's text, for a component input typed `string` (gsxui's `Toast title`) |
| `<option { v.Attrs()... }>{ v \|> plain }</option>` | an element that can't hold the Value's own, still redone by the browser |

Options are an Intl options object as JSON. Results that aren't strings (`resolvedOptions`, `select`, `compare`,
`segment`) render as JSON. In Go, the same values are `intl.Date(t, opts)`, `intl.Of(api, method, opts, args…)`
and the rest, and `intl.Text(ctx, x)` is the text (an HTTP error, a log line).

## Translating

| When | Run |
|---|---|
| You added or changed messages in the source catalog | `mise run intl`, then `mise run intl:translate` (Claude Code) or translate what `mise run intl:todo` lists |
| A translation's source changed | it is **stale**: `intl:todo` shows the old translation beside the new source. Update it, or keep it: `mise run intl:confirm de:order.items` |
| You add a language | `mise run intl:add -- fr-CA`, then `intl:translate` (fr-CA inherits from fr what it doesn't translate) |
| `gsxui add pagination` | `mise run intl` names each new text with the line to add under `[gsxui]`; add them, then translate |
| A person translates | `mise run intl:export` writes XLIFF 2.0 per locale for their tool; `mise run intl:import -- xliff/de.xlf` reads it back, checking each target |
| CI | `mise run intl:check` |
| Leftover English in the UI | `intl.Pseudo = true` renders the `en-XA` catalog: accented, bracketed (`⟦Ĥéļļó⟧`), 40% longer |

`intl:translate` runs Claude Code headless with the `intl-translate` skill. It reads `intl:todo -json`: each
message's source, context and arguments, the locale's plural categories from Intl, and the notes from `intl.toml`.
It writes the catalogs and repeats until `intl:check` passes. The skill is in `skills/` and works the same in an
interactive session ("translate the new messages").

`intl:check` fails on any of these:
- an ICU syntax error, or an argument or argument kind the source doesn't have;
- a plural or selectordinal without a case for a category the language uses (Russian: one, few, many, other);
- a missing or stale translation;
- a message no Go or gsx file uses;
- gsxui text without a message;
- an `intl.sum` that is out of date.

### The catalogs

- One `<locale>.toml` per shipped locale, with the source's tables and keys, in ICU MessageFormat 1 (`plural`,
  `selectordinal`, `select`, `{n, number}`). A key a translation lacks comes from the nearest shipped locale its id
  shortens to (`en-IN` ← `en`). Otherwise it is missing, and the source text shows.
- `intl.sum` records, for every translation, the source text it was translated from (as hashes). That's how a
  changed source shows up as stale. `intl:gen` writes it; editing a translation updates it.
- A locale is complete when nothing is missing or stale: `intl.Complete(id)`, for example to serve an incomplete
  locale `noindex`.

## Tasks

| Task | Does |
|---|---|
| `intl` | compile the catalogs (`go tool intl gen`) |
| `intl:status` | per locale: translated, inherited, stale, missing; exit 1 when one is incomplete |
| `intl:todo [--json] [locale…]` | what is left to translate, with what a translator needs |
| `intl:translate [locale…]` | translate it with Claude Code |
| `intl:check` | every problem, for CI |
| `intl:add <locale>…` | add a language |
| `intl:confirm <locale>[:<key>]…` | keep stale translations that are still right |
| `intl:export [--all] [-o dir] [locale…]`, `intl:import <file>…` | XLIFF 2.0 for human translators |
| `intl:skill` | install or update the agent skill |

Each one is a `go tool intl` command, so they also run without mise.

## The module

| Path | What |
|---|---|
| `*.go`, `render.gsx` | the package: `Msg`, `Value`, `Locale`, `Request`, the renderers |
| `render.x.go` | `render.gsx` generated, committed: apps import it as a Go package |
| `filters/` | the gsx filters |
| `js/intl.mjs` | the formatter, in the Worker and the browser |
| `js/intl-worker.mjs`, `js/intl-browser.mjs` | what Go calls; what redoes `[data-intl]` in the browser |
| `catalog/` | the catalogs: read, checked, compiled, what is left, intl.sum, XLIFF |
| `cmd/intl/` | `go tool intl` |
| `config/` | `intl.toml` |
| `intltest/` | workerd's Intl for `go test` and the tooling |
| `skills/intl-translate/` | the agent skill |
| `tasks.toml` | the mise tasks |
