# workerd config for workers-go's _examples/durable-object-counter (mise run upstream:workers-go:do).
#
# The task copies this file into the example's build/ dir (gitignored upstream); embed paths are
# relative to it. The example's own worker.mjs re-exports the Go Worker and defines the JS `Counter`
# Durable Object; wrangler.toml's `[durable_objects] COUNTER → Counter` and `new_sqlite_classes`
# become durableObjectNamespaces (enableSql) + a durableObjectNamespace binding here.

using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [
    (name = "main", worker = .worker),
    (name = "do-storage", disk = (path = "do-storage", writable = true)),
  ],
  sockets = [ (name = "http", address = "127.0.0.1:8914", http = (), service = "main") ],
);

const worker :Workerd.Worker = (
  modules = [
    # Module names mirror the relative imports: worker.mjs → ./build/worker.mjs → ./wasm_exec.js …
    (name = "worker.mjs", esModule = embed "../worker.mjs"),
    (name = "build/worker.mjs", esModule = embed "worker.mjs"),
    (name = "build/wasm_exec.js", esModule = embed "wasm_exec.js"),
    (name = "build/runtime.mjs", esModule = embed "runtime.mjs"),
    (name = "build/app.wasm", wasm = embed "app.wasm"),
  ],
  durableObjectNamespaces = [
    (className = "Counter", uniqueKey = "go-htmx4-upstream-do-counter", enableSql = true),
  ],
  durableObjectStorage = (localDisk = "do-storage"),
  bindings = [
    (name = "COUNTER", durableObjectNamespace = "Counter"),
  ],
  # Must not be newer than the workerd binary (1.20260911.1).
  compatibilityDate = "2026-09-11",
);
