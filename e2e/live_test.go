package e2e

import (
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

// testLiveVideo is the live page end to end (#156), through the real relay (Cloudflare's MoQ network, as the Worker
// names it): one Chrome broadcasts its built-in test camera, a second watches, and frames must decode in the watcher.
// Chrome's fake camera is its own capture path with a synthetic picture, not a stand-in for MoQ: everything from the
// encoder to the relay and back is the real thing.
func testLiveVideo(t *testing.T) {
	camera := []chromedp.ExecAllocatorOption{
		chromedp.Flag("use-fake-device-for-media-stream", true),
		chromedp.Flag("use-fake-ui-for-media-stream", true),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
		// MoQ runs over WebTransport, which is QUIC: newTab turns QUIC off for the other checks.
		chromedp.Flag("disable-quic", false),
	}
	caster := newTabWith(t, "broadcaster", camera...)
	caster.open("/live")
	check(t, "the broadcast starts idle", caster.waitJS(10*time.Second, `document.querySelector("[data-moq=broadcast]").dataset.state === "idle"`))
	caster.clickText("[data-moq=broadcast] button", caster.str(`document.querySelector("[data-moq=broadcast] [data-moq-action=start]").textContent.trim()`))
	onAir := caster.waitJS(60*time.Second, `document.querySelector("[data-moq=broadcast]").dataset.state === "live"`)
	check(t, "the broadcaster goes on air", onAir, caster.str(liveState))

	viewer := newTabWith(t, "viewer", camera...)
	viewer.open("/live")
	playing := viewer.waitJS(60*time.Second, `document.querySelector("[data-moq=watch]").dataset.state === "playing"`)
	check(t, "the viewer plays the broadcast", playing, viewer.str(liveState))
	frames := viewer.num(`document.querySelector("moq-watch").video.out.stats.peek()?.frameCount ?? 0`)
	check(t, "frames decode in the viewer", frames > 0, frames)

	if !playing {
		t.Logf("broadcaster while the viewer waited: %s", caster.str(liveState))
	}
	caster.clickText("[data-moq=broadcast] button", caster.str(`document.querySelector("[data-moq=broadcast] [data-moq-action=stop]").textContent.trim()`))
	check(t, "stopping turns the camera off", caster.waitJS(10*time.Second, `document.querySelector("[data-moq=broadcast]").dataset.state === "idle"`))
	caster.noProblems()
	viewer.noProblems()
}

// liveState is what both elements say about themselves, for a failure's detail: the page's state, the connection
// (status, transport, error), and the broadcast (the watcher's status and catalog, frames decoded; the publisher's
// producer and catalog).
const liveState = `(() => {
	const peek = (s) => { try { return s?.peek?.(); } catch (e) { return "throws " + e; } };
	const out = {};
	for (const s of document.querySelectorAll("[data-moq]")) out[s.dataset.moq] = { state: s.dataset.state, error: s.dataset.error };
	const w = document.querySelector("moq-watch"), p = document.querySelector("moq-publish");
	const conn = (c) => ({ status: peek(c?.status), transport: String(peek(c?.transport)), error: String(peek(c?.error)) });
	if (w) out.watchElement = { conn: conn(w.connection), broadcast: peek(w.broadcast?.out?.status), catalog: !!peek(w.broadcast?.out?.catalog), frames: peek(w.video?.out?.stats)?.frameCount ?? 0 };
	if (p) out.publishElement = { conn: conn(p.connection), producing: !!peek(p.broadcast?.net), media: !!peek(peek(p.sources?.video)?.out?.source) };
	return JSON.stringify(out);
})()`
