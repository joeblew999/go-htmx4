// /static/moq.js: @moq's <moq-watch> and <moq-publish> elements (vendored: mise run vendor:moq, bundled by
// internal/jsbundle from web/moq/npm-lock.json), and the glue between them and the live page's gsxui markup (#156).
//
// The glue only reads the elements' signals and writes data-* attributes on the page's sections; every label is
// already in the page, translated, and Tailwind's group-data variants show the one that applies (views/live.gsx). It
// decides no text, so Go and intl stay the one source of language: the connection details it writes are machine
// values.
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

// Each card's connection details (LiveDetails, while open): the @moq versions and patches the bundle was built from
// (__NPM_VERSIONS__ and __NPM_PATCHED__, which internal/jsbundle defines from the lock and the patches), the relay and
// the broadcast's name, the connection and its transport, what the card is doing, the browser and the features the
// elements need, and @moq's last warning. Machine values, for testing on phones (#156).
const moqVersions = Object.entries(__NPM_VERSIONS__)
	.filter(([name]) => name.startsWith("@moq/"))
	.map(([name, version]) => `${name.slice(5)}@${version}`)
	.join(" ");
const patched = __NPM_PATCHED__.length ? __NPM_PATCHED__.join(", ") : "none";

function details(section, el, doing) {
	const out = section.querySelector("[data-moq-debug]");
	if (!out?.closest("details")?.open) return;
	const c = el.connection;
	const url = el.getAttribute("url");
	out.textContent = [
		`moq ${moqVersions} | patched ${patched}`,
		`relay ${url ? new URL(url).host : "-"} ${el.getAttribute("name") ?? "-"}`,
		`connection ${peek(c?.status) ?? "-"} via ${peek(c?.transport) ?? "-"} | error ${peek(c?.error) ?? "-"}`,
		doing,
		`browser ${browser()} | ${features()}`,
		`last warning ${warnings.at(-1) ?? "none"}`,
	].join("\n");
}

// peek reads a signal of the elements' without subscribing to it.
function peek(signal) {
	try {
		return signal?.peek?.();
	} catch {
		return undefined;
	}
}

function browser() {
	const ua = navigator.userAgent;
	const os = ua.match(/iPhone|iPad|Android [\d.]+|Mac OS X|Windows|Linux/)?.[0] ?? "?";
	const name =
		ua.match(/CriOS\/[\d.]+|FxiOS\/[\d.]+|EdgiOS\/[\d.]+|Edg\/[\d.]+|Firefox\/[\d.]+|Chrome\/[\d.]+|Version\/[\d.]+(?=.*Safari)/)?.[0] ?? "?";
	return `${os} ${name.replace("Version/", "Safari/")}`;
}

function features() {
	const has = (k) => (k in globalThis ? "+" : "-");
	return `WebTransport${has("WebTransport")} VideoEncoder${has("VideoEncoder")} VideoDecoder${has("VideoDecoder")} AudioEncoder${has("AudioEncoder")} AudioDecoder${has("AudioDecoder")}`;
}

// @moq reports connection errors only as console warnings: the last few are kept for the details.
const warnings = [];
const warn = console.warn.bind(console);
console.warn = (...args) => {
	warnings.push(args.map((a) => (a instanceof Error ? `${a.name}: ${a.message}` : String(a))).join(" ").slice(0, 300));
	if (warnings.length > 5) warnings.shift();
	warn(...args);
};

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
		details(section, el, `watch ${state} | broadcast ${live ?? "-"} | frames ${count}`);
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
		details(section, el, `broadcast ${state} | camera ${media ? "on" : "off"} | producing ${producing}`);
	};
	const link = section.querySelector("[data-testid=moq-share]");
	section.dataset.canShare = String(typeof navigator.share === "function");
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
		else if (action === "copy") copy(section, link);
		else if (action === "share") navigator.share({ url: link.value }).catch(() => {});
		read();
	});
	read();
	setInterval(read, TICK);
}

// copy puts the link on the clipboard and shows "Copied" for COPIED_MS. Where the clipboard API refuses (no
// permission, an insecure context), the link is selected and copied the old way.
const COPIED_MS = 2000;
async function copy(section, link) {
	try {
		await navigator.clipboard.writeText(link.value);
	} catch {
		link.select();
		document.execCommand("copy");
	}
	section.dataset.copied = "true";
	setTimeout(() => (section.dataset.copied = "false"), COPIED_MS);
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
