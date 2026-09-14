# workerd config for demos/workers (mise run demo:workers:serve, run from demos/workers).
#
#   :8913 → assets-first (workerd/assets-first.mjs)
#             ├─ file in dist/site/ → disk service "public"         (stands in for Static Assets)
#             └─ everything else → "app": workerd/local-entry.mjs
#                                    ├─ /live/{topic} WebSocket → Room DO (room.mjs, hibernation)
#                                    └─ else → Go Worker (build/, TinyGo) with DB = LocalD1 shim
#
# Deployed, index.mjs is the main module and DB is real D1; workerd/ is local-only.
# Embed and disk paths are relative to this file. Module names mirror the relative imports.

using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [
    (name = "assets-first", worker = .assetsFirst),
    (name = "public", disk = "dist/site"),
    (name = "app", worker = .app),
    # Durable Object SQLite files (Room cache + LocalD1 board data); survives rebuilds.
    (name = "do-storage", disk = (path = ".workerd-state", writable = true)),
  ],
  sockets = [ (name = "http", address = "127.0.0.1:8913", http = (), service = "assets-first") ],
);

const assetsFirst :Workerd.Worker = (
  modules = [ (name = "assets-first.mjs", esModule = embed "workerd/assets-first.mjs") ],
  bindings = [
    (name = "ASSETS", service = "public"),
    (name = "APP", service = "app"),
  ],
  compatibilityDate = "2026-09-11",
);

const app :Workerd.Worker = (
  modules = [
    (name = "workerd/local-entry.mjs", esModule = embed "workerd/local-entry.mjs"),
    (name = "workerd/local-d1.mjs", esModule = embed "workerd/local-d1.mjs"),
    (name = "migrations/0001_board.sql", text = embed "migrations/0001_board.sql"),
    (name = "index.mjs", esModule = embed "index.mjs"),
    (name = "room.mjs", esModule = embed "room.mjs"),
    (name = "build/worker.mjs", esModule = embed "build/worker.mjs"),
    (name = "build/wasm_exec.js", esModule = embed "build/wasm_exec.js"),
    (name = "build/runtime.mjs", esModule = embed "build/runtime.mjs"),
    (name = "build/app.wasm", wasm = embed "build/app.wasm"),
  ],
  durableObjectNamespaces = [
    (className = "Room", uniqueKey = "go-htmx4-workers-demo-room", enableSql = true),
    (className = "LocalD1", uniqueKey = "go-htmx4-workers-demo-local-d1", enableSql = true),
  ],
  durableObjectStorage = (localDisk = "do-storage"),
  bindings = [
    (name = "DEMO_ENV", text = "workerd (local)"),
    (name = "ROOM", durableObjectNamespace = "Room"),
    (name = "LOCAL_D1", durableObjectNamespace = "LocalD1"),
  ],
  # Must not be newer than the workerd binary (1.20260911.1).
  compatibilityDate = "2026-09-11",
);
