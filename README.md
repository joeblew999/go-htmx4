# go-htmx4

A minimal starter for building server-rendered web apps with [Go](https://go.dev) and [htmx 4](https://four.htmx.org).

- Standard library only — no external Go dependencies
- Uses Go 1.22+ `net/http` routing patterns (`GET /{$}`, `POST /count`)
- Templates and static assets are embedded with `go:embed`, so the result is a single self-contained binary
- htmx 4.0.0 is vendored into `static/`, so there's no CDN dependency at runtime

## Stack

### In use

| Layer | Technology | Version | Notes |
| --- | --- | --- | --- |
| Language / server | [Go](https://go.dev) `net/http`, `html/template`, `embed` | 1.27.1 | Standard library only |
| Interactivity | [htmx](https://four.htmx.org) | 4.0.0 | Vendored in `static/` |
| Toolchain & tasks | [mise](https://mise.jdx.dev) | — | Pins every tool below; `mise tasks` lists tasks |
| Dev reload | [watchexec](https://github.com/watchexec/watchexec) | latest | Used by `mise run dev` |
| Wasm compiler | [TinyGo](https://tinygo.org) | 0.42.0 | Small wasm builds (supports Go ≤ 1.27) |
| Wasm optimiser | [binaryen](https://github.com/WebAssembly/binaryen) `wasm-opt` | 132 | Required by TinyGo wasm targets |
| Templates (demo) | [gsx](https://gsxhq.github.io): JSX-style, type-checked Go templates | v0.1.0 | `go tool gsx` in `demos/gsxui` |
| UI components (demo) | [gsxui](https://ui.gsxhq.dev): shadcn/ui for gsx, npm-free mode | `c7fd6a8` | Vendored with `gsxui add` |
| Client state (demo) | [hx-live](https://four.htmx.org/extensions/hx-live/) | 4.0.0 | htmx 4 extension |
| CSS (demo) | [Tailwind CSS standalone CLI](https://tailwindcss.com/docs/installation/tailwind-cli) | 4.3.3 | Single binary, no npm |
| Hosting (demo) | [Cloudflare Workers](https://workers.cloudflare.com) via [workers-go](https://github.com/syumai/workers-go) | v0.35.0 | TinyGo wasm in `demos/workers` |
| Local Workers runtime (demo) | [workerd](https://github.com/cloudflare/workerd) | 1.20260911.1 | Standalone binary; replaces `wrangler dev` |
| Database (demo) | [Cloudflare D1](https://developers.cloudflare.com/d1/) via workers-go's `database/sql` driver | — | Source of truth for the shared board |
| Live updates (demo) | [Durable Objects](https://developers.cloudflare.com/durable-objects/) (WebSocket Hibernation) + htmx 4 [`hx-ws`](https://four.htmx.org/extensions/hx-ws) | 4.0.0 | One `Room` per topic pushes fragments to every tab |
| Secrets | [fnox](https://github.com/jdx/fnox) | 1.35.0 | Cloudflare token + account id from the keychain |

No Node anywhere: no npm, no Vite, no wrangler.

### Planned

See [`.plans/`](.plans/) for details.

| Layer | Technology | Plan |
| --- | --- | --- |
| — | Nothing planned right now | — |

## Requirements

- Go 1.27 or newer, or [mise](https://mise.jdx.dev), which installs the pinned toolchain for you

## Quick start with mise

```sh
git clone https://github.com/joeblew999/go-htmx4.git
cd go-htmx4
mise install        # every tool pinned in mise.toml (Go, TinyGo, workerd, …)
mise run dev        # run and auto-restart on changes
```

Other tasks: `mise run run`, `build`, `check`, `fmt`, `htmx:update <version>` — see `mise tasks`.
Set `ADDR=:3000` to change the listen address.

## Demos

| Demo | Run | URL |
| --- | --- | --- |
| **Our gsxui demo** ([`demos/gsxui`](demos/gsxui)): gsx + gsxui + htmx 4 + hx-live. Form with an OOB toast, dialog and tabs loaded via `hx-get`, boosted nav with `outerMorph`, hx-live counter and filter. | `mise run demo:gsxui:run`<br>`mise run demo:gsxui:dev` (live rebuild)<br>`mise run demo:gsxui:test` | http://localhost:7777 |
| **gsxui's own demo** (showcase site, all component examples, theme editor), served by its Go harness from a gitignored checkout in `.upstream/gsxui` | `mise run upstream:gsxui:serve` | http://127.0.0.1:7799 |
| **Our Workers demo** ([`demos/workers`](demos/workers)): htmx 4 on Cloudflare Workers via workers-go, built with TinyGo. Fragment round-trip, form post, a counter that resets on every request, static files served by Workers Static Assets, and a **shared board** at `/board`: D1 + a Durable Object per topic push every change to all open tabs over `hx-ws`. | `mise run demo:workers:serve` (workerd)<br>`mise run demo:workers:run` (`go run .`)<br>`mise run demo:workers:test`<br>`mise run demo:workers:load` (1,000 WebSockets)<br>`mise run demo:workers:deploy` | http://localhost:8913 (`/board`)<br>http://localhost:9913<br>live: https://go-htmx4-workers-demo.gedw99.workers.dev/board |
| **Our gsxui demo on Workers**: the same `demos/gsxui` app, built with TinyGo (`platform_js.go`), static files as Workers Static Assets. HTML is byte-identical to the native server. | `mise run demo:gsxui:workers:serve`<br>`mise run demo:gsxui:workers:smoke`<br>`mise run demo:gsxui:workers:deploy` | http://localhost:8918<br>live: https://go-htmx4-gsxui-demo.gedw99.workers.dev |
| **workers-go's Durable Object example** and **Cloudflare's WebSocket Hibernation example**, on plain workerd | `mise run upstream:workers-go:do`<br>`mise run upstream:cf:ws-hibernation` | http://localhost:8914<br>ws://localhost:8915/ws |
| **workers-go's own template** (`worker-tinygo`) and `_examples/env`, from a gitignored checkout in `.upstream/workers-go` | `mise run upstream:workers-go:serve`<br>`mise run upstream:workers-go:env` | http://localhost:8911<br>http://localhost:8912 |

The gsxui demos use the npm-free tooling: `go tool gsx`, `gsxui`, and the standalone `tailwindcss` binary.

### Cloudflare Workers without Node

workers-go's docs use `npm create cloudflare` and wrangler. `tasks/workers.toml` does the same steps without them:

- **Build:** `workers-assets-gen -mode=tinygo` + `tinygo build -target wasm` (the template's own build script).
- **Run locally:** `workerd serve config.capnp`, the runtime `wrangler dev` uses. `workerd/assets-first.mjs` stands in for
  Static Assets (it runs inside workerd, not Node).
- **Deploy:** [`demos/workers/cmd/deploy`](demos/workers/cmd/deploy), a stdlib Go client for Cloudflare's
  [Static Assets Direct Upload](https://developers.cloudflare.com/workers/static-assets/direct-upload/) and script upload
  APIs. Credentials come from fnox (`fnox exec -- …`).

### Live updates on Workers (shared board)

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

TinyGo 0.42 caveats (details in the [plan](.plans/2026-09-13_1111_adopt-workers-go.md)): its `net/http` has the pre-Go 1.22
`ServeMux` (no `GET /path` patterns), and `html/template` compiles but panics at runtime. `go test` runs on standard Go and
can't see either, so `demo:workers:test` also curls the real TinyGo build under workerd.

## Quick start

```sh
git clone https://github.com/joeblew999/go-htmx4.git
cd go-htmx4
go run .
```

Open <http://localhost:8080>.

To use a different address:

```sh
go run . -addr :3000
```

## Build

```sh
go build -o go-htmx4 .
./go-htmx4
```

## Project layout

```
.
├── main.go              # HTTP server, routes and handlers
├── templates/
│   └── index.html       # Page template + named fragment templates
└── static/
    └── htmx.min.js      # Vendored htmx 4.0.0
```

## How it works

The page is rendered in full by `GET /`. htmx attributes on elements then call
small endpoints that return **HTML fragments**, which htmx swaps into the page:

| Route         | Returns                   | Used by                                             |
| ------------- | ------------------------- | --------------------------------------------------- |
| `GET /`       | Full page (`index.html`)  | Browser                                             |
| `POST /count` | `count` fragment          | `<button hx-post="/count" hx-target="#count">`      |
| `GET /time`   | `time` fragment           | `<p hx-get="/time" hx-trigger="load, every 1s">`    |
| `GET /static/`| Embedded static files     | `<script src="/static/htmx.min.js">`                |

Fragments are declared with `{{define "name"}}` in the templates and rendered
with `tmpl.ExecuteTemplate`, so the full page and the partial updates share the
same markup.

## Adding a feature

1. Add a named fragment to `templates/index.html` (or a new file in `templates/`):
   ```html
   {{define "greeting"}}<p>Hello, {{.}}!</p>{{end}}
   ```
2. Add a handler in `main.go`:
   ```go
   mux.HandleFunc("POST /greet", func(w http.ResponseWriter, r *http.Request) {
       render(w, "greeting", r.FormValue("name"))
   })
   ```
3. Wire it up in the page:
   ```html
   <form hx-post="/greet" hx-target="#greeting">
     <input name="name" placeholder="Your name">
     <button>Greet</button>
   </form>
   <div id="greeting"></div>
   ```

## Upgrading htmx

```sh
curl -sfL -o static/htmx.min.js https://cdn.jsdelivr.net/npm/htmx.org@<version>/dist/htmx.min.js
```

See the [htmx 2 → 4 migration notes](https://four.htmx.org/migration-guide-htmx-4/) if you're coming from htmx 2.
