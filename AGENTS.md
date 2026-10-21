# Rules for agents

Binding. How the app works is in [docs/architecture.md](docs/architecture.md), how it ships in
[docs/deploy.md](docs/deploy.md), every command in [docs/commands.md](docs/commands.md) (generated).

## Work

- Plans, open work and follow-ups are GitHub issues on joeblew999/go-htmx4. No plan files in the repo. Before
  non-trivial work, find the issue and update it rather than opening a duplicate; reference it as `#N` in commits, PRs
  and code comments. Close it when its PR merges (`Closes #N`) or say why it's not planned.
- Changes land by branch and PR. `main` is protected: a PR needs a green `check`. Squash merges only.
- End every report of finished work with a **Follow-ups** list, even a short one: what's next, what needs the user's
  OK and why, bugs or risks found but not fixed, checks skipped, leftovers (processes, resources, stale docs). Put the
  same list in the issue.
- **What needs the user's OK** is what ships or can't be taken back: a production deploy or a release (a release tag
  deploys), a Cloudflare write, a write to Google, repository settings, and anything on another owner's repository.
  Everyday GitHub work on this repo doesn't: branches, PRs and auto-merge, issue comments and edits, and deleting a
  tag or branch whose commits are on `main`.

## Toolchain and commands

- Every tool is pinned: `mise.toml`, and Go in `go.mod`'s `toolchain` line (mise installs it). `mise install` first;
  run `mise run <task>`, not ad-hoc commands. No browser installed by hand: e2e uses the pinned `chrome-headless-shell`.
- **No Node anywhere:** no `node`, `npm`, `npx`, Vite or wrangler. CI is shell `run:` steps only: no `actions/checkout`,
  `jdx/mise-action` or any JavaScript action. A new tool goes in `mise.toml`.
- Commands are mise tasks. Don't hand-list them in docs: change the task's description, `docs/commands.md` is
  generated from it (`test:go` fails when it's stale).
- Before pushing: `mise run check` (what CI runs, `ci/check.sh`), then `mise run e2e` after UI, board or JS changes.
  A task with `sources`/`outputs` is skipped when nothing it reads changed: a new input to the wasm goes in `sources`.
- Shell lives in `ci/`, one script per job; tasks call them; checks are Go tests (`e2e/smoke`, `e2e`), not curl.
- Stopping `gsx dev` can leave workerd on :8913; `ci/with-workerd.sh` then refuses to start. Stop it by PID.
- `mise run rename -- <module path> <app name>` turns the template into a new project: keep it working when a file
  mentions the module path or the app name.
- macOS runs bash 3.2: an empty array is `${args[@]+"${args[@]}"}` under `set -u`.

## Generated files: never edit

`*.x.go` (`go tool gsx generate`; not committed, except `intl/render.x.go`: apps import intl as a Go package),
`intl_gen.go` and `intl-messages.mjs` (`mise run intl`; not committed), `locales/intl.sum` (`mise run intl`;
committed), `.claude/skills/intl-translate` (`mise run intl:skill`; committed), `docs/commands.md`
(`mise run docs:commands`), and what `gsxui` manages (`gsxui.json` lists it).

## UI: gsx, gsxui, htmx 4

- **Nothing hand-written in the GUI.** Every visual piece is a gsxui component (`gsxui add <name>` at the repo root,
  never copied by hand), the gsxui theme/preset, or a pattern from gsxui's own site (https://ui.gsxhq.dev; cite it).
  No custom CSS files.
- **htmx 4 syntax only:** explicit `:inherited`, `hx-ignore` (not `hx-disable`), `htmx:after:swap`-style event names,
  no `hx-ext` / `hx-vars` / `hx-params`. gsx's completions still offer htmx 2 attributes: ignore them.
- JS-valued attributes (`hx-on:*`, `hx-live`, `:text`, …) are gsx `` js`…` `` literals. Format `.gsx` with
  `go tool gsx fmt -w .`.
- Handlers render with `s.render(w, r, name, node)` (`httpx.Render`): never string-built HTML or `html/template`.
- A full page must not render `hx-swap-oob` elements it wants kept: boosted navigation drops them. Only pushes and
  POST responses carry `BoardFragment` (OOB).
- Everything every page needs loads in `Layout` (`views/layout.gsx`): a boosted navigation doesn't run a page's own
  head scripts. `<meta name="htmx-config">` stays before htmx's script.

## Languages and search

- **Intl is the one source of language**, through `intl/` (its own module, for any gsx app: `intl/README.md`).
  Messages, plural forms, numbers, dates, locale names and direction all come from `intl/js/intl.mjs` with the
  runtime's Intl; Go never decides them. Outside the Worker the only Intl is workerd's: a test package that renders
  or formats anything has `func TestMain(m *testing.M) { os.Exit(intltest.Main(m)) }`.
- No literal user-facing text in `views/` or handlers: a message from `locales/*.toml` (ICU MessageFormat; `en.toml`
  is the source), written in markup as a value, `{ locales.NavHome }`, or with arguments, `{ locales.BoardNotes(n) }`.
  Our components take `intl.Msg`, not strings. A gsxui input typed `string` takes `{ locales.Key |> text }`; Go code
  (an HTTP error) takes `intl.Text(ctx, m)`. Messages use only `{n, number}` and `{n, number, integer}`. Brand
  names, code and machine values get `translate="no"`. `TestNoHardcodedText` checks.
- **Every message change is translated in the same PR.** A new or changed source message: `mise run intl:translate`
  (Claude Code with the `intl-translate` skill), or translate what `mise run intl:todo` lists by that skill's rules.
  A stale translation that is still right: `mise run intl:confirm -- <locale>:<key>`. `mise run intl:check` (in
  `test:go`) fails on anything missing, stale, unused or wrongly plural. How each language reads is `intl.toml`'s
  `[notes]`: change them there, not per message.
- Text arguments are bidi-isolated by the renderer; user text in its own element gets `dir="auto"`.
- gsxui components' own text (`i18n.T`) is the catalogs' `[gsxui]` table: after `gsxui add`, `mise run intl` names
  every string that needs a message.
- `mise run intl:check` also runs gsx's own checks on the app's markup: `gsx generate`, `gsx fmt` and the `gsx` skill's
  smell finder. Write what the `gsx` skill says, not what older code does.
- Every in-app `href`/`hx-*` URL goes through intl's `localize` filter, `href={ "/path" |> localize }` (not
  `/static/`, `/assets/`, `/gsxui/`, `/live/`). Locale URLs, cookies and preferences are `intl.Site` (`locale.go`).
  Logical Tailwind classes only (`ms-`, `pe-`, `start-`, `text-end`, …) so RTL mirrors (`TestLogicalClasses`).
- Dates and numbers for readers go through Intl, never `time.Format` or `strconv`: intl's gsx filters in markup,
  `{ t |> date(opts) }`, `{ t |> since }`, `{ n |> number(opts) }`, `{ d |> duration }`, `{ v |> intl(api, method,
  opts) }`, and `{ v |> plain }` in an attribute. Every Intl API is reachable that way: add no Go formatting of our
  own.
- A new page: a translated title and its own translated `description` key to `Layout`, an entry in
  `views.SitemapPages`. A query parameter that makes a different page goes in `pageKey`. Titles ≤ 60 and descriptions
  ≤ 160 wide (`TestSnippetWidths`).
- `/robots.txt` keeps one `*` group with no `Disallow` (`TestRobotsTxt`); absolute URLs come from `httpx.Origin(r)`,
  never a hard-coded host. No page routes under `/fragments/` (noindex) and no robots `Disallow` for them.
- Search tasks talk through exit codes (0 ok, 1 problems) and `--json`: never scrape their text. Writes to Google
  (`search:submit`, `search:setup`) ask first; agents pass `mise run --yes` only with the user's OK.

## Workers and TinyGo

- **workers-go `v0.36.0` exactly** (`github.com/syumai/workers-go`). What ships is built by charter's
  `go tool workerbuild` (TinyGo by default, `WORKER_COMPILER`); standard Go only for local code and `build:dev`.
- **TinyGo gaps:** register plain paths (`"/greet"`) and check the method with `httpx.Allow`, never `"GET /x"` or
  `{$}`; `html/template` panics. Reflection-heavy code needs the workerd smoke test, not just a build. Run
  `mise run check` after changing handlers: `go test` can't see TinyGo.
- **Memory between requests is a cache, never state:** any request may start a new runtime. Package variables hold
  only what is the same for every request; shared state needs a Cloudflare binding.
- `cloudflare/*` packages import `syscall/js`: call them from `platform_js.go` with a `platform_other.go` twin.
  Cloudflare's request data is `transport.CFOf(r)`; workers-go's `fetch.NewIncomingProperties` panics under charter's
  runtime.
- JS only in `worker/index.mjs` (entry), `worker/room.mjs` (Durable Object), `intl/js/` (Intl, and what Go calls of
  it) and `workerd/` (local shims). Keep it logic-free; logic stays in Go. The topic and locale rules are
  `internal/live`'s (`TestWorkerJSMatchesLive`).

## Cloudflare: outward-facing

- **Credentials only via fnox** (`fnox exec -- …`; single-quote `sh -c '…$CLOUDFLARE_API_TOKEN…'` so it expands after
  fnox). Never print tokens or add `.env` files.
- **Get an explicit OK before a production deploy, a release, or any Cloudflare write.** Branch previews are
  pre-approved. Never push to `production` by hand: CI moves it on a release.
- Deploy bindings for every kind (production, staging, preview) live in `ci/deploy.sh` only.
- The zone hosts other projects: only ever add `go-htmx4.*` records.

## The shared board

- D1 is the source of truth; the Room only fans out. Every write goes through a Go handler that writes D1 **then**
  publishes the whole `#board` fragment with its `data-version`.
- **`board.version` never goes backwards:** never delete `board` rows or reset versions; to clear a topic, delete
  notes, zero `value` and bump `version`.
- D1 has no interactive transactions: single statements with `RETURNING`.
- `BoardFragment` is the wire format the Room and the version guard rely on: `TestBoardFragmentWireFormat` keeps it.
- Limits stay in sync: the `WRITES` rate limit (60 / 10 s per IP) in `ci/deploy.sh` and `workerd/local-entry.mjs`;
  every write handler calls `s.limited` first; `keepNotes` (50); `MAX_SOCKETS` (1,000) in `worker/room.mjs`.
  New browser checks go into `TestE2E`'s parallel list: the write-limit check runs alone after it.
- Schema changes: a new idempotent `migrations/NNNN_name.sql`, also imported by `workerd/local-d1.mjs`. A new Durable
  Object class bumps `-migration-tag` in `ci/deploy.sh`.
- After a board change: `mise run check`; `mise run load` for the 1,000-socket check.
