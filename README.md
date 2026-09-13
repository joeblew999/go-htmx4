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

No Node anywhere.

### Planned

See [`.plans/`](.plans/) for details.

| Layer | Technology | Plan |
| --- | --- | --- |
| Hosting | [Cloudflare Workers](https://workers.cloudflare.com) via [workers-go](https://github.com/syumai/workers-go) (TinyGo wasm) | [adopt-workers-go](.plans/2026-09-13_1111_adopt-workers-go.md) |
| State | [Cloudflare D1](https://developers.cloudflare.com/d1/) | [adopt-workers-go](.plans/2026-09-13_1111_adopt-workers-go.md) |

## Requirements

- Go 1.27 or newer, or [mise](https://mise.jdx.dev), which installs the pinned toolchain for you

## Quick start with mise

```sh
git clone https://github.com/joeblew999/go-htmx4.git
cd go-htmx4
mise install        # Go, TinyGo, binaryen, watchexec, as pinned in mise.toml
mise run dev        # run and auto-restart on changes
```

Other tasks: `mise run run`, `build`, `check`, `fmt`, `htmx:update <version>` — see `mise tasks`.
Set `ADDR=:3000` to change the listen address.

## Demos

| Demo | Run | URL |
| --- | --- | --- |
| **Our gsxui demo** ([`demos/gsxui`](demos/gsxui)): gsx + gsxui + htmx 4 + hx-live. Form with an OOB toast, dialog and tabs loaded via `hx-get`, boosted nav with `outerMorph`, hx-live counter and filter. | `mise run demo:gsxui:run`<br>`mise run demo:gsxui:dev` (live rebuild)<br>`mise run demo:gsxui:test` | http://localhost:7777 |
| **gsxui's own demo** (showcase site, all component examples, theme editor), served by its Go harness from a gitignored checkout in `.upstream/gsxui` | `mise run upstream:gsxui:serve` | http://127.0.0.1:7799 |

Both use the npm-free tooling: `go tool gsx`, `gsxui`, and the standalone `tailwindcss` binary.

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
