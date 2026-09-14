# workerd config for Cloudflare's WebSocket Hibernation example (mise run upstream:cf:ws-hibernation).
# The task points the "do-storage" disk service at .upstream/ws-hibernation-storage via --directory-path.

using Workerd = import "/workerd/workerd.capnp";

const config :Workerd.Config = (
  services = [
    (name = "main", worker = .worker),
    (name = "do-storage", disk = (path = "do-storage", writable = true)),
  ],
  sockets = [ (name = "http", address = "127.0.0.1:8915", http = (), service = "main") ],
);

const worker :Workerd.Worker = (
  modules = [ (name = "ws-hibernation.mjs", esModule = embed "ws-hibernation.mjs") ],
  durableObjectNamespaces = [
    (className = "WebSocketHibernationServer", uniqueKey = "go-htmx4-upstream-ws-hibernation", enableSql = true),
  ],
  durableObjectStorage = (localDisk = "do-storage"),
  bindings = [
    (name = "WEBSOCKET_HIBERNATION_SERVER", durableObjectNamespace = "WebSocketHibernationServer"),
  ],
  # Must not be newer than the workerd binary (1.20260911.1). >= 2026-04-07 enables web_socket_auto_reply_to_close.
  compatibilityDate = "2026-09-11",
);
