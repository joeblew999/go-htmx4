# How go-htmx4 works

## Runtime and build

- **One Go app** (`app/`, started by `main.go`) serves pages and fragments rendered from gsx (`views/`, `ui/`). On
  Cloudflare it is a Worker: [charter](https://github.com/joeblew999/charter)'s runtime (`transport.Run`) over
  workers-go keeps Go runtimes alive between requests and warms two while the Worker loads (`worker/index.mjs`,
  `build/go.mjs`).
- **The build** is charter's `go tool workerbuild` (`ci/wasm-build.sh`): TinyGo, patched and tuned for Workers, cached,
  so an unchanged build takes under a second. `WORKER_COMPILER=go` builds with the standard compiler instead (seconds,
  bigger, slower per request); `build:dev` always does, for `mise run dev`.
- **Static files** (`dist/site`: htmx, gsxui CSS and behaviours, fonts) are Workers Static Assets, served before the
  Worker runs. Locally `worker/local/config.capnp` stands in for them and for D1 (`worker/local/local-d1.mjs`, over Durable Object
  SQLite), so the same `database/sql` code runs on workerd.

## The shared board

```
browser ─ hx-ws:connect /live/{topic} ─▶ worker/index.mjs ─▶ Room Durable Object (worker/room.mjs, hibernatable WebSockets)
browser ─ hx-post /board/add|note ─────▶ worker/index.mjs ─▶ Go Worker
                                           1. D1: INSERT … ON CONFLICT DO UPDATE … RETURNING version
                                           2. Room: POST /publish <#board fragment>
                                                 └▶ ws.send(fragment) to every tab (≤ 5 broadcasts/s, newest wins)
```

- **D1 is the source of truth; the Room only fans out.** It caches the last fragment for tabs that (re)connect.
- **Every fragment is the whole `#board` with a `data-version`.** The page drops any swap older than what's on screen,
  so responses and pushes can arrive in any order. Each browser words the dates in it for its own viewer.
- **Built for 1,000 tabs per topic:** receive-only sockets, automatic ping/pong, coalesced broadcasts, reconnects spread
  over 1–3 s after a deploy. `mise run load` checks 1,000 sockets and a burst of writes.
- **Presence:** the Room pushes the count of open sockets on connects and closes, coalesced like the board; the page
  renders the label. hx-ws closes a hidden tab's socket, so it counts foreground tabs.
- **Limits:** 60 writes per 10 s per client IP (Workers Rate Limiting; over it, 429 and a toast), the newest 50 notes per
  topic, at most 1,000 sockets per Room (then 503). Topics are 1–32 of `a-z 0-9 -`.
- **JS only where Go can't go:** the entry (WebSockets) and the Room (a Durable Object class).

## Languages

Intl is the one source of language, through `intl/`: its own Go module, which any gsx app can use
([intl/README.md](../intl/README.md)). Views write messages as typed values (`{ locales.NavHome }`,
`{ locales.BoardNotes(n) }`) and Intl calls as gsx filters (`{ t |> date(opts) }`, `{ n |> number }`), rendered by the
renderers in `gsx.toml`. `intl/js/intl.mjs` formats every message (ICU MessageFormat, plural forms by
`Intl.PluralRules`), every value and every locale fact with the runtime's V8 Intl: on the Worker, where Go calls it,
and in the browser, which redoes the viewer's own time zone and relative times as they age. Go routes and passes
values; `go test` uses workerd's Intl (`intl/intltest`).

`intl.toml` lists the languages and the translators' notes; `locales/*.toml` holds their text. intl's own mise
tasks (`intl/mise/tasks.toml`) are the workflow:
- `intl` compiles the catalogs.
- `intl:translate` translates what is missing or stale with Claude Code.
- `intl:todo`, `intl:export` and `intl:import` serve people translating.
- `intl:check` (in `test:go`) fails on anything missing, stale, unused or short of a plural case.

URLs: the default locale is unprefixed, every other is `/<lowercase id>/…`, `/en/…` and mixed-case prefixes redirect;
no Accept-Language redirects, and only `/` follows the `locale` cookie. A locale whose catalog isn't complete is
served `noindex`.

## Search

- `/robots.txt` and `/sitemap.xml` are Go routes built from the request's origin; the sitemap lists every page in every
  locale with `hreflang` alternates.
- Every page has a translated title and description, a canonical URL, reciprocal `hreflang` with `x-default`, and Open
  Graph tags. Fragments, `/healthz`, form posts and previews send `X-Robots-Tag: noindex`.
- The Worker's `*.workers.dev` URL redirects to the custom domain (`app/host.go`), so one host is indexed.
- `mise run deploy` then audits robots.txt and every sitemap URL live as Googlebot (`search:audit`, charter's
  `go tool searchconsole`); `search:todo` gives Google's verdict per URL.

## Layout

```
.
├── main.go                                      the entry point: transport.Run(app.Handler())
├── app/                                         handlers, locale URLs, redirect, crawl rules, D1 / memory stores,
│                                                Cloudflare-only calls (platform_js.go) and their twins
├── views/  ui/  web/gsxui/  web/htmx/           pages (gsx), gsxui components and behaviours, vendored htmx
├── intl/  intl.toml                             Intl for gsx (its own module); the app's languages
├── locales/                                     UI messages (*.toml) and their generated package
├── internal/                                    httpx, live (Room publish), ratelimit, wsload, classmerge
├── cmd/                                         wsload (live-board check)
├── worker/  worker/local/                       the deployed JS (entry, Room); workerd config and shims, locally
├── migrations/                                  D1 schema
├── e2e/                                         smoke (HTTP) and browser tests: its own module
├── ci/  tasks/  mise.toml                       scripts, mise tasks, pinned tools
└── docs/                                        this page, deploy.md, commands.md (generated), benchmarks.md
```
