# workerd config for demos/gsxui on Workers (mise run demo:gsxui:workers:serve, run from demos/gsxui).
#
#   :8918 → assets-first (../workers/workerd/assets-first.mjs, local-only stand-in for Static Assets)
#             ├─ file in build/assets/ (/static, /gsxui, /assets) → disk service
#             └─ everything else → Go Worker (build/worker, TinyGo)
#
# Embed and disk paths are relative to this file.

using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [
    (name = "assets-first", worker = .assetsFirst),
    (name = "assets", disk = "build/assets"),
    (name = "app", worker = .app),
  ],
  sockets = [ (name = "http", address = "127.0.0.1:8918", http = (), service = "assets-first") ],
);

const assetsFirst :Workerd.Worker = (
  modules = [ (name = "assets-first.mjs", esModule = embed "../workers/workerd/assets-first.mjs") ],
  bindings = [
    (name = "ASSETS", service = "assets"),
    (name = "APP", service = "app"),
  ],
  compatibilityDate = "2026-09-11",
);

const app :Workerd.Worker = (
  modules = [
    (name = "worker.mjs", esModule = embed "build/worker/worker.mjs"),
    (name = "wasm_exec.js", esModule = embed "build/worker/wasm_exec.js"),
    (name = "runtime.mjs", esModule = embed "build/worker/runtime.mjs"),
    (name = "app.wasm", wasm = embed "build/worker/app.wasm"),
  ],
  # Must not be newer than the workerd binary (1.20260911.1).
  compatibilityDate = "2026-09-11",
);
