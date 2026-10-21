---
name: intl-translate
description: Use when translating a gsx app's messages with github.com/joeblew999/go-htmx4/intl — after messages are added or changed in the source catalog, after `gsxui add`, when `go tool intl status` or `check` reports missing or stale translations, when adding a language, or when asked to translate or review translations.
---

# Translating with intl

The app's languages are in `intl.toml` (the source locale first); each has a catalog, `<catalogs>/<locale>.toml`, in
ICU MessageFormat. Run every command from the app's root. `mise run intl:<command>` runs the same thing.

## 1. See what to translate

```sh
go tool intl todo -json
```

For each locale it gives:
- `file`: the catalog to edit.
- `note`: the app's guidance for that language: tone, formality, terms to keep. Follow it.
- `plurals.cardinal` and `plurals.ordinal`: the categories this language's `Intl.PluralRules` uses.
- `todo`: the messages to translate. Each has `key`, `source` (ICU MessageFormat), `context` (where it appears),
  `args`, and `state`. A `stale` message also has `translation`: the current translation of a source that has since
  changed.

Nothing in `todo` means nothing to do.

## 2. Translate each message

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
  `go tool intl confirm <locale>:<key>`.

## 3. Write them

Edit the locale's `file`: the same table and key as the source catalog (`[board]` then `notes = "…"`), TOML basic
strings (escape `"` and `\`). Keep what the file already has; put new keys where the source has them.

Never edit the source catalog's text, `intl.sum`, or generated files (`intl_gen.go`, `intl-messages.mjs`).

## 4. Check

When you were asked to translate one locale while others are translated at the same time (`go tool intl
translate`), edit only that locale's file, never run `go tool intl gen`, and check with `go tool intl check <locale>`,
which writes nothing. Otherwise:

```sh
go tool intl gen && go tool intl check
```

`check` names every problem: a syntax error, an argument the source doesn't have, a plural without a category the
language needs, a stale or missing message. Fix them all and run it again until it passes. Then report, per locale,
how many messages you translated and confirmed.

## Other tasks

- **Add a language:** add its BCP 47 id (e.g. `fr-CA`) to `locales` in `intl.toml`, run `go tool intl gen` (it
  writes the new catalog), then translate everything `todo` lists for it. A regional locale can take what it doesn't
  translate from another of the same script: `[inherit] fr-CA = "fr"` in `intl.toml`.
- **After `gsxui add`:** `go tool intl gen` names each text the new components write that has no message, with the
  line to add to the source catalog's `[gsxui]` table. Add those lines, then translate them.
- **Review:** read each catalog beside the source and fix what reads wrong for a native speaker; then step 4.
- **Human translators:** `go tool intl export` writes XLIFF 2.0 per locale into `xliff/out`; their files go in
  `xliff/in`, and `go tool intl import` reads them back.
- **A key the source no longer has:** `go tool intl check` names it; `go tool intl prune` removes it everywhere.
