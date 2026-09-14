# go-htmx4

Server-rendered web apps with [Go](https://go.dev) and [htmx 4](https://four.htmx.org), in two demos:

- **[`demos/gsxui`](demos/gsxui)**: gsx + gsxui components + htmx 4 + hx-live. Runs as a native Go server **and** on
  Cloudflare Workers from the same code.
- **[`demos/workers`](demos/workers)**: htmx 4 on Cloudflare Workers with Go (TinyGo), including a live **shared
  board**: D1 + a Durable Object per topic push every change to all open tabs over WebSockets.

No Node anywhere: no npm, no Vite, no wrangler. Every tool is pinned in [`mise.toml`](mise.toml).

## Quick start

```sh
git clone https://github.com/joeblew999/go-htmx4.git
cd go-htmx4
mise install                      # every pinned tool (Go, TinyGo, workerd, Tailwind, gsx, …)
mise run demo:gsxui:run           # gsxui demo, native      → http://localhost:7777
mise run demo:workers:serve       # Workers demo on workerd → http://localhost:8913/board
mise run check                    # test both demos
```

`mise tasks` lists everything.

## Stack

| Layer | Technology | Version | Notes |
| --- | --- | --- | --- |
| Language | [Go](https://go.dev) | 1.27.1 | Native server for the gsxui demo; standard Go for tests and tooling |
| Interactivity | [htmx](https://four.htmx.org) | 4.0.0 | Vendored per demo |
| Toolchain & tasks | [mise](https://mise.jdx.dev) | — | Pins every tool; `mise tasks` lists tasks |
| Templates | [gsx](https://gsxhq.github.io): JSX-style, type-checked Go templates | v0.1.0 | `go tool gsx` in `demos/gsxui` |
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

Plans, decisions and measurements live in [`.plans/`](.plans/).

## Demos

| Demo | Run | URL |
| --- | --- | --- |
| **gsxui demo** ([`demos/gsxui`](demos/gsxui)): gsx + gsxui + htmx 4 + hx-live. Form with an OOB toast, dialog and tabs loaded via `hx-get`, boosted nav with `outerMorph`, hx-live counter and filter. | `mise run demo:gsxui:run`<br>`mise run demo:gsxui:dev` (live rebuild)<br>`mise run demo:gsxui:test` | http://localhost:7777 |
| **gsxui demo on Workers**: the same app built with TinyGo (`platform_js.go`), static files as Workers Static Assets. HTML is byte-identical to the native server. | `mise run demo:gsxui:workers:serve`<br>`mise run demo:gsxui:workers:smoke`<br>`mise run demo:gsxui:workers:deploy` | http://localhost:8918<br>live: https://go-htmx4-gsxui-demo.gedw99.workers.dev |
| **Workers demo** ([`demos/workers`](demos/workers)): htmx 4 on Cloudflare Workers via workers-go and TinyGo. Fragment round-trip, form post, a counter that resets on every request, Static Assets, and the **shared board** at `/board`. | `mise run demo:workers:serve` (workerd)<br>`mise run demo:workers:run` (`go run .`)<br>`mise run demo:workers:test`<br>`mise run demo:workers:load` (1,000 WebSockets)<br>`mise run demo:workers:deploy` | http://localhost:8913 (`/board`)<br>http://localhost:9913<br>live: https://go-htmx4-workers-demo.gedw99.workers.dev/board |

### Upstream references

Each tool's own demo, reproduced without Node. The checkouts live in gitignored `.upstream/`.

| Upstream demo | Run | URL |
| --- | --- | --- |
| **gsxui's own demo** (showcase site, all component examples, theme editor), served by its Go harness | `mise run upstream:gsxui:serve` | http://127.0.0.1:7799 |
| **workers-go's template** (`worker-tinygo`) and `_examples/env` | `mise run upstream:workers-go:serve`<br>`mise run upstream:workers-go:env` | http://localhost:8911<br>http://localhost:8912 |
| **workers-go's Durable Object example** and **Cloudflare's WebSocket Hibernation example**, on plain workerd | `mise run upstream:workers-go:do`<br>`mise run upstream:cf:ws-hibernation` | http://localhost:8914<br>ws://localhost:8915/ws |

## Cloudflare Workers without Node

workers-go's docs use `npm create cloudflare` and wrangler. `tasks/workers.toml` and `tasks/gsxui.toml` do the same steps
without them:

- **Build:** `workers-assets-gen -mode=tinygo` + `tinygo build -target wasm` (the template's own build script).
- **Run locally:** `workerd serve`, the runtime `wrangler dev` uses. `demos/workers/workerd/assets-first.mjs` stands in for
  Static Assets (it runs inside workerd, not Node).
- **Deploy:** [`demos/workers/cmd/deploy`](demos/workers/cmd/deploy), a stdlib Go client for Cloudflare's
  [Static Assets Direct Upload](https://developers.cloudflare.com/workers/static-assets/direct-upload/), script upload,
  D1 and Durable Object migration APIs. Credentials come from fnox (`fnox exec -- …`).

**TinyGo 0.42 caveats** (details in the [plan](.plans/2026-09-13_1111_adopt-workers-go.md)): its `net/http` has the
pre-Go 1.22 `ServeMux` (no `GET /path` patterns), and `html/template` compiles but panics at runtime. gsx + gsxui work
(verified byte-identical output). `go test` runs on standard Go and can't see TinyGo runtime issues, so the demos' test
tasks also curl the real TinyGo build under workerd.

## Live updates on Workers (shared board)

Design and measurements: [workers-realtime plan](.plans/2026-09-14_0754_workers-realtime-d1-do.md).

```
browser ─ hx-ws:connect /live/{topic} ─▶ index.mjs ─▶ Room Durable Object (room.mjs, hibernatable WebSockets)
browser ─ hx-post /board/add|note ─────▶ index.mjs ─▶ Go Worker (TinyGo)
                                           1. D1: INSERT … ON CONFLICT DO UPDATE … RETURNING version
                                           2. ROOM stub: POST /publish <#board fragment>
                                                 └▶ Room: ws.send(fragment) to every tab (≤ 5 broadcasts/s, newest wins)
```

- **D1 is the source of truth; the Room only fans out.** Its cached last fragment is sent to (re)connecting tabs.
- **Every fragment is the whole `#board` with a `data-version`;** `board.html` cancels any `htmx:before:swap` older than
  what's on screen, so HTTP responses and pushes can arrive in any order.
- **Designed for 1,000 tabs per topic:** receive-only sockets, `setWebSocketAutoResponse` ping/pong, coalesced broadcasts,
  reconnects spread over 1–3 s (a deploy drops every socket at once). Measured live: 1,000/1,000 delivered, p50 386 ms
  from the click on a phone hotspot.
- **JS only where Go can't go:** `index.mjs` (workers-go has no WebSocket support) and `room.mjs` (workers-go can only call
  Durable Objects). Locally, `workerd/local-d1.mjs` gives Go a D1-shaped `DB` over Durable Object SQLite, so the same
  `database/sql` code runs on workerd without miniflare.

## Project layout

```
.
├── mise.toml            # pinned tools + `check`; includes tasks/*.toml
├── tasks/
│   ├── gsxui.toml       # demo:gsxui:*, demo:gsxui:workers:*, upstream:gsxui:*
│   ├── workers.toml     # demo:workers:*, upstream:workers-go:*, upstream:cf:*
│   └── workerd/         # workerd configs for the upstream references
├── demos/
│   ├── gsxui/           # gsx + gsxui + htmx 4 (native server + Workers), own Go module
│   └── workers/         # htmx 4 on Workers + shared board (D1, Durable Objects), own Go module
└── .plans/              # timestamped plans with findings
```
