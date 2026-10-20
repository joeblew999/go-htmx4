// Worker entry (deployed). WebSocket upgrades on /live/{topic} go straight to that topic's Room
// Durable Object; everything else goes to the Go Worker (build/), run by charter's go.mjs, which keeps
// Go runtimes alive between requests and starts two while the module loads (charter go/lib/worker;
// copied into build/ by the build, from the version go.mod requires).

import * as intl from "./intl.mjs";
import { goWorker } from "./build/go.mjs";

// The app's Intl formatter, for Go (kit/i18n Format): the same code the browser runs (static/intl.js).
globalThis.goIntl = intl;

export { Room } from "./room.mjs";

const go = goWorker();
// Two runtimes, each warmed with a request that touches no binding.
await go.warm({ paths: ["/healthz"] });

// Same rules as kit/live (TopicPattern, LocalePattern); TestWorkerJSMatchesKitLive checks.
const TOPIC = /^[a-z0-9-]{1,32}$/;
const LOCALE = /^[a-z]{2,3}(-[a-z0-9]{2,8}){0,3}$/;

export default {
  async fetch(request, env, ctx) {
    const { pathname, searchParams } = new URL(request.url);
    if (pathname.startsWith("/live/")) {
      const topic = pathname.slice("/live/".length);
      if (!TOPIC.test(topic)) return new Response("bad topic", { status: 400 });
      const locale = searchParams.get("locale");
      if (locale !== null && !LOCALE.test(locale)) return new Response("bad locale", { status: 400 });
      if (request.headers.get("Upgrade") !== "websocket") {
        return new Response("expected a WebSocket upgrade", { status: 426 });
      }
      return env.ROOM.get(env.ROOM.idFromName(topic)).fetch(request);
    }
    return go.fetch(request, env, ctx);
  },
};
