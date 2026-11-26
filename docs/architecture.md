# How go-htmx4 works

## Runtime and build

- **One Go app** (`app/`, started by `main.go`) serves pages and fragments rendered from gsx (`views/`, `ui/`); the
  pages are structpages' tree (`views/pages.go`) on net/http's mux. On Cloudflare it is a Worker:
  [charter](https://github.com/joeblew999/charter)'s runtime (`transport.Run`) over workers-go keeps Go runtimes
  alive between requests and warms one while the Worker loads (`worker/index.mjs`, `build/go.mjs`). Off Workers the
  same handler is a plain Go server, which is what the dev loop and `go test` run.
- **The build** is `go tool charter build`, which wrangler runs itself (`wrangler.jsonc`'s build command): the
  messages package, gsx's Go, Vite's `dist/`, then the standard Go compiler to Wasm in `build/`.
- **The browser's files** (`web/`: htmx, gsxui's behaviours and CSS, the browser's Intl, the map, live video) are
  built by Vite into `dist/`, hashed, and served by Workers Static Assets before the Worker runs. The Go embeds
  Vite's manifest and names the files through `views/assets`.
- **Bindings** (D1, the Room Durable Object, the write limit, variables) are `wrangler.jsonc`'s. The Go reaches them
  through charter (`bindings`, `d1`, `hub`), each with a stand-in off Workers, so the same `database/sql`
  code runs everywhere.

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
  over 1–3 s after a deploy. `go tool charter test load` checks 1,000 sockets and a burst of writes.
- **Presence:** the Room pushes the count of open sockets on connects and closes, coalesced like the board; the page
  renders the label. hx-ws closes a hidden tab's socket, so it counts foreground tabs.
- **Limits:** 60 writes per 10 s per client IP (Workers Rate Limiting; over it, 429 and a toast), the newest 50 notes per
  topic, at most 1,000 sockets per Room (then 503). Topics are 1–32 of `a-z 0-9 -`.
- **JS only where Go can't go:** the entry (WebSockets) and the Room (a Durable Object class).

## Languages

Intl is the one source of language, through [charter](https://github.com/joeblew999/charter)'s `intl`. Views
write messages as typed values (`{ locales.NavHome }`, `{ locales.BoardNotes(n) }`) and Intl calls as gsx filters
(`{ t |> date(opts) }`, `{ n |> number }`), rendered by the renderers in `gsx.toml`. Every message (ICU MessageFormat,
plural forms by `Intl.PluralRules`), every value and every locale fact is formatted by the runtime's V8 Intl: on the
Worker, where Go calls it, and in the browser, which redoes the viewer's own time zone and relative times as they
age. Go routes and passes values; off Workers (`go test`, the dev loop) Intl is the workerd npm installs, which the
program starts itself.

`charter.toml`'s `[intl]` lists the languages and the translators' notes; `locales/*.toml` holds their text:
- `go tool charter build` compiles the catalogs.
- `go tool charter run translate` translates what is missing or stale with Claude Code.
- `go tool charter run todo` says what is left; `go tool charter run confirm` keeps a stale translation that is still right.
- `go tool charter check` fails while a language is incomplete.

`views/intl/` has the language screens (the language list, the language offer, the time zone and clock dialog), their
text already translated: `go tool charter add screen:<screen> -overwrite` writes them again.

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
- `go tool charter check` audits robots.txt and every sitemap URL as Googlebot reads them, on wrangler dev;
  `go tool charter run audit` does the same on the live site.

## Layout

[The README](../README.md#where-things-are) has the folders and the settings files.
