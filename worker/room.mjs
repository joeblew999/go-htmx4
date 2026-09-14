// Room: one Durable Object per board topic (deployed). It holds the browsers' WebSockets with the
// Hibernation API and pushes each published version to all of them. It owns no board data: D1 is
// the source of truth. "last" is only a cache so a (re)connecting browser gets the newest fragment
// at once, without a D1 read.
//
// Locales (plan: .plans/2026-09-14_1106_full-i18n.md, Phase 6): Go publishes one fragment per locale
// ({default, fragments}); each socket is tagged with its page's ?locale= and gets that locale's fragment
// (or the default's). Presence is a bare count, so the page renders its label in its own language.
//
// Designed for ~1,000 sockets per topic (plan: .plans/done/2026-09-14_0754_workers-realtime-d1-do.md):
// sockets are receive-only, pings are auto-answered without waking the object, and broadcasts are
// coalesced to at most one per FLUSH_MS, keeping only the newest version.
//
// Presence ("N online") is the one thing the Room knows that D1 doesn't: its open sockets. Connects
// and closes push a #presence fragment, coalesced the same way, so 1,000 tabs joining cost at most
// 1000 / FLUSH_MS presence broadcasts per second, not one per join.

import { DurableObject } from "cloudflare:workers";

const FLUSH_MS = 200;
// Design ceiling per topic: a Room handles ~1,000 requests/s, and every socket reconnects after a deploy.
// Socket 1,001 gets 503 (hx-ws retries with backoff).
const MAX_SOCKETS = 1000;

// fragmentFor picks a socket's fragment from a published version: its locale's, else the default's. A cache
// written before per-locale publishing ({version, html}) still serves until the topic's next publish.
function fragmentFor(msg, locale) {
  if (!msg.fragments) return msg.html;
  return msg.fragments[locale] ?? msg.fragments[msg.default];
}

export class Room extends DurableObject {
  constructor(ctx, env) {
    super(ctx, env);
    ctx.setWebSocketAutoResponse(new WebSocketRequestResponsePair("ping", "pong"));
  }

  async fetch(request) {
    // Only reachable from the Go Worker's stub; index.mjs forwards nothing but /live/* upgrades.
    if (new URL(request.url).pathname === "/publish") return this.publish(request);
    if (request.headers.get("Upgrade") !== "websocket") {
      return new Response("expected a WebSocket upgrade", { status: 426 });
    }
    if (this.ctx.getWebSockets().length >= MAX_SOCKETS) {
      return new Response("room full", { status: 503, headers: { "Retry-After": "10" } });
    }
    const [client, server] = Object.values(new WebSocketPair());
    // index.mjs has checked ?locale= against kit/live's LocalePattern.
    const locale = new URL(request.url).searchParams.get("locale") ?? "";
    this.ctx.acceptWebSocket(server, [locale]);
    const last = await this.ctx.storage.get("last");
    if (last) server.send(fragmentFor(last, locale));
    await this.presenceChanged();
    return new Response(null, { status: 101, webSocket: client });
  }

  async publish(request) {
    if (request.method !== "POST") return new Response("method not allowed", { status: 405 });
    const version = Number(request.headers.get("X-Board-Version"));
    if (!Number.isSafeInteger(version) || version < 1) {
      return new Response("bad X-Board-Version", { status: 400 });
    }
    let body;
    try {
      body = await request.json();
    } catch {
      return new Response("bad publish body", { status: 400 });
    }
    if (typeof body?.default !== "string" || typeof body?.fragments?.[body.default] !== "string") {
      return new Response("publish body needs fragments for its default locale", { status: 400 });
    }
    const storage = this.ctx.storage;
    const [last, pending] = await Promise.all([storage.get("last"), storage.get("pending")]);
    if (version <= Math.max(last?.version ?? 0, pending?.version ?? 0)) {
      return Response.json({ version, stale: true });
    }
    const msg = { version, default: body.default, fragments: body.fragments, at: Date.now() };
    if (!pending && msg.at - (last?.at ?? 0) >= FLUSH_MS) {
      return Response.json({ version, sent: await this.broadcast(msg) });
    }
    // Within FLUSH_MS of the last broadcast: keep only the newest version and flush on the alarm.
    await storage.put("pending", msg);
    await this.scheduleFlush((last?.at ?? 0) + FLUSH_MS);
    return Response.json({ version, queued: true });
  }

  async alarm() {
    const storage = this.ctx.storage;
    const [msg, presence] = await Promise.all([storage.get("pending"), storage.get("presencePending")]);
    if (msg) {
      await storage.delete("pending");
      await this.broadcast({ ...msg, at: Date.now() });
    }
    if (presence) {
      await storage.delete("presencePending");
      await this.broadcastPresence();
    }
  }

  // presenceChanged pushes the open-socket count now, or on the alarm if one went out within FLUSH_MS.
  async presenceChanged() {
    const storage = this.ctx.storage;
    const [lastAt, pending] = await Promise.all([storage.get("presenceAt"), storage.get("presencePending")]);
    if (!pending && Date.now() - (lastAt ?? 0) >= FLUSH_MS) {
      await this.broadcastPresence();
      return;
    }
    await storage.put("presencePending", true);
    await this.scheduleFlush((lastAt ?? 0) + FLUSH_MS);
  }

  async broadcastPresence() {
    // Count the socket being accepted too (not OPEN yet while its connect runs); skip closing ones.
    const open = this.ctx.getWebSockets().filter((ws) => ws.readyState < WebSocket.CLOSING);
    await this.ctx.storage.put("presenceAt", Date.now());
    const html = `<span id="presence" hx-swap-oob="true">${open.length}</span>`;
    for (const ws of open) {
      try {
        ws.send(html);
      } catch {
        // Closing in the meantime; its close event triggers another update.
      }
    }
  }

  // scheduleFlush sets the alarm to fire by `at` (keeps an earlier one).
  async scheduleFlush(at) {
    const current = await this.ctx.storage.getAlarm();
    if (current === null || at < current) await this.ctx.storage.setAlarm(at);
  }

  async broadcast(msg) {
    await this.ctx.storage.put("last", msg);
    let sent = 0;
    for (const ws of this.ctx.getWebSockets()) {
      try {
        ws.send(fragmentFor(msg, this.ctx.getTags(ws)[0] ?? ""));
        sent++;
      } catch {
        // The socket is already closing; the runtime drops it from getWebSockets().
      }
    }
    return sent;
  }

  // Receive-only sockets: writes go through the Go Worker, and pings are auto-answered.
  async webSocketMessage() {}

  async webSocketClose(ws, code, reason) {
    // Complete the close handshake. A bare ws.close() in the browser (what hx-ws does when htmx removes
    // its element, e.g. a boosted navigation away from /board) arrives as 1005 "no status", which can't be
    // sent back in a close frame: echoing it throws, the socket hangs half-closed and stays "online".
    try {
      ws.close(code === 1005 || code === 1006 ? 1000 : code, reason);
    } catch {
      // Already closed.
    }
    await this.presenceChanged();
  }

  async webSocketError() {
    await this.presenceChanged();
  }
}
