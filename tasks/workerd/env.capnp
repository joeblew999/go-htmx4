# workerd config for workers-go's _examples/env (mise run upstream:workers-go:env).
#
# The task copies this file into the example's build/ dir (embed paths are relative to it).
# wrangler.toml's `[vars] MY_ENV = 'my env value'` becomes a workerd text binding.

using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [ (name = "main", worker = .worker) ],
  sockets = [ (name = "http", address = "127.0.0.1:8912", http = (), service = "main") ],
);

const worker :Workerd.Worker = (
  modules = [
    (name = "worker.mjs", esModule = embed "worker.mjs"),
    (name = "wasm_exec.js", esModule = embed "wasm_exec.js"),
    (name = "runtime.mjs", esModule = embed "runtime.mjs"),
    (name = "app.wasm", wasm = embed "app.wasm"),
  ],
  bindings = [
    (name = "MY_ENV", text = "my env value"),
  ],
  # Must not be newer than the workerd binary (1.20260911.1).
  compatibilityDate = "2026-09-11",
);
