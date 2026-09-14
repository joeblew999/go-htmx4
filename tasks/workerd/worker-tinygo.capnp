# workerd config for workers-go's worker-tinygo template (mise run upstream:workers-go:serve).
#
# The task copies this file next to build/ in .upstream/worker-tinygo, because workerd
# resolves `embed` paths relative to the config file. Modeled on workerd's samples/hello-wasm.
# Module names must match the relative imports in the files workers-assets-gen generates:
#   worker.mjs  → ./wasm_exec.js, ./runtime.mjs
#   runtime.mjs → ./app.wasm, cloudflare:sockets (workerd built-in)

using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [ (name = "main", worker = .worker) ],
  sockets = [ (name = "http", address = "127.0.0.1:8911", http = (), service = "main") ],
);

const worker :Workerd.Worker = (
  modules = [
    (name = "worker.mjs", esModule = embed "build/worker.mjs"),
    (name = "wasm_exec.js", esModule = embed "build/wasm_exec.js"),
    (name = "runtime.mjs", esModule = embed "build/runtime.mjs"),
    (name = "app.wasm", wasm = embed "build/app.wasm"),
  ],
  # Must not be newer than the workerd binary (1.20260911.1).
  compatibilityDate = "2026-09-11",
);
