# go-htmx4

[![check](https://github.com/joeblew999/go-htmx4/actions/workflows/check.yml/badge.svg)](https://github.com/joeblew999/go-htmx4/actions/workflows/check.yml)

Server-rendered web apps with [Go](https://go.dev), [htmx 4](https://four.htmx.org) and [gsxui](https://ui.gsxhq.dev) on
Cloudflare Workers, compiled with TinyGo. No Node anywhere: no npm, no Vite, no wrangler.

- **Pages** in gsx from gsxui components: forms with out-of-band toasts, a dialog, tabs, client state with hx-live,
  boosted navigation, light/dark theme, 14 languages including right-to-left.
- **A live shared board** (`/board`): Go writes to D1, a Durable Object per topic pushes every change to every open
  tab over WebSockets, with presence.
- **Search-ready:** canonical URLs, `hreflang`, a sitemap and Open Graph on every page, audited live as Googlebot.

**Live:** https://go-htmx4.ubuntusoftware.net (the board: [/board](https://go-htmx4.ubuntusoftware.net/board)).

## Quick start

```sh
git clone https://github.com/joeblew999/go-htmx4.git && cd go-htmx4
mise install        # every pinned tool, and gsx for the editor
mise run dev        # rebuild on every save → http://localhost:8913
mise run check      # the production build under workerd, and go test
```

**Every command, and which to use when:** [docs/commands.md](docs/commands.md).

**VS Code:** install the recommended [gsx extension](https://marketplace.visualstudio.com/items?itemName=gsxhq.gsx).
`mise install` puts go.mod's gsx where its language server looks; then run **gsx: Restart Language Server**. Don't
use the extension's *Install gsx* prompt: it installs `@latest`, not the pinned version.

## Use it as a template

```sh
mise run rename -- github.com/you/my-app my-app   # module path and app name (Worker, D1, brand)
mise run check
```

To deploy, put a Cloudflare API token (Workers Scripts, D1 and Workers Tail edit) and your account id in
[fnox](https://github.com/jdx/fnox) as `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`, then `mise run deploy`.
Every branch gets its own preview, `main` deploys staging, and a release deploys production: [docs/deploy.md](docs/deploy.md).

## Where things are

| Path | What it is |
| --- | --- |
| `main.go` | the entry point: `transport.Run(app.Handler())` |
| `app/` | the server (`server.go`: routes), the board (`board.go`, its stores), locale URLs (`locale.go`), robots.txt and the sitemap (`seo.go`), Cloudflare-only calls (`platform_js.go`) and their twins; each file's tests beside it |
| `views/` | pages and fragments in gsx; a page's data in a `.go` file beside it |
| `ui/`, `web/gsxui/` | gsxui components and their behaviours, vendored by `gsxui add`: never edited by hand |
| `web/htmx/` | htmx 4, hx-ws and hx-live, vendored by `mise run vendor` |
| `locales/` | the messages: one catalog per language (`en.toml` is the source) and `intl.sum` |
| `intl/`, `intl.toml` | Intl for gsx apps, its own Go module ([intl/README.md](intl/README.md)); this app's languages |
| `internal/` | httpx (rendering, origin), live (publishing to the Room), ratelimit, classmerge (gsx's class merger, cached) |
| `worker/` | the deployed JavaScript: the Worker's entry and the board's Durable Object; `worker/local/` runs them under workerd |
| `migrations/` | D1's schema |
| `e2e/` | the checks against a running server, its own Go module: `smoke/` over HTTP (`mise run smoke`), browser tests (`mise run e2e`), `load/` with 1,000 WebSockets (`mise run load`) |
| `ci/`, `tasks/`, `mise.toml` | the scripts CI runs or several tasks share, the mise tasks, the pinned tools |
| `docs/` | how it works, how it ships, every command |

`build/` and `dist/` are build output, never committed.

**Adding something, one way each:**

- **A page:** a component in `views/<page>.gsx` inside `Layout`, with its own translated title and description; a
  route in `app/server.go` that renders it with `s.render`; its path in `views.SitemapPages`.
- **Text:** a key in `locales/en.toml`, written `{ locales.Key }`, then `mise run intl:translate` and
  `mise run intl:check`.
- **A date, a number, any Intl call:** a filter in markup, `{ t |> date("dateStyle", "long") }`, its options in pairs.
- **A language:** its id in `intl.toml`'s `locales`, then `mise run intl` and `mise run intl:translate`.
- **A UI component:** `gsxui add <name>` at the repo root; `mise run intl` then names any text it writes that needs a
  message.
- **A command:** a mise task in `tasks/` with a description (it lands in [docs/commands.md](docs/commands.md)); its
  shell in `ci/`.

## Stack

Versions are pinned in [`mise.toml`](mise.toml) and [`go.mod`](go.mod).

| Layer | Technology |
| --- | --- |
| Language | [Go](https://go.dev); [TinyGo](https://tinygo.org) for what ships to Workers, with [binaryen](https://github.com/WebAssembly/binaryen)'s `wasm-opt` |
| Templates and UI | [gsx](https://gsxhq.github.io) (type-checked Go templates), [gsxui](https://ui.gsxhq.dev) components, the [Tailwind](https://tailwindcss.com/docs/installation/tailwind-cli) standalone CLI |
| Interactivity | [htmx 4](https://four.htmx.org) with hx-live and hx-ws, vendored in `web/htmx/` |
| Hosting | [Cloudflare Workers](https://workers.cloudflare.com): [charter](https://github.com/joeblew999/charter)'s Go runtime and tools over [workers-go](https://github.com/syumai/workers-go); [workerd](https://github.com/cloudflare/workerd) locally |
| Data and live updates | [D1](https://developers.cloudflare.com/d1/), [Durable Objects](https://developers.cloudflare.com/durable-objects/) with WebSocket Hibernation |
| Tools and secrets | [mise](https://mise.jdx.dev) (tools and tasks), [fnox](https://github.com/jdx/fnox) (credentials), [Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/) (e2e) |

How it works: [docs/architecture.md](docs/architecture.md). Rules for changing it: [AGENTS.md](AGENTS.md). Plans and
open work: [GitHub issues](https://github.com/joeblew999/go-htmx4/issues). Licensed under [MIT](LICENSE).
