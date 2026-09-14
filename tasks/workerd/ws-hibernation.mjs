// Cloudflare's WebSocket Hibernation example, copied verbatim from
// https://developers.cloudflare.com/durable-objects/best-practices/websockets/#hibernation-example
// (mise run upstream:cf:ws-hibernation). Only the default `fetch` entry at the bottom is ours:
// the docs' step 1 ("Proxy the request from the Worker to the Durable Object").

import { DurableObject } from "cloudflare:workers";

// Durable Object
export class WebSocketHibernationServer extends DurableObject {
	async fetch(request) {
		// Creates two ends of a WebSocket connection.
		const webSocketPair = new WebSocketPair();
		const [client, server] = Object.values(webSocketPair);

		// Calling `acceptWebSocket()` connects the WebSocket to the Durable Object, allowing the WebSocket to send and receive messages.
		// Unlike `ws.accept()`, `state.acceptWebSocket(ws)` allows the Durable Object to be hibernated
		// When the Durable Object receives a message during Hibernation, it will run the `constructor` to be re-initialized
		this.ctx.acceptWebSocket(server);

		return new Response(null, {
			status: 101,
			webSocket: client,
		});
	}

	async webSocketMessage(ws, message) {
		// Upon receiving a message from the client, reply with the same message,
		// but will prefix the message with "[Durable Object]: " and return the number of connections.
		ws.send(
			`[Durable Object] message: ${message}, connections: ${this.ctx.getWebSockets().length}`,
		);
	}

	async webSocketClose(ws, code, reason, wasClean) {
		// With web_socket_auto_reply_to_close (compat date >= 2026-04-07), the runtime
		// auto-replies to Close frames. Calling close() is safe but no longer required.
		ws.close(code, reason);
	}
}

// Ours: route /ws upgrades to one named Durable Object.
export default {
	async fetch(request, env) {
		const url = new URL(request.url);
		if (url.pathname !== "/ws" || request.headers.get("Upgrade") !== "websocket") {
			return new Response("expected a WebSocket upgrade on /ws", { status: 426 });
		}
		const id = env.WEBSOCKET_HIBERNATION_SERVER.idFromName("room");
		return env.WEBSOCKET_HIBERNATION_SERVER.get(id).fetch(request);
	},
};
