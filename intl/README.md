# intl

The runtime's own `Intl` for [gsx](https://github.com/gsxhq/gsx) apps, written in markup. Translated messages and
every Intl API (dates, numbers, lists, plurals, display names, collation, segmentation, `Intl.Locale`) render on the
server with the Worker's V8 Intl, and the browser redoes them in the viewer's own time zone with the same code. Go
decides nothing about a language: no CLDR tables, no fallbacks of its own.

It follows gsx's own design: a typed value in markup, turned into text by a renderer that takes the request context
([Translation keys](https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/i18n-keys.md),
[Package renderers](https://github.com/gsxhq/gsx/blob/main/docs/guide/patterns/package-renderers.md)). The text
[gsxui](https://github.com/gsxhq/gsxui) components write themselves comes from the same catalogs.

```gsx
<h1>{ locales.HomeHeading() }</h1>
<p title={ locales.BoardNotesHint() }>{ locales.BoardNotes(len(notes)) }</p>
<p>{ intl.Date(note.CreatedAt, `{"dateStyle":"full","timeStyle":"short"}`) }</p>
<p>{ intl.Since(note.CreatedAt) } · { intl.Int(views) }</p>
<p>{ intl.Of("DisplayNames", "of", `{"type":"region"}`, intl.JSON("JP")) }</p>
```

For a German request in Berlin that is `Samstag, 4. Juli 2026 um 17:30`, `vor 3 Minuten · 1.234.567` and `Japan`.
Each Value arrives in an element that carries its call, which the browser redoes:

```html
<time datetime="2026-07-04T15:30:00Z" data-intl="DateTimeFormat" data-opts='{"dateStyle":"full","timeStyle":"short"}'
      data-args="[1783179000000]">Samstag, 4. Juli 2026 um 17:30</time>
```

## Requirements

[mise](https://mise.jdx.dev) installs everything: Go (from `go.mod`'s `toolchain`), gsx (a `go tool`), and
[workerd](https://github.com/cloudflare/workerd), the Workers runtime, whose Intl runs the tests:

```toml
# mise.toml
[tools]
workerd = "1.20261019.1"
```

## Set up an app

1. Add the module and its command:

   ```sh
   go get github.com/joeblew999/go-htmx4/intl
   go get -tool github.com/joeblew999/go-htmx4/intl/cmd/intl
   ```

2. Say which languages, the source first, in `intl.toml` at the module root:

   ```toml
   locales = ["en", "de", "ar"]
   catalogs = "locales"   # locales/<locale>.toml, and the generated package
   gsxui = "ui"           # optional: where gsxui add vendors components
   ```

3. Write the catalogs, ICU MessageFormat (plural, selectordinal, select, number), one per locale:

   ```toml
   # locales/en.toml
   page_title = "{title} · My app"
   [board]
   notes = "{n, plural, one {# note} other {# notes}}"
   [gsxui]
   close = "Close"
   ```

   A key missing from a translation comes from the nearest shipped locale its id shortens to (`en-IN` → `en`), else
   the source; `intl.Complete(id)` says whether a catalog translates everything.

4. Add the mise task that compiles them, and run it before `gsx generate`:

   ```toml
   # mise.toml
   [tasks.intl]
   sources = ["intl.toml", "locales/*.toml", "ui/*.gsx"]
   outputs = ["locales/intl_gen.go", "locales/intl-messages.mjs", "ui/i18n/intl_gen.go"]
   run = "go tool intl gen"
   ```

   `go tool intl gen` checks every message against the source (arguments and their kinds) and writes:

   | File | What |
   |---|---|
   | `locales/intl_gen.go` | a constructor per key returning an `intl.Msg` (`board.notes` → `locales.BoardNotes(n)`), and the shipped locales |
   | `locales/intl-messages.mjs` | the catalogs as data for the formatter, plus `en-XA`, pseudo-localized |
   | `ui/i18n/intl_gen.go` | with `gsxui`: the renderer for gsxui's `i18n.T` |

   All three are generated: don't commit them.

5. Register the renderers in `gsx.toml`:

   ```toml
   [renderers]
   "github.com/joeblew999/go-htmx4/intl.Msg" = "github.com/joeblew999/go-htmx4/intl.Translate"
   "github.com/joeblew999/go-htmx4/intl.Value" = "github.com/joeblew999/go-htmx4/intl.Render"
   "github.com/joeblew999/go-htmx4/intl.Plain" = "github.com/joeblew999/go-htmx4/intl.PlainText"
   "example.com/app/ui/i18n.T" = "example.com/app/ui/i18n.Translate"   # with gsxui
   ```

6. Put the request's language in the context, in middleware:

   ```go
   ctx := intl.WithRequest(r.Context(), intl.Request{
   	Locale:   locales.Shipped.Find("de"), // from the path, a cookie or Accept-Language
   	TimeZone: "Europe/Berlin",            // the viewer's choice, else the connection's guess
   })
   ```

   A render without a request uses the source locale.

7. Wire the Worker and the page. `go tool intl assets` copies the JavaScript, embedded in this module, where they
   load it:

   ```sh
   go tool intl assets -worker build           # intl.mjs, intl-worker.mjs, intl-messages.mjs
   go tool intl assets -browser dist/static    # intl.mjs, intl-browser.mjs
   ```

   ```js
   // the Worker's entry module, before Go runs
   import { goIntl } from "./build/intl-worker.mjs";
   globalThis.goIntl = goIntl;
   ```

   ```gsx
   <html lang={ intl.Current(ctx).ID() } dir={ intl.Current(ctx).Dir() } data-tz={ chosenZone } data-hc={ chosenClock }>
   <script type="module" src="/static/intl-browser.mjs"></script>
   ```

   The browser redoes every `[data-intl]` element in the viewer's zone (`<html data-tz>` when they chose one) and
   clock (`data-hc`), keeps `intl.Since` fresh, and redoes what htmx swaps in.

8. Run the tests with workerd's Intl:

   ```go
   func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }

   // every plural has a case for each category its language uses (Russian: one, few, many, other)
   func TestPlurals(t *testing.T) { intltest.Plurals(t) }
   ```

## In markup

| Write | Renders |
|---|---|
| `{ locales.Key(args…) }` | the message, in text or an attribute; text arguments are bidi-isolated |
| `{ intl.Date(t, opts) }`, `intl.DateRange(a, b, opts)` | `Intl.DateTimeFormat` `format` / `formatRange`, in a `<time>` |
| `{ intl.Since(t) }` | `Intl.RelativeTimeFormat` in the largest readable unit, kept fresh |
| `{ intl.Number("1234.5", opts) }`, `{ intl.Int(n) }` | `Intl.NumberFormat`, in a `<data>`; numbers stay exact decimals |
| `{ intl.Duration(d, opts) }` | `Intl.DurationFormat` |
| `{ intl.List(items, opts) }` | `Intl.ListFormat` |
| `{ intl.Zone(id) }` | a time zone's name: `Europe/Berlin · Mitteleuropäische Zeit (GMT+2)` |
| `{ intl.Of(api, method, opts, args…) }` | any call: `new Intl[api](lang, opts)[method](...args)`; `api` `"Intl"` for `Intl.supportedValuesOf` and the like |
| `title={ v.Plain() }` | a Value's text alone: attributes take no markup |
| `<option { v.Attrs()... }>{ v.Plain() }</option>` | an element that can't hold the Value's own, still redone by the browser |

Options are JSON objects, arguments JSON values (`intl.JSON(s)` for strings, `intl.Millis(t)` for instants).
Results that aren't strings (`resolvedOptions`, `select`, `compare`, `segment`) render as JSON.

gsx renderers apply where a value renders, not to Go arguments, so a component prop typed `string` (gsxui's
`Toast title`) or a message argument takes the text: `intl.Text(ctx, m)`. Go code (an HTTP error, a log line) uses
the same.

## gsxui

gsxui components write the text the caller can't reach (`Close`, `Next slide`) as `i18n.T`, its English text the id.
With `gsxui` in `intl.toml`, `go tool intl gen` finds every `i18n.T` in the vendored components and translates it from
the catalogs' `[gsxui]` messages, keyed by the English in snake case. After `gsxui add pagination`:

```text
intl gen: gsxui text without a message:
  ui/pagination.gsx writes i18n.T("More pages"): add gsxui.more_pages = "More pages" to locales/en.toml
  ui/pagination.gsx writes i18n.T("Next"): add gsxui.next = "Next" to locales/en.toml
  ui/pagination.gsx writes i18n.T("Previous"): add gsxui.previous = "Previous" to locales/en.toml
```

## Finding leftover English

`intl.Pseudo = true` renders every message from the `en-XA` catalog: accented, bracketed (`⟦Ĥéļļó⟧`) and 40% longer.
Text without brackets on a rendered page didn't come from a catalog, and the length shows where layouts break.

## The module

| Path | What |
|---|---|
| `*.go`, `render.gsx` | the package: `Msg`, `Value`, `Locale`, `Request`, the renderers |
| `render.x.go` | `render.gsx` generated, committed: apps import it as a Go package |
| `js/intl.mjs` | the formatter, in the Worker and the browser: the generic call, messages, relative times, zone names |
| `js/intl-worker.mjs` | what Go calls (`globalThis.goIntl`) |
| `js/intl-browser.mjs` | redoes `[data-intl]` in the viewer's zone and clock |
| `gen/` | the catalog compiler and the gsxui translator |
| `config/` | `intl.toml` |
| `cmd/intl/` | `go tool intl gen`, `assets`, `locales` |
| `intltest/` | workerd's Intl for `go test` |

In this repository `mise run intl` runs it (`tasks/intl.toml`), and `mise run test:go` tests the module.
