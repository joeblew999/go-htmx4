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

## Stack

Versions are pinned in [`mise.toml`](mise.toml) and [`go.mod`](go.mod).

| Layer | Technology |
| --- | --- |
| Language | [Go](https://go.dev); [TinyGo](https://tinygo.org) for what ships to Workers, with [binaryen](https://github.com/WebAssembly/binaryen)'s `wasm-opt` |
| Templates and UI | [gsx](https://gsxhq.github.io) (type-checked Go templates), [gsxui](https://ui.gsxhq.dev) components, the [Tailwind](https://tailwindcss.com/docs/installation/tailwind-cli) standalone CLI |
| Interactivity | [htmx 4](https://four.htmx.org) with hx-live and hx-ws, vendored in `static/` |
| Hosting | [Cloudflare Workers](https://workers.cloudflare.com): [charter](https://github.com/joeblew999/charter)'s Go runtime and tools over [workers-go](https://github.com/syumai/workers-go); [workerd](https://github.com/cloudflare/workerd) locally |
| Data and live updates | [D1](https://developers.cloudflare.com/d1/), [Durable Objects](https://developers.cloudflare.com/durable-objects/) with WebSocket Hibernation |
| Tools and secrets | [mise](https://mise.jdx.dev) (tools and tasks), [fnox](https://github.com/jdx/fnox) (credentials), [Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/) (e2e) |

How it works: [docs/architecture.md](docs/architecture.md). Rules for changing it: [AGENTS.md](AGENTS.md). Plans and
open work: [GitHub issues](https://github.com/joeblew999/go-htmx4/issues). Licensed under [MIT](LICENSE).
