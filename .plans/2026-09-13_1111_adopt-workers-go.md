# Adopt workers-go (deploy to Cloudflare Workers)

**Status:** proposed · **Created:** 2026-09-13 11:11

## Goal

Run the same `http.Handler` on [Cloudflare Workers](https://workers.cloudflare.com/) via
[syumai/workers-go](https://github.com/syumai/workers-go), while keeping the local
`go run .` server. One app, two entrypoints.

## What we're adopting

| | |
| --- | --- |
| **What** | Go module that serves an `http.Handler` on Workers. Go is compiled to `GOOS=js GOARCH=wasm`; a generated JS shim (`workers-assets-gen`) passes each fetch event to Go. Also wraps KV, R2, D1 (alpha), Cache API, Queues, Cron, TCP sockets, env vars, and Durable Object *stubs*. |
| **Version** | `v0.35.0` (2026-09-05). **Renamed** from `github.com/syumai/workers` in v0.35.0; the old path still forwards but is deprecated. Use `github.com/syumai/workers-go`. |
| **Maturity** | Self-described "experimental". About 1.1k stars, actively maintained. MIT. |
| **Precedent** | We already ship it in `irgo` (`docs-templ/main_cloudflare.go`, `examples/morpheus`) on `github.com/syumai/workers v0.33.0`. Follow the same layout. |

## Key constraints (they shape the design)

1. **No memory between requests.** The shim (`assets/common/worker.mjs`) creates a new
   `WebAssembly.Instance` and Go runtime **on every fetch**. Our `atomic.Int64` counter
   would reset on every request, so shared state has to live in a Cloudflare binding.
2. **Durable Objects are call-only.** workers-go can call DO stubs, but the DO class
   itself has to be written in JS (see `_examples/durable-object-counter/worker.mjs`).
3. **Bundle size.** Standard Go wasm for an irgo-sized app is about 2.4 MB compressed,
   under the free plan's 3 MB limit. Embedding assets into the wasm pushes it toward the
   limit, so serve static files with **Workers Static Assets** instead.
4. **TinyGo** gives much smaller builds and the README says TinyGo ≥ 0.42 can now build
   `net/http` (irgo's code comment saying otherwise predates this). Treat it as an
   optional later optimisation. Stay on standard Go first.
5. **Request cost.** `hx-trigger="every 1s"` is 86,400 requests/day per open tab, close
   to the free plan's 100k/day. Slow the poll on Workers, or move the clock to the client
   or SSE.
6. **Local fallback.** On non-js builds `workers.Serve` is a plain `net/http` server
   (PORT env, default 9900). Bindings only work under `wrangler dev`.

## Design

```
main.go              //go:build !js   → flag parsing, http.ListenAndServe (today's behaviour)
main_cloudflare.go   //go:build js && wasm → workers.Serve(app.Handler(store))
app/handler.go       routes + handlers, no platform imports
app/store.go         type CounterStore interface { Get(ctx) (int64, error); Incr(ctx) (int64, error) }
store_memory.go      //go:build !js   → atomic implementation (local)
store_d1.go          //go:build js && wasm → D1-backed implementation
wrangler.jsonc       main=build/worker.mjs, assets dir, d1_databases binding
```

**Counter store: D1** (recommended). An atomic
`UPDATE counters SET n = n + 1 WHERE id = 1 RETURNING n` gives exact counts, and
workers-go supports D1 through `database/sql`.
Rejected: **KV** (eventually consistent, so concurrent clicks lose increments) and
**Durable Object** (strongest option, but needs a JS class; revisit if we need
per-user or realtime state).

**Static assets:** the wrangler `assets` directory (`static/`) is served by Cloudflare
before the Worker, so the wasm stays small. Locally we keep `go:embed`.

## Plan

### Phase 0: Spike (scratch dir)
- [ ] `npm create cloudflare@latest -- --template github.com/syumai/workers-go/_templates/cloudflare/worker-go`
- [ ] Drop in our current handlers; `wrangler dev`; confirm htmx page, `POST /count`,
      `GET /time` work
- [ ] Confirm the counter resets per request (constraint 1)
- [ ] Measure compressed wasm size with and without `go:embed` templates
- [ ] Record findings below; go/no-go

### Phase 1: Split platform code
- [ ] Branch `adopt-workers-go`
- [ ] Move routes/handlers into `app/` behind `CounterStore`; `main.go` gets `//go:build !js`
- [ ] Add `store_memory.go` (local) and verify `mise run check` still passes
- [ ] Add `app` handler tests with `httptest` and the memory store

### Phase 2: Workers target
- [ ] `go get github.com/syumai/workers-go@v0.35.0` (pin exactly: the generated
      `wasm_exec.js` must match the module)
- [ ] `main_cloudflare.go` → `workers.Serve(app.Handler(d1Store))`
- [ ] `store_d1.go` + `migrations/0001_counters.sql`
- [ ] `wrangler.jsonc`: `main`, `compatibility_date`, `assets: { directory: "./static" }`,
      `d1_databases` binding, `build.command` pointing at the mise build task
- [ ] Choose how templates reach the wasm: keep `go:embed` if size allows, otherwise
      move the page shell into static assets
- [ ] Workers-only tweak: slow the `/time` poll, e.g. `every 10s`, set by a build-tagged const
- [ ] Add `build/` to `.gitignore`

### Phase 3: Tooling
- [ ] `mise.toml`
  - tools: `node` (for wrangler)
  - include the shared libs from `joeblew999/.github` (`tool-wrangler.toml`,
    `tool-cf.toml`, `tool-fnox.toml`, pinned to the current tag `v0.70.0`)
  - tasks:
    - `cf:build` = `go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=go && GOOS=js GOARCH=wasm go build -o build/app.wasm .`
    - `cf:dev` → `wrangler:dev`
    - `cf:deploy` → `wrangler:deploy`
    - `cf:db-create` / `cf:db-migrate` → `cf:d1-create` / `cf:d1-migrate`
- [ ] Secrets: `CLOUDFLARE_API_TOKEN` / `CLOUDFLARE_ACCOUNT_ID` via fnox keychain,
      same as the other joeblew999 CF repos. Never commit them.
- [ ] `check` also runs `GOOS=js GOARCH=wasm go vet ./...` so the wasm target can't rot

### Phase 4: Deploy & document
- [ ] Create the D1 database, apply the migration, `mise run cf:deploy`, smoke-test the
      `*.workers.dev` URL
- [ ] Update `README.md` (Deploy to Cloudflare section) and `AGENTS.md` (build tags, "no
      package-level state in handlers", cf tasks)
- [ ] PR → merge

## Interaction with the gsxui plan

See `2026-09-13_1109_adopt-gsxui.md`. The two plans are compatible: the gsx runtime is
stdlib-only and compiles to wasm, and the Vite `dist/` output can be the Workers
`assets` directory. Suggested order: **workers-go first** (small change, sets up the
`app/` split), then gsxui on top. If gsxui lands first, re-check wasm size, because
Tailwind merge and generated components add code.

## Risks

- **Experimental library, recent rename.** Pin the exact version and upgrade on purpose.
  Migrating `irgo` from `syumai/workers` to `workers-go` is a separate task.
- **Cold start per request** (new Go runtime each fetch) adds latency. Measure it in
  Phase 0. If it's unacceptable, TinyGo is the main fix.
- **D1 in workers-go is "alpha".** Keep the `CounterStore` interface narrow so switching
  to a Durable Object is cheap.
- **Wasm size creep** past 3 MB (free) or 10 MB (paid). Add a size check to `cf:build`.

## Open questions

- Which Cloudflare account and worker name? Custom domain or `*.workers.dev`?
- Free or paid plan? This decides the size and request budgets and whether the 1s poll
  can stay.
- Should the local server also use D1 (via `wrangler dev`) for parity, or keep the
  in-memory store?

## References

- https://github.com/syumai/workers-go (README, `_examples/`, `_templates/cloudflare/`)
- Shim source: `cmd/workers-assets-gen/assets/common/worker.mjs`
- Examples: `_examples/d1-blog-server`, `_examples/kv-counter`,
  `_examples/durable-object-counter`
- In-house precedent: `../irgo/docs-templ/main_cloudflare.go`,
  `../irgo/cmd/irgo/app_cloudflare_build.go`
- Shared mise task libs: `joeblew999/.github/tasks/{tool-wrangler,tool-cf,tool-fnox}.toml`

## Findings log

### 2026-09-13 11:20: wasm size of the current app (before any workers-go code)

Toolchain via mise: Go 1.27.1, TinyGo 0.42.0 (supports Go ≤ 1.27), binaryen/wasm-opt 132
(TinyGo needs `wasm-opt`; mise's aqua registry is stale at 121, so it's pinned as
`github:WebAssembly/binaryen`).

| Build | Raw | gzip -9 |
| --- | --- | --- |
| `GOOS=js GOARCH=wasm go build` | 17.5 MB | **4.1 MB** (over the 3 MB free limit) |
| `tinygo build -target wasm -no-debug` | 2.4 MB | **0.84 MB** |

The starter already exceeds the free-plan limit with standard Go, so **constraint 4 flips:
TinyGo is the default target, not a later optimisation.** Phase 0 still needs to confirm
TinyGo + workers-go at runtime (`_templates/cloudflare/worker-tinygo`, `-mode=tinygo`) and
that `html/template` / `embed` behave under TinyGo.
