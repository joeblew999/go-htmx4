// Worker entry (deployed). WebSocket upgrades on /live/{topic} go straight to that topic's Room
// Durable Object; everything else goes to the Go Worker (TinyGo build in build/). workers-go has
// no WebSocket support, so upgrades can't pass through Go. Same shape as workers-go's
// _examples/durable-object-counter/worker.mjs.

import goWorker from "./build/worker.mjs";

export { Room } from "./room.mjs";

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
    return goWorker.fetch(request, env, ctx);
  },
};
