# go-htmx4

[![check](https://github.com/joeblew999/go-htmx4/actions/workflows/check.yml/badge.svg)](https://github.com/joeblew999/go-htmx4/actions/workflows/check.yml)

Server-rendered web apps with [Go](https://go.dev), [htmx 4](https://four.htmx.org) and [gsxui](https://ui.gsxhq.dev) on
Cloudflare Workers. One app at the repo root:

- **gsx + gsxui pages**: a form with an out-of-band toast, a dialog and tabs loaded via `hx-get`, client state with hx-live,
  boosted navigation with `outerMorph`, light/dark theme.
- **A live shared board** (`/board`): Go writes to D1, a Durable Object per topic pushes every change to all open tabs over
  hx-ws WebSockets (hibernation), with presence ("N online").
- **Go compiled with TinyGo**, served by [workers-go](https://github.com/syumai/workers-go); the same handlers run under
  `go run .`.

**Live:** https://go-htmx4.gedw99.workers.dev (board: [/board](https://go-htmx4.gedw99.workers.dev/board)).

No Node anywhere: no npm, no Vite, no wrangler. Every tool is pinned in [`mise.toml`](mise.toml).

## Use it as a template

```sh
# GitHub: "Use this template" (or clone), then:
mise install
mise run rename -- github.com/you/my-app my-app   # module path + app name (Worker, D1, brand)
mise run check                                    # proves the renamed copy builds and passes
```

To deploy, put a Cloudflare API token (Workers Scripts, D1 and Workers Tail edit) and your account id in
[fnox](https://github.com/jdx/fnox) as `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`, then `mise run deploy`. Nothing
reads a `.env` file.

## Quick start

```sh
git clone https://github.com/joeblew999/go-htmx4.git
cd go-htmx4
mise install        # every pinned tool (Go, TinyGo, workerd, Tailwind, gsx, …)
mise run dev        # gsx dev: save → regenerate → TinyGo → workerd restart → http://localhost:8913
mise run check      # TinyGo build + workerd smoke + go test
```

| Task | What |
| --- | --- |
| `mise run dev` | `gsx dev` drives the TinyGo build and workerd, restarting on every `.gsx`/`.go` save (~25 s; the real runtime: Durable Objects, D1 shim, pushes) → http://localhost:8913 |
| `mise run serve` | Build once, serve on workerd (static files first, D1 shim, Room DO) → http://localhost:8913 |
| `mise run dev:native` | `gsx dev` on the native server + Tailwind `--watch`: a save is served in ~1 s (memory store, no live push) → http://localhost:9913 |
| `mise run run` | `go run .` (standard Go, memory store, no live push) → http://localhost:9913 |
| `mise run test` / `check` | TinyGo build, workerd smoke (curls every route + a 2-socket push), gsx fmt, gofmt, vet, go test |
| `mise run load` | 1,000 WebSockets on one topic + a 50-write burst + the 1,001st socket refused (local, or `LOAD_BASE=https://…`) |
| `mise run e2e` | Headless Chrome ([`e2e/`](e2e), chromedp): pages, the live board in two browsers, boosted nav ↔ presence, write limit (local, or `E2E_BASE=https://…`) |
| `mise run deploy` | Test, then deploy to `https://$APP_NAME.<subdomain>.workers.dev` (credentials via fnox) |
| `mise run smoke-remote` | Curl every route of the deployed Worker |
| `mise run tail` | Live logs of the deployed Worker (requests, console output, exceptions), no wrangler |

`mise tasks` lists everything. CI runs `mise run check` and `mise run e2e` on every push and pull request
([`ci/check.sh`](ci/check.sh), shell steps only, no JavaScript actions).

## Stack

| Layer | Technology | Version | Notes |
| --- | --- | --- | --- |
| Language | [Go](https://go.dev) | 1.27.1 | Standard Go for tests, `go run .` and tooling |
| Interactivity | [htmx](https://four.htmx.org) | 4.0.0 | Vendored in `static/` with hx-live and hx-ws |
| Toolchain & tasks | [mise](https://mise.jdx.dev) | — | Pins every tool; `mise tasks` lists tasks |
| Templates | [gsx](https://gsxhq.github.io): JSX-style, type-checked Go templates | v0.1.0 | `go tool gsx` |
| UI components | [gsxui](https://ui.gsxhq.dev): shadcn/ui for gsx, npm-free mode | `c7fd6a8` | Vendored with `gsxui add` |
| Client state | [hx-live](https://four.htmx.org/extensions/hx-live/) | 4.0.0 | htmx 4 extension |
| CSS | [Tailwind CSS standalone CLI](https://tailwindcss.com/docs/installation/tailwind-cli) | 4.3.3 | Single binary, no npm |
| Hosting | [Cloudflare Workers](https://workers.cloudflare.com) via [workers-go](https://github.com/syumai/workers-go) | v0.35.0 | Go compiled to wasm |
| Wasm compiler | [TinyGo](https://tinygo.org) | 0.42.0 | Required for everything that ships to Workers |
| Wasm optimiser | [binaryen](https://github.com/WebAssembly/binaryen) `wasm-opt` | 132 | Required by TinyGo wasm targets |
| Local Workers runtime | [workerd](https://github.com/cloudflare/workerd) | 1.20260911.1 | Standalone binary; replaces `wrangler dev` |
| Database | [Cloudflare D1](https://developers.cloudflare.com/d1/) via workers-go's `database/sql` driver | — | Source of truth for the shared board |
| Live updates | [Durable Objects](https://developers.cloudflare.com/durable-objects/) (WebSocket Hibernation) + htmx 4 [`hx-ws`](https://four.htmx.org/extensions/hx-ws) | 4.0.0 | One `Room` per topic pushes fragments to every tab |
| Secrets | [fnox](https://github.com/jdx/fnox) | 1.35.0 | Cloudflare token + account id from the keychain |

Plans, decisions and measurements live in [`.plans/`](.plans/). Licensed under [MIT](LICENSE).

## Upstream references

Read-only checkouts in gitignored `.upstream/`, reproduced without Node.

| Upstream | Run | URL |
| --- | --- | --- |
| **gsxui's own demo** (showcase site, all component examples, theme editor), served by its Go harness. The app's UI patterns are copied from here. | `mise run upstream:gsxui:serve` | http://127.0.0.1:7799 |
| **workers-go v0.35.0** source | `mise run upstream:workers-go:fetch` | |

## Cloudflare Workers without Node

workers-go's docs use `npm create cloudflare` and wrangler. `tasks/app.toml` does the same steps
without them:

- **Build:** `workers-assets-gen -mode=tinygo` + `tinygo build -target wasm` (the template's own build script).
- **Run locally:** `workerd serve`, the runtime `wrangler dev` uses. `workerd/assets-first.mjs` stands in for
  Static Assets (it runs inside workerd, not Node).
- **Deploy:** [`cmd/deploy`](cmd/deploy), a stdlib Go client for Cloudflare's
  [Static Assets Direct Upload](https://developers.cloudflare.com/workers/static-assets/direct-upload/), script upload,
  D1 and Durable Object migration APIs. Credentials come from fnox (`fnox exec -- …`).

**TinyGo 0.42 caveats** (details in the [plan](.plans/done/2026-09-13_1111_adopt-workers-go.md)): its `net/http` has the
pre-Go 1.22 `ServeMux` (no `GET /path` patterns), and `html/template` compiles but panics at runtime. gsx + gsxui work
(verified byte-identical output). `go test` runs on standard Go and can't see TinyGo runtime issues, so the app's test
tasks also curl the real TinyGo build under workerd.

## Live updates on Workers (shared board)

Design and measurements: [workers-realtime plan](.plans/done/2026-09-14_0754_workers-realtime-d1-do.md).

```
browser ─ hx-ws:connect /live/{topic} ─▶ worker/index.mjs ─▶ Room Durable Object (worker/room.mjs, hibernatable WebSockets)
browser ─ hx-post /board/add|note ─────▶ worker/index.mjs ─▶ Go Worker (TinyGo)
                                           1. D1: INSERT … ON CONFLICT DO UPDATE … RETURNING version
                                           2. ROOM stub: POST /publish <#board fragment>
                                                 └▶ Room: ws.send(fragment) to every tab (≤ 5 broadcasts/s, newest wins)
```

- **D1 is the source of truth; the Room only fans out.** Its cached last fragment is sent to (re)connecting tabs.
- **Every fragment is the whole `#board` with a `data-version`;** the page cancels any `htmx:before:swap` older than
  what's on screen, so HTTP responses and pushes can arrive in any order.
- **Designed for 1,000 tabs per topic:** receive-only sockets, `setWebSocketAutoResponse` ping/pong, coalesced broadcasts,
  reconnects spread over 1–3 s (a deploy drops every socket at once). Measured live: 1,000/1,000 delivered, p50 386 ms
  from the click on a phone hotspot.
- **Abuse limits:** writes are rate limited per client IP (Workers Rate Limiting binding `WRITES`, 60 per 10 s per
  location; over it: 429 + a toast), each topic keeps its newest 50 notes, and a Room accepts at most 1,000 sockets
  (the 1,001st gets 503). Topics are 1–32 of `a-z 0-9 -`.
- **Presence:** the Room pushes "N online" on connects and closes, through the same coalescing (one update for a
  1,000-socket join in the local test).
  It counts foreground tabs: hx-ws closes a hidden tab's socket (`pauseOnBackground`) and reconnects when it's shown.
- **Markup is gsx** (`views/board.gsx`), rendered by TinyGo; a test pins the pushed fragment's wire format (`id`, `hx-swap-oob`, `data-version`).
- **JS only where Go can't go:** `index.mjs` (workers-go has no WebSocket support) and `room.mjs` (workers-go can only call
  Durable Objects). Locally, `workerd/local-d1.mjs` gives Go a D1-shaped `DB` over Durable Object SQLite, so the same
  `database/sql` code runs on workerd without miniflare.

## Reusable packages (`kit/`)

Importable from other repos with `go get github.com/joeblew999/go-htmx4/kit/…`. They don't import the app (checked by
`mise run test`) and are tested without Cloudflare (fake API, fake Room).

| Package | What |
| --- | --- |
| [`kit/cfdeploy`](kit/cfdeploy) | Deploy a workers-go Worker with the REST API: Static Assets Direct Upload, script upload, plain-text/D1/Durable Object/rate limiting bindings, D1 create + migrations, Durable Object migrations, workers.dev. No wrangler. |
| [`kit/wsload`](kit/wsload) | End-to-end live-board check over real WebSockets: delivery latency, coalescing, presence, heartbeat, cached fragment. |
| [`kit/cftail`](kit/cftail) | Stream a deployed Worker's live logs: start a tail with the REST API, read its WebSocket, delete it on exit. No wrangler. |
| [`kit/ratelimit`](kit/ratelimit) | Call a Workers Rate Limiting binding from Go (TinyGo), plus the per-client key (`CF-Connecting-IP`). |
| [`kit/live`](kit/live) | Go side of per-topic live updates: `Publish` to a Room Durable Object (js/wasm), the shared topic rule and version header. |
| [`kit/httpx`](kit/httpx) | TinyGo-safe HTTP helpers: `Allow` (method check for plain-path routes), `Render` (buffered gsx, Content-Length), `GetOnly`. |

The Room and Worker entry are JavaScript (`worker/room.mjs`, `worker/index.mjs`) and come with the template, not `kit/`.

## Project layout

```
.
├── mise.toml            # pinned tools, APP_NAME, `check`; includes tasks/*.toml
├── go.mod               # module github.com/joeblew999/go-htmx4 (the app)
├── main.go board.go store_*.go platform_{js,other}.go   # handlers, D1/memory stores, platform split
├── views/               # pages + fragments (.gsx) composed from gsxui
├── ui/ web/gsxui/       # gsxui components, behaviours, CSS entry, fonts (gsxui add)
├── static/              # vendored htmx 4 + hx-live + hx-ws
├── worker/              # index.mjs (Worker entry) + room.mjs (Room Durable Object): deployed
├── workerd/             # config.capnp + local-only shims (static files first, D1 over DO SQLite)
├── migrations/          # D1 schema
├── kit/                 # importable packages (see below)
├── cmd/deploy cmd/tail cmd/wsload  # CLIs over kit/cfdeploy, kit/cftail, kit/wsload
├── e2e/                 # headless Chrome checks (own Go module: chromedp stays out of the app)
├── tasks/
│   ├── app.toml         # dev, serve, run, test, smoke, load, deploy, smoke-remote, …
│   └── upstream.toml    # upstream:gsxui:*, upstream:workers-go:fetch
└── .plans/              # timestamped plans with findings
```
