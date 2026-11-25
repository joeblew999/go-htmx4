# go-htmx4

[![check](https://github.com/joeblew999/go-htmx4/actions/workflows/charter.yml/badge.svg)](https://github.com/joeblew999/go-htmx4/actions/workflows/charter.yml)
[![release](https://img.shields.io/github/v/release/joeblew999/go-htmx4)](https://github.com/joeblew999/go-htmx4/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/joeblew999/go-htmx4)](go.mod)
[![license](https://img.shields.io/github/license/joeblew999/go-htmx4)](LICENSE)
[![intl: Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/mise-intl.svg)](https://pkg.go.dev/github.com/joeblew999/mise-intl)
[![seo: Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/mise-seo.svg)](https://pkg.go.dev/github.com/joeblew999/mise-seo)

[![performance](docs/badges/lighthouse-performance.svg)](#lighthouse)
[![accessibility](docs/badges/lighthouse-accessibility.svg)](#lighthouse)
[![best practices](docs/badges/lighthouse-best-practices.svg)](#lighthouse)
[![seo](docs/badges/lighthouse-seo.svg)](#lighthouse)
[![seo audit](docs/badges/seo-audit.svg)](https://github.com/joeblew999/mise-seo/blob/main/templates/docs/seo/README.md)
[![languages](docs/badges/intl.svg)](https://github.com/joeblew999/mise-intl/blob/main/templates/docs/intl/README.md)

**The problem.** A server-rendered Go web app on Cloudflare Workers takes many pieces, each found and fitted by hand:
Go built small and fast enough for Workers, type-checked templates and real UI components, htmx for what's
interactive, a change pushed live to every open tab, every language at its own URLs, what Google sees, tests on the
runtime Workers has, previews and deploys. Each is a project of its own, the usual way to most of them brings a Node
toolchain with it, and keeping them working together as each one moves is most of the work.

**What go-htmx4 does.** It is such an app, with [Go](https://go.dev), [htmx 4](https://four.htmx.org) and
[gsxui](https://ui.gsxhq.dev) on Cloudflare Workers, and no Node anywhere: no npm, no Vite, no wrangler. Every piece works together, live, and
checked on every change:

- **Pages** in gsx from gsxui components: forms with out-of-band toasts, a dialog, tabs, client state with hx-live,
  boosted navigation, light/dark theme, 14 languages including right-to-left.
- **A live shared board** (`/board`): Go writes to D1, a Durable Object per topic pushes every change to every open
  tab over WebSockets, with presence.
- **Live video** (`/live`) from a browser camera over Media over QUIC through Cloudflare's relays, and **a map**
  (`/map`) on Leaflet and OpenStreetMap, as gsx components of the app's own (`views/blocks/`).
- **Search-ready:** canonical URLs, `hreflang`, a sitemap and Open Graph on every page, audited live as Googlebot.

**Live:** https://go-htmx4.ubuntusoftware.net (the board: [/board](https://go-htmx4.ubuntusoftware.net/board)).

## Lighthouse

Every page of the live site, scored by Lighthouse; each badge above is the lowest page's score.

<!-- seo scores -->

| Page | Performance | Accessibility | Best practices | SEO |
|---|---:|---:|---:|---:|
| [/](https://go-htmx4.ubuntusoftware.net/) | 99 | 100 | 100 | 100 |
| [/board](https://go-htmx4.ubuntusoftware.net/board) | 100 | 100 | 96 | 100 |
| [/formats](https://go-htmx4.ubuntusoftware.net/formats) | 99 | 100 | 100 | 100 |
| [/components](https://go-htmx4.ubuntusoftware.net/components) | 100 | 97 | 100 | 100 |
| [/live](https://go-htmx4.ubuntusoftware.net/live) | 100 | 100 | 77 | 100 |
| [/map](https://go-htmx4.ubuntusoftware.net/map) | 90 | 100 | 96 | 100 |
| [/about](https://go-htmx4.ubuntusoftware.net/about) | 99 | 100 | 100 | 100 |

Lighthouse mobile on Google's servers (PageSpeed Insights), 2026-11-25: `mise run seo:badges` writes this table and the badges.

<!-- /seo scores -->

## What it is built on

The app is only the app. What it stands on are projects of their own, each pinned in `go.mod` and `mise.toml`:

| | What it gives | Where |
|---|---|---|
| **Languages** | Messages in ICU catalogs; dates, numbers, plurals and names from V8's Intl, on the server and in the reader's browser; Claude translating what's new | [mise-intl](https://github.com/joeblew999/mise-intl) ([mise-intl's guide](https://github.com/joeblew999/mise-intl/blob/main/templates/docs/intl/README.md)) |
| **Search engines** | robots.txt, the sitemap, canonical addresses; an audit of every page as Googlebot reads it; Lighthouse | [mise-seo](https://github.com/joeblew999/mise-seo) ([mise-seo's guide](https://github.com/joeblew999/mise-seo/blob/main/templates/docs/seo/README.md)) |
| **Cloudflare** | The build, local workerd, deploys, previews, CI, and the bindings (D1, Durable Objects, the rate limit) with stand-ins off Workers, so `go test` runs the real SQL | [charter](https://github.com/joeblew999/charter) |
| **Pages and components** | Type-checked templates, UI components, routes as a tree of Go types | [gsx](https://gsxhq.github.io), [gsxui](https://ui.gsxhq.dev), [structpages](https://github.com/jackielii/structpages) |

## Quick start

```sh
git clone https://github.com/joeblew999/go-htmx4.git && cd go-htmx4
mise install        # every pinned tool, and gsx for the editor
mise run dev        # rebuild on every save → http://localhost:8913
mise run check      # the production build under workerd, and go test
mise run e2e        # the browser tests
```

**Every command, and which to use when:** [docs/commands.md](docs/commands.md).

**VS Code:** install the recommended [gsx extension](https://marketplace.visualstudio.com/items?itemName=gsxhq.gsx).
`mise install` puts go.mod's gsx where its language server looks; then run **gsx: Restart Language Server**. Don't
use the extension's *Install gsx* prompt: it installs `@latest`, not the pinned version.

## Where things are

| Path | What it is |
| --- | --- |
| `main.go` | the entry point: `transport.Run(app.Handler())` |
| `app/` | the server: routes, the board and its stores, the site's languages and search engines (`site.go`), what it takes from Cloudflare (`platform.go`) |
| `views/` | pages and fragments in gsx, and the page tree (`pages.go`); `views/intl/`, mise-intl's screens; `views/blocks/`, the map, the places picker and live video |
| `ui/`, `web/gsxui/` | gsxui's components and their behaviours, written by `gsxui add`: never edited by hand |
| `web/` | what the browser loads: htmx (`mise run vendor`), the map's and the video's bundles (`mise run js:bundle`) |
| `locales/`, `intl.toml` | the messages, one catalog per language (`en.toml` is the source), and the languages |
| `migrations/` | D1's schema |
| `worker/`, `charter.toml` | the two JavaScript files Cloudflare needs (the entry, the board's Durable Object), and everything about the Worker on Cloudflare |
| `e2e/` | the checks against a running server: `smoke/` over HTTP, the browser tests, `load/` with 1,000 WebSockets |
| `internal/` | the board's message rules (`live`), Nominatim (`places`), and the repo's own tools: `deps` (pins), `jsbundle` (npm packages bundled without Node) |
| `tasks/`, `ci/`, `mise.toml` | the commands, the scripts several of them share, the pinned tools |
| `docs/` | how it works, how it ships, every command |

`build/` and `dist/` are build output, never committed.

**Adding something, one way each:**

- **A page:** a type in the page tree (`views/pages.go`: a field in `Pages` with its route, an entry in `All`, its
  data through `Props`), its `Page` method (`views/pages.gsx`), and a component in `views/<page>.gsx` inside `Layout`
  with its own translated title and description. Links to it are `{ About{} |> url }`.
- **Text:** a key in `locales/en.toml`, written `{ locales.Key }`, then `mise run intl:translate`.
- **A date, a number, any Intl call:** a filter in markup, `{ t |> date("dateStyle", "long") }`, its options in pairs.
- **A language:** its id in `intl.toml`'s `locales`, then `mise run intl:gen` and `mise run intl:translate`.
- **A UI component:** `gsxui add <name>` at the repo root; `mise run intl:gen` then names any text it writes that
  needs a message.
- **A command:** a mise task in `tasks/` with a description (it lands in [docs/commands.md](docs/commands.md)).

## Stack

Versions are pinned in [`mise.toml`](mise.toml) and [`go.mod`](go.mod).

| Layer | Technology |
| --- | --- |
| Language | [Go](https://go.dev), compiled to WebAssembly by the standard compiler |
| Templates and UI | [gsx](https://gsxhq.github.io) (type-checked Go templates), [gsxui](https://ui.gsxhq.dev) components, the [Tailwind](https://tailwindcss.com/docs/installation/tailwind-cli) standalone CLI |
| Interactivity | [htmx 4](https://four.htmx.org) with hx-live and hx-ws, vendored in `web/htmx/` |
| Hosting | [Cloudflare Workers](https://workers.cloudflare.com): [charter](https://github.com/joeblew999/charter)'s Go runtime and tools over [workers-go](https://github.com/syumai/workers-go); [workerd](https://github.com/cloudflare/workerd) locally |
| Data and live updates | [D1](https://developers.cloudflare.com/d1/), [Durable Objects](https://developers.cloudflare.com/durable-objects/) with WebSocket Hibernation |
| Tools and secrets | [mise](https://mise.jdx.dev) (tools and tasks), [fnox](https://github.com/jdx/fnox) (credentials), [Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/) (e2e) |

How it works: [docs/architecture.md](docs/architecture.md). Rules for changing it: [AGENTS.md](AGENTS.md). Plans and
open work: [GitHub issues](https://github.com/joeblew999/go-htmx4/issues). Licensed under [MIT](LICENSE).
