# Adopt workers-go (deploy to Cloudflare Workers)

**Status:** Phases 1–5 done; live at https://go-htmx4-workers-demo.gedw99.workers.dev and https://go-htmx4-gsxui-demo.gedw99.workers.dev · **Created:** 2026-09-13 11:11 · **Revised:** 2026-09-14

## Goal

1. **Run workers-go's own demo fully:** the `worker-tinygo` template, built and served locally in the
   real Workers runtime, and optionally deployed.
2. **Then build our own demo** with the same tooling: htmx 4 on Cloudflare Workers via workers-go.

## Ground rules

- **MUST use workers-go [`v0.35.0`](https://github.com/syumai/workers-go/releases/tag/v0.35.0)**
  (tag → commit `b2086b4`). Module path `github.com/syumai/workers-go`; never the old
  `github.com/syumai/workers` path. Pin exactly (`require github.com/syumai/workers-go v0.35.0`); upgrade
  on purpose only.
- **MUST use TinyGo** (0.42.0, pinned) for every Workers build: the `worker-tinygo` template,
  `workers-assets-gen -mode=tinygo`, `tinygo build -target wasm`. Never `GOOS=js GOARCH=wasm go build`, never
  the `worker-go` template or `-mode=go`. Upstream `_examples` that use `-mode=go` are rebuilt with TinyGo. If
  something won't build under TinyGo, log it and work around it in code; don't fall back to standard Go.
  **Scope (agreed 2026-09-13):** TinyGo covers everything that ships to Workers. Standard Go is allowed only for
  local-only code: the `go run .` fallback, `go test` / `go vet`, and the Phase 3 deploy tool. `demo:workers:test`
  always includes a `tinygo build` of the worker, so TinyGo breakage fails the check.
- **MUST NOT use any Node stuff.** No `node`, `npm`, `npx`, `pnpm`, `create-cloudflare`, **`wrangler`**
  (npm package) or miniflare. Only Go and standalone binaries, pinned in `mise.toml`.
- **Upstream tooling guides only.** Every step cites the workers-go, workerd or Cloudflare doc it comes from.
  Where upstream only documents a Node path, say so and use the closest Node-free step.
- **Don't port the existing starter.** The root app (`main.go`, `templates/`) stays as it is. *(2026-09-14: the root starter was removed; the repo is now just the two demos.)*
- **Work on `main`.** No branches. **No Docker.**
- **Nothing deploys without an explicit OK** (outward-facing).

## Where upstream uses Node, and what we do instead

workers-go's **build** is already Node-free. Node only appears in scaffolding, dev and deploy, all of
which go through wrangler.

| Upstream step (source) | Node? | Our replacement (source) |
| --- | --- | --- |
| `npm create cloudflare@latest -- --template github.com/syumai/workers-go/_templates/cloudflare/worker-tinygo` (README Quick Start) | yes | It only copies the template dir. We `git clone --branch v0.35.0` and copy `_templates/cloudflare/worker-tinygo`, then `go mod init` + `go mod tidy` as the template README says. |
| `npm run build` = `go run github.com/syumai/workers-go/cmd/workers-assets-gen && tinygo build -o ./build/app.wasm -target wasm -no-debug ./...` (template `package.json`) | npm only runs it | Run the same two commands from a mise task. |
| `make gen-wasm-exec` (repo `Makefile`, uses pnpm) | yes | Not needed. That's a maintainer step; v0.35.0 already embeds `wasm_exec.js` built for **Go 1.27.1 / TinyGo 0.42.0**, which match our mise pins. |
| `npm start` → `wrangler dev` (template README) | yes | **(a)** `go run .`: the template README's "run dev server without Wrangler (Cloudflare-related features are not available)" (non-js `workers.Serve` = `net/http` on `PORT`, default 9900). **(b)** `workerd serve config.capnp`: the real runtime (`wrangler dev` itself runs workerd, per the Wrangler commands docs). Config follows workerd's `samples/hello-wasm` (wasm module) and `samples/static-files-from-disk`. |
| `wrangler deploy` (template, examples) | yes | Cloudflare REST API: [Workers Script Upload](https://developers.cloudflare.com/workers/platform/infrastructure-as-code/#cloudflare-rest-api) (multipart + [metadata](https://developers.cloudflare.com/workers/configuration/multipart-upload-metadata/)), plus [Static Assets Direct Upload](https://developers.cloudflare.com/workers/static-assets/direct-upload/) (manifest → upload → deploy version). |
| `wrangler d1 create` / `d1 execute --remote` (`_examples/d1-blog-server/Makefile`) | yes | [D1 REST API](https://developers.cloudflare.com/api/resources/d1/): `POST /accounts/{id}/d1/database`, then `…/d1/database/{db}/query` with the schema SQL. |
| `wrangler d1 execute --local` | yes | **No Node-free equivalent.** Local D1 is miniflare (JS); workerd has no D1 binding. D1 is only tested against the deployed Worker. |
| joeblew999 shared `tool-wrangler.toml` / `tool-cf.toml` | yes (`npm:wrangler`) | **Not included.** Reuse only the curl-based `cf:token-check` pattern. |

## Tools (to add to `mise.toml`)

| Tool | Pin | Why |
| --- | --- | --- |
| Go | 1.27.1 (already pinned) | Matches v0.35.0's `GO_VERSION` (repo `Makefile`) |
| TinyGo | 0.42.0 (already pinned) | Template requirement: TinyGo ≥ 0.42.0 for `net/http` on wasm; matches `TINYGO_VERSION` |
| binaryen | `version_132` (already pinned) | `wasm-opt` for `tinygo -target wasm` |
| workerd | `workerd = "1.20260911.1"` (registry → `github:cloudflare/workerd`) | Standalone Workers runtime binary from the GitHub releases. The two newer releases (`…0912.1`, `…0913.1`) have no binaries attached yet. |
| workers-assets-gen | none: `go run github.com/syumai/workers-go/cmd/workers-assets-gen` | Version comes from `go.mod` (v0.35.0), exactly as the template does it |
| fnox | `fnox = "1.35.0"` (mise registry; the version installed on this machine) | Cloudflare credentials come from fnox **via mise**: `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` are already in the global fnox config (keychain provider). Every task that calls the Cloudflare API wraps its command in `fnox exec -- …`, the same pattern as `cf-do-locator/mise.toml`. No repo `fnox.toml`, no `.env`; secrets are never printed or committed. |

## Layout

```
.upstream/workers-go/        # gitignored checkout @ v0.35.0 (their repo, untouched)
.upstream/worker-tinygo/     # gitignored copy of _templates/cloudflare/worker-tinygo (what create-cloudflare would scaffold)
demos/workers/               # our demo: its own Go module
  main.go                    #   handlers + workers.Serve(mux), shared by workerd and `go run .` (template pattern)
  platform_js.go             #   //go:build js && wasm: cloudflare.* calls
  platform_other.go          #   //go:build !js: os.Getenv etc. for `go run .`
  page.html                  #   embedded page with {{target}}/{{env}} placeholders (no html/template under TinyGo)
  main_test.go               #   httptest on standard Go (demo:workers:smoke covers the TinyGo runtime)
  cmd/deploy/                #   Phase 3: stdlib Go Direct Upload tool (local-only, standard Go)
  config.capnp               #   workerd config: assets-first → public/ disk service, else Go Worker (build/)
  workerd/assets-first.mjs   #   local-only stand-in for Static Assets (runs inside workerd, not Node)
  public/                    #   static assets (htmx.min.js, css)
  build/                     #   gitignored: workers-assets-gen + tinygo output
tasks/workers.toml           # mise tasks (upstream:workers-go:*, demo:workers:*)
tasks/workerd/*.capnp        # tracked workerd configs for the upstream template and _examples/env
```

## Key constraints (they shape the design)

1. **No memory between requests.** `build/worker.mjs` (v0.35.0) compiles the module once but creates a
   new `WebAssembly.Instance` and Go runtime **on every fetch**. Package-level state resets; shared state
   has to live in a binding.
2. **Durable Objects are call-only.** workers-go can call DO stubs; the DO class has to be JS
   (`_examples/durable-object-counter`).
3. **TinyGo only** (ground rule). It also fixes size: standard Go wasm is already over the 3 MB free limit
   for the starter, TinyGo is 0.84 MB (see Findings). Always pass `-mode=tinygo` explicitly, even though it's
   the default. Serve static files with Workers Static Assets, not `go:embed`.
4. **Local bindings without wrangler are limited.** workerd natively gives text/JSON vars, wasm modules and
   disk directories. D1 (and a real local KV) are miniflare features, so they're deployed-only for us.
5. **`cloudflare:sockets`** is imported by the generated `runtime.mjs`. workerd has it built in, but Phase 1
   must confirm it loads under plain workerd.
6. **Request cost.** Avoid tight `hx-trigger="every …"` polling (100k requests/day on free).

## Plan

### Phase 1: Run workers-go's demo (upstream, Node-free)

- [x] mise wiring: add `workerd = "1.20260911.1"` and `fnox = "1.35.0"` to `[tools]`, add `tasks/workers.toml` to
      `[task_config] includes`, then `mise install`. Confirm `workerd --version` runs (the release asset is a
      single gzipped binary; check mise unpacks it). — `workerd 2026-09-11`, `fnox 1.35.0`
- [x] `mise run upstream:workers-go:fetch`: `git clone --depth 1 --branch v0.35.0` into
      `.upstream/workers-go` and check it's at `b2086b4`. Copy `_templates/cloudflare/worker-tinygo` to
      `.upstream/worker-tinygo` (this is all `create-cloudflare --template` does). `.upstream/` is already
      gitignored.
- [x] `mise run upstream:workers-go:init` (template README "Initialize a project"): `go mod init` +
      `go get github.com/syumai/workers-go@v0.35.0` + `go mod tidy`. Confirm `go.mod` has exactly v0.35.0.
- [x] `mise run upstream:workers-go:build`: the template's `build` script, run directly:
      `go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=tinygo` then
      `tinygo build -o ./build/app.wasm -target wasm -no-debug ./...`. Record raw and gzip sizes. — 746 KB raw / 284 KB gzip
- [x] `mise run upstream:workers-go:run`: `go run .` on :9900. `curl /hello` → `Hello!`,
      `curl -X POST -d "test message" /echo` → `test message` (template README "Testing dev server").
- [x] `mise run upstream:workers-go:serve`: `workerd serve` on **:8911** (8787 and 8797 are taken by shadcn-places'
      wrangler dev on this machine). The config is **tracked** at
      `tasks/workerd/worker-tinygo.capnp`; the task copies it next to `build/` in `.upstream/worker-tinygo/`,
      because workerd resolves `embed` paths relative to the config file. Modules: `worker.mjs` (esModule, main),
      `wasm_exec.js` + `runtime.mjs` (esModule), `app.wasm` (wasm), with names matching the relative imports.
      `compatibilityDate = "2026-09-11"` (not newer than the workerd binary). Same two curl checks.
- [x] Confirm constraint 1: add a `/count` handler with a package-level counter **to our copy** of the
      template. It should return 1 on every request under workerd and count up under `go run .`.
- [x] Bindings under workerd (**:8912**): `_examples/env` (its `go.mod` has `replace github.com/syumai/workers-go => ../../`,
      so it builds from the v0.35.0 checkout). Its Makefile uses `-mode=go`, so build it with the TinyGo commands
      above instead. Map its wrangler `[vars] MY_ENV` to a workerd `text` binding
      (`tasks/workerd/env.capnp`) and expect `MY_ENV: my env value`. Build output in `.upstream/` is fine (same as
      the gsxui plan); source files there stay untouched. Note which examples (`kv-counter`, `d1-blog-server`,
      `cache`, `cron`) can't run locally.
- [ ] ⚠ **Optional, needs your OK (not done):** deploy the template to `*.workers.dev` with the documented
      curl multipart upload, run as `fnox exec -- curl …` (`main_module: worker.mjs`; `application/javascript+module` parts for the .mjs/.js
      files, `application/wasm` for `app.wasm`). Look up the enable-workers.dev-subdomain call in the Workers API
      docs before this step.
- [x] Record findings below. **Stop and review with the user before Phase 2.**

### Phase 2: Our demo (`demos/workers/`), local only

- [x] `demos/workers/go.mod` (Go 1.27). `go get github.com/syumai/workers-go@v0.35.0`.
- [x] `main.go`: stdlib `http.ServeMux` passed to `workers.Serve`. htmx 4 page plus fragment endpoints,
      htmx 4 syntax only (same rules as the gsxui demo). — **plain path patterns + in-handler method check**: TinyGo's
      `net/http` has the pre-1.22 mux (see Findings)
- [x] Platform split: everything under `cloudflare/…` imports `syscall/js` (e.g. `cloudflare.Getenv`), so it
      won't compile for `go run .`. Keep handlers platform-free and put platform calls in `platform_js.go`
      (`//go:build js && wasm`, `cloudflare.Getenv`) and `platform_other.go` (`//go:build !js`, `os.Getenv`).
- [x] Rendering under TinyGo: try `html/template` first. If TinyGo can't build it or it's too big, log it and fall
      back to string/`io.WriteString` fragments. (The gsx runtime under TinyGo is a later question, see Phase 5.)
      — `html/template` **builds but panics at runtime**; fell back to embedded `page.html` with `{{name}}` placeholders
      + `html.EscapeString`
- [x] Demo content that shows what's Workers-specific:
  - stateless request/response fragments (`hx-get` / `hx-post` → HTML fragment)
  - an env var binding read with `cloudflare.Getenv`, shown on the page
  - a "request counter" that visibly **resets** per request (constraint 1 as a teaching point)
  - static assets (`public/htmx.min.js`, CSS) served by workerd's disk service, not by Go
- [x] `config.capnp`: route `/` to the worker, with static files from `public/` served first. Mirror production
      Static Assets behaviour as closely as workerd samples allow, and log any differences. — via
      `workerd/assets-first.mjs` (see Findings)
- [x] mise tasks (`tasks/workers.toml`): — plus `demo:workers:smoke` (below)
  - `demo:workers:build`: `workers-assets-gen -mode=tinygo` + `tinygo build -o build/app.wasm -target wasm
    -no-debug .`, then **fail if gzip size > 3 MB**. Build `.`, not the template's `./...`, so the Phase 3 deploy
    tool under `cmd/` isn't pulled into the wasm build.
  - `demo:workers:run`: `go run .` (non-js, fast loop)
  - `demo:workers:serve`: build + `workerd serve config.capnp`
  - `demo:workers:test`: `gofmt` check + `go vet` + `go test` (httptest, non-js) + `tinygo build` so the wasm
    target can't rot
  - `demo:workers:smoke` (added): serve the TinyGo build on workerd and curl every route. `demo:workers:test` depends
    on it, because `go test` on standard Go missed both TinyGo runtime bugs.
- [x] `.gitignore`: `demos/workers/build/`
- [x] Verify with curl against both `go run .` and workerd.
- [ ] ⚠ Browser check (needs your OK): click through the three htmx interactions on :8913, no console errors.

### Phase 3: Deploy (OK'd 2026-09-13; name `go-htmx4-workers-demo`, no D1)

- [x] Secrets from fnox via mise: `demo:workers:token-check` =
      `fnox exec -- sh -c 'curl -sS -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" …/user/tokens/verify'` (same
      check as the shared `cf:token-check`, without including that file). The inner `sh -c '…'` with single quotes
      matters: a bare `fnox exec -- curl … $CLOUDFLARE_API_TOKEN` expands the variable in the task shell **before**
      fnox injects it, which sends an empty token (same reason `cf-do-locator` uses `fnox exec -- bash -c '…'`).
      Deploy and D1 tasks run as `fnox exec -- go run ./cmd/deploy …` (reads the env inside Go) or
      `fnox exec -- sh -c 'curl …'`. The token check is the precondition for every Phase 3 task.
- [x] Read the account's Workers plan (free/paid) through the API with that token, and set the size gate
      (3 MB / 10 MB) from it. — `GET /accounts/{id}/subscriptions` → **Workers Paid**; gate set to 10 MB
- [x] Deploy tool: a small **stdlib Go** program (`demos/workers/cmd/deploy`) that follows Direct Upload
      step by step: build the manifest (hash + size per file in `public/`), POST the `assets-upload-session`,
      upload the buckets (base64 multipart, upload JWT), then PUT the script with modules + metadata (compat
      date, bindings, completion JWT). Use Go rather than curl because the manifest/bucket loop is multi-step
      JSON. — also enables workers.dev (`POST …/scripts/{name}/subdomain`), refuses to overwrite an existing
      script without `-allow-existing`, and has `-dry-run`
- [x] `demo:workers:deploy` → smoke-test the `*.workers.dev` URL with curl. — `demo:workers:smoke-remote`, 9/9
- [ ] Optional D1 (**skipped** for the first deploy, your call 2026-09-13; possible follow-up): create the database + apply `migrations/0001.sql` via the D1 REST API; add a D1 binding in
      metadata; D1 code behind `//go:build js && wasm` (the `cloudflare/d1` package is js-only), with a memory
      store for `go run .`. Deployed-only testing (constraint 4).

### Phase 4: Docs

- [x] README: Stack table (workers-go v0.35.0, workerd), Demos section with run/serve/deploy commands. — also fnox row,
      "Cloudflare Workers without Node" section with the TinyGo caveats; Hosting moved from Planned to In use
- [x] AGENTS.md: no wrangler/Node for Workers; `demos/workers` tasks; no package-level state in handlers
      that's expected to persist; TinyGo is the Workers target. — new "Demo (Cloudflare Workers)" section, incl. TinyGo
      `ServeMux`/`html/template` gaps, platform split, fnox quoting, ports, deploy needs OK

### Phase 5 (later, separate go-ahead): gsxui demo on Workers

**Read-only analysis (2026-09-14, nothing built):**

- Runtime deps of `demos/gsxui` are small: `gsx`, `gsx/internal/htmlattr`, `gsx/std`, `tailwind-merge-go`
  (`twmerge`, `cache`, `lru`), plus our `ui`, `ui/icon`, `ui/merge`, `views`.
- Generated code only calls `gw.Node/BoolAttr/Class/StyleMerged/Spread/Text/AttrValue/NodeResult/ClassMerged/Nonce/IntInto/URL`.
- **Reflection that is reached:** `anyRenderVal` (`renderval.go`) runs for spread and dynamic attribute values:
  `fmt.Stringer` check, then `reflect.ValueOf(v).Kind()` switch (`reflect.Copy` only for named byte slices). This is basic
  reflection, not the method calls (`NumOut`) that broke `html/template`, but it needs a runtime check.
- **Reflection not reached by this demo:** `js.go` `jsValEscaper` (`encoding/json` + `reflect.Type.Implements`) only runs
  for Go values interpolated into JS. The demo's `hx-on`/`hx-live` use `js` literals (`RawJS`).
- `tailwind-merge-go` is `regexp` + `sync` + an LRU: should work, but adds size.
- **Must port for TinyGo:** `main.go` uses Go 1.22 patterns (`GET /{$}`, `GET /static/`, `POST /greet`, `DELETE /greet`, …)
  that TinyGo's pre-1.22 mux can't route; static files come from `embed` + `http.FileServer` (should be Static Assets on
  Workers); the `requests`/`stats` counters reset per request on Workers.

**Steps (each stops for review):**

- [x] **5a. Spike, local:** `go tool gsx generate`, then TinyGo-build the demo as-is (record compile errors + size); then
      a scratch TinyGo Worker that renders `views.Home` / `views.About` / the fragments on workerd, curling for
      panics (the `anyRenderVal` + `twmerge` paths). Go/no-go. — **GO** (2026-09-14):
  - Demo as-is compiles with TinyGo 0.42 for wasm with **no errors**: 2,674,322 B raw / 986,731 B gzip (includes the
    embedded static files and fonts).
  - Scratch Worker (scratchpad, not committed): `views.Home`, `About`, `Greeting` (POST, shout + `<b>` markup),
    `GreetingError` (with `<name> & "more"`), `GreetingCleared`, `ServerInfoView`, `Stats`, `DialogCard`, `GreetCard`,
    `LiveCard`, `TabsCard` via `workers.Serve` with plain-path routes. TinyGo build **1,927,491 B raw / 617,585 B gzip**
    (gsx + all gsxui components + tailwind-merge-go, no embedded assets).
  - **All 11 responses are byte-identical** between TinyGo on workerd and the same code on standard Go (`go run .`):
    home 47,979 B, about 16,809 B, cards 3.7–13.6 KB, fragments 133 B–1.9 KB. So `anyRenderVal`'s reflection, spread
    attributes, `ClassMerged`/`StyleMerged` (tailwind-merge regexp + LRU) and escaping behave identically.
  - No panics or errors in the workerd log. Single renders took 5–15 ms on workerd (fresh Go runtime per request). 50
    concurrent `GET /` gave 50/50 identical output, p50 342 ms / p95 507 ms (one local workerd thread, CPU-bound
    queueing).
  - Remaining porting work is exactly what the analysis listed (routes, static assets, counters); nothing in gsx/gsxui
    needs changing.
- [x] **5b. Port `demos/gsxui` to one app, two entrypoints:** plain-path routes + method checks (incl. `DELETE /greet`),
      platform split (`workers.Serve` on js; `gsx dev`/`GO_PORT` server on `!js`), static files as a Static Assets dir
      assembled by a task (`/static`, `/gsxui`, `/assets` = `dist` + fonts), counters labelled per-request on Workers.
      Existing `go test` stays green. — done 2026-09-14:
  - `main.go` (shared): plain-path routes + `allow(w, r, methods…)` (GET also allows HEAD; `/greet` = POST, DELETE),
    exact-path 404 for `/`. `platform_other.go` (`!js`): the native `main` (GO_PORT/PORT/7777), the `embed` FS and
    `staticRoutes` (GET-only). `platform_js.go` (`js && wasm`): `workers.Serve(newServer().routes())`, `staticRoutes` a
    no-op. `workers-go v0.35.0` added to the module.
  - `views.ServerInfo` gained `Note` (rendered only when set); server-info shows `runtime.Compiler + Version + GOOS/GOARCH`:
    `gc go1.27.1 darwin/arm64` natively, **`tinygo 0.42.0 js/wasm`** on Workers (TinyGo's `runtime.Version()` is the
    TinyGo version, not `go1.x`), plus the per-request note.
  - `go test` + new `TestRouteRules` (405 + `Allow` for GET /greet, POST /, POST /about, POST /static/…; 404 for unknown
    paths; HEAD /) and `demo:gsxui:test` pass.
- [x] **5c. Tasks:** `demo:gsxui:workers:{build,serve,smoke}` (TinyGo, size gate, workerd + assets-first, curl checks). — done:
  - `demo:gsxui:workers:assets` → `build/assets` (27 files, 748 K), `:build` → `build/worker` (`workers-assets-gen -o`),
    **1,938,823 B raw / 622,054 B gzip**; `workers.capnp` (:8918) reuses `../workers/workerd/assets-first.mjs`.
  - `demo:gsxui:workers:smoke`: **17/17** (home markers incl. `hx-boost:inherited` + hx-live, about, POST greet escaped +
    empty, DELETE greet, server-info on TinyGo + note, stats, 5 assets with content types, 405, 404, no panics).
  - Native build (GO_PORT=7788) vs TinyGo on workerd, byte-for-byte: `/`, `/about`, POST/DELETE `/greet`, and all 5
    assets are **identical**. `/fragments/stats` differs by design (native accumulates, Workers starts empty per request).
  - ⚠ Incident: port 7777 was already held by a demo server I didn't start (probably yours or the other session's);
    my first comparison accidentally measured it, and my cleanup killed it. Redone on 7788 with PID-scoped cleanup.
    Rule: only kill processes by the PID you started.
- [x] **5d. Browser check (chromedp) on workerd:** boosted nav + `outerMorph`, dialog `hx-get`, OOB toast, lazy tab,
      hx-live counter/filter, no console errors. — done 2026-09-14, **18/18** on the TinyGo build under workerd:
  - page loads htmx + hx-live + toaster; web fonts load from `/assets/`; hx-live counter, menu (opens, closes on click
    outside), `aria.pressed` + `:class` toggle, filter (`dia` → `dialog`); greet form swaps `#greeting` with `<b>`
    escaped + success toast adopted by the toaster; Clear (`hx-action` + `hx-method=delete`) + info toast; dialog opens
    (`showModal`, `data-state=open`), loads server info via `hx-get` (`tinygo 0.42.0 js/wasm` + note), closes on Escape;
    Stats tab loads lazily; boosted nav to `/about` in the same document; **Back** restores `/` in the same document;
    no console errors, exceptions or HTTP ≥ 400 (except the demo's missing `/favicon.ico`, same natively).
  - **Bug found and fixed:** Back left the About content on screen on workerd (native was fine). htmx 4's history
    restore sent `GET /` with `HX-History-Restore-Request: true`, got 200 `text/html`, but swapped nothing. The only
    difference from native was framing: workers-go streamed the response chunked (no `Content-Length`). `render` now
    renders into a buffer and sets `Content-Length` (also turns render errors into a clean 500). Back then passed.
    htmx core has no length-specific code, so the exact mechanism (possibly the local assets-first streaming layer) is
    not pinned down; the fix is verified locally and must be re-checked on the deployed Worker.
  - Harness: a scratchpad Go program with chromedp (not committed); the same checks against the native server pass
    except the TinyGo-specific server-info text.
- [x] **5e. Deploy (⚠ needs OK):** separate Worker `go-htmx4-gsxui-demo` via `demos/workers/cmd/deploy`, remote smoke. — done 2026-09-14:
  - Name checked free (read-only), then `mise run demo:gsxui:workers:deploy` = local workerd smoke (17/17) + token
    check + `cmd/deploy -main "" -build ../gsxui/build/worker -assets ../gsxui/build/assets`. 27 manifest entries,
    **21 files uploaded** (the 6 fonts under both `/gsxui/fonts/` and `/assets/` share hashes), 4 modules.
  - **Live:** https://go-htmx4-gsxui-demo.gedw99.workers.dev. `demo:gsxui:workers:smoke-remote` (same script, `SMOKE_BASE`)
    **17/17**; headless Chrome **18/18** incl. boosted nav + **Back** (same document), toasts, dialog, lazy tab, hx-live,
    no console errors.
  - Live responses carry no `Content-Length` header (Cloudflare re-frames them) and Back still works, which suggests
    the local `assets-first.mjs` streaming layer, not htmx itself, triggered the Back bug on workerd. Buffering stays.

## Risks

- **Experimental library, fresh rename** (v0.35.0, 2026-09-05). Pin exactly. Migrating `irgo` off
  `syumai/workers v0.33.0` is a separate task (irgo also uses wrangler, so only its Go layout is precedent).
- **Plain workerd ≠ wrangler dev.** No miniflare means no local D1/KV/Queues and possibly different
  static-asset routing. Keep the local surface small and test bindings deployed.
- **Upload API drift.** Cloudflare marks the newer Workers API as beta; use the stable multipart script upload
  and log exact endpoints in Findings.
- **Cold start per request** (new Go runtime each fetch). Measure in Phase 1; TinyGo helps.
- **TinyGo stdlib gaps** (`html/template`, `embed`, reflection-heavy code). TinyGo is mandatory, so log each gap
  and work around it in code (simpler rendering, fewer deps). Standard Go is never the fallback.
- **D1 free-tier limits are enforced since 2026-09-01** (queries fail past daily row limits), and workers-go D1
  is "alpha".
- **workerd releases are daily** and some have no binaries. Pin a version that has one; compatibility date must
  not be newer than the binary.

## Open questions

- Custom domain later, or stay on `go-htmx4-workers-demo.gedw99.workers.dev`?
- Phase 1's optional upstream template deploy: still wanted, or skip now that our demo is live?
- D1 counter as a follow-up? → superseded by `2026-09-14_0754_workers-realtime-d1-do.md` (D1 as truth + Durable
  Object fan-out over hx-ws, decided 2026-09-14)

## References

- workers-go v0.35.0: https://github.com/syumai/workers-go/tree/v0.35.0 (README, `_templates/cloudflare/worker-tinygo`,
  `cmd/workers-assets-gen`, `_examples/{hello,env,kv-counter,d1-blog-server,durable-object-counter}`, `Makefile`)
- Generated shim: `cmd/workers-assets-gen/assets/common/worker.mjs`, `assets/runtime/cloudflare.mjs`
- workerd: https://github.com/cloudflare/workerd (README, `samples/hello-wasm`, `samples/static-files-from-disk`,
  `src/workerd/server/workerd.capnp`), releases https://github.com/cloudflare/workerd/releases
- Cloudflare: Infrastructure as Code → REST API, Multipart upload metadata, Static Assets Direct Upload,
  D1 REST API, Compatibility dates, Account plan limits
- In-house precedent (Go layout only): `../irgo/docs-templ/main_cloudflare.go`, `../irgo/cmd/irgo/app_cloudflare_build.go`

## Findings log

### 2026-09-13 14:40: Phase 3 results (deployed)

**Live:** https://go-htmx4-workers-demo.gedw99.workers.dev (account workers.dev subdomain `gedw99`, plan **Workers Paid**).
`mise run demo:workers:deploy` = token check + TinyGo build + workerd smoke + go test, then
`fnox exec -- go run ./cmd/deploy -name go-htmx4-workers-demo -allow-existing -var "DEMO_ENV=cloudflare (workers.dev)"`.

- **Flow worked first time with the documented Direct Upload API**, no wrangler: `assets-upload-session` returned 2
  buckets (1 file each) → 2 `assets/upload?base64=true` posts → completion JWT → multipart `PUT …/workers/scripts/{name}`
  (metadata `main_module: worker.mjs`, `compatibility_date: 2026-09-11`, `plain_text` binding, `assets.jwt`; modules
  `worker.mjs`/`wasm_exec.js`/`runtime.mjs` as `application/javascript+module`, `app.wasm` as `application/wasm`) →
  `POST …/scripts/{name}/subdomain {"enabled":true,"previews_enabled":false}` → `GET …/workers/subdomain`. About 7 s.
- **Asset hash:** `sha256(base64(content) + extension)`, first 32 hex (from the cloudflare-typescript example in the docs).
  Cloudflare serves it back as the ETag (`"75c852f9…"` for `htmx.min.js`).
- **Redeploy** of an unchanged build: the upload session returns no buckets ("assets unchanged"), script updated.
  Without `-allow-existing` the tool refuses because the script exists (guard verified).
- **Remote smoke (`demo:workers:smoke-remote`), 9/9:** `tinygo js/wasm`, `DEMO_ENV` = `cloudflare (workers.dev)`,
  fragment, escaped POST, count `1 1` (reset per request), `htmx.min.js` 200 `text/javascript` (`cf-cache-status: HIT`),
  `demo.css` 200 `text/css`, 405, 404.
- **Latency** from this Mac: page about 65–110 ms end-to-end (network included; fresh Go runtime per request).
- **Pre-checks were read-only:** 45 existing scripts in the account, none named `go-htmx4*`.
- `assets-first.mjs` is not deployed: in production Static Assets answers file paths before the Worker, and the Go
  `staticFiles()` on js returns 404 for anything that slips through (the `/nope.txt` check).
- Not done: D1 (skipped), browser check, upstream template deploy.

### 2026-09-13 14:10: Phase 2 results (local only, nothing deployed)

**Our demo runs on workerd and under `go run .`.** `mise run demo:workers:serve` → http://localhost:8913,
`mise run demo:workers:run` → http://localhost:9913, `mise run demo:workers:test` (build + smoke + gofmt + vet + test) passes.

| Check (curl) | workerd :8913 | `go run .` :9913 |
| --- | --- | --- |
| page `target` / `env` | `tinygo js/wasm` / `workerd (local)` | `gc darwin/arm64` / `go run . (local)` |
| `GET /fragments/now` | `<time …>` from `tinygo js/wasm` | same, from `gc darwin/arm64` |
| `POST /greet` (Ada / `<script>` / empty) | `Hello, Ada.` / escaped / error fragment | same |
| `POST /count` ×4 | `1 1 1 1` | `1 2 3 4` |
| `/htmx.min.js`, `/demo.css` | 200, `text/javascript` / `text/css` (from `public/`, not Go) | 200 (Go file server) |
| `/nope.txt`, `GET /greet` | 404, 405 `Allow: POST` | 404, 405 `Allow: POST` |
| page latency (loopback) | about 4 ms (fresh Go runtime per request) | about 0.3 ms |

- **wasm size:** 950,755 B raw / **360,048 B gzip** (12% of the 3 MB free limit). With `html/template` it was
  1,803,209 / 635,937 B.
- **TinyGo gap 1, routing:** TinyGo 0.42 ships its own `net/http` fork (`src/net/http/server.go`, "copied and
  modified from Go 1.26.2") whose `ServeMux` is the **old map-based mux**: no `GET /path` method patterns, no `{$}`
  (only a partial `{id}` placeholder hack). `//go:debug httpmuxgo121=0` and `godebug` in `go.mod` don't change it.
  Symptom: every route returned Go's `404 page not found` on workerd while `go test` passed. Fix: plain path patterns
  and an `allow(w, r, method)` check in each handler.
- **TinyGo gap 2, `html/template`:** compiles, then panics at execute time with
  `unimplemented: (reflect.Type).NumOut()`, surfacing as a 500 and "Worker's code had hung". Fix: embedded `page.html`
  with `{{target}}` / `{{env}}` placeholders filled via `strings.NewReplacer` + `html.EscapeString`, and fragments as
  escaped strings. **This is a warning for Phase 5:** anything reflection-heavy (template engines, maybe the gsx
  runtime) needs a TinyGo runtime check, not just a build.
- **Why `go test` wasn't enough:** handler tests run on standard Go. Hence `demo:workers:smoke`, which curls the real
  TinyGo wasm under workerd. It checks target, binding, fragment, escaped POST, per-request reset, static-first, 405
  and 404, and fails on `panic`/`uncaught` in the workerd log.
- **Static Assets locally:** `workerd/assets-first.mjs` (~20 lines, modeled on workerd `samples/static-files-from-disk`)
  serves a `public/` file if the last path segment has a dot and the file exists, otherwise forwards to the Go Worker.
  That matches production's default (`run_worker_first = false`) for our routes. It's JavaScript run **inside
  workerd**, not Node, and it's local-only: on Cloudflare, `public/` is uploaded as Static Assets. Known differences from
  production: no `index.html` / `html_handling` rules, a tiny content-type map, no ETag/caching headers.
- **`count`** is a package-level `int` on purpose, to show the reset. It isn't goroutine-safe under `go run .`; fine for
  the demo, but not a pattern to copy.

### 2026-09-13 13:40: Phase 1 results (local only, nothing deployed)

**workers-go's template runs Node-free under both targets.** Tasks: `tasks/workers.toml`
`upstream:workers-go:{fetch,init,build,run,serve,env}`; workerd configs tracked in `tasks/workerd/`.

- **Toolchain:** mise installed `workerd@1.20260911.1` from `workerd-darwin-arm64.gz` (unpacked fine, reports
  `workerd 2026-09-11`). fnox 1.35.0 pinned (not used yet: no Cloudflare calls in Phase 1).
- **Pin:** checkout verified at `b2086b4` = tag `v0.35.0`. Template copy `go.mod`: `github.com/syumai/workers-go v0.35.0`,
  `go 1.27.1`.
- **TinyGo build** (`-mode=tinygo`, TinyGo 0.42.0): template `app.wasm` **763,700 B raw / 290,450 B gzip** (with the
  `/count` probe), about 10% of the 3 MB free limit. `_examples/env`: 323,066 B gzip. No TinyGo build errors.
- **`go run .`** (:9900): `/hello` → `Hello!`, `/echo` → `test message`, `/count` → `1 2 3 4 5`.
- **workerd** (:8911): `/hello` → `Hello!`, `/echo` → `test message`, unknown path → Go's `404 page not found`.
  `cloudflare:sockets` resolves, and no module errors in the log. Module names without `./` (`worker.mjs`, `wasm_exec.js`,
  `runtime.mjs`, `app.wasm`) match the generated relative imports. About **2.5 ms per request** on loopback, including
  the fresh Go runtime per request.
- **Constraint 1 confirmed:** workerd `/count` → `1 1 1 1 1`. Package-level state resets on every request.
- **Env binding:** `_examples/env` built with TinyGo (not its Makefile's `-mode=go`). The workerd `text` binding
  `MY_ENV` → `cloudflare.Getenv` → `MY_ENV: my env value`. The upstream checkout's source is untouched (`git status` clean;
  `build/` is ignored upstream).
- **Ports:** 8787 and 8797 are held by `shadcn-places`' wrangler dev on this machine, so our workerd uses 8911 (template)
  and 8912 (env). Phase 2 should pick its own fixed port too.
- **Upstream examples that need more than plain workerd:** `durable-object-counter` (JS DO class), `queues`,
  `r2-image-*`, `browser`, `cron` / `multiple-handlers` (cron triggers, which have no local scheduler without wrangler),
  `d1-blog-server` (D1). `kv-counter`'s and `d1-blog-server`'s wrangler.toml don't declare their bindings; they're added
  by hand. `service-bindings` should map to workerd services (not tried). None of these were run.
- **Not done:** the optional upstream deploy (needs your OK).

### 2026-09-13 12:50: tooling audit for the Node-free rewrite (research only)

Read v0.35.0 at `b2086b4` in a scratch clone; nothing installed or built in the repo.

- The README and both templates list **Node.js as a requirement**, but only for `create-cloudflare` and
  wrangler. The `build` script is pure Go/TinyGo.
- `workers-assets-gen` just copies embedded files into `build/`: `wasm_exec.js` (Go or TinyGo variant per
  `-mode`, **default `tinygo`**), `runtime.mjs` (`-runtime`, only `cloudflare`), and `worker.mjs`. `runtime.mjs`
  imports `cloudflare:sockets` and `./app.wasm`.
- Non-js `workers.Serve` is a plain `net/http` server on `PORT` (default 9900); `Ready`/`Done`/`ServeNonBlock`
  panic outside js.
- The repo `Makefile` pins `GO_VERSION 1.27.1`, `TINYGO_VERSION 0.42.0`, the same as our `mise.toml`.
- workerd is in the mise registry (`github:cloudflare/workerd`). Latest release with binaries:
  `v1.20260911.1` (darwin-arm64/64, linux, windows). Its config schema supports `esModule` and `wasm` modules,
  disk directory services and `kvNamespace` bindings (needs a backing service), but has **no D1 binding**.
- The joeblew999 shared `tool-wrangler.toml` / `tool-cf.toml` pin `npm:wrangler`, so we can't use them.
- Readiness check: `mise ls-remote workerd` lists `1.20260911.1`. `fnox` is in the mise registry. workerd has a
  built-in `cloudflare:sockets` module (`src/cloudflare/sockets.ts`), so `runtime.mjs`'s import should resolve.
  `cloudflare.Getenv` (and the rest of `cloudflare/…`) imports `syscall/js`, which means the demo needs a
  build-tag split. `.upstream/` is already gitignored.

### 2026-09-13 11:20: wasm size of the current app (before any workers-go code)

Toolchain via mise: Go 1.27.1, TinyGo 0.42.0 (supports Go ≤ 1.27), binaryen/wasm-opt 132
(TinyGo needs `wasm-opt`; mise's aqua registry is stale at 121, so it's pinned as
`github:WebAssembly/binaryen`).

| Build | Raw | gzip -9 |
| --- | --- | --- |
| `GOOS=js GOARCH=wasm go build` | 17.5 MB | **4.1 MB** (over the 3 MB free limit) |
| `tinygo build -target wasm -no-debug` | 2.4 MB | **0.84 MB** |

The starter already exceeds the free-plan limit with standard Go, one more reason for the **TinyGo MUST**. Phase 1 still needs to confirm TinyGo + workers-go at runtime and whether
`html/template` / `embed` behave under TinyGo.
