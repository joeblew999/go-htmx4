# go-htmx4 as a real repo: one app, a GitHub template, importable kit packages

**Status:** decisions made, waiting for go · **Created:** 2026-09-14 10:37

## Goal

Lock in what the two demos proved (gsx + gsxui + htmx 4 + hx-ws, Go on Workers via workers-go + TinyGo, D1 as the source of
truth, Durable Object fan-out with hibernation, workerd locally, a Go deploy client, mise + fnox, `gsx dev` on Workers) as
**one proper, reusable project**:

- **One app at the repo root, Workers-first.** It combines the gsxui demo's pages and the Workers demo's live board, with
  one layout, one theme toggle and one gsxui install.
- **A GitHub template:** clone it, rename it, `mise run dev`, `mise run deploy`.
- **Importable Go packages (`kit/`)** for the reusable parts, so other repos can `go get` them without copying:
  - the Cloudflare deploy client
  - the WebSocket load tester
  - log tail
  - the TinyGo-safe HTTP helpers
  - the Go side of the Durable Object fan-out
- **Extras:** protection against board abuse, a fast native `gsx dev` loop, a live log tail task, a LICENSE and a
  `v0.1.0` tag.

## Decisions (2026-09-14, your answers)

- **Restructure go-htmx4 in place.** Same public repo; history and `.plans/done` kept. `demos/` goes away.
- **Template + importable packages.**
- **One app, Workers-first.** workerd locally, with `go run .` as the fast native fallback.
- **Extras: all four.** Board abuse protection, LICENSE + `v0.1.0` tag, fast native gsx dev loop, live log tail task.
- **Names: new.** A new `go-htmx4` Worker + D1 (`APP_NAME`). The old `go-htmx4-workers-demo` (+ its D1) and
  `go-htmx4-gsxui-demo` Workers are deleted after the cutover is verified.
- **License: MIT.**
- **JS pieces (`room.mjs`, `index.mjs`, workerd shims): copied with the template.** No `go:embed`/sync command in v0.1.0.
- **e2e runs in CI** from the start: headless Chrome against local workerd, shell steps only (no Node, no JS actions).

## Ground rules (unchanged)

- **No Node, no wrangler, no miniflare.** Upstream guides only.
- **Toolchain:** workers-go `v0.35.0` exactly, and TinyGo 0.42 for everything that ships to Workers.
- **GUI is gsxui only.** Nothing hand-written: components come from `gsxui add`, plus the gsxui preset and site patterns.
  Never edit `*.x.go`.
- **Secrets:** Cloudflare credentials only via `fnox exec --`.
- **Outward-facing steps need your OK:** deploys, Cloudflare writes, GitHub settings, tags. Each is marked ⚠.
- **Working style:** work on `main`; in the dev loop use local checks (`mise run check`), not CI.
- **Moves:** use `git mv` so file history follows.

## Target layout

```
.
├── go.mod                   module github.com/joeblew999/go-htmx4 (app + kit, one module)
├── mise.toml                tools + APP_NAME / ports in [env]; includes tasks/*.toml
├── gsx.toml gsxui.json gsxui.preset.json
├── main.go routes.go        handlers: pages, fragments, board (plain paths + allow(): TinyGo ServeMux)
├── board.go store_sql.go store_mem.go
├── platform_js.go           workers.Serve, D1 store, kit/live publish, rate limit
├── platform_other.go        native server, embedded static files, mem store, GO_PORT for gsx dev
├── views/                   layout, theme, home, about, board, fragments (.gsx)
├── ui/                      gsxui components (union of both demos), merge/, icon/
├── web/gsxui/               gsxui behaviours + CSS entry + fonts
├── static/                  vendored htmx 4.0.0, hx-live.js, hx-ws.js (+ .htmx-version)
├── worker/                  index.mjs (entry), room.mjs (Room DO): deployed JS, logic-free
├── workerd/                 config.capnp, assets-first.mjs, local-entry.mjs, local-d1.mjs: local only
├── migrations/              0001_board.sql, 0002_note_retention.sql …
├── kit/                     importable, documented, tested; no app imports
│   ├── cfdeploy/            Direct Upload, script upload, bindings (vars, D1, DO, rate limit), D1 migrations, DO migrations
│   ├── cftail/              Workers tail session (live logs)
│   ├── wsload/              WebSocket connect/presence/push/load tester
│   ├── httpx/               Allow (GET⇒HEAD), WriteNode (buffered + Content-Length), static file helpers
│   └── live/                Publish(topic, version, fragment) to a Room DO (js) / no-op (native); ValidTopic
├── cmd/deploy cmd/tail cmd/wsload   thin CLIs over kit/
├── e2e/                     real-browser checks (Go + chromedp, no Node), from the scratchpad harnesses
├── tasks/app.toml tasks/upstream.toml
├── ci/check.sh .github/workflows/check.yml
├── LICENSE README.md AGENTS.md CLAUDE.md
└── .plans/
```

Merged app: `/` home (gsxui demo's form + OOB toast, dialog and tabs via `hx-get`, hx-live counter + filter, plus the
"no memory between requests" lesson), `/board` (live board), `/about` (stack). Boosted nav with `outerMorph` and the
theme toggle across all pages.

## Unknowns to settle early (they can change the design)

1. **Boosted nav vs hx-ws.** Boost + morph into `/board` has to open the socket, and leaving has to close it (presence
   drops). `hx-ws.js` and the version guard would load in the layout on every page, not per page. Check it in the browser
   before merging the pages. If it fails, `/board` links opt out of boost with `hx-boost:inherited="false"`.
2. **Rate limit from Go.** Does workers-go v0.35.0 expose arbitrary env bindings to Go (`cloudflare` package / `syscall/js`
   on the env)? If yes, rate limiting stays in `platform_js.go`. If not, `worker/index.mjs` does a one-line
   `env.WRITES.limit({key})` before handing the request to Go. Check the Cloudflare Rate Limiting binding docs for the
   upload-metadata shape (`type: "ratelimit"`, namespace id, simple limit/period).
3. **Tail without wrangler.** Confirm the REST flow in Cloudflare's API docs: create a tail for a script → WebSocket URL
   (`trace-v1` subprotocol) → JSON events → delete the tail on exit. The `golang.org/x/net/websocket` dependency is
   already in use.
4. **One module for app + kit.** Importers of `kit/*` get the app's requirements in their module graph (pruned, but
   `go.sum` grows). This is fine for now; a nested `kit/go.mod` stays an option if it gets in the way.

## Plan

### Phase 1: root app from the Workers demo (no behaviour change)

- [ ] `git mv demos/workers/*` → root layout above (`worker/`, `workerd/`, `static/`, `views/`).
- [ ] Module `github.com/joeblew999/go-htmx4`, then fix imports, `gsx.toml` `class_merger` and `gsxui.json` paths.
- [ ] Tasks move to `tasks/app.toml` with short names: `dev`, `dev:native`, `serve`, `run`, `test`, `smoke`, `load`,
      `deploy`, `smoke-remote`, `tail`. `check` = test + smoke. Ports: workerd 8913, native 9913.
- [ ] `APP_NAME` (Worker + D1 name) set once in `mise.toml` `[env]`.
- [ ] `mise run check` green; the board browser check (scratchpad harness) green against local workerd. **Commit.**

### Phase 2: merge the gsxui demo in

- [ ] Settle unknown 1 (boost + hx-ws) with a browser check first.
- [ ] `gsxui add` the union of components (dialog, tabs, toast, toaster, switch, native-select, …) into the root `ui/`.
      Move the views (`home`, `about`, `fragments`) and handlers (`/greet` POST/DELETE, `/fragments/server-info`,
      `/fragments/stats`). Static URL layout: `/static/`, `/gsxui/`, `/assets/`.
- [ ] Tests: port `demos/gsxui/main_test.go`. Workers smoke gets the gsxui checks (byte-identical native vs TinyGo HTML
      for the pages that don't depend on runtime state).
- [ ] `git rm -r demos/`. README/AGENTS point at the root app. `mise run check` green. **Commit.**

### Phase 3: `kit/` packages + CLIs

- [ ] `kit/cfdeploy` from `cmd/deploy` (549 lines): a `Config` struct plus `Deploy(ctx, cfg)`; `cmd/deploy` becomes flag
      parsing only. Unit tests against an `httptest` fake of the Cloudflare endpoints (assets session, buckets, script
      upload multipart, D1 query, DO migration). No real API calls in tests.
- [ ] `kit/wsload` from `cmd/wsload`; `kit/httpx` (Allow, WriteNode, file helpers); `kit/live` (Publish, ValidTopic
      shared with `index.mjs`'s regex, with a test that the two agree).
- [ ] Package docs (`doc.go`), examples (`Example_…`), `go vet` clean, no `kit/` → app imports. **Commit.**

### Phase 4: dev loops + tail

- [ ] `dev` = `gsx dev` → TinyGo + workerd (already proven, ~25 s per save).
- [ ] `dev:native` = Tailwind `--watch` + `gsx dev` on `go run .` (seconds per save; mem store, no live push; the board
      page says so with a gsxui `Badge`).
- [ ] One `gsx.toml` `[dev]` `upstream` from `APP_DEV_PORT` (the pattern proven in the gsxui demo).
- [ ] `kit/cftail` + `cmd/tail` + `mise run tail` (settle unknown 3). Pretty-print request line, status, `console.*`
      and exceptions. Credentials via fnox.
- [ ] Measure both loops' save → served times. **Commit.**

### Phase 5: board abuse protection

- [ ] Topic names: already `^[a-z0-9-]{1,32}$` in Go and `index.mjs`; pin both with the kit/live test.
- [ ] **Per-IP write limit** (`/board/add`, `/board/note`), keyed on `CF-Connecting-IP`, via the Workers Rate Limiting
      binding (settle unknown 2). Starting point: 20 writes / 10 s. Over the limit → 429 plus a gsxui toast. Local workerd
      has no rate limit binding, so a no-op locally and a unit test for the 429 path.
- [ ] **Note retention:** keep the newest 50 notes per topic in D1 (the page shows 5). A single `DELETE … WHERE topic = ?
      AND id < (SELECT id … ORDER BY id DESC LIMIT 1 OFFSET 49)` after each insert (no transactions in D1). Mirror it in
      the mem store and the local D1 shim.
- [ ] **Room socket cap:** refuse the 1,001st socket on a topic with 503. That is the designed ceiling (Room's ~1,000
      requests/s soft limit).
- [ ] `mise run load` still green (1,000 sockets). **Commit.**

### Phase 6: make it a proper template

- [ ] `e2e/`: move the chromedp harnesses (board two-tab, gsxui pages, theme, reconnect) into the repo as
      `mise run e2e`. It starts local workerd itself; `E2E_BASE=https://…` runs it against a live URL. Their scratchpad
      copies are the source.
- [ ] **e2e in CI:** `ci/check.sh` runs `mise run check` then `mise run e2e`. It uses the Chrome preinstalled on GitHub's
      Ubuntu runner image (check the runner-images docs; fail with a clear message if Chrome is missing). No credentials,
      no live URLs. Local `check` stays e2e-free so the dev loop stays fast.
- [ ] `mise run rename -- github.com/you/app my-app`: rewrites the module path, imports, `gsx.toml`/`gsxui.json` paths
      and `APP_NAME`; verified by running it in a scratch clone + `mise run check`.
- [ ] README rewritten as a starter:
  - what you get
  - quick start
  - architecture diagram
  - dev loops
  - deploy (fnox setup)
  - the TinyGo rules
  - using `kit/` from another repo
- [ ] AGENTS.md follows the new layout. `.plans/README.md` unchanged.
- [ ] `LICENSE` (MIT, holder "joeblew999"). **Commit + push** (no CI watching).

### Phase 7: deploy + cut over (⚠ needs OK)

- [ ] Deploy the app as the new `go-htmx4` Worker with a new `go-htmx4` D1, D1 migrations, the Room DO migration and the rate limit
      binding.
- [ ] Remote smoke, `mise run e2e` against the live URL, 1,000-socket load against live, and `mise run tail` during it.
- [ ] ⚠ After it's verified, delete the old `go-htmx4-workers-demo` Worker + its D1 and the `go-htmx4-gsxui-demo` Worker
      (REST, via fnox); confirm both URLs 404.

### Phase 8: release (⚠ needs OK)

- [ ] ⚠ Tag `v0.1.0` and push the tag.
- [ ] ⚠ `gh repo edit --template` and update the repo description.
- [ ] Check a fresh `go get github.com/joeblew999/go-htmx4/kit/cfdeploy@v0.1.0` from a scratch module.
- [ ] Move this plan to `.plans/done/`.

## Findings

(none yet)
