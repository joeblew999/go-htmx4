// /static/moq.js: @moq's <moq-watch> and <moq-publish> elements (vendored: mise run vendor:moq, bundled by
// internal/jsbundle from web/moq/npm-lock.json), and the glue between them and the live page's gsxui markup (#156).
//
// The glue only reads the elements' signals and writes data-* attributes on the page's sections; every label is
// already in the page, translated, and Tailwind's group-data variants show the one that applies (views/live.gsx). It
// decides no text, so Go and intl stay the one source of language.
import "@moq/watch/element";
import "@moq/publish/element";

// The elements' state is read every quarter second rather than subscribed to: a handful of signals, and polling
// keeps this file free of their lifetimes.
const TICK = 250;

// A watcher is "playing" while frames keep decoding, not while a socket is open: a connected relay with no
// broadcast behind it shows nothing. Ported from remy-sport's WatchProgress.
const STALL_MS = 3000;

// Without broadcast discovery (Cloudflare's draft-14 relays), a subscription made before the broadcast starts, or
// one that ended with it, is not picked up again: the watcher subscribes afresh after this long with no frames, as
// remy-sport re-mounted its element.
const RESUBSCRIBE_MS = 10_000;

// A broadcast found live normally plays within a second or two. Joining an iPhone's broadcast minutes in, the first
// two subscriptions got no frames and the third did (#156): before its first frame, a watcher whose broadcast is live
// subscribes again after this long instead.
const RESUBSCRIBE_LIVE_MS = 3000;

// Cloudflare's IETF relays carry no bandwidth probe, so the encoder's automatic bitrate clamp never runs and a phone
// on a weak uplink would attempt more than it can send. A fixed ceiling, as remy-sport's ENCODER.
const ENCODER = { maxBitrate: 2_000_000, maxPixels: 1280 * 720 };

// DIAGNOSTICS on screen (#156), for testing on phones; whether they stay is decided before #157 merges. Each card's
// [data-moq-debug] shows the @moq versions and patches (__NPM_VERSIONS__, __NPM_PATCHED__: internal/jsbundle defines
// them from the lock), the relay, the connection (status, transport, the session's protocol version), the browser and
// its features, and @moq's last warning; the broadcast card also two WebTransport sessions to the relay, as is and
// with @moq's options.
const moqVersions = Object.entries(__NPM_VERSIONS__)
	.filter(([name]) => name.startsWith("@moq/"))
	.map(([name, version]) => `${name.slice(5)}@${version}`)
	.join(" ");
const patched = __NPM_PATCHED__.length ? __NPM_PATCHED__.join(", ") : "none";

function debug(section, el, extra) {
	const out = section.querySelector("[data-moq-debug]");
	if (!out) return;
	const peek = (s) => {
		try {
			return s?.peek?.();
		} catch {
			return undefined;
		}
	};
	const c = el.connection;
	const session = peek(c?.established);
	const version = session?.version;
	const url = el.getAttribute("url");
	out.textContent = [
		`moq ${moqVersions} | patched ${patched}`,
		`relay ${url ? new URL(url).host : "-"} ${el.getAttribute("name") ?? "-"}`,
		`conn ${peek(c?.status) ?? "-"} via ${peek(c?.transport) ?? "-"} | version ${typeof version === "number" ? "0x" + version.toString(16) : (version ?? "-")}${session?.constructor?.name ? " (" + session.constructor.name + ")" : ""} | error ${peek(c?.error) ?? "-"}`,
		extra,
		`browser ${browser()} | ${features()}`,
		`last warning ${warnings.at(-1) ?? "none"}`,
	].join("\n");
}

function browser() {
	const ua = navigator.userAgent;
	const os = ua.match(/iPhone|iPad|Android [\d.]+|Mac OS X|Windows|Linux/)?.[0] ?? "?";
	const name = ua.match(/CriOS\/[\d.]+|FxiOS\/[\d.]+|EdgiOS\/[\d.]+|Edg\/[\d.]+|Firefox\/[\d.]+|Chrome\/[\d.]+|Version\/[\d.]+(?=.*Safari)/)?.[0] ?? "?";
	return `${os} ${name.replace("Version/", "Safari/")}`;
}

function features() {
	const has = (k) => (k in globalThis ? "+" : "-");
	return `WT${has("WebTransport")} VE${has("VideoEncoder")} VD${has("VideoDecoder")} AE${has("AudioEncoder")} AD${has("AudioDecoder")} MSTP${has("MediaStreamTrackProcessor")} COI${globalThis.crossOriginIsolated ? "+" : "-"}`;
}

const warnings = [];
const warn = console.warn.bind(console);
console.warn = (...args) => {
	const show = (a) => (a instanceof Error || a?.stack ? `${a.name}(${a.message}) code=${a.code ?? a.source ?? "-"} at ${String(a.stack).split("\n").slice(0, 3).join(" < ")}` : String(a));
	warnings.push(args.map(show).join(" ").slice(0, 600));
	warn(...args);
};
let probed = false;
const probes = { plain: "…", moq: "…" };
async function probe(name, url, options) {
	try {
		const wt = new WebTransport(url, options);
		wt.closed.catch(() => {});
		probes[name] = await Promise.race([
			wt.ready.then(() => "ready"),
			new Promise((r) => setTimeout(() => r("timeout 10s"), 10_000)),
		]).catch((e) => "rejected " + (e?.message ?? e));
		wt.close();
	} catch (e) {
		probes[name] = "throws " + (e?.message ?? e);
	}
}

for (const section of document.querySelectorAll("[data-moq=watch]")) watch(section);
for (const section of document.querySelectorAll("[data-moq=broadcast]")) broadcast(section);

function watch(section) {
	const el = section.querySelector("moq-watch");
	if (!el) return;
	if (!("VideoDecoder" in globalThis)) {
		section.dataset.state = "unsupported";
		el.paused = true;
		return;
	}
	let frames = 0;
	let progressed = 0;
	let played = false;
	let nudged = Date.now();
	let liveSince = 0;
	const read = () => {
		const now = Date.now();
		const paused = el.paused;
		const count = el.video?.out?.stats?.peek()?.frameCount ?? 0;
		const connected = el.connection?.status?.peek() === "connected";
		const live = el.broadcast?.out?.status?.peek();
		if (count !== frames) {
			frames = count;
			progressed = now;
			played = true;
		}
		let state;
		if (paused) state = "paused";
		else if (!connected) state = played ? "reconnecting" : "connecting";
		else if (played && now - progressed < STALL_MS) state = "playing";
		else if (live === "offline") state = played ? "ended" : "waiting";
		else state = played ? "reconnecting" : "waiting";
		if ((state === "waiting" || state === "ended" || state === "reconnecting") && connected && !document.hidden) {
			if (live === "live" && !liveSince) liveSince = now;
			const wait = !played && live === "live" ? RESUBSCRIBE_LIVE_MS : RESUBSCRIBE_MS;
			if (now - Math.max(progressed, nudged, liveSince) >= wait) {
				nudged = now;
				resubscribe(el);
			}
		}
		section.dataset.state = state;
		section.dataset.paused = String(paused);
		section.dataset.muted = String(el.muted);
		debug(section, el, `state ${state} | broadcast ${live ?? "-"} | frames ${count}`);
	};
	section.addEventListener("click", (event) => {
		const action = event.target.closest("[data-moq-action]")?.dataset.moqAction;
		if (action === "pause") el.paused = true;
		else if (action === "play") el.paused = false;
		else if (action === "unmute") el.muted = false;
		else if (action === "mute") el.muted = true;
		else if (action === "fullscreen") {
			const frame = section.querySelector("[data-moq-frame]");
			if (document.fullscreenElement) document.exitFullscreen();
			else frame?.requestFullscreen?.();
		}
		read();
	});
	read();
	setInterval(read, TICK);
}

function broadcast(section) {
	const el = section.querySelector("moq-publish");
	if (!el) return;
	if (!("VideoEncoder" in globalThis) || !navigator.mediaDevices?.getUserMedia) {
		section.dataset.state = "unsupported";
		return;
	}
	el.video?.config?.set(ENCODER);
	let started = false;
	let wasLive = false;
	const read = () => {
		const source = el.sources.video.peek();
		const error = source?.out?.error?.peek();
		const media = source?.out?.source?.peek();
		const connected = el.connection?.status?.peek() === "connected";
		const producing = !!el.broadcast?.net?.peek();
		let state;
		if (!started) state = wasLive ? "stopped" : "idle";
		else if (error) state = "error";
		else if (!media) state = "requesting";
		else if (connected && producing) state = "live";
		else state = wasLive ? "reconnecting" : "connecting";
		if (state === "live") wasLive = true;
		// "stopped" shows the idle label: the camera is off either way.
		section.dataset.state = state === "stopped" ? "idle" : state;
		section.dataset.error = state === "error" ? errorKind(error) : "";
		section.dataset.muted = String(el.muted);
		if (started && !probed && "WebTransport" in globalThis) {
			probed = true;
			probe("plain", el.getAttribute("url"));
			probe("moq", el.getAttribute("url"), { allowPooling: false, congestionControl: "low-latency", protocols: ["moq-lite-03", "moqt-16", "moqt-15"] });
		}
		debug(section, el, `state ${state} | camera ${media ? "on" : "off"} | producing ${producing} | probe plain ${probes.plain}, moq ${probes.moq}`);
	};
	section.addEventListener("click", (event) => {
		const action = event.target.closest("[data-moq-action]")?.dataset.moqAction;
		if (action === "start") {
			started = true;
			el.source = "camera";
		} else if (action === "stop") {
			started = false;
			el.source = undefined;
		} else if (action === "mute") el.muted = true;
		else if (action === "unmute") el.muted = false;
		read();
	});
	read();
	setInterval(read, TICK);
}

// errorKind sorts a capture failure into the three the page explains: refused, no camera, anything else.
function errorKind(error) {
	if (error?.name === "NotAllowedError" || error?.name === "SecurityError") return "denied";
	if (error?.name === "NotFoundError" || error?.name === "NotReadableError" || error?.name === "OverconstrainedError")
		return "unavailable";
	return "failed";
}

// resubscribe takes the element out of the page and puts it back on the next task: it disconnects, which closes its
// subscription, and connects again, which makes a new one. Its signals coalesce per microtask, so it has to be out
// for a whole task for the element to see it.
function resubscribe(el) {
	const parent = el.parentNode;
	const next = el.nextSibling;
	el.remove();
	setTimeout(() => parent.insertBefore(el, next), 0);
}
