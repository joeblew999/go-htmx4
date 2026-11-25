# Rules for agents

Binding. How to run it, and where things are: [README.md](README.md). How it works:
[docs/architecture.md](docs/architecture.md). How it ships: [docs/deploy.md](docs/deploy.md).

The repo is only the app. Its tools are upstream's: gsx, gsxui, Vite, wrangler, and charter to drive them.

## Work

- Plans, open work and follow-ups are GitHub issues here, never files. Name the issue (`#N`) in commits.
- Work is on `main`. Before a push: `go tool charter check`, and `go tool charter test e2e` after a change to pages,
  the board or browser code. `git status` must be clean first.
- End every report with a **Follow-ups** list: what's next, what was found and not fixed, checks skipped, leftovers.

## Tools

- **Go and Node are all a machine needs.** `go.mod` (its toolchain line, its `tool` lines) and `package-lock.json`
  pin everything. A Go tool is `go get -tool`, an npm tool `npm install --save-dev`. No mise, no other installer.
- **Commands are charter's** (`go tool charter …`), which drives gsx, Vite and wrangler. Don't add scripts that copy
  what one of them does; read the upstream guide first.
- **One settings file per tool:** `wrangler.jsonc` (the Worker and its bindings), `charter.toml` (tests, `[intl]`,
  `[seo]`), `gsx.toml`, `gsxui.json`, `vite.config.ts`, `package.json`.
- Checks are Go tests (`e2e/smoke`, `e2e`, `e2e/load`), never curl or a grep of a command's output.
- Stop a process by its PID, never by name: other sessions run workerd and wrangler too.

## Generated files: never edit, never commit

`*.x.go`, `locales/intl_gen.go`, `ui/i18n/intl_gen.go`, `locales/intl-messages.mjs`, `locales/intl-workerd/`,
`web/intl/`, `build/`, `dist/`: `go tool charter build` writes them.

Committed, and written by a tool: what `gsxui.json` lists (`ui/`, `web/gsxui/`: `go tool gsxui add`),
`views/intl/*.gsx` (`go tool charter intl add --overwrite <screen>`), `locales/intl.sum`, `docs/badges/` and the
README's scores table (`go tool charter seo badges`), `patches/` (`npx patch-package <package>`).

## UI: gsx, gsxui, htmx 4

- **Nothing hand-written in the GUI.** Every visual piece is a gsxui component (`go tool gsxui add <name>`), its
  theme, or a pattern from https://ui.gsxhq.dev. No CSS files of our own.
- `views/blocks/` holds the components gsxui doesn't have (a map, a places picker, live video), made of gsxui
  components. They are the app's own.
- **htmx 4 syntax only:** explicit `:inherited`, `hx-ignore`, `htmx:after:swap`-style event names; no `hx-ext`,
  `hx-vars` or `hx-params`.
- Write gsx as the `gsx` skill says. JS-valued attributes are `` js`…` `` literals. Format with
  `go tool gsx fmt -w .`.
- Handlers render with `s.render(w, r, name, node)`: never string-built HTML or `html/template`.
- A full page must not render `hx-swap-oob` elements it wants kept. Everything every page needs loads in `Layout`.
- **Browser code** is in `web/`, built by Vite. `web/main.js` is every page's; a page's own entry (`map.js`,
  `moq.js`) is loaded with `<assets.Entry name="web/…"/>`. Keep it logic-free: logic stays in Go, labels in the gsx.

## Languages

- **Intl is the one source of language** (charter's `site/intl`). Messages, plurals, numbers, dates and direction
  come from the runtime's Intl; Go never decides them. A test package that renders or formats has
  `func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }`.
- No literal reader-facing text in `views/` or handlers: a message in `locales/en.toml`, written `{ locales.Key }` or
  `{ locales.Key(n) }`. Components take `intl.Msg`; Go code takes `intl.Text(ctx, m)`. Brand names, code and machine
  values get `translate="no"`.
- **Every message change is translated in the same commit:** `go tool charter intl translate`, or
  `go tool charter intl confirm` for a stale one that is still right. The check fails while a language is incomplete.
- Dates and numbers go through Intl's gsx filters (`{ t |> date("dateStyle", "long") }`, `{ n |> number }`), never
  `time.Format` or `strconv`. Options are always pairs.
- User text in its own element gets `dir="auto"`. Logical Tailwind classes only (`ms-`, `pe-`, `start-`).
- Never redirect on `Accept-Language`: the server offers the reader's language (`intl.Site`).

## Pages and search

- **Pages are the page tree, `views/pages.go`** (structpages): a type in `views.Pages` with its route in the field's
  tag, a `Page` method, `Props` for data, an entry in `All`, a translated title and description.
- Every in-app URL comes from the tree through the `url` filter (`href={ About{} |> url }`), never a path as a
  string. `structpages-lint` fails on one.
- Titles are at most 60 wide and descriptions 160. `/robots.txt` keeps one `*` group with no `Disallow`. Absolute
  URLs come from `web.Origin(r)`.

## Workers

- **Memory between requests is a cache, never state:** any request may start a new Go runtime. Shared state needs a
  Cloudflare binding.
- **Bindings come through charter, never workers-go directly:** `bindings.Var`, `bindings.Allow`,
  `bindings.StaticAssets`, `bindings.HTTPClient`, `d1.Open`, `hub.DurableObject`, `transport.CFOf(r)`. Each has its
  stand-in off Workers, so there is no file per platform and `go test` runs the real SQL.
- JavaScript only in `worker/` (the entry, the Durable Object) and `web/`.

## Cloudflare

- **Credentials only through fnox** (`fnox exec -- …`). Never print a token or add a `.env` file.
- The zone hosts other projects: only ever add `go-htmx4.*` records.

## The shared board

- D1 is the source of truth; the Room only fans out. Every write goes through a Go handler that writes D1, then
  publishes the whole `#board` fragment with its `data-version`.
- **`board.version` never goes backwards:** never delete `board` rows or reset versions.
- D1 has no interactive transactions: single statements with `RETURNING`, or `d1.Batch`.
- Limits stay in sync: the `WRITES` rate limit in `wrangler.jsonc`, `keepNotes`, `MAX_SOCKETS` in `worker/room.mjs`.
  Every write handler calls `s.limited` first.
- A schema change is a new idempotent `migrations/NNNN_name.sql`. After a board change: `go tool charter check`, and
  `go tool charter test load`.

## Other owners' projects: gsx, gsxui, structpages

Nothing goes to another owner's repository until the user has read the exact text. A report is short: a repro
checked on their `main`, the cause with links to their code, an easy fix when there is one.
