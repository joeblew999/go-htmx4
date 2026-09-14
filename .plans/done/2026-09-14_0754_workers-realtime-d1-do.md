# Realtime on Workers: D1 + Durable Objects + htmx 4

**Status:** **Done** (phases 1–7: board, gsx rendering, presence; live at https://go-htmx4-workers-demo.gedw99.workers.dev/board) · **Created:** 2026-09-14 07:54 · **Revised:** 2026-09-14 08:10 · Builds on `2026-09-13_1111_adopt-workers-go.md`
(done: `demos/workers`, live at https://go-htmx4-workers-demo.gedw99.workers.dev)

## Goal

The pattern for a **real project on Cloudflare Workers**: server-rendered htmx 4 from Go (TinyGo, workers-go) with
**live updates to every connected browser**, at Cloudflare scale.

1. **Run the upstream references fully:** workers-go's Durable Object example, and Cloudflare's WebSocket Hibernation
   example, on workerd (Node-free).
2. **Then build our own demo:** a shared board where a change in one tab shows up in every tab immediately.

## Decision (2026-09-14)

- **Run on Workers** for Cloudflare's scale (not plain Go on a server).
- **D1 is the source of truth.** It's queryable, with migrations and point-in-time restore.
- **One Durable Object per topic handles fan-out only.** It holds connections, not data; losing it loses nothing.
- **WebSocket Hibernation + htmx 4 `hx-ws`**, not SSE (confirmed by you). The DO docs say "Billable Duration (GB-s)
  charges do not accrue during hibernation"; a DO holding SSE streams stays awake.
- **Extend `demos/workers`** (confirmed): same module, deploy tool, tasks and URL.
- **Design for 1,000 concurrent connections per topic** (confirmed). See "Designing for 1,000 per topic".
- **D1 can't push changes.** It has no change feed, and Queues event subscriptions don't cover D1 (sources: Access,
  Artifacts, Email Sending, R2, Super Slurper, Vectorize, Workers AI, Workers Builds, KV, Workflows). So whoever writes
  announces the change.

## Ground rules

Same as the workers-go plan, plus one addition (**confirmed 2026-09-14**):

- **MUST** use workers-go `v0.35.0`, **MUST** use TinyGo for everything that ships to Workers, **MUST NOT** use Node
  (no npm, wrangler, miniflare). Upstream docs only. Work on `main`. No Docker.
- **Every Cloudflare write (D1 create, deploys) needs an explicit OK.** Credentials via `fnox exec`.
- **JS only where Go can't go (confirmed):** the Durable Object class and a thin entry module. workers-go can only
  *call* DOs (the DO class is JS in upstream `_examples/durable-object-counter/worker.mjs`), and it has no WebSocket
  support (no `webSocket` handling anywhere in v0.35.0), so WebSocket upgrades can't pass through Go. This JS runs
  in workerd / Workers, not Node, with no bundler.

## Architecture

```
                    ┌──────────────── index.mjs (JS entry, deployed) ─────────────────┐
browser ─ hx-ws:connect /live/{topic} ─▶ env.ROOM.idFromName(topic) → Room DO (JS)
                                          ctx.acceptWebSocket(ws)  (hibernatable)
browser ─ GET /board, hx-post /board/… ─▶ Go Worker (TinyGo, build/worker.mjs default export)
                                          1. D1: UPDATE … RETURNING value, version   (truth)
                                          2. ROOM stub.Fetch(POST /publish, <html fragment>)
                                                 └▶ Room DO: for ws of ctx.getWebSockets(): ws.send(html)
                                          3. respond to the poster with the same fragment
browser ◀─ <div id="board" hx-swap-oob="true" data-version="42">…</div>  (all tabs)
```

- **Entry module:** `index.mjs` re-exports the Go Worker's default `fetch` for everything except `/live/*`, and
  exports `class Room` (same shape as upstream `_examples/durable-object-counter/worker.mjs`). Metadata `main_module`
  becomes `index.mjs`.
- **Resync, not an outbox:** every fragment carries `data-version`. On (re)connect the DO sends its last broadcast
  fragment (cache); the client re-fetches `/board` from Go (D1) only on a version gap, plus a slow periodic check. A
  failed publish only delays an update; the periodic check also catches D1 writes made outside the Worker (dashboard,
  REST API).
- **All app writes go through the Go Worker**, so every change gets published.
- **Scale knobs:** one DO per topic (`idFromName`), so hot topics shard by name. Idle connections hibernate. Broadcast
  cost grows with connections per topic. D1 is single-primary for writes.

## Designing for 1,000 per topic

| Concern | Limit / fact (source) | Design |
| --- | --- | --- |
| Connections per DO | DO limits page lists **no per-object WebSocket cap** | One `Room` DO per topic (`idFromName(topic)`) holds all ~1,000 hibernatable sockets |
| Requests per DO | "soft limit of **1,000 requests per second**" per object (DO limits); connects, publishes and incoming WS messages all count | **Browsers never send over the socket**: writes are `hx-post` to Go. The socket is receive-only. |
| Heartbeats waking the DO | `setWebSocketAutoResponse` replies "without waking WebSockets in hibernation" (DO state API) | Set a ping/pong auto-response in the constructor; no app-level heartbeats |
| Broadcast fan-out cost | Each broadcast = ~1,000 `ws.send()` in one invocation; CPU 30 s per invocation, reset per request (DO limits) | **Coalesce by version:** the DO keeps only the latest pending fragment and flushes at most every ~200 ms, so a busy topic sends ≤ 5 broadcasts/s whatever the write rate. Stale versions are dropped. |
| Fragment size | Only received messages are capped (32 MiB); outbound size drives cost/latency | Keep pushed fragments small (target < 2 KB): OOB-swap just the changed element, not the page |
| Reconnect storms (deploys, network blips) | `hx-ws` reconnects with `ws.reconnectDelay` / `ws.reconnectMaxAttempts`; `htmx:ws:after:connection` fires on each connect | Reconnect delay with jitter; on connect the DO immediately sends its **last broadcast fragment** (a cache, not truth), so 1,000 reconnects don't become 1,000 D1 reads. The client resyncs from Go/D1 only on a version gap and on a slow periodic check (~60 s ± jitter). |
| Write path | D1 is single-primary for writes | Writes are small `UPDATE … RETURNING version`; the Go handler publishes once per write; the DO coalesces |
| Per-connection data | `serializeAttachment` survives hibernation (DO WebSockets docs); tags ≤ 10 per socket | Attach only `{connectedAt, lastVersion}`; use tags only if a topic later needs sub-views |

**Verify in the spikes:** whether deploying new code disconnects all sockets (plan for it: that's the reconnect storm
case), and real broadcast latency p50/p95 with 1,000 sockets.

## Unknowns to spike first (they decide go/no-go)

1. **D1 from TinyGo at runtime.** ✅ **Works** (Phase 2). workers-go's `cloudflare/d1` is a `database/sql` driver, and `database/sql` uses
   reflection for `Scan`. `html/template` compiled but panicked under TinyGo, so this may too. **D1 is deployed-only**
   (no D1 in workerd without miniflare), so the spike needs a D1 database = a Cloudflare write (OK needed).
   Fallbacks: a thin `syscall/js` wrapper over the D1 binding (`prepare().bind().first()`), skipping `database/sql`;
   or DO SQLite as the store.
2. **DO stub call from TinyGo.** ✅ Works (Phase 1). Upstream's DO example builds with `-mode=go`; confirm `stub.Fetch` under TinyGo on
   workerd.
3. **Hibernatable WebSockets on workerd locally:** ✅ Works (Phase 1). `durableObjectNamespaces = [(className = "Room", uniqueKey = …,
   enableSql = true)]` + `durableObjectStorage = (localDisk = …)` (both in `workerd.capnp`).
4. **WebSocket smoke + load tooling:** ✅ Decided: `golang.org/x/net/websocket` (worked for 1,000 sockets in Phase 1). Go's stdlib has no WebSocket client. Plan: a local-only Go tool
   (`demos/workers/cmd/wsload`) using `golang.org/x/net/websocket` (first external dependency, local-only, never in the
   wasm) that opens N sockets, posts one change, and reports how many received it plus p50/p95 latency. The same tool
   covers the 2-client smoke and the 1,000-socket load test.
5. **Upload metadata for DO + D1:** D1 binding ✅ verified in Phase 2 (`-d1 DB=<uuid>` in `cmd/deploy`); DO binding + migration still to verify in Phase 4. bindings `{"type":"durable_object_namespace","name":"ROOM","class_name":"Room"}` and
   `{"type":"d1","name":"DB","id":"<uuid>"}` (multipart metadata docs); `migrations` is listed as "array[object]" for
   immediate deploys. Confirm the exact migration shape (`new_sqlite_classes: ["Room"]`) against the API reference.
6. **`hx-ws` details:** vendor `dist/ext/hx-ws.js` from htmx 4.0.0; resync hooks on `htmx:ws:after:connection` (hx-ws
   docs); confirm pushed `hx-swap-oob` fragments apply.

## Plan

### Phase 1: Upstream references, local (Node-free)

- [x] `upstream:workers-go:do`: build `_examples/durable-object-counter` with **TinyGo**; workerd config with its JS
      `Counter` class, `enableSql`, `localDisk` storage. Check `/increment` → count survives across requests
      (unlike Go memory), via the Go stub. Answers unknowns 2 and 3. — 0→1→2→3→2, and 2 after a workerd restart
- [x] Cloudflare's WebSocket Hibernation example (DO WebSockets docs) as a plain JS module on workerd: two clients
      connect, one sends, both receive, and `getWebSockets().length` is right. Answers unknown 3 for WebSockets and
      settles the smoke tooling (unknown 4). — the docs example echoes to the sender only; `connections: 2` and `1000`.
      Broadcast measured with a scratch variant (see Findings).
- [x] Record findings. **Stop for review.**

### Phase 2: D1 under TinyGo spike (⚠ creates Cloudflare resources, needs OK)

- [x] Create D1 `go-htmx4-spike` via REST (`POST /accounts/{id}/d1/database`), apply a one-table migration via
      `/query`.
- [x] Deploy a throwaway Worker `go-htmx4-d1-spike` (our `cmd/deploy` + a `d1` binding): `database/sql` insert, select,
      `RETURNING`, `Scan` into int/string/time.
- [x] If it panics: try the `syscall/js` D1 wrapper; record which works. — no panic; **both** work
- [x] Clean up the spike Worker and database (with your OK). **Stop for review.** — both deleted, URL 404

### Phase 3: Our demo, local (extend `demos/workers`)

- [x] `index.mjs` entry + `room.mjs` `Room` DO (hibernation API: `acceptWebSocket`, auto-response ping/pong, coalesced
      flush, last-fragment cache on connect), vendored `hx-ws.js`.
- [x] Go: `/board` page (current state from the store), `hx-post /board/incr` etc.; write → publish via `ROOM` stub. — `/board`, `POST /board/add` (±1), `POST /board/note`
- [x] Store behind a small Go interface: **D1** implementation (deployed); for **local workerd**, where D1 doesn't exist,
      a stand-in using a `Store` DO with SQLite (JS) or the Phase 2 wrapper against it. Log the local/production gap. — a **D1-shaped shim** (`workerd/local-d1.mjs`), so Go uses the same `database/sql` + d1 driver code locally
- [x] workerd config: entry worker + Room DO + local store; `demo:workers:smoke` gains a WebSocket check via
      `cmd/wsload` (2 sockets: one post, both see the new `data-version`), plus a local 1,000-socket run on workerd.
- [x] TinyGo size gate, httptest for handlers, workerd smoke for the runtime (same rules as `demos/workers`).
- [x] ⚠ Browser check of the real `hx-ws` client (connect, OOB swap, version guard): needs your OK. `wsload` covers the server side with raw sockets only. — done live 2026-09-14 (see Findings)

### Phase 4: Deploy (⚠ needs OK)

- [x] Create the production D1 database + migration via REST; extend `cmd/deploy`: multi-module entry (`index.mjs`,
      `room.mjs`, `build/*`), `ROOM` DO binding + migration, `DB` D1 binding.
- [x] Deploy; remote smoke with two live WebSocket clients + resync after reconnect.
- [x] **Load test at design size:** `cmd/wsload` with 1,000 sockets on one topic against the live URL; bursts of
      writes; record delivered/1,000, p50/p95 latency, coalesced broadcasts/s, and a redeploy-during-load reconnect storm.
      — 1,000/1,000 once; repeats from this network were capped by the client side (see Findings); redeploy measured with 200

### Phase 5: Docs

- [x] README (Demos, "Realtime on Workers" section), AGENTS.md (JS allowed only for DO/entry; writes go through Go;
      resync-by-version rule; D1 deployed-only).

### Phase 6 (done 2026-09-14): render the board with gsx

gsx + gsxui were verified under TinyGo on Workers (byte-identical output, `adopt-workers-go` plan Phase 5a/5d), so the
hand-escaped strings in `board.go` / `board.html` can become gsx components.

- [x] Add gsx v0.1.0 to `demos/workers` (`go get -tool`), `gsx.toml` (htmx URL preset + `hx-action`), `views/board.gsx`:
      `BoardPage(topic, board)`, `Board(board)` (the OOB `#board` fragment with `data-version`), notes list. — `board.gsx`
      (package main): `BoardPage`, `BoardFragment`; `gsx.toml` adds `hx-ws:connect` to URL attrs; `board.html` removed
- [x] Keep the wire format identical: a test asserts the gsx fragment equals today's `renderBoard` output (or differs
      only in whitespace), so the Room, the version guard and `wsload` need no change. — `TestBoardFragmentWireFormat`:
      **byte-identical** to the old renderer (escaping, Unicode, spacing, empty notes)
- [x] `demo:workers:build` runs `go tool gsx generate` first; size gate; `demo:workers:test` + `load` + browser check.
      — new `demo:workers:generate` (build/run depend on it), `gsx fmt -l` in test; from no generated code: TinyGo
      **1,203,138 B raw / 445,677 B gzip** (+65 KB), smoke 13/13, `go test` ok; two-tab browser check on local workerd
      9/9 (+1 push 71 ms, escaped note, form reset via `js` literal, version guard, no console errors).
- [x] Deploy (OK'd 2026-09-14): `mise run demo:workers:deploy` (local test green; migration v1 already applied, assets
      unchanged, script uploaded). Live `demo:workers:smoke-remote` 14/14 (2-socket push, pong, cache); two-tab browser
      check on https://go-htmx4-workers-demo.gedw99.workers.dev/board 9/9 (+1 push 231 ms, escaped note, form reset,
      version guard, no console errors).
- [-] *Future idea, not pursued.* Optional: gsxui components (card, button, input) for the board UI, with Tailwind via the standalone CLI.

### Phase 7 (done + deployed 2026-09-14): presence per topic

The Room already knows its sockets, so it can push "N online" without a D1 write.

- [x] Room: on connect and on `webSocketClose`/`webSocketError`, queue a `<span id="presence" hx-swap-oob="true">N
      online</span>` broadcast through the same ≤ 5/s coalescing (latest count wins); count = `ctx.getWebSockets().length`.
      — `presenceChanged` / `broadcastPresence` / shared `scheduleFlush` alarm. **Bug found by `wsload -n 1`:** filtering
      on `readyState === OPEN` skipped the socket being accepted (not OPEN yet while its connect runs), so a lone visitor
      stayed on "connecting…"; now counts `readyState < CLOSING`.
- [x] Page: a `#presence` element inside the `hx-ws:connect` container; no version guard needed (latest count wins).
      — in `BoardPage`'s header (OOB swaps target by id); the guard ignores it (no `data-version`).
- [x] Cost check at design size: 1,000 sockets joining = at most 5 presence broadcasts/s, not 1,000 × 1,000 sends.
      Measure with `wsload` (connect storm → count converges to 1,000; disconnect → 0). — local workerd: n=1 → 1;
      n=2 → 2 then 1; n=1000 → every socket sees 1000, **socket 0 received 1 presence update for the whole 1,000-socket
      join**, closing 500 → 500. `wsload` fails on wrong counts, so `demo:workers:smoke` (and CI) cover it. Browser check
      11/11: both tabs "2 online", tab A "1 online" after tab B leaves, no console errors.
- [x] ⚠ Deploy presence (needs OK), then remote smoke + live 1,000-socket presence run + browser check. — OK'd; live:
      smoke-remote 14/14 (2 sockets: 2 → 1 online); `wsload -n 1000 -dialers 20` from the hotspot: 1000/1000 connected
      in 11.3 s, every socket sees **1000 online**, closing 500 → **500**, push to 1000/1000, late joiner cached; socket 0
      received **52 presence updates over the 11.6 s join (~4.5/s)**, i.e. the ≤ 5/s coalescing holds when joins trickle
      in; two-tab browser check 11/11 (2 online both, 1 online after tab B leaves, no console errors).
- [-] *Future idea, not pursued.* A second topic *type* (e.g. a shared list or poll) reusing Room + version guard, if still wanted after presence.

## Risks

- **Three experimental layers:** workers-go (self-described experimental), its D1 driver ("alpha"), and TinyGo's
  stdlib gaps. Phase 2 exists to fail fast.
- **Two languages:** Go for pages and writes, JS for the DO and entry. Keep the JS tiny and logic-free (fan-out only).
- **Local ≠ production for D1:** no local D1 without Node. Keep the store interface narrow and smoke-test D1 remotely.
- **Cost model:** broadcasts wake the DO; very chatty topics need batching or throttling. D1 row limits are enforced on
  Free (we're on Workers Paid).
- **Out-of-band D1 writes** aren't pushed; only the periodic resync sees them.

## Open questions

- Production D1 database name for Phase 4 (proposal: `go-htmx4-workers-demo`, matching the Worker)?
- Resolved 2026-09-14: spike names + cleanup **yes** (done); `golang.org/x/net/websocket` for local tooling **yes**.
- Resolved 2026-09-14: JS for DO + entry **yes**; extend `demos/workers` **yes**; WebSocket Hibernation **yes**;
  design size **1,000 per topic**.

## References

- workers-go v0.35.0: `cloudflare/dostub.go` (`NewDurableObjectNamespace`, `IdFromName`, `Get`, `Stub.Fetch`),
  `cloudflare/d1`, `_examples/durable-object-counter`, `_examples/d1-blog-server`, `internal/jshttp/responsewriter.go`
- Cloudflare: Durable Objects WebSockets (Hibernation API), DO migrations (`new_sqlite_classes`), DO limits, multipart
  upload metadata (`d1`, `durable_object_namespace`, `migrations`), D1 REST API, Queues event subscriptions (no D1 source)
- workerd: `src/workerd/server/workerd.capnp` (`durableObjectNamespaces`, `enableSql`, `durableObjectStorage.localDisk`)
- htmx 4: https://four.htmx.org/extensions/hx-ws, https://four.htmx.org/extensions/hx-sse, `<hx-partial>`, `hx-swap-oob`

## Findings log

### 2026-09-14 11:20: 1,000 sockets + 50-write burst, live (the run that failed earlier)

Same Mac, same iPhone hotspot (`172.20.10.x`), dialing 20 at a time instead of 50:
`wsload -base https://go-htmx4-workers-demo.gedw99.workers.dev -n 1000 -writes 50 -dialers 20 -topic load-burst`.

| Connected | 50 writes | Delivered newest (v50) | Broadcasts to a socket | Latency from first POST | Ping | Late joiner |
| --- | --- | --- | --- | --- | --- | --- |
| **1000/1000** in 9.2 s, 0 dial errors | 3.4 s | **1000/1000** | **6** (coalesced) | p50 3.44 s, p95 3.58 s (≈ time the 50 writes took) | pong ✓ | cached v50 ✓ |

The earlier 897/960 results were the client network's dial rate (50 concurrent handshakes right after another 1,000),
not the Room: with gentler dialing, 1,000 sockets on one topic take a burst and all converge on the newest version.
The "repeat 1,000-socket bursts" to-do is done.

### 2026-09-14 10:50: browser check, cleanup, docs

- **Browser check (headless Chrome via chromedp, a scratchpad Go program; live URL, two tabs on one topic):** both tabs open
  the `hx-ws` socket; `htmx.config.ws` = `{"delay":"2s","jitter":0.5}` from the `<meta>`; **+1 in tab A → tab B updates
  in 215–231 ms** (5 runs); a note with `<img onerror>` markup from B shows in A as text (0 `img` elements); the note form
  resets; a stale v1 fragment sent through `htmx.swap` is **cancelled by the version guard** (board stays at the current
  version); **no console errors or exceptions** in either tab. Screenshot checked.
- **One unexplained miss:** in the very first run (first write to a fresh topic shortly after a deploy), tab A's +1 was
  stored (value 1 seen later) but neither tab showed it within the 5 s window; the following note push arrived normally.
  Not reproduced in 5 reruns with request logging. Most likely a slow cold write (Worker isolate + first D1 query), but
  not proven. Watch first-write latency after deploys.
- **Cleanup of live test data:** deleted 6 test notes and reset 9 test topics to `value 0` **with `version + 1`**, never
  lower (see invariant below). `lobby` (5 notes, v5) was left untouched: none of our tests use it.
- **Invariant found while cleaning:** the Room drops publishes whose version isn't above its cached one, so deleting a
  `board` row (versions restart at 1) would silence that topic's pushes. `board.version` must only increase; recorded in
  AGENTS.md.
- **Docs:** README Stack (D1, Durable Objects + hx-ws), Demos (board, load task, upstream DO/hibernation examples), a "Live
  updates on Workers" section; AGENTS.md board rules (write path, version invariant, no transactions, where JS is allowed,
  migrations, DO migration tags, tests), ports, structure.

### 2026-09-14 10:30: Phase 4 results (deployed)

**Live:** https://go-htmx4-workers-demo.gedw99.workers.dev/board. `mise run demo:workers:deploy` runs the local test +
token check, then:
`cmd/deploy -main index.mjs -module room.mjs -d1 DB=go-htmx4-workers-demo -d1-create -migrations migrations -do ROOM=Room
-migration-tag v1 -new-sqlite-class Room -var DEMO_ENV=…`.

- **Deploy tool (new):** modules named `index.mjs`, `room.mjs`, `build/worker.mjs`, `build/wasm_exec.js`,
  `build/runtime.mjs`, `build/app.wasm`; D1 looked up by name (`GET …/d1/database?name=`), created if missing; migrations
  applied once each via `/query`, tracked in `_migrations`; DO binding `{"type":"durable_object_namespace","name":"ROOM",
  "class_name":"Room"}`; migration sent as a **single object** `{"new_tag":"v1","new_sqlite_classes":["Room"]}` (shape
  from cloudflare-go `ScriptUpdateParamsMetadataMigrations`; the docs page just says "array[object]") and **only if** the
  script's `migration_tag` (from the scripts list) differs. The redeploy logged "migration v1 already applied".
- **D1 `/query` runs a multi-statement file:** `0001_board.sql` (2 tables + index) created `board`, `note`,
  `note_topic_id` in one call (verified via `sqlite_master`).
- **Remote smoke:** all 13 HTTP checks + `wsload -n 2`: live push, `ping` → `pong` from `setWebSocketAutoResponse` in
  production, late joiner gets the cached fragment.
- **Load, from this Mac on an iPhone Personal Hotspot (`172.20.10.x`)**, so client network numbers, not Cloudflare's:

  | Run | Connected | Delivered | Latency after POST start | Broadcasts to a socket | Late joiner |
  | --- | --- | --- | --- | --- | --- |
  | 1,000 sockets, 1 write | **1000/1000** in 4.1 s | **1000/1000** | p50 386 ms, p95 404 ms (POST itself 405 ms: the push lands before the poster's response) | 1 | ✓ |
  | 300 sockets, 50 concurrent writes | 300/300 in 10.2 s | 300/300 got v50 | ≈ 2.1 s = time for the 50 writes | **5** (coalesced, ≤ 5/s) | ✓ |
  | 1,000 sockets, 50 writes (2 tries, right after run 1) | 897 and 960/1000 | — | — | — | — |

  The failed dials were `connection reset by peer` during the handshake, and got worse with each rapid repeat. Most
  likely the hotspot's NAT or per-IP edge limits after ~2–3k connections in a few minutes, not the Room (run 1 delivered
  to all 1,000). **To do from a better network (or several clients):** repeat 1,000-socket bursts.
- **A deploy drops every socket, all at once:** 200 held sockets went 200 → 0 within 0.1 s as the new script went live
  (`wsload -hold 75s` + a redeploy at t+9 s). DO storage survives: a new socket then got the cached fragment.
- **Fix applied from that measurement:** hx-ws reconnects after `ws.reconnectDelay` 500 ms ±30% by default, so 1,000 tabs
  would return within ~0.3 s, about 3k requests/s at one Room (soft limit 1,000/s). `board.html` now sets
  `<meta name="htmx-config" content="ws.reconnectDelay:2s ws.reconnectJitter:0.5">`, spreading reconnects over 1–3 s
  (~500/s for 1,000 tabs). Redeployed; the page serves it; smoke green.
- **Production data** now contains test topics `smoke`, `load`, `burst300`, `load-burst`, `redeploy` (small rows in
  `board`/`note`). Harmless; can be cleared with one `DELETE` via `/query` if wanted.
- Still not done: **browser check** of the real `hx-ws` client (needs OK); repeated 1,000-socket bursts from a better
  network.

### 2026-09-14 09:40: Phase 3 results (local only, nothing deployed)

**The shared board runs on workerd** (`mise run demo:workers:serve` → http://localhost:8913/board) and under `go run .`
(memory store, no push). `mise run demo:workers:test` passes (TinyGo build + workerd smoke + go test);
`mise run demo:workers:load` passes.

What was added to `demos/workers`:

| File | Role | Deployed? |
| --- | --- | --- |
| `board.go`, `board.html`, `store_sql.go` | handlers, page, `database/sql` store (UPSERT … RETURNING) | yes (Go/TinyGo) |
| `store_mem.go`, `platform_other.go` | memory store + no-op publish for `go run .` / tests | no (`!js`) |
| `platform_js.go` | `sql.Open("d1","DB")`, `publish` via `ROOM` stub | yes |
| `index.mjs`, `room.mjs` | entry (`/live/*` → Room), Room DO (hibernation, auto ping/pong, 200 ms coalescing via alarm, `last` cache sent on connect) | yes |
| `workerd/local-entry.mjs`, `workerd/local-d1.mjs` | local entry injecting a D1-shaped `DB` over a `LocalD1` DO (SQLite), migrations applied on first use | no |
| `migrations/0001_board.sql` | `board(topic, value, version)`, `note(…)` + index | yes (Phase 4 applies it to D1) |
| `public/hx-ws.js`, `.htmx-version` | htmx 4.0.0 `hx-ws` extension (integrity-checked, `demo:workers:vendor`) | asset |
| `cmd/wsload` | WebSocket smoke/load tool (`golang.org/x/net/websocket` v0.59.0) | no |

- **Same Go code path locally and in production.** workers-go's D1 driver only calls `prepare().bind().run()` (reads
  `meta.changes` / `meta.last_row_id`) and `.raw({columnNames:true})`. `local-d1.mjs` implements exactly that over
  `ctx.storage.sql.exec` via DO RPC, so `sqlStore` runs unchanged on workerd. Verified: UPSERT … RETURNING versions
  1→2→3, notes insert, `LIMIT` query, escaping.
- **Version guard for out-of-order swaps:** htmx 4 routes both HTTP responses and `hx-ws` messages through
  `htmx.swap(ctx)`, which fires cancelable `htmx:before:swap` with `ctx.text` (read in `dist/htmx.js` 4.0.0, and
  `hx-ws.js` calls `htmx.swap`). `board.html` cancels any swap whose `data-version` is below the one on screen.
  **Not yet exercised in a browser.**
- **wsload results on local workerd** (same Mac for client and runtime, so a lower bound on server cost):

  | Scenario | Delivered | Broadcasts to a socket | Latency after POST | Late joiner (cache) | Ping |
  | --- | --- | --- | --- | --- | --- |
  | 2 sockets, 1 write | 2/2 | 1 | 26 ms | ✓ | pong ✓ |
  | 1,000 sockets, 1 write | 1000/1000 | 1 | p50 26–44 ms, p95 30–50 ms | ✓ | pong ✓ |
  | 1,000 sockets, 50 concurrent writes | 1000/1000 got v50 | **2** (coalesced) | ≈ 0.5 s = time for the 50 writes | ✓ | pong ✓ |

  The 50 writes serialize through the single local `LocalD1` object (~10 ms each). Real D1 will differ.
- TinyGo wasm grew from 360 KB to **380 KB gzip** with `database/sql` + the d1 driver.
- Design change: the plan's "slow periodic resync (~60 s)" is **not implemented**. Each pushed fragment is the whole
  `#board`, the Room re-sends its cache on (re)connect, and a periodic GET would race with pushes. Revisit only if
  out-of-band D1 writes matter.
- Local state lives in `demos/workers/.workerd-state/` (gitignored), so it survives rebuilds.

### 2026-09-14 09:10: Phase 2 results (D1 spike; resources created, tested, deleted)

- **Created (approved):** D1 `go-htmx4-spike` (`POST /accounts/{id}/d1/database {"name"}` → `result.uuid`), schema via
  `POST …/d1/database/{uuid}/query {"sql"}` (one statement per call; tables `board`, `log`). Worker `go-htmx4-d1-spike`
  deployed with `cmd/deploy -assets "" -d1 DB=<uuid>` (new flags; binding `{"type":"d1","name":"DB","id":…}` accepted).
- **workers-go's `database/sql` D1 driver works under TinyGo 0.42 at runtime**, unlike `html/template`. Spike build:
  343,389 B gzip.

  | Route | Result on `tinygo js/wasm` |
  | --- | --- |
  | `sql.Open("d1", "DB")` | ok |
  | `QueryRow(… WHERE id = ?).Scan(&int64, &int64)` | `value=0 version=0` |
  | `UPDATE … RETURNING value, version` → `Scan` | `1 1`, then `2 2` |
  | `Exec(INSERT … VALUES (?))` → `RowsAffected`, `LastInsertId` | `1`, `1` / `2` (no errors) |
  | `Query` + `rows.Next` + `Scan(&int64, &string, &string, &sql.NullString)` | 2 rows; `created_at` as text; NULL → `{"" false}` |
  | `db.Begin()` | `d1: transaction is not currently supported` (D1 has no interactive transactions; use single statements / `RETURNING`) |
  | Fallback: `cloudflare.GetBinding("DB").prepare().bind().first()` + a Go promise `await` | `value=2 version=2`: also works |

- **Atomic increments under concurrency:** 50 parallel `UPDATE board SET value = value + 1, version = version + 1 …
  RETURNING` → version 2 → **52**, **50 distinct versions**, 0 errors. So `RETURNING version` is a safe source for
  `data-version`.
- **Latency** (this Mac → Cloudflare, cold-ish, 50 at once): p50 0.28 s, p95 0.94 s per write request (fresh Go runtime +
  D1 write + network). Sequential single requests were 0.14–0.27 s. Worth measuring from the DO/Worker side in Phase 4.
- **Gotcha (tooling):** SQL containing double quotes broke a `jq` program built by string interpolation, so one
  statement didn't run. Pass SQL with `jq --rawfile`/`--arg` and SQL-standard single quotes.
- **Cleanup (approved):** `DELETE …/workers/scripts/go-htmx4-d1-spike` and `DELETE …/d1/database/{uuid}` both
  `success: true`; neither listed afterwards; spike URL 404; `go-htmx4-workers-demo` untouched.

### 2026-09-14 08:05: Phase 1 results (local only, nothing deployed or created)

New tasks: `upstream:workers-go:do` (:8914) and `upstream:cf:ws-hibernation` (:8915); configs in `tasks/workerd/`
(`do-counter.capnp`, `ws-hibernation.capnp` + `ws-hibernation.mjs`, the docs example verbatim plus a 6-line entry).

- **Go → Durable Object under TinyGo works (unknown 2 ✓).** `_examples/durable-object-counter` built with TinyGo
  (325,786 B gzip) instead of its `-mode=go` Makefile. `cloudflare.NewDurableObjectNamespace("COUNTER")` → `IdFromName`
  → `Get` → `stub.Fetch` reaches the JS `Counter`: `/` 0, `/increment` ×3 → 3, `/decrement` → 2.
- **SQLite-backed DOs on plain workerd (unknown 3 ✓).** `durableObjectNamespaces (enableSql = true)` +
  `durableObjectStorage = (localDisk = "do-storage")` + a writable disk service. State is written to
  `…/<uniqueKey>/<id>.sqlite` and **survived a workerd restart** (still 2). No miniflare, no Node.
- **Module naming for a JS entry + Go build:** the example's `worker.mjs` does `export { default } from "./build/worker.mjs"`,
  so modules are named `worker.mjs`, `build/worker.mjs`, `build/wasm_exec.js`, `build/runtime.mjs`, `build/app.wasm`.
  Relative imports resolve against those names. This is the layout Phase 3's `index.mjs` and the deploy tool need.
- **Hibernation API on workerd (unknown 3, WebSockets ✓):** Cloudflare's example as-is. `acceptWebSocket` +
  `getWebSockets()` reported `connections: 2`, then `connections: 1000` with 1,000 sockets on one DO.
  `setWebSocketAutoResponse(new WebSocketRequestResponsePair("ping","pong"))` in the constructor answers `ping` → `pong`.
- **Broadcast at design size (scratch variant, not committed):** one `Room` DO, 1,000 sockets, `POST /publish` with a
  1,559 B OOB fragment, `for (ws of getWebSockets()) ws.send(html)`:

  | Run | DO send loop | Delivered | Client p50 | p95 | max |
  | --- | --- | --- | --- | --- | --- |
  | 1 | 4 ms | 1000/1000 | 15.9 ms | 18.4 ms | 18.5 ms |
  | 2 | 4 ms | 1000/1000 | 9.9 ms | 12.5 ms | 12.6 ms |
  | 3 | 3 ms | 1000/1000 | 10.7 ms | 13.2 ms | 13.4 ms |

  Loopback on one Mac (client and runtime share CPU), so it's a lower bound on server cost, not production latency.
  `sent: 1001` in runs 2–3 means a socket from the previous (exited) client run was still listed; `send` to it was
  caught. Broadcast code must tolerate dead sockets.
- **WebSocket tooling (unknown 4):** `golang.org/x/net/websocket` (v0.59.0) worked for 2 and 1,000 sockets in a
  scratchpad probe (`ulimit -n 8192`). Still your call whether it becomes a repo dependency for `cmd/wsload`.
- **Gotcha:** a plain HTTP request to a DO that returns a WebSocket fails with "Worker tried to return a WebSocket in a
  response to a request which did not contain the header \"Upgrade: websocket\"". The entry must only forward real
  upgrades to `/live/*`, which the hibernation example's entry already checks.
- Still open for later phases: D1 under TinyGo (Phase 2), whether a deploy disconnects sockets, production latency.

### 2026-09-14 07:54: research for this plan (nothing built)

- Go response path in workers-go is `io.Pipe` → `ReadableStream` (`internal/jshttp/responsewriter.go`), so Go can stream
  a response (`Flush` is a no-op but the pipe is unbuffered). There's no WebSocket code anywhere in v0.35.0.
- `cloudflare/dostub.go` only calls existing DOs; the DO class itself is JS in the upstream example.
- htmx 4.0.0 ships `dist/ext/hx-ws.js` (persistent `hx-ws:connect`, `hx-ws:send` as JSON, `hx-swap-oob` /
  `<hx-partial>` in pushed messages) and `dist/ext/hx-sse.js` (`hx-sse:connect`, `Last-Event-ID` on reconnect).
- workerd supports SQLite-backed DOs locally (`enableSql`, `localDisk`), with no miniflare.
- Account is Workers Paid (checked 2026-09-13).
