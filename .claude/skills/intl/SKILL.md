---
name: intl
description: Use when writing or translating what a reader of a Go app sees (gsx or not), with github.com/joeblew999/mise-intl/intl — adding or changing text (messages in the source catalog), writing them, dates, numbers and other formats in markup, after `gsxui add`, when `mise-intl status` or `check` reports missing or stale translations, when adding a language, or when asked to translate or review translations.
---

# Text, formats and translations with intl

## Writing text, dates and numbers

Every word a reader sees is a message in the source catalog (`<catalogs>/<source>.toml`, ICU MessageFormat), never a
literal in markup or Go. Add the key under its table with a comment saying where it appears (translators get it as
context), then `mise-intl gen`:

```toml
[shop]
# The checkout button.
checkout = "Go to checkout"
```

| Write | For |
|---|---|
| `{ locales.ShopCheckout }`, `title={ locales.ShopCheckout }` | a message (the catalogs' package, here `locales`), in text or an attribute |
| `{ locales.ShopCart(n) }` | a message with arguments: a text argument takes a string, another message or a Value; `{n, number}` and `plural` take any Go number |
| `{ locales.ShopTitle \|> text }` | a message for a component input typed `string`; in Go code (an HTTP error) `intl.Text(ctx, m)` |
| `{ t \|> date("dateStyle", "long") }`, `{ t \|> since }`, `{ d \|> duration }` | a date, a relative time, a duration |
| `{ n \|> number("style", "currency", "currency", "EUR") }`, `{ names \|> list }` | a number, a list |
| `{ "JP" \|> intl("DisplayNames", "of", "type", "region") }`, `{ intl.Args{a, b} \|> intl("Collator", "compare") }` | any other Intl call, with one argument or several |
| `title={ v \|> plain }` | a formatted value in an attribute |
| `locales.Placed(intl.Date(t, intl.Options("dateStyle", "long")))` | a formatted value inside a message's arguments (gsx allows no pipeline there) |
| `href={ "/about" \|> localize }`, `href={ pages.About{} \|> url }` | an in-app link, in the reader's language: a path, or a page type with a `Path()` |
| `{ locales.IntlAlsoIn(s.Name).In(s.Lang) }` | a message written in another shipped language than the page's (the offer `intl.Suggested` makes, with `intl.Decline` beside it); the element around it gets `lang` |

Options are Intl's own names and values, in pairs. Never format for a reader in Go (`time.Format`, `strconv`): the
runtime's Intl does it. A new or changed message is translated in the same change: the steps below.

## Translating

The app's languages are in `intl.toml` (the source locale first); each has a catalog, `<catalogs>/<locale>.toml`, in
ICU MessageFormat. Run every command from the app's root. With mise, each is a task too, named for it
(`mise run intl:gen` runs `mise-intl gen`; `mise run intl:show-help` lists them), and the repo's
`docs/intl/README.md` has everything else. In a gsx app, the components, the renderer and the filters are
`github.com/joeblew999/mise-intl/gsxintl`.

### 1. See what to translate

```sh
mise-intl todo -json
```

It is `{"source": "<source locale>", "locales": [ … ]}`, one entry for each other locale:
- `id` (the BCP 47 id, e.g. `pt-BR`) and `name`.
- `file`: the catalog to edit.
- `note`: the app's guidance for that language: tone, formality, terms to keep. Follow it.
- `plurals.cardinal` and `plurals.ordinal`: the categories this language's `Intl.PluralRules` uses.
- `todo`: the messages to translate. Each has `key`, `source` (ICU MessageFormat), `context` (where it appears),
  `args`, and `state`. A `stale` message also has `translation`: the current translation of a source that has since
  changed.

Nothing in `todo` means nothing to do.

### 2. Translate each message

- Keep every `{argument}` name exactly as in the source. Translate the text around it. Arguments may move.
- Keep the structure of `plural`, `selectordinal` and `select`. In a `plural`, write one case for **each** category
  in that locale's `plurals.cardinal`, no more; in a `selectordinal`, each in `plurals.ordinal`. `other` is always
  one of them. Exact cases (`=0 {…}`) may stay. `#` is the number.
- An apostrophe before `{`, `}` or `#` starts quoted text in ICU: write a literal one as `’` or `''`.
- Product names, code and technical names stay as the source writes them, unless the note says otherwise.
- Match the source's register and length: a button label stays a short button label. Use the wording people of
  that language expect from software (`gsxui.*` keys are the text UI components write themselves: Close, Next,
  Previous, More pages).
- A `stale` message: compare `translation` with the new `source`. Update it, or, when it is still right, keep it:
  `mise-intl confirm <locale>:<key>`.

### 3. Write them

Edit the locale's `file`: the same table and key as the source catalog (`[board]` then `notes = "…"`), TOML basic
strings (escape `"` and `\`). Keep what the file already has; put new keys where the source has them.

Never edit the source catalog's text, `intl.sum`, or generated files (`intl_gen.go`, `intl-messages.mjs`).

### 4. Check

When you were asked to translate one locale while others are translated at the same time (`mise-intl
translate`), edit only that locale's file, never run `mise-intl gen`, and check with `mise-intl check <locale>`,
which writes nothing. Otherwise:

```sh
mise-intl gen && mise-intl check
```

`check` names every problem: a syntax error, an argument the source doesn't have, a plural without a category the
language needs, a stale or missing message. Fix them all and run it again until it passes. Then report, per locale,
how many messages you translated and confirmed.

## Every task

Each runs the mise-intl command of its name (`mise run intl:gen` runs `mise-intl gen`). Start with `mise run intl`:
it says what's done and what to run next.

| Task | Run it | What it does |
|---|---|---|
| `mise run intl` | first, and whenever you're unsure | what's done for the app's languages, and what to run next |
| `mise run intl:init` | after upgrading intl | intl's files written again: its tasks, the workflow, this skill, the docs, the hook |
| `mise run intl:check` | after any change, before pushing | every problem: syntax, arguments, plural forms, missing and stale translations, unused keys |
| `mise run intl:show` | to see where the app is | the same as `mise run intl` |
| `mise run intl:gen` | after any change to a catalog | the catalogs compiled into the app's messages package, and `intl.sum` |
| `mise run intl:status` | to see how far each language is | per language: translated, inherited, stale, missing |
| `mise run intl:add-locale` | to add a language | its id as Intl writes it, into `intl.toml`, and its catalog |
| `mise run intl:todo` | before translating | what's left to translate, with what a translator needs |
| `mise run intl:translate` | when the person asks for every language translated | a Claude Code worker per language, each with this skill |
| `mise run intl:export` | for human translators | XLIFF 2.0 per language into `xliff/out` |
| `mise run intl:import` | when their files are back | their XLIFF, from `xliff/in`, read into the catalogs |
| `mise run intl:confirm` | for a stale translation that is still right | kept as it is: a language's, or one message's |
| `mise run intl:prune` | when check names a key the source no longer has | the key removed from every catalog |
| `mise run intl:verify` | before every push | what `.github/workflows/intl.yml` runs on GitHub: the repo's `setup` and `test`, then check |
| `mise run intl:template` | only in a repo with no app yet | the template's little app; it never writes over a file |
| `mise run intl:add` | for a language list, the language offer, or preferences | intl's screens copied into a gsx app, their text translated |
| `mise run intl:daemon` | never by hand: pitchfork runs it (`pitchfork start intl`, which intl does when nothing answers; `pitchfork stop intl`) | Intl outside a Worker: workerd serving it on the files gen writes |
| `mise run intl:assets` | for a Worker build or a page's own modules | the JavaScript they import: intl's, and the app's catalogs |
| `mise run intl:locales` | to list the shipped languages | them, comma-separated |
| `mise run intl:catalogs` | to name the catalogs' Go package | its name |
| `mise run intl:markup` | for gsx's checks | the app's own markup files |
| `mise run intl:show-version` | to say which mise-intl this is | its version |
| `mise run intl:show-help` | to list every command | every command |

Flags and arguments:

- `--json`: one JSON document on stdout instead of text, on `mise run intl`, `05-show` and `09-todo`. Read that,
  never the text.
- `--locales <ids>`, `--catalogs <dir>`, `--gsxui <dir>`: on `03-init`, the languages (the source first: `en,de`),
  the catalogs' folder, and in a gsx app where `gsxui add` puts its components. Once set up, `intl.toml` keeps them.
- `--missing <how>`: on `04-check`, `fail` or `warn` on missing and stale translations (default: `INTL_MISSING`,
  else warn on a GitHub pull request, else fail).
- `--all`: on `11-export`, every message, not only what's left. `--out <dir>`: the folder, `xliff/out`.
- `--diff`: on `add`, how the copies in the app differ, writing nothing. `--overwrite`: replace copies the app
  changed; only when the person says so.
- `--worker <dir>`, `--browser <dir>`: on `assets`, the folder for a Worker build's modules, and the one a page loads
  its modules from.
- Languages after `04-check`, `09-todo`, `10-translate`, `11-export` and `13-confirm` limit them to those: `de fr`.

## Other tasks

- **Add a language:** `mise-intl add-locale fr-CA` (it checks the id with Intl, adds it to `intl.toml`'s
  `locales` and writes its catalog), then translate everything `todo` lists for it. A regional locale can take what it doesn't
  translate from another of the same script: `[inherit] fr-CA = "fr"` in `intl.toml`.
- **A language list, the offer of the page in the reader's language, or a time zone and clock dialog:** don't write
  them; `mise-intl add language-list language-offer preferences` copies intl's own (gsx files in `views/intl`,
  package `intlui`, made of gsxui components) and writes their text, already translated, into the catalogs under
  `[intl]`. It prints where each goes. A language intl doesn't ship: translate what `todo` lists for it.
- **After `gsxui add`:** `mise-intl gen` names each text the new components write that has no message, with the
  line to add to the source catalog's `[gsxui]` table. Add those lines, then translate them.
- **Review:** read each catalog beside the source and fix what reads wrong for a native speaker; then step 4.
- **Human translators:** `mise-intl export` writes XLIFF 2.0 per locale into `xliff/out`; their files go in
  `xliff/in`, and `mise-intl import` reads them back.
- **A key the source no longer has:** `mise-intl check` names it; `mise-intl prune` removes it everywhere.
