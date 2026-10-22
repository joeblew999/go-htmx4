# go-htmx4

[![check](https://github.com/joeblew999/go-htmx4/actions/workflows/charter.yml/badge.svg)](https://github.com/joeblew999/go-htmx4/actions/workflows/charter.yml)
[![release](https://img.shields.io/github/v/release/joeblew999/go-htmx4)](https://github.com/joeblew999/go-htmx4/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/joeblew999/go-htmx4)](go.mod)
[![license](https://img.shields.io/github/license/joeblew999/go-htmx4)](LICENSE)
[![intl: Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/go-htmx4/intl.svg)](https://pkg.go.dev/github.com/joeblew999/go-htmx4/intl)
[![seo: Go Reference](https://pkg.go.dev/badge/github.com/joeblew999/go-htmx4/seo.svg)](https://pkg.go.dev/github.com/joeblew999/go-htmx4/seo)

[![performance](docs/badges/lighthouse-performance.svg)](#lighthouse)
[![accessibility](docs/badges/lighthouse-accessibility.svg)](#lighthouse)
[![best practices](docs/badges/lighthouse-best-practices.svg)](#lighthouse)
[![seo](docs/badges/lighthouse-seo.svg)](#lighthouse)
[![seo audit](docs/badges/seo-audit.svg)](seo/README.md)
[![languages](docs/badges/intl.svg)](intl/README.md)

Server-rendered web apps with [Go](https://go.dev), [htmx 4](https://four.htmx.org) and [gsxui](https://ui.gsxhq.dev) on
Cloudflare Workers, compiled with TinyGo. No Node anywhere: no npm, no Vite, no wrangler.

- **Pages** in gsx from gsxui components: forms with out-of-band toasts, a dialog, tabs, client state with hx-live,
  boosted navigation, light/dark theme, 14 languages including right-to-left.
- **A live shared board** (`/board`): Go writes to D1, a Durable Object per topic pushes every change to every open
  tab over WebSockets, with presence.
- **Search-ready:** canonical URLs, `hreflang`, a sitemap and Open Graph on every page, audited live as Googlebot.

**Live:** https://go-htmx4.ubuntusoftware.net (the board: [/board](https://go-htmx4.ubuntusoftware.net/board)).

## Lighthouse

Every page of the live site, scored by Lighthouse; each badge above is the lowest page's score.

<!-- seo:scores -->

| Page | Performance | Accessibility | Best practices | SEO |
|---|---:|---:|---:|---:|
| [/](https://go-htmx4.ubuntusoftware.net/) | 97 | 100 | 100 | 100 |
| [/board](https://go-htmx4.ubuntusoftware.net/board) | 99 | 100 | 96 | 100 |
| [/formats](https://go-htmx4.ubuntusoftware.net/formats) | 95 | 100 | 100 | 100 |
| [/components](https://go-htmx4.ubuntusoftware.net/components) | 95 | 95 | 100 | 100 |
| [/about](https://go-htmx4.ubuntusoftware.net/about) | 98 | 100 | 100 | 100 |

Lighthouse mobile on Google's servers (PageSpeed Insights), 2026-10-22: `mise run seo:badges` writes this table and the badges.

<!-- /seo:scores -->

## Two tools that come with it

The app is built on two Go modules from this repository, which any gsx app can use as well, on Cloudflare Workers or
an ordinary Go server:

- **[intl](intl/README.md): every language, done properly.** Text lives in ICU message catalogs; dates, numbers,
  plurals, lists and names come from V8's Intl (a Worker's own, or workerd's beside an ordinary Go server), on the
  server and again in the reader's browser, in their time zone, right-to-left included. `mise run intl:translate` has Claude translate every new message into
  every language, in parallel, in about two minutes; `mise run intl:check` keeps every catalog complete. Each language
  gets its own URLs, `hreflang` links and sitemap entries, so Google indexes every language version and shows it to
  the people who read that language. Its screens (a language list, the offer of a page in the reader's language, a
  time zone and clock dialog) come with it, already translated: `go tool intl add`.
- **[seo](seo/README.md): what Google sees, checked.** One middleware answers robots.txt and the sitemap, keeps what
  isn't a page out of the index, and gives every page one https address. Every `mise run check` then fetches every
  page as Googlebot does and fails on anything that would keep a page out or look wrong in a result. After a deploy,
  `mise run seo:lighthouse` scores every page on Google's servers (performance, accessibility, best practices, SEO,
  with the failing elements named), and `mise run seo:todo` reports what Google has indexed.

Each is set up by one command (`go tool intl init`, `go tool seo init`), which writes a config file into the project
(`intl.toml`, `seo.toml`). Their mise tasks read it, so every task runs with no arguments, from a terminal or the VS
Code mise panel, and an argument still overrides it for one run. Each also installs an agent skill, so Claude Code
knows how to use it. Both use gsx's own mechanisms (renderers, filters, `gsx fmt` and its checks); issues and pull
requests that make them, or gsx, better are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) says how; you can leave the 13 translations
of any text you add to us.

On Cloudflare Workers, [charter](https://github.com/joeblew999/charter)'s Go side is the platform, the same way:
- **Setup:** `go tool charter init` writes `charter.toml`, the Worker's every fact.
- **Running and shipping:** `go tool charter` builds it with TinyGo, runs it on local workerd, deploys it, and serves CI and Workers Builds.
- **Bindings:** they come through its packages (`d1`, `bindings`, `hub`), which stand in for themselves off Workers, so `go test` runs the real SQL.

`e2e/fresh`'s `TestFreshWorker` makes such an app from nothing.

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

## Start your own app

Don't copy this repo: start from nothing and add the pieces, each with one command. It is what the fresh-app tests
(`e2e/fresh`) do on every check, so it keeps working.

```sh
mkdir my-app && cd my-app && go mod init github.com/you/my-app
go get -tool github.com/gsxhq/gsx/cmd/gsx github.com/joeblew999/charter/go/lib/cmd/charter \
  github.com/joeblew999/go-htmx4/intl/cmd/intl github.com/joeblew999/go-htmx4/seo/cmd/seo \
  github.com/joeblew999/go-htmx4/gsxkit/cmd/gsxkit
gsxui init                        # gsxui's theme and components, npm-free (https://ui.gsxhq.dev)
go tool gsxkit tasks && go tool gsxkit skill   # gsx's and gsxui's tasks, and the skills gsx and structpages ship
go tool charter init              # charter.toml: the Worker on Cloudflare, local workerd, deploys
go tool intl init -locales en,de  # intl.toml, the catalogs, intl's tasks and skill
go tool intl add language-list language-offer preferences   # intl's screens, already translated
go tool seo init -base https://my-app.example.com          # seo.toml
```

Then [intl/README.md](intl/README.md), [seo/README.md](seo/README.md) and charter's
[libraries page](https://github.com/joeblew999/charter/blob/main/docs/self/libraries.md) take it from there. To deploy,
put a Cloudflare API token (Workers Scripts, D1 and Workers Tail edit) and your account id in the environment or in
[fnox](https://github.com/jdx/fnox) as `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`, then `mise run
charter:deploy production`. This repo's own previews, staging and releases: [docs/deploy.md](docs/deploy.md).

## Where things are

| Path | What it is |
| --- | --- |
| `main.go` | the entry point: `transport.Run(app.Handler())` |
| `app/` | the server (`server.go`: routes), the board (`board.go`, its stores), the app on the web (`site.go`: its languages, `intl.Site`, and search engines, `seo.Site`, with the sitemap's pages), what it takes from Cloudflare (`platform.go`, through charter's bindings, the same off Workers); each file's tests beside it |
| `views/` | pages and fragments in gsx; a page's data in a `.go` file beside it; `views/intl/`, intl's screens, written by `go tool intl add` |
| `ui/`, `web/gsxui/` | gsxui components and their behaviours, vendored by `gsxui add`: never edited by hand |
| `web/htmx/` | htmx 4, hx-ws and hx-live, vendored by `mise run vendor` |
| `locales/` | the messages: one catalog per language (`en.toml` is the source) and `intl.sum` |
| `intl/`, `intl.toml` | Intl for gsx apps, its own Go module ([intl/README.md](intl/README.md)); this app's languages |
| `seo/`, `seo.toml` | search engines for Go apps, its own Go module ([seo/README.md](seo/README.md)): the middleware, the audit, Lighthouse, Search Console; this app's settings |
| `gsxkit/` | gsx's and gsxui's mise tasks and the skills gsx and structpages ship, for any app, its own Go module ([gsxkit/README.md](gsxkit/README.md)) |
| `badge/` | the README badges intl and seo draw (SVG, no service), a tiny module of its own so neither depends on the other |
| `internal/` | live (the board's message to the Room, topic and locale rules), classmerge (gsx's class merger, cached) |
| `worker/`, `charter.toml` | the deployed JavaScript (the Worker's entry and the board's Durable Object), and everything about the Worker on Cloudflare, from which `go tool charter` deploys it and runs it on local workerd |
| `migrations/` | D1's schema |
| `e2e/` | the checks against a running server, its own Go module: `smoke/` over HTTP (`mise run smoke`), browser tests (`mise run e2e`), `load/` with 1,000 WebSockets (`mise run load`) |
| `ci/`, `tasks/`, `mise.toml` | the scripts CI runs or several tasks share, the mise tasks, the pinned tools |
| `docs/` | how it works, how it ships, every command |

`build/` and `dist/` are build output, never committed.

**Adding something, one way each:**

- **A page:** a type in the page table (`pages/pages.go`: its `Path`, an entry in `pages.All`), a component in
  `views/<page>.gsx` inside `Layout` with its own translated title and description, and a route in `app/server.go`
  (`pages.Route`) that renders it with `s.render`. Links to it are `{ pages.About{} |> url }`; the sitemap and the
  noindex rules follow from the table.
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
