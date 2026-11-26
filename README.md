# go-htmx4

[![ci](https://github.com/joeblew999/go-htmx4/actions/workflows/ci.yml/badge.svg)](https://github.com/joeblew999/go-htmx4/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/joeblew999/go-htmx4)](go.mod)
[![license](https://img.shields.io/github/license/joeblew999/go-htmx4)](LICENSE)
[![performance](docs/badges/lighthouse-performance.svg)](#lighthouse)
[![accessibility](docs/badges/lighthouse-accessibility.svg)](#lighthouse)
[![best practices](docs/badges/lighthouse-best-practices.svg)](#lighthouse)
[![seo](docs/badges/lighthouse-seo.svg)](#lighthouse)
[![seo audit](docs/badges/seo-audit.svg)](#lighthouse)
[![languages](docs/badges/intl.svg)](locales)

A server-rendered Go web app on Cloudflare Workers: [gsx](https://gsxhq.dev) and [gsxui](https://ui.gsxhq.dev) for
the pages, [htmx 4](https://four.htmx.org) for what is interactive, [Vite](https://vite.dev) for the browser's files,
[wrangler](https://developers.cloudflare.com/workers/wrangler/) for the Worker, and
[charter](https://github.com/joeblew999/charter) to drive them and to add languages, search engines and Go runtimes
kept between requests.

- **Pages** in gsx from gsxui components: forms with out-of-band toasts, a dialog, tabs, client state with hx-live,
  boosted navigation, light/dark theme, 14 languages including right-to-left.
- **A live shared board** (`/board`): Go writes to D1, a Durable Object per topic pushes every change to every open
  tab over WebSockets, with presence.
- **Live video** (`/live`) from a browser camera over Media over QUIC through Cloudflare's relays, and **a map**
  (`/map`) on Leaflet and OpenStreetMap.
- **Search-ready:** canonical URLs, `hreflang`, a sitemap and Open Graph on every page, audited as Googlebot reads it.

**Live:** https://go-htmx4.ubuntusoftware.net (the board: [/board](https://go-htmx4.ubuntusoftware.net/board)).

## Quick start

You need [Go](https://go.dev/dl/) and [Node](https://nodejs.org). Nothing else: `go.mod` and `package-lock.json` pin
every tool, on macOS, Linux and Windows.

```sh
git clone https://github.com/joeblew999/go-htmx4.git && cd go-htmx4
npm install
go tool charter dev        # work on it: a save rebuilds and reloads
go tool charter check      # before a push: format, vet, lint, languages
go tool charter test       # go test, then the Worker on wrangler dev and the search-engine audit
go tool charter test e2e   # the browser tests (Chrome)
```

| Command | What it does |
|---|---|
| `go tool charter dev` | gsx's dev loop on a Go server, with Vite: a save regenerates, rebuilds and reloads the browser |
| `go tool charter preview` | The Worker as Cloudflare runs it, on wrangler dev → http://localhost:8913 |
| `go tool charter check` | What can be seen without running it: format, vet, lint, languages |
| `go tool charter test` | `go test`, the Worker on wrangler dev, the search-engine audit; `--env staging` asks the deployed one |
| `go tool charter test e2e` | The browser tests against wrangler dev; `--base https://…` against a deployed site |
| `go tool charter test load` | 1,000 WebSockets on one topic |
| `go tool charter deploy staging` | The D1 migrations, then wrangler deploy. `production` likewise |
| `go tool charter run languages` | The languages: what is translated, what is left. `go tool charter run translate` translates it |
| `go tool charter run lighthouse` | Lighthouse on the live site |

`go tool charter` lists the rest. **VS Code:** install the recommended
[gsx extension](https://marketplace.visualstudio.com/items?itemName=gsxhq.gsx).

## Where things are

| Path | What it is |
| --- | --- |
| `main.go` | the entry point: `transport.Run(app.Handler(dist))` |
| `app/` | the server: routes, the board and its stores, the site's languages and search engines (`site.go`), what it takes from Cloudflare (`platform.go`) |
| `views/` | pages and fragments in gsx, and the page tree (`pages.go`); `views/intl/`, the language screens; `views/blocks/`, the map, the places picker and live video |
| `ui/`, `web/gsxui/` | gsxui's components and their behaviours, written by `gsxui add`: never edited by hand |
| `web/` | the browser's files, which Vite builds: `main.js` (every page), `map.js`, `moq.js` |
| `public/` | files served as they are (the icon) |
| `locales/` | the messages, one catalog per language (`en.toml` is the source) |
| `migrations/` | D1's schema |
| `worker/` | the two JavaScript files Cloudflare needs: the entry, and the board's Durable Object |
| `e2e/` | the checks against a running Worker: `smoke/` over HTTP, the browser tests, `load/` |
| `internal/` | the board's message rules (`live`), Nominatim (`places`), gsxui's class merger |
| `patches/` | two fixes to @moq packages, which npm install applies |

The settings, one file per tool:

| File | Whose |
|---|---|
| `wrangler.jsonc` | The Worker: its name, D1, the Durable Object, the rate limit, variables, staging and production |
| `charter.toml` | What wrangler has no place for: the tests, the languages (`[intl]`), search engines (`[seo]`) |
| `gsx.toml`, `gsxui.json` | gsx's and gsxui's |
| `vite.config.ts`, `package.json` | Vite's, and what npm installs |

`build/` and `dist/` are build output, never committed.

**Adding something, one way each:**

- **A page:** a type in the page tree (`views/pages.go`: a field in `Pages` with its route, an entry in `All`, its
  data through `Props`), its `Page` method (`views/pages.gsx`), and a component in `views/<page>.gsx` inside `Layout`
  with its own translated title and description. Links to it are `{ About{} |> url }`.
- **Text:** a key in `locales/en.toml`, written `{ locales.Key }`, then `go tool charter run translate`.
- **A date, a number, any Intl call:** a filter in markup, `{ t |> date("dateStyle", "long") }`, its options in pairs.
- **A language:** `go tool charter add locale:de`, then translate.
- **A UI component:** `go tool gsxui add <name>`; the build then names any text it writes that needs a message.
- **Browser code:** in `web/`, imported from `web/main.js`; an npm package is `npm install <name>`.
- **A binding or a variable:** `wrangler.jsonc`.

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

Lighthouse mobile on Google's servers (PageSpeed Insights), 2026-11-25: `go tool charter run badges` writes this table and the badges.

<!-- /seo scores -->

How it works: [docs/architecture.md](docs/architecture.md). How it ships: [docs/deploy.md](docs/deploy.md). Rules for
changing it: [AGENTS.md](AGENTS.md). Open work: [GitHub issues](https://github.com/joeblew999/go-htmx4/issues).
Licensed under [MIT](LICENSE).
