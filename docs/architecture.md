# How go-htmx4 works

## Runtime and build

- **One Go app** (`app/`, started by `main.go`) serves pages and fragments rendered from gsx (`views/`, `ui/`); the
  pages are structpages' tree (`views/pages.go`) on charter's `web.NewServeMux`. On
  Cloudflare it is a Worker: [charter](https://github.com/joeblew999/charter)'s runtime (`transport.Run`) over
  workers-go keeps Go runtimes alive between requests and warms two while the Worker loads (`worker/index.mjs`,
  `build/go.mjs`).
- **The build** is `charter build -compiler=go` (`ci/wasm-build.sh`): the standard Go compiler, in seconds.
  `WORKER_COMPILER=tinygo` builds with TinyGo instead (patched and tuned for Workers: a smaller wasm, faster per
  request, minutes to build, #188); `build:dev` is always standard Go, for `mise run dev`.
- **Static files** (`dist/site`: htmx, gsxui CSS and behaviours, fonts) are Workers Static Assets, served before the
  Worker runs. Locally `charter` stands in for them and for D1 (`.charter/`, generated from `charter.toml`: D1
  over Durable Object SQLite), so the same `database/sql` code runs on workerd.

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

Intl is the one source of language, through [mise-intl](https://github.com/joeblew999/mise-intl)
([mise-intl's guide](https://github.com/joeblew999/mise-intl/blob/main/templates/docs/intl/README.md)): its Go packages in `go.mod`, its binary in `mise.toml`, the same release.
Views write messages as typed values (`{ locales.NavHome }`, `{ locales.BoardNotes(n) }`) and Intl calls as gsx
filters (`{ t |> date(opts) }`, `{ n |> number }`), rendered by the renderers in `gsx.toml`. Every message (ICU
MessageFormat, plural forms by `Intl.PluralRules`), every value and every locale fact is formatted by the runtime's V8
Intl: on the Worker, where Go calls it, and in the browser, which redoes the viewer's own time zone and relative times
as they age. Go routes and passes values; `go test` uses workerd's Intl (`intltest`), which `ci/with-intl.sh` starts.

`intl.toml` lists the languages and the translators' notes; `locales/*.toml` holds their text. The app's own tasks
(`tasks/app.toml`) call mise-intl's binary:
- `intl:gen` compiles the catalogs.
- `intl:translate` translates what is missing or stale with Claude Code.
- `intl:todo` says what is left; `intl:confirm` keeps a stale translation that is still right.
- `intl:status` (as `mise-intl status`, in `test:go`) fails while a language is incomplete.

`views/intl/` has mise-intl's screens (the language list, the language offer, the time zone and clock dialog), their
text already translated: `mise-intl add -overwrite <screen>` writes them again.

URLs: the default locale is unprefixed, every other is `/<lowercase id>/…`, `/en/…` and mixed-case prefixes redirect;
no Accept-Language redirects, and only `/` follows the `locale` cookie. A locale whose catalog isn't complete is
served `noindex`.

## Search

- `/robots.txt` and `/sitemap.xml` are answered by `seo.Site` (middleware), built from the request's origin; the sitemap lists every page in every
  locale with `hreflang` alternates.
- Every page has a translated title and description, a canonical URL, reciprocal `hreflang` with `x-default`, and Open
  Graph tags. Fragments, `/healthz`, form posts and previews send `X-Robots-Tag: noindex`.
- Every page has one address: the Worker's `*.workers.dev` URL and plain http redirect to `https://` the custom domain
  (`seo.Site` in `app/site.go`), so one host is indexed.
- `mise run deploy` then audits robots.txt and every sitemap URL live as Googlebot (`seo:audit`). It is
  [mise-seo](https://github.com/joeblew999/mise-seo)'s, pinned in `mise.toml` and `go.mod` ([its guide](https://github.com/joeblew999/mise-seo/blob/main/templates/docs/seo/README.md)).

## Layout

```
.
├── main.go                                      the entry point: transport.Run(app.Handler())
├── app/                                         server.go (routes), board.go + store.go (D1), site.go (intl.Site,
│                                                seo.Site), platform.go (Cloudflare through charter's bindings)
├── views/  ui/  web/gsxui/  web/htmx/           pages (structpages' tree: pages.go; gsx; views/intl: intl's screens), gsxui components and
│                                                behaviours, vendored htmx
├── views/blocks/  web/blocks/                   the map, the places picker and live video, and their browser bundles
├── intl.toml  seo.toml                          the languages' and search engines' settings (mise-intl, mise-seo)
├── locales/                                     UI messages (*.toml) and their generated package
├── internal/                                    live (the Room's message rules), places, classmerge, deps and jsbundle (the repo's tools)
├── worker/  charter.toml                        the deployed JS (entry, Room); the Worker on Cloudflare (charter)
├── migrations/                                  D1 schema
├── e2e/                                         smoke/ (HTTP), browser tests, load/ (1,000 WebSockets): its own module
├── ci/  tasks/  mise.toml                       shared scripts, mise tasks, pinned tools
└── docs/                                        this page, deploy.md, commands.md (generated), benchmarks.md
```
