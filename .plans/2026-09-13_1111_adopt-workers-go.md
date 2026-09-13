# Adopt workers-go (deploy to Cloudflare Workers)

**Status:** ready, waiting for go on Phase 1 · **Created:** 2026-09-13 11:11 · **Revised:** 2026-09-13 12:50

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
- **Don't port the existing starter.** The root app (`main.go`, `templates/`) stays as it is.
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
  main.go                    #   workers.Serve(handler), same file for workerd and `go run .` (template pattern)
  config.capnp               #   workerd config: build/*.mjs + app.wasm modules, public/ as disk service
  public/                    #   static assets (htmx.min.js, css)
  build/                     #   gitignored: workers-assets-gen + tinygo output
tasks/workers.toml           # mise tasks (upstream:workers-go:*, demo:workers:*)
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

- [ ] mise wiring: add `workerd = "1.20260911.1"` and `fnox = "1.35.0"` to `[tools]`, add `tasks/workers.toml` to
      `[task_config] includes`, then `mise install`. Confirm `workerd --version` runs (the release asset is a
      single gzipped binary; check mise unpacks it).
- [ ] `mise run upstream:workers-go:fetch`: `git clone --depth 1 --branch v0.35.0` into
      `.upstream/workers-go` and check it's at `b2086b4`. Copy `_templates/cloudflare/worker-tinygo` to
      `.upstream/worker-tinygo` (this is all `create-cloudflare --template` does). `.upstream/` is already
      gitignored.
- [ ] `mise run upstream:workers-go:init` (template README "Initialize a project"): `go mod init` +
      `go get github.com/syumai/workers-go@v0.35.0` + `go mod tidy`. Confirm `go.mod` has exactly v0.35.0.
- [ ] `mise run upstream:workers-go:build`: the template's `build` script, run directly:
      `go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=tinygo` then
      `tinygo build -o ./build/app.wasm -target wasm -no-debug ./...`. Record raw and gzip sizes.
- [ ] `mise run upstream:workers-go:run`: `go run .` on :9900. `curl /hello` → `Hello!`,
      `curl -X POST -d "test message" /echo` → `test message` (template README "Testing dev server").
- [ ] `mise run upstream:workers-go:serve`: `workerd serve config.capnp` on :8787. Config modules: `worker.mjs`
      (esModule, main), `wasm_exec.js` + `runtime.mjs` (esModule), `app.wasm` (wasm), with module names
      matching the relative imports. Same two curl checks.
- [ ] Confirm constraint 1: a handler with a package-level counter returns 1 on every request under workerd
      (and counts up under `go run .`).
- [ ] Bindings under workerd: run `_examples/env` (its Makefile uses `-mode=go`; build it with the TinyGo
      commands above instead) with a text binding and check what works without miniflare. Note which examples (`kv-counter`, `d1-blog-server`, `cache`, `cron`) can't run locally.
- [ ] ⚠ **Optional, needs your OK:** deploy the template to `*.workers.dev` with the documented
      curl multipart upload, run as `fnox exec -- curl …` (`main_module: worker.mjs`; `application/javascript+module` parts for the .mjs/.js
      files, `application/wasm` for `app.wasm`). Look up the enable-workers.dev-subdomain call in the Workers API
      docs before this step.
- [ ] Record findings below. **Stop and review with the user before Phase 2.**

### Phase 2: Our demo (`demos/workers/`), local only

- [ ] `demos/workers/go.mod` (Go 1.27). `go get github.com/syumai/workers-go@v0.35.0`.
- [ ] `main.go`: stdlib `http.ServeMux` passed to `workers.Serve`. htmx 4 page plus fragment endpoints,
      htmx 4 syntax only (same rules as the gsxui demo).
- [ ] Platform split: everything under `cloudflare/…` imports `syscall/js` (e.g. `cloudflare.Getenv`), so it
      won't compile for `go run .`. Keep handlers platform-free and put platform calls in `platform_js.go`
      (`//go:build js && wasm`, `cloudflare.Getenv`) and `platform_other.go` (`//go:build !js`, `os.Getenv`).
- [ ] Rendering under TinyGo: try `html/template` first. If TinyGo can't build it or it's too big, log it and fall
      back to string/`io.WriteString` fragments. (The gsx runtime under TinyGo is a later question, see Phase 5.)
- [ ] Demo content that shows what's Workers-specific:
  - stateless request/response fragments (`hx-get` / `hx-post` → HTML fragment)
  - an env var binding read with `cloudflare.Getenv`, shown on the page
  - a "request counter" that visibly **resets** per request (constraint 1 as a teaching point)
  - static assets (`public/htmx.min.js`, CSS) served by workerd's disk service, not by Go
- [ ] `config.capnp`: route `/` to the worker, with static files from `public/` served first. Mirror production
      Static Assets behaviour as closely as workerd samples allow, and log any differences.
- [ ] mise tasks (`tasks/workers.toml`):
  - `demo:workers:build`: `workers-assets-gen -mode=tinygo` + `tinygo build -o build/app.wasm -target wasm
    -no-debug .`, then **fail if gzip size > 3 MB**. Build `.`, not the template's `./...`, so the Phase 3 deploy
    tool under `cmd/` isn't pulled into the wasm build.
  - `demo:workers:run`: `go run .` (non-js, fast loop)
  - `demo:workers:serve`: build + `workerd serve config.capnp`
  - `demo:workers:test`: `gofmt` check + `go vet` + `go test` (httptest, non-js) + `tinygo build` so the wasm
    target can't rot
- [ ] `.gitignore`: `demos/workers/build/`
- [ ] Verify with curl against both `go run .` and workerd. Browser check only with your OK.

### Phase 3: Deploy (⚠ needs your OK, account and plan choice)

- [ ] Secrets from fnox via mise: `demo:workers:token-check` =
      `fnox exec -- curl -sS -H "Authorization: Bearer $CLOUDFLARE_API_TOKEN" …/user/tokens/verify` (same check
      as the shared `cf:token-check`, without including that file). Deploy and D1 tasks run as
      `fnox exec -- go run ./cmd/deploy …` / `fnox exec -- curl …`, reading `CLOUDFLARE_API_TOKEN` and
      `CLOUDFLARE_ACCOUNT_ID` from the environment. The token check is the precondition for every Phase 3 task.
- [ ] Read the account's Workers plan (free/paid) through the API with that token, and set the size gate
      (3 MB / 10 MB) from it.
- [ ] Deploy tool: a small **stdlib Go** program (`demos/workers/cmd/deploy`) that follows Direct Upload
      step by step: build the manifest (hash + size per file in `public/`), POST the `assets-upload-session`,
      upload the buckets (base64 multipart, upload JWT), then PUT the script with modules + metadata (compat
      date, bindings, completion JWT). Use Go rather than curl because the manifest/bucket loop is multi-step
      JSON.
- [ ] `demo:workers:deploy` → smoke-test the `*.workers.dev` URL with curl.
- [ ] Optional D1: create the database + apply `migrations/0001.sql` via the D1 REST API; add a D1 binding in
      metadata; D1 code behind `//go:build js && wasm` (the `cloudflare/d1` package is js-only), with a memory
      store for `go run .`. Deployed-only testing (constraint 4).

### Phase 4: Docs

- [ ] README: Stack table (workers-go v0.35.0, workerd), Demos section with run/serve/deploy commands.
- [ ] AGENTS.md: no wrangler/Node for Workers; `demos/workers` tasks; no package-level state in handlers
      that's expected to persist; TinyGo is the Workers target.

### Phase 5 (later, separate go-ahead): gsxui demo on Workers

- [ ] Check whether the gsx runtime + gsxui components build with TinyGo 0.42 and fit under 3 MB gzip.
      If yes, host `demos/gsxui` behind `workers.Serve` with `dist/` + `web/gsxui/` as static assets.

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

- Worker name? `*.workers.dev` or a custom domain? (Account settled: `CLOUDFLARE_ACCOUNT_ID` from fnox. The
  free/paid plan is read from the API in Phase 3.)
- Is Phase 1's optional upstream deploy wanted, or stay local until our demo?
- Do we want D1 at all in the first demo, given it's deployed-only without Node?

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
