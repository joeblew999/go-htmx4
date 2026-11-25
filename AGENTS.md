# Rules for agents

Binding. How the app works: [docs/architecture.md](docs/architecture.md). How it ships:
[docs/deploy.md](docs/deploy.md). Every command: [docs/commands.md](docs/commands.md) (generated). People start with
[CONTRIBUTING.md](CONTRIBUTING.md). Steps marked **(maintainer)** are the repository owner's.

The repo is only the app. Don't add modules, registries, copy steps or another tool's installed files to it.

## Work

- Plans, open work and follow-ups are GitHub issues here, never files. Find the issue before non-trivial work; name it
  (`#N`) in commits and PRs; close it when its PR merges.
- Changes land by branch and PR, squash-merged once the local checks pass: `mise run check`, and `mise run e2e` after
  UI, board or JS changes. `git status` must be clean first. **(maintainer)** `gh pr merge --admin --squash`: nobody
  waits for CI.
- End every report with a **Follow-ups** list: what's next, what needs the user's OK, what was found and not fixed,
  checks skipped, leftovers. Put it in the issue too.
- **What needs the user's OK** is what ships or can't be taken back: a production deploy or release, a Cloudflare
  write, a write to Google, repository settings, anything on another owner's repository. Branch previews, PRs and
  issue edits here don't.

## Tools and commands

- Every tool is pinned: `mise.toml`, and Go in `go.mod`'s `toolchain` line. `mise install` first; run
  `mise run <task>`, not ad-hoc commands. A new tool goes in `mise.toml`.
- **No Node anywhere:** no `node`, `npm`, `npx`, Vite or wrangler, and no JavaScript actions in CI.
- Commands are mise tasks in `tasks/`, each with its script inline. `docs/commands.md` is generated from their
  descriptions (`mise run docs:commands`).
- charter, mise-intl and mise-seo each have a library in `go.mod` and a command in `mise.toml`, at the same version:
  `mise run update` moves both. Of what their `init` commands write, the repo has only charter's (`charter init`:
  `charter.yml`, Workers Builds' launcher, `docs/charter/`, its skill; `tasks = false` leaves its task file out).
  Never edit those.
- **The Worker on Cloudflare is `charter.toml`:** its name, D1, Durable Objects, rate limit, vars and domains. Deploys
  and local workerd (`.charter/`, never committed) come from it.
- Checks are Go tests (`e2e/smoke`, `e2e`, `e2e/load`), never curl or a grep of a command's output.
- Stop a process by its PID, never by name: other sessions run workerd too.

## Generated files: never edit

| File | Written by | Committed |
|---|---|---|
| `*.x.go` | `mise run gsx:generate` | no |
| `locales/intl_gen.go`, `ui/i18n/intl_gen.go`, `locales/intl-messages.mjs`, `locales/intl-workerd/` | `mise run intl:gen` | no |
| `locales/intl.sum` | `mise run intl:gen` | yes |
| `web/blocks/*.js`, `web/blocks/map.css` | `mise run js:bundle` | yes |
| `web/htmx/` | `mise run vendor` | yes |
| `.claude/skills/gsx`, `gsxui`, `structpages`, `templ-to-gsx-migration` | `mise run gsx:skill` | yes |
| `docs/commands.md` | `mise run docs:commands` | yes |
| `docs/badges/`, the README's scores table | `mise run seo:badges`, after a release | yes |
| What `gsxui.json` lists (`ui/`, `web/gsxui/`) | `gsxui add`, `gsxui init` | yes |
| `views/intl/*.gsx` | `mise-intl add -overwrite <screen>` | yes |

## UI: gsx, gsxui, htmx 4

- **Nothing hand-written in the GUI.** Every visual piece is a gsxui component (`gsxui add <name>`), its theme, or a
  pattern from https://ui.gsxhq.dev. No CSS files of our own. The look is gsxui's nova theme; change it only with
  `mise run gsxui:theme`.
- `views/blocks/` holds the components gsxui doesn't have (a map, a places picker, live video), made of gsxui
  components. They are the app's own: change them where they are.
- **htmx 4 syntax only:** explicit `:inherited`, `hx-ignore`, `htmx:after:swap`-style event names; no `hx-ext`,
  `hx-vars` or `hx-params`.
- Write gsx as the `gsx` skill says. JS-valued attributes are `` js`…` `` literals. Format with `mise run gsx:fmt`.
- Handlers render with `s.render(w, r, name, node)`: never string-built HTML or `html/template`.
- A full page must not render `hx-swap-oob` elements it wants kept. Everything every page needs loads in `Layout`.

## Languages

- **Intl is the one source of language** (mise-intl). Messages, plurals, numbers, dates and direction come from the
  runtime's Intl; Go never decides them. A test package that renders or formats has
  `func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }`.
- No literal reader-facing text in `views/` or handlers: a message in `locales/en.toml`, written `{ locales.Key }` or
  `{ locales.Key(n) }`. Components take `intl.Msg`; Go code takes `intl.Text(ctx, m)`. Brand names, code and machine
  values get `translate="no"`.
- **Every message change is translated in the same PR:** `mise run intl:translate`, or `mise run intl:confirm` for a
  stale one that is still right. The check fails while a language is incomplete. How a language reads is
  `intl.toml`'s `[notes]`.
- Dates and numbers go through Intl's gsx filters (`{ t |> date("dateStyle", "long") }`, `{ n |> number }`), never
  `time.Format` or `strconv`. Options are always pairs.
- User text in its own element gets `dir="auto"`. Logical Tailwind classes only (`ms-`, `pe-`, `start-`), so
  right-to-left mirrors.
- Never redirect on `Accept-Language`: the server offers the reader's language (`intl.Site`).

## Pages and search

- **Pages are the page tree, `views/pages.go`** (structpages, as released): a type in `views.Pages` with its route in
  the field's tag, a `Page` method, `Props` for data, an entry in `All`, a translated title and description.
- Every in-app URL comes from the tree through the `url` filter (`href={ About{} |> url }`), never a path as a
  string. `structpages-lint` fails on one.
- Titles are at most 60 wide and descriptions 160. `/robots.txt` keeps one `*` group with no `Disallow`. Absolute
  URLs come from `web.Origin(r)`.
- How Google sees the site is mise-seo and this app's own code (`app/site.go`). `mise run check` audits every page of
  the local build; `mise run seo:lighthouse` scores the live one. Writes to Google are run by hand with the user's OK.

## Workers

- What ships is built with the standard Go compiler (`WORKER_COMPILER` in `mise.toml` says why).
- **Memory between requests is a cache, never state:** any request may start a new runtime. Shared state needs a
  Cloudflare binding.
- **Bindings come through charter, never workers-go directly:** `bindings.Var`, `bindings.Allow`,
  `bindings.StaticAssets`, `bindings.HTTPClient`, `d1.Open`, `hub.DurableObject`, `transport.CFOf(r)`. Each has its
  stand-in off Workers, so there is no file per platform and `go test` runs the real SQL.
- JavaScript only in `worker/` (the entry, the Durable Object) and `views/blocks/js/` (a component's glue to the npm
  library it wraps). Keep it logic-free: logic stays in Go, labels in the gsx.

## Cloudflare

- **Credentials only through fnox** (`fnox exec -- …`). Never print a token or add a `.env` file.
- **An explicit OK before a production deploy, a release, or any Cloudflare write.** Branch previews are
  pre-approved. Never push to `production` by hand: CI moves it on a release.
- The zone hosts other projects: only ever add `go-htmx4.*` records.

## The shared board

- D1 is the source of truth; the Room only fans out. Every write goes through a Go handler that writes D1, then
  publishes the whole `#board` fragment with its `data-version`.
- **`board.version` never goes backwards:** never delete `board` rows or reset versions.
- D1 has no interactive transactions: single statements with `RETURNING`, or `d1.Batch`.
- Limits stay in sync: the `WRITES` rate limit in `charter.toml`, `keepNotes`, `MAX_SOCKETS` in `worker/room.mjs`.
  Every write handler calls `s.limited` first.
- A schema change is a new idempotent `migrations/NNNN_name.sql`. After a board change: `mise run check`, and
  `mise run load`.

## Other owners' projects: gsx, gsxui, structpages

- **The relationships matter more than any one change.** Nothing goes to another owner's repository until the user
  has read the exact text.
- A report is short: a repro checked on their `main` (name the commit), the cause with links to their code, an easy
  fix when there is one. Check every claim. Follow their conventions.
- `mise run upstream:*` works on their checkouts from here.
