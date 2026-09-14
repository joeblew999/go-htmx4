// Room: one Durable Object per board topic (deployed). It holds the browsers' WebSockets with the
// Hibernation API and pushes each published fragment to all of them. It owns no board data: D1 is
// the source of truth. "last" is only a cache so a (re)connecting browser gets the newest fragment
// at once, without a D1 read.
//
// Designed for ~1,000 sockets per topic (plan: .plans/2026-09-14_0754_workers-realtime-d1-do.md):
// sockets are receive-only, pings are auto-answered without waking the object, and broadcasts are
// coalesced to at most one per FLUSH_MS, keeping only the newest version.

import { DurableObject } from "cloudflare:workers";

const FLUSH_MS = 200;

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
    const [client, server] = Object.values(new WebSocketPair());
    this.ctx.acceptWebSocket(server);
    const last = await this.ctx.storage.get("last");
    if (last) server.send(last.html);
    return new Response(null, { status: 101, webSocket: client });
  }

  async publish(request) {
    if (request.method !== "POST") return new Response("method not allowed", { status: 405 });
    const version = Number(request.headers.get("X-Board-Version"));
    if (!Number.isSafeInteger(version) || version < 1) {
      return new Response("bad X-Board-Version", { status: 400 });
    }
    const html = await request.text();
    const storage = this.ctx.storage;
    const [last, pending] = await Promise.all([storage.get("last"), storage.get("pending")]);
    if (version <= Math.max(last?.version ?? 0, pending?.version ?? 0)) {
      return Response.json({ version, stale: true });
    }
    const msg = { version, html, at: Date.now() };
    if (!pending && msg.at - (last?.at ?? 0) >= FLUSH_MS) {
      return Response.json({ version, sent: await this.broadcast(msg) });
    }
    // Within FLUSH_MS of the last broadcast: keep only the newest version and flush on the alarm.
    await storage.put("pending", msg);
    if ((await storage.getAlarm()) === null) await storage.setAlarm((last?.at ?? 0) + FLUSH_MS);
    return Response.json({ version, queued: true });
  }

  async alarm() {
    const msg = await this.ctx.storage.get("pending");
    if (!msg) return;
    await this.ctx.storage.delete("pending");
    await this.broadcast({ ...msg, at: Date.now() });
  }

  async broadcast(msg) {
    await this.ctx.storage.put("last", msg);
    let sent = 0;
    for (const ws of this.ctx.getWebSockets()) {
      try {
        ws.send(msg.html);
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
    ws.close(code, reason);
  }
}
